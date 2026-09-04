package db

import (
	"context"
	"time"
)

// StuckTransaction is a transaction row the recovery worker should re-drive.
type StuckTransaction struct {
	TransactionID int32
	OperationID   string
	CardNumber    string
	Amount        int32
	MerchantID    int32
	Status        string
	UpdatedAt     time.Time
}

// StuckTransfer is a transfer row the recovery worker should re-drive.
// GuardTransactionStatus atomically moves a transaction from an expected status
// to a new one. It returns pgx.ErrNoRows when the row is not in the expected
// status (another actor already moved it), which callers use to detect races.
func (q *Queries) GuardTransactionStatus(ctx context.Context, id int32, fromStatus, toStatus, reason string) (*UpdateTransactionStatusRow, error) {
	row := q.db.QueryRow(ctx, `
		UPDATE transactions
		SET status = $3,
		    failure_reason = CASE WHEN $4 = '' THEN failure_reason ELSE $4 END,
		    updated_at = current_timestamp
		WHERE transaction_id = $1 AND status = $2 AND deleted_at IS NULL
		RETURNING transaction_id, transaction_no, card_number, amount, payment_method,
		          merchant_id, transaction_time, status, created_at, updated_at`,
		id, fromStatus, toStatus, reason)
	var i UpdateTransactionStatusRow
	err := row.Scan(
		&i.TransactionID, &i.TransactionNo, &i.CardNumber, &i.Amount,
		&i.PaymentMethod, &i.MerchantID, &i.TransactionTime, &i.Status,
		&i.CreatedAt, &i.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &i, nil
}

// ListStuckTransactions returns transactions stuck in a recoverable state that
// have not been touched for at least olderThan, limited to maxRows.
func (q *Queries) ListStuckTransactions(ctx context.Context, olderThan time.Duration, maxRows int32) ([]*StuckTransaction, error) {
	rows, err := q.db.Query(ctx, `
		SELECT transaction_id, operation_id::text, card_number, amount, merchant_id, status, updated_at
		FROM transactions
		WHERE status IN ('processing', 'compensating', 'unknown')
		  AND deleted_at IS NULL
		  AND updated_at < current_timestamp - ($1 * interval '1 second')
		ORDER BY updated_at
		LIMIT $2`, int64(olderThan.Seconds()), maxRows)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*StuckTransaction
	for rows.Next() {
		var s StuckTransaction
		if err := rows.Scan(&s.TransactionID, &s.OperationID, &s.CardNumber, &s.Amount, &s.MerchantID, &s.Status, &s.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, &s)
	}
	return out, rows.Err()
}

