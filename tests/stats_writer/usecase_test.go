package stats_writer_test

import (
	"context"
	"testing"
	"time"

	"github.com/MamangRust/microservice-payment-gateway-grpc/service/stats-writer/repository"
	"github.com/MamangRust/microservice-payment-gateway-grpc/service/stats-writer/usecase"
	"github.com/MamangRust/microservice-payment-gateway-grpc/shared/domain/events"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- mock repo ---

type mockRepo struct {
	insertTransaction func(ctx context.Context, event events.TransactionEvent) error
	insertTopup       func(ctx context.Context, event events.TopupEvent) error
	insertTransfer    func(ctx context.Context, event events.TransferEvent) error
	insertWithdraw    func(ctx context.Context, event events.WithdrawEvent) error
	insertSaldo       func(ctx context.Context, event events.SaldoEvent) error
	insertMerchant    func(ctx context.Context, event events.MerchantEvent) error
	insertCard        func(ctx context.Context, event events.CardEvent) error
	flush             func(ctx context.Context) error
	closeFn           func() error

	insertTransactionCalled bool
	insertTopupCalled       bool
	insertTransferCalled    bool
	insertWithdrawCalled    bool
	insertSaldoCalled       bool
	insertMerchantCalled    bool
	insertCardCalled        bool
}

func (m *mockRepo) InsertTransactionEvent(ctx context.Context, event events.TransactionEvent) error {
	m.insertTransactionCalled = true
	if m.insertTransaction != nil {
		return m.insertTransaction(ctx, event)
	}
	return nil
}

func (m *mockRepo) InsertTopupEvent(ctx context.Context, event events.TopupEvent) error {
	m.insertTopupCalled = true
	if m.insertTopup != nil {
		return m.insertTopup(ctx, event)
	}
	return nil
}

func (m *mockRepo) InsertTransferEvent(ctx context.Context, event events.TransferEvent) error {
	m.insertTransferCalled = true
	if m.insertTransfer != nil {
		return m.insertTransfer(ctx, event)
	}
	return nil
}

func (m *mockRepo) InsertWithdrawEvent(ctx context.Context, event events.WithdrawEvent) error {
	m.insertWithdrawCalled = true
	if m.insertWithdraw != nil {
		return m.insertWithdraw(ctx, event)
	}
	return nil
}

func (m *mockRepo) InsertSaldoEvent(ctx context.Context, event events.SaldoEvent) error {
	m.insertSaldoCalled = true
	if m.insertSaldo != nil {
		return m.insertSaldo(ctx, event)
	}
	return nil
}

func (m *mockRepo) InsertMerchantEvent(ctx context.Context, event events.MerchantEvent) error {
	m.insertMerchantCalled = true
	if m.insertMerchant != nil {
		return m.insertMerchant(ctx, event)
	}
	return nil
}

func (m *mockRepo) InsertCardEvent(ctx context.Context, event events.CardEvent) error {
	m.insertCardCalled = true
	if m.insertCard != nil {
		return m.insertCard(ctx, event)
	}
	return nil
}

func (m *mockRepo) Flush(ctx context.Context) error {
	if m.flush != nil {
		return m.flush(ctx)
	}
	return nil
}

func (m *mockRepo) Close() error {
	if m.closeFn != nil {
		return m.closeFn()
	}
	return nil
}

var _ repository.Repository = (*mockRepo)(nil)

// --- tests ---

func TestUseCase_SaveTransactionEvent(t *testing.T) {
	repo := &mockRepo{}
	uc := usecase.NewStatsUseCase(repo)

	event := events.TransactionEvent{
		TransactionID: 1,
		TransactionNo: "TXN-001",
		CardNumber:    "1234567890",
		CardType:      "credit",
		CardProvider:  "visa",
		Amount:        50000,
		PaymentMethod: "credit_card",
		MerchantID:    10,
		MerchantName:  "Test Merchant",
		Status:        "success",
		ApiKey:        "test-api-key",
		CreatedAt:     time.Now(),
	}

	err := uc.SaveTransactionEvent(context.Background(), event)
	require.NoError(t, err)
	assert.True(t, repo.insertTransactionCalled)
}

func TestUseCase_SaveTransactionEvent_RepoError(t *testing.T) {
	repo := &mockRepo{
		insertTransaction: func(_ context.Context, _ events.TransactionEvent) error {
			return assert.AnError
		},
	}
	uc := usecase.NewStatsUseCase(repo)

	err := uc.SaveTransactionEvent(context.Background(), events.TransactionEvent{TransactionID: 1})
	require.Error(t, err)
	assert.True(t, repo.insertTransactionCalled)
}

func TestUseCase_SaveTopupEvent(t *testing.T) {
	repo := &mockRepo{}
	uc := usecase.NewStatsUseCase(repo)

	event := events.TopupEvent{
		TopupID:       1,
		TopupNo:       "TOP-001",
		CardNumber:    "1234567890",
		CardType:      "debit",
		CardProvider:  "mastercard",
		Amount:        100000,
		PaymentMethod: "bank_transfer",
		Status:        "success",
		CreatedAt:     time.Now(),
	}

	err := uc.SaveTopupEvent(context.Background(), event)
	require.NoError(t, err)
	assert.True(t, repo.insertTopupCalled)
}

func TestUseCase_SaveTransferEvent(t *testing.T) {
	repo := &mockRepo{}
	uc := usecase.NewStatsUseCase(repo)

	event := events.TransferEvent{
		TransferID:      1,
		TransferNo:      "TRF-001",
		SourceCard:      "1234567890",
		DestinationCard: "0987654321",
		Amount:          75000,
		Status:          "success",
		CreatedAt:       time.Now(),
	}

	err := uc.SaveTransferEvent(context.Background(), event)
	require.NoError(t, err)
	assert.True(t, repo.insertTransferCalled)
}

func TestUseCase_SaveWithdrawEvent(t *testing.T) {
	repo := &mockRepo{}
	uc := usecase.NewStatsUseCase(repo)

	event := events.WithdrawEvent{
		WithdrawID: 1,
		WithdrawNo: "WD-001",
		CardNumber: "1234567890",
		CardType:   "debit",
		Amount:     30000,
		Status:     "success",
		CreatedAt:  time.Now(),
	}

	err := uc.SaveWithdrawEvent(context.Background(), event)
	require.NoError(t, err)
	assert.True(t, repo.insertWithdrawCalled)
}

func TestUseCase_SaveSaldoEvent(t *testing.T) {
	repo := &mockRepo{}
	uc := usecase.NewStatsUseCase(repo)

	event := events.SaldoEvent{
		CardNumber:   "1234567890",
		TotalBalance: 5000000,
		CreatedAt:    time.Now(),
	}

	err := uc.SaveSaldoEvent(context.Background(), event)
	require.NoError(t, err)
	assert.True(t, repo.insertSaldoCalled)
}

func TestUseCase_SaveMerchantEvent(t *testing.T) {
	repo := &mockRepo{}
	uc := usecase.NewStatsUseCase(repo)

	event := events.MerchantEvent{
		MerchantID: 10,
		UserID:     100,
		Name:       "Test Merchant",
		Email:      "merchant@test.com",
		Status:     "active",
		CreatedAt:  time.Now(),
	}

	err := uc.SaveMerchantEvent(context.Background(), event)
	require.NoError(t, err)
	assert.True(t, repo.insertMerchantCalled)
}

func TestUseCase_SaveCardEvent(t *testing.T) {
	repo := &mockRepo{}
	uc := usecase.NewStatsUseCase(repo)

	event := events.CardEvent{
		CardID:       1,
		UserID:       100,
		CardNumber:   "1234567890",
		CardType:     "credit",
		CardProvider: "visa",
		Status:       "active",
		CreatedAt:    time.Now(),
	}

	err := uc.SaveCardEvent(context.Background(), event)
	require.NoError(t, err)
	assert.True(t, repo.insertCardCalled)
}

func TestUseCase_Close(t *testing.T) {
	repo := &mockRepo{}
	uc := usecase.NewStatsUseCase(repo)

	err := uc.Close()
	require.NoError(t, err)
}

func TestUseCase_Close_Error(t *testing.T) {
	repo := &mockRepo{
		closeFn: func() error { return assert.AnError },
	}
	uc := usecase.NewStatsUseCase(repo)

	err := uc.Close()
	require.Error(t, err)
}
