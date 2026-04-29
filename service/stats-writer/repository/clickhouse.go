package repository

import (
	"context"
	chDriver "github.com/ClickHouse/clickhouse-go/v2"
	"github.com/MamangRust/microservice-payment-gateway-grpc/pkg/logger"
	"github.com/MamangRust/microservice-payment-gateway-grpc/shared/domain/events"
)

type clickhouseRepository struct {
	conn chDriver.Conn
	log  logger.LoggerInterface
}

func NewClickhouseRepository(conn chDriver.Conn, log logger.LoggerInterface) Repository {
	return &clickhouseRepository{
		conn: conn,
		log:  log,
	}
}

func (r *clickhouseRepository) InsertTransactionEvent(ctx context.Context, event events.TransactionEvent) error {
	query := `
		INSERT INTO transaction_events (
			transaction_id, transaction_no, card_number, card_type, card_provider, amount, 
			payment_method, merchant_id, merchant_name, status, apikey, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	return r.conn.Exec(ctx, query,
		event.TransactionID, event.TransactionNo, event.CardNumber, event.CardType, event.CardProvider, event.Amount,
		event.PaymentMethod, event.MerchantID, event.MerchantName, event.Status, event.ApiKey, event.CreatedAt,
	)
}

func (r *clickhouseRepository) InsertTopupEvent(ctx context.Context, event events.TopupEvent) error {
	query := `
		INSERT INTO topup_events (
			topup_id, topup_no, card_number, card_type, card_provider, amount, 
			payment_method, status, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	return r.conn.Exec(ctx, query,
		event.TopupID, event.TopupNo, event.CardNumber, event.CardType, event.CardProvider,
		event.Amount, event.PaymentMethod, event.Status, event.CreatedAt,
	)
}

func (r *clickhouseRepository) InsertTransferEvent(ctx context.Context, event events.TransferEvent) error {
	query := `
		INSERT INTO transfer_events (
			transfer_id, transfer_no, source_card, destination_card, amount, status, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?)
	`
	return r.conn.Exec(ctx, query,
		event.TransferID, event.TransferNo, event.SourceCard, event.DestinationCard,
		event.Amount, event.Status, event.CreatedAt,
	)
}

func (r *clickhouseRepository) InsertWithdrawEvent(ctx context.Context, event events.WithdrawEvent) error {
	query := `
		INSERT INTO withdraw_events (
			withdraw_id, withdraw_no, card_number, card_type, amount, status, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?)
	`
	return r.conn.Exec(ctx, query,
		event.WithdrawID, event.WithdrawNo, event.CardNumber, event.CardType,
		event.Amount, event.Status, event.CreatedAt,
	)
}

func (r *clickhouseRepository) InsertSaldoEvent(ctx context.Context, event events.SaldoEvent) error {
	query := `
		INSERT INTO saldo_events (
			card_number, total_balance, created_at
		) VALUES (?, ?, ?)
	`
	return r.conn.Exec(ctx, query,
		event.CardNumber, event.TotalBalance, event.CreatedAt,
	)
}

func (r *clickhouseRepository) InsertMerchantEvent(ctx context.Context, event events.MerchantEvent) error {
	query := `
		INSERT INTO merchant_events (
			merchant_id, user_id, name, email, status, created_at
		) VALUES (?, ?, ?, ?, ?, ?)
	`
	return r.conn.Exec(ctx, query,
		event.MerchantID, event.UserID, event.Name, event.Email, event.Status, event.CreatedAt,
	)
}

func (r *clickhouseRepository) InsertCardEvent(ctx context.Context, event events.CardEvent) error {
	query := `
		INSERT INTO card_events (
			card_id, user_id, card_number, card_type, card_provider, status, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?)
	`
	return r.conn.Exec(ctx, query,
		event.CardID, event.UserID, event.CardNumber, event.CardType, event.CardProvider, event.Status, event.CreatedAt,
	)
}
