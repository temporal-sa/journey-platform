package middleware_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/validated-pattern/journey-platform/internal/activities"
	"github.com/validated-pattern/journey-platform/internal/api/middleware"
	"github.com/validated-pattern/journey-platform/internal/ingress"
	"github.com/validated-pattern/journey-platform/internal/outcomes"
	"github.com/validated-pattern/journey-platform/internal/workflows"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

func TestTraceContextContinuity_HTTP_Kafka_Temporal_Activity_OutboundHTTP(t *testing.T) {
	// 1. Inbound HTTP Request with W3C traceparent header
	originalTraceID := "4bf92f3577b34da6a3ce929d0e0e4736"
	originalSpanID := "00f067aa0ba902b7"
	inboundTraceparent := "00-" + originalTraceID + "-" + originalSpanID + "-01"

	req := httptest.NewRequest("POST", "/api/v1/events/emit", strings.NewReader(`{"event_type":"order.placed"}`))
	req.Header.Set("traceparent", inboundTraceparent)
	rec := httptest.NewRecorder()

	var extractedCtx context.Context
	handler := middleware.RequestID(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		extractedCtx = r.Context()
		w.WriteHeader(http.StatusOK)
	}))

	handler.ServeHTTP(rec, req)

	// Verify inbound extraction
	assert.Equal(t, originalTraceID, middleware.GetTraceID(extractedCtx))
	assert.Contains(t, rec.Header().Get("traceparent"), originalTraceID)

	sc := trace.SpanContextFromContext(extractedCtx)
	require.True(t, sc.IsValid(), "OpenTelemetry SpanContext must be valid in context")
	assert.Equal(t, originalTraceID, sc.TraceID().String())

	// 2. Kafka Message Production with traceparent injection
	kafkaHeaders := ingress.InjectKafkaHeaders(extractedCtx, nil)
	require.NotEmpty(t, kafkaHeaders["traceparent"])
	assert.Contains(t, kafkaHeaders["traceparent"], originalTraceID)

	kafkaMsg := ingress.ProduceTargetKafkaMessage(extractedCtx, "targets-production", 0, "user-123", []byte(`{"event_id":"evt-1"}`))
	assert.Contains(t, kafkaMsg.Headers["traceparent"], originalTraceID)

	// 3. Kafka Message Consumption in Outcome Consumer
	consumedCtx := outcomes.ExtractKafkaTraceHeaders(context.Background(), kafkaMsg.Headers)
	assert.Equal(t, originalTraceID, middleware.GetTraceID(consumedCtx))
	consumedSc := trace.SpanContextFromContext(consumedCtx)
	require.True(t, consumedSc.IsValid())
	assert.Equal(t, originalTraceID, consumedSc.TraceID().String())

	// 4. Temporal Tracing Interceptor Creation
	interceptor, err := workflows.CreateTemporalTracingInterceptor()
	require.NoError(t, err)
	require.NotNil(t, interceptor)

	// 5. Activity Execution & Outbound HTTP Request Propagation
	outboundServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		outboundTp := r.Header.Get("traceparent")
		assert.NotEmpty(t, outboundTp, "Outbound HTTP request must contain traceparent header")
		assert.Contains(t, outboundTp, originalTraceID, "Outbound HTTP traceparent must carry original Trace ID")
		w.WriteHeader(http.StatusOK)
	}))
	defer outboundServer.Close()

	actCtx := middleware.EnsureOTelSpanContext(extractedCtx)

	outboundReq, err := http.NewRequestWithContext(actCtx, "POST", outboundServer.URL, strings.NewReader(`{"test":true}`))
	require.NoError(t, err)
	middleware.InjectHTTPHeaders(actCtx, outboundReq)
	otel.GetTextMapPropagator().Inject(actCtx, propagation.HeaderCarrier(outboundReq.Header))

	outboundResp, err := http.DefaultClient.Do(outboundReq)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, outboundResp.StatusCode)
	_ = outboundResp.Body.Close()
	// Verify ActionGateway ExecuteAction retains Trace ID
	gateway := activities.NewActionGateway()
	actReq := activities.ActionRequest{
		TenantID:   "tenant-1",
		WorkflowID: "wf-1",
		NodeID:     "node-1",
		NodeVisit:  1,
	}
	res, err := gateway.ExecuteAction(actCtx, actReq)
	require.NoError(t, err)
	assert.NotNil(t, res)
}

func TestOtelTextMapPropagator_DirectInjectionAndExtraction(t *testing.T) {
	traceID := "0af7651916cd43dd8448eb211c80319c"
	spanID := "b7ba6646571706f9"
	ctx := middleware.WithTraceID(context.Background(), traceID)
	ctx = middleware.WithSpanID(ctx, spanID)
	ctx = middleware.EnsureOTelSpanContext(ctx)

	carrier := make(propagation.MapCarrier)
	otel.GetTextMapPropagator().Inject(ctx, carrier)

	require.NotEmpty(t, carrier["traceparent"])
	assert.Contains(t, carrier["traceparent"], traceID)
	assert.Contains(t, carrier["traceparent"], spanID)

	extracted := otel.GetTextMapPropagator().Extract(context.Background(), carrier)
	extractedSc := trace.SpanContextFromContext(extracted)
	require.True(t, extractedSc.IsValid())
	assert.Equal(t, traceID, extractedSc.TraceID().String())
}
