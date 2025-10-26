package api

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"testing"
	"time"

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

func TestRegisterUserEndpoint(t *testing.T) {
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

func TestLoginUser_Success(t *testing.T) {
	// Setup: First register a user
	registerUrl := userServiceUrl + "/register"
	loginUrl := userServiceUrl + "/login"
	test_id := uuid.NewString()[:6]
	test_username := "testuser_" + test_id
	test_password := "testpass"

	userPayload := map[string]any{
		"first_name":  "foo",
		"last_name":   "bar",
		"username":    test_username,
		"password":    test_password,
		"income":      1000,
		"income_rate": "weekly",
	}

	// Register the user
	registerResp := makeRequest(t, client, http.MethodPost, registerUrl, userPayload)
	require.Equal(t, http.StatusCreated, registerResp.StatusCode)

	// Now test login
	loginPayload := map[string]any{
		"username": test_username,
		"password": test_password,
	}

	loginResp := makeRequest(t, client, http.MethodPost, loginUrl, loginPayload)

	// Check status
	assert.Equal(t, http.StatusOK, loginResp.StatusCode)

	// Check that cookie was set
	cookies := loginResp.Cookies()
	require.NotEmpty(t, cookies, "Expected cookies to be set")

	var sessionCookie *http.Cookie
	for _, cookie := range cookies {
		if cookie.Name == "purch_token" { // Use your actual cookie name
			sessionCookie = cookie
			break
		}
	}

	require.NotNil(t, sessionCookie, "Session cookie not found")
	assert.NotEmpty(t, sessionCookie.Value, "Session cookie value should not be empty")

	// Optionally check cookie attributes
	assert.True(t, sessionCookie.HttpOnly, "Cookie should be HttpOnly")
	assert.Equal(t, "/", sessionCookie.Path)
	// assert.True(t, sessionCookie.Secure) // If using HTTPS
}

func TestLoginUser_InvalidPassword(t *testing.T) {
	// Setup: Register a user first
	registerUrl := userServiceUrl + "/register"
	loginUrl := userServiceUrl + "/login"
	test_id := uuid.NewString()[:6]
	test_username := "testuser_" + test_id
	test_password := "testpass"

	userPayload := map[string]any{
		"first_name":  "foo",
		"last_name":   "bar",
		"username":    test_username,
		"password":    test_password,
		"income":      1000,
		"income_rate": "weekly",
	}

	registerResp := makeRequest(t, client, http.MethodPost, registerUrl, userPayload)
	require.Equal(t, http.StatusCreated, registerResp.StatusCode)

	// Try to login with wrong password
	loginPayload := map[string]any{
		"username": test_username,
		"password": "bad",
	}

	loginResp := makeRequest(t, client, http.MethodPost, loginUrl, loginPayload)

	assert.Equal(t, http.StatusUnauthorized, loginResp.StatusCode)

	// No cookie should be set
	cookies := loginResp.Cookies()
	for _, cookie := range cookies {
		if cookie.Name == "purch_token" {
			t.Error("Session cookie should not be set for failed login")
		}
	}
}

func TestLoginUser_NonexistentUser(t *testing.T) {
	loginUrl := userServiceUrl + "/login"

	loginPayload := map[string]any{
		"username": "nonexistent_user_" + uuid.NewString()[:6],
		"password": "somepass",
	}

	loginResp := makeRequest(t, client, http.MethodPost, loginUrl, loginPayload)

	assert.Equal(t, http.StatusUnauthorized, loginResp.StatusCode)

	// No cookie should be set
	cookies := loginResp.Cookies()
	for _, cookie := range cookies {
		if cookie.Name == "purch_token" {
			t.Error("Session cookie should not be set for nonexistent user")
		}
	}
}

func TestLoginUser_MissingCredentials(t *testing.T) {
	loginUrl := userServiceUrl + "/login"

	testCases := []struct {
		name    string
		payload map[string]any
	}{
		{
			name:    "missing username",
			payload: map[string]any{"password": "testpass"},
		},
		{
			name:    "missing password",
			payload: map[string]any{"username": "testuser"},
		},
		{
			name:    "empty payload",
			payload: map[string]any{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			loginResp := makeRequest(t, client, http.MethodPost, loginUrl, tc.payload)
			assert.Equal(t, http.StatusBadRequest, loginResp.StatusCode)
		})
	}
}
