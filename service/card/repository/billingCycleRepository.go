package repository

import (
	"context"
	"time"

	db "github.com/MamangRust/microservice-payment-gateway-grpc/service/card/database/schema"
	sharedErrors "github.com/MamangRust/microservice-payment-gateway-grpc/shared/errors"
)

type billingCycleQueries interface {
	GetBillingCyclesByCardNumber(ctx context.Context, cardNumber string) ([]*db.BillingCycle, error)
	CreateBillingCycles(ctx context.Context, arg db.CreateBillingCyclesParams) ([]*db.BillingCycle, error)
}

type billingCycleRepository struct {
	db billingCycleQueries
}

func NewBillingCycleRepository(q billingCycleQueries) BillingCycleRepository {
	return &billingCycleRepository{db: q}
}

func (r *billingCycleRepository) GetBillingCyclesByCardNumber(ctx context.Context, cardNumber string) ([]*db.BillingCycle, error) {
	res, err := r.db.GetBillingCyclesByCardNumber(ctx, cardNumber)
	if err != nil {
		return nil, sharedErrors.ErrFailed("get billing cycles").WithInternal(err)
	}
	return res, nil
}

func (r *billingCycleRepository) CreateBillingCycles(ctx context.Context, cycleStart, cycleEnd, dueDate time.Time) (int, error) {
	created, err := r.db.CreateBillingCycles(ctx, db.CreateBillingCyclesParams{
		CycleStart: cycleStart,
		CycleEnd:   cycleEnd,
		DueDate:    dueDate,
	})
	if err != nil {
		return 0, sharedErrors.ErrFailed("create billing cycles").WithInternal(err)
	}
	return len(created), nil
}
