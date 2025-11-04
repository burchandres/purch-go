package tasks

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/plaid/plaid-go/v40/plaid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

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
	return database.User{
		FirstName: "foo",
		LastName: "bar",
		Username: "testuser_" + uuid.NewString()[:8],
		Password: "testpass",
		Income: &income,
		IncomeRate: &incomeRate,
	}
}

func storeTestUser(t *testing.T) database.User {
	testUser := getTestUser()
	err := database.StoreUser(context.Background(), testUser)
	assert.Nil(t, err)
	// pull the user from database to get the id
	testUser, err = database.GetUserByUsername(context.Background(), testUser.Username)
	assert.Nil(t, err)
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

func TestSyncItem(t *testing.T) {
	plaidClient := utils.GetPlaidClient()
	config := utils.GetConfig()
	ctx := context.Background()
	// create sandbox item
	publicTokenExchangeResp := createSandboxItem(t, ctx, plaidClient, FIRST_PLATYPUS_BANK, config.GetPlaidProducts())
	// store a test user unique to this test
	testUser := storeTestUser(t)
	// test the task SyncItem
	itemID := publicTokenExchangeResp.ItemId
	accessToken := publicTokenExchangeResp.AccessToken
	err := SyncItem(ctx, testUser.ID, itemID, accessToken)
	require.NotNil(t, err)
	// pull from the database and make sure it is persisted
	items, err := database.GetUserItems(ctx, testUser.ID)
	require.NotNil(t, err)
	assert.Equal(t, 1, len(items))

	t.Cleanup(func() {
		database.DeleteUser(ctx, testUser)
	})
}

func TestSyncAccounts(t *testing.T) {}

func TestSyncTransactions(t *testing.T) {}