package handlers

import (
	"bufio"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/validated-pattern/journey-platform/internal/telemetry/logging"
)

func TestStreamLogsSSE(t *testing.T) {
	buf := logging.NewLogBuffer(10)
	buf.Write([]byte(`{"level":"info","message":"initial log entry"}`))

	h := New(nil, nil, nil)
	h.SetLogBuffer(buf)

	r := chi.NewRouter()
	h.RegisterRoutes(r)

	req := httptest.NewRequest("GET", "/api/v1/logs/stream", nil)
	ctx, cancel := context.WithTimeout(req.Context(), 500*time.Millisecond)
	defer cancel()
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()

	done := make(chan struct{})
	go func() {
		r.ServeHTTP(w, req)
		close(done)
	}()

	// Write a live log entry while stream is open
	time.Sleep(50 * time.Millisecond)
	buf.Write([]byte(`{"level":"error","message":"streamed error log"}`))

	<-done

	res := w.Result()
	assert.Equal(t, http.StatusOK, res.StatusCode)
	assert.Equal(t, "text/event-stream", res.Header.Get("Content-Type"))

	scanner := bufio.NewScanner(w.Body)
	var events []string
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "data: ") {
			events = append(events, strings.TrimPrefix(line, "data: "))
		}
	}

	assert.GreaterOrEqual(t, len(events), 2)

	var entry1, entry2 logging.LogEntry
	assert.NoError(t, json.Unmarshal([]byte(events[0]), &entry1))
	assert.Equal(t, "initial log entry", entry1.Message)

	assert.NoError(t, json.Unmarshal([]byte(events[1]), &entry2))
	assert.Equal(t, "streamed error log", entry2.Message)
	assert.Equal(t, "error", entry2.Level)
}
