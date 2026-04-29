package adapter

import (
	"context"

	"github.com/MamangRust/microservice-payment-gateway-grpc/pb/user"
	db "github.com/MamangRust/microservice-payment-gateway-grpc/pkg/database/schema"
	"github.com/MamangRust/microservice-payment-gateway-grpc/service/user/repository"
)

type UserAdapter interface {
	FindById(ctx context.Context, userID int) (*db.User, error)
}

type userGRPCAdapter struct {
	queryClient user.UserQueryServiceClient
}

func NewUserAdapter(queryClient user.UserQueryServiceClient) UserAdapter {
	return &userGRPCAdapter{
		queryClient: queryClient,
	}
}

func (a *userGRPCAdapter) FindById(ctx context.Context, userID int) (*db.User, error) {
	resp, err := a.queryClient.FindById(ctx, &user.FindByIdUserRequest{
		Id: int32(userID),
	})
	if err != nil {
		return nil, err
	}

	return &db.User{
		UserID:    resp.Data.Id,
		Email:     resp.Data.Email,
		Firstname: resp.Data.Firstname,
		Lastname:  resp.Data.Lastname,
	}, nil
}

type localUserAdapter struct {
	repo repository.UserQueryRepository
}

func NewLocalUserAdapter(repo repository.UserQueryRepository) UserAdapter {
	return &localUserAdapter{
		repo: repo,
	}
}

func (a *localUserAdapter) FindById(ctx context.Context, userID int) (*db.User, error) {
	res, err := a.repo.FindById(ctx, userID)
	if err != nil {
		return nil, err
	}

	return &db.User{
		UserID:    res.UserID,
		Email:     res.Email,
		Firstname: res.Firstname,
		Lastname:  res.Lastname,
	}, nil
}
