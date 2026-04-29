package repository

import (
	db "github.com/MamangRust/microservice-payment-gateway-grpc/pkg/database/schema"
)

type Repositories interface {
	CardRepository
	SaldoRepository
	WithdrawQueryRepository
	WithdrawCommandRepository
}

type repositories struct {
	CardRepository
	SaldoRepository
	WithdrawQueryRepository
	WithdrawCommandRepository
}

func NewRepositories(
	db *db.Queries,
	card CardRepository,
	saldo SaldoRepository,
) Repositories {
	return &repositories{
		CardRepository:                card,
		SaldoRepository:               saldo,
		WithdrawQueryRepository:       NewWithdrawQueryRepository(db),
		WithdrawCommandRepository:     NewWithdrawCommandRepository(db),
	}
}
