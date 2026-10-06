package repository

import (
	pbcard "github.com/MamangRust/microservice-payment-gateway-grpc/pb/card"
	"github.com/MamangRust/microservice-payment-gateway-grpc/pkg/adapter"
	db "github.com/MamangRust/microservice-payment-gateway-grpc/service/saldo/database/schema"
)

// GuardOptions configures the guarded gRPC adapters used by the saldo
// repositories (e.g. the card adapter).
type GuardOptions struct {
	Card []adapter.GuardOption
}

// Repositories is a struct containing all saldo repositories.
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

func NewRepositories(
	db *db.Queries,
	cardQueryClient pbcard.CardQueryServiceClient,
	cardCommandClient pbcard.CardCommandServiceClient,
	guards ...GuardOptions,
) Repositories {
	var g GuardOptions

	if len(guards) > 0 {
		g = guards[0]
	}

	return &repositories{
		SaldoQueryRepository:   NewSaldoQueryRepository(db),
		SaldoCommandRepository: NewSaldoCommandRepository(db),
		CardRepository:         adapter.NewCardAdapter(cardQueryClient, cardCommandClient, g.Card...),
	}
}
