package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"

	"purch/internal/api"
	"purch/internal/database"
	"purch/internal/utils"
)

var slogLevels = map[string]slog.Level{
	"DEBUG": slog.LevelDebug,
	"INFO":  slog.LevelInfo,
	"WARN":  slog.LevelWarn,
	"ERROR": slog.LevelError,
}

func main() {
	// get service configurations
	config := utils.GetConfig()
	// configure logging with config.LogLevel
	configureLogging(config.LogLevel)
	slog.Debug("loaded config.", "config", *config)

	// setup database connection pool
	if err := database.Init(config.PostgresUrl); err != nil {
		panic(err)
	}
	defer database.Close()
	slog.Info("initialized database pool.")

	// get API server
	server := getServer()
	// run server
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			panic(err)
		}
	}()

	// watch for shutdown signals
	signalChan := make(chan os.Signal, 1)

	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)
	<-signalChan
	slog.Info("shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		slog.Info("error shutting down server.", "error", err.Error())
	}
	slog.Info("shutdown complete.")
}

func configureLogging(logLevel string) {
	logger := slog.New(slog.NewTextHandler(
		os.Stdout,
		&slog.HandlerOptions{
			Level: slogLevels[logLevel],
		},
	),
	)
	slog.SetDefault(logger)
}

func getServer() *http.Server {
	router := gin.Default()

	router.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, "pong")
	})

	api.SetupUserEndpoints(router)
	api.SetupBudgetEndpoints(router)

	return &http.Server{
		Addr:    ":8080",
		Handler: router,
	}
}
