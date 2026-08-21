package logging

import (
	"errors"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
)

func TestTemporalZerologAdapter(t *testing.T) {
	zerolog.SetGlobalLevel(zerolog.DebugLevel)
	defer zerolog.SetGlobalLevel(zerolog.InfoLevel)

	buf := NewLogBuffer(20)
	logger := NewZerologLogger(buf).Level(zerolog.DebugLevel)
	tempLogger := NewTemporalLogger(logger)
	// 1. Test Info with keyvals
	tempLogger.Info("temporal workflow started", "workflow_id", "wf-123", "attempt", 1)

	entries := buf.GetEntries()
	assert.NotEmpty(t, entries)
	lastEntry := entries[len(entries)-1]
	assert.Equal(t, "info", lastEntry.Level)
	assert.Equal(t, "temporal workflow started", lastEntry.Message)
	assert.Equal(t, "wf-123", lastEntry.Fields["workflow_id"])

	// 2. Test Warn
	tempLogger.Warn("activity retry warning", "node_id", "node-email", "retry", 2)
	entries = buf.GetEntries()
	lastEntry = entries[len(entries)-1]
	assert.Equal(t, "warn", lastEntry.Level)
	assert.Equal(t, "activity retry warning", lastEntry.Message)
	assert.Equal(t, "node-email", lastEntry.Fields["node_id"])

	// 3. Test Error with error object
	testErr := errors.New("network error")
	tempLogger.Error("activity execution failed", "node_id", "node-sms", "error", testErr)
	entries = buf.GetEntries()
	lastEntry = entries[len(entries)-1]
	assert.Equal(t, "error", lastEntry.Level)
	assert.Equal(t, "activity execution failed", lastEntry.Message)
	assert.Equal(t, "network error", lastEntry.Fields["error"])

	// 4. Test Debug
	tempLogger.Debug("internal state evaluated", "evaluated", true)
	entries = buf.GetEntries()
	lastEntry = entries[len(entries)-1]
	assert.Equal(t, "debug", lastEntry.Level)
	assert.Equal(t, "internal state evaluated", lastEntry.Message)
}
