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

// Helper to register a new user, login, and return credentials + cookie
func registerLoginAndGetCookie(t *testing.T) (username, password string, cookie *http.Cookie) {
	test_id := uuid.NewString()[:6]
	username = "testuser_" + test_id
	password = "testpass123"
	
	// Register the user
	registerUrl := userServiceUrl + "/register"
	userPayload := map[string]any{
		"first_name":  "Test",
		"last_name":   "User",
		"username":    username,
		"password":    password,
		"income":      1000,
		"income_rate": "weekly",
	}
	
	registerResp := makeRequest(t, client, http.MethodPost, registerUrl, userPayload)
	require.Equal(t, http.StatusCreated, registerResp.StatusCode, "Failed to register test user")
	
	// Login and get cookie
	loginUrl := userServiceUrl + "/login"
	loginPayload := map[string]any{
		"username": username,
		"password": password,
	}
	
	loginResp := makeRequest(t, client, http.MethodPost, loginUrl, loginPayload)
	require.Equal(t, http.StatusOK, loginResp.StatusCode, "Failed to login test user")
	
	// Extract the purch_token cookie
	var sessionCookie *http.Cookie
	for _, cookie := range loginResp.Cookies() {
		if cookie.Name == "purch_token" {
			sessionCookie = cookie
			break
		}
	}
	require.NotNil(t, sessionCookie, "purch_token cookie should be set")
	
	// Cleanup
	t.Cleanup(func() {
		// Optional: delete user from database
		// database.DeleteUserByUsername(context.Background(), username)
	})
	
	return username, password, sessionCookie
}

func registerLoginWithData(t *testing.T, userData map[string]any) (username, password string, cookie *http.Cookie) {
	test_id := uuid.NewString()[:8]
	username = "testuser_" + test_id
	password = "testpass123"
	
	// Default payload
	registerUrl := userServiceUrl + "/register"
	userPayload := map[string]any{
		"first_name":  "Test",
		"last_name":   "User",
		"username":    username,
		"password":    password,
		"income":      1000,
		"income_rate": "weekly",
	}
	
	// Override with custom data
	for k, v := range userData {
		userPayload[k] = v
	}
	
	registerResp := makeRequest(t, client, http.MethodPost, registerUrl, userPayload)
	require.Equal(t, http.StatusCreated, registerResp.StatusCode)
	
	_, _, cookie = registerLoginAndGetCookie(t)
	
	t.Cleanup(func() {
		// database.DeleteUserByUsername(context.Background(), username)
	})
	
	return username, password, cookie
}

func loginAndGetCookie(t *testing.T, username, password string) *http.Cookie {
	// Login and get cookie
	loginUrl := userServiceUrl + "/login"
	loginPayload := map[string]any{
		"username": username,
		"password": password,
	}
	
	loginResp := makeRequest(t, client, http.MethodPost, loginUrl, loginPayload)
	require.Equal(t, http.StatusOK, loginResp.StatusCode, "Failed to login test user")
	
	// Extract the purch_token cookie
	var sessionCookie *http.Cookie
	for _, cookie := range loginResp.Cookies() {
		if cookie.Name == "purch_token" {
			sessionCookie = cookie
			break
		}
	}
	require.NotNil(t, sessionCookie, "purch_token cookie should be set")
	
	// Cleanup
	t.Cleanup(func() {
		// Optional: delete user from database
		// database.DeleteUserByUsername(context.Background(), username)
	})
	
	return sessionCookie
}

// Helper to make authenticated requests
func makeAuthenticatedRequest(
	t *testing.T,
	client *http.Client,
	method string,
	url string,
	payload map[string]any,
	cookie *http.Cookie,
) *http.Response {
	body, err := json.Marshal(payload)
	require.NoError(t, err)
	
	req, err := http.NewRequestWithContext(context.Background(), method, url, bytes.NewBuffer(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(cookie) // Add the session cookie
	
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

func TestGetUserInfo_Success(t *testing.T) {
	// Register and login to get authenticated cookie
	username, _, cookie := registerLoginAndGetCookie(t)
	
	// Make authenticated request to /user/info
	userInfoUrl := userServiceUrl + "/info"
	resp := makeAuthenticatedRequest(t, client, http.MethodGet, userInfoUrl, nil, cookie)
	
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	
	// Parse and validate response
	respPayload := parseResponse(t, resp)
	assert.Equal(t, username, respPayload["username"])
	assert.Equal(t, "Test", respPayload["first_name"])
	assert.Equal(t, "User", respPayload["last_name"])
	assert.Equal(t, float64(1000), respPayload["income"])
	assert.Equal(t, "weekly", respPayload["income_rate"])
	assert.NotEmpty(t, respPayload["id"], "User ID should be present")
}

func TestGetUserInfo_Unauthorized_NoCookie(t *testing.T) {
	// Try to access /user/info without authentication
	userInfoUrl := userServiceUrl + "/info"
	resp := makeRequest(t, client, http.MethodGet, userInfoUrl, nil)
	
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestGetUserInfo_Unauthorized_InvalidToken(t *testing.T) {
	// Try with an invalid JWT token
	userInfoUrl := userServiceUrl + "/info"
	
	invalidCookie := &http.Cookie{
		Name:  "purch_token",
		Value: "invalid.jwt.token",
	}
	
	resp := makeAuthenticatedRequest(t, client, http.MethodGet, userInfoUrl, nil, invalidCookie)
	
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestLogout_Success(t *testing.T) {
	// Register and login to get authenticated cookie
	_, _, cookie := registerLoginAndGetCookie(t)
	
	// Verify we can access protected endpoint before logout
	userInfoUrl := userServiceUrl + "/info"
	infoResp := makeAuthenticatedRequest(t, client, http.MethodGet, userInfoUrl, nil, cookie)
	assert.Equal(t, http.StatusOK, infoResp.StatusCode, "Should be able to access user info before logout")
	
	// Logout
	logoutUrl := userServiceUrl + "/logout"
	logoutResp := makeAuthenticatedRequest(t, client, http.MethodPost, logoutUrl, nil, cookie)
	
	assert.Equal(t, http.StatusOK, logoutResp.StatusCode)
	
	// Verify response message
	respPayload := parseResponse(t, logoutResp)
	assert.Contains(t, respPayload["message"], "logout successful")
	
	// Verify cookie is cleared (should have MaxAge=-1 or be empty)
	var clearedCookie *http.Cookie
	for _, c := range logoutResp.Cookies() {
		if c.Name == "purch_token" {
			clearedCookie = c
			break
		}
	}
	
	require.NotNil(t, clearedCookie, "purch_token cookie should be present in logout response")
	assert.Equal(t, "", clearedCookie.Value, "Cookie value should be empty")
	assert.Equal(t, -1, clearedCookie.MaxAge, "Cookie MaxAge should be -1 to delete it")
	
	// TODO: uncomment when tokens get blacklisted
	// // Try to access protected endpoint with old cookie (should fail)
	// infoRespAfterLogout := makeAuthenticatedRequest(t, client, http.MethodGet, userInfoUrl, nil, cookie)
	// assert.Equal(t, http.StatusUnauthorized, infoRespAfterLogout.StatusCode, "Should not be able to access user info after logout")
}

func TestLogout_WithoutAuthentication(t *testing.T) {
	// Try to logout without being logged in
	logoutUrl := userServiceUrl + "/logout"
	resp := makeRequest(t, client, http.MethodPost, logoutUrl, nil)
	
	// Depending on your middleware, this might be 401 or 200
	// If logout doesn't require auth, it should return 200
	// If it requires auth middleware, it should return 401
	assert.NotEqual(t, http.StatusInternalServerError, resp.StatusCode)
}

func TestLogout_InvalidToken(t *testing.T) {
	logoutUrl := userServiceUrl + "/logout"
	
	invalidCookie := &http.Cookie{
		Name:  "purch_token",
		Value: "invalid.jwt.token",
	}
	
	resp := makeAuthenticatedRequest(t, client, http.MethodPost, logoutUrl, nil, invalidCookie)
	
	// Should either return 401 (if auth middleware blocks) or 200 (if logout always succeeds)
	assert.NotEqual(t, http.StatusInternalServerError, resp.StatusCode)
}

func TestLogout_MultipleTimes(t *testing.T) {
	// Register and login
	_, _, cookie := registerLoginAndGetCookie(t)
	
	logoutUrl := userServiceUrl + "/logout"
	
	// First logout - should succeed
	logoutResp1 := makeAuthenticatedRequest(t, client, http.MethodPost, logoutUrl, nil, cookie)
	assert.Equal(t, http.StatusOK, logoutResp1.StatusCode)
	
	// Second logout with same (now invalid) cookie
	// Should either return 401 or 200 depending on implementation
	logoutResp2 := makeAuthenticatedRequest(t, client, http.MethodPost, logoutUrl, nil, cookie)
	assert.NotEqual(t, http.StatusInternalServerError, logoutResp2.StatusCode)
}

func TestLogout_ThenLoginAgain(t *testing.T) {
	// Register and login
	username, password, cookie := registerLoginAndGetCookie(t)
	
	// Logout
	logoutUrl := userServiceUrl + "/logout"
	logoutResp := makeAuthenticatedRequest(t, client, http.MethodPost, logoutUrl, nil, cookie)
	assert.Equal(t, http.StatusOK, logoutResp.StatusCode)
	
	// Login again with same credentials
	newCookie := loginAndGetCookie(t, username, password)
	assert.NotNil(t, newCookie)
	assert.NotEmpty(t, newCookie.Value)
	
	// Verify can access protected endpoint with new cookie
	userInfoUrl := userServiceUrl + "/info"
	infoResp := makeAuthenticatedRequest(t, client, http.MethodGet, userInfoUrl, nil, newCookie)
	assert.Equal(t, http.StatusOK, infoResp.StatusCode)
}

func TestLogout_DifferentUsers(t *testing.T) {
	// Create two users
	_, _, cookie1 := registerLoginAndGetCookie(t)
	username2, _, cookie2 := registerLoginAndGetCookie(t)
	
	logoutUrl := userServiceUrl + "/logout"
	
	// User 1 logs out
	logoutResp1 := makeAuthenticatedRequest(t, client, http.MethodPost, logoutUrl, nil, cookie1)
	assert.Equal(t, http.StatusOK, logoutResp1.StatusCode)
	
	// TODO: uncomment when tokens get blacklisted
	// // User 1 can no longer access protected endpoints
	userInfoUrl := userServiceUrl + "/info"
	// infoResp1 := makeAuthenticatedRequest(t, client, http.MethodGet, userInfoUrl, nil, cookie1)
	// assert.Equal(t, http.StatusUnauthorized, infoResp1.StatusCode)
	
	// User 2 should still be able to access protected endpoints
	infoResp2 := makeAuthenticatedRequest(t, client, http.MethodGet, userInfoUrl, nil, cookie2)
	assert.Equal(t, http.StatusOK, infoResp2.StatusCode)
	
	respPayload := parseResponse(t, infoResp2)
	assert.Equal(t, username2, respPayload["username"])
}

func TestLogout_CookieAttributes(t *testing.T) {
	// Register and login
	_, _, cookie := registerLoginAndGetCookie(t)
	
	// Logout
	logoutUrl := userServiceUrl + "/logout"
	logoutResp := makeAuthenticatedRequest(t, client, http.MethodPost, logoutUrl, nil, cookie)
	
	assert.Equal(t, http.StatusOK, logoutResp.StatusCode)
	
	// Check cookie attributes are correct
	var clearedCookie *http.Cookie
	for _, c := range logoutResp.Cookies() {
		if c.Name == "purch_token" {
			clearedCookie = c
			break
		}
	}
	
	require.NotNil(t, clearedCookie)
	assert.Equal(t, "purch_token", clearedCookie.Name)
	assert.Equal(t, "", clearedCookie.Value)
	assert.Equal(t, -1, clearedCookie.MaxAge)
	assert.Equal(t, "/", clearedCookie.Path)
	assert.Equal(t, "localhost", clearedCookie.Domain)
	assert.True(t, clearedCookie.HttpOnly, "Cookie should be HttpOnly")
	assert.False(t, clearedCookie.Secure, "Cookie Secure flag should match login")
}

// TODO: uncomment when token blacklisting is implemented
// func TestLogout_VerifyTokenInvalidated(t *testing.T) {
// 	// Register and login
// 	_, _, cookie := registerLoginAndGetCookie(t)
	
// 	// Store the original token value
// 	originalToken := cookie.Value
// 	assert.NotEmpty(t, originalToken)
	
// 	// Logout
// 	logoutUrl := userServiceUrl + "/logout"
// 	logoutResp := makeAuthenticatedRequest(t, client, http.MethodPost, logoutUrl, nil, cookie)
// 	assert.Equal(t, http.StatusOK, logoutResp.StatusCode)
	
// 	// Try to use the original token after logout
// 	userInfoUrl := userServiceUrl + "/info"
	
// 	// Recreate cookie with original token
// 	oldCookie := &http.Cookie{
// 		Name:  "purch_token",
// 		Value: originalToken,
// 	}
	
	
// 	infoResp := makeAuthenticatedRequest(t, client, http.MethodGet, userInfoUrl, nil, oldCookie)
// 	assert.Equal(t, http.StatusUnauthorized, infoResp.StatusCode, "Old token should not work after logout")
// }

func TestDeleteUser_Success(t *testing.T) {
	username, _, cookie := registerLoginAndGetCookie(t)
	
	// Verify user exists before deletion
	userInfoUrl := userServiceUrl + "/info"
	infoResp := makeAuthenticatedRequest(t, client, http.MethodGet, userInfoUrl, nil, cookie)
	assert.Equal(t, http.StatusOK, infoResp.StatusCode)
	
	// Delete user
	deleteUrl := userServiceUrl + "/delete"
	deleteResp := makeAuthenticatedRequest(t, client, http.MethodDelete, deleteUrl, nil, cookie)
	
	assert.Equal(t, http.StatusOK, deleteResp.StatusCode)
	
	// Verify response message
	respPayload := parseResponse(t, deleteResp)
	assert.Contains(t, respPayload["message"], "user deleted")
	assert.Contains(t, respPayload["message"], "cookie session cleared")
	
	// Verify cookie is cleared
	var clearedCookie *http.Cookie
	for _, c := range deleteResp.Cookies() {
		if c.Name == "purch_token" {
			clearedCookie = c
			break
		}
	}
	require.NotNil(t, clearedCookie, "Cookie should be cleared")
	assert.Equal(t, "", clearedCookie.Value)
	assert.Equal(t, -1, clearedCookie.MaxAge)
	
	// // Try to access user info with old cookie (should fail)
	// infoRespAfter := makeAuthenticatedRequest(t, client, http.MethodGet, userInfoUrl, nil, cookie)
	// assert.Equal(t, http.StatusUnauthorized, infoRespAfter.StatusCode)
	
	// Try to login with deleted user credentials (should fail)
	loginUrl := userServiceUrl + "/login"
	loginPayload := map[string]any{
		"username": username,
		"password": "testpass123",
	}
	loginResp := makeRequest(t, client, http.MethodPost, loginUrl, loginPayload)
	assert.Equal(t, http.StatusUnauthorized, loginResp.StatusCode)
}

func TestDeleteUser_Unauthorized_NoCookie(t *testing.T) {
	// Try to delete without authentication
	deleteUrl := userServiceUrl + "/delete"
	resp := makeRequest(t, client, http.MethodDelete, deleteUrl, nil)
	
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestDeleteUser_Unauthorized_InvalidToken(t *testing.T) {
	deleteUrl := userServiceUrl + "/delete"
	
	invalidCookie := &http.Cookie{
		Name:  "purch_token",
		Value: "invalid.jwt.token",
	}
	
	resp := makeAuthenticatedRequest(t, client, http.MethodDelete, deleteUrl, nil, invalidCookie)
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

// TOOD: blacklist expired
// func TestDeleteUser_CannotAccessAfterDeletion(t *testing.T) {
// 	_, _, cookie := registerLoginAndGetCookie(t)
	
// 	// Delete user
// 	deleteUrl := userServiceUrl + "/delete"
// 	deleteResp := makeAuthenticatedRequest(t, client, http.MethodDelete, deleteUrl, nil, cookie)
// 	assert.Equal(t, http.StatusOK, deleteResp.StatusCode)
	
// 	// Try to access various endpoints with old cookie
// 	endpoints := []string{
// 		userServiceUrl + "/info",
// 		userServiceUrl + "/update",
// 	}
	
// 	for _, endpoint := range endpoints {
// 		resp := makeAuthenticatedRequest(t, client, http.MethodGet, endpoint, nil, cookie)
// 		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode, "Should not access %s after deletion", endpoint)
// 	}
// }

func TestDeleteUser_CannotDeleteTwice(t *testing.T) {
	_, _, cookie := registerLoginAndGetCookie(t)
	
	deleteUrl := userServiceUrl + "/delete"
	
	// First deletion - should succeed
	deleteResp1 := makeAuthenticatedRequest(t, client, http.MethodDelete, deleteUrl, nil, cookie)
	assert.Equal(t, http.StatusOK, deleteResp1.StatusCode)
	
	// Second deletion with same cookie - should fail
	deleteResp2 := makeAuthenticatedRequest(t, client, http.MethodDelete, deleteUrl, nil, cookie)
	assert.Equal(t, http.StatusUnauthorized, deleteResp2.StatusCode)
}

func TestDeleteUser_DifferentUsers(t *testing.T) {
	// Create two users
	username1, _, cookie1 := registerLoginAndGetCookie(t)
	username2, password2, cookie2 := registerLoginAndGetCookie(t)
	
	// User 1 deletes their account
	deleteUrl := userServiceUrl + "/delete"
	deleteResp1 := makeAuthenticatedRequest(t, client, http.MethodDelete, deleteUrl, nil, cookie1)
	assert.Equal(t, http.StatusOK, deleteResp1.StatusCode)
	
	// User 1 can no longer access their info
	userInfoUrl := userServiceUrl + "/info"
	infoResp1 := makeAuthenticatedRequest(t, client, http.MethodGet, userInfoUrl, nil, cookie1)
	assert.Equal(t, http.StatusUnauthorized, infoResp1.StatusCode)
	
	// User 2 should still exist and be able to access their info
	infoResp2 := makeAuthenticatedRequest(t, client, http.MethodGet, userInfoUrl, nil, cookie2)
	assert.Equal(t, http.StatusOK, infoResp2.StatusCode)
	
	respPayload2 := parseResponse(t, infoResp2)
	assert.Equal(t, username2, respPayload2["username"])
	
	// User 1 cannot login anymore
	loginResp1 := makeRequest(t, client, http.MethodPost, userServiceUrl+"/login", map[string]any{
		"username": username1,
		"password": "testpass123",
	})
	assert.NotEqual(t, http.StatusOK, loginResp1.StatusCode)
	
	// User 2 can still login
	loginResp2 := makeRequest(t, client, http.MethodPost, userServiceUrl+"/login", map[string]any{
		"username": username2,
		"password": password2,
	})
	assert.Equal(t, http.StatusOK, loginResp2.StatusCode)
}

func TestDeleteUser_WithUserData(t *testing.T) {
	// Create user with custom data
	username, password, cookie := registerLoginWithData(t, map[string]any{
		"first_name":  "DeleteMe",
		"last_name":   "TestUser",
		"income":      50000,
		"income_rate": "yearly",
	})
	
	// Verify user exists with data
	userInfoUrl := userServiceUrl + "/info"
	infoResp := makeAuthenticatedRequest(t, client, http.MethodGet, userInfoUrl, nil, cookie)
	assert.Equal(t, http.StatusOK, infoResp.StatusCode)
	
	infoPayload := parseResponse(t, infoResp)
	assert.Equal(t, "DeleteMe", infoPayload["first_name"])
	assert.Equal(t, float64(50000), infoPayload["income"])
	
	// Delete user
	deleteUrl := userServiceUrl + "/delete"
	deleteResp := makeAuthenticatedRequest(t, client, http.MethodDelete, deleteUrl, nil, cookie)
	assert.Equal(t, http.StatusOK, deleteResp.StatusCode)
	
	// TODO: uncomment after token blacklisting is implemented
	// // User should be gone
	// loginResp := makeRequest(t, client, http.MethodPost, userServiceUrl+"/login", map[string]any{
	// 	"username": username,
	// 	"password": password,
	// })
	// assert.NotEqual(t, http.StatusOK, loginResp.StatusCode)
}

func TestDeleteUser_ThenRegisterSameUsername(t *testing.T) {
	// Register and delete user
	username, password, cookie := registerLoginAndGetCookie(t)
	
	deleteUrl := userServiceUrl + "/delete"
	deleteResp := makeAuthenticatedRequest(t, client, http.MethodDelete, deleteUrl, nil, cookie)
	assert.Equal(t, http.StatusOK, deleteResp.StatusCode)
	
	// Register a new user with the same username
	registerUrl := userServiceUrl + "/register"
	newUserPayload := map[string]any{
		"first_name":  "NewUser",
		"last_name":   "SameName",
		"username":    username, // Same username as deleted user
		"password":    password,
		"income":      2000,
		"income_rate": "monthly",
	}
	
	registerResp := makeRequest(t, client, http.MethodPost, registerUrl, newUserPayload)
	assert.Equal(t, http.StatusCreated, registerResp.StatusCode, "Should be able to reuse username after deletion")
	
	// Login with new user
	newCookie := loginAndGetCookie(t, username, password)
	assert.NotNil(t, newCookie)
	
	// Verify it's a new user (different data)
	userInfoUrl := userServiceUrl + "/info"
	infoResp := makeAuthenticatedRequest(t, client, http.MethodGet, userInfoUrl, nil, newCookie)
	infoPayload := parseResponse(t, infoResp)
	
	assert.Equal(t, username, infoPayload["username"])
	assert.Equal(t, "NewUser", infoPayload["first_name"])
	assert.Equal(t, float64(2000), infoPayload["income"])
}

func TestDeleteUser_CascadeDelete(t *testing.T) {
	// This test assumes you have related data (categories, items)
	// that should be deleted when user is deleted
	
	_, _, cookie := registerLoginAndGetCookie(t)
	
	// TODO: Create some categories/items for the user
	// createCategory(t, cookie, ...)
	// createItem(t, cookie, ...)
	
	// Delete user
	deleteUrl := userServiceUrl + "/delete"
	deleteResp := makeAuthenticatedRequest(t, client, http.MethodDelete, deleteUrl, nil, cookie)
	assert.Equal(t, http.StatusOK, deleteResp.StatusCode)
	
	// TODO: Verify categories/items are also deleted
	// This depends on your database schema and cascade delete rules
}

func TestDeleteUser_AfterUpdate(t *testing.T) {
	username, password, cookie := registerLoginAndGetCookie(t)
	
	// Update user first
	updateUrl := userServiceUrl + "/update"
	updatePayload := map[string]any{
		"first_name": "Updated",
		"income":     99999,
	}
	updateResp := makeAuthenticatedRequest(t, client, http.MethodPut, updateUrl, updatePayload, cookie)
	assert.Equal(t, http.StatusOK, updateResp.StatusCode)
	
	// Verify update worked
	userInfoUrl := userServiceUrl + "/info"
	infoResp := makeAuthenticatedRequest(t, client, http.MethodGet, userInfoUrl, nil, cookie)
	infoPayload := parseResponse(t, infoResp)
	assert.Equal(t, "Updated", infoPayload["first_name"])
	
	// Delete user
	deleteUrl := userServiceUrl + "/delete"
	deleteResp := makeAuthenticatedRequest(t, client, http.MethodDelete, deleteUrl, nil, cookie)
	assert.Equal(t, http.StatusOK, deleteResp.StatusCode)
	
	// User should be gone
	loginResp := makeRequest(t, client, http.MethodPost, userServiceUrl+"/login", map[string]any{
		"username": username,
		"password": password,
	})
	assert.NotEqual(t, http.StatusOK, loginResp.StatusCode)
}

func TestDeleteUser_ResponseFormat(t *testing.T) {
	_, _, cookie := registerLoginAndGetCookie(t)
	
	deleteUrl := userServiceUrl + "/delete"
	deleteResp := makeAuthenticatedRequest(t, client, http.MethodDelete, deleteUrl, nil, cookie)
	
	assert.Equal(t, http.StatusOK, deleteResp.StatusCode)
	
	respPayload := parseResponse(t, deleteResp)
	
	// Verify response structure
	message, ok := respPayload["message"].(string)
	require.True(t, ok, "Response should have a 'message' field")
	assert.NotEmpty(t, message)
	assert.Contains(t, message, "deleted")
}