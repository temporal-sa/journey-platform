package logging

import (
	"encoding/json"
	"os"
	"sync"
	"time"

	"github.com/rs/zerolog"
)

// LogEntry represents a structured log event captured by LogBuffer.
type LogEntry struct {
	Timestamp string                 `json:"timestamp"`
	Level     string                 `json:"level"`
	Message   string                 `json:"message"`
	Raw       string                 `json:"raw,omitempty"`
	Fields    map[string]interface{} `json:"fields,omitempty"`
}

// LogBuffer is a thread-safe ring-buffer log sink that captures Zerolog log events.
type LogBuffer struct {
	mu          sync.RWMutex
	capacity    int
	entries     []LogEntry
	subscribers map[chan LogEntry]struct{}
}

var (
	defaultBuffer     *LogBuffer
	defaultBufferOnce sync.Once
)

// NewLogBuffer creates a thread-safe ring-buffer log sink with the specified capacity.
func NewLogBuffer(capacity int) *LogBuffer {
	if capacity <= 0 {
		capacity = 1000
	}
	return &LogBuffer{
		capacity:    capacity,
		entries:     make([]LogEntry, 0, capacity),
		subscribers: make(map[chan LogEntry]struct{}),
	}
}

// DefaultBuffer returns the global default LogBuffer singleton instance.
func DefaultBuffer() *LogBuffer {
	defaultBufferOnce.Do(func() {
		defaultBuffer = NewLogBuffer(1000)
	})
	return defaultBuffer
}

// Write captures log entries written by Zerolog or any io.Writer.
// It parses the entry, stores it in the ring buffer, and broadcasts it to active subscribers.
func (b *LogBuffer) Write(p []byte) (n int, err error) {
	n = len(p)
	if n == 0 {
		return n, nil
	}

	entry := parseLogEntry(p)

	b.mu.Lock()
	if len(b.entries) >= b.capacity {
		// Ring buffer: drop oldest entry by shifting slice left
		copy(b.entries, b.entries[1:])
		b.entries[len(b.entries)-1] = entry
	} else {
		b.entries = append(b.entries, entry)
	}

	// Broadcast to subscribers
	for ch := range b.subscribers {
		select {
		case ch <- entry:
		default:
			// Non-blocking send to prevent slow clients from blocking logger
		}
	}
	b.mu.Unlock()

	return n, nil
}

// GetEntries returns a thread-safe snapshot slice of all captured log entries.
func (b *LogBuffer) GetEntries() []LogEntry {
	b.mu.RLock()
	defer b.mu.RUnlock()

	result := make([]LogEntry, len(b.entries))
	copy(result, b.entries)
	return result
}

// Subscribe registers a new listener channel for real-time log events.
// It returns the channel and an unsubscribe cleanup function.
func (b *LogBuffer) Subscribe() (<-chan LogEntry, func()) {
	b.mu.Lock()
	defer b.mu.Unlock()

	ch := make(chan LogEntry, 100)
	b.subscribers[ch] = struct{}{}

	unsubscribe := func() {
		b.mu.Lock()
		defer b.mu.Unlock()
		if _, exists := b.subscribers[ch]; exists {
			delete(b.subscribers, ch)
			close(ch)
		}
	}

	return ch, unsubscribe
}

// Clear flushes all stored entries from the ring buffer.
func (b *LogBuffer) Clear() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.entries = b.entries[:0]
}

// NewZerologLogger constructs a Zerolog logger that writes output to both os.Stdout and the LogBuffer.
func NewZerologLogger(buf *LogBuffer) zerolog.Logger {
	if buf == nil {
		buf = DefaultBuffer()
	}
	multi := zerolog.MultiLevelWriter(os.Stdout, buf)
	return zerolog.New(multi).With().Timestamp().Logger()
}

// parseLogEntry extracts timestamp, level, message, and extra fields from Zerolog JSON bytes.
func parseLogEntry(p []byte) LogEntry {
	raw := string(p)
	entry := LogEntry{
		Timestamp: time.Now().Format(time.RFC3339),
		Level:     "info",
		Message:   raw,
		Raw:       raw,
	}

	var m map[string]interface{}
	if err := json.Unmarshal(p, &m); err == nil {
		fields := make(map[string]interface{})
		for k, v := range m {
			switch k {
			case "level":
				if s, ok := v.(string); ok {
					entry.Level = s
				}
			case "time", "timestamp", "@timestamp":
				if s, ok := v.(string); ok {
					entry.Timestamp = s
				}
			case "message", "msg":
				if s, ok := v.(string); ok {
					entry.Message = s
				}
			default:
				fields[k] = v
			}
		}
		if len(fields) > 0 {
			entry.Fields = fields
		}
	}

	return entry
}
