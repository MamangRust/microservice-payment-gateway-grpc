package seeder

import (
	"context"
	"fmt"

	apikey "github.com/MamangRust/microservice-payment-gateway-grpc/pkg/api-key"
	db "github.com/MamangRust/microservice-payment-gateway-grpc/pkg/database/schema"
	"github.com/MamangRust/microservice-payment-gateway-grpc/pkg/logger"
	"go.uber.org/zap"
)

// merchantSeeder is a struct that represents a seeder for the merchants table.
type merchantSeeder struct {
	db     *db.Queries
	ctx    context.Context
	logger logger.LoggerInterface
}

// NewMerchantSeeder creates a new instance of the merchantSeeder, which is
// responsible for populating the merchants table with fake data.
//
// Args:
// db: a pointer to the database queries
// ctx: a context.Context object
// logger: a logger.LoggerInterface object
//
// Returns:
// a pointer to the merchantSeeder struct
func NewMerchantSeeder(db *db.Queries, ctx context.Context, logger logger.LoggerInterface) *merchantSeeder {
	return &merchantSeeder{
		db:     db,
		ctx:    ctx,
		logger: logger,
	}
}

// Seed populates the merchants table with fake data.
//
// It creates a total of 10 merchants, with 5 of them being active and 5 of them being
// trashed. The active merchants have a status of "active", while the trashed merchants
// have a status of "deactive". The merchant names are generated randomly from the
// combination of the adjectives and nouns provided. The API key is generated using
// the GenerateApiKey function from the api-key package.
func (r *merchantSeeder) Seed() error {
	adjectives := []string{"Blue", "Green", "Red", "Yellow", "Fast"}
	nouns := []string{"Shop", "Store", "Mart", "Market", "Hub"}

	totalMerchants := 40
	activeMerchants := 35
	trashedMerchants := totalMerchants - activeMerchants

	for i := 0; i < totalMerchants; i++ {
		adjective := adjectives[i%len(adjectives)]
		noun := nouns[i%len(nouns)]
		merchantName := fmt.Sprintf("%s %s", adjective, noun)

		apiKey, _ := apikey.GenerateApiKey()

		req := db.CreateMerchantParams{
			Name:   merchantName,
			UserID: int32((i % 5) + 1),
			ApiKey: apiKey,
		}

		merchant, err := r.db.CreateMerchant(r.ctx, req)
		if err != nil {
			r.logger.Error("failed to seed merchant", zap.Int("merchant", i+1), zap.Error(err))
			return fmt.Errorf("failed to seed merchant %d: %w", i+1, err)
		}

		var status string
		if i < activeMerchants {
			status = "active"
		} else {
			status = "deactive"
		}

		_, err = r.db.UpdateMerchantStatus(r.ctx, db.UpdateMerchantStatusParams{
			MerchantID: merchant.MerchantID,
			Status:     status,
		})
		if err != nil {
			r.logger.Error("failed to update merchant status", zap.Int("merchantID", int(merchant.MerchantID)), zap.String("status", status), zap.Error(err))
			return fmt.Errorf("failed to update status for merchant ID %d: %w", merchant.MerchantID, err)
		}

		if i >= activeMerchants {
			_, err = r.db.TrashMerchant(r.ctx, merchant.MerchantID)
			if err != nil {
				r.logger.Error("failed to trash merchant", zap.Int("merchant", i+1), zap.Error(err))
				return fmt.Errorf("failed to trash merchant %d: %w", i+1, err)
			}
		}
	}

	r.logger.Info("merchant seeded successfully",
		zap.Int("totalMerchants", totalMerchants),
		zap.Int("activeMerchants", activeMerchants),
		zap.Int("trashedMerchants", trashedMerchants))

	return nil
}
