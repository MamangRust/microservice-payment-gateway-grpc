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

-- name: GetUserEmailByCardNumber :one
SELECT c.card_id, u.email, c.user_id, c.card_number, c.card_type, c.expire_date, c.cvv, c.card_provider, c.created_at, c.updated_at
FROM cards c
    JOIN users u ON u.user_id = c.user_id
WHERE
    c.card_number = $1
    AND c.deleted_at IS NULL
    AND u.deleted_at IS NULL;

-- GetActiveCardsWithCount: Retrieves paginated list of active cards with search capability
-- Purpose: List all active cards for management UI (alternative to GetCards with same functionality)
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
-- name: GetTotalBalance :one
SELECT SUM(s.total_balance) AS total_balance
FROM saldos s
    JOIN cards c ON s.card_number = c.card_number
WHERE
    s.deleted_at IS NULL
    AND c.deleted_at IS NULL;

-- GetTotalTopupAmount: Calculates the sum of all top-up transactions
-- Purpose: Get the total amount ever topped up across all active cards
-- Returns:
--   Single column 'total_topup_amount' containing sum of all non-deleted topups
-- Business Logic:
--   - Only includes amounts from active topups (t.deleted_at IS NULL)
--   - Only includes amounts from active cards (c.deleted_at IS NULL)
--   - Useful for financial reporting and reconciliation
--   - Returns NULL if no topups exist
-- name: GetTotalTopupAmount :one
SELECT SUM(t.topup_amount) AS total_topup_amount
FROM topups t
    JOIN cards c ON t.card_number = c.card_number
WHERE
    t.deleted_at IS NULL
    AND c.deleted_at IS NULL;

-- GetTotalWithdrawAmount: Calculates the sum of all withdrawal transactions
-- Purpose: Get the total amount ever withdrawn from all active cards
-- Returns:
--   Single column 'total_withdraw_amount' containing sum of all non-deleted withdrawals
-- Business Logic:
--   - Only includes amounts from active withdrawals (s.deleted_at IS NULL)
--   - Only includes amounts from active cards (c.deleted_at IS NULL)
--   - Useful for cash flow analysis and auditing
--   - Returns NULL if no withdrawals exist
-- name: GetTotalWithdrawAmount :one
SELECT SUM(s.withdraw_amount) AS total_withdraw_amount
FROM withdraws s
    JOIN cards c ON s.card_number = c.card_number
WHERE
    s.deleted_at IS NULL
    AND c.deleted_at IS NULL;

-- GetTotalTransactionAmount: Calculates the sum of all payment transactions
-- Purpose: Get the total amount processed through all card transactions
-- Returns:
--   Single column 'total_transaction_amount' containing sum of all non-deleted transactions
-- Business Logic:
--   - Only includes amounts from active transactions (t.deleted_at IS NULL)
--   - Only includes amounts from active cards (c.deleted_at IS NULL)
--   - Useful for sales reporting and revenue analysis
--   - Returns NULL if no transactions exist
-- name: GetTotalTransactionAmount :one
SELECT SUM(t.amount) AS total_transaction_amount
FROM transactions t
    JOIN cards c ON t.card_number = c.card_number
WHERE
    t.deleted_at IS NULL
    AND c.deleted_at IS NULL;

-- GetTotalTransferAmount: Calculates the sum of all transfer transactions
-- Purpose: Get the total amount transferred between accounts
-- Returns:
--   Single column 'total_transfer_amount' containing sum of all non-deleted transfers
-- Business Logic:
--   - Includes amounts from both sides of transfers (using UNION ALL)
--   - Only includes active transfer records (deleted_at IS NULL)
--   - Counts both outgoing and incoming transfers in the total
--   - Useful for monitoring money movement in the system
--   - Returns NULL if no transfers exist
-- Note: The current implementation appears to double-count transfers by including
--       the same table twice in the UNION ALL. This may need review.
-- name: GetTotalTransferAmount :one
SELECT SUM(transfer_amount) AS total_transfer_amount
FROM (
        SELECT transfer_amount
        FROM transfers
        WHERE
            deleted_at IS NULL
        UNION ALL
        SELECT transfer_amount
        FROM transfers
        WHERE
            deleted_at IS NULL
    ) AS transfer_data;

-- GetTotalBalanceByCardNumber: Calculates the total balance for a specific card
-- Purpose: Get the current balance of a particular active card
-- Parameters:
--   $1: card_number - The card number to query balance for
-- Returns:
--   Single column 'total_balance' containing the sum balance for the specified card
-- Business Logic:
--   - Only includes balance from active saldos records (s.deleted_at IS NULL)
--   - Only includes balance if card is active (c.deleted_at IS NULL)
--   - Returns NULL if card doesn't exist or has been deleted
--   - Useful for displaying individual card balances
-- name: GetTotalBalanceByCardNumber :one
SELECT SUM(s.total_balance) AS total_balance
FROM saldos s
    JOIN cards c ON s.card_number = c.card_number
WHERE
    s.deleted_at IS NULL
    AND c.deleted_at IS NULL
    AND c.card_number = $1;

-- GetTotalTopupAmountByCardNumber: Calculates total top-ups for a specific card
-- Purpose: Get the lifetime top-up amount for a particular card
-- Parameters:
--   $1: card_number - The card number to query top-ups for
-- Returns:
--   Single column 'total_topup_amount' containing sum of all top-ups
-- Business Logic:
--   - Only includes active topup records (t.deleted_at IS NULL)
--   - Only includes amounts when card is active (c.deleted_at IS NULL)
--   - Useful for card activity analysis and user statements
-- name: GetTotalTopupAmountByCardNumber :one
SELECT SUM(t.topup_amount) AS total_topup_amount
FROM topups t
    JOIN cards c ON t.card_number = c.card_number
WHERE
    t.deleted_at IS NULL
    AND c.deleted_at IS NULL
    AND c.card_number = $1;

-- GetTotalWithdrawAmountByCardNumber: Calculates total withdrawals for a card
-- Purpose: Get the lifetime withdrawal amount for a specific card
-- Parameters:
--   $1: card_number - The card number to query withdrawals for
-- Returns:
--   Single column 'total_withdraw_amount' containing sum of all withdrawals
-- Business Logic:
--   - Only includes active withdrawal records (s.deleted_at IS NULL)
--   - Only includes amounts when card is active (c.deleted_at IS NULL)
--   - Useful for cash flow analysis per card
-- Note: Verify table name consistency (saldos vs withdraws)
-- name: GetTotalWithdrawAmountByCardNumber :one
SELECT SUM(s.withdraw_amount) AS total_withdraw_amount
FROM saldos s
    JOIN cards c ON s.card_number = c.card_number
WHERE
    s.deleted_at IS NULL
    AND c.deleted_at IS NULL
    AND c.card_number = $1;

-- GetTotalTransactionAmountByCardNumber: Calculates total transactions for a card
-- Purpose: Get the lifetime transaction amount for a specific card
-- Parameters:
--   $1: card_number - The card number to query transactions for
-- Returns:
--   Single column 'total_transaction_amount' containing sum of all transactions
-- Business Logic:
--   - Only includes active transaction records (t.deleted_at IS NULL)
--   - Only includes amounts when card is active (c.deleted_at IS NULL)
--   - Useful for spending analysis and card statements
-- name: GetTotalTransactionAmountByCardNumber :one
SELECT SUM(t.amount) AS total_transaction_amount
FROM transactions t
    JOIN cards c ON t.card_number = c.card_number
WHERE
    t.deleted_at IS NULL
    AND c.deleted_at IS NULL
    AND c.card_number = $1;

-- GetTotalTransferAmountBySender: Calculates total outgoing transfers from an account
-- Purpose: Get the total amount sent from a specific card/account
-- Parameters:
--   $1: transfer_from - The account/card number that initiated transfers
-- Returns:
--   Single column 'total_transfer_amount' containing sum of all outgoing transfers
-- Business Logic:
--   - Only includes active transfer records (deleted_at IS NULL)
--   - Useful for tracking money sent by a particular account
-- name: GetTotalTransferAmountBySender :one
SELECT SUM(transfer_amount) AS total_transfer_amount
FROM transfers
WHERE
    transfer_from = $1
    AND deleted_at IS NULL;

-- GetTotalTransferAmountByReceiver: Calculates total incoming transfers to an account
-- Purpose: Get the total amount received by a specific card/account
-- Parameters:
--   $1: transfer_to - The account/card number that received transfers
-- Returns:
--   Single column 'total_transfer_amount' containing sum of all incoming transfers
-- Business Logic:
--   - Only includes active transfer records (deleted_at IS NULL)
--   - Useful for tracking money received by a particular account
-- name: GetTotalTransferAmountByReceiver :one
SELECT SUM(transfer_amount) AS total_transfer_amount
FROM transfers
WHERE
    transfer_to = $1
    AND deleted_at IS NULL;

-- GetMonthlyBalances: Retrieves monthly balance totals for a given year
-- Purpose: Provide monthly balance trends for dashboard visualizations
-- Parameters:
--   $1: reference_date - A date used to determine the year to analyze
-- Returns:
--   month: 3-letter month abbreviation (e.g., 'Jan')
--   total_balance: Sum of balances for that month (zero if no data)
-- Business Logic:
--   - Generates a complete 12-month series for the year
--   - Includes only active saldos and cards (deleted_at IS NULL)
--   - Uses LEFT JOIN to ensure all months appear in results
--   - COALESCE returns 0 for months with no data
--   - Results ordered chronologically
-- name: GetMonthlyBalances :many
WITH
    months AS (
        SELECT generate_series(
                date_trunc('year', $1::timestamp), date_trunc('year', $1::timestamp) + interval '1 year' - interval '1 day', interval '1 month'
            ) AS month
    )
SELECT TO_CHAR(m.month, 'Mon') AS month, COALESCE(SUM(s.total_balance), 0)::int AS total_balance
FROM
    months m
    LEFT JOIN saldos s ON EXTRACT(
        MONTH
        FROM s.created_at
    ) = EXTRACT(
        MONTH
        FROM m.month
    )
    AND EXTRACT(
        YEAR
        FROM s.created_at
    ) = EXTRACT(
        YEAR
        FROM m.month
    )
    AND s.deleted_at IS NULL
    LEFT JOIN cards c ON s.card_number = c.card_number
    AND c.deleted_at IS NULL
GROUP BY
    m.month
ORDER BY m.month;

-- GetYearlyBalances: Retrieves yearly balance totals for last 5 years
-- Purpose: Provide annual balance trends for financial reporting
-- Parameters:
--   $1: reference_year - The target year (includes this year plus previous 4)
-- Returns:
--   year: The 4-digit year
--   total_balance: Sum of balances for that year
-- Business Logic:
--   - Covers a 5-year rolling window (reference_year-4 to reference_year)
--   - Only includes active saldos and cards
--   - Groups by calendar year
--   - Results ordered chronologically
-- name: GetYearlyBalances :many
WITH
    last_five_years AS (
        SELECT EXTRACT(
                YEAR
                FROM s.created_at
            ) AS year, SUM(s.total_balance) AS total_balance
        FROM saldos s
            JOIN cards c ON s.card_number = c.card_number
        WHERE
            s.deleted_at IS NULL
            AND c.deleted_at IS NULL
            AND EXTRACT(
                YEAR
                FROM s.created_at
            ) >= $1 - 4
            AND EXTRACT(
                YEAR
                FROM s.created_at
            ) <= $1
        GROUP BY
            EXTRACT(
                YEAR
                FROM s.created_at
            )
    )
SELECT year, total_balance
FROM last_five_years
ORDER BY year;

-- GetMonthlyTopupAmount: Retrieves monthly top-up totals for a given year
-- Purpose: Analyze monthly top-up patterns and trends
-- Parameters:
--   $1: reference_date - A date used to determine the year to analyze
-- Returns:
--   month: 3-letter month abbreviation
--   total_topup_amount: Sum of top-ups for that month (zero if no data)
-- Business Logic:
--   - Generates complete 12-month series
--   - Only includes active topups and cards
--   - Uses LEFT JOIN to ensure all months appear
--   - COALESCE returns 0 for months with no activity
--   - Results ordered chronologically
-- name: GetMonthlyTopupAmount :many
WITH
    months AS (
        SELECT generate_series(
                date_trunc('year', $1::timestamp), date_trunc('year', $1::timestamp) + interval '1 year' - interval '1 day', interval '1 month'
            ) AS month
    )
SELECT TO_CHAR(m.month, 'Mon') AS month, COALESCE(SUM(t.topup_amount), 0)::int AS total_topup_amount
FROM
    months m
    LEFT JOIN topups t ON EXTRACT(
        MONTH
        FROM t.topup_time
    ) = EXTRACT(
        MONTH
        FROM m.month
    )
    AND EXTRACT(
        YEAR
        FROM t.topup_time
    ) = EXTRACT(
        YEAR
        FROM m.month
    )
    AND t.deleted_at IS NULL
    LEFT JOIN cards c ON t.card_number = c.card_number
    AND c.deleted_at IS NULL
GROUP BY
    m.month
ORDER BY m.month;

-- GetYearlyTopupAmount: Retrieves yearly top-up totals for last 5 years
-- Purpose: Analyze long-term top-up trends and growth
-- Parameters:
--   $1: reference_year - The target year (includes this year plus previous 4)
-- Returns:
--   year: The 4-digit year
--   total_topup_amount: Sum of top-ups for that year
-- Business Logic:
--   - Covers a 5-year rolling window
--   - Only includes active topups and cards
--   - Groups by calendar year
--   - Results ordered chronologically
--   - Useful for identifying annual growth patterns
-- name: GetYearlyTopupAmount :many
WITH
    last_five_years AS (
        SELECT EXTRACT(
                YEAR
                FROM t.topup_time
            ) AS year, SUM(t.topup_amount) AS total_topup_amount
        FROM topups t
            JOIN cards c ON t.card_number = c.card_number
        WHERE
            t.deleted_at IS NULL
            AND c.deleted_at IS NULL
            AND EXTRACT(
                YEAR
                FROM t.topup_time
            ) >= $1 - 4
            AND EXTRACT(
                YEAR
                FROM t.topup_time
            ) <= $1
        GROUP BY
            EXTRACT(
                YEAR
                FROM t.topup_time
            )
    )
SELECT year, total_topup_amount
FROM last_five_years
ORDER BY year;

-- GetMonthlyWithdrawAmount: Retrieves monthly withdraw totals for a given year
-- Purpose: Analyze monthly withdraw patterns and trends
-- Parameters:
--   $1: reference_date - A date used to determine the year to analyze
-- Returns:
--   month: 3-letter month abbreviation
--   total_withdraw_amount: Sum of withdraws for that month (zero if no data)
-- Business Logic:
--   - Generates complete 12-month series
--   - Only includes active withdraws and cards
--   - Uses LEFT JOIN to ensure all months appear
--   - COALESCE returns 0 for months with no activity
--   - Results ordered chronologically
-- name: GetMonthlyWithdrawAmount :many
WITH
    months AS (
        SELECT generate_series(
                date_trunc('year', $1::timestamp), date_trunc('year', $1::timestamp) + interval '1 year' - interval '1 day', interval '1 month'
            ) AS month
    )
SELECT TO_CHAR(m.month, 'Mon') AS month, COALESCE(SUM(w.withdraw_amount), 0)::int AS total_withdraw_amount
FROM
    months m
    LEFT JOIN withdraws w ON EXTRACT(
        MONTH
        FROM w.withdraw_time
    ) = EXTRACT(
        MONTH
        FROM m.month
    )
    AND EXTRACT(
        YEAR
        FROM w.withdraw_time
    ) = EXTRACT(
        YEAR
        FROM m.month
    )
    AND w.deleted_at IS NULL
    LEFT JOIN cards c ON w.card_number = c.card_number
    AND c.deleted_at IS NULL
GROUP BY
    m.month
ORDER BY m.month;

-- GetYearlyWithdrawAmount: Retrieves yearly withdraw totals for last 5 years
-- Purpose: Analyze long-term withdraw trends and growth
-- Parameters:
--   $1: reference_year - The target year (includes this year plus previous 4)
-- Returns:
--   year: The 4-digit year
--   total_withdraw_amount: Sum of withdraws for that year
-- Business Logic:
--   - Covers a 5-year rolling window
--   - Only includes active withdraws and cards
--   - Groups by calendar year
--   - Results ordered chronologically
--   - Useful for identifying annual growth patterns
-- name: GetYearlyWithdrawAmount :many
WITH
    last_five_years AS (
        SELECT EXTRACT(
                YEAR
                FROM w.withdraw_time
            ) AS year, SUM(w.withdraw_amount) AS total_withdraw_amount
        FROM withdraws w
            JOIN cards c ON w.card_number = c.card_number
        WHERE
            w.deleted_at IS NULL
            AND c.deleted_at IS NULL
            AND EXTRACT(
                YEAR
                FROM w.withdraw_time
            ) >= $1 - 4
            AND EXTRACT(
                YEAR
                FROM w.withdraw_time
            ) <= $1
        GROUP BY
            EXTRACT(
                YEAR
                FROM w.withdraw_time
            )
    )
SELECT year, total_withdraw_amount
FROM last_five_years
ORDER BY year;

-- GetMonthlyTransactionAmount: Retrieves monthly transaction totals for a given year
-- Purpose: Analyze monthly transaction patterns and trends
-- Parameters:
--   $1: reference_date - A date used to determine the year to analyze
-- Returns:
--   month: 3-letter month abbreviation
--   total_transaction_amount: Sum of transactions for that month (zero if no data)
-- Business Logic:
--   - Generates complete 12-month series
--   - Only includes active transactions and cards
--   - Uses LEFT JOIN to ensure all months appear
--   - COALESCE returns 0 for months with no activity
--   - Results ordered chronologically
-- name: GetMonthlyTransactionAmount :many
WITH
    months AS (
        SELECT generate_series(
                date_trunc('year', $1::timestamp), date_trunc('year', $1::timestamp) + interval '1 year' - interval '1 day', interval '1 month'
            ) AS month
    )
SELECT
    TO_CHAR(m.month, 'Mon') AS month,
    COALESCE(SUM(t.amount), 0)::int AS total_transaction_amount
FROM
    months m
    LEFT JOIN transactions t ON EXTRACT(
        MONTH
        FROM t.transaction_time
    ) = EXTRACT(
        MONTH
        FROM m.month
    )
    AND EXTRACT(
        YEAR
        FROM t.transaction_time
    ) = EXTRACT(
        YEAR
        FROM m.month
    )
    AND t.deleted_at IS NULL
    LEFT JOIN cards c ON t.card_number = c.card_number
    AND c.deleted_at IS NULL
GROUP BY
    m.month
ORDER BY m.month;

-- GetYearlyTransactionAmount: Retrieves yearly transaction totals for last 5 years
-- Purpose: Analyze long-term transaction trends and growth
-- Parameters:
--   $1: reference_year - The target year (includes this year plus previous 4)
-- Returns:
--   year: The 4-digit year
--   total_transaction_amount: Sum of transactions for that year
-- Business Logic:
--   - Covers a 5-year rolling window
--   - Only includes active transactions and cards
--   - Groups by calendar year
--   - Results ordered chronologically
--   - Useful for identifying annual growth patterns
-- name: GetYearlyTransactionAmount :many
WITH
    last_five_years AS (
        SELECT EXTRACT(
                YEAR
                FROM t.transaction_time
            ) AS year, SUM(t.amount) AS total_transaction_amount
        FROM transactions t
            JOIN cards c ON t.card_number = c.card_number
        WHERE
            t.deleted_at IS NULL
            AND c.deleted_at IS NULL
            AND EXTRACT(
                YEAR
                FROM t.transaction_time
            ) >= $1 - 4
            AND EXTRACT(
                YEAR
                FROM t.transaction_time
            ) <= $1
        GROUP BY
            EXTRACT(
                YEAR
                FROM t.transaction_time
            )
    )
SELECT year, total_transaction_amount
FROM last_five_years
ORDER BY year;

-- GetMonthlyTransferAmountSender: Retrieves monthly transfer totals for a given year
-- Purpose: Analyze monthly transfer patterns and trends
-- Parameters:
--   $1: reference_date - A date used to determine the year to analyze
-- Returns:
--   month: 3-letter month abbreviation
--   total_sent_amount: Sum of transfers for that month (zero if no data)
-- Business Logic:
--   - Generates complete 12-month series
--   - Only includes active transfers and cards
--   - Uses LEFT JOIN to ensure all months appear
--   - COALESCE returns 0 for months with no activity
--   - Results ordered chronologically
-- name: GetMonthlyTransferAmountSender :many
WITH
    months AS (
        SELECT generate_series(
                date_trunc('year', $1::timestamp), date_trunc('year', $1::timestamp) + interval '1 year' - interval '1 day', interval '1 month'
            ) AS month
    )
SELECT TO_CHAR(m.month, 'Mon') AS month, COALESCE(SUM(t.transfer_amount), 0)::int AS total_sent_amount
FROM months m
    LEFT JOIN transfers t ON EXTRACT(
        MONTH
        FROM t.transfer_time
    ) = EXTRACT(
        MONTH
        FROM m.month
    )
    AND EXTRACT(
        YEAR
        FROM t.transfer_time
    ) = EXTRACT(
        YEAR
        FROM m.month
    )
    AND t.deleted_at IS NULL
GROUP BY
    m.month
ORDER BY m.month;

-- GetMonthlyTransferAmountReceiver: Retrieves monthly transfer totals for a given year
-- Purpose: Analyze monthly transfer patterns and trends
-- Parameters:
--   $1: reference_date - A date used to determine the year to analyze
-- Returns:
--   month: 3-letter month abbreviation
--   total_received_amount: Sum of transfers for that month (zero if no data)
-- Business Logic:
--   - Generates complete 12-month series
--   - Only includes active transfers and cards
--   - Uses LEFT JOIN to ensure all months appear
--   - COALESCE returns 0 for months with no activity
--   - Results ordered chronologically
-- name: GetMonthlyTransferAmountReceiver :many
WITH
    months AS (
        SELECT generate_series(
                date_trunc('year', $1::timestamp), date_trunc('year', $1::timestamp) + interval '1 year' - interval '1 day', interval '1 month'
            ) AS month
    )
SELECT TO_CHAR(m.month, 'Mon') AS month, COALESCE(SUM(t.transfer_amount), 0)::int AS total_received_amount
FROM months m
    LEFT JOIN transfers t ON EXTRACT(
        MONTH
        FROM t.transfer_time
    ) = EXTRACT(
        MONTH
        FROM m.month
    )
    AND EXTRACT(
        YEAR
        FROM t.transfer_time
    ) = EXTRACT(
        YEAR
        FROM m.month
    )
    AND t.deleted_at IS NULL
GROUP BY
    m.month
ORDER BY m.month;

-- GetYearlyTransferAmountSender: Retrieves yearly transfer totals for last 5 years
-- Purpose: Analyze long-term transfer trends and growth
-- Parameters:
--   $1: reference_year - The target year (includes this year plus previous 4)
-- Returns:
--   year: The 4-digit year
--   total_sent_amount: Sum of transfers for that year
-- Business Logic:
--   - Covers a 5-year rolling window
--   - Only includes active topups and cards
--   - Groups by calendar year
--   - Results ordered chronologically
--   - Useful for identifying annual growth patterns
-- name: GetYearlyTransferAmountSender :many
WITH
    last_five_years AS (
        SELECT EXTRACT(
                YEAR
                FROM t.transfer_time
            ) AS year, SUM(t.transfer_amount) AS total_sent_amount
        FROM transfers t
        WHERE
            t.deleted_at IS NULL
            AND EXTRACT(
                YEAR
                FROM t.transfer_time
            ) >= $1 - 4
            AND EXTRACT(
                YEAR
                FROM t.transfer_time
            ) <= $1
        GROUP BY
            EXTRACT(
                YEAR
                FROM t.transfer_time
            )
    )
SELECT year, total_sent_amount
FROM last_five_years
ORDER BY year;

-- GetYearlyTransferAmountReceiver: Retrieves yearly transfer totals for last 5 years
-- Purpose: Analyze long-term transfer trends and growth
-- Parameters:
--   $1: reference_year - The target year (includes this year plus previous 4)
-- Returns:
--   year: The 4-digit year
--   total_received_amount: Sum of transfers for that year
-- Business Logic:
--   - Covers a 5-year rolling window
--   - Only includes active topups and cards
--   - Groups by calendar year
--   - Results ordered chronologically
--   - Useful for identifying annual growth patterns
-- name: GetYearlyTransferAmountReceiver :many
WITH
    last_five_years AS (
        SELECT EXTRACT(
                YEAR
                FROM t.transfer_time
            ) AS year, SUM(t.transfer_amount) AS total_received_amount
        FROM transfers t
        WHERE
            t.deleted_at IS NULL
            AND EXTRACT(
                YEAR
                FROM t.transfer_time
            ) >= $1 - 4
            AND EXTRACT(
                YEAR
                FROM t.transfer_time
            ) <= $1
        GROUP BY
            EXTRACT(
                YEAR
                FROM t.transfer_time
            )
    )
SELECT year, total_received_amount
FROM last_five_years
ORDER BY year;

-- GetMonthlyBalancesByCardNumber: Retrieves monthly balance history for a specific card
-- Purpose: Track monthly balance trends for individual card statements
-- Parameters:
--   $1: reference_date - Date to determine the analysis year
--   $2: card_number - Specific card to analyze
-- Returns:
--   month: 3-letter month abbreviation (e.g., 'Jan')
--   total_balance: Monthly balance total (0 if no data)
-- Business Logic:
--   - Generates complete 12-month series for the year
--   - Filters for specific card number
--   - Only includes active saldos and cards
--   - Ensures all months appear with COALESCE default
--   - Useful for cardholder spending pattern analysis
-- name: GetMonthlyBalancesByCardNumber :many
WITH
    months AS (
        SELECT generate_series(
                date_trunc('year', $1::timestamp), date_trunc('year', $1::timestamp) + interval '1 year' - interval '1 day', interval '1 month'
            ) AS month
    )
SELECT TO_CHAR(m.month, 'Mon') AS month, COALESCE(SUM(s.total_balance), 0)::int AS total_balance
FROM
    months m
    LEFT JOIN saldos s ON EXTRACT(
        MONTH
        FROM s.created_at
    ) = EXTRACT(
        MONTH
        FROM m.month
    )
    AND EXTRACT(
        YEAR
        FROM s.created_at
    ) = EXTRACT(
        YEAR
        FROM m.month
    )
    AND s.deleted_at IS NULL
    LEFT JOIN cards c ON s.card_number = c.card_number
    AND c.deleted_at IS NULL
    AND c.card_number = $2
GROUP BY
    m.month
ORDER BY m.month;

-- GetYearlyBalancesByCardNumber: Retrieves 5-year balance history for a specific card
-- Purpose: Show annual balance trends for individual cardholders
-- Parameters:
--   $1: reference_year - Central year for 5-year window
--   $2: card_number - Specific card to analyze
-- Returns:
--   year: 4-digit year
--   total_balance: Annual balance total
-- Business Logic:
--   - Covers reference_year-4 to reference_year (5 years)
--   - Strictly filters for specified card
--   - Only includes active records
--   - Useful for long-term financial planning
-- name: GetYearlyBalancesByCardNumber :many
WITH
    last_five_years AS (
        SELECT EXTRACT(
                YEAR
                FROM s.created_at
            ) AS year, SUM(s.total_balance) AS total_balance
        FROM saldos s
            JOIN cards c ON s.card_number = c.card_number
        WHERE
            s.deleted_at IS NULL
            AND c.deleted_at IS NULL
            AND EXTRACT(
                YEAR
                FROM s.created_at
            ) >= $1 - 4
            AND EXTRACT(
                YEAR
                FROM s.created_at
            ) <= $1
            AND c.card_number = $2
        GROUP BY
            EXTRACT(
                YEAR
                FROM s.created_at
            )
    )
SELECT year, total_balance
FROM last_five_years
ORDER BY year;

-- GetMonthlyTopupAmountByCardNumber: Retrieves monthly top-up history for a card
-- Purpose: Analyze monthly top-up patterns for individual cards
-- Parameters:
--   $1: card_number - Specific card to analyze
--   $2: reference_date - Date to determine analysis year
-- Returns:
--   month: 3-letter month abbreviation
--   total_topup_amount: Monthly top-up total (0 if none)
-- Business Logic:
--   - Complete 12-month coverage
--   - Card-specific filtering
--   - Active records only
--   - Zero-filled for missing months
--   - Helps identify top-up habit seasonality
-- name: GetMonthlyTopupAmountByCardNumber :many
WITH
    months AS (
        SELECT generate_series(
                date_trunc('year', $2::timestamp), date_trunc('year', $2::timestamp) + interval '1 year' - interval '1 day', interval '1 month'
            ) AS month
    )
SELECT TO_CHAR(m.month, 'Mon') AS month, COALESCE(SUM(t.topup_amount), 0)::int AS total_topup_amount
FROM
    months m
    LEFT JOIN topups t ON EXTRACT(
        MONTH
        FROM t.topup_time
    ) = EXTRACT(
        MONTH
        FROM m.month
    )
    AND EXTRACT(
        YEAR
        FROM t.topup_time
    ) = EXTRACT(
        YEAR
        FROM m.month
    )
    AND t.deleted_at IS NULL
    LEFT JOIN cards c ON t.card_number = c.card_number
    AND c.deleted_at IS NULL
    AND t.card_number = $1
GROUP BY
    m.month
ORDER BY m.month;

-- GetYearlyTopupAmountByCardNumber: Retrieves 5-year top-up history for a card
-- Purpose: Track long-term top-up trends for individual cards
-- Parameters:
--   $1: card_number - Specific card to analyze
--   $2: reference_year - Central year for 5-year window
-- Returns:
--   year: 4-digit year
--   total_topup_amount: Annual top-up total
-- Business Logic:
--   - 5-year rolling window analysis
--   - Strict card number filtering
--   - Active records only
--   - Chronological ordering
--   - Useful for identifying annual top-up growth/decline
-- name: GetYearlyTopupAmountByCardNumber :many
WITH
    last_five_years AS (
        SELECT EXTRACT(
                YEAR
                FROM t.topup_time
            ) AS year, SUM(t.topup_amount) AS total_topup_amount
        FROM topups t
            JOIN cards c ON t.card_number = c.card_number
        WHERE
            t.deleted_at IS NULL
            AND c.deleted_at IS NULL
            AND t.card_number = $1
            AND EXTRACT(
                YEAR
                FROM t.topup_time
            ) >= $2 - 4
            AND EXTRACT(
                YEAR
                FROM t.topup_time
            ) <= $2
        GROUP BY
            EXTRACT(
                YEAR
                FROM t.topup_time
            )
    )
SELECT year, total_topup_amount
FROM last_five_years
ORDER BY year;

-- GetMonthlyWithdrawAmountByCardNumber: Retrieves monthly withdraw history for a card
-- Purpose: Analyze monthly withdraw patterns for individual cards
-- Parameters:
--   $1: card_number - Specific card to analyze
--   $2: reference_date - Date to determine analysis year
-- Returns:
--   month: 3-letter month abbreviation
--   total_withdraw_amount: Monthly withdraw total (0 if none)
-- Business Logic:
--   - Complete 12-month coverage
--   - Card-specific filtering
--   - Active records only
--   - Zero-filled for missing months
--   - Helps identify withdraw habit seasonality
-- name: GetMonthlyWithdrawAmountByCardNumber :many
WITH
    months AS (
        SELECT generate_series(
                date_trunc('year', $2::timestamp), date_trunc('year', $2::timestamp) + interval '1 year' - interval '1 day', interval '1 month'
            ) AS month
    )
SELECT TO_CHAR(m.month, 'Mon') AS month, COALESCE(SUM(w.withdraw_amount), 0)::int AS total_withdraw_amount
FROM
    months m
    LEFT JOIN withdraws w ON EXTRACT(
        MONTH
        FROM w.withdraw_time
    ) = EXTRACT(
        MONTH
        FROM m.month
    )
    AND EXTRACT(
        YEAR
        FROM w.withdraw_time
    ) = EXTRACT(
        YEAR
        FROM m.month
    )
    AND w.deleted_at IS NULL
    LEFT JOIN cards c ON w.card_number = c.card_number
    AND c.deleted_at IS NULL
    AND w.card_number = $1
GROUP BY
    m.month
ORDER BY m.month;

-- GetYearlyWithdrawAmountByCardNumber: Retrieves 5-year withdraw history for a card
-- Purpose: Track long-term withdraw trends for individual cards
-- Parameters:
--   $1: card_number - Specific card to analyze
--   $2: reference_year - Central year for 5-year window
-- Returns:
--   year: 4-digit year
--   total_withdraw_amount: Annual withdraw total
-- Business Logic:
--   - 5-year rolling window analysis
--   - Strict card number filtering
--   - Active records only
--   - Chronological ordering
--   - Useful for identifying annual withdraw growth/decline
-- name: GetYearlyWithdrawAmountByCardNumber :many
WITH
    last_five_years AS (
        SELECT EXTRACT(
                YEAR
                FROM w.withdraw_time
            ) AS year, SUM(w.withdraw_amount) AS total_withdraw_amount
        FROM withdraws w
            JOIN cards c ON w.card_number = c.card_number
        WHERE
            w.deleted_at IS NULL
            AND c.deleted_at IS NULL
            AND w.card_number = $1
            AND EXTRACT(
                YEAR
                FROM w.withdraw_time
            ) >= $2 - 4
            AND EXTRACT(
                YEAR
                FROM w.withdraw_time
            ) <= $2
        GROUP BY
            EXTRACT(
                YEAR
                FROM w.withdraw_time
            )
    )
SELECT year, total_withdraw_amount
FROM last_five_years
ORDER BY year;

-- GetMonthlyTransactionAmountByCardNumber: Retrieves monthly transaction history for a card
-- Purpose: Analyze monthly transaction patterns for individual cards
-- Parameters:
--   $1: card_number - Specific card to analyze
--   $2: reference_date - Date to determine analysis year
-- Returns:
--   month: 3-letter month abbreviation
--   total_transaction_amount: Monthly transaction total (0 if none)
-- Business Logic:
--   - Complete 12-month coverage
--   - Card-specific filtering
--   - Active records only
--   - Zero-filled for missing months
--   - Helps identify transaction habit seasonality
-- name: GetMonthlyTransactionAmountByCardNumber :many
WITH
    months AS (
        SELECT generate_series(
                date_trunc('year', $2::timestamp), date_trunc('year', $2::timestamp) + interval '1 year' - interval '1 day', interval '1 month'
            ) AS month
    )
SELECT
    TO_CHAR(m.month, 'Mon') AS month,
    COALESCE(SUM(t.amount), 0)::int AS total_transaction_amount
FROM
    months m
    LEFT JOIN transactions t ON EXTRACT(
        MONTH
        FROM t.transaction_time
    ) = EXTRACT(
        MONTH
        FROM m.month
    )
    AND EXTRACT(
        YEAR
        FROM t.transaction_time
    ) = EXTRACT(
        YEAR
        FROM m.month
    )
    AND t.deleted_at IS NULL
    LEFT JOIN cards c ON t.card_number = c.card_number
    AND c.deleted_at IS NULL
    AND t.card_number = $1
GROUP BY
    m.month
ORDER BY m.month;

-- GetYearlyTransactionAmountByCardNumber: Retrieves 5-year transaction history for a card
-- Purpose: Track long-term transaction trends for individual cards
-- Parameters:
--   $1: card_number - Specific card to analyze
--   $2: reference_year - Central year for 5-year window
-- Returns:
--   year: 4-digit year
--   total_transaction_amount: Annual transaction total
-- Business Logic:
--   - 5-year rolling window analysis
--   - Strict card number filtering
--   - Active records only
--   - Chronological ordering
--   - Useful for identifying annual transaction growth/decline
-- name: GetYearlyTransactionAmountByCardNumber :many
WITH
    last_five_years AS (
        SELECT EXTRACT(
                YEAR
                FROM t.transaction_time
            ) AS year, SUM(t.amount) AS total_transaction_amount
        FROM transactions t
            JOIN cards c ON t.card_number = c.card_number
        WHERE
            t.deleted_at IS NULL
            AND c.deleted_at IS NULL
            AND t.card_number = $1
            AND EXTRACT(
                YEAR
                FROM t.transaction_time
            ) >= $2 - 4
            AND EXTRACT(
                YEAR
                FROM t.transaction_time
            ) <= $2
        GROUP BY
            EXTRACT(
                YEAR
                FROM t.transaction_time
            )
    )
SELECT year, total_transaction_amount
FROM last_five_years
ORDER BY year;

-- GetMonthlyTransferAmountBySender: Retrieves monthly transfer history for a card
-- Purpose: Analyze monthly transfer patterns for individual cards
-- Parameters:
--   $1: card_number - Specific card to analyze
--   $2: reference_date - Date to determine analysis year
-- Returns:
--   month: 3-letter month abbreviation
--   total_sent_amount: Monthly transfer total (0 if none)
-- Business Logic:
--   - Complete 12-month coverage
--   - Card-specific filtering
--   - Active records only
--   - Zero-filled for missing months
--   - Helps identify transfer habit seasonality
-- name: GetMonthlyTransferAmountBySender :many
WITH
    months AS (
        SELECT generate_series(
                date_trunc('year', $2::timestamp), date_trunc('year', $2::timestamp) + interval '1 year' - interval '1 day', interval '1 month'
            ) AS month
    )
SELECT TO_CHAR(m.month, 'Mon') AS month, COALESCE(SUM(t.transfer_amount), 0)::int AS total_sent_amount
FROM
    months m
    LEFT JOIN transfers t ON EXTRACT(
        MONTH
        FROM t.transfer_time
    ) = EXTRACT(
        MONTH
        FROM m.month
    )
    AND EXTRACT(
        YEAR
        FROM t.transfer_time
    ) = EXTRACT(
        YEAR
        FROM m.month
    )
    AND t.deleted_at IS NULL
    AND t.transfer_from = $1
GROUP BY
    m.month
ORDER BY m.month;

-- GetMonthlyTransferAmountByReceiver: Retrieves monthly transfer history for a card
-- Purpose: Analyze monthly transfer patterns for individual cards
-- Parameters:
--   $1: card_number - Specific card to analyze
--   $2: reference_date - Date to determine analysis year
-- Returns:
--   month: 3-letter month abbreviation
--   total_sent_amount: Monthly transfer total (0 if none)
-- Business Logic:
--   - Complete 12-month coverage
--   - Card-specific filtering
--   - Active records only
--   - Zero-filled for missing months
--   - Helps identify transfer habit seasonality
-- name: GetMonthlyTransferAmountByReceiver :many
WITH
    months AS (
        SELECT generate_series(
                date_trunc('year', $2::timestamp), date_trunc('year', $2::timestamp) + interval '1 year' - interval '1 day', interval '1 month'
            ) AS month
    )
SELECT TO_CHAR(m.month, 'Mon') AS month, COALESCE(SUM(t.transfer_amount), 0)::int AS total_received_amount
FROM
    months m
    LEFT JOIN transfers t ON EXTRACT(
        MONTH
        FROM t.transfer_time
    ) = EXTRACT(
        MONTH
        FROM m.month
    )
    AND EXTRACT(
        YEAR
        FROM t.transfer_time
    ) = EXTRACT(
        YEAR
        FROM m.month
    )
    AND t.deleted_at IS NULL
    AND t.transfer_to = $1
GROUP BY
    m.month
ORDER BY m.month;

-- GetYearlyTransferAmountBySender: Retrieves 5-year transfer history for a card
-- Purpose: Track long-term transfer trends for individual cards
-- Parameters:
--   $1: card_number - Specific card to analyze
--   $2: reference_year - Central year for 5-year window
-- Returns:
--   year: 4-digit year
--   total_sent_amount: Annual transfer total
-- Business Logic:
--   - 5-year rolling window analysis
--   - Strict card number filtering
--   - Active records only
--   - Chronological ordering
--   - Useful for identifying annual transfer growth/decline
-- name: GetYearlyTransferAmountBySender :many
WITH
    last_five_years AS (
        SELECT EXTRACT(
                YEAR
                FROM t.transfer_time
            ) AS year, SUM(t.transfer_amount) AS total_sent_amount
        FROM transfers t
        WHERE
            t.deleted_at IS NULL
            AND t.transfer_from = $1
            AND EXTRACT(
                YEAR
                FROM t.transfer_time
            ) >= $2 - 4
            AND EXTRACT(
                YEAR
                FROM t.transfer_time
            ) <= $2
        GROUP BY
            EXTRACT(
                YEAR
                FROM t.transfer_time
            )
    )
SELECT year, total_sent_amount
FROM last_five_years
ORDER BY year;

-- GetYearlyTransferAmountByReceiver: Retrieves 5-year transfer history for a card
-- Purpose: Track long-term transfer trends for individual cards
-- Parameters:
--   $1: card_number - Specific card to analyze
--   $2: reference_year - Central year for 5-year window
-- Returns:
--   year: 4-digit year
--   total_sent_amount: Annual transfer total
-- Business Logic:
--   - 5-year rolling window analysis
--   - Strict card number filtering
--   - Active records only
--   - Chronological ordering
--   - Useful for identifying annual transfer growth/decline
-- name: GetYearlyTransferAmountByReceiver :many
WITH
    last_five_years AS (
        SELECT EXTRACT(
                YEAR
                FROM t.transfer_time
            ) AS year, SUM(t.transfer_amount) AS total_received_amount
        FROM transfers t
        WHERE
            t.deleted_at IS NULL
            AND t.transfer_to = $1
            AND EXTRACT(
                YEAR
                FROM t.transfer_time
            ) >= $2 - 4
            AND EXTRACT(
                YEAR
                FROM t.transfer_time
            ) <= $2
        GROUP BY
            EXTRACT(
                YEAR
                FROM t.transfer_time
            )
    )
SELECT year, total_received_amount
FROM last_five_years
ORDER BY year;

-- CreateCard: Creates a new card record
-- Purpose: Add a new card to the system for a specific user
-- Parameters:
--   $1: user_id - Owner of the card
--   $2: card_number - Unique number of the card
--   $3: card_type - Type of the card (e.g., debit, credit)
--   $4: expire_date - Expiration date of the card
--   $5: cvv - Card verification value
--   $6: card_provider - Provider/issuer of the card
-- Returns: Complete created card record
-- Business Logic:
--   - Automatically sets created_at and updated_at timestamps
--   - Requires all fields to be provided
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