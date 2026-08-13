package postgres

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/validated-pattern/journey-platform/internal/api/middleware"
)

// TracedDB wraps a DBTX connection to instrument raw SQL queries with OpenTelemetry spans sent to Jaeger.
type TracedDB struct {
	db DBTX
}

// NewTracedDB returns a new DBTX implementation that traces queries to Jaeger.
func NewTracedDB(db DBTX) DBTX {
	if db == nil {
		return nil
	}
	return &TracedDB{db: db}
}

func extractQueryOperation(query string) string {
	q := strings.TrimSpace(query)
	if idx := strings.IndexAny(q, " \t\n\r"); idx > 0 {
		return strings.ToUpper(q[:idx])
	}
	return "SQL"
}

func exportDBSpan(ctx context.Context, query string, duration time.Duration, queryErr error, start time.Time) {
	otlpURL := os.Getenv("JAEGER_OTLP_HTTP_ENDPOINT")
	if otlpURL == "" {
		otlpURL = "http://127.0.0.1:4318/v1/traces"
	}

	traceID := middleware.GetTraceID(ctx)
	parentSpanID := middleware.GetSpanID(ctx)

	if traceID == "" {
		return
	}

	spanID := fmt.Sprintf("%016x", time.Now().UnixNano()&0x7FFFFFFFFFFFFFFF)
	endTime := start.Add(duration)
	opName := extractQueryOperation(query)

	spanName := fmt.Sprintf("DB %s", opName)
	cleanQuery := strings.Join(strings.Fields(query), " ")

	attrs := []map[string]interface{}{
		{"key": "db.system", "value": map[string]interface{}{"stringValue": "postgresql"}},
		{"key": "db.statement", "value": map[string]interface{}{"stringValue": cleanQuery}},
		{"key": "db.name", "value": map[string]interface{}{"stringValue": "journeydb"}},
		{"key": "service.name", "value": map[string]interface{}{"stringValue": "control-api"}},
	}

	if queryErr != nil {
		attrs = append(attrs, map[string]interface{}{
			"key": "error", "value": map[string]interface{}{"boolValue": true},
		}, map[string]interface{}{
			"key": "error.message", "value": map[string]interface{}{"stringValue": queryErr.Error()},
		})
	}

	span := map[string]interface{}{
		"traceId":           traceID,
		"spanId":            spanID,
		"name":              spanName,
		"kind":              3, // CLIENT
		"startTimeUnixNano": fmt.Sprintf("%d", start.UnixNano()),
		"endTimeUnixNano":   fmt.Sprintf("%d", endTime.UnixNano()),
		"attributes":        attrs,
	}
	if parentSpanID != "" {
		span["parentSpanId"] = parentSpanID
	}

	spanPayload := map[string]interface{}{
		"resourceSpans": []map[string]interface{}{
			{
				"resource": map[string]interface{}{
					"attributes": []map[string]interface{}{
						{"key": "service.name", "value": map[string]interface{}{"stringValue": "control-api"}},
						{"key": "service.version", "value": map[string]interface{}{"stringValue": "1.0.0"}},
					},
				},
				"scopeSpans": []map[string]interface{}{
					{
						"spans": []map[string]interface{}{span},
					},
				},
			},
		},
	}

	bodyBytes, err := json.Marshal(spanPayload)
	if err != nil {
		return
	}

	req, err := http.NewRequest("POST", otlpURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "application/json")
	c := &http.Client{Timeout: 2 * time.Second}
	resp, err := c.Do(req)
	if err == nil && resp != nil {
		_ = resp.Body.Close()
	}
}

func (t *TracedDB) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	start := time.Now()
	res, err := t.db.ExecContext(ctx, query, args...)
	dur := time.Since(start)
	go exportDBSpan(ctx, query, dur, err, start)
	return res, err
}

func (t *TracedDB) PrepareContext(ctx context.Context, query string) (*sql.Stmt, error) {
	start := time.Now()
	stmt, err := t.db.PrepareContext(ctx, query)
	dur := time.Since(start)
	go exportDBSpan(ctx, query, dur, err, start)
	return stmt, err
}

func (t *TracedDB) QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	start := time.Now()
	rows, err := t.db.QueryContext(ctx, query, args...)
	dur := time.Since(start)
	go exportDBSpan(ctx, query, dur, err, start)
	return rows, err
}

func (t *TracedDB) QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row {
	start := time.Now()
	row := t.db.QueryRowContext(ctx, query, args...)
	dur := time.Since(start)
	go exportDBSpan(ctx, query, dur, nil, start)
	return row
}
