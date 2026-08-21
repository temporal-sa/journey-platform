package tracing

import (
	"context"
	"os"
	"strings"
	"sync"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"
)

var (
	tpMu sync.Mutex
	tp   *sdktrace.TracerProvider
)

// InitTracerProvider initializes and registers the global OpenTelemetry TracerProvider exporting to Jaeger via OTLP.
func InitTracerProvider(ctx context.Context, serviceName string) (func(context.Context) error, error) {
	tpMu.Lock()
	defer tpMu.Unlock()

	endpoint := os.Getenv("JAEGER_OTLP_HTTP_ENDPOINT")
	if endpoint == "" {
		endpoint = "http://127.0.0.1:4318/v1/traces"
	}
	endpoint = strings.TrimPrefix(endpoint, "http://")
	endpoint = strings.TrimPrefix(endpoint, "https://")

	hostPort := endpoint
	urlPath := "/v1/traces"
	if idx := strings.Index(endpoint, "/"); idx != -1 {
		hostPort = endpoint[:idx]
		urlPath = endpoint[idx:]
	}

	exporter, err := otlptracehttp.New(ctx,
		otlptracehttp.WithEndpoint(hostPort),
		otlptracehttp.WithURLPath(urlPath),
		otlptracehttp.WithInsecure(),
	)
	if err != nil {
		return func(context.Context) error { return nil }, err
	}

	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceNameKey.String(serviceName),
			semconv.ServiceVersionKey.String("1.0.0"),
		),
	)
	if err != nil {
		res = resource.Default()
	}

	bsp := sdktrace.NewBatchSpanProcessor(exporter,
		sdktrace.WithBatchTimeout(200*time.Millisecond),
		sdktrace.WithMaxExportBatchSize(512),
	)

	tp = sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithResource(res),
		sdktrace.WithSpanProcessor(bsp),
	)

	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	shutdown := func(sCtx context.Context) error {
		return tp.Shutdown(sCtx)
	}
	return shutdown, nil
}

// Tracer returns a named tracer from the global TracerProvider.
func Tracer(name string) trace.Tracer {
	return otel.GetTracerProvider().Tracer(name)
}
