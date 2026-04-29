package repository

import (
	db "github.com/MamangRust/microservice-payment-gateway-grpc/pkg/database/schema"
)

type Repositories interface {
	TopupQueryRepository
	TopupCommandRepository
	CardRepository
	SaldoRepository
}

type repositories struct {
	TopupQueryRepository
	TopupCommandRepository
	CardRepository
	SaldoRepository
}

func NewRepositories(
	db *db.Queries,
	card CardRepository,
	saldo SaldoRepository,
) Repositories {
	return &repositories{
		TopupQueryRepository:       NewTopupQueryRepository(db),
		TopupCommandRepository:     NewTopupCommandRepository(db),
		CardRepository:             card,
		SaldoRepository:            saldo,
	}
}
