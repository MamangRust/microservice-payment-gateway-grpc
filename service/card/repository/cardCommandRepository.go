package repository

import (
	"context"

	db "github.com/MamangRust/microservice-payment-gateway-grpc/pkg/database/schema"
	"github.com/MamangRust/microservice-payment-gateway-grpc/pkg/randomvcc"
	"github.com/MamangRust/microservice-payment-gateway-grpc/shared/domain/requests"
	sharedErrors "github.com/MamangRust/microservice-payment-gateway-grpc/shared/errors"
	card_errors "github.com/MamangRust/microservice-payment-gateway-grpc/shared/errors/card_errors/repository"
	"github.com/jackc/pgx/v5/pgtype"
)

type cardCommandRepository struct {
	db *db.Queries
}

func NewCardCommandRepository(db *db.Queries) CardCommandRepository {
	return &cardCommandRepository{
		db: db,
	}
}

func (r *cardCommandRepository) CreateCard(ctx context.Context, request *requests.CreateCardRequest) (*db.CreateCardRow, error) {
	number, err := randomvcc.RandomCardNumber()
	if err != nil {
		return nil, sharedErrors.ErrInternal.WithInternal(err)
	}

	expireDate := pgtype.Date{
		Time:  request.ExpireDate,
		Valid: true,
	}

	req := db.CreateCardParams{
		UserID:       int32(request.UserID),
		CardNumber:   number,
		CardType:     request.CardType,
		ExpireDate:   expireDate,
		Cvv:          request.CVV,
		CardProvider: request.CardProvider,
	}

	res, err := r.db.CreateCard(ctx, req)
	if err != nil {
		return nil, card_errors.ErrCreateCardFailed.WithInternal(err)
	}

	return res, nil
}

func (r *cardCommandRepository) UpdateCard(ctx context.Context, request *requests.UpdateCardRequest) (*db.UpdateCardRow, error) {
	expireDate := pgtype.Date{
		Time:  request.ExpireDate,
		Valid: true,
	}

	req := db.UpdateCardParams{
		CardID:       int32(request.CardID),
		CardType:     request.CardType,
		ExpireDate:   expireDate,
		Cvv:          request.CVV,
		CardProvider: request.CardProvider,
	}

	res, err := r.db.UpdateCard(ctx, req)
	if err != nil {
		return nil, card_errors.ErrUpdateCardFailed.WithInternal(err)
	}

	return res, nil
}

func (r *cardCommandRepository) TrashedCard(ctx context.Context, card_id int) (*db.Card, error) {
	res, err := r.db.TrashCard(ctx, int32(card_id))

	if err != nil {
		return nil, card_errors.ErrTrashCardFailed.WithInternal(err)
	}

	return res, nil
}

func (r *cardCommandRepository) RestoreCard(ctx context.Context, card_id int) (*db.Card, error) {
	res, err := r.db.RestoreCard(ctx, int32(card_id))

	if err != nil {
		return nil, card_errors.ErrRestoreCardFailed.WithInternal(err)
	}

	return res, nil
}

func (r *cardCommandRepository) DeleteCardPermanent(ctx context.Context, card_id int) (bool, error) {
	err := r.db.DeleteCardPermanently(ctx, int32(card_id))

	if err != nil {
		return false, card_errors.ErrDeleteCardPermanentFailed.WithInternal(err)
	}

	return true, nil
}

func (r *cardCommandRepository) RestoreAllCard(ctx context.Context) (bool, error) {
	err := r.db.RestoreAllCards(ctx)

	if err != nil {
		return false, card_errors.ErrRestoreAllCardsFailed.WithInternal(err)
	}

	return true, nil
}

func (r *cardCommandRepository) DeleteAllCardPermanent(ctx context.Context) (bool, error) {
	err := r.db.DeleteAllPermanentCards(ctx)

	if err != nil {
		return false, card_errors.ErrDeleteAllCardsPermanentFailed.WithInternal(err)
	}

	return true, nil
}
