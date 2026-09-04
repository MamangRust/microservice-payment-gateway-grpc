package adapter

import (
	"context"
	"time"

	pbcard "github.com/MamangRust/microservice-payment-gateway-grpc/pb/card"
	pbsaldo "github.com/MamangRust/microservice-payment-gateway-grpc/pb/saldo"
	db "github.com/MamangRust/microservice-payment-gateway-grpc/service/saldo/database/schema"
	"github.com/MamangRust/microservice-payment-gateway-grpc/pkg/resilience"
	"github.com/MamangRust/microservice-payment-gateway-grpc/service/saldo/repository"
	"github.com/MamangRust/microservice-payment-gateway-grpc/shared/domain/requests"
	sharedErrors "github.com/MamangRust/microservice-payment-gateway-grpc/shared/errors"
	"github.com/jackc/pgx/v5/pgtype"
)

type SaldoAdapter interface {
	FindByCardNumber(ctx context.Context, card_number string) (*db.Saldo, error)
	UpdateSaldoBalance(ctx context.Context, request *requests.UpdateSaldoBalance) (*db.UpdateSaldoBalanceRow, error)
	DebitSaldo(ctx context.Context, request *requests.DebitSaldoRequest) (*db.DebitSaldoRow, error)
	CreditSaldo(ctx context.Context, request *requests.CreditSaldoRequest) (*db.CreditSaldoRow, error)
	UpdateSaldoWithdraw(ctx context.Context, request *requests.UpdateSaldoWithdraw) (*db.UpdateSaldoWithdrawRow, error)
}

type saldoGRPCAdapter struct {
	QueryClient   pbsaldo.SaldoQueryServiceClient
	CommandClient pbsaldo.SaldoCommandServiceClient
	guard         *resilience.DependencyGuard
}

func (a *saldoGRPCAdapter) setGuard(g *resilience.DependencyGuard) {
	a.guard = g
}

func NewSaldoAdapter(queryClient pbsaldo.SaldoQueryServiceClient, commandClient pbsaldo.SaldoCommandServiceClient, opts ...func(guardSetter)) SaldoAdapter {
	a := &saldoGRPCAdapter{
		QueryClient:   queryClient,
		CommandClient: commandClient,
	}
	for _, opt := range opts {
		opt(a)
	}
	return a
}

func (a *saldoGRPCAdapter) FindByCardNumber(ctx context.Context, card_number string) (*db.Saldo, error) {
	var resp *pbsaldo.ApiResponseSaldo
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

	return MapSaldoResponseToDB(resp.Data)
}

func (a *saldoGRPCAdapter) UpdateSaldoBalance(ctx context.Context, request *requests.UpdateSaldoBalance) (*db.UpdateSaldoBalanceRow, error) {
	var resp *pbsaldo.ApiResponseSaldo
	err := a.guard.Call(ctx, func(callCtx context.Context) error {
		var callErr error
		resp, callErr = a.CommandClient.UpdateSaldo(callCtx, &pbsaldo.UpdateSaldoRequest{
			CardNumber:   request.CardNumber,
			TotalBalance: int64(request.TotalBalance),
		})
		return callErr
	})
	if err != nil {
		return nil, err
	}

	return &db.UpdateSaldoBalanceRow{
		SaldoID:      resp.Data.SaldoId,
		CardNumber:   resp.Data.CardNumber,
		TotalBalance: resp.Data.TotalBalance,
	}, nil
}

func (a *saldoGRPCAdapter) DebitSaldo(ctx context.Context, request *requests.DebitSaldoRequest) (*db.DebitSaldoRow, error) {
	var resp *pbsaldo.ApiResponseSaldo
	err := a.guard.Call(ctx, func(callCtx context.Context) error {
		var callErr error
		resp, callErr = a.CommandClient.DebitSaldo(callCtx, &pbsaldo.DebitSaldoRequest{
			CardNumber:  request.CardNumber,
			Amount:      int64(request.Amount),
			OperationId: request.OperationID,
			SourceType:  request.SourceType,
			SourceId:    request.SourceID,
		})
		return callErr
	})
	if err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, sharedErrors.NewBadRequestError("saldo response is required")
	}
	return mapDebitSaldoResponse(resp.Data)
}

func (a *saldoGRPCAdapter) CreditSaldo(ctx context.Context, request *requests.CreditSaldoRequest) (*db.CreditSaldoRow, error) {
	var resp *pbsaldo.ApiResponseSaldo
	err := a.guard.Call(ctx, func(callCtx context.Context) error {
		var callErr error
		resp, callErr = a.CommandClient.CreditSaldo(callCtx, &pbsaldo.CreditSaldoRequest{
			CardNumber:  request.CardNumber,
			Amount:      int64(request.Amount),
			OperationId: request.OperationID,
			SourceType:  request.SourceType,
			SourceId:    request.SourceID,
		})
		return callErr
	})
	if err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, sharedErrors.NewBadRequestError("saldo response is required")
	}
	return mapCreditSaldoResponse(resp.Data)
}

func mapDebitSaldoResponse(response *pbsaldo.SaldoResponse) (*db.DebitSaldoRow, error) {
	if response == nil {
		return nil, sharedErrors.NewBadRequestError("saldo response is required")
	}
	return &db.DebitSaldoRow{SaldoID: response.SaldoId, CardNumber: response.CardNumber, TotalBalance: response.TotalBalance}, nil
}

func mapCreditSaldoResponse(response *pbsaldo.SaldoResponse) (*db.CreditSaldoRow, error) {
	if response == nil {
		return nil, sharedErrors.NewBadRequestError("saldo response is required")
	}
	return &db.CreditSaldoRow{SaldoID: response.SaldoId, CardNumber: response.CardNumber, TotalBalance: response.TotalBalance}, nil
}

func (a *saldoGRPCAdapter) UpdateSaldoWithdraw(ctx context.Context, request *requests.UpdateSaldoWithdraw) (*db.UpdateSaldoWithdrawRow, error) {
	var resp *pbsaldo.ApiResponseSaldo
	err := a.guard.Call(ctx, func(callCtx context.Context) error {
		var callErr error
		resp, callErr = a.CommandClient.UpdateSaldoWithdraw(callCtx, &pbsaldo.UpdateSaldoWithdrawRequest{
			CardNumber:     request.CardNumber,
			TotalBalance:   int64(request.TotalBalance),
			WithdrawAmount: int64(*request.WithdrawAmount),
			WithdrawTime:   request.WithdrawTime.Format(time.RFC3339),
		})
		return callErr
	})
	if err != nil {
		return nil, err
	}

	return &db.UpdateSaldoWithdrawRow{
		SaldoID:      resp.Data.SaldoId,
		CardNumber:   resp.Data.CardNumber,
		TotalBalance: resp.Data.TotalBalance,
	}, nil
}

type localSaldoAdapter struct {
	repo repository.Repositories
}

func NewLocalSaldoAdapter(repo repository.Repositories) SaldoAdapter {
	return &localSaldoAdapter{
		repo: repo,
	}
}

func (a *localSaldoAdapter) FindByCardNumber(ctx context.Context, card_number string) (*db.Saldo, error) {
	return a.repo.FindByCardNumber(ctx, card_number)
}

func (a *localSaldoAdapter) UpdateSaldoBalance(ctx context.Context, request *requests.UpdateSaldoBalance) (*db.UpdateSaldoBalanceRow, error) {
	return a.repo.UpdateSaldoBalance(ctx, request)
}

func (a *localSaldoAdapter) DebitSaldo(ctx context.Context, request *requests.DebitSaldoRequest) (*db.DebitSaldoRow, error) {
	return a.repo.DebitSaldo(ctx, request)
}

func (a *localSaldoAdapter) CreditSaldo(ctx context.Context, request *requests.CreditSaldoRequest) (*db.CreditSaldoRow, error) {
	return a.repo.CreditSaldo(ctx, request)
}

func (a *localSaldoAdapter) UpdateSaldoWithdraw(ctx context.Context, request *requests.UpdateSaldoWithdraw) (*db.UpdateSaldoWithdrawRow, error) {
	return a.repo.UpdateSaldoWithdraw(ctx, request)
}

func MapSaldoResponseToDB(s *pbsaldo.SaldoResponse) (*db.Saldo, error) {
	if s == nil {
		return nil, sharedErrors.NewBadRequestError("saldo response is required")
	}

	saldo := &db.Saldo{
		SaldoID:      s.SaldoId,
		CardNumber:   s.CardNumber,
		TotalBalance: s.TotalBalance,
	}

	if s.WithdrawAmount != 0 {
		saldo.WithdrawAmount = &s.WithdrawAmount
	}

	parseTime := func(ts string) pgtype.Timestamp {
		if ts == "" {
			return pgtype.Timestamp{Valid: false}
		}
		t, err := time.Parse(time.RFC3339, ts)
		if err != nil {
			return pgtype.Timestamp{Valid: false}
		}
		return pgtype.Timestamp{Time: t, Valid: true}
	}

	saldo.WithdrawTime = parseTime(s.WithdrawTime)
	saldo.CreatedAt = parseTime(s.CreatedAt)
	saldo.UpdatedAt = parseTime(s.UpdatedAt)

	return saldo, nil
}

