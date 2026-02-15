package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"

	"purch/internal/api"
	"purch/internal/config"
	"purch/internal/database"
)

func main() {
	// get service configurations
	config := config.GetCachedConfig()

	// setup database connection pool
	if err := database.Init(config.PostgresUrl); err != nil {
		panic(err)
	}
	defer database.Close()
	slog.Info("initialized database pool.")

	var wg sync.WaitGroup

	// get API server
	apiServer := getApiServer(config)
	// run server
	wg.Go(func() {
		defer wg.Done()
		
		if err := apiServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			panic(err)
		}
	})
	slog.Info("api server running", "port", config.ApiPort)

	// watch for shutdown signals
	signalChan := make(chan os.Signal, 1)

	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)
	<-signalChan
	slog.Info("shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := apiServer.Shutdown(ctx); err != nil {
		slog.Error("error shutting down api server.", "error", err.Error())
	}
	wg.Wait()
	slog.Info("shutdown complete.")
}

func getApiServer(config config.Config) *http.Server {
	router := gin.Default()

	router.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, "pong")
	})

	api.SetupUserEndpoints(router)
	api.SetupBudgetEndpoints(router)

	return &http.Server{
		Addr:    fmt.Sprintf(":%d", config.ApiPort),
		Handler: router,
	}
}
