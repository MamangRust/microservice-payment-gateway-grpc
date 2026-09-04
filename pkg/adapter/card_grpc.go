package adapter

import (
	"context"
	"fmt"
	"time"

	pbcard "github.com/MamangRust/microservice-payment-gateway-grpc/pb/card"
	db "github.com/MamangRust/microservice-payment-gateway-grpc/service/card/database/schema"
	"github.com/MamangRust/microservice-payment-gateway-grpc/pkg/resilience"
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
	guard         *resilience.DependencyGuard
}

func (a *cardGRPCAdapter) setGuard(g *resilience.DependencyGuard) {
	a.guard = g
}

func NewCardAdapter(queryClient pbcard.CardQueryServiceClient, commandClient pbcard.CardCommandServiceClient, opts ...func(guardSetter)) CardAdapter {
	a := &cardGRPCAdapter{
		QueryClient:   queryClient,
		CommandClient: commandClient,
	}
	for _, opt := range opts {
		opt(a)
	}
	return a
}

func (a *cardGRPCAdapter) FindCardByUserId(ctx context.Context, user_id int) (*db.GetCardByUserIDRow, error) {
	var resp *pbcard.ApiResponseCard
	err := a.guard.Call(ctx, func(callCtx context.Context) error {
		var callErr error
		resp, callErr = a.QueryClient.FindByUserIdCard(callCtx, &pbcard.FindByUserIdCardRequest{
			UserId: int32(user_id),
		})
		return callErr
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
	var resp *pbcard.CardWithEmailResponse
	err := a.guard.Call(ctx, func(callCtx context.Context) error {
		var callErr error
		resp, callErr = a.QueryClient.FindUserCardByCardNumber(callCtx, &pbcard.FindByCardNumberRequest{
			CardNumber: card_number,
		})
		return callErr
	})
	if err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, fmt.Errorf("card lookup returned an empty response for card_number=%q", card_number)
	}

	// Map the full card identity so flows using this lookup (transaction,
	// transfer) get the owner metadata they rely on, not just the email.
	return &db.GetUserEmailByCardNumberRow{
		CardID:       resp.Id,
		UserID:       resp.UserId,
		CardNumber:   resp.CardNumber,
		CardType:     resp.CardType,
		ExpireDate:   parseDate(resp.ExpireDate),
		Cvv:          resp.Cvv,
		CardProvider: resp.CardProvider,
		Email:        resp.Email,
		CreatedAt:    parseTimestamp(resp.CreatedAt),
		UpdatedAt:    parseTimestamp(resp.UpdatedAt),
	}, nil
}

func (a *cardGRPCAdapter) FindCardByCardNumber(ctx context.Context, card_number string) (*db.GetCardByCardNumberRow, error) {
	var resp *pbcard.ApiResponseCard
	err := a.guard.Call(ctx, func(callCtx context.Context) error {
		var callErr error
		resp, callErr = a.QueryClient.FindByCardNumber(callCtx, &pbcard.FindByCardNumberRequest{
			CardNumber: card_number,
		})
		return callErr
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
	var resp *pbcard.ApiResponseCard
	err := a.guard.Call(ctx, func(callCtx context.Context) error {
		var callErr error
		resp, callErr = a.CommandClient.UpdateCard(callCtx, &pbcard.UpdateCardRequest{
			CardId:       int32(request.CardID),
			UserId:       int32(request.UserID),
			CardType:     request.CardType,
			ExpireDate:   timestamppb.New(request.ExpireDate),
			Cvv:          request.CVV,
			CardProvider: request.CardProvider,
		})
		return callErr
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

func parseTimestamp(ts string) pgtype.Timestamp {
	if ts == "" {
		return pgtype.Timestamp{Valid: false}
	}
	t, err := time.Parse(time.RFC3339, ts)
	if err != nil {
		return pgtype.Timestamp{Valid: false}
	}
	return pgtype.Timestamp{Time: t, Valid: true}
}
