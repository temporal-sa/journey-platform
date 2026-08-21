package main

import (
	"os"
	"os/signal"
	"syscall"
	"time"

	"context"

	"github.com/validated-pattern/journey-platform/internal/telemetry/logging"
	"github.com/validated-pattern/journey-platform/internal/telemetry/tracing"
)

func main() {
	logging.Init(false)
	ctx := context.Background()
	shutdownTracer, _ := tracing.InitTracerProvider(ctx, "target-dispatcher")
	if shutdownTracer != nil {
		defer func() {
			sCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			_ = shutdownTracer(sCtx)
		}()
	}
	logging.Info().Str("service", "target-dispatcher").Msg("Target Dispatcher online. OwnerID: target-dispatcher-worker-1, Topic: targets-production, Partition: 0, Generation: 1")
	logging.Info().Str("service", "target-dispatcher").Msg("Target Dispatcher service initialized successfully.")

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-stop:
			logging.Info().Str("service", "target-dispatcher").Msg("Stopping Target Dispatcher service...")
			return
		case <-ticker.C:
			// Heartbeat
		}
	}
}
