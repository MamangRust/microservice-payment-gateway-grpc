package repository

import (
	db "github.com/MamangRust/microservice-payment-gateway-grpc/pkg/database/schema"
)

type Repositories interface {
	SaldoRepository
	TransferQueryRepository
	TransferCommandRepository
	CardRepository
}

type repositories struct {
	SaldoRepository
	TransferQueryRepository
	TransferCommandRepository
	CardRepository
}

func NewRepositories(
	db *db.Queries,
	saldo SaldoRepository,
	card CardRepository,
) Repositories {
	return &repositories{
		SaldoRepository:               saldo,
		TransferQueryRepository:       NewTransferQueryRepository(db),
		TransferCommandRepository:     NewTransferCommandRepository(db),
		CardRepository:                card,
	}
}
