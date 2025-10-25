package database

import (
	"context"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// TODO: refactor all this into BudgetService and UserService structs

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
	return db.NewInsert().Model(user).Scan(ctx)
}

func UpdateUser(ctx context.Context, id uuid.UUID, updateParams UpdateUserParams) error {
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
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(*updateParams.Password), bcrypt.DefaultCost)
		if err != nil {
			return err
		}
		stmt = stmt.Set("password = ?", hashedPassword)
	}
	if updateParams.Income != nil {
		stmt = stmt.Set("income = ?", *updateParams.Income)
	}
	if updateParams.IncomeRate != nil {
		stmt = stmt.Set("income_rate = ?", *updateParams.IncomeRate)
	}
	return stmt.Where("id = ?", id).Scan(ctx)
}

func DeleteUser(ctx context.Context, user User) error {
	_, err := db.NewDelete().
		Model(user).
		WherePK().
		Exec(ctx)
	return err
}

// -------- Item Queries --------

func GetItem(ctx context.Context, id string) (Item, error) {
	var item Item
	err := db.NewSelect().
		Model(&item).
		Where("id = ?", id).
		Scan(ctx)
	return item, err
}

func StoreItem(ctx context.Context, item Item) error {
	return db.NewInsert().Model(item).Scan(ctx)
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

func StoreAccounts(ctx context.Context, accounts []*Account) error {
	return db.NewInsert().Model(accounts).Scan(ctx)
}
