package seeder

import (
	"context"
	"fmt"
	"math/rand"

	db "github.com/MamangRust/microservice-payment-gateway-grpc/pkg/database/schema"
	"github.com/MamangRust/microservice-payment-gateway-grpc/pkg/hash"
	"github.com/MamangRust/microservice-payment-gateway-grpc/pkg/logger"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// userSeeder is a struct that represents a seeder for the users table.
type userSeeder struct {
	db     *db.Queries
	hash   hash.HashPassword
	ctx    context.Context
	logger logger.LoggerInterface
}

// NewUserSeeder creates a new instance of the userSeeder, which is
// responsible for populating the users table with fake data.
//
// Args:
// db: a pointer to the database queries
// ctx: a context.Context object
// hash: a hash.HashPassword object
// logger: a logger.LoggerInterface object
//
// Returns:
// a pointer to the userSeeder struct
func NewUserSeeder(db *db.Queries, ctx context.Context, hash hash.HashPassword, logger logger.LoggerInterface) *userSeeder {
	return &userSeeder{
		db:     db,
		hash:   hash,
		ctx:    ctx,
		logger: logger,
	}
}

// Seed populates the users table with fake data.
//
// It creates a total of 15 users, with 10 of them being active and 5 of them being
// trashed. The active users have a status of "active", while the trashed users have a
// status of "deactive". The user names are generated randomly from the combination of
// the adjectives and nouns provided. The API key is generated using the GenerateApiKey
// function from the api-key package.
//
// If any errors occur during the seeding process, an error is returned.
//
// Returns:
// an error if any of the users fail to be created, otherwise nil
func (r *userSeeder) Seed() error {
	totalUsers := 100
	activeUsers := 80
	trashedUsers := totalUsers - activeUsers

	randomRoles := []string{
		"Super Admin",
		"Admin",
		"Merchant Admin",
		"Merchant Operator",
		"Finance",
		"Compliance",
		"Auditor",
		"Support",
		"Viewer",
		"User",
	}

	for i := 1; i <= totalUsers; i++ {
		email := fmt.Sprintf("user_%s@example.com", uuid.NewString())

		hash, err := r.hash.HashPassword("password")
		if err != nil {
			r.logger.Error("failed to hash password", zap.Int("user", i), zap.Error(err))
			return fmt.Errorf("failed to hash password for user %d: %w", i, err)
		}

		isVerified := true

		user := db.CreateUserParams{
			Firstname:        fmt.Sprintf("User%d", i),
			Lastname:         fmt.Sprintf("Last%d", i),
			Email:            email,
			Password:         hash,
			VerificationCode: uuid.NewString(),
			IsVerified:       &isVerified,
		}

		createdUser, err := r.db.CreateUser(r.ctx, user)
		if err != nil {
			r.logger.Error("failed to create user", zap.Int("user", i), zap.Error(err))
			return fmt.Errorf("failed to create user %d: %w", i, err)
		}

		randomRole := randomRoles[rand.Intn(len(randomRoles))]
		role, err := r.db.GetRoleByName(r.ctx, randomRole)
		if err != nil {
			r.logger.Error("failed to get role", zap.String("role", randomRole), zap.Error(err))
			return fmt.Errorf("failed to get role %s: %w", randomRole, err)
		}

		_, err = r.db.AssignRoleToUser(r.ctx, db.AssignRoleToUserParams{
			RoleID: role.RoleID,
			UserID: createdUser.UserID,
		})
		if err != nil {
			r.logger.Error("failed to assign role to user", zap.Int("userID", int(createdUser.UserID)), zap.String("role", randomRole), zap.Error(err))
			return fmt.Errorf("failed to assign role %s to user %d: %w", randomRole, createdUser.UserID, err)
		}

		if i > activeUsers {
			_, err := r.db.TrashUser(r.ctx, createdUser.UserID)
			if err != nil {
				r.logger.Error("failed to trash user", zap.Int("user", i), zap.Error(err))
				return fmt.Errorf("failed to trash user %d: %w", i, err)
			}
		}
	}

	r.logger.Info("user seeded successfully", zap.Int("totalUsers", totalUsers), zap.Int("activeUsers", activeUsers), zap.Int("trashedUsers", trashedUsers))

	return nil
}
