package main

import (
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/validated-pattern/journey-platform/internal/telemetry/logging"
)

func main() {
	logging.Init(false)
	logging.Info().Str("service", "target-dispatcher").Msg("Initializing Target Dispatcher service...")
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
