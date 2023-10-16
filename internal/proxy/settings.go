package proxy

import (
	"time"
)

type Settings struct {
	readTimeout  time.Duration
	writeTimeout time.Duration
}

func DefaultSettings() Settings {
	return Settings{
		readTimeout:  time.Minute,
		writeTimeout: time.Minute,
	}
}

type Option func(s *Settings)

func WithReadTimeout(d time.Duration) Option {
	return func(s *Settings) {
		s.readTimeout = d
	}
}

func WithWriteTimeout(d time.Duration) Option {
	return func(s *Settings) {
		s.writeTimeout = d
	}
}
