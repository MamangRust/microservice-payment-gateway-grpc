-- GetCards: Retrieves paginated list of active cards with search capability
-- Purpose: List all active cards for management UI
-- Parameters:
--   $1: search_term - Optional text to filter cards by number, type or provider (NULL for no filter)
--   $2: limit - Maximum number of records to return
--   $3: offset - Number of records to skip for pagination
-- Returns:
--   All card fields plus total_count of matching records
-- Business Logic:
--   - Excludes soft-deleted cards (deleted_at IS NULL)
--   - Supports partial text matching on card_number, card_type and card_provider fields (case-insensitive)
--   - Returns cards ordered by card_id
--   - Provides total_count for pagination calculations
-- name: GetCards :many
SELECT
    card_id,
    user_id,
    card_number,
    card_type,
    expire_date,
    cvv,
    card_provider,
    created_at,
    updated_at,
    COUNT(*) OVER () AS total_count
FROM cards
WHERE
    deleted_at IS NULL
    AND (
        $1::TEXT IS NULL
        OR card_number ILIKE '%' || $1 || '%'
        OR card_type ILIKE '%' || $1 || '%'
        OR card_provider ILIKE '%' || $1 || '%'
    )
ORDER BY card_id
LIMIT $2
OFFSET
    $3;

-- GetCardByID: Retrieves a single card by its ID
-- Purpose: Get detailed information about a specific card
-- Parameters:
--   $1: card_id - The ID of the card to retrieve
-- Returns:
--   All fields for the specified card
-- Business Logic:
--   - Only returns active cards (deleted_at IS NULL)
--   - Returns NULL if card is not found or has been soft-deleted
-- name: GetCardByID :one
SELECT
    card_id,
    user_id,
    card_number,
    card_type,
    expire_date,
    cvv,
    card_provider,
    created_at,
    updated_at
FROM cards
WHERE
    card_id = $1
    AND deleted_at IS NULL;

-- name: GetActiveCardsWithCount :many
SELECT
    card_id,
    user_id,
    card_number,
    card_type,
    expire_date,
    cvv,
    card_provider,
    created_at,
    updated_at,
    deleted_at,
    COUNT(*) OVER () AS total_count
FROM cards
WHERE
    deleted_at IS NULL
    AND (
        $1::TEXT IS NULL
        OR card_number ILIKE '%' || $1 || '%'
        OR card_type ILIKE '%' || $1 || '%'
        OR card_provider ILIKE '%' || $1 || '%'
    )
ORDER BY card_id
LIMIT $2
OFFSET
    $3;

-- GetTrashedCardsWithCount: Retrieves paginated list of soft-deleted cards with search capability
-- Purpose: List all trashed (soft-deleted) cards for recovery or audit purposes
-- Parameters:
--   $1: search_term - Optional text to filter cards by number, type or provider (NULL for no filter)
--   $2: limit - Maximum number of records to return
--   $3: offset - Number of records to skip for pagination
-- Returns:
--   All card fields plus total_count of matching records
-- Business Logic:
--   - Includes only soft-deleted cards (deleted_at IS NOT NULL)
--   - Supports partial text matching on card_number, card_type and card_provider fields (case-insensitive)
--   - Returns cards ordered by card_id
--   - Provides total_count for pagination calculations
-- name: GetTrashedCardsWithCount :many
SELECT
    card_id,
    user_id,
    card_number,
    card_type,
    expire_date,
    cvv,
    card_provider,
    created_at,
    updated_at,
    deleted_at,
    COUNT(*) OVER () AS total_count
FROM cards
WHERE
    deleted_at IS NOT NULL
    AND (
        $1::TEXT IS NULL
        OR card_number ILIKE '%' || $1 || '%'
        OR card_type ILIKE '%' || $1 || '%'
        OR card_provider ILIKE '%' || $1 || '%'
    )
ORDER BY card_id
LIMIT $2
OFFSET
    $3;

-- GetCardByUserID: Retrieves a single active card associated with a specific user
-- Purpose: Get the card information for a particular user
-- Parameters:
--   $1: user_id - The ID of the user whose card should be retrieved
-- Returns:
--   All fields for the user's card or NULL if no active card exists
-- Business Logic:
--   - Only returns active cards (deleted_at IS NULL)
--   - Returns at most one card (LIMIT 1) even if multiple cards exist for the user
--   - Useful for displaying a user's primary/default card
-- name: GetCardByUserID :one
SELECT
    card_id,
    user_id,
    card_number,
    card_type,
    expire_date,
    cvv,
    card_provider,
    created_at,
    updated_at
FROM cards
WHERE
    user_id = $1
    AND deleted_at IS NULL
LIMIT 1;

-- GetCardByCardNumber: Retrieves a single active card by its card number
-- Purpose: Lookup card information using the physical card number
-- Parameters:
--   $1: card_number - The exact card number to search for
-- Returns:
--   All fields for the matching card or NULL if not found or deleted
-- Business Logic:
--   - Only returns active cards (deleted_at IS NULL)
--   - Performs exact match on card_number field (case-sensitive)
--   - Useful for card verification during transactions
-- name: GetCardByCardNumber :one
SELECT
    card_id,
    user_id,
    card_number,
    card_type,
    expire_date,
    cvv,
    card_provider,
    created_at,
    updated_at
FROM cards
WHERE
    card_number = $1
    AND deleted_at IS NULL;

-- GetUserEmailByCardNumber: Retrieves an active card by number.
-- The card database owns cards, while user profile data lives in user_db.
-- Email is enriched by the card service through UserQueryService; do not join users here.
-- name: GetUserEmailByCardNumber :one
SELECT
    card_id,
    ''::TEXT AS email,
    user_id,
    card_number,
    card_type,
    expire_date,
    cvv,
    card_provider,
    created_at,
    updated_at
FROM cards
WHERE
    card_number = $1
    AND deleted_at IS NULL;

-- GetTrashedCardByID: Retrieves a single soft-deleted card by its ID
-- Purpose: View details of a specific trashed card for recovery or audit
-- Parameters:
--   $1: card_id - The ID of the card to retrieve
-- Returns:
--   All fields for the specified trashed card or NULL if not found or not deleted
-- Business Logic:
--   - Only returns soft-deleted cards (deleted_at IS NOT NULL)
--   - Useful for admin interfaces showing deleted items
--   - Can be used before restoring a deleted card
-- name: GetTrashedCardByID :one
SELECT * FROM cards WHERE card_id = $1 AND deleted_at IS NOT NULL;

-- GetTotalBalance: Calculates the sum of all active card balances
-- Purpose: Get the total balance across all active cards in the system
-- Returns:
--   Single column 'total_balance' containing the sum of all non-deleted card balances
-- Business Logic:
--   - Only includes balances from active saldos records (s.deleted_at IS NULL)
--   - Only includes balances from active cards (c.deleted_at IS NULL)
--   - Useful for financial dashboards and system health monitoring
--   - Returns NULL if no active balances exist
-- name: CreateCard :one
INSERT INTO
    cards (
        user_id,
        card_number,
        card_type,
        expire_date,
        cvv,
        card_provider,
        created_at,
        updated_at
    )
VALUES (
        $1,
        $2,
        $3,
        $4,
        $5,
        $6,
        current_timestamp,
        current_timestamp
    )
RETURNING
    card_id,
    user_id,
    card_number,
    card_type,
    expire_date,
    cvv,
    card_provider,
    created_at,
    updated_at;

-- UpdateCard: Updates an existing card's details
-- Purpose: Modify card attributes for a specific card
-- Parameters:
--   $1: card_id - Identifier of the card to update
--   $2: card_type - New card type
--   $3: expire_date - New expiration date
--   $4: cvv - New CVV
--   $5: card_provider - New card provider
-- Returns: Nothing
-- Business Logic:
--   - Automatically updates updated_at timestamp
--   - Only updates cards that are not soft-deleted
-- name: UpdateCard :one
UPDATE cards
SET
    card_type = $2,
    expire_date = $3,
    cvv = $4,
    card_provider = $5,
    updated_at = current_timestamp
WHERE
    card_id = $1
    AND deleted_at IS NULL
RETURNING
    card_id,
    user_id,
    card_number,
    card_type,
    expire_date,
    cvv,
    card_provider,
    created_at,
    updated_at;

-- TrashCard: Soft-deletes a card by marking deleted_at
-- Purpose: Temporarily remove a card without deleting it permanently
-- Parameters:
--   $1: card_id - Identifier of the card to be trashed
-- Returns: Nothing
-- Business Logic:
--   - Sets deleted_at to current timestamp
--   - Only affects cards not already trashed
-- name: TrashCard :one
UPDATE cards
SET
    deleted_at = current_timestamp
WHERE
    card_id = $1
    AND deleted_at IS NULL
RETURNING
    card_id,
    user_id,
    card_number,
    card_type,
    expire_date,
    cvv,
    card_provider,
    created_at,
    updated_at,
    deleted_at;

-- RestoreCard: Restores a previously trashed card
-- Purpose: Undo soft-delete of a card
-- Parameters:
--   $1: card_id - Identifier of the card to restore
-- Returns: Nothing
-- Business Logic:
--   - Sets deleted_at to NULL
--   - Only affects cards that are currently trashed
-- name: RestoreCard :one
UPDATE cards
SET
    deleted_at = NULL
WHERE
    card_id = $1
    AND deleted_at IS NOT NULL
RETURNING
    card_id,
    user_id,
    card_number,
    card_type,
    expire_date,
    cvv,
    card_provider,
    created_at,
    updated_at,
    deleted_at;

-- DeleteCardPermanently: Removes a trashed card from the database
-- Purpose: Permanently delete a card that has been soft-deleted
-- Parameters:
--   $1: card_id - Identifier of the card to permanently delete
-- Returns: Nothing
-- Business Logic:
--   - Only deletes cards that are currently trashed
-- name: DeleteCardPermanently :exec
DELETE FROM cards WHERE card_id = $1 AND deleted_at IS NOT NULL;

-- InsertAuthTransaction: Creates an idempotent pending card authorization.
-- name: InsertAuthTransaction :one
INSERT INTO card_auth_transactions (
    txn_id, card_number, merchant_id, amount, currency, mcc,
    pos_entry_mode, idempotency_key
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING auth_id, txn_id, card_number, merchant_id, amount, currency, mcc,
          pos_entry_mode, status, idempotency_key, risk_score, created_at, updated_at;

-- name: ApproveAuthTransaction :one
UPDATE card_auth_transactions
SET status = 'approved', updated_at = CURRENT_TIMESTAMP
WHERE txn_id = $1 AND status = 'pending'
RETURNING auth_id, txn_id, card_number, merchant_id, amount, currency, mcc,
          pos_entry_mode, status, idempotency_key, risk_score, created_at, updated_at;

-- name: DeclineAuthTransaction :one
UPDATE card_auth_transactions
SET status = 'declined', updated_at = CURRENT_TIMESTAMP
WHERE txn_id = $1 AND status = 'pending'
RETURNING auth_id, txn_id, card_number, merchant_id, amount, currency, mcc,
          pos_entry_mode, status, idempotency_key, risk_score, created_at, updated_at;

-- name: ReverseAuthTransaction :one
UPDATE card_auth_transactions
SET status = 'reversed', updated_at = CURRENT_TIMESTAMP
WHERE txn_id = $1 AND status = 'approved'
RETURNING auth_id, txn_id, card_number, merchant_id, amount, currency, mcc,
          pos_entry_mode, status, idempotency_key, risk_score, created_at, updated_at;

-- name: GetAuthTransactionByIdempotencyKey :one
SELECT auth_id, txn_id, card_number, merchant_id, amount, currency, mcc,
       pos_entry_mode, status, idempotency_key, risk_score, created_at, updated_at
FROM card_auth_transactions
WHERE idempotency_key = $1;

-- name: GetAuthTransactionByTxnId :one
SELECT auth_id, txn_id, card_number, merchant_id, amount, currency, mcc,
       pos_entry_mode, status, idempotency_key, risk_score, created_at, updated_at
FROM card_auth_transactions
WHERE txn_id = $1;

-- name: GetAuthTransactionsByCardNumber :many
SELECT auth_id, txn_id, card_number, merchant_id, amount, currency, mcc,
       pos_entry_mode, status, idempotency_key, risk_score, created_at, updated_at,
       COUNT(*) OVER () AS total_count
FROM card_auth_transactions
WHERE card_number = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: CountRecentAuthByCardNumber :one
SELECT COUNT(*)::INT AS total_count
FROM card_auth_transactions
WHERE card_number = $1 AND created_at >= $2;

-- name: UpdateAuthTransactionRiskScore :exec
UPDATE card_auth_transactions
SET risk_score = $2, updated_at = CURRENT_TIMESTAMP
WHERE txn_id = $1;

-- name: InsertCardPayment :one
INSERT INTO card_payments (
    payment_uuid, card_number, billing_id, amount, payment_channel, reference_id
)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING payment_id, payment_uuid, card_number, billing_id, amount,
          payment_channel, reference_id, status, created_at, updated_at;

-- name: GetCardPaymentsByCardNumber :many
SELECT payment_id, payment_uuid, card_number, billing_id, amount,
       payment_channel, reference_id, status, created_at, updated_at,
       COUNT(*) OVER () AS total_count
FROM card_payments
WHERE card_number = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: CountCardPayments :one
SELECT COUNT(*)::INT AS total_count
FROM card_payments
WHERE card_number = $1;

-- name: InsertCardReward :one
INSERT INTO card_rewards (
    card_number, txn_id, amount, mcc, points_earned, expires_at
)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING reward_id, card_number, txn_id, amount, mcc, points_earned,
          expires_at, redeemed, created_at;

-- name: GetCardRewardBalance :one
SELECT COALESCE(SUM(points_earned), 0)::BIGINT AS balance
FROM card_rewards
WHERE card_number = $1 AND redeemed = FALSE AND expires_at > CURRENT_TIMESTAMP;

-- name: GetCardRewardHistory :many
SELECT reward_id, card_number, txn_id, amount, mcc, points_earned,
       expires_at, redeemed, created_at
FROM card_rewards
WHERE card_number = $1
ORDER BY created_at DESC;

-- name: GetRedeemableRewardIds :many
SELECT reward_id, points_earned
FROM card_rewards
WHERE card_number = $1 AND redeemed = FALSE AND expires_at > CURRENT_TIMESTAMP
ORDER BY created_at;

-- name: MarkRewardsRedeemed :one
WITH updated AS (
    UPDATE card_rewards
    SET redeemed = TRUE
    WHERE reward_id = ANY($1::INT[])
    RETURNING points_earned
)
SELECT COALESCE(SUM(points_earned), 0)::BIGINT AS points
FROM updated;

-- name: GetBillingCyclesByCardNumber :many
SELECT billing_id, card_number, cycle_start, cycle_end, amount_due, due_date,
       status, created_at, updated_at
FROM billing_cycles
WHERE card_number = $1
ORDER BY cycle_start DESC;

-- name: InsertBillingCycle :one
INSERT INTO billing_cycles (
    card_number, cycle_start, cycle_end, amount_due, due_date, status
)
VALUES ($1, $2, $3, $4, $5, 'unpaid')
RETURNING billing_id, card_number, cycle_start, cycle_end, amount_due, due_date,
          status, created_at, updated_at;

-- CreateBillingCycles creates one statement per active card for the period.
-- The unique period index makes retries and concurrent schedulers safe.
-- name: CreateBillingCycles :many
INSERT INTO billing_cycles (
    card_number, cycle_start, cycle_end, amount_due, due_date, status
)
SELECT card_number, $1, $2, outstanding_balance, $3, 'unpaid'
FROM cards
WHERE deleted_at IS NULL
ON CONFLICT (card_number, cycle_start) DO NOTHING
RETURNING billing_id, card_number, cycle_start, cycle_end, amount_due, due_date,
          status, created_at, updated_at;

-- name: UpdateBillingCycleStatus :one
UPDATE billing_cycles
SET status = $2, updated_at = CURRENT_TIMESTAMP
WHERE billing_id = $1
RETURNING billing_id, card_number, cycle_start, cycle_end, amount_due, due_date,
          status, created_at, updated_at;

-- name: ToggleCardStatus :one
UPDATE cards
SET status = CASE WHEN status = 'active' THEN 'suspended' ELSE 'active' END,
    updated_at = CURRENT_TIMESTAMP
WHERE card_id = $1 AND deleted_at IS NULL
RETURNING card_id, user_id, card_number, card_type, expire_date, cvv, card_provider,
          status, credit_limit, outstanding_balance, reward_points, created_at, updated_at;

-- name: UpdateCardStatus :one
UPDATE cards
SET status = $2, updated_at = CURRENT_TIMESTAMP
WHERE card_id = $1 AND deleted_at IS NULL
RETURNING card_id, user_id, card_number, card_type, expire_date, cvv, card_provider,
          status, credit_limit, outstanding_balance, reward_points, created_at, updated_at;

-- name: UpdateCreditLimit :one
UPDATE cards
SET credit_limit = $2, updated_at = CURRENT_TIMESTAMP
WHERE card_id = $1 AND deleted_at IS NULL
RETURNING card_id, user_id, card_number, card_type, expire_date, cvv, card_provider,
          status, credit_limit, outstanding_balance, reward_points, created_at, updated_at;

-- name: RedeemRewardPoints :one
UPDATE cards
SET reward_points = GREATEST(0, reward_points - $2), updated_at = CURRENT_TIMESTAMP
WHERE card_id = $1 AND deleted_at IS NULL AND reward_points >= $2
RETURNING card_id, user_id, card_number, card_type, expire_date, cvv, card_provider,
          status, credit_limit, outstanding_balance, reward_points, created_at, updated_at;

-- RestoreAllCards: Restores all trashed cards
-- Purpose: Bulk-restore all soft-deleted cards
-- Parameters: None
-- Returns: Nothing
-- Business Logic:
--   - Sets deleted_at to NULL for all trashed cards
-- name: RestoreAllCards :exec
UPDATE cards
SET
    deleted_at = NULL
WHERE
    deleted_at IS NOT NULL;

-- DeleteAllPermanentCards: Permanently deletes all trashed cards
-- Purpose: Bulk-delete all cards that have been soft-deleted
-- Parameters: None
-- Returns: Nothing
-- Business Logic:
--   - Deletes only cards with deleted_at set
-- name: DeleteAllPermanentCards :exec
DELETE FROM cards WHERE deleted_at IS NOT NULL;