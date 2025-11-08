package api

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"

	"purch/internal/config"
)

func GetWebhookServer(config *config.Config) *http.Server {
	router := gin.Default()

	router.POST("/webhook/plaid", servePlaidWebhook)

	return &http.Server{
		Addr:    fmt.Sprintf(":%d", config.WebhookPort),
		Handler: router,
	}
}

func servePlaidWebhook(c *gin.Context) {
	c.JSON(http.StatusOK, "To be implemented...")
}
