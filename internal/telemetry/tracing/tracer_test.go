package tracing

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
)

func TestInitTracerProvider(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	shutdown, err := InitTracerProvider(ctx, "test-service")
	require.NoError(t, err)
	require.NotNil(t, shutdown)

	tracer := Tracer("test-tracer")
	assert.NotNil(t, tracer)

	// Create test span
	_, span := tracer.Start(context.Background(), "test-span")
	assert.True(t, span.SpanContext().IsValid())
	span.End()

	assert.NotNil(t, otel.GetTextMapPropagator())

	sCtx, sCancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer sCancel()
	err = shutdown(sCtx)
	assert.NoError(t, err)
}
