package api

import (
	"io"
	"testing"
	"encoding/json"

	"github.com/stretchr/testify/assert"
	// "github.com/stretchr/testify/require"

	"purch/database"
)

// login credentials
const (
	testuser = "testuser"
	testpass = "testpass"
	budgetServiceUrl = "http://localhost:8080/budget"
)

// These tests assume postgres is seeded with the dummy data in ./initdb/01_init.sql

func TestGetCategories(t *testing.T) {
	cookie := loginAndGetCookie(t, testuser, testpass)
	// hit /budget/categories
	resp := makeAuthenticatedRequest(t, client, "GET", budgetServiceUrl + "/categories", nil, cookie)
	// parse the response
	body, err := io.ReadAll(resp.Body)
	var categories []database.Category
	err = json.Unmarshal(body, &categories)
	assert.NoError(t, err)
	defer resp.Body.Close()
	// check we get expected categories response for test data
	assert.Equal(t, 3, len(categories))
}