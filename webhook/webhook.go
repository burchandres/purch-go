/*
Separate webhook binary to receive plaid webhook notifications 
to only expose this host/port and not the rest of the application.
*/
package webhook

import (
	"fmt"
	"net/http"
	
	"github.com/gin-gonic/gin"

	"purch/internal/config"
)



func GetWebhookServer(config *config.Config) *http.Server {
	router := gin.Default()
	// quick healthcheck
	router.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, "pong")
	})
	// handle plaid webhook update
	router.POST("/webhook/plaid", servePlaidWebhook)

	return &http.Server{
		Addr:    fmt.Sprintf(":%d", config.WebhookPort),
		Handler: router,
	}
}

func servePlaidWebhook(c *gin.Context) {
	c.JSON(http.StatusOK, "To be implemented...")
}