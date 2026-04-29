package adapter

import (
	"context"

	pbmerchant "github.com/MamangRust/microservice-payment-gateway-grpc/pb/merchant"
	db "github.com/MamangRust/microservice-payment-gateway-grpc/pkg/database/schema"
	"github.com/MamangRust/microservice-payment-gateway-grpc/service/merchant/repository"
)

type MerchantAdapter interface {
	FindByApiKey(ctx context.Context, api_key string) (*db.GetMerchantByApiKeyRow, error)
}

type merchantGRPCAdapter struct {
	QueryClient pbmerchant.MerchantQueryServiceClient
}

func NewMerchantAdapter(queryClient pbmerchant.MerchantQueryServiceClient) MerchantAdapter {
	return &merchantGRPCAdapter{
		QueryClient: queryClient,
	}
}

func (a *merchantGRPCAdapter) FindByApiKey(ctx context.Context, api_key string) (*db.GetMerchantByApiKeyRow, error) {
	resp, err := a.QueryClient.FindByApiKey(ctx, &pbmerchant.FindByApiKeyRequest{
		ApiKey: api_key,
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
