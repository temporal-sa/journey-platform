package logging

import (
	"io"
	"os"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// Logger is a reference to the global Zerolog logger instance.
var Logger = &log.Logger

func init() {
	Init(isDevMode())
}

// Init initializes and returns the global Zerolog logger set to InfoLevel.
// In dev mode (APP_ENV=dev/development, LOG_FORMAT=pretty, or DEV_MODE=true),
// pretty console output (zerolog.ConsoleWriter) is used.
func Init(devMode bool) zerolog.Logger {
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	zerolog.TimeFieldFormat = time.RFC3339

	var writer io.Writer = os.Stdout
	if devMode {
		writer = zerolog.ConsoleWriter{
			Out:        os.Stdout,
			TimeFormat: time.RFC3339,
		}
	}
	multi := zerolog.MultiLevelWriter(writer, DefaultBuffer())

	logger := zerolog.New(multi).With().Timestamp().Logger()
	log.Logger = logger
	Logger = &log.Logger
	zerolog.DefaultContextLogger = &log.Logger
	return logger
}

func isDevMode() bool {
	env := strings.ToLower(os.Getenv("APP_ENV"))
	if env == "" {
		env = strings.ToLower(os.Getenv("ENV"))
	}
	if env == "" {
		env = strings.ToLower(os.Getenv("GO_ENV"))
	}
	if env == "dev" || env == "development" || strings.ToLower(os.Getenv("LOG_FORMAT")) == "pretty" || strings.ToLower(os.Getenv("DEV_MODE")) == "true" {
		return true
	}
	return false
}

// Info starts a new message with Info level.
func Info() *zerolog.Event {
	return log.Info()
}

// Debug starts a new message with Debug level.
func Debug() *zerolog.Event {
	return log.Debug()
}

// Warn starts a new message with Warn level.
func Warn() *zerolog.Event {
	return log.Warn()
}

// Error starts a new message with Error level.
func Error() *zerolog.Event {
	return log.Error()
}

// Fatal starts a new message with Fatal level.
func Fatal() *zerolog.Event {
	return log.Fatal()
}
