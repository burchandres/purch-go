package database

import (
	"github.com/uptrace/bun"
	"time"
)

// -------- Core Schemas --------

type User struct {
	bun.BaseModel `bun:"table:users,alias:u"`

	ID         string   `bun:"id,pk,default:gen_random_uuid()"`
	FirstName  string   `bun:"first_name,notnull"`
	LastName   string   `bun:"last_name,notnull"`
	Username   string   `bun:",unique,notnull"`
	Password   string   `bun:",notnull"`
	Income     *float64 `bun:"type:numeric"`
	IncomeRate *string  `bun:"income_rate"`
}

type Category struct {
	bun.BaseModel `bun:"table:categories,alias:c"`

	ID                string  `bun:",pk,default:gen_random_uuid()"`
	UserID            string  `bun:",notnull"`
	Label             string  `bun:",notnull"`
	CurrentSpending   float64 `bun:"type:numeric,notnull,default:0"`
	AllocatedSpending float64 `bun:"type:numeric,notnull"`

	User *User `bun:"rel:belongs-to,join:user_id=id"`
}

type Item struct {
	bun.BaseModel `bun:"table:items,alias:i"`

	ID                string `bun:",pk"`
	UserID            string `bun:",notnull"`
	AccessToken       string `bun:"access_token,notnull"`
	Name              string `bun:",notnull"`
	TransactionCursor string `bun:"transaction_cursor,notnull,default:''"`

	User *User `bun:"rel:belongs-to,join:user_id=id"`
}

type Account struct {
	bun.BaseModel `bun:"table:accounts,alias:a"`

	ID     string `bun:",pk"`
	ItemID string `bun:"item_id,notnull"`
	Name   string `bun:",notnull"`

	Item *Item `bun:"rel:belongs-to,join:item_id=id"`
}

type Transaction struct {
	bun.BaseModel `bun:"table:transactions,alias:t"`

	ID             string     `bun:",pk"`
	AccountID      string     `bun:"account_id,notnull"`
	CategoryLabel  *string    `bun:"category_label"`
	AuthorizedDate time.Time  `bun:"authorized_date,notnull,default:current_date"`
	SettledDate    *time.Time `bun:"settled_date"`
	Merchant       *string    `bun:"merchant"`
	Amount         float64    `bun:"type:numeric,notnull,default:0"`
	CurrencyCode   *string    `bun:"currency_code"`
	Pending        bool       `bun:",notnull,default:false"`

	Account *Account `bun:"rel:belongs-to,join:account_id=id"`
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
