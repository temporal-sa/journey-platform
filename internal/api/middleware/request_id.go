package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

func init() {
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))
}

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

// EnsureOTelSpanContext ensures that ctx contains a valid OpenTelemetry SpanContext
// matching GetTraceID(ctx) and GetSpanID(ctx).
func EnsureOTelSpanContext(ctx context.Context) context.Context {
	if sc := trace.SpanContextFromContext(ctx); sc.IsValid() {
		tStr := sc.TraceID().String()
		sStr := sc.SpanID().String()
		if GetTraceID(ctx) == "" && tStr != "" {
			ctx = WithTraceID(ctx, tStr)
		}
		if GetSpanID(ctx) == "" && sStr != "" {
			ctx = WithSpanID(ctx, sStr)
		}
		return ctx
	}

	traceID := GetTraceID(ctx)
	if traceID == "" {
		rawUUID := uuid.New().String()
		traceID = strings.ReplaceAll(rawUUID, "-", "")
		ctx = WithTraceID(ctx, traceID)
	}

	spanID := GetSpanID(ctx)
	if spanID == "" {
		spanID = fmt.Sprintf("%016x", time.Now().UnixNano()&0x7FFFFFFFFFFFFFFF)
		ctx = WithSpanID(ctx, spanID)
	}

	tid, errT := trace.TraceIDFromHex(traceID)
	sid, errS := trace.SpanIDFromHex(spanID)
	if errT == nil && errS == nil {
		sc := trace.NewSpanContext(trace.SpanContextConfig{
			TraceID:    tid,
			SpanID:     sid,
			TraceFlags: trace.FlagsSampled,
		})
		ctx = trace.ContextWithRemoteSpanContext(ctx, sc)
	}
	return ctx
}

// InjectHTTPHeaders injects W3C traceparent headers into an outbound HTTP request header using otel.GetTextMapPropagator().
func InjectHTTPHeaders(ctx context.Context, req *http.Request) {
	if req == nil {
		return
	}
	ctx = EnsureOTelSpanContext(ctx)
	otel.GetTextMapPropagator().Inject(ctx, propagation.HeaderCarrier(req.Header))
	if req.Header.Get(HeaderTraceparent) == "" {
		tID := GetTraceID(ctx)
		sID := GetSpanID(ctx)
		if tID != "" {
			if sID == "" {
				sID = fmt.Sprintf("%016x", time.Now().UnixNano()&0x7FFFFFFFFFFFFFFF)
			}
			req.Header.Set(HeaderTraceparent, "00-"+tID+"-"+sID+"-01")
		}
	}
}

// RequestID middleware generates or propagates X-Request-ID and X-Trace-ID using otel.GetTextMapPropagator().
func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := otel.GetTextMapPropagator().Extract(r.Context(), propagation.HeaderCarrier(r.Header))

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
			if sc := trace.SpanContextFromContext(ctx); sc.IsValid() {
				traceID = sc.TraceID().String()
			}
		}
		if traceID == "" {
			rawUUID := uuid.New().String()
			traceID = strings.ReplaceAll(rawUUID, "-", "")
		}

		spanID := fmt.Sprintf("%016x", time.Now().UnixNano()&0x7FFFFFFFFFFFFFFF)
		if sc := trace.SpanContextFromContext(ctx); sc.IsValid() {
			spanID = sc.SpanID().String()
		}

		ctx = WithRequestID(ctx, reqID)
		ctx = WithTraceID(ctx, traceID)
		ctx = WithSpanID(ctx, spanID)
		ctx = EnsureOTelSpanContext(ctx)

		w.Header().Set(HeaderRequestID, reqID)
		w.Header().Set(HeaderTraceID, traceID)
		w.Header().Set(HeaderTraceparent, "00-"+traceID+"-"+spanID+"-01")
		otel.GetTextMapPropagator().Inject(ctx, propagation.HeaderCarrier(w.Header()))

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
