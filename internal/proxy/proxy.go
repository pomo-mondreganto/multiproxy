package proxy

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"go.uber.org/atomic"
)

const bufferSize = 32 * 1024

var bufferPool = sync.Pool{
	New: func() interface{} {
		tmp := make([]byte, bufferSize)
		return &tmp
	},
}

type Proxy struct {
	settings Settings

	openPorts   *atomic.Int64
	totalConns  *atomic.Int64
	activeConns *atomic.Int64
}

func New(opts ...Option) *Proxy {
	settings := DefaultSettings()
	for _, opt := range opts {
		opt(&settings)
	}
	return &Proxy{
		settings: settings,

		openPorts:   atomic.NewInt64(0),
		totalConns:  atomic.NewInt64(0),
		activeConns: atomic.NewInt64(0),
	}
}

func (p *Proxy) RunLogger(ctx context.Context) {
	t := time.NewTicker(5 * time.Second)
	defer t.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			logrus.WithFields(logrus.Fields{
				"open_ports":   p.openPorts.Load(),
				"total_conns":  p.totalConns.Load(),
				"active_conns": p.activeConns.Load(),
				"goroutines":   runtime.NumGoroutine(),
			}).Info("proxy stats")
		}
	}
}

func (p *Proxy) ProxyPort(ctx context.Context, listenAddr, targetAddr string) error {
	logger := logrus.WithFields(logrus.Fields{
		"listen": listenAddr,
		"target": targetAddr,
	})

	lis, err := net.Listen("tcp", listenAddr)
	if err != nil {
		return fmt.Errorf("listening on %s: %w", listenAddr, err)
	}

	p.openPorts.Inc()
	defer p.openPorts.Dec()

	go func() {
		<-ctx.Done()
		if err := lis.Close(); err != nil {
			logger.Warnf("error closing listener: %v", err)
		}
	}()

	var wg sync.WaitGroup
	defer func() {
		<-ctx.Done()
		logger.Info("waiting for connections to finish")
		wg.Wait()
		logger.Info("all connections finished")
	}()

	for {
		conn, err := lis.Accept()
		if err != nil {
			select {
			case <-ctx.Done():
				return nil
			default:
				return fmt.Errorf("accepting connection on %s: %w", listenAddr, err)
			}
		}
		p.totalConns.Inc()

		wg.Add(1)
		go p.serveConn(ctx, &wg, conn, targetAddr, logger)
	}
}

func (p *Proxy) serveConn(ctx context.Context, wg *sync.WaitGroup, conn net.Conn, targetAddr string, logger *logrus.Entry) {
	defer wg.Done()

	connLogger := logger.WithField("remote", conn.RemoteAddr())
	connLogger.Info("accepted connection")

	defer func() {
		if err := conn.Close(); err != nil && !errors.Is(err, net.ErrClosed) {
			connLogger.Errorf("error closing connection: %v", err)
		} else {
			connLogger.Info("closed connection")
		}
	}()

	p.activeConns.Inc()
	defer p.activeConns.Dec()

	dialer := net.Dialer{
		Timeout: 5 * time.Second,
	}
	targetConn, err := dialer.DialContext(ctx, "tcp", targetAddr)
	if err != nil {
		logger.Errorf("dialing target: %v", err)
		return
	}
	defer func() {
		if err := targetConn.Close(); err != nil && !errors.Is(err, net.ErrClosed) {
			connLogger.Errorf("error closing target connection: %v", err)
		}
	}()

	proxyWg := sync.WaitGroup{}
	proxyWg.Add(2)
	go p.proxyConn(&proxyWg, conn, targetConn, connLogger.WithField("direction", "in"))
	go p.proxyConn(&proxyWg, targetConn, conn, connLogger.WithField("direction", "out"))
	proxyWg.Wait()

	connLogger.Info("finished")
}

func (p *Proxy) proxyConn(wg *sync.WaitGroup, dst net.Conn, src net.Conn, logger *logrus.Entry) {
	defer wg.Done()

	defer func() {
		if err := dst.Close(); err != nil && !errors.Is(err, net.ErrClosed) {
			logger.Errorf("error closing dst connection: %v", err)
		}
	}()

	buf := bufferPool.Get().(*[]byte)
	defer bufferPool.Put(buf)

	for {
		if err := src.SetReadDeadline(time.Now().Add(p.settings.readTimeout)); err != nil {
			logger.Errorf("error setting src deadline: %v", err)
		}

		var rerr, werr error
		n, rerr := src.Read(*buf)
		if n > 0 {
			if err := dst.SetWriteDeadline(time.Now().Add(p.settings.writeTimeout)); err != nil {
				logger.Errorf("error setting dst deadline: %v", err)
			}
			_, werr = dst.Write((*buf)[:n])
		}

		if errors.Is(rerr, io.EOF) {
			return
		}
		if rerr != nil {
			if !expectedConnIOError(rerr) {
				logger.Warnf("reading from src: %v", rerr)
			}
			return
		}
		if werr != nil {
			if !expectedConnIOError(werr) {
				logger.Warnf("writing to dst: %v", werr)
			}
			return
		}
	}
}

func expectedConnIOError(err error) bool {
	if errors.Is(err, net.ErrClosed) {
		return true
	}
	if strings.Contains(err.Error(), "connection reset by peer") {
		return true
	}
	if strings.Contains(err.Error(), "broken pipe") {
		return true
	}
	return false
}
