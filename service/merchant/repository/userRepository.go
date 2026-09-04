package repository

import (
	"context"
	userdb "github.com/MamangRust/microservice-payment-gateway-grpc/service/user/database/schema"
	sharedErrors "github.com/MamangRust/microservice-payment-gateway-grpc/shared/errors"
	"time"

	pb "github.com/MamangRust/microservice-payment-gateway-grpc/pb/user"
	"github.com/jackc/pgx/v5/pgtype"
)

// userRepository is a struct that represents a repository for user operations using gRPC.
type userRepository struct {
	userQueryClient pb.UserQueryServiceClient
}

// NewUserRepository creates a new instance of userRepository.
func NewUserRepository(userQueryClient pb.UserQueryServiceClient) UserRepository {
	return &userRepository{
		userQueryClient: userQueryClient,
	}
}

func (r *userRepository) FindById(ctx context.Context, user_id int) (*userdb.GetUserByIDRow, error) {
	resp, err := r.userQueryClient.FindById(ctx, &pb.FindByIdUserRequest{
		Id: int32(user_id),
	})

	if err != nil {
		return nil, sharedErrors.ErrUserNotFound.WithInternal(err)
	}

	if resp == nil || resp.Data == nil {
		return nil, sharedErrors.ErrUserNotFound
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

	// Map back to db struct
	return &userdb.GetUserByIDRow{
		UserID:    resp.Data.Id,
		Firstname: resp.Data.Firstname,
		Lastname:  resp.Data.Lastname,
		Email:     resp.Data.Email,
		CreatedAt: parseTime(resp.Data.CreatedAt),
		UpdatedAt: parseTime(resp.Data.UpdatedAt),
	}, nil
}
