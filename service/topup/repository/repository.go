package repository

import (
	db "github.com/MamangRust/microservice-payment-gateway-grpc/service/topup/database/schema"
	"github.com/MamangRust/microservice-payment-gateway-grpc/shared/outbox"
)

type Repositories interface {
	TopupQueryRepository
	TopupCommandRepository
	CardRepository
	SaldoRepository
	IdempotencyRepository
	OutboxRepository
}

type repositories struct {
	TopupQueryRepository
	TopupCommandRepository
	CardRepository
	SaldoRepository
	IdempotencyRepository
	OutboxRepository
}

func NewRepositories(
	db *db.Queries,
	card CardRepository,
	saldo SaldoRepository,
) Repositories {
	return &repositories{
		TopupQueryRepository:   NewTopupQueryRepository(db),
		TopupCommandRepository: NewTopupCommandRepository(db),
		CardRepository:         card,
		SaldoRepository:        saldo,
		IdempotencyRepository:  NewTopupIdempotencyRepository(db),
		OutboxRepository:       outbox.NewStore(db.InsertOutbox),
	}
}
