// Package models holds transport-agnostic domain types shared between
// services. Adapters in pkg/adapter map gRPC responses onto these types so
// consumers never depend on another service's database schema package.
package models

import "time"

// User mirrors the user service's public user representation.
type User struct {
	UserID    int32
	Firstname string
	Lastname  string
	Email     string
	Password  string
	CreatedAt *time.Time
	UpdatedAt *time.Time
}

// Card mirrors the card service's public card representation.
type Card struct {
	CardID       int32
	UserID       int32
	CardNumber   string
	CardType     string
	ExpireDate   *time.Time
	Cvv          string
	CardProvider string
	Email        string
	CreatedAt    *time.Time
	UpdatedAt    *time.Time
}

// Merchant mirrors the merchant service's public merchant representation.
type Merchant struct {
	MerchantID int32
	Name       string
	ApiKey     string
	UserID     int32
	Status     string
	CreatedAt  *time.Time
	UpdatedAt  *time.Time
}

// Saldo mirrors the saldo service's public balance representation.
type Saldo struct {
	SaldoID        int32
	CardNumber     string
	TotalBalance   int64
	WithdrawAmount *int64
	WithdrawTime   *time.Time
	CreatedAt      *time.Time
	UpdatedAt      *time.Time
}

// SaldoMutationResult is the reduced balance payload returned by saldo
// mutation calls (debit, credit, withdraw, balance update).
type SaldoMutationResult struct {
	SaldoID      int32
	CardNumber   string
	TotalBalance int64
}

// Role mirrors the role service's public role representation.
type Role struct {
	RoleID   int32
	RoleName string
}

// UserRole is the assignment between a user and a role.
type UserRole struct {
	UserID int32
	RoleID int32
}

// Transaction mirrors the transaction service's public transaction
// representation. MerchantName is filled in by the owning merchant service
// after the cross-owner read, not by the transaction service itself.
type Transaction struct {
	TransactionID   int
	TransactionNo   string
	CardNumber      string
	Amount          int64
	PaymentMethod   string
	MerchantID      int
	MerchantName    string
	TransactionTime time.Time
	CreatedAt       time.Time
	UpdatedAt       time.Time
	DeletedAt       *time.Time
}
