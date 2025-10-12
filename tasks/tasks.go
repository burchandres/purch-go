package tasks

import (
	"context"
	"errors"
	"log/slog"

	"github.com/jackc/pgx/v5"
	"github.com/plaid/plaid-go/v40/plaid"

	"purch/database"
	"purch/utils"
)

var (
	ErrRequestingItem = errors.New("error requesting new item information")
	ErrStoringItem = errors.New("error storing new item information")

	ErrRequestingAccounts = errors.New("error requesting new accounts information")
	ErrStoringAccounts = errors.New("error storing new accounts information")

	ErrRequestingTransactions = errors.New("error requesting transactions")
	ErrStoringTransactions = errors.New("error storing transactions")
)

func StoreItemAccountsTransactionsPipeline(
	ctx context.Context,
	userID int64,
	itemID string,
	accessToken string,
) error {
	if err := StoreItem(ctx, userID, itemID, accessToken); err != nil {
		slog.Error("error storing item in item->accounts->transactions intial sync pipeline", "error", err.Error())
		return err
	}
	if err := StoreAccounts(ctx, itemID, accessToken); err != nil {
		slog.Error("error storing accounts in item->accounts->transactions initial sync pipeline", "error", err.Error())
		return err
	}
	// TODO: finish this function
	if err := SyncTransactions(ctx, accessToken); err != nil {
		slog.Error("error syncing transactions in item->accounts->transactions initial sync pipeline", "error", err.Error())
		return err
	}
	return nil
}

func StoreItem(
	ctx context.Context, 
	userID int64, 
	itemID string, 
	accessToken string,
) error {
	plaidClient := utils.GetPlaidClient()
	db := database.GetPool()
	queries := database.New(db)
	slog.Info("pulling item info to store", "item", itemID, "user", userID)
	// create itemGetRequest
	request := plaid.NewItemGetRequest(accessToken)
	// execute itemGetRequest
	resp, _, err := plaidClient.PlaidApi.ItemGet(ctx).ItemGetRequest(*request).Execute()
	if err != nil {
		slog.Error("failed to get item", "error", err, "endpoint", "/api/user/exchange-public-token")
		return ErrRequestingItem
	}
	item := resp.GetItem()
	// create query params for storing item
	var storeItemParams database.StoreItemParams
	storeItemParams.ID = itemID
	storeItemParams.AccessToken = accessToken
	storeItemParams.UserID = userID
	storeItemParams.Name = item.GetInstitutionName()
	// store the item
	_, err = queries.StoreItem(ctx, storeItemParams)
	if err != nil {
		slog.Error("failed to store item", "error", err, "endpoint", "/api/user/exchange-public-token")
		return ErrStoringItem
	}

	return nil
}

func StoreAccounts(
	ctx context.Context,
	itemID string,
	accessToken string,
) error {
	plaidClient := utils.GetPlaidClient()
	db := database.GetPool()
	slog.Info("pulling all accounts", "item", itemID)
	// create GetAccountsRequest
	accountsGetRequest := plaid.NewAccountsGetRequest(accessToken)
	// execute GetAccountsRequest
	accountsGetResp, _, err := plaidClient.PlaidApi.AccountsGet(ctx).AccountsGetRequest(*accountsGetRequest).Execute()
	if err != nil {
		slog.Error("error requesting accounts info", "item", itemID)
		return ErrRequestingAccounts
	}
	// store all accounts
	accounts, ok := accountsGetResp.GetAccountsOk()
	if !ok {
		slog.Error("error pulling accounts", "item", itemID)
		return ErrRequestingAccounts
	}
	// start database transaction
	tx, err := db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		slog.Error("error starting transaction for account storage", "item", itemID)
		return ErrStoringAccounts
	}
	defer func() {if err == nil {tx.Commit(ctx)}}()

	queriesTx := database.New(tx)

	for _, account := range *accounts {
		var storeAccountParams database.StoreAccountParams

		storeAccountParams.ID = account.GetAccountId()
		storeAccountParams.ItemID = itemID
		storeAccountParams.Name = account.GetName()
		// push using the transaction so we commit all at once to avoid overhead
		if _, err = queriesTx.StoreAccount(ctx, storeAccountParams); err != nil {
			tx.Rollback(ctx)
			slog.Error("error storing account", "itemID", itemID, "accountName", account.GetName())
			return ErrStoringAccounts
		}
	}

	return nil
}

// TODO: Finish this function
func SyncTransactions(
	ctx context.Context,
	accessToken string,
) error {
	return nil
}