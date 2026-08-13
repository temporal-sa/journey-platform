package logging

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
)

func TestLogBufferRingBuffer(t *testing.T) {
	buf := NewLogBuffer(3)

	buf.Write([]byte(`{"level":"info","message":"msg1"}`))
	buf.Write([]byte(`{"level":"warn","message":"msg2"}`))
	buf.Write([]byte(`{"level":"error","message":"msg3"}`))

	entries := buf.GetEntries()
	assert.Len(t, entries, 3)
	assert.Equal(t, "msg1", entries[0].Message)
	assert.Equal(t, "msg2", entries[1].Message)
	assert.Equal(t, "msg3", entries[2].Message)

	// Writing 4th entry should evict msg1
	buf.Write([]byte(`{"level":"info","message":"msg4"}`))

	entries = buf.GetEntries()
	assert.Len(t, entries, 3)
	assert.Equal(t, "msg2", entries[0].Message)
	assert.Equal(t, "msg3", entries[1].Message)
	assert.Equal(t, "msg4", entries[2].Message)
}

func TestLogBufferSubscribe(t *testing.T) {
	buf := NewLogBuffer(10)
	ch, unsubscribe := buf.Subscribe()
	defer unsubscribe()

	buf.Write([]byte(`{"level":"error","message":"test subscription"}`))

	select {
	case entry := <-ch:
		assert.Equal(t, "error", entry.Level)
		assert.Equal(t, "test subscription", entry.Message)
	case <-time.After(1 * time.Second):
		t.Fatal("timed out waiting for subscription entry")
	}
}

func TestZerologIntegration(t *testing.T) {
	buf := NewLogBuffer(10)
	logger := zerolog.New(buf).With().Timestamp().Logger()

	logger.Info().Str("service", "control-api").Msg("server started")
	logger.Error().Int("code", 500).Msg("internal error")

	entries := buf.GetEntries()
	assert.Len(t, entries, 2)

	assert.Equal(t, "info", entries[0].Level)
	assert.Equal(t, "server started", entries[0].Message)
	assert.Equal(t, "control-api", entries[0].Fields["service"])

	assert.Equal(t, "error", entries[1].Level)
	assert.Equal(t, "internal error", entries[1].Message)
	assert.Equal(t, float64(500), entries[1].Fields["code"])
}

func TestConcurrentLogBuffer(t *testing.T) {
	buf := NewLogBuffer(100)
	var wg sync.WaitGroup

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				buf.Write([]byte(fmt.Sprintf(`{"level":"info","message":"worker %d msg %d"}`, workerID, j)))
			}
		}(i)
	}

	wg.Wait()
	entries := buf.GetEntries()
	assert.Len(t, entries, 100)
}
