package db

import (
	"context"
	"time"
)

// StuckTransaction is a transaction row the recovery worker should re-drive.
// StuckTransfer is a transfer row the recovery worker should re-drive.
type StuckTransfer struct {
	TransferID     int32
	OperationID    string
	TransferFrom   string
	TransferTo     string
	TransferAmount int32
	Status         string
	UpdatedAt      time.Time
}

// GuardTransferStatus is the transfer equivalent of GuardTransactionStatus.
func (q *Queries) GuardTransferStatus(ctx context.Context, id int32, fromStatus, toStatus, reason string) (*UpdateTransferStatusRow, error) {
	row := q.db.QueryRow(ctx, `
		UPDATE transfers
		SET status = $3,
		    failure_reason = CASE WHEN $4 = '' THEN failure_reason ELSE $4 END,
		    updated_at = current_timestamp
		WHERE transfer_id = $1 AND status = $2 AND deleted_at IS NULL
		RETURNING transfer_id, transfer_no, transfer_from, transfer_to,
		          transfer_amount, transfer_time, status, created_at, updated_at`,
		id, fromStatus, toStatus, reason)
	var i UpdateTransferStatusRow
	err := row.Scan(
		&i.TransferID, &i.TransferNo, &i.TransferFrom, &i.TransferTo,
		&i.TransferAmount, &i.TransferTime, &i.Status, &i.CreatedAt, &i.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &i, nil
}


// ListStuckTransfers returns transfers stuck in a recoverable state.
func (q *Queries) ListStuckTransfers(ctx context.Context, olderThan time.Duration, maxRows int32) ([]*StuckTransfer, error) {
	rows, err := q.db.Query(ctx, `
		SELECT transfer_id, operation_id::text, transfer_from, transfer_to, transfer_amount, status, updated_at
		FROM transfers
		WHERE status IN ('processing', 'compensating', 'unknown')
		  AND deleted_at IS NULL
		  AND updated_at < current_timestamp - ($1 * interval '1 second')
		ORDER BY updated_at
		LIMIT $2`, int64(olderThan.Seconds()), maxRows)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*StuckTransfer
	for rows.Next() {
		var s StuckTransfer
		if err := rows.Scan(&s.TransferID, &s.OperationID, &s.TransferFrom, &s.TransferTo, &s.TransferAmount, &s.Status, &s.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, &s)
	}
	return out, rows.Err()
}
