package repository

import (
	db "github.com/MamangRust/microservice-payment-gateway-grpc/service/transfer/database/schema"
	"github.com/MamangRust/microservice-payment-gateway-grpc/shared/outbox"
)

type Repositories interface {
	SaldoRepository
	TransferQueryRepository
	TransferCommandRepository
	CardRepository
	IdempotencyRepository
	OutboxRepository
}

type repositories struct {
	SaldoRepository
	TransferQueryRepository
	TransferCommandRepository
	CardRepository
	IdempotencyRepository
	OutboxRepository
}

func NewRepositories(
	db *db.Queries,
	saldo SaldoRepository,
	card CardRepository,
) Repositories {
	return &repositories{
		SaldoRepository:           saldo,
		TransferQueryRepository:   NewTransferQueryRepository(db),
		TransferCommandRepository: NewTransferCommandRepository(db),
		CardRepository:            card,
		IdempotencyRepository:     NewTransferIdempotencyRepository(db),
		OutboxRepository:          outbox.NewStore(db.InsertOutbox),
	}
}
