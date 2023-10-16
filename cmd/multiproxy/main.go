package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"syscall"
	"time"

	"github.com/pomo-mondreganto/multiproxy/internal/ports"
	"github.com/pomo-mondreganto/multiproxy/internal/proxy"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
)

func main() {
	initLogger()

	portsDef := flag.String("ports", "", "ports definition, e.g. 10000-10100:10000-10100")
	target := flag.String("target", "", "target address")
	listen := flag.String("listen", "", "listen address")
	readTimeout := flag.Duration("read-timeout", time.Minute, "socket read timeout")
	writeTimeout := flag.Duration("write-timeout", time.Minute, "socket write timeout")
	flag.Parse()

	if *portsDef == "" || *target == "" {
		logrus.Fatal("ports and target must be specified")
	}

	portsDefinition, err := ports.Parse(*portsDef)
	if err != nil {
		logrus.Fatal(err)
	}

	runCtx, runCancel := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer runCancel()

	p := proxy.New(
		proxy.WithReadTimeout(*readTimeout),
		proxy.WithWriteTimeout(*writeTimeout),
	)

	g, gctx := errgroup.WithContext(runCtx)

	g.Go(func() error {
		p.RunLogger(gctx)
		return nil
	})

	logrus.Infof("proxying %d tcp ports", portsDefinition.SourceEnd-portsDefinition.SourceStart+1)
	for source := portsDefinition.SourceStart; source <= portsDefinition.SourceEnd; source++ {
		dest := portsDefinition.DestStart + (source - portsDefinition.SourceStart)

		listenAddr := fmt.Sprintf("%s:%d", *listen, source)
		targetAddr := fmt.Sprintf("%s:%d", *target, dest)

		g.Go(func() error {
			if err := p.ProxyPort(gctx, listenAddr, targetAddr); err != nil {
				return fmt.Errorf("proxying %s -> %s: %w", listenAddr, targetAddr, err)
			}
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		logrus.Fatal(err)
	}
	logrus.Info("finished successfully")
}

func initLogger() {
	logrus.SetFormatter(&logrus.TextFormatter{
		ForceColors:            true,
		FullTimestamp:          true,
		TimestampFormat:        "2006-01-02 15:04:05",
		DisableLevelTruncation: false,
		CallerPrettyfier: func(f *runtime.Frame) (string, string) {
			filename := filepath.Base(f.File)
			return "", fmt.Sprintf(" %s:%d", filename, f.Line)
		},
	})
	logrus.SetReportCaller(true)

	if ll := os.Getenv("MP_LOG_LEVEL"); ll != "" {
		level, err := logrus.ParseLevel(ll)
		if err != nil {
			logrus.Fatalf("Failed to parse log level %v: %v", ll, err)
		}
		logrus.SetLevel(level)
	}
}
