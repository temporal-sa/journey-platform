package logging

import (
	"fmt"

	"github.com/rs/zerolog"
	temporalLog "go.temporal.io/sdk/log"
)

type temporalZerologAdapter struct {
	logger zerolog.Logger
}

// NewTemporalLogger creates a Temporal SDK compatible Logger backed by Zerolog.
func NewTemporalLogger(zlog ...zerolog.Logger) temporalLog.Logger {
	var l zerolog.Logger
	if len(zlog) > 0 {
		l = zlog[0]
	} else if Logger != nil {
		l = *Logger
	} else {
		l = NewZerologLogger(DefaultBuffer())
	}
	return &temporalZerologAdapter{logger: l}
}

func (a *temporalZerologAdapter) logEvent(event *zerolog.Event, msg string, keyvals ...interface{}) {
	if event == nil {
		return
	}
	for i := 0; i < len(keyvals); i += 2 {
		key := fmt.Sprintf("%v", keyvals[i])
		if i+1 < len(keyvals) {
			val := keyvals[i+1]
			if err, ok := val.(error); ok {
				event.Err(err)
			} else {
				event.Interface(key, val)
			}
		} else {
			event.Interface(key, nil)
		}
	}
	event.Msg(msg)
}

func (a *temporalZerologAdapter) Debug(msg string, keyvals ...interface{}) {
	a.logEvent(a.logger.Debug(), msg, keyvals...)
}

func (a *temporalZerologAdapter) Info(msg string, keyvals ...interface{}) {
	a.logEvent(a.logger.Info(), msg, keyvals...)
}

func (a *temporalZerologAdapter) Warn(msg string, keyvals ...interface{}) {
	a.logEvent(a.logger.Warn(), msg, keyvals...)
}

func (a *temporalZerologAdapter) Error(msg string, keyvals ...interface{}) {
	a.logEvent(a.logger.Error(), msg, keyvals...)
}
