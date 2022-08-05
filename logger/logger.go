package logger

import (
	"context"

	"log"
)

type Logger interface {
	// SetContext is set context to logger.
	// work for appengine
	SetContext(ctx context.Context)

	// Debugf formats its arguments according to the format, analogous to fmt.Printf,
	// and records the text as a log message at Debug level.
	Debugf(format string, args ...interface{})

	// Infof is like Debugf, but at Info level.
	Infof(format string, args ...interface{})

	// Warningf is like Debugf, but at Warning level.
	Warningf(format string, args ...interface{})

	// Errorf is like Debugf, but at Error level.
	Errorf(format string, args ...interface{})

	// Criticalf is like Debugf, but at Critical level.
	Criticalf(format string, args ...interface{})
}

var _ Logger = (*StdLogger)(nil)

type StdLogger struct {
	Ctx context.Context
	*log.Logger
}

func (log *StdLogger) SetContext(ctx context.Context) {
	// do nothing
}

func (log *StdLogger) Debugf(format string, args ...interface{}) {
	log.Logger.SetPrefix("[Debug]")
	log.Logger.Printf(format, args...)
}

func (log *StdLogger) Infof(format string, args ...interface{}) {
	log.Logger.SetPrefix("[Info]")
	log.Logger.Printf(format, args...)
}

func (log *StdLogger) Warningf(format string, args ...interface{}) {
	log.Logger.SetPrefix("[Warning]")
	log.Logger.Printf(format, args...)
}

func (log *StdLogger) Errorf(format string, args ...interface{}) {
	log.Logger.SetPrefix("[Error]")
	log.Logger.Printf(format, args...)
}

func (log *StdLogger) Criticalf(format string, args ...interface{}) {
	log.Logger.SetPrefix("[Critical]")
	log.Logger.Printf(format, args...)
}
