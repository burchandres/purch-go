package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"purch/internal/config"
	"purch/webhook"
)

func main() {
	config := config.GetConfig()
	server := webhook.GetWebhookServer(config)

	var wg sync.WaitGroup

	wg.Go(func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			panic(err)
		}
	})
	slog.Info("webhook server running", "port", config.WebhookPort)
	// watch for shutdown signals
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)
	<-signalChan

	slog.Info("shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		slog.Error("error shutting down webhooks server", "error", err.Error())
	}
	wg.Wait()
	slog.Info("shutdown complete.")
}
