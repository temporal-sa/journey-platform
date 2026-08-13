package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/validated-pattern/journey-platform/internal/telemetry/logging"
)

// StreamLogs handles the GET /api/v1/logs/stream Server-Sent Events (SSE) endpoint.
// It streams initial buffered logs from LogBuffer and real-time log entries to connected clients.
func (h *Handlers) StreamLogs(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	var logBuf *logging.LogBuffer = h.GetLogBuffer()

	// Send initial buffered history entries
	entries := logBuf.GetEntries()
	for _, entry := range entries {
		data, err := json.Marshal(entry)
		if err == nil {
			_, _ = fmt.Fprintf(w, "data: %s\n\n", data)
		}
	}
	flusher.Flush()

	// Subscribe to live log events
	ch, unsubscribe := logBuf.Subscribe()
	defer unsubscribe()

	ctx := r.Context()
	for {
		select {
		case <-ctx.Done():
			return
		case entry, ok := <-ch:
			if !ok {
				return
			}
			data, err := json.Marshal(entry)
			if err == nil {
				_, _ = fmt.Fprintf(w, "data: %s\n\n", data)
				flusher.Flush()
			}
		}
	}
}
