package tests

import (
	"context"
	"time"

	"github.com/MamangRust/microservice-payment-gateway-grpc/pkg/adapter"
	"github.com/MamangRust/microservice-payment-gateway-grpc/pkg/database/models"
	card_repo "github.com/MamangRust/microservice-payment-gateway-grpc/service/card/repository"
	merchant_repo "github.com/MamangRust/microservice-payment-gateway-grpc/service/merchant/repository"
	saldo_repo "github.com/MamangRust/microservice-payment-gateway-grpc/service/saldo/repository"
	user_repo "github.com/MamangRust/microservice-payment-gateway-grpc/service/user/repository"
	"github.com/MamangRust/microservice-payment-gateway-grpc/shared/domain/requests"
	"github.com/jackc/pgx/v5/pgtype"
)

// Local adapters wrap a service's own repositories so integration tests can
// exercise a service without dialing its gRPC dependencies. They live here
// (not in pkg/adapter) so the shared adapter package never has to import
// service/*/repository.

func pgTimePtr(ts pgtype.Timestamp) *time.Time {
	if !ts.Valid {
		return nil
	}
	t := ts.Time
	return &t
}

func pgDatePtr(d pgtype.Date) *time.Time {
	if !d.Valid {
		return nil
	}
	t := d.Time
	return &t
}

type localUserAdapter struct {
	repo user_repo.UserQueryRepository
}

func NewLocalUserAdapter(repo user_repo.UserQueryRepository) adapter.UserAdapter {
	return &localUserAdapter{repo: repo}
}

func (a *localUserAdapter) FindById(ctx context.Context, userID int) (*models.User, error) {
	u, err := a.repo.FindById(ctx, userID)
	if err != nil {
		return nil, err
	}
	if u == nil {
		return nil, nil
	}
	return &models.User{
		UserID:    u.UserID,
		Firstname: u.Firstname,
		Lastname:  u.Lastname,
		Email:     u.Email,
		Password:  u.Password,
	}, nil
}

type localCardAdapter struct {
	queryRepo   card_repo.CardQueryRepository
	commandRepo card_repo.CardCommandRepository
}

func NewLocalCardAdapter(queryRepo card_repo.CardQueryRepository, commandRepo card_repo.CardCommandRepository) adapter.CardAdapter {
	return &localCardAdapter{queryRepo: queryRepo, commandRepo: commandRepo}
}

func (a *localCardAdapter) FindCardByUserId(ctx context.Context, user_id int) (*models.Card, error) {
	c, err := a.queryRepo.FindCardByUserId(ctx, user_id)
	if err != nil {
		return nil, err
	}
	if c == nil {
		return nil, nil
	}
	return &models.Card{
		CardID:       c.CardID,
		UserID:       c.UserID,
		CardNumber:   c.CardNumber,
		CardType:     c.CardType,
		ExpireDate:   pgDatePtr(c.ExpireDate),
		Cvv:          c.Cvv,
		CardProvider: c.CardProvider,
		CreatedAt:    pgTimePtr(c.CreatedAt),
		UpdatedAt:    pgTimePtr(c.UpdatedAt),
	}, nil
}

func (a *localCardAdapter) FindUserCardByCardNumber(ctx context.Context, card_number string) (*models.Card, error) {
	c, err := a.queryRepo.FindUserCardByCardNumber(ctx, card_number)
	if err != nil {
		return nil, err
	}
	if c == nil {
		return nil, nil
	}
	return &models.Card{
		CardID:       c.CardID,
		UserID:       c.UserID,
		CardNumber:   c.CardNumber,
		CardType:     c.CardType,
		ExpireDate:   pgDatePtr(c.ExpireDate),
		Cvv:          c.Cvv,
		CardProvider: c.CardProvider,
		Email:        c.Email,
		CreatedAt:    pgTimePtr(c.CreatedAt),
		UpdatedAt:    pgTimePtr(c.UpdatedAt),
	}, nil
}

func (a *localCardAdapter) FindCardByCardNumber(ctx context.Context, card_number string) (*models.Card, error) {
	c, err := a.queryRepo.FindCardByCardNumber(ctx, card_number)
	if err != nil {
		return nil, err
	}
	if c == nil {
		return nil, nil
	}
	return &models.Card{
		CardID:       c.CardID,
		UserID:       c.UserID,
		CardNumber:   c.CardNumber,
		CardType:     c.CardType,
		ExpireDate:   pgDatePtr(c.ExpireDate),
		Cvv:          c.Cvv,
		CardProvider: c.CardProvider,
		CreatedAt:    pgTimePtr(c.CreatedAt),
		UpdatedAt:    pgTimePtr(c.UpdatedAt),
	}, nil
}

func (a *localCardAdapter) UpdateCard(ctx context.Context, request *requests.UpdateCardRequest) (*models.Card, error) {
	c, err := a.commandRepo.UpdateCard(ctx, request)
	if err != nil {
		return nil, err
	}
	if c == nil {
		return nil, nil
	}
	return &models.Card{
		CardID:       c.CardID,
		UserID:       c.UserID,
		CardNumber:   c.CardNumber,
		CardType:     c.CardType,
		ExpireDate:   pgDatePtr(c.ExpireDate),
		Cvv:          c.Cvv,
		CardProvider: c.CardProvider,
		CreatedAt:    pgTimePtr(c.CreatedAt),
		UpdatedAt:    pgTimePtr(c.UpdatedAt),
	}, nil
}

type localMerchantAdapter struct {
	repo merchant_repo.MerchantQueryRepository
}

func NewLocalMerchantAdapter(repo merchant_repo.MerchantQueryRepository) adapter.MerchantAdapter {
	return &localMerchantAdapter{repo: repo}
}

func (a *localMerchantAdapter) FindByApiKey(ctx context.Context, api_key string) (*models.Merchant, error) {
	m, err := a.repo.FindByApiKey(ctx, api_key)
	if err != nil {
		return nil, err
	}
	if m == nil {
		return nil, nil
	}
	return &models.Merchant{
		MerchantID: m.MerchantID,
		Name:       m.Name,
		ApiKey:     m.ApiKey,
		UserID:     m.UserID,
		Status:     m.Status,
		CreatedAt:  pgTimePtr(m.CreatedAt),
		UpdatedAt:  pgTimePtr(m.UpdatedAt),
	}, nil
}

func (a *localMerchantAdapter) FindByMerchantId(ctx context.Context, merchant_id int) (*models.Merchant, error) {
	m, err := a.repo.FindByMerchantId(ctx, merchant_id)
	if err != nil {
		return nil, err
	}
	if m == nil {
		return nil, nil
	}
	return &models.Merchant{
		MerchantID: m.MerchantID,
		Name:       m.Name,
		ApiKey:     m.ApiKey,
		UserID:     m.UserID,
		Status:     m.Status,
		CreatedAt:  pgTimePtr(m.CreatedAt),
		UpdatedAt:  pgTimePtr(m.UpdatedAt),
	}, nil
}

type localSaldoAdapter struct {
	repo saldo_repo.Repositories
}

func NewLocalSaldoAdapter(repo saldo_repo.Repositories) adapter.SaldoAdapter {
	return &localSaldoAdapter{repo: repo}
}

func (a *localSaldoAdapter) FindByCardNumber(ctx context.Context, card_number string) (*models.Saldo, error) {
	s, err := a.repo.FindByCardNumber(ctx, card_number)
	if err != nil {
		return nil, err
	}
	if s == nil {
		return nil, nil
	}
	var withdrawAmount *int64
	if s.WithdrawAmount != nil {
		v := *s.WithdrawAmount
		withdrawAmount = &v
	}
	return &models.Saldo{
		SaldoID:        s.SaldoID,
		CardNumber:     s.CardNumber,
		TotalBalance:   s.TotalBalance,
		WithdrawAmount: withdrawAmount,
		WithdrawTime:   pgTimePtr(s.WithdrawTime),
		CreatedAt:      pgTimePtr(s.CreatedAt),
		UpdatedAt:      pgTimePtr(s.UpdatedAt),
	}, nil
}

func (a *localSaldoAdapter) UpdateSaldoBalance(ctx context.Context, request *requests.UpdateSaldoBalance) (*models.SaldoMutationResult, error) {
	r, err := a.repo.UpdateSaldoBalance(ctx, request)
	if err != nil {
		return nil, err
	}
	if r == nil {
		return nil, nil
	}
	return &models.SaldoMutationResult{SaldoID: r.SaldoID, CardNumber: r.CardNumber, TotalBalance: r.TotalBalance}, nil
}

func (a *localSaldoAdapter) DebitSaldo(ctx context.Context, request *requests.DebitSaldoRequest) (*models.SaldoMutationResult, error) {
	r, err := a.repo.DebitSaldo(ctx, request)
	if err != nil {
		return nil, err
	}
	if r == nil {
		return nil, nil
	}
	return &models.SaldoMutationResult{SaldoID: r.SaldoID, CardNumber: r.CardNumber, TotalBalance: r.TotalBalance}, nil
}

func (a *localSaldoAdapter) CreditSaldo(ctx context.Context, request *requests.CreditSaldoRequest) (*models.SaldoMutationResult, error) {
	r, err := a.repo.CreditSaldo(ctx, request)
	if err != nil {
		return nil, err
	}
	if r == nil {
		return nil, nil
	}
	return &models.SaldoMutationResult{SaldoID: r.SaldoID, CardNumber: r.CardNumber, TotalBalance: r.TotalBalance}, nil
}

func (a *localSaldoAdapter) UpdateSaldoWithdraw(ctx context.Context, request *requests.UpdateSaldoWithdraw) (*models.SaldoMutationResult, error) {
	r, err := a.repo.UpdateSaldoWithdraw(ctx, request)
	if err != nil {
		return nil, err
	}
	if r == nil {
		return nil, nil
	}
	return &models.SaldoMutationResult{SaldoID: r.SaldoID, CardNumber: r.CardNumber, TotalBalance: r.TotalBalance}, nil
}
