package tasks

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"github.com/plaid/plaid-go/v40/plaid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"

	"purch/internal/config"
	"purch/internal/database"
	"purch/internal/utils"
)

const (
	FIRST_PLATYPUS_BANK = "ins_109508"
	RFC3339Nano         = "2006-01-02T15:04:05.999999999Z07:00"
)

func getTestUser() database.User {
	income := 100000.0
	incomeRate := "annual"
	password, _ := bcrypt.GenerateFromPassword([]byte("testpass"), bcrypt.DefaultCost)
	return database.User{
		FirstName:  "foo",
		LastName:   "bar",
		Username:   "testuser_" + uuid.NewString()[:8],
		Password:   string(password),
		Income:     &income,
		IncomeRate: &incomeRate,
	}
}

func storeTestUser(t *testing.T) database.User {
	testUser := getTestUser()
	err := database.StoreUser(context.Background(), testUser)
	if err != nil {
		t.Logf("test user: %v", testUser)
		t.Logf("error from storing user: %v", err)
	}
	require.Nil(t, err)
	// pull the user from database to get the id
	testUser, err = database.GetUserByUsername(context.Background(), testUser.Username)
	if err != nil {
		t.Logf("pulled test user after storing: %v", testUser)
	}
	require.Nil(t, err)
	return testUser
}

func createSandboxItem(t *testing.T, ctx context.Context, client *plaid.APIClient, institutionID string, products []plaid.Products) plaid.ItemPublicTokenExchangeResponse {
	// good transactions test user credentials -- taken from: https://plaid.com/docs/sandbox/test-credentials/
	username := "user_transactions_dynamic"
	usernameOverride := plaid.NullableString{}
	usernameOverride.Set(&username)
	password := "any-nonempty-password"
	passwordOverride := plaid.NullableString{}
	passwordOverride.Set(&password)
	overrideUserCreds := plaid.SandboxPublicTokenCreateRequestOptions{
		OverrideUsername: usernameOverride,
		OverridePassword: passwordOverride,
	}
	createRequest := plaid.NewSandboxPublicTokenCreateRequest(
		institutionID,
		products,
	)
	createRequest.SetOptions(overrideUserCreds)
	// generate a sandbox public_token
	sandboxPublicTokenResp, httpResp, err := client.PlaidApi.
		SandboxPublicTokenCreate(ctx).
		SandboxPublicTokenCreateRequest(*createRequest).
		Execute()

	if err != nil {
		t.Logf("error in getting public token response: %v", err)
		if httpResp != nil {
			t.Logf("http response: %v", httpResp)
		}
		t.Logf("Institution ID: %s", institutionID)
		t.Logf("Products: %v", products)
		t.Logf("Plaid client config: %v", client.GetConfig())
	}
	require.NoError(t, err)

	// exchange the public_token for an access_token
	exchangePublicTokenResp, _, err := client.PlaidApi.ItemPublicTokenExchange(ctx).ItemPublicTokenExchangeRequest(
		*plaid.NewItemPublicTokenExchangeRequest(sandboxPublicTokenResp.GetPublicToken()),
	).Execute()

	assert.NotNil(t, exchangePublicTokenResp)
	require.NoError(t, err)
	assert.NotEqual(t, "", exchangePublicTokenResp.AccessToken)
	assert.NotEqual(t, "", exchangePublicTokenResp.ItemId)

	return exchangePublicTokenResp
}

func TestMain(m *testing.M) {
	// Load env before running tests
	if err := godotenv.Load("../../.env"); err != nil {
		panic(err)
	}
	if err := database.Init("postgres://postgres:password@localhost:5432/purch?sslmode=disable"); err != nil {
		panic(err)
	}
	os.Exit(m.Run())
}

func TestSyncItem(t *testing.T) {
	plaidClient := utils.GetPlaidClient()
	config := config.GetConfig()
	ctx := context.Background()
	// create sandbox item
	publicTokenExchangeResp := createSandboxItem(t, ctx, plaidClient, FIRST_PLATYPUS_BANK, config.GetPlaidProducts())
	// store a test user unique to this test
	testUser := storeTestUser(t)
	// test the task SyncItem
	itemID := publicTokenExchangeResp.ItemId
	accessToken := publicTokenExchangeResp.AccessToken
	err := SyncItem(ctx, testUser.ID, itemID, accessToken)
	if err != nil {
		t.Logf("error from SyncItem task in TestSyncItem: %v", err)
	}
	require.Nil(t, err)
	// pull from the database and make sure it is persisted
	items, err := database.GetUserItems(ctx, testUser.ID)
	if err != nil {
		t.Logf("error from pulling user items in database: %v", err)
	}
	require.Nil(t, err)
	assert.Equal(t, 1, len(items))

	t.Cleanup(func() {
		// only need to delete the user as postgres is conigured to perform cascading deletes
		database.DeleteUser(ctx, testUser)
	})
}

func TestSyncAccounts(t *testing.T) {
	// setup
	plaidClient := utils.GetPlaidClient()
	config := config.GetConfig()
	ctx := context.Background()
	// create sandbox item
	publicTokenExchangeResp := createSandboxItem(t, ctx, plaidClient, FIRST_PLATYPUS_BANK, config.GetPlaidProducts())
	// store a test user unique to this test
	testUser := storeTestUser(t)
	// run SyncItem
	itemID := publicTokenExchangeResp.ItemId
	accessToken := publicTokenExchangeResp.AccessToken
	err := SyncItem(ctx, testUser.ID, itemID, accessToken)
	if err != nil {
		t.Logf("error from SyncItem task in TestSyncAccounts: %v", err)
	}
	require.Nil(t, err)
	// now get all accounts and persist them
	err = SyncAccounts(ctx, itemID, accessToken)
	if err != nil {
		t.Logf("error from SyncAccounts task in TestSyncAccounts: %v", err)
	}
	require.Nil(t, err)
	accounts, err := database.GetUserAccounts(ctx, testUser.ID)
	assert.NotEmpty(t, accounts)
	t.Cleanup(func() {
		// only need to delete the user as postgres is conigured to perform cascading deletes
		database.DeleteUser(ctx, testUser)
	})
}

func TestSyncTransactions(t *testing.T) {
	// setup
	plaidClient := utils.GetPlaidClient()
	config := config.GetConfig()
	ctx := context.Background()
	// create sandbox item
	publicTokenExchangeResp := createSandboxItem(t, ctx, plaidClient, FIRST_PLATYPUS_BANK, config.GetPlaidProducts())
	// store a test user unique to this test
	testUser := storeTestUser(t)
	// run SyncItem
	itemID := publicTokenExchangeResp.ItemId
	accessToken := publicTokenExchangeResp.AccessToken
	err := SyncItem(ctx, testUser.ID, itemID, accessToken)
	if err != nil {
		t.Logf("error from SyncItem task in TestSyncTransactions: %v", err)
	}
	require.Nil(t, err)
	// run SyncAccount
	err = SyncAccounts(ctx, itemID, accessToken)
	if err != nil {
		t.Logf("error from SyncAccounts task in TestSyncTransactions: %v", err)
	}
	require.Nil(t, err)
	// keep asking for transactions until we get some
	var transactions []database.Transaction
	retry := true
	for retry {
		// now sync all transactions
		err = SyncTransactions(ctx, itemID, accessToken, "")
		if err != nil {
			t.Logf("error from SyncTransactions task in TestSyncTransactions: %v", err)
		}
		require.Nil(t, err)
		// make sure we actually get transactions
		var count int
		transactions, count, err = database.GetUserTransactions(ctx, testUser.ID)
		if count == 0 {
			t.Logf("no transactions, sleeping for 3s then trying again...")
			time.Sleep(3 * time.Second)
		} else {
			retry = false
		}
	}
	if err != nil {
		t.Logf("error pulling transactions persisted in SyncTransactions task in TestSyncTransactions: %v", err)
	}
	assert.Nil(t, err)
	assert.NotEmpty(t, transactions)
	// t.Cleanup(func() {
	// 	// delete user and everything follows due to cascading deletes
	// 	database.DeleteUser(ctx, testUser)
	// })
}
