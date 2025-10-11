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
	
	"purch/api"
	"purch/utils"
	"purch/database"
)

func main() {
	// get service configurations
	config, err := utils.LoadConfig()
	if err != nil {
		slog.Error("failed to load config", "error", err)
		panic(err)
	}
	slog.Info("loaded config", "config", config)
	// get API server
	server := getServer(config)
	// setup database connection pool
	if err = database.Init(config.GetPostgresURL()); err != nil {
		slog.Error("failed to initialize database pool", "error", err)
		panic(err)
	}
	defer database.Close()
	slog.Info("initialized database pool")
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
		slog.Info("error shutting down server", "error", err)
	}
	slog.Info("shutdown complete")
}

func getServer(config *utils.Config) *http.Server {
	router := gin.Default()

	router.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{
		  "message": "pong",
		})
	  })

	api.SetupUserEndpoints(router)
	
	return &http.Server{
		Addr: ":8080",
		Handler: router.Handler(),
	}
}