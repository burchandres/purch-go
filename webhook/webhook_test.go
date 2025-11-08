package webhook

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var client *http.Client

const (
	webhookUrl = "http://localhost:8081/webhook/plaid"
)

func makeRequest(
	t *testing.T,
	client *http.Client,
	requestMethod string,
	url string,
	payload map[string]any,
) *http.Response {
	body, err := json.Marshal(payload)
	require.NoError(t, err)

	t.Logf("Making %s request to %s", requestMethod, url)
	t.Logf("Request body: %s", string(body))

	req, err := http.NewRequestWithContext(context.Background(), requestMethod, url, bytes.NewBuffer(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	require.NoError(t, err)

	return resp
}

func TestMain(m *testing.M) {
	client = &http.Client{
		Timeout: 5 * time.Second,
	}
	exitVal := m.Run()
	client.CloseIdleConnections()
	os.Exit(exitVal)
}

func TestWebhookPlaid(t *testing.T) {
	resp := makeRequest(t, client, "POST", webhookUrl, nil)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}
