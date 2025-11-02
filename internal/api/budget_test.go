package api

import (
	"encoding/json"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"

	"purch/internal/database"
)

// login credentials
const (
	testuser         = "testuser"
	testpass         = "testpass"
	budgetServiceUrl = "http://localhost:8080/budget"
)

// These tests assume postgres is seeded with the dummy data in ./initdb/01_init.sql

func TestGetCategories(t *testing.T) {
	cookie := loginAndGetCookie(t, testuser, testpass)
	// hit /budget/categories
	resp := makeAuthenticatedRequest(t, client, "GET", budgetServiceUrl+"/categories", nil, cookie)
	// parse the response
	body, err := io.ReadAll(resp.Body)
	var categories []database.Category
	err = json.Unmarshal(body, &categories)
	assert.NoError(t, err)
	defer resp.Body.Close()
	// check we get expected categories response for test data
	assert.Equal(t, 3, len(categories))
}

func TestGetItems(t *testing.T) {
	cookie := loginAndGetCookie(t, testuser, testpass)
	// hit /budget/categories
	resp := makeAuthenticatedRequest(t, client, "GET", budgetServiceUrl+"/items", nil, cookie)
	// parse the response
	body, err := io.ReadAll(resp.Body)
	var items []database.Item
	err = json.Unmarshal(body, &items)
	assert.NoError(t, err)
	defer resp.Body.Close()
	// check we get expected items response for test data
	assert.Equal(t, 2, len(items))
}

func TestGetAccounts(t *testing.T) {
	cookie := loginAndGetCookie(t, testuser, testpass)
	// hit /budget/categories
	resp := makeAuthenticatedRequest(t, client, "GET", budgetServiceUrl+"/accounts", nil, cookie)
	// parse the response
	body, err := io.ReadAll(resp.Body)
	var accounts []database.Account
	err = json.Unmarshal(body, &accounts)
	assert.NoError(t, err)
	defer resp.Body.Close()
	// check we get expected accounts response for test data
	assert.Equal(t, 2, len(accounts))
}

func TestGetTransactions(t *testing.T) {
	cookie := loginAndGetCookie(t, testuser, testpass)
	// hit /budget/categories
	resp := makeAuthenticatedRequest(t, client, "GET", budgetServiceUrl+"/transactions", nil, cookie)
	// parse the response
	body, err := io.ReadAll(resp.Body)
	var transactions []database.Transaction
	err = json.Unmarshal(body, &transactions)
	assert.NoError(t, err)
	defer resp.Body.Close()
	// check we get expected transactions response for test data
	assert.Equal(t, 3, len(transactions))
}
