/*
Separate webhook binary to receive plaid webhook notifications
to only expose this host/port and not the rest of the application.

For now it only watches for new transactions for already registered accounts.

Next on development roadmap:
- Watching for new accounts added to an item (i.e. a new checking or saving account)
*/
package webhook

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/plaid/plaid-go/v40/plaid"

	"purch/internal/config"
)

// Example payload:
/*
	{
	  "webhook_type": "TRANSACTIONS",
	  "webhook_code": "SYNC_UPDATES_AVAILABLE",
	  "item_id": "wz666MBjYWTp2PDzzggYhM6oWWmBb",
	  "user_id": "usr_9nSp2KuZ2x4JDw",
	  "initial_update_complete": true,
	  "historical_update_complete": false,
	  "environment": "production"
	}
*/
type WebhookPayLoad struct {
	WebhookType              string `json:"webhook_type"`
	WebhookCode              string `json:"webhook_code"`
	ItemId                   string `json:"item_id"`
	UserId                   string `json:"user_id"`
	InitialUpdateComplete    bool   `json:"initial_update_complete"`
	HistoricalUpdateComplete bool   `json:"historical_update_complete"`
	Environment              string `json:"environment"`
}

func GetWebhookServer(config config.Config) *http.Server {
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
