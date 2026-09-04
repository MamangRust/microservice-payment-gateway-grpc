package outbox

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// OutboxRecord represents a row in the outbox_events table. It is defined here
// (schema-agnostic) so the shared store/publisher work against any service that
// owns the identical outbox_events table.
type OutboxRecord struct {
	ID            int64      `json:"id"`
	AggregateType string     `json:"aggregate_type"`
	AggregateID   string     `json:"aggregate_id"`
	EventID       string     `json:"event_id"`
	EventType     string     `json:"event_type"`
	EventVersion  int32      `json:"event_version"`
	Payload       []byte     `json:"payload"`
	Status        string     `json:"status"`
	Attempts      int32      `json:"attempts"`
	LastError     *string    `json:"last_error"`
	AvailableAt   time.Time  `json:"available_at"`
	CreatedAt     time.Time  `json:"created_at"`
	PublishedAt   *time.Time `json:"published_at"`
}

const (
	OutboxStatusPending   = "pending"
	OutboxStatusPublished = "published"
	OutboxStatusFailed    = "failed"
)

// DBTX is the minimal executor surface the outbox SQL needs. Both
// *pgxpool.Pool and pgx.Tx satisfy it, so the store/publisher can run against
// a pool or an explicit transaction.
type DBTX interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// InsertOutbox inserts a single event into the outbox. The insert runs outside
// any transaction — it is the caller's responsibility to ensure the outbox row
// is inserted atomically within the same DB transaction as the business
// mutation.
func InsertOutbox(ctx context.Context, db DBTX, record OutboxRecord) error {
	if record.EventID == "" {
		record.EventID = uuid.New().String()
	}
	if record.EventVersion == 0 {
		record.EventVersion = 1
	}
	if record.Status == "" {
		record.Status = OutboxStatusPending
	}
	if record.AvailableAt.IsZero() {
		record.AvailableAt = time.Now()
	}

	const query = `
		INSERT INTO outbox_events (
			aggregate_type, aggregate_id, event_id, event_type,
			event_version, payload, status, attempts,
			last_error, available_at, created_at
		) VALUES (
			$1, $2, $3, $4,
			$5, $6, $7, 0,
			NULL, $8, now()
		)
		RETURNING id, created_at
	`

	return db.QueryRow(ctx, query,
		record.AggregateType,
		record.AggregateID,
		record.EventID,
		record.EventType,
		record.EventVersion,
		record.Payload,
		record.Status,
		record.AvailableAt,
	).Scan(&record.ID, &record.CreatedAt)
}

// ClaimPendingOutbox locks up to maxRows pending/retriable outbox rows
// and returns them for publishing. Uses FOR UPDATE SKIP LOCKED so
// multiple publisher instances can safely poll concurrently.
// Stale 'processing' rows (crashed publisher) are also reclaimed after
// a grace period.
func ClaimPendingOutbox(ctx context.Context, db DBTX, maxRows int32, maxAttempts int32) ([]*OutboxRecord, error) {
	const query = `
		UPDATE outbox_events
		SET status = 'processing',
			attempts = attempts + 1,
			available_at = now() + interval '5 minutes'
		WHERE id IN (
			SELECT id FROM outbox_events
			WHERE (
			    status IN ('pending', 'failed')
			    OR (status = 'processing' AND available_at <= now() - interval '10 minutes')
			)
			  AND attempts < $1
			  AND available_at <= now()
			ORDER BY available_at
			LIMIT $2
			FOR UPDATE SKIP LOCKED
		)
		RETURNING id, aggregate_type, aggregate_id, event_id,
			event_type, event_version, payload, status, attempts,
			last_error, available_at, created_at, published_at
	`

	rows, err := db.Query(ctx, query, maxAttempts, maxRows)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []*OutboxRecord
	for rows.Next() {
		var r OutboxRecord
		var le *string
		var pa *time.Time
		err := rows.Scan(
			&r.ID, &r.AggregateType, &r.AggregateID, &r.EventID,
			&r.EventType, &r.EventVersion, &r.Payload, &r.Status,
			&r.Attempts, &le, &r.AvailableAt, &r.CreatedAt, &pa,
		)
		if err != nil {
			return nil, err
		}
		r.LastError = le
		r.PublishedAt = pa
		records = append(records, &r)
	}
	return records, rows.Err()
}

// CompleteOutbox marks an outbox event as successfully published.
func CompleteOutbox(ctx context.Context, db DBTX, eventID string) error {
	const query = `
		UPDATE outbox_events
		SET status = $1, published_at = now()
		WHERE event_id = $2
	`
	_, err := db.Exec(ctx, query, OutboxStatusPublished, eventID)
	return err
}

// FailOutbox marks an outbox event as failed with error details.
func FailOutbox(ctx context.Context, db DBTX, eventID string, errMsg string) error {
	const query = `
		UPDATE outbox_events
		SET status = $1, last_error = $2, available_at = now() + interval '30 seconds'
		WHERE event_id = $3
	`
	_, err := db.Exec(ctx, query, OutboxStatusFailed, errMsg, eventID)
	return err
}

// CountPendingOutbox returns the number of events still awaiting delivery
// (pending or retriable failed). Used to expose outbox lag as a gauge.
func CountPendingOutbox(ctx context.Context, db DBTX) (int64, error) {
	const query = `
		SELECT COUNT(*)
		FROM outbox_events
		WHERE status IN ('pending', 'failed')
	`
	var count int64
	err := db.QueryRow(ctx, query).Scan(&count)
	return count, err
}
