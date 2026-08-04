package middleware

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"
)

// responseWriterInterceptor wraps http.ResponseWriter to capture response status code and written size.
type responseWriterInterceptor struct {
	http.ResponseWriter
	statusCode   int
	bytesWritten int64
}

func (rw *responseWriterInterceptor) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriterInterceptor) Write(b []byte) (int, error) {
	if rw.statusCode == 0 {
		rw.statusCode = http.StatusOK
	}
	n, err := rw.ResponseWriter.Write(b)
	rw.bytesWritten += int64(n)
	return n, err
}

// RequestLogger returns a middleware that logs structured details for every HTTP request using slog and exports OpenTelemetry trace spans to Jaeger.
func RequestLogger(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			ww := &responseWriterInterceptor{ResponseWriter: w, statusCode: http.StatusOK}

			next.ServeHTTP(ww, r)

			duration := time.Since(start)
			ctx := r.Context()
			reqID := GetRequestID(ctx)
			traceID := GetTraceID(ctx)
			tenantID := GetTenantID(ctx)
			userID := GetUserID(ctx)

			if logger != nil {
				logger.InfoContext(ctx, "http request",
					slog.String("method", r.Method),
					slog.String("path", r.URL.Path),
					slog.Int("status", ww.statusCode),
					slog.Duration("duration", duration),
					slog.Int64("bytes_written", ww.bytesWritten),
					slog.String("request_id", reqID),
					slog.String("trace_id", traceID),
					slog.String("tenant_id", tenantID),
					slog.String("user_id", userID),
				)
			}

			// Asynchronously export OpenTelemetry trace span to Jaeger OTLP collector
			go func(reqMethod, reqPath string, statusCode int, dur time.Duration, bytesWritten int64, rID, tID, tenID, uID string, startTime time.Time) {
				otlpURL := os.Getenv("JAEGER_OTLP_HTTP_ENDPOINT")
				if otlpURL == "" {
					otlpURL = "http://127.0.0.1:4318/v1/traces"
				}

				endTime := startTime.Add(dur)
				spanID := fmt.Sprintf("%016x", time.Now().UnixNano()&0x7FFFFFFFFFFFFFFF)

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
									"spans": []map[string]interface{}{
										{
											"traceId":           tID,
											"spanId":            spanID,
											"name":              fmt.Sprintf("HTTP %s %s", reqMethod, reqPath),
											"kind":              1,
											"startTimeUnixNano": fmt.Sprintf("%d", startTime.UnixNano()),
											"endTimeUnixNano":   fmt.Sprintf("%d", endTime.UnixNano()),
											"attributes": []map[string]interface{}{
												{"key": "http.method", "value": map[string]interface{}{"stringValue": reqMethod}},
												{"key": "http.target", "value": map[string]interface{}{"stringValue": reqPath}},
												{"key": "http.status_code", "value": map[string]interface{}{"intValue": statusCode}},
												{"key": "http.response_content_length", "value": map[string]interface{}{"intValue": bytesWritten}},
												{"key": "request_id", "value": map[string]interface{}{"stringValue": rID}},
												{"key": "tenant_id", "value": map[string]interface{}{"stringValue": tenID}},
												{"key": "user_id", "value": map[string]interface{}{"stringValue": uID}},
											},
										},
									},
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
			}(r.Method, r.URL.Path, ww.statusCode, duration, ww.bytesWritten, reqID, traceID, tenantID, userID, start)
		})
	}
}
