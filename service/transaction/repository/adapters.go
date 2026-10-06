package repository

import (
	"github.com/MamangRust/microservice-payment-gateway-grpc/pkg/adapter"
	db "github.com/MamangRust/microservice-payment-gateway-grpc/service/transaction/database/schema"
	"github.com/MamangRust/microservice-payment-gateway-grpc/shared/outbox"
)

// CardRepository exposes the card gRPC adapter to the transaction repository layer.
type CardRepository = adapter.CardAdapter

// SaldoRepository exposes the saldo gRPC adapter to the transaction repository layer.
type SaldoRepository = adapter.SaldoAdapter

// MerchantRepository exposes the merchant gRPC adapter to the transaction repository layer.
type MerchantRepository = adapter.MerchantAdapter

// NewOutboxGormStore builds the OutboxRepository backed by the service's own
// generated outbox insert query.
func NewOutboxGormStore(queries *db.Queries) OutboxRepository {
	return outbox.NewStore(queries.InsertOutbox)
}
