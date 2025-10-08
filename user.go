package main 

import (
	"gorm.io/gorm"
)

var (
	// to be used to verify provided income rate for a user
	IncomeRates = [6]string{"hourly", "weekly", "biweekly", "bimonthly", "monthly", "annually"}
)

// User.ID is the Primary Key for postgres 
// and is pointed to from Foreign Key columns of Item table
// and float64 is used for income/transaction amounts but decimal package is used for arithmetic
type User struct {
	gorm.Model 

	FirstName string 
	LastName string
	Username string
	Password string
	IsActive bool
	Income float64 // left as float64 since plaid transaction amount uses float64
	IncomeRate string
}
