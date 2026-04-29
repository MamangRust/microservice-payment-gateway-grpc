package seeder

import (
	"context"
	"fmt"

	db "github.com/MamangRust/microservice-payment-gateway-grpc/pkg/database/schema"
	"github.com/MamangRust/microservice-payment-gateway-grpc/pkg/logger"
	"go.uber.org/zap"
)

// roleSeeder is a struct that represents a seeder for the roles table.
type roleSeeder struct {
	db     *db.Queries
	ctx    context.Context
	logger logger.LoggerInterface
}

// NewRoleSeeder creates a new instance of the roleSeeder, which is
// responsible for populating the roles table with fake data.
//
// Args:
// db: a pointer to the database queries
// ctx: a context.Context object
// logger: a logger.LoggerInterface object
//
// Returns:
// a pointer to the roleSeeder struct
func NewRoleSeeder(db *db.Queries, ctx context.Context, logger logger.LoggerInterface) *roleSeeder {
	return &roleSeeder{
		db:     db,
		ctx:    ctx,
		logger: logger,
	}
}

// Seed populates the roles table with fake data.
//
// It creates a total of 10 roles, with names randomly selected from the
// following list:
//
// - Super Admin
// - Admin
// - Merchant Admin
// - Merchant Operator
// - Finance
// - Compliance
// - Auditor
// - Support
// - Viewer
// - User
//
// If any of the roles fail to be created, the function returns an error.
//
// Returns:
// an error if any of the roles fail to be created, otherwise nil
func (r *roleSeeder) Seed() error {
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

	totalRoles := len(randomRoles)

	for i, roleName := range randomRoles {
		_, err := r.db.CreateRole(r.ctx, roleName)
		if err != nil {
			r.logger.Error("failed to seed role", zap.Int("role", i+1), zap.String("roleName", roleName), zap.Error(err))
			return fmt.Errorf("failed to seed role %d (%s): %w", i+1, roleName, err)
		}
	}

	r.logger.Debug("role seeded successfully", zap.Int("totalRoles", totalRoles))
	return nil
}
