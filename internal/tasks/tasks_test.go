package tasks

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/plaid/plaid-go/v40/plaid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/joho/godotenv"
	"golang.org/x/crypto/bcrypt"

	"purch/internal/database"
	"purch/internal/utils"
	"purch/internal/config"
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
		FirstName: "foo",
		LastName: "bar",
		Username: "testuser_" + uuid.NewString()[:8],
		Password: string(password),
		Income: &income,
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
	// generate a sandbox public_token
	sandboxPublicTokenResp, httpResp, err := client.PlaidApi.SandboxPublicTokenCreate(ctx).SandboxPublicTokenCreateRequest(
		*plaid.NewSandboxPublicTokenCreateRequest(
			institutionID,
			products,
		),
	).Execute()

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


func pollForTransactionsSync(t *testing.T, ctx context.Context, plaidClient *plaid.APIClient, request *plaid.TransactionsSyncRequest) (*plaid.TransactionsSyncResponse, error) {

	for i := 0; i < 10; i++ {
		response, _, err := plaidClient.PlaidApi.TransactionsSync(ctx).TransactionsSyncRequest(*request).Execute()

		if err == nil {
			return &response, nil
		}

		plaidErr, conversionErr := plaid.ToPlaidError(err)
		assert.NoError(t, conversionErr)
		if plaidErr.ErrorCode == "PRODUCT_NOT_READY" {
			time.Sleep(2 * time.Second)
			continue
		}

		return &response, err
	}

	return nil, errors.New("failed to get transactions")
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
	// now sync all transactions
	err = SyncTransactions(ctx, itemID, accessToken, "")
	if err != nil {
		t.Logf("error from SyncTransactions task in TestSyncTransactions: %v", err)
	}
	require.Nil(t, err)
	transactions, err := database.GetUserTransactions(ctx, testUser.ID)
	if err != nil {
		t.Logf("error pulling transactions persisted in SyncTransactions task in TestSyncTransactions: %v", err)
	}
	assert.NotEmpty(t, transactions)
	t.Cleanup(func() {
		// delete user and everything follows due to cascading deletes
		database.DeleteUser(ctx, testUser)
	})
}