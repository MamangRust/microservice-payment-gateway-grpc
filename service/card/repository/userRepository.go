package repository

import (
	"context"
	"time"

	pb "github.com/MamangRust/microservice-payment-gateway-grpc/pb/user"
	db "github.com/MamangRust/microservice-payment-gateway-grpc/pkg/database/schema"
	user_errors "github.com/MamangRust/microservice-payment-gateway-grpc/shared/errors/user_errors/repository"
	"github.com/jackc/pgx/v5/pgtype"
)

type userRepository struct {
	userQueryClient pb.UserQueryServiceClient
}

func NewUserRepository(userQueryClient pb.UserQueryServiceClient) UserRepository {
	return &userRepository{
		userQueryClient: userQueryClient,
	}
}

func (r *userRepository) FindById(ctx context.Context, user_id int) (*db.GetUserByIDRow, error) {
	resp, err := r.userQueryClient.FindById(ctx, &pb.FindByIdUserRequest{
		Id: int32(user_id),
	})

	if err != nil {
		return nil, user_errors.ErrUserNotFound.WithInternal(err)
	}

	if resp == nil || resp.Data == nil {
		return nil, user_errors.ErrUserNotFound
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

	return &db.GetUserByIDRow{
		UserID:    resp.Data.Id,
		Firstname: resp.Data.Firstname,
		Lastname:  resp.Data.Lastname,
		Email:     resp.Data.Email,
		CreatedAt: parseTime(resp.Data.CreatedAt),
		UpdatedAt: parseTime(resp.Data.UpdatedAt),
	}, nil
}
