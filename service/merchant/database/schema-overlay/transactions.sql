-- sqlc-only schema overlay (NOT applied by goose).
--
-- The merchant service runs dashboard/stats queries against the shared
-- payment-gateway database, reading the transactions table that belongs to the
-- transaction service. This overlay exists only so sqlc can type-check those
-- queries; it is never mounted as a migration.
CREATE TABLE "transactions" (
    "transaction_id" SERIAL PRIMARY KEY,
    "transaction_no" UUID NOT NULL DEFAULT gen_random_uuid (),
    "card_number" VARCHAR(16) NOT NULL,
    "amount" BIGINT NOT NULL,
    "payment_method" VARCHAR(50) NOT NULL,
    "merchant_id" INT NOT NULL,
    "transaction_time" TIMESTAMP NOT NULL,
    "status" VARCHAR(20) NOT NULL DEFAULT 'pending',
    "created_at" timestamp DEFAULT current_timestamp,
    "updated_at" timestamp DEFAULT current_timestamp,
    "deleted_at" TIMESTAMP DEFAULT NULL
);
