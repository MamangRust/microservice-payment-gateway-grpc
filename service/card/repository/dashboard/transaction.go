package repositorydashboard

import (
	"context"

	db "github.com/MamangRust/microservice-payment-gateway-grpc/pkg/database/schema"
	card_errors "github.com/MamangRust/microservice-payment-gateway-grpc/shared/errors/card_errors/repository"
)

type cardDashboardTransactionRepository struct {
	db *db.Queries
}

func NewCardDashboardTransactionRepository(db *db.Queries) CardDashboardTransactionRepository {
	return &cardDashboardTransactionRepository{
		db: db,
	}
}

func (r *cardDashboardTransactionRepository) GetTotalTransactionAmount(ctx context.Context) (*int64, error) {
	res, err := r.db.GetTotalTransactionAmount(ctx)

	if err != nil {
		return nil, card_errors.ErrGetTotalTransactionsFailed.WithInternal(err)
	}

	return &res, nil
}

func (r *cardDashboardTransactionRepository) GetTotalTransactionAmountByCardNumber(ctx context.Context, cardNumber string) (*int64, error) {
	res, err := r.db.GetTotalTransactionAmountByCardNumber(ctx, cardNumber)

	if err != nil {
		return nil, card_errors.ErrGetTotalTransactionAmountByCardFailed.WithInternal(err)
	}

	return &res, nil
}
