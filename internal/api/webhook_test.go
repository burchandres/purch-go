package api

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

const (
	webhookUrl = "http://localhost:8080/webhook/plaid"
)

func TestWebhookPlaid(t *testing.T) {
	resp := makeRequest(t, client, "POST", webhookUrl, nil)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}