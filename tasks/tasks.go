package tasks

import (
	"context"
	"errors"
	"log/slog"
	"sync"

	"github.com/jackc/pgx/v5"
	"github.com/plaid/plaid-go/v40/plaid"

	"purch/database"
	"purch/utils"
)

const (
	iso8601TimeFormat = "2006-01-02"
	unknownCategory = "unknown_category"
)

var (
	ErrRequestingItem = errors.New("error requesting new item information")
	ErrStoringItem = errors.New("error storing new item information")

	ErrRequestingAccounts = errors.New("error requesting new accounts information")
	ErrStoringAccounts = errors.New("error storing new accounts information")

	ErrRequestingTransactions = errors.New("error requesting transactions")
	ErrStoringTransactions = errors.New("error storing transactions")
)

func SyncItemAccountsTransactionsPipeline(
	ctx context.Context,
	userID int64,
	itemID string,
	accessToken string,
) error {
	if err := SyncItem(ctx, userID, itemID, accessToken); err != nil {
		slog.Error("error storing item in item->accounts->transactions intial sync pipeline", "error", err.Error())
		return err
	}
	if err := SyncAccounts(ctx, itemID, accessToken); err != nil {
		slog.Error("error storing accounts in item->accounts->transactions initial sync pipeline", "error", err.Error())
		return err
	}
	if err := SyncTransactions(ctx, itemID, accessToken, ""); err != nil {
		slog.Error("error syncing transactions in item->accounts->transactions initial sync pipeline", "error", err.Error())
		return err
	}
	return nil
}

func SyncItem(
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

func SyncAccounts(
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
	itemID string,
	accessToken string,
	cursor string,
) error {
	plaidClient := utils.GetPlaidClient()
	db := database.GetPool()
	// added, modified and removed transactions are all disjoint
	// so should be fine to sync each in their own goroutines
	var wg sync.WaitGroup

	// channels to push transactions to for below goroutines to read from for batching
	addedChan := make(chan []plaid.Transaction)
	modifiedChan := make(chan []plaid.Transaction)
	removedChan := make(chan []plaid.RemovedTransaction)
	errChan := make(chan error)

	// get all newly added transactions and push
	// TODO: move out to separate helper function that takes addedChan and itemID as input
	wg.Go(func() {
		tx, err := db.BeginTx(ctx, pgx.TxOptions{})
		if err != nil {
			slog.Error("error beginning db tx for added transactions", "error", err.Error())
			errChan<-err
			return
		}
		queries := database.New(tx)
		for transactions := range addedChan {
			for _, transaction := range transactions {
				storeTransactionParams := getStoreTransactionParams(transaction)
				if _, err := queries.StoreTransaction(ctx, storeTransactionParams); err != nil {
					slog.Error("error storing transaction", "error", err.Error(), "itemID", itemID)
					errChan<-err
					tx.Rollback(ctx)
					return
				}
			}
			tx.Commit(ctx)
		}
	})

	// get all modified transactions and update
	// TODO: move out to separate helper function that takes modifiedChan and itemID as input
	wg.Go(func() {
		tx, err := db.BeginTx(ctx, pgx.TxOptions{})
		if err != nil {
			slog.Error("error beginning db tx for modified transactions", "error", err.Error())
			errChan<-err
			return
		}
		queries := database.New(tx)
		for transactions := range modifiedChan {
			for _, transaction := range transactions {
				updateTransactionParams := getUpdateTransactionParams(transaction)
				if _, err := queries.UpdateTransaction(ctx, updateTransactionParams); err != nil {
					slog.Error("error updating transaction", "error", err.Error(), "itemID", itemID)
					errChan<-err
					tx.Rollback(ctx)
					return
				}
			}
			tx.Commit(ctx)
		}
	})

	// get all deleted transactions and remove
	// TODO: move out to separate helper function that takes removedChan and itemID as input
	wg.Go(func() {
		tx, err := db.BeginTx(ctx, pgx.TxOptions{})
		if err != nil {
			slog.Error("error beginning db tx for deleted transactions", "error", err.Error())
			errChan<-err
			return
		}
		queries := database.New(tx)
		for transactions := range removedChan {
			for _, transaction := range transactions {
				transactionID := transaction.GetTransactionId()
				if err := queries.DeleteTransaction(ctx, transactionID); err != nil {
					slog.Error("error deleting transaction", "error", err.Error(), "itemID", itemID, "transactionID", transactionID)
					errChan<-err
					tx.Rollback(ctx)
					return
				}
			}
			tx.Commit(ctx)
		}
	})
	// loop through until there are no more transactions according to plaid
	hasMore := true
	for hasMore {
		// TODO: move all this to helper function as well
		// create TransactionsSyncRequest
		transactionsSyncRequest := plaid.NewTransactionsSyncRequest(accessToken)
		transactionsSyncRequest.SetCursor(cursor)
		// execute TransactionsSyncRequest
		transactionsSyncResp, _, err := plaidClient.PlaidApi.TransactionsSync(ctx).TransactionsSyncRequest(*transactionsSyncRequest).Execute()
		if err != nil {
			slog.Error("error pulling transactions", "itemID", itemID)
			return ErrRequestingTransactions
		}
		// update hasMore and cursor
		hasMore = transactionsSyncResp.GetHasMore()
		cursor = transactionsSyncResp.GetNextCursor()
		// push to addedChan for processing
		added := transactionsSyncResp.GetAdded()
		if len(added) == 0 {
			close(addedChan)
		} else {
			addedChan <- added
		}
		// push to modifiedChan for processing
		modified := transactionsSyncResp.GetModified()
		if len(modified) == 0 {
			close(modifiedChan)
		} else {
			modifiedChan <- modified
		}
		// push to removedChan for processing
		removed := transactionsSyncResp.GetRemoved()
		if len(removed) == 0 {
			close(removedChan)
		} else {
			removedChan <- removed
		}
		if !hasMore {
			close(addedChan)
			close(modifiedChan)
			close(removedChan)
		}
	}
	// wait for all transaction syncing to finish
	wg.Wait()
	return nil
}

func getStoreTransactionParams(transaction plaid.Transaction) database.StoreTransactionParams {
	// TODO: finish this function
	// use this link: https://github.com/plaid/plaid-go/blob/master/plaid/model_transaction.go
	// and this link: https://plaid.com/docs/api/products/transactions/#transactionssync
	return database.StoreTransactionParams{}
}

func getUpdateTransactionParams(transaction plaid.Transaction) database.UpdateTransactionParams {
	// TODO: finsh this function
	// use this link: https://github.com/plaid/plaid-go/blob/master/plaid/model_transaction.go
	// and this link: https://plaid.com/docs/api/products/transactions/#transactionssync
	return database.UpdateTransactionParams{}
}