package repositorydashboard

import (
	"context"

	db "github.com/MamangRust/microservice-payment-gateway-grpc/pkg/database/schema"
	card_errors "github.com/MamangRust/microservice-payment-gateway-grpc/shared/errors/card_errors/repository"
)

type cardDashboardTransferRepository struct {
	db *db.Queries
}

func NewCardDashboardTransferRepository(db *db.Queries) CardDashboardTransferRepository {
	return &cardDashboardTransferRepository{
		db: db,
	}
}

func (r *cardDashboardTransferRepository) GetTotalTransferAmount(ctx context.Context) (*int64, error) {
	res, err := r.db.GetTotalTransferAmount(ctx)

	if err != nil {
		return nil, card_errors.ErrGetTotalTransfersFailed.WithInternal(err)
	}

	return &res, nil
}

func (r *cardDashboardTransferRepository) GetTotalTransferAmountBySender(ctx context.Context, senderCardNumber string) (*int64, error) {
	res, err := r.db.GetTotalTransferAmountBySender(ctx, senderCardNumber)

	if err != nil {
		return nil, card_errors.ErrGetTotalTransferSenderByCardFailed.WithInternal(err)
	}

	return &res, nil
}

func (r *cardDashboardTransferRepository) GetTotalTransferAmountByReceiver(ctx context.Context, receiverCardNumber string) (*int64, error) {
	res, err := r.db.GetTotalTransferAmountByReceiver(ctx, receiverCardNumber)

	if err != nil {
		return nil, card_errors.ErrGetTotalTransferSenderByCardFailed.WithInternal(err)
	}

	return &res, nil
}
