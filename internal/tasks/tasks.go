package tasks

import (
	"context"
	"errors"
	"log/slog"

	"github.com/google/uuid"
	"github.com/plaid/plaid-go/v40/plaid"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/sync/errgroup"

	"purch/internal/config"
	"purch/internal/database"
	"purch/internal/utils"
)

const (
	YYYYMMDD        = "2006-01-02"
	unknownCategory = "unknown_category"
)

var (
	ErrRequestingItem     = errors.New("error requesting new item information")
	ErrStoringItem        = errors.New("error storing new item information")
	ErrHashingAccessToken = errors.New("error hashing access token for storage")

	ErrRequestingAccounts = errors.New("error requesting new accounts information")
	ErrStoringAccounts    = errors.New("error storing new accounts information")

	ErrRequestingTransactions = errors.New("error requesting transactions")
	ErrStoringTransactions    = errors.New("error storing transactions")
)

/*
Used to sync a user's bank, bank accounts and transactions
upon initial registration with Purch.

Synchronously it will:

  - Retrieve and store the item (i.e. Bank like "Wells Fargo")

  - Retrieve all accounts the user chose to link with that item

  - Retreive all transactions for those accounts
*/
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

// Single routine to retrieve an item associated with the provided accessToken
func SyncItem(
	ctx context.Context,
	userID uuid.UUID,
	itemID string,
	accessToken string,
) error {
	config := config.GetConfig()
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
	
	hashedAccessToken, err := bcrypt.GenerateFromPassword([]byte(accessToken), config.BcryptCost)
	if err != nil {
		slog.Error("error hashing access token for item", "item-id", itemID, "user-id", userID)
		return ErrHashingAccessToken
	}
	itemParams.AccessToken = string(hashedAccessToken)
	
	itemParams.UserID = userID
	itemParams.Name = item.GetInstitutionName()
	// store the item
	if err = database.StoreItem(ctx, itemParams); err != nil {
		slog.Error("failed to store item", "error", err, "userID", userID)
		return ErrStoringItem
	}

	return nil
}

// Single routine to retrieve all accounts selected associated with the accessToken
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
	accountsToStore := make([]database.Account, len(*accounts))
	for i, account := range *accounts {
		var storeAccountParams database.Account

		storeAccountParams.ID = account.GetAccountId()
		storeAccountParams.ItemID = itemID
		storeAccountParams.Name = account.GetName()

		accountsToStore[i] = storeAccountParams
	}

	return database.StoreAccounts(ctx, accountsToStore)
}

// A single routine to retrieve all transactions associated with the accounts selected for the accessToken
func SyncTransactions(
	ctx context.Context,
	itemID string,
	accessToken string,
	cursor string,
) error {
	worker := NewTransactionsWorker(ctx, itemID, accessToken)
	return worker.Work(cursor)
}

type TransactionsWorker struct {
	itemID      string
	accessToken string
	ctx         context.Context
}

func NewTransactionsWorker(ctx context.Context, itemID, accessToken string) *TransactionsWorker {
	return &TransactionsWorker{
		itemID:      itemID,
		accessToken: accessToken,
		ctx:         ctx,
	}
}

func (w *TransactionsWorker) Work(cursor string) error {
	var err error

	plaidClient := utils.GetPlaidClient()
	transactionsSyncRequest := plaid.NewTransactionsSyncRequest(w.accessToken)
	hasMore := true
	addedTransactions := []plaid.Transaction{}
	modifiedTransactions := []plaid.Transaction{}
	removedTransactions := []plaid.RemovedTransaction{}
	for hasMore {
		// set cursor value
		transactionsSyncRequest.SetCursor(cursor)
		// execute TransactionsSyncRequest
		var transactionsSyncResp plaid.TransactionsSyncResponse
		transactionsSyncResp, _, err = plaidClient.PlaidApi.TransactionsSync(w.ctx).TransactionsSyncRequest(*transactionsSyncRequest).Execute()
		if err != nil {
			slog.Error("error pulling transactions", "error", err.Error(), "item-id", w.itemID, "cursor", cursor)
			break
		}
		addedTransactions = append(addedTransactions, transactionsSyncResp.GetAdded()...)
		modifiedTransactions = append(modifiedTransactions, transactionsSyncResp.GetModified()...)
		removedTransactions = append(removedTransactions, transactionsSyncResp.GetRemoved()...)
		// update hasMore and transaction cursor
		hasMore = transactionsSyncResp.GetHasMore()
		cursor = transactionsSyncResp.GetNextCursor()
	}
	if err != nil {
		return err
	}
	// send transactions for processing
	g, ctx := errgroup.WithContext(w.ctx)
	// process newly added transactions
	g.Go(func() error {
		return w.syncAddedTransactionsFromPlaid(ctx, addedTransactions)
	})
	// process modified transactions
	g.Go(func() error {
		return w.syncModifiedTransactionsFromPlaid(ctx, modifiedTransactions)
	})
	// process removed transactions
	g.Go(func() error {
		return w.syncRemovedTransactionsFromPlaid(ctx, removedTransactions)
	})
	// wait for the three goroutines to finish incase we need to break and restart at another time from the recorded cursor
	if err := g.Wait(); err != nil {
		slog.Error("error syncing transactions from plaid", "error", err.Error(), "item-id", w.itemID)
		return err
	}
	// update cursor for item after syncing all transactions
	return database.UpdateItemCursor(w.ctx, cursor, w.itemID)
}

func (w *TransactionsWorker) syncAddedTransactionsFromPlaid(ctx context.Context, addedTransactions []plaid.Transaction) error {
	if len(addedTransactions) == 0 {
		return nil
	}
	// parse plaid transactions into purch transaction
	transactions := make([]database.Transaction, len(addedTransactions))
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
	if len(modifiedTransactions) == 0 {
		return nil
	}
	// parse plaid transactions into purch transaction
	transactions := make([]database.Transaction, len(modifiedTransactions))
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
	if len(removedTransactions) == 0 {
		return nil
	}
	// parse transactions for the transaction ids of the ones to be deleted
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

func parsePlaidTransaction(transaction plaid.Transaction) database.Transaction {
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

	return t
}

func parseModifiedPlaidTransaction(transaction plaid.Transaction) database.Transaction {
	var t database.Transaction

	t.ID = transaction.GetTransactionId()
	settledDate := transaction.GetDatetime()
	t.SettledDate = &settledDate
	t.Pending = transaction.GetPending()

	return t
}
