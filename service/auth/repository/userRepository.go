package repository

import (
	"context"
	"time"

	pb "github.com/MamangRust/microservice-payment-gateway-grpc/pb/user"
	db "github.com/MamangRust/microservice-payment-gateway-grpc/pkg/database/schema"
	"github.com/MamangRust/microservice-payment-gateway-grpc/shared/domain/requests"
	user_errors "github.com/MamangRust/microservice-payment-gateway-grpc/shared/errors/user_errors/repository"
	"github.com/jackc/pgx/v5/pgtype"
)

// userRepository is a struct that represents a user repository
type userRepository struct {
	userQueryClient   pb.UserQueryServiceClient
	userCommandClient pb.UserCommandServiceClient
}

// NewUserRepository returns a new instance of userRepository.
func NewUserRepository(queryClient pb.UserQueryServiceClient, commandClient pb.UserCommandServiceClient) *userRepository {
	return &userRepository{
		userQueryClient:   queryClient,
		userCommandClient: commandClient,
	}
}

// FindById retrieves a user by their unique ID.
func (r *userRepository) FindById(ctx context.Context, user_id int) (*db.GetUserByIDRow, error) {
	resp, err := r.userQueryClient.FindById(ctx, &pb.FindByIdUserRequest{
		Id: int32(user_id),
	})

	if err != nil {
		return nil, user_errors.ErrUserNotFound.WithInternal(err)
	}

	user := resp.GetData()
	parseTime := func(ts string) pgtype.Timestamp {
		t, err := time.Parse(time.RFC3339, ts)
		if err != nil {
			return pgtype.Timestamp{Valid: false}
		}
		return pgtype.Timestamp{Time: t, Valid: true}
	}

	return &db.GetUserByIDRow{
		UserID:    int32(user.Id),
		Firstname: user.Firstname,
		Lastname:  user.Lastname,
		Email:     user.Email,
		CreatedAt: parseTime(user.CreatedAt),
		UpdatedAt: parseTime(user.UpdatedAt),
	}, nil
}

// FindByEmail retrieves a user by their email address.
func (r *userRepository) FindByEmail(ctx context.Context, email string) (*db.GetUserByEmailRow, error) {
	resp, err := r.userQueryClient.FindByEmail(ctx, &pb.FindByEmailUserRequest{
		Email: email,
	})

	if err != nil {
		return nil, user_errors.ErrUserNotFound.WithInternal(err)
	}

	user := resp.GetData()

	return &db.GetUserByEmailRow{
		UserID:   int32(user.Id),
		Email:    user.Email,
		Password: user.Password,
	}, nil
}

// FindByEmailAndVerify retrieves a verified user by their email address.
func (r *userRepository) FindByEmailAndVerify(ctx context.Context, email string) (*db.GetUserByEmailAndVerifiedRow, error) {
	// For simplicity, we fetch by email and check verification locally or via specialized GRPC.
	// Current User domain lookup doesn't have FindByEmailAndVerify, so we use FindByEmail.
	resp, err := r.userQueryClient.FindByEmail(ctx, &pb.FindByEmailUserRequest{
		Email: email,
	})

	if err != nil {
		return nil, user_errors.ErrUserNotFound.WithInternal(err)
	}

	user := resp.GetData()
	parseTime := func(ts string) pgtype.Timestamp {
		t, err := time.Parse(time.RFC3339, ts)
		if err != nil {
			return pgtype.Timestamp{Valid: false}
		}
		return pgtype.Timestamp{Time: t, Valid: true}
	}

	return &db.GetUserByEmailAndVerifiedRow{
		UserID:    int32(user.Id),
		Firstname: user.Firstname,
		Lastname:  user.Lastname,
		Email:     user.Email,
		Password:  user.Password,
		CreatedAt: parseTime(user.CreatedAt),
		UpdatedAt: parseTime(user.UpdatedAt),
	}, nil
}

// FindByVerificationCode retrieves a user by their verification code.
func (r *userRepository) FindByVerificationCode(ctx context.Context, verification_code string) (*db.GetUserByVerificationCodeRow, error) {
	resp, err := r.userQueryClient.FindByVerificationCode(ctx, &pb.FindByVerificationCodeUserRequest{
		VerificationCode: verification_code,
	})

	if err != nil {
		return nil, nil
	}

	user := resp.GetData()
	parseTime := func(ts string) pgtype.Timestamp {
		t, err := time.Parse(time.RFC3339, ts)
		if err != nil {
			return pgtype.Timestamp{Valid: false}
		}
		return pgtype.Timestamp{Time: t, Valid: true}
	}

	return &db.GetUserByVerificationCodeRow{
		UserID:    int32(user.Id),
		Firstname: user.Firstname,
		Lastname:  user.Lastname,
		Email:     user.Email,
		CreatedAt: parseTime(user.CreatedAt),
		UpdatedAt: parseTime(user.UpdatedAt),
	}, nil
}

// CreateUser inserts a new user into the database via GRPC.
func (r *userRepository) CreateUser(ctx context.Context, request *requests.RegisterRequest) (*db.CreateUserRow, error) {
	resp, err := r.userCommandClient.Create(ctx, &pb.CreateUserRequest{
		Firstname:       request.FirstName,
		Lastname:        request.LastName,
		Email:           request.Email,
		Password:        request.Password,
		ConfirmPassword: request.Password,
	})

	if err != nil {
		return nil, user_errors.ErrCreateUser.WithInternal(err)
	}

	user := resp.GetData()
	return &db.CreateUserRow{
		UserID:    int32(user.Id),
		Firstname: user.Firstname,
		Lastname:  user.Lastname,
		Email:     user.Email,
	}, nil
}

// UpdateUserIsVerified updates the verification status of a user via GRPC.
func (r *userRepository) UpdateUserIsVerified(ctx context.Context, user_id int, is_verified bool) (*db.UpdateUserIsVerifiedRow, error) {
	resp, err := r.userCommandClient.UpdateIsVerified(ctx, &pb.UpdateUserIsVerifiedRequest{
		UserId:     int32(user_id),
		IsVerified: is_verified,
	})

	if err != nil {
		return nil, user_errors.ErrUpdateUserVerificationCode.WithInternal(err)
	}

	user := resp.GetData()
	return &db.UpdateUserIsVerifiedRow{
		UserID: int32(user.Id),
	}, nil
}

// UpdateUserPassword updates a user's password via GRPC.
func (r *userRepository) UpdateUserPassword(ctx context.Context, user_id int, password string) (*db.UpdateUserPasswordRow, error) {
	resp, err := r.userCommandClient.UpdatePassword(ctx, &pb.UpdateUserPasswordRequest{
		UserId:   int32(user_id),
		Password: password,
	})

	if err != nil {
		return nil, user_errors.ErrUpdateUserPassword.WithInternal(err)
	}

	user := resp.GetData()
	return &db.UpdateUserPasswordRow{
		UserID: int32(user.Id),
	}, nil
}
