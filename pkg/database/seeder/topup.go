package seeder

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/MamangRust/microservice-payment-gateway-grpc/pkg/logger"
	carddb "github.com/MamangRust/microservice-payment-gateway-grpc/service/card/database/schema"
	db "github.com/MamangRust/microservice-payment-gateway-grpc/service/topup/database/schema"
	"go.uber.org/zap"
)

// topupSeeder is a struct that represents a seeder for the topups table.
type topupSeeder struct {
	db     *db.Queries
	carddb *carddb.Queries
	ctx    context.Context
	logger logger.LoggerInterface
}

// NewTopupSeeder creates a new instance of the topupSeeder, which is
// responsible for populating the topups table with fake data.
//
// Args:
// db: a pointer to the database queries
// ctx: a context.Context object
// logger: a logger.LoggerInterface object
//
// Returns:
// a pointer to the topupSeeder struct
func NewTopupSeeder(db *db.Queries, carddb *carddb.Queries, ctx context.Context, logger logger.LoggerInterface) *topupSeeder {
	return &topupSeeder{
		db:     db,
		carddb: carddb,
		ctx:    ctx,
		logger: logger,
	}
}

// Seed populates the topups table with fake data.
//
// It creates a total of 10 topups, with 5 of them being active and 5 of them being
// trashed. The topups are seeded with a random card number, topup amount, topup method,
// and topup time. The status of the topup is randomly set to one of the following:
// pending, success, or failed.
//
// If any errors occur during the seeding process, an error is returned.
//
// Returns:
// an error if any of the topups fail to be created, otherwise nil
func (r *topupSeeder) Seed() error {
	totalTopups := 250
	activeTopups := 230
	trashedTopups := totalTopups - activeTopups

	var cards []carddb.GetCardByUserIDRow

	for i := 1; i <= totalTopups; i++ {
		cardList, err := r.carddb.GetCardByUserID(r.ctx, int32(i))
		if err != nil {
			r.logger.Error("failed to get card for user", zap.Int("userID", i), zap.Error(err))
			return fmt.Errorf("failed to get card for user %d: %w", i, err)
		}

		if cardList == nil {
			r.logger.Error("no card found for user", zap.Int("userID", i))
			continue
		}
		cards = append(cards, *cardList)
	}

	if len(cards) < totalTopups {
		r.logger.Error("not enough cards found to seed topups", zap.Int("found", len(cards)))
		return fmt.Errorf("not enough cards to seed topups")
	}

	topupMethods := []string{"Bank Alpha", "Bank Beta", "Bank Gamma"}
	statusOptions := []string{"pending", "success", "failed"}

	months := make([]time.Time, 12)
	currentYear := time.Now().Year()
	for i := 0; i < 12; i++ {
		months[i] = time.Date(currentYear, time.Month(i+1), 1, 0, 0, 0, 0, time.UTC)
	}

	for i := 0; i < totalTopups; i++ {
		card := cards[i]
		cardNumber := card.CardNumber

		monthIndex := i % 12
		topupTime := months[monthIndex].Add(time.Duration(rand.Intn(28)) * 24 * time.Hour)

		topupAmount := amountToInt64(int64(rand.Intn(10000000) + 1000000))
		request := db.CreateTopupParams{
			CardNumber:  cardNumber,
			TopupAmount: topupAmount,
			TopupMethod: topupMethods[rand.Intn(len(topupMethods))],
			TopupTime:   topupTime,
		}

		topup, err := r.db.CreateTopup(r.ctx, request)
		if err != nil {
			r.logger.Error("failed to seed topup", zap.String("card", cardNumber), zap.Error(err))
			return fmt.Errorf("failed to seed topup for card %s: %w", cardNumber, err)
		}

		status := statusOptions[rand.Intn(len(statusOptions))]
		_, err = r.db.UpdateTopupStatus(r.ctx, db.UpdateTopupStatusParams{
			TopupID: topup.TopupID,
			Status:  status,
		})
		if err != nil {
			r.logger.Error("failed to update topup status", zap.Int("topupID", int(topup.TopupID)), zap.Error(err))
			return fmt.Errorf("failed to update status for topup %d: %w", topup.TopupID, err)
		}

		if i >= activeTopups {
			_, err = r.db.TrashTopup(r.ctx, topup.TopupID)
			if err != nil {
				r.logger.Error("failed to trash topup", zap.Int("topup", i+1), zap.String("card", cardNumber), zap.Error(err))
				return fmt.Errorf("failed to trash topup %d for card %s: %w", i+1, cardNumber, err)
			}
		}
	}

	r.logger.Info("topup seeded successfully", zap.Int("totalTopups", totalTopups), zap.Int("activeTopups", activeTopups), zap.Int("trashedTopups", trashedTopups))
	return nil
}
