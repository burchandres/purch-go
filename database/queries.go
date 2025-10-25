package database

import (
	"context"

	"golang.org/x/crypto/bcrypt"
)

// TODO: refactor all this into BudgetService and UserService structs

func GetUserByUsername(ctx context.Context, username string) (User, error) {
	var user User
	err := db.NewSelect().
		Model(&user).
		Where("username = ?", username).
		Scan(ctx)
	return user, err
}

func GetUserByID(ctx context.Context, id string) (User, error) {
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

func UpdateUser(ctx context.Context, id string, updateParams UpdateUserParams) error {
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
