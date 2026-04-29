package seeder

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	db "github.com/MamangRust/microservice-payment-gateway-grpc/pkg/database/schema"
	"github.com/MamangRust/microservice-payment-gateway-grpc/pkg/logger"
	"go.uber.org/zap"
)

// transactionSeeder is a struct that represents a seeder for the transactions table.
type transactionSeeder struct {
	db     *db.Queries
	ctx    context.Context
	logger logger.LoggerInterface
}

// NewTransactionSeeder creates a new instance of the transactionSeeder, which is
// responsible for populating the transactions table with fake data.
//
// Args:
// db: a pointer to the database queries
// ctx: a context.Context object
// logger: a logger.LoggerInterface object
//
// Returns:
// a pointer to the transactionSeeder struct
func NewTransactionSeeder(db *db.Queries, ctx context.Context, logger logger.LoggerInterface) *transactionSeeder {
	return &transactionSeeder{
		db:     db,
		ctx:    ctx,
		logger: logger,
	}
}

// Seed populates the transactions table with fake data.
//
// It generates a total of 10 transactions, with 5 of them being active and 5 of them being
// trashed. The transactions are seeded with random payment methods, statuses, and transaction
// times. The card numbers are randomly selected from the cards table. The merchant IDs are
// randomly selected from the merchants table.
//
// If any of the transactions fail to be created, the function returns an error.
//
// Returns:
// an error if any of the transactions fail to be created, otherwise nil
func (r *transactionSeeder) Seed() error {
	total := 500
	active := 450
	trashed := total - active

	paymentMethods := []string{"Bank Alpha", "Bank Beta", "Bank Gamma"}
	statusOptions := []string{"pending", "success", "failed"}

	var cards []db.GetCardByUserIDRow
	for i := 1; i <= total; i++ {
		card, err := r.db.GetCardByUserID(r.ctx, int32(i))
		if err != nil {
			r.logger.Error("failed to get card for user", zap.Int("userID", i), zap.Error(err))
			return fmt.Errorf("failed to get card for user %d: %w", i, err)
		}
		if card == nil {
			r.logger.Error("no card found for user", zap.Int("userID", i))
			continue
		}
		cards = append(cards, *card)
	}

	if len(cards) < total {
		r.logger.Error("not enough cards for transaction seeding", zap.Int("required", total), zap.Int("available", len(cards)))
		return fmt.Errorf("not enough cards for transaction seeding: required %d, got %d", total, len(cards))
	}

	merchants, err := r.db.GetMerchants(r.ctx, db.GetMerchantsParams{
		Column1: "",
		Limit:   int32(total),
		Offset:  0,
	})
	if err != nil {
		r.logger.Error("failed to get merchant list", zap.Error(err))
		return fmt.Errorf("failed to get merchant list: %w", err)
	}

	if len(merchants) < total {
		r.logger.Error("not enough merchants for transaction seeding", zap.Int("required", total), zap.Int("available", len(merchants)))
		return fmt.Errorf("not enough merchants: required %d, got %d", total, len(merchants))
	}

	months := make([]time.Time, 12)
	currentYear := time.Now().Year()
	for i := 0; i < 12; i++ {
		months[i] = time.Date(currentYear, time.Month(i+1), 1, 0, 0, 0, 0, time.UTC)
	}

	for i := 0; i < total; i++ {
		card := cards[i]
		merchant := merchants[i%len(merchants)]
		paymentMethod := paymentMethods[i%len(paymentMethods)]
		status := statusOptions[i%len(statusOptions)]

		monthIndex := i % 12
		transactionTime := months[monthIndex].Add(time.Duration(rand.Intn(28)) * 24 * time.Hour)

		request := db.CreateTransactionParams{
			CardNumber:      card.CardNumber,
			Amount:          int32(rand.Intn(1000000-50000) + 50000),
			PaymentMethod:   paymentMethod,
			MerchantID:      merchant.MerchantID,
			TransactionTime: transactionTime,
		}

		transaction, err := r.db.CreateTransaction(r.ctx, request)
		if err != nil {
			r.logger.Error("failed to seed transaction", zap.Int("index", i), zap.Error(err))
			return fmt.Errorf("failed to seed transaction %d: %w", i, err)
		}

		_, err = r.db.UpdateTransactionStatus(r.ctx, db.UpdateTransactionStatusParams{
			TransactionID: transaction.TransactionID,
			Status:        status,
		})
		if err != nil {
			r.logger.Error("failed to update transaction status", zap.Int("transactionID", int(transaction.TransactionID)), zap.String("status", status), zap.Error(err))
			return fmt.Errorf("failed to update status for transaction ID %d: %w", transaction.TransactionID, err)
		}

		if i >= active {
			_, err = r.db.TrashTransaction(r.ctx, transaction.TransactionID)
			if err != nil {
				r.logger.Error("failed to trash transaction", zap.Int("index", i), zap.Error(err))
				return fmt.Errorf("failed to trash transaction %d: %w", i, err)
			}
		}
	}

	r.logger.Info("transaction seeded successfully", zap.Int("total", total), zap.Int("active", active), zap.Int("trashed", trashed))
	return nil
}
