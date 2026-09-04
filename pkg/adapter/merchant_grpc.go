package adapter

import (
	"context"

	pbmerchant "github.com/MamangRust/microservice-payment-gateway-grpc/pb/merchant"
	db "github.com/MamangRust/microservice-payment-gateway-grpc/service/merchant/database/schema"
	"github.com/MamangRust/microservice-payment-gateway-grpc/pkg/resilience"
	"github.com/MamangRust/microservice-payment-gateway-grpc/service/merchant/repository"
)

type MerchantAdapter interface {
	FindByApiKey(ctx context.Context, api_key string) (*db.GetMerchantByApiKeyRow, error)
	FindByMerchantId(ctx context.Context, merchant_id int) (*db.GetMerchantByIDRow, error)
}

type merchantGRPCAdapter struct {
	QueryClient pbmerchant.MerchantQueryServiceClient
	guard       *resilience.DependencyGuard
}

func (a *merchantGRPCAdapter) setGuard(g *resilience.DependencyGuard) {
	a.guard = g
}

func NewMerchantAdapter(queryClient pbmerchant.MerchantQueryServiceClient, opts ...func(guardSetter)) MerchantAdapter {
	a := &merchantGRPCAdapter{
		QueryClient: queryClient,
	}
	for _, opt := range opts {
		opt(a)
	}
	return a
}

func (a *merchantGRPCAdapter) FindByApiKey(ctx context.Context, api_key string) (*db.GetMerchantByApiKeyRow, error) {
	var resp *pbmerchant.ApiResponseMerchant
	err := a.guard.Call(ctx, func(callCtx context.Context) error {
		var callErr error
		resp, callErr = a.QueryClient.FindByApiKey(callCtx, &pbmerchant.FindByApiKeyRequest{
			ApiKey: api_key,
		})
		return callErr
	})
	if err != nil {
		return nil, err
	}

	return &db.GetMerchantByApiKeyRow{
		MerchantID: resp.Data.Id,
		Name:       resp.Data.Name,
		ApiKey:     resp.Data.ApiKey,
		UserID:     resp.Data.UserId,
		Status:     resp.Data.Status,
	}, nil
}

func (a *merchantGRPCAdapter) FindByMerchantId(ctx context.Context, merchant_id int) (*db.GetMerchantByIDRow, error) {
	var resp *pbmerchant.ApiResponseMerchant
	err := a.guard.Call(ctx, func(callCtx context.Context) error {
		var callErr error
		resp, callErr = a.QueryClient.FindByIdMerchant(callCtx, &pbmerchant.FindByIdMerchantRequest{
			MerchantId: int32(merchant_id),
		})
		return callErr
	})
	if err != nil {
		return nil, err
	}

	return &db.GetMerchantByIDRow{
		MerchantID: resp.Data.Id,
		Name:       resp.Data.Name,
		ApiKey:     resp.Data.ApiKey,
		Status:     resp.Data.Status,
		UserID:     resp.Data.UserId,
	}, nil
}

type localMerchantAdapter struct {
	repo repository.MerchantQueryRepository
}

func NewLocalMerchantAdapter(repo repository.MerchantQueryRepository) MerchantAdapter {
	return &localMerchantAdapter{
		repo: repo,
	}
}

func (a *localMerchantAdapter) FindByApiKey(ctx context.Context, api_key string) (*db.GetMerchantByApiKeyRow, error) {
	return a.repo.FindByApiKey(ctx, api_key)
}

func (a *localMerchantAdapter) FindByMerchantId(ctx context.Context, merchant_id int) (*db.GetMerchantByIDRow, error) {
	return a.repo.FindByMerchantId(ctx, merchant_id)
}
