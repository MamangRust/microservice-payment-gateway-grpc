package stats_reader_test

import (
	"context"
	"testing"
	"time"

	chdriver "github.com/ClickHouse/clickhouse-go/v2"
	"github.com/MamangRust/microservice-payment-gateway-grpc/pkg/logger"
	"github.com/MamangRust/microservice-payment-gateway-grpc/service/stats-reader/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	chcontainer "github.com/testcontainers/testcontainers-go/modules/clickhouse"
	sdklog "go.opentelemetry.io/otel/sdk/log"
)

func setupClickHouse(t *testing.T) (chdriver.Conn, func()) {
	t.Helper()
	ctx := context.Background()

	chContainer, err := chcontainer.Run(ctx,
		"clickhouse/clickhouse-server:24.3-alpine",
		chcontainer.WithUsername("testuser"),
		chcontainer.WithPassword("testpass"),
		chcontainer.WithDatabase("testdb"),
	)
	require.NoError(t, err)

	t.Cleanup(func() {
		_ = testcontainers.TerminateContainer(chContainer)
	})

	host, err := chContainer.ConnectionHost(ctx)
	require.NoError(t, err)

	conn, err := chdriver.Open(&chdriver.Options{
		Addr: []string{host},
		Auth: chdriver.Auth{
			Database: "testdb",
			Username: "testuser",
			Password: "testpass",
		},
		DialTimeout: 10 * time.Second,
	})
	require.NoError(t, err)
	require.NoError(t, conn.Ping(ctx))

	log, _ := logger.NewLogger("test", sdklog.NewLoggerProvider())
	require.NoError(t, applySchema(ctx, conn, log))

	return conn, func() { conn.Close() }
}

func seedTransactionEvents(t *testing.T, conn chdriver.Conn, createdAt time.Time, txID uint64, cardNumber, paymentMethod, status string, amount int64) {
	t.Helper()
	ctx := context.Background()
	err := conn.Exec(ctx, `INSERT INTO transaction_events (transaction_id, transaction_no, card_number, card_type, card_provider, amount, payment_method, merchant_id, merchant_name, status, apikey, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		txID, "TXN-"+string(rune('A'+txID)), cardNumber, "credit", "visa", amount, paymentMethod, 10, "Test Merchant", status, "test-key", createdAt)
	require.NoError(t, err)
}

func seedTopupEvents(t *testing.T, conn chdriver.Conn, createdAt time.Time, topupID uint64, cardNumber, paymentMethod, status string, amount int64) {
	t.Helper()
	ctx := context.Background()
	err := conn.Exec(ctx, `INSERT INTO topup_events (topup_id, topup_no, card_number, card_type, card_provider, amount, payment_method, status, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		topupID, "TOP-"+string(rune('A'+topupID)), cardNumber, "debit", "mastercard", amount, paymentMethod, status, createdAt)
	require.NoError(t, err)
}

func seedTransferEvents(t *testing.T, conn chdriver.Conn, createdAt time.Time, transferID uint64, sourceCard, destCard, status string, amount int64) {
	t.Helper()
	ctx := context.Background()
	err := conn.Exec(ctx, `INSERT INTO transfer_events (transfer_id, transfer_no, source_card, destination_card, amount, status, created_at) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		transferID, "TRF-"+string(rune('A'+transferID)), sourceCard, destCard, amount, status, createdAt)
	require.NoError(t, err)
}

func seedWithdrawEvents(t *testing.T, conn chdriver.Conn, createdAt time.Time, withdrawID uint64, cardNumber, status string, amount int64) {
	t.Helper()
	ctx := context.Background()
	err := conn.Exec(ctx, `INSERT INTO withdraw_events (withdraw_id, withdraw_no, card_number, card_type, amount, status, created_at) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		withdrawID, "WD-"+string(rune('A'+withdrawID)), cardNumber, "debit", amount, status, createdAt)
	require.NoError(t, err)
}

func seedSaldoEvents(t *testing.T, conn chdriver.Conn, createdAt time.Time, cardNumber string, balance int64) {
	t.Helper()
	ctx := context.Background()
	err := conn.Exec(ctx, `INSERT INTO saldo_events (card_number, total_balance, created_at) VALUES (?, ?, ?)`,
		cardNumber, balance, createdAt)
	require.NoError(t, err)
}

// --- Transaction Stats ---

func TestRepo_GetMonthlyAmounts_Transaction(t *testing.T) {
	conn, cleanup := setupClickHouse(t)
	defer cleanup()

	createdAt := time.Date(2026, 3, 15, 10, 0, 0, 0, time.UTC)
	seedTransactionEvents(t, conn, createdAt, 1, "1234567890", "credit_card", "success", 50000)
	seedTransactionEvents(t, conn, createdAt, 2, "1234567890", "bank_transfer", "success", 30000)

	repo := repository.NewClickHouseReaderRepository(conn)

	results, err := repo.GetMonthlyAmounts(context.Background(), "transaction_events", "", nil, 2026)
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "2026", results[0].Year)
	assert.Equal(t, "Mar", results[0].Month)
	assert.Equal(t, int64(80000), results[0].TotalAmount)
}

func TestRepo_GetMonthlyAmounts_NoData(t *testing.T) {
	conn, cleanup := setupClickHouse(t)
	defer cleanup()

	repo := repository.NewClickHouseReaderRepository(conn)
	results, err := repo.GetMonthlyAmounts(context.Background(), "transaction_events", "", nil, 2099)
	require.NoError(t, err)
	assert.Empty(t, results)
}

func TestRepo_GetYearlyAmounts_Transaction(t *testing.T) {
	conn, cleanup := setupClickHouse(t)
	defer cleanup()

	createdAt2026 := time.Date(2026, 6, 1, 10, 0, 0, 0, time.UTC)
	createdAt2025 := time.Date(2025, 8, 1, 10, 0, 0, 0, time.UTC)

	seedTransactionEvents(t, conn, createdAt2026, 1, "1234567890", "credit_card", "success", 100000)
	seedTransactionEvents(t, conn, createdAt2025, 2, "1234567890", "bank_transfer", "success", 75000)

	repo := repository.NewClickHouseReaderRepository(conn)
	results, err := repo.GetYearlyAmounts(context.Background(), "transaction_events", "", nil, 2025, 2026)
	require.NoError(t, err)
	assert.Len(t, results, 2)
}

func TestRepo_GetMonthlyMethodStats(t *testing.T) {
	conn, cleanup := setupClickHouse(t)
	defer cleanup()

	createdAt := time.Date(2026, 4, 10, 10, 0, 0, 0, time.UTC)
	seedTransactionEvents(t, conn, createdAt, 1, "1234567890", "credit_card", "success", 50000)
	seedTransactionEvents(t, conn, createdAt, 2, "1234567890", "credit_card", "success", 30000)
	seedTransactionEvents(t, conn, createdAt, 3, "1234567890", "bank_transfer", "success", 10000)

	repo := repository.NewClickHouseReaderRepository(conn)
	results, err := repo.GetMonthlyMethodStats(context.Background(), "transaction_events", "", nil, 2026)
	require.NoError(t, err)
	require.Len(t, results, 2)
}

func TestRepo_GetMonthlyStatusStats(t *testing.T) {
	conn, cleanup := setupClickHouse(t)
	defer cleanup()

	createdAt := time.Date(2026, 5, 12, 10, 0, 0, 0, time.UTC)
	seedTransactionEvents(t, conn, createdAt, 1, "1234567890", "credit_card", "success", 50000)
	seedTransactionEvents(t, conn, createdAt, 2, "1234567890", "credit_card", "failed", 0)

	repo := repository.NewClickHouseReaderRepository(conn)
	results, err := repo.GetMonthlyStatusStats(context.Background(), "transaction_events", "", nil, 2026, "success")
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "success", results[0].Status)
	assert.Equal(t, uint64(1), results[0].TotalTransactions)
}

func TestRepo_GetYearlyStatusStats(t *testing.T) {
	conn, cleanup := setupClickHouse(t)
	defer cleanup()

	createdAt2026 := time.Date(2026, 6, 1, 10, 0, 0, 0, time.UTC)
	createdAt2025 := time.Date(2025, 6, 1, 10, 0, 0, 0, time.UTC)

	seedTransactionEvents(t, conn, createdAt2026, 1, "1234567890", "credit_card", "success", 50000)
	seedTransactionEvents(t, conn, createdAt2025, 2, "1234567890", "bank_transfer", "success", 30000)

	repo := repository.NewClickHouseReaderRepository(conn)
	results, err := repo.GetYearlyStatusStats(context.Background(), "transaction_events", "", nil, 2026, "success")
	require.NoError(t, err)
	assert.Len(t, results, 2)
}

// --- Topup Stats ---

func TestRepo_GetMonthlyAmounts_Topup(t *testing.T) {
	conn, cleanup := setupClickHouse(t)
	defer cleanup()

	createdAt := time.Date(2026, 3, 15, 10, 0, 0, 0, time.UTC)
	seedTopupEvents(t, conn, createdAt, 1, "1234567890", "bank_transfer", "success", 100000)

	repo := repository.NewClickHouseReaderRepository(conn)
	results, err := repo.GetMonthlyAmounts(context.Background(), "topup_events", "", nil, 2026)
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, int64(100000), results[0].TotalAmount)
}

// --- Transfer Stats ---

func TestRepo_GetMonthlyAmounts_Transfer(t *testing.T) {
	conn, cleanup := setupClickHouse(t)
	defer cleanup()

	createdAt := time.Date(2026, 4, 10, 10, 0, 0, 0, time.UTC)
	seedTransferEvents(t, conn, createdAt, 1, "1234567890", "0987654321", "success", 75000)

	repo := repository.NewClickHouseReaderRepository(conn)
	results, err := repo.GetMonthlyAmounts(context.Background(), "transfer_events", "", nil, 2026)
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, int64(75000), results[0].TotalAmount)
}

func TestRepo_GetMonthlyStatusStats_Transfer(t *testing.T) {
	conn, cleanup := setupClickHouse(t)
	defer cleanup()

	createdAt := time.Date(2026, 5, 12, 10, 0, 0, 0, time.UTC)
	seedTransferEvents(t, conn, createdAt, 1, "1234567890", "0987654321", "success", 75000)
	seedTransferEvents(t, conn, createdAt, 2, "1234567890", "0987654321", "failed", 0)

	repo := repository.NewClickHouseReaderRepository(conn)
	results, err := repo.GetMonthlyStatusStats(context.Background(), "transfer_events", "", nil, 2026, "success")
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, uint64(1), results[0].TotalTransactions)
}

// --- Withdraw Stats ---

func TestRepo_GetMonthlyAmounts_Withdraw(t *testing.T) {
	conn, cleanup := setupClickHouse(t)
	defer cleanup()

	createdAt := time.Date(2026, 6, 1, 10, 0, 0, 0, time.UTC)
	seedWithdrawEvents(t, conn, createdAt, 1, "1234567890", "success", 30000)

	repo := repository.NewClickHouseReaderRepository(conn)
	results, err := repo.GetMonthlyAmounts(context.Background(), "withdraw_events", "", nil, 2026)
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, int64(30000), results[0].TotalAmount)
}

func TestRepo_GetMonthlyStatusStats_Withdraw(t *testing.T) {
	conn, cleanup := setupClickHouse(t)
	defer cleanup()

	createdAt := time.Date(2026, 7, 1, 10, 0, 0, 0, time.UTC)
	seedWithdrawEvents(t, conn, createdAt, 1, "1234567890", "success", 30000)
	seedWithdrawEvents(t, conn, createdAt, 2, "1234567890", "failed", 0)

	repo := repository.NewClickHouseReaderRepository(conn)
	results, err := repo.GetMonthlyStatusStats(context.Background(), "withdraw_events", "", nil, 2026, "success")
	require.NoError(t, err)
	require.Len(t, results, 1)
}

// --- Filter by card_number ---

func TestRepo_GetMonthlyAmounts_ByCardNumber(t *testing.T) {
	conn, cleanup := setupClickHouse(t)
	defer cleanup()

	createdAt := time.Date(2026, 3, 15, 10, 0, 0, 0, time.UTC)
	seedTransactionEvents(t, conn, createdAt, 1, "1234567890", "credit_card", "success", 50000)
	seedTransactionEvents(t, conn, createdAt, 2, "0987654321", "bank_transfer", "success", 30000)

	repo := repository.NewClickHouseReaderRepository(conn)
	results, err := repo.GetMonthlyAmounts(context.Background(), "transaction_events", "card_number", "1234567890", 2026)
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, int64(50000), results[0].TotalAmount)
}

func TestRepo_GetYearlyMethodStats(t *testing.T) {
	conn, cleanup := setupClickHouse(t)
	defer cleanup()

	createdAt := time.Date(2026, 3, 15, 10, 0, 0, 0, time.UTC)
	seedTransactionEvents(t, conn, createdAt, 1, "1234567890", "credit_card", "success", 50000)
	seedTransactionEvents(t, conn, createdAt, 2, "1234567890", "bank_transfer", "success", 30000)

	repo := repository.NewClickHouseReaderRepository(conn)
	results, err := repo.GetYearlyMethodStats(context.Background(), "transaction_events", "", nil, 2026, 2026)
	require.NoError(t, err)
	require.Len(t, results, 2)
}

func TestRepo_GetMonthlyAmounts_Saldo(t *testing.T) {
	conn, cleanup := setupClickHouse(t)
	defer cleanup()

	createdAt := time.Date(2026, 3, 15, 10, 0, 0, 0, time.UTC)
	seedSaldoEvents(t, conn, createdAt, "1234567890", 5000000)

	repo := repository.NewClickHouseReaderRepository(conn)
	results, err := repo.GetMonthlyAmounts(context.Background(), "saldo_events", "", nil, 2026)
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, int64(5000000), results[0].TotalAmount)
}

func TestRepo_FindMerchantTransactions(t *testing.T) {
	conn, cleanup := setupClickHouse(t)
	defer cleanup()

	createdAt := time.Date(2026, 3, 15, 10, 0, 0, 0, time.UTC)
	seedTransactionEvents(t, conn, createdAt, 1, "1234567890", "credit_card", "success", 50000)

	repo := repository.NewClickHouseReaderRepository(conn)
	results, err := repo.FindMerchantTransactions(context.Background(), 10)
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, int64(50000), results[0]["amount"])
}
