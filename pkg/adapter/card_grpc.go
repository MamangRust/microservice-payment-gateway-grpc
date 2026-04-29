package adapter

import (
	"context"
	"time"

	pbcard "github.com/MamangRust/microservice-payment-gateway-grpc/pb/card"
	db "github.com/MamangRust/microservice-payment-gateway-grpc/pkg/database/schema"
	"github.com/MamangRust/microservice-payment-gateway-grpc/service/card/repository"
	"github.com/MamangRust/microservice-payment-gateway-grpc/shared/domain/requests"
	"github.com/jackc/pgx/v5/pgtype"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type CardAdapter interface {
	FindCardByUserId(ctx context.Context, user_id int) (*db.GetCardByUserIDRow, error)
	FindUserCardByCardNumber(ctx context.Context, card_number string) (*db.GetUserEmailByCardNumberRow, error)
	FindCardByCardNumber(ctx context.Context, card_number string) (*db.GetCardByCardNumberRow, error)
	UpdateCard(ctx context.Context, request *requests.UpdateCardRequest) (*db.UpdateCardRow, error)
}

type cardGRPCAdapter struct {
	QueryClient   pbcard.CardQueryServiceClient
	CommandClient pbcard.CardCommandServiceClient
}

func NewCardAdapter(queryClient pbcard.CardQueryServiceClient, commandClient pbcard.CardCommandServiceClient) CardAdapter {
	return &cardGRPCAdapter{
		QueryClient:   queryClient,
		CommandClient: commandClient,
	}
}

func (a *cardGRPCAdapter) FindCardByUserId(ctx context.Context, user_id int) (*db.GetCardByUserIDRow, error) {
	resp, err := a.QueryClient.FindByUserIdCard(ctx, &pbcard.FindByUserIdCardRequest{
		UserId: int32(user_id),
	})
	if err != nil {
		return nil, err
	}

	return &db.GetCardByUserIDRow{
		CardID:       resp.Data.Id,
		UserID:       resp.Data.UserId,
		CardNumber:   resp.Data.CardNumber,
		CardType:     resp.Data.CardType,
		ExpireDate:   parseDate(resp.Data.ExpireDate),
		Cvv:          resp.Data.Cvv,
		CardProvider: resp.Data.CardProvider,
	}, nil
}

func (a *cardGRPCAdapter) FindUserCardByCardNumber(ctx context.Context, card_number string) (*db.GetUserEmailByCardNumberRow, error) {
	resp, err := a.QueryClient.FindUserCardByCardNumber(ctx, &pbcard.FindByCardNumberRequest{
		CardNumber: card_number,
	})
	if err != nil {
		return nil, err
	}

	return &db.GetUserEmailByCardNumberRow{
		CardNumber: resp.CardNumber,
		Email:      resp.Email,
	}, nil
}

func (a *cardGRPCAdapter) FindCardByCardNumber(ctx context.Context, card_number string) (*db.GetCardByCardNumberRow, error) {
	resp, err := a.QueryClient.FindByCardNumber(ctx, &pbcard.FindByCardNumberRequest{
		CardNumber: card_number,
	})
	if err != nil {
		return nil, err
	}

	return &db.GetCardByCardNumberRow{
		CardID:       resp.Data.Id,
		UserID:       resp.Data.UserId,
		CardNumber:   resp.Data.CardNumber,
		CardType:     resp.Data.CardType,
		ExpireDate:   parseDate(resp.Data.ExpireDate),
		Cvv:          resp.Data.Cvv,
		CardProvider: resp.Data.CardProvider,
	}, nil
}

func (a *cardGRPCAdapter) UpdateCard(ctx context.Context, request *requests.UpdateCardRequest) (*db.UpdateCardRow, error) {
	resp, err := a.CommandClient.UpdateCard(ctx, &pbcard.UpdateCardRequest{
		CardId:       int32(request.CardID),
		UserId:       int32(request.UserID),
		CardType:     request.CardType,
		ExpireDate:   timestamppb.New(request.ExpireDate),
		Cvv:          request.CVV,
		CardProvider: request.CardProvider,
	})

	if err != nil {
		return nil, err
	}

	return &db.UpdateCardRow{
		CardID:       resp.Data.Id,
		UserID:       resp.Data.UserId,
		CardNumber:   resp.Data.CardNumber,
		CardType:     resp.Data.CardType,
		ExpireDate:   parseDate(resp.Data.ExpireDate),
		Cvv:          resp.Data.Cvv,
		CardProvider: resp.Data.CardProvider,
	}, nil
}

type localCardAdapter struct {
	queryRepo   repository.CardQueryRepository
	commandRepo repository.CardCommandRepository
}

func NewLocalCardAdapter(queryRepo repository.CardQueryRepository, commandRepo repository.CardCommandRepository) CardAdapter {
	return &localCardAdapter{
		queryRepo:   queryRepo,
		commandRepo: commandRepo,
	}
}

func (a *localCardAdapter) FindCardByUserId(ctx context.Context, user_id int) (*db.GetCardByUserIDRow, error) {
	return a.queryRepo.FindCardByUserId(ctx, user_id)
}

func (a *localCardAdapter) FindUserCardByCardNumber(ctx context.Context, card_number string) (*db.GetUserEmailByCardNumberRow, error) {
	return a.queryRepo.FindUserCardByCardNumber(ctx, card_number)
}

func (a *localCardAdapter) FindCardByCardNumber(ctx context.Context, card_number string) (*db.GetCardByCardNumberRow, error) {
	return a.queryRepo.FindCardByCardNumber(ctx, card_number)
}

func (a *localCardAdapter) UpdateCard(ctx context.Context, request *requests.UpdateCardRequest) (*db.UpdateCardRow, error) {
	return a.commandRepo.UpdateCard(ctx, request)
}

func parseDate(ts string) pgtype.Date {
	if ts == "" {
		return pgtype.Date{Valid: false}
	}
	t, err := time.Parse(time.RFC3339, ts)
	if err != nil {
		return pgtype.Date{Valid: false}
	}
	return pgtype.Date{Time: t, Valid: true}
}
