package main

import (
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"context"
	"time"
	"syscall"

	"github.com/gin-gonic/gin"
)

func main() {
	server := getServer()

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server error.", "error", err)
		}
	}()

	signalChan := make(chan os.Signal, 1)

	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)
	<-signalChan
	slog.Info("shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		slog.Info("error shutting down server", "error", err)
	}
	slog.Info("shutdown complete")
}

func getServer() *http.Server {
	router := gin.Default()

	router.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{
		  "message": "pong",
		})
	  })

	return &http.Server{
		Addr: ":8080",
		Handler: router.Handler(),
	}
}