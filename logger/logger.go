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

// SetContext does nothing, same as original
func (log *StdLogger) SetContext(ctx context.Context) {
	// no-op
}

// Only change: do not call SetPrefix; add level in Printf instead
func (log *StdLogger) Debugf(format string, args ...interface{}) {
	log.Logger.Printf("[Debug] "+format, args...)
}

func (log *StdLogger) Infof(format string, args ...interface{}) {
	log.Logger.Printf("[Info] "+format, args...)
}

func (log *StdLogger) Warningf(format string, args ...interface{}) {
	log.Logger.Printf("[Warning] "+format, args...)
}

func (log *StdLogger) Errorf(format string, args ...interface{}) {
	log.Logger.Printf("[Error] "+format, args...)
}

func (log *StdLogger) Criticalf(format string, args ...interface{}) {
	log.Logger.Printf("[Critical] "+format, args...)
}
