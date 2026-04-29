package seeder

import (
	"context"
	"fmt"

	db "github.com/MamangRust/microservice-payment-gateway-grpc/pkg/database/schema"
	"github.com/MamangRust/microservice-payment-gateway-grpc/pkg/date"
	"github.com/MamangRust/microservice-payment-gateway-grpc/pkg/logger"
	"github.com/MamangRust/microservice-payment-gateway-grpc/pkg/randomvcc"
	"github.com/jackc/pgx/v5/pgtype"
	"go.uber.org/zap"
)

// cardSeeder is a struct that represents a seeder for the cards table.
type cardSeeder struct {
	db     *db.Queries
	ctx    context.Context
	logger logger.LoggerInterface
}

// NewCardSeeder creates a new instance of the cardSeeder, which is
// responsible for populating the cards table with fake data.
func NewCardSeeder(db *db.Queries, ctx context.Context, logger logger.LoggerInterface) *cardSeeder {
	return &cardSeeder{
		db:     db,
		ctx:    ctx,
		logger: logger,
	}
}

// Seed populates the cards table with fake data.
//
// It generates 10 cards, 5 of which are active and 5 of which are trashed.
// The cards are seeded with random card numbers, card types, expire dates, and
// CVV numbers. The card provider is randomly selected from a list of providers.
//
// If any errors occur during the seeding process, an error is returned.
func (r *cardSeeder) Seed() error {
	cardTypes := []string{"credit", "debit"}
	cardProviders := []string{"mandiri", "bni", "bri"}

	totalCards := 120
	activeCards := 100
	trashedCards := totalCards - activeCards

	cardNumbers := make([]string, totalCards)
	for i := 0; i < totalCards; i++ {
		cardNumber, err := randomvcc.RandomCardNumber()
		if err != nil {
			r.logger.Error("failed to generate card number", zap.Int("index", i), zap.Error(err))
			return fmt.Errorf("failed to generate card number: %w", err)
		}
		cardNumbers[i] = cardNumber
	}

	for i := 0; i < totalCards; i++ {
		expireDate := pgtype.Date{
			Time:  date.GenerateExpireDate(),
			Valid: true,
		}

		request := db.CreateCardParams{
			UserID:       int32(i + 1),
			CardNumber:   cardNumbers[i],
			CardType:     cardTypes[i%len(cardTypes)],
			ExpireDate:   expireDate,
			Cvv:          fmt.Sprintf("%03d", i%1000),
			CardProvider: cardProviders[i%len(cardProviders)],
		}

		card, err := r.db.CreateCard(r.ctx, request)
		if err != nil {
			r.logger.Error("failed to seed card", zap.Int("card", i+1), zap.Error(err))
			return fmt.Errorf("failed to seed card %d: %w", i+1, err)
		}

		if i >= activeCards {
			_, err = r.db.TrashCard(r.ctx, card.CardID)
			if err != nil {
				r.logger.Error("failed to trash card", zap.Int("card", i+1), zap.Error(err))
				return fmt.Errorf("failed to trash card %d: %w", i+1, err)
			}
		}
	}

	r.logger.Info("card seeded successfully", zap.Int("totalCards", totalCards), zap.Int("activeCards", activeCards), zap.Int("trashedCards", trashedCards))
	return nil
}
