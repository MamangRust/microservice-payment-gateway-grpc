package repository

import (
	"context"
	"math"

	db "github.com/MamangRust/microservice-payment-gateway-grpc/service/card/database/schema"
	"github.com/MamangRust/microservice-payment-gateway-grpc/shared/domain/requests"
	sharedErrors "github.com/MamangRust/microservice-payment-gateway-grpc/shared/errors"
	"github.com/google/uuid"
)

type cardPaymentRepository struct {
	db *db.Queries
}

func NewCardPaymentRepository(q *db.Queries) CardPaymentRepository {
	return &cardPaymentRepository{db: q}
}

func (r *cardPaymentRepository) PostPayment(ctx context.Context, req *requests.PostPaymentRequest) (*db.CardPayment, error) {
	var billingID *int32
	if req.BillingID != nil {
		billingID = new(int32)
		*billingID = int32(*req.BillingID)
	}

	res, err := r.db.InsertCardPayment(ctx, db.InsertCardPaymentParams{
		PaymentUuid:    uuid.New().String(),
		CardNumber:     req.CardNumber,
		BillingID:      billingID,
		Amount:         req.Amount,
		PaymentChannel: req.PaymentChannel,
		ReferenceID:    req.ReferenceID,
	})
	if err != nil {
		return nil, sharedErrors.ErrFailed("create card payment").WithInternal(err)
	}
	return res, nil
}

func (r *cardPaymentRepository) GetPaymentHistory(ctx context.Context, cardNumber string, page, pageSize int) ([]*db.GetCardPaymentsByCardNumberRow, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}

	res, err := r.db.GetCardPaymentsByCardNumber(ctx, db.GetCardPaymentsByCardNumberParams{
		CardNumber: cardNumber,
		Limit:      int32(pageSize),
		Offset:     int32(math.Max(0, float64((page-1)*pageSize))),
	})
	if err != nil {
		return nil, sharedErrors.ErrFailed("get card payments").WithInternal(err)
	}
	return res, nil
}

func (r *cardPaymentRepository) CountPayments(ctx context.Context, cardNumber string) (int, error) {
	count, err := r.db.CountCardPayments(ctx, cardNumber)
	if err != nil {
		return 0, sharedErrors.ErrFailed("count card payments").WithInternal(err)
	}
	return int(count), nil
}
