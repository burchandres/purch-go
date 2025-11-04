package tasks

import (
	"context"
	"errors"
	"log/slog"

	"github.com/google/uuid"
	"github.com/plaid/plaid-go/v40/plaid"
	"golang.org/x/sync/errgroup"

	"purch/internal/database"
	"purch/internal/utils"
)

const (
	YYYYMMDD        = "2006-01-02"
	unknownCategory = "unknown_category"
)

var (
	ErrRequestingItem = errors.New("error requesting new item information")
	ErrStoringItem    = errors.New("error storing new item information")

	ErrRequestingAccounts = errors.New("error requesting new accounts information")
	ErrStoringAccounts    = errors.New("error storing new accounts information")

	ErrRequestingTransactions = errors.New("error requesting transactions")
	ErrStoringTransactions    = errors.New("error storing transactions")
)

func SyncItemAccountsTransactionsPipeline(
	ctx context.Context,
	userID uuid.UUID,
	itemID string,
	accessToken string,
) error {
	if err := SyncItem(ctx, userID, itemID, accessToken); err != nil {
		return err
	}
	if err := SyncAccounts(ctx, itemID, accessToken); err != nil {
		return err
	}
	if err := SyncTransactions(ctx, itemID, accessToken, ""); err != nil {
		return err
	}
	return nil
}

func SyncItem(
	ctx context.Context,
	userID uuid.UUID,
	itemID string,
	accessToken string,
) error {
	plaidClient := utils.GetPlaidClient()
	slog.Debug("pulling item info from plaid for local persistence.", "itemID", itemID, "userID", userID)
	// create itemGetRequest
	request := plaid.NewItemGetRequest(accessToken)
	// execute itemGetRequest
	resp, _, err := plaidClient.PlaidApi.ItemGet(ctx).ItemGetRequest(*request).Execute()
	if err != nil {
		slog.Error("failed to get item info.", "error", err.Error(), "userID", userID.String())
		return ErrRequestingItem
	}
	item := resp.GetItem()
	// create query params for storing item
	var itemParams database.Item
	itemParams.ID = itemID
	itemParams.AccessToken = accessToken
	itemParams.UserID = userID
	itemParams.Name = item.GetInstitutionName()
	// store the item
	if err = database.StoreItem(ctx, itemParams); err != nil {
		slog.Error("failed to store item", "error", err, "userID", userID)
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
	slog.Debug("pulling all accounts' information from plaid.", "item", itemID)
	// create GetAccountsRequest
	accountsGetRequest := plaid.NewAccountsGetRequest(accessToken)
	// execute GetAccountsRequest
	accountsGetResp, _, err := plaidClient.PlaidApi.AccountsGet(ctx).AccountsGetRequest(*accountsGetRequest).Execute()
	if err != nil {
		slog.Error("error requesting accounts' information.", "item", itemID)
		return ErrRequestingAccounts
	}
	// store all accounts
	accounts, ok := accountsGetResp.GetAccountsOk()
	if !ok {
		slog.Error("error pulling accounts.", "item", itemID)
		return ErrRequestingAccounts
	}
	// bulk insert accounts
	accountsToStore := make([]*database.Account, len(*accounts))
	for i, account := range *accounts {
		var storeAccountParams database.Account

		storeAccountParams.ID = account.GetAccountId()
		storeAccountParams.ItemID = itemID
		storeAccountParams.Name = account.GetName()

		accountsToStore[i] = &storeAccountParams
	}

	return database.StoreAccounts(ctx, accountsToStore)
}

func SyncTransactions(
	ctx context.Context,
	itemID string,
	accessToken string,
	cursor string,
) error {
	worker := NewTransactionsWorker(ctx, itemID, accessToken, cursor)
	return worker.Work()
}

type TransactionsWorker struct {
	itemID      string
	accessToken string
	cursor      string
	ctx         context.Context
}

func NewTransactionsWorker(ctx context.Context, itemID, accessToken, cursor string) *TransactionsWorker {
	return &TransactionsWorker{
		itemID:       itemID,
		accessToken:  accessToken,
		cursor:       cursor,
		ctx:          ctx,
	}
}

func (w *TransactionsWorker) Work() error {
	plaidClient := utils.GetPlaidClient()
	transactionsSyncRequest := plaid.NewTransactionsSyncRequest(w.accessToken)
	hasMore := true
	nextCursor := w.cursor
	for hasMore {
		// set cursor value
		transactionsSyncRequest.SetCursor(nextCursor)
		// execute TransactionsSyncRequest
		transactionsSyncResp, _, err := plaidClient.PlaidApi.TransactionsSync(w.ctx).TransactionsSyncRequest(*transactionsSyncRequest).Execute()
		if err != nil {
			slog.Error("error pulling transactions", "error", err.Error(), "itemID", w.itemID, "cursor", nextCursor)
			break
		}
		// send transactions for processing
		g, ctx := errgroup.WithContext(w.ctx)
		// process newly added transactions
		g.Go(func() error {
			return w.syncAddedTransactionsFromPlaid(ctx, transactionsSyncResp.GetAdded())
		})
		// process modified transactions
		g.Go(func() error {
			return w.syncModifiedTransactionsFromPlaid(ctx, transactionsSyncResp.GetModified())
		})
		// process removed transactions
		g.Go(func() error {
			return w.syncRemovedTransactionsFromPlaid(ctx, transactionsSyncResp.GetRemoved())
		})
		// wait for the three goroutines to finish incase we need to break and restart at another time from the recorded cursor
		if err := g.Wait(); err != nil {
			slog.Error("error syncing transactions from plaid", "error", err.Error(), "item-id", w.itemID)
			break
		}
		// update hasMore and transaction cursor
		hasMore = transactionsSyncResp.GetHasMore()
		nextCursor = w.cursor
	}
	// update cursor for item after syncing all transactions
	return database.UpdateItemCursor(w.ctx, w.cursor, w.itemID)
}

func (w *TransactionsWorker) syncAddedTransactionsFromPlaid(ctx context.Context, addedTransactions []plaid.Transaction) error {
	// parse plaid transactions into purch transaction
	transactions := make([]*database.Transaction, len(addedTransactions))
	for i := range transactions {
		transactions[i] = parsePlaidTransaction(addedTransactions[i])
	}
	// persist transactions for the user
	if err := database.StoreTransactions(ctx, transactions); err != nil {
		slog.Error("error persisting user's transactions", "error", err.Error(), "item-id", w.itemID)
		return err
	}
	return nil
}

func (w *TransactionsWorker) syncModifiedTransactionsFromPlaid(ctx context.Context, modifiedTransactions []plaid.Transaction) error {
	// parse plaid transactions into purch transaction
	transactions := make([]*database.Transaction, len(modifiedTransactions))
	for i := range transactions {
		transactions[i] = parseModifiedPlaidTransaction(modifiedTransactions[i])
	}
	// persist transactions for the user
	if err := database.UpdateTransactions(ctx, transactions); err != nil {
		slog.Error("error updating user's transactions", "error", err.Error(), "item-id", w.itemID)
		return err
	}
	return nil
}

func (w *TransactionsWorker) syncRemovedTransactionsFromPlaid(ctx context.Context, removedTransactions []plaid.RemovedTransaction) error {
	transactions := make([]string, len(removedTransactions))
	for i := range transactions {
		transactions[i] = removedTransactions[i].GetTransactionId()
	}
	if err := database.DeleteTransactions(ctx, transactions); err != nil {
		slog.Error("error deleting removed transactions", "error", err.Error(), "item-id", w.itemID)
		return err
	}
	return nil
}

func parsePlaidTransaction(transaction plaid.Transaction) *database.Transaction {
	var t database.Transaction
	// tx id and account id
	t.ID = transaction.GetTransactionId()
	t.AccountID = transaction.GetAccountId()
	// category label
	// TODO: perform semantic search to map via foreign key to user defined category
	categoryLabel := unknownCategory
	if len(transaction.GetCategory()) > 0 {
		categoryLabel = transaction.GetCategory()[0]
	}
	t.CategoryLabel = &categoryLabel
	// transaction dates
	t.AuthorizedDate = transaction.GetAuthorizedDatetime()
	settledDate := transaction.GetDatetime()
	t.SettledDate = &settledDate
	// merchant
	merchant := transaction.GetMerchantName()
	t.Merchant = &merchant
	// transaction amount
	t.Amount = transaction.GetAmount()
	// currency code
	currencyCode := transaction.GetIsoCurrencyCode()
	t.CurrencyCode = &currencyCode
	// pending
	t.Pending = transaction.GetPending()

	return &t
}

func parseModifiedPlaidTransaction(transaction plaid.Transaction) *database.Transaction {
	var t database.Transaction

	t.ID = transaction.GetTransactionId()
	settledDate := transaction.GetDatetime()
	t.SettledDate = &settledDate
	t.Pending = transaction.GetPending()

	return &t
}
