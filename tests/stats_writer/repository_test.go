package stats_writer_test

import (
	"context"
	"testing"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/MamangRust/microservice-payment-gateway-grpc/pkg/logger"
	"github.com/MamangRust/microservice-payment-gateway-grpc/service/stats-writer/repository"
	"github.com/MamangRust/microservice-payment-gateway-grpc/shared/domain/events"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	chcontainer "github.com/testcontainers/testcontainers-go/modules/clickhouse"
	sdklog "go.opentelemetry.io/otel/sdk/log"
)

func setupClickHouse(t *testing.T) (clickhouse.Conn, func()) {
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

	conn, err := clickhouse.Open(&clickhouse.Options{
		Addr: []string{host},
		Auth: clickhouse.Auth{
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

func TestRepo_InsertTransactionEvent_And_Flush(t *testing.T) {
	conn, cleanup := setupClickHouse(t)
	defer cleanup()

	log, _ := logger.NewLogger("test", sdklog.NewLoggerProvider())
	repo := repository.NewClickhouseRepository(conn, log)
	defer repo.Close()

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
		ApiKey:        "test-key",
		CreatedAt:     time.Now().UTC(),
	}

	err := repo.InsertTransactionEvent(context.Background(), event)
	require.NoError(t, err)

	err = repo.Flush(context.Background())
	require.NoError(t, err)

	var count uint64
	err = conn.QueryRow(context.Background(), "SELECT count() FROM transaction_events").Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, uint64(1), count)
}

func TestRepo_InsertTopupEvent_And_Flush(t *testing.T) {
	conn, cleanup := setupClickHouse(t)
	defer cleanup()

	log, _ := logger.NewLogger("test", sdklog.NewLoggerProvider())
	repo := repository.NewClickhouseRepository(conn, log)
	defer repo.Close()

	event := events.TopupEvent{
		TopupID:       1,
		TopupNo:       "TOP-001",
		CardNumber:    "1234567890",
		CardType:      "debit",
		CardProvider:  "mastercard",
		Amount:        100000,
		PaymentMethod: "bank_transfer",
		Status:        "success",
		CreatedAt:     time.Now().UTC(),
	}

	err := repo.InsertTopupEvent(context.Background(), event)
	require.NoError(t, err)

	err = repo.Flush(context.Background())
	require.NoError(t, err)

	var count uint64
	err = conn.QueryRow(context.Background(), "SELECT count() FROM topup_events").Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, uint64(1), count)
}

func TestRepo_InsertTransferEvent_And_Flush(t *testing.T) {
	conn, cleanup := setupClickHouse(t)
	defer cleanup()

	log, _ := logger.NewLogger("test", sdklog.NewLoggerProvider())
	repo := repository.NewClickhouseRepository(conn, log)
	defer repo.Close()

	event := events.TransferEvent{
		TransferID:      1,
		TransferNo:      "TRF-001",
		SourceCard:      "1234567890",
		DestinationCard: "0987654321",
		Amount:          75000,
		Status:          "success",
		CreatedAt:       time.Now().UTC(),
	}

	err := repo.InsertTransferEvent(context.Background(), event)
	require.NoError(t, err)

	err = repo.Flush(context.Background())
	require.NoError(t, err)

	var count uint64
	err = conn.QueryRow(context.Background(), "SELECT count() FROM transfer_events").Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, uint64(1), count)
}

func TestRepo_InsertWithdrawEvent_And_Flush(t *testing.T) {
	conn, cleanup := setupClickHouse(t)
	defer cleanup()

	log, _ := logger.NewLogger("test", sdklog.NewLoggerProvider())
	repo := repository.NewClickhouseRepository(conn, log)
	defer repo.Close()

	event := events.WithdrawEvent{
		WithdrawID: 1,
		WithdrawNo: "WD-001",
		CardNumber: "1234567890",
		CardType:   "debit",
		Amount:     30000,
		Status:     "success",
		CreatedAt:  time.Now().UTC(),
	}

	err := repo.InsertWithdrawEvent(context.Background(), event)
	require.NoError(t, err)

	err = repo.Flush(context.Background())
	require.NoError(t, err)

	var count uint64
	err = conn.QueryRow(context.Background(), "SELECT count() FROM withdraw_events").Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, uint64(1), count)
}

func TestRepo_InsertSaldoEvent_And_Flush(t *testing.T) {
	conn, cleanup := setupClickHouse(t)
	defer cleanup()

	log, _ := logger.NewLogger("test", sdklog.NewLoggerProvider())
	repo := repository.NewClickhouseRepository(conn, log)
	defer repo.Close()

	event := events.SaldoEvent{
		CardNumber:   "1234567890",
		TotalBalance: 5000000,
		CreatedAt:    time.Now().UTC(),
	}

	err := repo.InsertSaldoEvent(context.Background(), event)
	require.NoError(t, err)

	err = repo.Flush(context.Background())
	require.NoError(t, err)

	var count uint64
	err = conn.QueryRow(context.Background(), "SELECT count() FROM saldo_events").Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, uint64(1), count)
}

func TestRepo_InsertMerchantEvent_And_Flush(t *testing.T) {
	conn, cleanup := setupClickHouse(t)
	defer cleanup()

	log, _ := logger.NewLogger("test", sdklog.NewLoggerProvider())
	repo := repository.NewClickhouseRepository(conn, log)
	defer repo.Close()

	event := events.MerchantEvent{
		MerchantID: 10,
		UserID:     100,
		Name:       "Test Merchant",
		Email:      "merchant@test.com",
		Status:     "active",
		CreatedAt:  time.Now().UTC(),
	}

	err := repo.InsertMerchantEvent(context.Background(), event)
	require.NoError(t, err)

	err = repo.Flush(context.Background())
	require.NoError(t, err)

	var count uint64
	err = conn.QueryRow(context.Background(), "SELECT count() FROM merchant_events").Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, uint64(1), count)
}

func TestRepo_InsertCardEvent_And_Flush(t *testing.T) {
	conn, cleanup := setupClickHouse(t)
	defer cleanup()

	log, _ := logger.NewLogger("test", sdklog.NewLoggerProvider())
	repo := repository.NewClickhouseRepository(conn, log)
	defer repo.Close()

	event := events.CardEvent{
		CardID:       1,
		UserID:       100,
		CardNumber:   "1234567890",
		CardType:     "credit",
		CardProvider: "visa",
		Status:       "active",
		CreatedAt:    time.Now().UTC(),
	}

	err := repo.InsertCardEvent(context.Background(), event)
	require.NoError(t, err)

	err = repo.Flush(context.Background())
	require.NoError(t, err)

	var count uint64
	err = conn.QueryRow(context.Background(), "SELECT count() FROM card_events").Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, uint64(1), count)
}

func TestRepo_MultipleInserts_Flush(t *testing.T) {
	conn, cleanup := setupClickHouse(t)
	defer cleanup()

	log, _ := logger.NewLogger("test", sdklog.NewLoggerProvider())
	repo := repository.NewClickhouseRepository(conn, log)
	defer repo.Close()

	now := time.Now().UTC()

	for i := uint64(1); i <= 5; i++ {
		err := repo.InsertTransactionEvent(context.Background(), events.TransactionEvent{
			TransactionID: i,
			TransactionNo: "TXN-" + string(rune('A'+i)),
			CardNumber:    "1234567890",
			CardType:      "credit",
			CardProvider:  "visa",
			Amount:        10000,
			PaymentMethod: "credit_card",
			MerchantID:    10,
			MerchantName:  "Test",
			Status:        "success",
			CreatedAt:     now,
		})
		require.NoError(t, err)
	}

	err := repo.Flush(context.Background())
	require.NoError(t, err)

	var count uint64
	err = conn.QueryRow(context.Background(), "SELECT count() FROM transaction_events").Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, uint64(5), count)
}

func TestRepo_Flush_EmptyBatches(t *testing.T) {
	conn, cleanup := setupClickHouse(t)
	defer cleanup()

	log, _ := logger.NewLogger("test", sdklog.NewLoggerProvider())
	repo := repository.NewClickhouseRepository(conn, log)
	defer repo.Close()

	err := repo.Flush(context.Background())
	assert.NoError(t, err)
}

func TestRepo_Close(t *testing.T) {
	conn, cleanup := setupClickHouse(t)
	defer cleanup()

	log, _ := logger.NewLogger("test", sdklog.NewLoggerProvider())
	repo := repository.NewClickhouseRepository(conn, log)

	err := repo.Close()
	assert.NoError(t, err)
}
