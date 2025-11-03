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
	worker.Start()
	return worker.Wait()
}

type TransactionsWorker struct {
	itemID      string
	accessToken string
	cursor      string
	ctx         context.Context

	g    *errgroup.Group
	gCtx context.Context

	addedChan    chan []plaid.Transaction
	modifiedChan chan []plaid.Transaction
	removedChan  chan []plaid.RemovedTransaction
}

func NewTransactionsWorker(ctx context.Context, itemID, accessToken, cursor string) *TransactionsWorker {
	g, gCtx := errgroup.WithContext(ctx)
	return &TransactionsWorker{
		itemID:       itemID,
		accessToken:  accessToken,
		cursor:       cursor,
		g:            g,
		gCtx:         gCtx,
		ctx:          ctx,
		addedChan:    make(chan []plaid.Transaction),
		modifiedChan: make(chan []plaid.Transaction),
		removedChan:  make(chan []plaid.RemovedTransaction),
	}
}

func (w *TransactionsWorker) Start() {
	w.g.Go(func() error {
		return w.pullTransactionsFromPlaid()
	})
	w.g.Go(func() error {
		return w.syncAddedTransactionsFromPlaid()
	})
	w.g.Go(func() error {
		return w.syncModifiedTransactionsFromPlaid()
	})
	w.g.Go(func() error {
		return w.syncRemovedTransactionsFromPlaid()
	})
}

func (w *TransactionsWorker) Wait() error {
	err := w.g.Wait()
	w.shutdown()
	return err
}

func (w *TransactionsWorker) shutdown() {
	// check if any of the channels aren't closed and close them
	if _, ok := <- w.addedChan; ok {
		close(w.addedChan)
	}
	if _, ok := <- w.modifiedChan; ok {
		close(w.modifiedChan)
	}
	if _, ok := <- w.removedChan; ok {
		close(w.removedChan)
	}
}

func (w *TransactionsWorker) pullTransactionsFromPlaid() error {
	hasMore := true
	// create TransactionsSyncRequest
	plaidClient := utils.GetPlaidClient()
	transactionsSyncRequest := plaid.NewTransactionsSyncRequest(w.accessToken)
	for hasMore {
		select {
		case <-w.gCtx.Done():
			slog.Debug("group context cancelled, recorded in pulling transactions", "error", w.gCtx.Err(), "itemID", w.itemID)
			return w.gCtx.Err()
		default:
			// set cursor value
			transactionsSyncRequest.SetCursor(w.cursor)
			// execute TransactionsSyncRequest
			transactionsSyncResp, _, err := plaidClient.PlaidApi.TransactionsSync(w.gCtx).TransactionsSyncRequest(*transactionsSyncRequest).Execute()
			if err != nil {
				slog.Error("error pulling transactions", "itemID", w.itemID, "cursor", w.cursor, )
				return ErrRequestingTransactions
			}
			// update hasMore and transaction cursor
			hasMore = transactionsSyncResp.GetHasMore()
			w.cursor = transactionsSyncResp.GetNextCursor()
			// send added transactions for processing
			added := transactionsSyncResp.GetAdded()
			if len(added) == 0 {
				close(w.addedChan)
			} else {
				w.addedChan <- added
			}
			// send modified transactions for processing
			modified := transactionsSyncResp.GetModified()
			if len(modified) == 0 {
				close(w.modifiedChan)
			} else {
				w.modifiedChan <- modified
			}
			// send removed transactions for processing
			removed := transactionsSyncResp.GetRemoved()
			if len(removed) == 0 {
				close(w.removedChan)
			} else {
				w.removedChan <- removed
			}
		}
	}
	return nil
}

func (w *TransactionsWorker) syncAddedTransactionsFromPlaid() error {
	for {
		select {
		case <-w.gCtx.Done():
			slog.Debug("group context cancelled, recorded in syncing added transactions routine", "error", w.gCtx.Err(), "itemID", w.itemID)
			return w.gCtx.Err()
		case addedTransactions := <-w.addedChan:
			// parse plaid transactions into purch transaction
			transactions := make([]*database.Transaction, len(addedTransactions))
			for i := range transactions {
				transactions[i] = parsePlaidTransaction(addedTransactions[i])
			}
			// persist transactions for the user
			if err := database.StoreTransactions(w.gCtx, transactions); err != nil {
				slog.Error("error persisting user's transactions", "error", err.Error(), "itemID", w.itemID)
			}
		default:
			slog.Debug("addedChan is closed and group context is still valid, exiting add transactions routine", "itemID", w.itemID)
			return nil
		}
	}
}

func (w *TransactionsWorker) syncModifiedTransactionsFromPlaid() error {
	for {
		select {
		case <-w.gCtx.Done():
			slog.Debug("group context cancelled, recorded in syncing modified transactions routine", "error", w.gCtx.Err(), "item id", w.itemID)
			return w.gCtx.Err()
		case modifiedTransactions, ok := <-w.modifiedChan:
			if !ok {
				slog.Debug("modifiedChan closed", "error")
			}
			// parse plaid transactions into purch transaction
			transactions := make([]*database.Transaction, len(modifiedTransactions))
			for i := range transactions {
				transactions[i] = parseModifiedPlaidTransaction(modifiedTransactions[i])
			}
			// persist transactions for the user
			if err := database.UpdateTransactions(w.gCtx, transactions); err != nil {
				slog.Error("error updating user's transactions", "error", err.Error(), "itemID", w.itemID)
			}
		default:
			slog.Debug("exiting modified transactions routine", "item id", w.itemID)
			return nil
		}
	}
}


func (w *TransactionsWorker) syncRemovedTransactionsFromPlaid() error {
	for {
		select {
		case <- w.gCtx.Done():
			slog.Debug("group context cancelled, recorded in syncing removed transactions routine", "error", w.gCtx.Err(), "item id", w.itemID)
			return w.gCtx.Err()
		case removedPlaidTransactions, ok := <-w.removedChan:
			if !ok {
				slog.Debug("removedChan closed, exiting syncing removed transactions routine")
				return nil
			}
			removedTransactions := make([]string, len(removedPlaidTransactions))
			for i := range removedTransactions {
				removedTransactions[i] = removedPlaidTransactions[i].GetTransactionId()
			}
			if err := database.DeleteTransactions(w.gCtx, removedTransactions); err != nil {
				slog.Error("error deleting removed transactions", "error", err.Error(), "itemID", w.itemID)
				return err
			}
		default:
			slog.Debug("exiting removed transactions routine", "item id", w.itemID)
			return nil
		}
	}
}

// TODO: separate added vs modified to only extract fields we care about that were updated
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

// func SyncTransactions(
// 	ctx context.Context,
// 	itemID string,
// 	accessToken string,
// 	cursor string,
// ) error {
// 	// added, modified and removed transactions are all disjoint so running each in separate goroutines
// 	var wg sync.WaitGroup

// 	// channels to push transactions to for below goroutines to read from for batching
// 	addedChan := make(chan []plaid.Transaction)
// 	modifiedChan := make(chan []plaid.Transaction)
// 	removedChan := make(chan []plaid.RemovedTransaction)
// 	errChan := make(chan error, 3)

// 	processCtx, cancel := context.WithCancel(ctx)

// 	// get all newly added transactions and push
// 	wg.Go(func() {
// 		processAddedTransactions(processCtx, itemID, addedChan, errChan)
// 	})

// 	// get all modified transactions and update
// 	wg.Go(func() {
// 		processModifiedTransactions(processCtx, itemID, modifiedChan, errChan)
// 	})

// 	// get all deleted transactions and remove
// 	wg.Go(func() {
// 		processRemovedTransactions(processCtx, itemID, removedChan, errChan)
// 	})
// 	// loop through until there are no more transactions according to plaid
// 	var err error
// 	var nextCursor string
// 	hasMore := true
// HasMore:
// 	for hasMore {
// 		select {
// 		case <-ctx.Done():
// 			cancel()
// 			break HasMore
// 		default:
// 			hasMore, nextCursor, err = gatherTransactionsForProcessing(
// 				ctx,
// 				itemID,
// 				accessToken,
// 				cursor,
// 				addedChan,
// 				modifiedChan,
// 				removedChan,
// 				errChan,
// 			)
// 			if err != nil {
// 				errChan <- err
// 				closeChannels(addedChan, modifiedChan, removedChan)
// 				cancel()
// 				break HasMore
// 			}
// 			cursor = nextCursor
// 		}
// 	}
// 	// wait for all transaction syncing to finish
// 	closeChannels(addedChan, modifiedChan, removedChan)
// 	wg.Wait()
// 	close(errChan)
// 	// to prevent any leaks
// 	cancel()

// 	// push cursor to item
// 	go func() {
// 		if err := updateItemCursor(ctx, itemID, cursor); err != nil {
// 			slog.Error("error updating item cursor", "error", err.Error(), "itemID", itemID)
// 		}
// 	}()

// 	for err = range errChan {
// 		slog.Error("error syncing transactions", "error", err.Error(), "itemID", itemID, "lastCursor", cursor)
// 		return err
// 	}
// 	return nil
// }

// func closeChannels(addedChan chan []plaid.Transaction, modifiedChan chan []plaid.Transaction, removedChan chan []plaid.RemovedTransaction) {
// 	close(addedChan)
// 	close(modifiedChan)
// 	close(removedChan)
// }

// func gatherTransactionsForProcessing(
// 	ctx context.Context,
// 	itemID string,
// 	accessToken string,
// 	cursor string,
// 	addedChan chan []plaid.Transaction,
// 	modifiedChan chan []plaid.Transaction,
// 	removedChan chan []plaid.RemovedTransaction,
// 	errChan chan error,
// ) (bool, string, error) {
// 	select {
// 	case err := <-errChan:
// 		slog.Error("error received from transaction sync process", "error", err.Error(), "itemID", itemID)
// 		return false, cursor, err
// 	default:
// 		plaidClient := utils.GetPlaidClient()
// 		// create TransactionsSyncRequest
// 		transactionsSyncRequest := plaid.NewTransactionsSyncRequest(accessToken)
// 		transactionsSyncRequest.SetCursor(cursor)
// 		// execute TransactionsSyncRequest
// 		transactionsSyncResp, _, err := plaidClient.PlaidApi.TransactionsSync(ctx).TransactionsSyncRequest(*transactionsSyncRequest).Execute()
// 		if err != nil {
// 			slog.Error("error pulling transactions", "itemID", itemID)
// 			return false, cursor, ErrRequestingTransactions
// 		}
// 		// update hasMore and transaction cursor
// 		hasMore := transactionsSyncResp.GetHasMore()
// 		nextCursor := transactionsSyncResp.GetNextCursor()
// 		// push to addedChan for processing
// 		added := transactionsSyncResp.GetAdded()
// 		if len(added) == 0 {
// 			close(addedChan)
// 		} else {
// 			addedChan <- added
// 		}
// 		// push to modifiedChan for processing
// 		modified := transactionsSyncResp.GetModified()
// 		if len(modified) == 0 {
// 			close(modifiedChan)
// 		} else {
// 			modifiedChan <- modified
// 		}
// 		// push to removedChan for processing
// 		removed := transactionsSyncResp.GetRemoved()
// 		if len(removed) == 0 {
// 			close(removedChan)
// 		} else {
// 			removedChan <- removed
// 		}
// 		return hasMore, nextCursor, nil
// 	}
// }

// func processAddedTransactions(
// 	ctx context.Context,
// 	itemID string,
// 	addedChan chan []plaid.Transaction,
// 	errChan chan error,
// ) {
// 	db := database.GetPool()
// 	tx, err := db.BeginTx(ctx, pgx.TxOptions{})
// 	if err != nil {
// 		slog.Error("error beginning db tx for added transactions", "error", err.Error())
// 		errChan <- err
// 		return
// 	}
// 	queries := database.New(tx)
// 	for {
// 		select {
// 		case <-ctx.Done():
// 			return
// 		case transactions, ok := <-addedChan:
// 			if !ok {
// 				tx.Commit(ctx)
// 				return
// 			}
// 			for _, transaction := range transactions {
// 				storeTransactionParams := getStoreTransactionParams(transaction)
// 				if _, err := queries.StoreTransaction(ctx, storeTransactionParams); err != nil {
// 					slog.Error("error storing added transaction", "error", err.Error(), "itemID", itemID)
// 					errChan <- err
// 					tx.Rollback(ctx)
// 					return
// 				}
// 			}
// 			tx.Commit(ctx)
// 		case err := <-errChan:
// 			slog.Error("error received from other transaction process routine", "error", err.Error(), "current-process", "added", "itemID", itemID)
// 			tx.Rollback(ctx)
// 			return
// 		default:
// 			tx.Commit(ctx)
// 			return
// 		}
// 	}
// }

// func getStoreTransactionParams(transaction plaid.Transaction) database.StoreTransactionParams {
// 	// use this link: https://github.com/plaid/plaid-go/blob/master/plaid/model_transaction.go
// 	// and this link: https://plaid.com/docs/api/products/transactions/#transactionssync
// 	var storeTransactionParams database.StoreTransactionParams

// 	storeTransactionParams.ID = transaction.GetTransactionId()
// 	storeTransactionParams.AccountID = transaction.GetAccountId()

// 	categoryLabel := unknownCategory
// 	if len(transaction.GetCategory()) > 0 {
// 		categoryLabel = transaction.GetCategory()[0]
// 	}
// 	storeTransactionParams.CategoryLabel = categoryLabel

// 	authorizedDate, _ := time.Parse(YYYYMMDD, transaction.GetAuthorizedDate())
// 	storeTransactionParams.AuthorizedDate = pgtype.Date{Time: authorizedDate, Valid: true}
// 	storeTransactionParams.Merchant = pgtype.Text{String: transaction.GetMerchantName(), Valid: true}

// 	var amount pgtype.Numeric
// 	// TODO: read error from this later
// 	amount.Scan(transaction.GetAmount())
// 	storeTransactionParams.Amount = amount

// 	storeTransactionParams.CurrencyCode = pgtype.Text{String: transaction.GetIsoCurrencyCode(), Valid: true}
// 	storeTransactionParams.Pending = transaction.GetPending()

// 	return storeTransactionParams
// }

// func processModifiedTransactions(
// 	ctx context.Context,
// 	itemID string,
// 	modifiedChan chan []plaid.Transaction,
// 	errChan chan error,
// ) {
// 	db := database.GetPool()
// 	tx, err := db.BeginTx(ctx, pgx.TxOptions{})
// 	if err != nil {
// 		slog.Error("error beginning db tx for modified transactions", "error", err.Error())
// 		errChan <- err
// 		return
// 	}
// 	queries := database.New(tx)
// 	for {
// 		select {
// 		case <-ctx.Done():
// 			return
// 		case transactions := <-modifiedChan:
// 			for _, transaction := range transactions {
// 				updateTransactionParams := getUpdateTransactionParams(transaction)
// 				if _, err := queries.UpdateTransaction(ctx, updateTransactionParams); err != nil {
// 					slog.Error("error processing modified transactions", "error", err.Error(), "itemID", itemID)
// 					errChan <- err
// 					tx.Rollback(ctx)
// 					return
// 				}
// 			}
// 			tx.Commit(ctx)
// 		case err := <-errChan:
// 			slog.Error("error received from other process routine, stopping", "error", err.Error(), "current-process", "modified", "itemID", itemID)
// 			tx.Rollback(ctx)
// 			return
// 		default:
// 			tx.Commit(ctx)
// 			return
// 		}
// 	}
// }

// func getUpdateTransactionParams(transaction plaid.Transaction) database.UpdateTransactionParams {
// 	// use this link: https://github.com/plaid/plaid-go/blob/master/plaid/model_transaction.go
// 	// and this link: https://plaid.com/docs/api/products/transactions/#transactionssync
// 	var updateTransactionParams database.UpdateTransactionParams

// 	updateTransactionParams.ID = transaction.GetTransactionId()

// 	settledDate, _ := time.Parse(YYYYMMDD, transaction.GetDate())
// 	updateTransactionParams.SettledDate = pgtype.Date{Time: settledDate, Valid: true}

// 	var amount pgtype.Numeric
// 	// TODO: read error from below scan later
// 	amount.Scan(transaction.GetAmount())
// 	updateTransactionParams.Amount = amount

// 	updateTransactionParams.Pending = transaction.GetPending()
// 	return updateTransactionParams
// }

// func processRemovedTransactions(
// 	ctx context.Context,
// 	itemID string,
// 	removedChan chan []plaid.RemovedTransaction,
// 	errChan chan error,
// ) {
// 	db := database.GetPool()
// 	tx, err := db.BeginTx(ctx, pgx.TxOptions{})
// 	if err != nil {
// 		slog.Error("error beginning db tx for deleted transactions", "error", err.Error())
// 		errChan <- err
// 		return
// 	}
// 	queries := database.New(tx)
// 	for {
// 		select {
// 		case <-ctx.Done():
// 			return
// 		case transactions := <-removedChan:
// 			for _, transaction := range transactions {
// 				transactionID := transaction.GetTransactionId()
// 				if err := queries.DeleteTransaction(ctx, transactionID); err != nil {
// 					slog.Error("error deleting transaction", "error", err.Error(), "itemID", itemID, "transactionID", transactionID)
// 					errChan <- err
// 					tx.Rollback(ctx)
// 					return
// 				}
// 			}
// 			tx.Commit(ctx)
// 		case err := <-errChan:
// 			slog.Error("error received from other process routine, stopping", "error", err.Error(), "current-process", "removed", "itemID", itemID)
// 			tx.Rollback(ctx)
// 			return
// 		default:
// 			tx.Commit(ctx)
// 			return
// 		}
// 	}
// }

// func updateItemCursor(ctx context.Context, itemID string, cursor string) error {
// 	db := database.GetPool()
// 	queries := database.New(db)
// 	item, err := queries.GetItem(ctx, itemID)
// 	if err != nil {
// 		slog.Error("error getting item for updating cursor", "error", err.Error(), "itemID", itemID)
// 		return err
// 	}
// 	updateItemParams := database.UpdateItemParams{
// 		TransactionCursor: cursor,
// 		Name:              item.Name,
// 	}
// 	if _, err := queries.UpdateItem(ctx, updateItemParams); err != nil {
// 		slog.Error("error updating item cursor", "error", err.Error(), "itemID", itemID)
// 		return err
// 	}
// 	return nil
// }
