package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
)

const (
	requestIDKey contextKey = "request_id"
	traceIDKey   contextKey = "trace_id"
	spanIDKey    contextKey = "span_id"

	HeaderRequestID   = "X-Request-ID"
	HeaderTraceID     = "X-Trace-ID"
	HeaderTraceparent = "traceparent"
)

// WithRequestID returns a new context with the request ID.
func WithRequestID(ctx context.Context, reqID string) context.Context {
	return context.WithValue(ctx, requestIDKey, reqID)
}

// GetRequestID retrieves the request ID from context.
func GetRequestID(ctx context.Context) string {
	if val, ok := ctx.Value(requestIDKey).(string); ok {
		return val
	}
	return ""
}

// WithTraceID returns a new context with the trace ID.
func WithTraceID(ctx context.Context, traceID string) context.Context {
	return context.WithValue(ctx, traceIDKey, traceID)
}

// GetTraceID retrieves the trace ID from context.
func GetTraceID(ctx context.Context) string {
	if val, ok := ctx.Value(traceIDKey).(string); ok {
		return val
	}
	return ""
}
// WithSpanID returns a new context with the span ID.
func WithSpanID(ctx context.Context, spanID string) context.Context {
	return context.WithValue(ctx, spanIDKey, spanID)
}

// GetSpanID retrieves the span ID from context.
func GetSpanID(ctx context.Context) string {
	if val, ok := ctx.Value(spanIDKey).(string); ok {
		return val
	}
	return ""
}

// RequestID middleware generates or propagates X-Request-ID and X-Trace-ID.
func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqID := r.Header.Get(HeaderRequestID)
		if reqID == "" {
			reqID = uuid.New().String()
		}

		traceID := r.Header.Get(HeaderTraceID)
		if traceID == "" {
			if tp := r.Header.Get(HeaderTraceparent); tp != "" {
				parts := strings.Split(tp, "-")
				if len(parts) >= 2 && len(parts[1]) == 32 {
					traceID = parts[1]
				}
			}
		}
		if traceID == "" {
			rawUUID := uuid.New().String()
			traceID = strings.ReplaceAll(rawUUID, "-", "")
		}

		spanID := fmt.Sprintf("%016x", time.Now().UnixNano()&0x7FFFFFFFFFFFFFFF)

		ctx := WithRequestID(r.Context(), reqID)
		ctx = WithTraceID(ctx, traceID)
		ctx = WithSpanID(ctx, spanID)

		w.Header().Set(HeaderRequestID, reqID)
		w.Header().Set(HeaderTraceID, traceID)
		w.Header().Set(HeaderTraceparent, "00-"+traceID+"-"+spanID+"-01")

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
