package repository

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	database "github.com/MamangRust/microservice-payment-gateway-grpc/pkg/database"
	db "github.com/MamangRust/microservice-payment-gateway-grpc/service/card/database/schema"
	"github.com/MamangRust/microservice-payment-gateway-grpc/pkg/randomvcc"
	"github.com/MamangRust/microservice-payment-gateway-grpc/shared/domain/requests"
	sharedErrors "github.com/MamangRust/microservice-payment-gateway-grpc/shared/errors"
)

type cardCommandRepository struct {
	db *db.Queries
}

func NewCardCommandRepository(db *db.Queries) CardCommandRepository {
	return &cardCommandRepository{db: db}
}

func (r *cardCommandRepository) CreateCard(ctx context.Context, request *requests.CreateCardRequest) (*db.CreateCardRow, error) {
	number, err := randomvcc.RandomCardNumber()
	if err != nil {
		return nil, sharedErrors.ErrInternal.WithInternal(err)
	}

	res, err := r.db.CreateCard(ctx, db.CreateCardParams{
		UserID:       int32(request.UserID),
		CardNumber:   number,
		CardType:     request.CardType,
		ExpireDate:   pgtype.Date{Time: request.ExpireDate, Valid: true},
		Cvv:          request.CVV,
		CardProvider: request.CardProvider,
	})
	if err != nil {
		if database.IsUniqueViolation(err) {
			return nil, sharedErrors.NewConflictError("card number already exists").WithInternal(err)
		}
		return nil, sharedErrors.ErrFailed("create card").WithInternal(err)
	}
	return res, nil
}

func (r *cardCommandRepository) UpdateCard(ctx context.Context, request *requests.UpdateCardRequest) (*db.UpdateCardRow, error) {
	res, err := r.db.UpdateCard(ctx, db.UpdateCardParams{
		CardID:       int32(request.CardID),
		CardType:     request.CardType,
		ExpireDate:   pgtype.Date{Time: request.ExpireDate, Valid: true},
		Cvv:          request.CVV,
		CardProvider: request.CardProvider,
	})
	if err != nil {
		// The update query only returns rows for existing, non-deleted cards.
		// A missing row means the card does not exist (or was soft-deleted), so
		// surface a 404 instead of a misleading 500 INTERNAL from pgx.ErrNoRows.
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, sharedErrors.ErrNotFoundResponse("card").WithInternal(err)
		}
		return nil, sharedErrors.ErrFailed("update card").WithInternal(err)
	}
	return res, nil
}

func (r *cardCommandRepository) TrashedCard(ctx context.Context, cardID int) (*db.Card, error) {
	res, err := r.db.TrashCard(ctx, int32(cardID))
	if err != nil {
		// The trash query only returns rows for existing, non-deleted cards.
		// A missing row means the card does not exist, was soft-deleted, or is
		// already trashed — surface a 404 instead of a misleading 500 INTERNAL
		// from pgx.ErrNoRows.
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, sharedErrors.ErrNotFoundResponse("card").WithInternal(err)
		}
		return nil, sharedErrors.ErrFailed("trash card").WithInternal(err)
	}
	return &db.Card{
		CardID: res.CardID, UserID: res.UserID, CardNumber: res.CardNumber,
		CardType: res.CardType, ExpireDate: res.ExpireDate, Cvv: res.Cvv,
		CardProvider: res.CardProvider, CreatedAt: res.CreatedAt,
		UpdatedAt: res.UpdatedAt, DeletedAt: res.DeletedAt,
	}, nil
}

func (r *cardCommandRepository) RestoreCard(ctx context.Context, cardID int) (*db.Card, error) {
	res, err := r.db.RestoreCard(ctx, int32(cardID))
	if err != nil {
		// The restore query only returns rows for existing, currently-trashed
		// cards. A missing row means the card does not exist or is not in a
		// trashed state — surface a 404 instead of a misleading 500 INTERNAL
		// from pgx.ErrNoRows.
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, sharedErrors.ErrNotFoundResponse("card").WithInternal(err)
		}
		return nil, sharedErrors.ErrFailed("restore card").WithInternal(err)
	}
	return &db.Card{
		CardID: res.CardID, UserID: res.UserID, CardNumber: res.CardNumber,
		CardType: res.CardType, ExpireDate: res.ExpireDate, Cvv: res.Cvv,
		CardProvider: res.CardProvider, CreatedAt: res.CreatedAt,
		UpdatedAt: res.UpdatedAt, DeletedAt: res.DeletedAt,
	}, nil
}

func (r *cardCommandRepository) DeleteCardPermanent(ctx context.Context, cardID int) (bool, error) {
	if err := r.db.DeleteCardPermanently(ctx, int32(cardID)); err != nil {
		// Map a missing row to 404 for consistency with the other card
		// mutations instead of a misleading 500 INTERNAL from pgx.ErrNoRows.
		return false, sharedErrors.ErrNoRowsOrFailed(err, "card", "delete card permanently")
	}
	return true, nil
}

func (r *cardCommandRepository) RestoreAllCard(ctx context.Context) (bool, error) {
	if err := r.db.RestoreAllCards(ctx); err != nil {
		return false, sharedErrors.ErrFailed("restore all cards").WithInternal(err)
	}
	return true, nil
}

func (r *cardCommandRepository) DeleteAllCardPermanent(ctx context.Context) (bool, error) {
	if err := r.db.DeleteAllPermanentCards(ctx); err != nil {
		return false, sharedErrors.ErrFailed("delete all cards permanently").WithInternal(err)
	}
	return true, nil
}

func (r *cardCommandRepository) ToggleCardStatus(ctx context.Context, request *requests.ToggleCardStatusRequest) (*db.UpdateCardStatusRow, error) {
	// The target schema extension performs the toggle atomically, so concurrent
	// requests cannot overwrite a status read by a previous request.
	res, err := r.db.ToggleCardStatus(ctx, int32(request.CardID))
	if err != nil {
		// The toggle query only returns rows for existing, non-deleted cards.
		// A missing row means the card does not exist (or was soft-deleted), so
		// surface a 404 instead of a misleading 500 INTERNAL from pgx.ErrNoRows.
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, sharedErrors.ErrNotFoundResponse("card").WithInternal(err)
		}
		return nil, sharedErrors.ErrFailed("update card status").WithInternal(err)
	}
	return &db.UpdateCardStatusRow{
		CardID: res.CardID, UserID: res.UserID, CardNumber: res.CardNumber,
		CardType: res.CardType, ExpireDate: res.ExpireDate, Cvv: res.Cvv,
		CardProvider: res.CardProvider, Status: res.Status,
		CreditLimit: res.CreditLimit, OutstandingBalance: res.OutstandingBalance,
		RewardPoints: res.RewardPoints, CreatedAt: res.CreatedAt, UpdatedAt: res.UpdatedAt,
	}, nil
}

func (r *cardCommandRepository) UpdateCreditLimit(ctx context.Context, request *requests.UpdateCreditLimitRequest) (*db.UpdateCreditLimitRow, error) {
	res, err := r.db.UpdateCreditLimit(ctx, db.UpdateCreditLimitParams{
		CardID:      int32(request.CardID),
		CreditLimit: int32(request.CreditLimit),
	})
	if err != nil {
		// The credit-limit query only returns rows for existing, non-deleted
		// cards. A missing row means the card does not exist (or was
		// soft-deleted), so surface a 404 instead of a misleading 500 INTERNAL
		// from pgx.ErrNoRows.
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, sharedErrors.ErrNotFoundResponse("card").WithInternal(err)
		}
		return nil, sharedErrors.ErrFailed("update credit limit").WithInternal(err)
	}
	return res, nil
}

func (r *cardCommandRepository) RedeemPoints(ctx context.Context, request *requests.RedeemPointsRequest) (*db.RedeemRewardPointsRow, error) {
	res, err := r.db.RedeemRewardPoints(ctx, db.RedeemRewardPointsParams{
		CardID:       int32(request.CardID),
		RewardPoints: int32(request.Points),
	})
	if err != nil {
		// The redeem query only returns rows when reward_points >= points. A
		// missing row means either the card does not exist (or is deleted) or
		// the balance is insufficient. Distinguish the two so callers get a
		// 404 for an unknown card and a 400 for an exhausted balance instead of
		// a misleading 500 INTERNAL from pgx.ErrNoRows.
		if errors.Is(err, pgx.ErrNoRows) {
			if _, cardErr := r.db.GetCardByID(ctx, int32(request.CardID)); cardErr != nil {
				if errors.Is(cardErr, pgx.ErrNoRows) {
					return nil, sharedErrors.ErrNotFoundResponse("card").WithInternal(err)
				}
				return nil, sharedErrors.ErrFailed("redeem reward points").WithInternal(cardErr)
			}
			return nil, sharedErrors.NewBadRequestError("insufficient reward points").WithInternal(err)
		}
		return nil, sharedErrors.ErrFailed("redeem reward points").WithInternal(err)
	}
	return res, nil
}
