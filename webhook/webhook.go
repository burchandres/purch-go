/*
Separate webhook binary to receive plaid webhook notifications
to only expose this host/port and not the rest of the application.
*/
package webhook

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	// "github.com/plaid/plaid-go/v40/plaid"

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

/*
Below is an example of the plaid.SyncUpdatesAvailableWebhook message format we'd expect:
{
  "webhook_type": "TRANSACTIONS",
  "webhook_code": "SYNC_UPDATES_AVAILABLE",
  "item_id": "wz666MBjYWTp2PDzzggYhM6oWWmBb",
  "initial_update_complete": true,
  "historical_update_complete": false,
  "environment": "production"
}
*/

// TODO: Follow this guide for verifying plaid webhooks: https://plaid.com/docs/api/webhooks/webhook-verification/
func plaidAuthWebhookMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {}
}

func servePlaidWebhook(c *gin.Context) {
	// config := config.GetConfig()
	
	c.JSON(http.StatusOK, "To be implemented...")
}
