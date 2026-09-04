package repository

import (
	"context"
	"time"

	db "github.com/MamangRust/microservice-payment-gateway-grpc/service/card/database/schema"
	"github.com/MamangRust/microservice-payment-gateway-grpc/shared/domain/requests"
	sharedErrors "github.com/MamangRust/microservice-payment-gateway-grpc/shared/errors"
)

type cardRewardRepository struct {
	db *db.Queries
}

func NewCardRewardRepository(q *db.Queries) CardRewardRepository {
	return &cardRewardRepository{db: q}
}

func earnPoints(amount int64, mcc string) int32 {
	points := int32(amount / 10000)
	switch mcc {
	case "5812", "4511", "5541":
		points *= 2
	}
	if points < 1 {
		points = 1
	}
	return points
}

func (r *cardRewardRepository) EarnRewards(ctx context.Context, req *requests.EarnRewardsRequest) (*db.CardReward, error) {
	res, err := r.db.InsertCardReward(ctx, db.InsertCardRewardParams{
		CardNumber:   req.CardNumber,
		TxnID:        req.TxnID,
		Amount:       req.Amount,
		Mcc:          req.Mcc,
		PointsEarned: earnPoints(req.Amount, req.Mcc),
		ExpiresAt:    time.Now().AddDate(1, 0, 0),
	})
	if err != nil {
		return nil, sharedErrors.ErrFailed("earn card rewards").WithInternal(err)
	}
	return res, nil
}

func (r *cardRewardRepository) GetBalance(ctx context.Context, cardNumber string) (int64, error) {
	balance, err := r.db.GetCardRewardBalance(ctx, cardNumber)
	if err != nil {
		return 0, sharedErrors.ErrFailed("get reward balance").WithInternal(err)
	}
	return balance, nil
}

func (r *cardRewardRepository) GetHistory(ctx context.Context, cardNumber string) ([]*db.CardReward, error) {
	res, err := r.db.GetCardRewardHistory(ctx, cardNumber)
	if err != nil {
		return nil, sharedErrors.ErrFailed("get reward history").WithInternal(err)
	}
	return res, nil
}

func (r *cardRewardRepository) RedeemRewards(ctx context.Context, cardNumber string, points int64) (int64, error) {
	rows, err := r.db.GetRedeemableRewardIds(ctx, cardNumber)
	if err != nil {
		return 0, sharedErrors.ErrFailed("find redeemable rewards").WithInternal(err)
	}

	var ids []int32
	var total int64
	for _, row := range rows {
		if total >= points {
			break
		}
		ids = append(ids, row.RewardID)
		total += int64(row.PointsEarned)
	}
	if total < points {
		return 0, sharedErrors.ErrBadRequest.WithMessage("Insufficient reward points")
	}

	redeemed, err := r.db.MarkRewardsRedeemed(ctx, ids)
	if err != nil {
		return 0, sharedErrors.ErrFailed("redeem rewards").WithInternal(err)
	}
	return redeemed, nil
}
