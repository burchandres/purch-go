package database

import (
	"context"
	// "database/sql"
	"purch/internal/config"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// TODO: refactor all this into BudgetService and UserService structs for dependency injection pattern

// -------- User Queries --------

func GetUserByUsername(ctx context.Context, username string) (User, error) {
	var user User
	err := db.NewSelect().
		Model(&user).
		Where("username = ?", username).
		Scan(ctx)
	return user, err
}

func GetUserByID(ctx context.Context, id uuid.UUID) (User, error) {
	var user User
	err := db.NewSelect().
		Model(&user).
		Where("id = ?", id).
		Scan(ctx)
	return user, err
}

func StoreUser(ctx context.Context, user User) error {
	return db.NewInsert().Model(&user).Scan(ctx)
}

func UpdateUser(ctx context.Context, id uuid.UUID, updateParams UpdateUserParams) error {
	config := config.GetConfig()
	stmt := db.NewUpdate().Model((*User)(nil))
	// build up query
	if updateParams.FirstName != nil {
		stmt = stmt.Set("first_name = ?", *updateParams.FirstName)
	}
	if updateParams.LastName != nil {
		stmt = stmt.Set("last_name = ?", *updateParams.LastName)
	}
	if updateParams.Username != nil {
		stmt = stmt.Set("username = ?", *updateParams.Username)
	}
	if updateParams.Password != nil {
		newPassword := *(updateParams.Password)
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), config.BcryptCost)
		if err != nil {
			return err
		}
		stmt = stmt.Set("password = ?", string(hashedPassword))
	}
	if updateParams.Income != nil {
		stmt = stmt.Set("income = ?", *updateParams.Income)
	}
	if updateParams.IncomeRate != nil {
		stmt = stmt.Set("income_rate = ?", *updateParams.IncomeRate)
	}
	_, err := stmt.Where("id = ?", id).Exec(ctx)
	return err
}

func DeleteUser(ctx context.Context, user User) error {
	_, err := db.NewDelete().
		Model(&user).
		WherePK().
		Exec(ctx)
	return err
}

func GetUserCategories(ctx context.Context, userID uuid.UUID) ([]Category, error) {
	var categories []Category
	err := db.NewSelect().
		Model(&categories).
		Where("user_id = ?", userID).
		Scan(ctx)
	return categories, err
}

func GetUserItems(ctx context.Context, userID uuid.UUID) ([]Item, error) {
	var items []Item
	err := db.NewSelect().
		Model(&items).
		Where("user_id = ?", userID).
		Scan(ctx)
	return items, err
}

func GetUserAccounts(ctx context.Context, userID uuid.UUID) ([]Account, error) {
	var accounts []Account
	err := db.NewSelect().
		Model(&accounts).
		ColumnExpr("a.*").
		Join("JOIN items AS i").JoinOn("i.id = a.item_id").
		Where("i.user_id = ?", userID).
		Scan(ctx)
	return accounts, err
}

func GetUserTransactions(ctx context.Context, userID uuid.UUID) ([]Transaction, int, error) {
	var transactions []Transaction
	count, err := db.NewSelect().
		Model(&transactions).
		ColumnExpr("t.*").
		Join("JOIN accounts AS a").JoinOn("a.id = t.account_id").
		Join("JOIN items AS i").JoinOn("i.id = a.item_id").
		Where("i.user_id = ?", userID).
		// Order("t.authorized_date").
		ScanAndCount(ctx)
	return transactions, count, err
}

// -------- Item Queries --------

// TODO: update this to decrypt the item's accountToken once encrypting it is implemented
func GetItem(ctx context.Context, id string) (Item, error) {
	var item Item
	err := db.NewSelect().
		Model(&item).
		Where("id = ?", id).
		Scan(ctx)
	return item, err
}

func UpdateItemCursor(ctx context.Context, cursor, itemID string) error {
	// var results sql.Result
	_, err := db.NewUpdate().
		Model((*Item)(nil)).
		Set("transaction_cursor = ?", cursor).
		Where("id = ?", itemID).
		Returning("NULL").
		Exec(ctx)
	return err
}

func StoreItem(ctx context.Context, item Item) error {
	return db.NewInsert().Model(&item).Scan(ctx)
}

// -------- Account Queries --------

func GetAccount(ctx context.Context, id string) (Account, error) {
	var account Account
	err := db.NewSelect().
		Model(&account).
		Where("id = ?", id).
		Scan(ctx)
	return account, err
}

// Insert a single account
func StoreAccount(ctx context.Context, account Account) error {
	return db.NewInsert().Model(&account).Scan(ctx)
}

// Batch insert a slice of accounts
func StoreAccounts(ctx context.Context, accounts []Account) error {
	return db.NewInsert().Model(&accounts).Scan(ctx)
}

// -------- Transaction Queries --------

func GetTransaction(ctx context.Context, id string) (Transaction, error) {
	var transaction Transaction
	err := db.NewSelect().
		Model(&transaction).
		Where("id = ?", id).
		Scan(ctx)
	return transaction, err
}

func StoreTransactions(ctx context.Context, transactions []Transaction) error {
	return db.NewInsert().Model(&transactions).Scan(ctx)
}

// Bulk update transactions. Updates their amount, settled_date and pending status.
// Requires the inputted list to have those fields populated so data doesn't become corrupted along with transaction id.
func UpdateTransactions(ctx context.Context, transactions []Transaction) error {
	values := db.NewValues(&transactions)
	return db.NewUpdate().
		With("_data", values).
		Model((*Transaction)(nil)).
		TableExpr("_data").
		Set("amount = _data.amount").
		Set("settled_date = _data.settled_date").
		Set("pending = _data.pending").
		Where("id = _data.id").
		Scan(ctx)
}

func DeleteTransactions(ctx context.Context, ids []string) error {
	return db.NewDelete().Model((*Transaction)(nil)).Where("id IN ?", ids).Scan(ctx)
}
