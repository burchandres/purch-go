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

	Categories []*Category `bun:"rel:has-many,join:id=user_id" json:"-"`
	Items      []*Item     `bun:"rel:has-many,join:id=user_id" json:"-"`
}

type Category struct {
	bun.BaseModel `bun:"table:categories,alias:c"`

	ID                uuid.UUID `bun:"id,pk,type:uuid,default:gen_random_uuid()"`
	UserID            uuid.UUID `bun:"user_id,type:uuid,notnull"`
	Label             string    `bun:",notnull"`
	CurrentSpending   float64   `bun:"current_spending,type:numeric,notnull,default:0"`
	AllocatedSpending float64   `bun:"allocated_spending,type:numeric,notnull"`

	User *User `bun:"rel:belongs-to,join:user_id=id"`
}

type Item struct {
	bun.BaseModel `bun:"table:items,alias:i"`

	ID                string    `bun:"id,pk"`
	UserID            uuid.UUID `bun:"user_id,type:uuid,notnull"`
	AccessToken       string    `bun:"access_token,notnull"`
	Name              string    `bun:",notnull"`
	TransactionCursor string    `bun:"transaction_cursor,notnull,default:''"`

	User     *User      `bun:"rel:belongs-to,join:user_id=id"`
	Accounts []*Account `bun:"rel:has-many,join:id=item_id"`
}

type Account struct {
	bun.BaseModel `bun:"table:accounts,alias:a"`

	ID     string `bun:"id,pk"`
	ItemID string `bun:"item_id,notnull"`
	Name   string `bun:"name,notnull"`

	Item         *Item          `bun:"rel:belongs-to,join:item_id=id"`
	Transactions []*Transaction `bun:"rel:has-many,join:id=account_id"`
}

type Transaction struct {
	bun.BaseModel `bun:"table:transactions,alias:t"`

	ID             string     `bun:"id,pk"`
	AccountID      string     `bun:"account_id,notnull"`
	CategoryLabel  *string    `bun:"category_label"`
	AuthorizedDate time.Time  `bun:"authorized_date,notnull,default:current_date"`
	SettledDate    *time.Time `bun:"settled_date"`
	Merchant       *string    `bun:"merchant"`
	Amount         float64    `bun:"type:numeric,notnull,default:0"`
	CurrencyCode   *string    `bun:"currency_code"`
	Pending        bool       `bun:"pending,notnull,default:false"`

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
