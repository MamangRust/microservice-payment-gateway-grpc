package repository

import (
	db "github.com/MamangRust/microservice-payment-gateway-grpc/service/withdraw/database/schema"
	"github.com/MamangRust/microservice-payment-gateway-grpc/shared/outbox"
)

type Repositories interface {
	CardRepository
	SaldoRepository
	WithdrawQueryRepository
	WithdrawCommandRepository
	IdempotencyRepository
	OutboxRepository
}

type repositories struct {
	CardRepository
	SaldoRepository
	WithdrawQueryRepository
	WithdrawCommandRepository
	IdempotencyRepository
	OutboxRepository
}

func NewRepositories(
	db *db.Queries,
	card CardRepository,
	saldo SaldoRepository,
) Repositories {
	return &repositories{
		CardRepository:            card,
		SaldoRepository:           saldo,
		WithdrawQueryRepository:   NewWithdrawQueryRepository(db),
		WithdrawCommandRepository: NewWithdrawCommandRepository(db),
		IdempotencyRepository:     NewWithdrawIdempotencyRepository(db),
		OutboxRepository:          outbox.NewStore(db.InsertOutbox),
	}
}
