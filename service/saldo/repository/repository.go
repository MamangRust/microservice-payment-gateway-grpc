package repository

import (
	db "github.com/MamangRust/microservice-payment-gateway-grpc/pkg/database/schema"
)

type Repositories interface {
	SaldoQueryRepository
	SaldoCommandRepository
	CardRepository
}

type repositories struct {
	SaldoQueryRepository
	SaldoCommandRepository
	CardRepository
}

func NewRepositories(db *db.Queries, card CardRepository) Repositories {
	return &repositories{
		SaldoQueryRepository:   NewSaldoQueryRepository(db),
		SaldoCommandRepository: NewSaldoCommandRepository(db),
		CardRepository:         card,
	}
}
