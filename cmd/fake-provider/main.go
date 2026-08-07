package main

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = os.Getenv("FAKE_PROVIDER_PORT")
	}
	if port == "" {
		port = "8089"
	}

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"healthy","service":"fake-provider"}`))
	})

	r.Post("/send", handleMockSend)
	r.Post("/webhook/crm-sync", handleMockWebhook)

	server := &http.Server{
		Addr:    ":" + port,
		Handler: r,
	}

	log.Printf("Starting fake-provider service on :%s...", port)

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Fake-Provider server failed: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	log.Println("Shutting down fake-provider gracefully...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = server.Shutdown(ctx)
}

func handleMockSend(w http.ResponseWriter, r *http.Request) {
	bodyBytes, _ := io.ReadAll(r.Body)
	var req map[string]interface{}
	if len(bodyBytes) > 0 {
		_ = json.Unmarshal(bodyBytes, &req)
	}

	now := time.Now().UTC()
	resp := map[string]interface{}{
		"status":          "accepted",
		"message_id":      "msg-fake-" + now.Format("150405"),
		"executed_at":     now.Format(time.RFC3339),
		"mock_provider":   "mock-dispatcher",
		"request_payload": req,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(resp)
}

func handleMockWebhook(w http.ResponseWriter, r *http.Request) {
	bodyBytes, _ := io.ReadAll(r.Body)
	var req map[string]interface{}
	if len(bodyBytes) > 0 {
		_ = json.Unmarshal(bodyBytes, &req)
	}

	now := time.Now().UTC()
	resp := map[string]interface{}{
		"status":       "synced",
		"sync_id":      "sync-fake-" + now.Format("150405"),
		"synced_at":    now.Format(time.RFC3339),
		"crm_response": "200 OK",
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(resp)
}
