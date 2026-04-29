package repository

import (
	"context"

	db "github.com/MamangRust/microservice-payment-gateway-grpc/pkg/database/schema"
	"github.com/MamangRust/microservice-payment-gateway-grpc/shared/domain/requests"
	user_errors "github.com/MamangRust/microservice-payment-gateway-grpc/shared/errors/user_errors/repository"
	"github.com/google/uuid"
)

// userCommandRepository is a struct that implements the UserCommandRepository interface
type userCommandRepository struct {
	db *db.Queries
}

// NewUserCommandRepository creates a new instance of userCommandRepository.
func NewUserCommandRepository(db *db.Queries) UserCommandRepository {
	return &userCommandRepository{
		db: db,
	}
}

// CreateUser inserts a new user record into the database.
func (r *userCommandRepository) CreateUser(ctx context.Context, request *requests.CreateUserRequest) (*db.CreateUserRow, error) {
	verified := false
	verifyCode := uuid.New().String()

	req := db.CreateUserParams{
		Firstname:        request.FirstName,
		Lastname:         request.LastName,
		Email:            request.Email,
		Password:         request.Password,
		VerificationCode: verifyCode,
		IsVerified:       &verified,
	}

	user, err := r.db.CreateUser(ctx, req)

	if err != nil {
		return nil, user_errors.ErrCreateUser.WithInternal(err)
	}

	return user, nil
}

// UpdateUser updates an existing user record in the database.
func (r *userCommandRepository) UpdateUser(ctx context.Context, request *requests.UpdateUserRequest) (*db.UpdateUserRow, error) {
	req := db.UpdateUserParams{
		UserID:    int32(*request.UserID),
		Firstname: request.FirstName,
		Lastname:  request.LastName,
		Email:     request.Email,
		Password:  request.Password,
	}

	res, err := r.db.UpdateUser(ctx, req)

	if err != nil {
		return nil, user_errors.ErrUpdateUser.WithInternal(err)
	}

	return res, nil
}

// TrashedUser soft-deletes a user by marking it as trashed.
func (r *userCommandRepository) TrashedUser(ctx context.Context, user_id int) (*db.TrashUserRow, error) {
	res, err := r.db.TrashUser(ctx, int32(user_id))

	if err != nil {
		return nil, user_errors.ErrTrashedUser.WithInternal(err)
	}

	return res, nil
}

// RestoreUser restores a soft-deleted (trashed) user.
func (r *userCommandRepository) RestoreUser(ctx context.Context, user_id int) (*db.RestoreUserRow, error) {
	res, err := r.db.RestoreUser(ctx, int32(user_id))

	if err != nil {
		return nil, user_errors.ErrRestoreUser.WithInternal(err)
	}

	return res, nil
}

// DeleteUserPermanent permanently deletes a user from the database.
func (r *userCommandRepository) DeleteUserPermanent(ctx context.Context, user_id int) (bool, error) {
	err := r.db.DeleteUserPermanently(ctx, int32(user_id))

	if err != nil {
		return false, user_errors.ErrDeleteUserPermanent.WithInternal(err)
	}

	return true, nil
}

// RestoreAllUser restores all soft-deleted users in the database.
func (r *userCommandRepository) RestoreAllUser(ctx context.Context) (bool, error) {
	err := r.db.RestoreAllUsers(ctx)

	if err != nil {
		return false, user_errors.ErrRestoreAllUsers.WithInternal(err)
	}

	return true, nil
}

// DeleteAllUserPermanent permanently deletes all trashed users from the database.
func (r *userCommandRepository) DeleteAllUserPermanent(ctx context.Context) (bool, error) {
	err := r.db.DeleteAllPermanentUsers(ctx)

	if err != nil {
		return false, user_errors.ErrDeleteAllUsers.WithInternal(err)
	}
	return true, nil
}

func (r *userCommandRepository) UpdateIsVerified(ctx context.Context, userID int, isVerified bool) (*db.UpdateUserIsVerifiedRow, error) {
	res, err := r.db.UpdateUserIsVerified(ctx, db.UpdateUserIsVerifiedParams{
		UserID:     int32(userID),
		IsVerified: &isVerified,
	})

	if err != nil {
		return nil, user_errors.ErrUpdateUser.WithInternal(err)
	}

	return res, nil
}

func (r *userCommandRepository) UpdatePassword(ctx context.Context, userID int, password string) (*db.UpdateUserPasswordRow, error) {
	res, err := r.db.UpdateUserPassword(ctx, db.UpdateUserPasswordParams{
		UserID:   int32(userID),
		Password: password,
	})

	if err != nil {
		return nil, user_errors.ErrUpdateUser.WithInternal(err)
	}

	return res, nil
}
