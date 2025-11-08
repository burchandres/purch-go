package database

import (
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

// -------- Core Schemas --------

type User struct {
	bun.BaseModel `bun:"table:users,alias:u" json:"-"`

	ID         uuid.UUID `bun:"id,pk,type:uuid,default:gen_random_uuid()" json:"id"`
	FirstName  string    `bun:"first_name,notnull" json:"first_name"`
	LastName   string    `bun:"last_name,notnull" json:"last_name"`
	Username   string    `bun:"username,unique,notnull" json:"username"`
	Password   string    `bun:"password,notnull" json:"password"`
	Income     *float64  `bun:"income,type:numeric" json:"income,omitempty"`
	IncomeRate *string   `bun:"income_rate" json:"income_rate,omitempty"`
}

type Category struct {
	bun.BaseModel `bun:"table:categories,alias:c" json:"-"`

	ID                uuid.UUID `bun:"id,pk,type:uuid,default:gen_random_uuid()" json:"id"`
	UserID            uuid.UUID `bun:"user_id,type:uuid,notnull" json:"user_id"`
	Label             string    `bun:",notnull" json:"label"`
	CurrentSpending   float64   `bun:"current_spending,type:numeric,notnull,default:0" json:"current_spending"`
	AllocatedSpending float64   `bun:"allocated_spending,type:numeric,notnull" json:"allocated_spending"`

	User *User `bun:"rel:belongs-to,join:user_id=id" json:"-"`
}

type Item struct {
	bun.BaseModel `bun:"table:items,alias:i" json:"-"`

	ID                string    `bun:"id,pk" json:"id"`
	UserID            uuid.UUID `bun:"user_id,type:uuid,notnull" json:"user_id"`
	AccessToken       string    `bun:"access_token,notnull" json:"access_token"`
	Name              string    `bun:",notnull" json:"name"`
	TransactionCursor string    `bun:"transaction_cursor,notnull,default:''" json:"transaction_cursor"`

	User *User `bun:"rel:belongs-to,join:user_id=id" json:"-"`
}

type Account struct {
	bun.BaseModel `bun:"table:accounts,alias:a" json:"-"`

	ID               string   `bun:"id,pk" json:"id"`
	ItemID           string   `bun:"item_id,notnull" json:"item_id"`
	Name             string   `bun:"name,notnull" json:"name"`
	AvailableBalance *float64 `bun:"available_balance" json:"available_balance"`
	CurrentBalance   *float64 `bun:"current_balance" json:"current_balance"`
	// Would be any of:
	// - Depository (e.g. checking, savings)
	// - Credit (e.g. credit cards)
	// - Investment (e.g. 401k, IRA, etc.)
	Type string `bun:"type,notnull" json:"type"`
	// A specific example of the type: e.g. If Type=Depository, then SubType=Checking or Savings
	SubType *string `bun:"sub_type" json:"sub_type"`

	Item *Item `bun:"rel:belongs-to,join:item_id=id" json:"-"`
}

type Transaction struct {
	bun.BaseModel `bun:"table:transactions,alias:t" json:"-"`

	ID        string `bun:"id,pk" json:"id"`
	AccountID string `bun:"account_id,notnull" json:"account_id"`
	// TODO: make this a FK mapping to the category's uuid that this transaction falls under
	CategoryLabel  *string    `bun:"category_label" json:"category_label"`
	AuthorizedDate time.Time  `bun:"authorized_date,notnull,default:current_date" json:"authorized_date"`
	SettledDate    *time.Time `bun:"settled_date" json:"settled_date"`
	Merchant       *string    `bun:"merchant" json:"merchant"`
	Amount         float64    `bun:"type:numeric,notnull,default:0" json:"amount"`
	CurrencyCode   *string    `bun:"currency_code" json:"currency_code"`
	Pending        bool       `bun:"pending,notnull,default:false" json:"pending"`

	Account *Account `bun:"rel:belongs-to,join:account_id=id" json:"-"`
	// TODO: add a category for rel:belongs-to
}

// -------- Update Schemas --------

type UpdateUserParams struct {
	FirstName  *string  `json:"first_name,omitempty"`
	LastName   *string  `json:"last_name,omitempty"`
	Username   *string  `json:"username,omitempty"`
	Password   *string  `json:"password,omitempty"`
	Income     *float64 `json:"income,omitempty"`
	IncomeRate *string  `json:"income_rate,omitempty"`
}

type UpdateCategoryParams struct {
	Label             *string  `json:"label,omitempty"`
	CurrentSpending   *float64 `json:"current_spending,omitempty"`
	AllocatedSpending *float64 `json:"allocated_spending,omitempty"`
}

type UpdateItemParams struct {
	AccessToken       *string `json:"access_token,omitempty"`
	Name              *string `json:"name,omitempty"`
	TransactionCursor *string `json:"transaction_cursor,omitempty"`
}

type UpdateAccountParams struct {
	Name *string `json:"name,omitempty"`
}

type UpdateTransactionParams struct {
	CategoryLabel  *string    `json:"category_label,omitempty"`
	AuthorizedDate *time.Time `json:"authorized_date,omitempty"`
	SettledDate    *time.Time `json:"settled_date,omitempty"`
	Merchant       *string    `json:"merchant,omitempty"`
	Amount         *float64   `json:"amount,omitempty"`
	CurrencyCode   *string    `json:"currency_code,omitempty"`
	Pending        *bool      `json:"pending,omitempty"`
}
