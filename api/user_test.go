package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"os"
	"testing"
	"time"
	"io"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const userServiceUrl = "http://localhost:8080/user"

var client *http.Client

func parseResponse(t *testing.T, w *http.Response) map[string]any {
	// Read the entire response body
		body, err := io.ReadAll(w.Body)
		require.NoError(t, err)
		defer w.Body.Close()
		
		// DEBUG: Print the raw response
		t.Logf("Response status: %d", w.StatusCode)
		t.Logf("Response body: %s", string(body))
		
		// convert body into payload
		var response map[string]any
		err = json.Unmarshal(body, &response)
		require.NoError(t, err)
		
		return response
}

func makeRequest(
	t *testing.T,
	client *http.Client,
	requestMethod string,
	url string,
	payload map[string]any,
) *http.Response {
	body, err := json.Marshal(payload)
	require.NoError(t, err)

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

func TestRegisterUser(t *testing.T) {
	// test setup
	registerUrl := userServiceUrl + "/register"
	test_id := uuid.NewString()[:6]
	// user register request
	userPayload := map[string]any{
		"first_name":  "abc",
		"last_name":   "def",
		"username":    "testuser_" + test_id,
		"password":    "testpass",
		"income":      1000.0,
		"income_rate": "weekly",
	}
	// make register request
	resp := makeRequest(t, client, http.MethodPost, registerUrl, userPayload)
	// make sure it was successful
	assert.Equal(t, http.StatusCreated, resp.StatusCode)
	// check that the returned user is what we made the request with
	respPayload := parseResponse(t, resp)
	assert.Equal(t, userPayload["first_name"], respPayload["first_name"])
	assert.Equal(t, userPayload["last_name"], respPayload["last_name"])
	assert.Equal(t, userPayload["username"], respPayload["username"])
	assert.Equal(t, userPayload["income"], respPayload["income"])
	assert.Equal(t, userPayload["income_rate"], respPayload["income_rate"])
	// just check password exists
	_, ok := respPayload["password"]
	assert.True(t, ok)
}
