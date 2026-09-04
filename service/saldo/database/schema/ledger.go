package db

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

// ErrLedgerOperationConflict indicates that an operation ID was reused with
// different mutation details. Replaying such a request must never be treated as
// a successful no-op.
var ErrLedgerOperationConflict = errors.New("ledger operation identity conflict")

// LedgerMutationParams carries the durable identity for a saldo mutation.
type LedgerMutationParams struct {
	CardNumber  string
	Amount      int64
	OperationID string
	SourceType  string
	SourceID    string
}

// LedgerAdjustmentParams represents an append-only signed correction. Positive
// delta credits the saldo; negative delta debits it.
type LedgerAdjustmentParams struct {
	CardNumber  string
	Delta       int64
	OperationID string
	SourceType  string
	SourceID    string
	Note        string
}

// LedgerEntry is the immutable audit representation of a balance change.
type LedgerEntry struct {
	EntryID       int64
	OperationID   string
	CardNumber    string
	Direction     string
	Amount        int64
	Delta         int64
	BalanceBefore int64
	BalanceAfter  int64
	SourceType    string
	SourceID      *string
	Note          *string
	CreatedAt     time.Time
}

// ReconciliationRow compares current saldo with opening + ledger deltas.
type ReconciliationRow struct {
	SaldoID        int32
	CardNumber     string
	CurrentBalance int64
	LedgerBalance  int64
	Difference     int64
	LedgerEntries  int64
}

// ReconciliationQueueRow is the durable operator-facing mismatch record.
type ReconciliationQueueRow struct {
	QueueID               int64
	SaldoID               int32
	CardNumber            string
	CurrentBalance        int64
	LedgerBalance         int64
	Difference            int64
	LedgerEntries         int64
	Status                string
	MismatchCount         int64
	FirstSeenAt           time.Time
	LastSeenAt            time.Time
	ResolvedAt            *time.Time
	ResolutionOperationID *string
	ResolutionNote        *string
}

// SaldoAdjustmentRow is returned by an append-only correction.
type SaldoAdjustmentRow struct {
	SaldoID      int32
	CardNumber   string
	TotalBalance int64
}

// DebitSaldoWithLedger atomically updates saldo and appends one debit ledger
// entry. The operation identity makes a retried command idempotent.
func (q *Queries) existingLedgerMutation(ctx context.Context, arg LedgerMutationParams, direction string) (bool, error) {
	var amount int64
	var sourceType string
	var sourceID *string
	err := q.db.QueryRow(ctx, `
		SELECT amount, source_type, source_id
		FROM balance_ledger
		WHERE operation_id = $1 AND card_number = $2 AND direction = $3
	`, arg.OperationID, arg.CardNumber, direction).Scan(&amount, &sourceType, &sourceID)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	requestedSourceID := arg.SourceID
	storedSourceID := ""
	if sourceID != nil {
		storedSourceID = *sourceID
	}
	if amount != arg.Amount || sourceType != arg.SourceType || storedSourceID != requestedSourceID {
		return false, fmt.Errorf("%w: operation_id=%s", ErrLedgerOperationConflict, arg.OperationID)
	}
	return true, nil
}

func ledgerWithdrawAmount(value *int64) *int64 {
	if value == nil {
		return nil
	}
	converted := *value
	return &converted
}

func (q *Queries) DebitSaldoWithLedger(ctx context.Context, arg LedgerMutationParams) (*DebitSaldoRow, error) {
	const query = `
		WITH candidate AS MATERIALIZED (
			SELECT saldo_id, card_number, total_balance, withdraw_amount,
			       withdraw_time, created_at, updated_at
			FROM saldos
			WHERE card_number = $1 AND deleted_at IS NULL
			FOR UPDATE
		), inserted AS (
			INSERT INTO balance_ledger (
				operation_id, card_number, direction, amount, delta,
				balance_before, balance_after, source_type, source_id, note
			)
			SELECT $3, card_number, 'debit', $2::bigint, -$2::bigint, total_balance,
			       total_balance - $2::bigint, $4, NULLIF($5, ''), NULL
			FROM candidate
			WHERE $2::bigint > 0 AND total_balance >= $2::bigint
			ON CONFLICT (operation_id, card_number, direction) DO NOTHING
			RETURNING card_number
		)
		UPDATE saldos s
		SET total_balance = s.total_balance - $2::bigint,
		    updated_at = current_timestamp
		FROM inserted i
		WHERE s.card_number = i.card_number AND s.deleted_at IS NULL
		RETURNING s.saldo_id, s.card_number, s.total_balance,
		          s.withdraw_amount, s.withdraw_time, s.created_at, s.updated_at
	`
	row := q.db.QueryRow(ctx, query, arg.CardNumber, arg.Amount, arg.OperationID, arg.SourceType, arg.SourceID)
	var result DebitSaldoRow
	err := row.Scan(&result.SaldoID, &result.CardNumber, &result.TotalBalance,
		&result.WithdrawAmount, &result.WithdrawTime, &result.CreatedAt, &result.UpdatedAt)
	if err == pgx.ErrNoRows {
		if existing, lookupErr := q.existingLedgerMutation(ctx, arg, "debit"); lookupErr != nil {
			return nil, lookupErr
		} else if existing {
			current, currentErr := q.GetSaldoByCardNumber(ctx, arg.CardNumber)
			if currentErr != nil {
				return nil, currentErr
			}
			return &DebitSaldoRow{SaldoID: current.SaldoID, CardNumber: current.CardNumber, TotalBalance: current.TotalBalance, WithdrawAmount: ledgerWithdrawAmount(current.WithdrawAmount), WithdrawTime: current.WithdrawTime, CreatedAt: current.CreatedAt, UpdatedAt: current.UpdatedAt}, nil
		}
	}
	return &result, err
}

// CreditSaldoWithLedger atomically updates saldo and appends one credit ledger entry.
func (q *Queries) CreditSaldoWithLedger(ctx context.Context, arg LedgerMutationParams) (*CreditSaldoRow, error) {
	const query = `
		WITH candidate AS MATERIALIZED (
			SELECT saldo_id, card_number, total_balance, withdraw_amount,
			       withdraw_time, created_at, updated_at
			FROM saldos
			WHERE card_number = $1 AND deleted_at IS NULL
			FOR UPDATE
		), inserted AS (
			INSERT INTO balance_ledger (
				operation_id, card_number, direction, amount, delta,
				balance_before, balance_after, source_type, source_id, note
			)
			SELECT $3, card_number, 'credit', $2::bigint, $2::bigint, total_balance,
			       total_balance + $2::bigint, $4, NULLIF($5, ''), NULL
			FROM candidate
			WHERE $2::bigint > 0
			ON CONFLICT (operation_id, card_number, direction) DO NOTHING
			RETURNING card_number
		)
		UPDATE saldos s
		SET total_balance = s.total_balance + $2::bigint,
		    updated_at = current_timestamp
		FROM inserted i
		WHERE s.card_number = i.card_number AND s.deleted_at IS NULL
		RETURNING s.saldo_id, s.card_number, s.total_balance,
		          s.withdraw_amount, s.withdraw_time, s.created_at, s.updated_at
	`
	row := q.db.QueryRow(ctx, query, arg.CardNumber, arg.Amount, arg.OperationID, arg.SourceType, arg.SourceID)
	var result CreditSaldoRow
	err := row.Scan(&result.SaldoID, &result.CardNumber, &result.TotalBalance,
		&result.WithdrawAmount, &result.WithdrawTime, &result.CreatedAt, &result.UpdatedAt)
	if err == pgx.ErrNoRows {
		if existing, lookupErr := q.existingLedgerMutation(ctx, arg, "credit"); lookupErr != nil {
			return nil, lookupErr
		} else if existing {
			current, currentErr := q.GetSaldoByCardNumber(ctx, arg.CardNumber)
			if currentErr != nil {
				return nil, currentErr
			}
			return &CreditSaldoRow{SaldoID: current.SaldoID, CardNumber: current.CardNumber, TotalBalance: current.TotalBalance, WithdrawAmount: ledgerWithdrawAmount(current.WithdrawAmount), WithdrawTime: current.WithdrawTime, CreatedAt: current.CreatedAt, UpdatedAt: current.UpdatedAt}, nil
		}
	}
	return &result, err
}

func (q *Queries) existingLedgerAdjustment(ctx context.Context, arg LedgerAdjustmentParams) (bool, error) {
	var amount, delta int64
	var sourceType string
	var sourceID, note *string
	err := q.db.QueryRow(ctx, `
		SELECT amount, delta, source_type, source_id, note
		FROM balance_ledger
		WHERE operation_id = $1 AND card_number = $2 AND direction = 'reversal'
	`, arg.OperationID, arg.CardNumber).Scan(&amount, &delta, &sourceType, &sourceID, &note)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	storedSourceID := ""
	if sourceID != nil {
		storedSourceID = *sourceID
	}
	storedNote := ""
	if note != nil {
		storedNote = *note
	}
	if delta != arg.Delta || amount != absInt64(arg.Delta) || sourceType != arg.SourceType || storedSourceID != arg.SourceID || storedNote != arg.Note {
		return false, fmt.Errorf("%w: operation_id=%s", ErrLedgerOperationConflict, arg.OperationID)
	}
	return true, nil
}

func absInt64(value int64) int64 {
	if value < 0 {
		return -value
	}
	return value
}

// ApplyLedgerAdjustment changes the current balance only through a new
// immutable reversal entry. delta > 0 credits; delta < 0 debits.
func (q *Queries) ApplyLedgerAdjustment(ctx context.Context, arg LedgerAdjustmentParams) (*SaldoAdjustmentRow, error) {
	if arg.Delta == 0 {
		return nil, errors.New("ledger adjustment delta must not be zero")
	}
	const query = `
		WITH candidate AS MATERIALIZED (
			SELECT saldo_id, card_number, total_balance
			FROM saldos
			WHERE card_number = $1 AND deleted_at IS NULL
			FOR UPDATE
		), inserted AS (
			INSERT INTO balance_ledger (
				operation_id, card_number, direction, amount, delta,
				balance_before, balance_after, source_type, source_id, note
			)
			SELECT $3, card_number, 'reversal', ABS($2::bigint), $2::bigint,
			       total_balance, total_balance + $2::bigint, $4, NULLIF($5, ''), NULLIF($6, '')
			FROM candidate
			WHERE total_balance + $2::bigint >= 0
			ON CONFLICT (operation_id, card_number, direction) DO NOTHING
			RETURNING card_number
		)
		UPDATE saldos s
		SET total_balance = s.total_balance + $2::bigint,
		    updated_at = current_timestamp
		FROM inserted i
		WHERE s.card_number = i.card_number AND s.deleted_at IS NULL
		RETURNING s.saldo_id, s.card_number, s.total_balance
	`
	row := q.db.QueryRow(ctx, query, arg.CardNumber, arg.Delta, arg.OperationID, arg.SourceType, arg.SourceID, arg.Note)
	var result SaldoAdjustmentRow
	err := row.Scan(&result.SaldoID, &result.CardNumber, &result.TotalBalance)
	if err == pgx.ErrNoRows {
		if existing, lookupErr := q.existingLedgerAdjustment(ctx, arg); lookupErr != nil {
			return nil, lookupErr
		} else if existing {
			current, currentErr := q.GetSaldoByCardNumber(ctx, arg.CardNumber)
			if currentErr != nil {
				return nil, currentErr
			}
			return &SaldoAdjustmentRow{SaldoID: current.SaldoID, CardNumber: current.CardNumber, TotalBalance: current.TotalBalance}, nil
		}
	}
	return &result, err
}

// ListReconciliationMismatches returns only accounts whose current balance
// differs from the immutable ledger total.
func (q *Queries) ListReconciliationMismatches(ctx context.Context, limit int32) ([]*ReconciliationRow, error) {
	const query = `
		WITH ledger_totals AS (
			SELECT card_number, COALESCE(SUM(delta), 0) AS ledger_balance,
			       COUNT(*) AS ledger_entries
			FROM balance_ledger
			GROUP BY card_number
		)
		SELECT s.saldo_id, s.card_number, s.total_balance,
		       COALESCE(l.ledger_balance, 0),
		       s.total_balance - COALESCE(l.ledger_balance, 0),
		       COALESCE(l.ledger_entries, 0)
		FROM saldos s
		LEFT JOIN ledger_totals l ON l.card_number = s.card_number
		WHERE s.deleted_at IS NULL
		  AND s.total_balance <> COALESCE(l.ledger_balance, 0)
		ORDER BY ABS(s.total_balance - COALESCE(l.ledger_balance, 0)) DESC, s.saldo_id
		LIMIT $1
	`
	rows, err := q.db.Query(ctx, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []*ReconciliationRow
	for rows.Next() {
		var item ReconciliationRow
		if err := rows.Scan(&item.SaldoID, &item.CardNumber, &item.CurrentBalance, &item.LedgerBalance, &item.Difference, &item.LedgerEntries); err != nil {
			return nil, err
		}
		result = append(result, &item)
	}
	return result, rows.Err()
}

func (q *Queries) EnqueueReconciliationMismatch(ctx context.Context, item *ReconciliationRow) error {
	const query = `
		INSERT INTO reconciliation_queue (
			saldo_id, card_number, current_balance, ledger_balance,
			difference, ledger_entries
		) VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (saldo_id) WHERE status IN ('pending', 'investigating')
		DO UPDATE SET current_balance = EXCLUDED.current_balance,
		              ledger_balance = EXCLUDED.ledger_balance,
		              difference = EXCLUDED.difference,
		              ledger_entries = EXCLUDED.ledger_entries,
		              mismatch_count = reconciliation_queue.mismatch_count + 1,
		              last_seen_at = current_timestamp
	`
	_, err := q.db.Exec(ctx, query, item.SaldoID, item.CardNumber, item.CurrentBalance, item.LedgerBalance, item.Difference, item.LedgerEntries)
	return err
}

func (q *Queries) ListReconciliationQueue(ctx context.Context, status string, limit int32) ([]*ReconciliationQueueRow, error) {
	const query = `
		SELECT queue_id, saldo_id, card_number, current_balance, ledger_balance,
		       difference, ledger_entries, status, mismatch_count, first_seen_at,
		       last_seen_at, resolved_at, resolution_operation_id, resolution_note
		FROM reconciliation_queue
		WHERE ($1 = '' OR status = $1)
		ORDER BY CASE status WHEN 'pending' THEN 0 WHEN 'investigating' THEN 1 ELSE 2 END,
		         last_seen_at DESC, queue_id
		LIMIT $2
	`
	rows, err := q.db.Query(ctx, query, status, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []*ReconciliationQueueRow
	for rows.Next() {
		var item ReconciliationQueueRow
		if err := rows.Scan(&item.QueueID, &item.SaldoID, &item.CardNumber, &item.CurrentBalance, &item.LedgerBalance, &item.Difference, &item.LedgerEntries, &item.Status, &item.MismatchCount, &item.FirstSeenAt, &item.LastSeenAt, &item.ResolvedAt, &item.ResolutionOperationID, &item.ResolutionNote); err != nil {
			return nil, err
		}
		result = append(result, &item)
	}
	return result, rows.Err()
}

func (q *Queries) ResolveReconciliation(ctx context.Context, queueID int64, operationID, note string) error {
	const query = `
		UPDATE reconciliation_queue q
		SET status = 'resolved', resolved_at = current_timestamp,
		    resolution_operation_id = $2, resolution_note = NULLIF($3, '')
		WHERE q.queue_id = $1
		  AND q.status IN ('pending', 'investigating')
		  AND EXISTS (
			SELECT 1 FROM saldos s
			WHERE s.saldo_id = q.saldo_id AND s.deleted_at IS NULL
			  AND s.total_balance = (
				SELECT COALESCE(SUM(delta), 0) FROM balance_ledger
				WHERE card_number = q.card_number
			  )
		  )
	`
	result, err := q.db.Exec(ctx, query, queueID, operationID, note)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

// ListLedgerEntries returns immutable audit entries for one card.
func (q *Queries) ListLedgerEntries(ctx context.Context, cardNumber string, limit int32) ([]*LedgerEntry, error) {
	const query = `
		SELECT entry_id, operation_id, card_number, direction, amount, delta,
		       balance_before, balance_after, source_type, source_id, note, created_at
		FROM balance_ledger WHERE card_number = $1
		ORDER BY created_at, entry_id LIMIT $2
	`
	rows, err := q.db.Query(ctx, query, cardNumber, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []*LedgerEntry
	for rows.Next() {
		var item LedgerEntry
		if err := rows.Scan(&item.EntryID, &item.OperationID, &item.CardNumber, &item.Direction, &item.Amount, &item.Delta, &item.BalanceBefore, &item.BalanceAfter, &item.SourceType, &item.SourceID, &item.Note, &item.CreatedAt); err != nil {
			return nil, err
		}
		result = append(result, &item)
	}
	return result, rows.Err()
}
