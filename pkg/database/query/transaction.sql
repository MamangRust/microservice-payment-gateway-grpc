-- GetTransactions: Retrieves paginated transaction records with search capability
-- Purpose: List all transactions for management UI with filtering options
-- Parameters:
--   $1: search_term - Optional text to filter transactions by card number, payment method, or status (NULL for no filter)
--   $2: limit - Maximum number of records to return
--   $3: offset - Number of records to skip for pagination
-- Returns:
--   All transaction fields plus total_count of matching records
-- Business Logic:
--   - Excludes soft-deleted transactions (deleted_at IS NULL)
--   - Supports partial text matching on multiple fields (case-insensitive)
--   - Orders by transaction_time (newest first)
--   - Provides total_count for pagination calculations
--   - Useful for transaction monitoring and auditing
-- name: GetTransactions :many
SELECT
    transaction_id,
    transaction_no,
    card_number,
    amount,
    payment_method,
    merchant_id,
    transaction_time,
    status,
    created_at,
    updated_at,
    COUNT(*) OVER () AS total_count
FROM transactions
WHERE
    deleted_at IS NULL
    AND (
        $1::TEXT IS NULL
        OR card_number ILIKE '%' || $1 || '%'
        OR payment_method ILIKE '%' || $1 || '%'
        OR status ILIKE '%' || $1 || '%'
    )
ORDER BY transaction_time DESC
LIMIT $2
OFFSET
    $3;

-- GetActiveTransactions: Retrieves paginated active transactions with search
-- Purpose: List all non-deleted transactions with filtering options
-- Parameters:
--   $1: search_term - Optional text to filter by card number or payment method
--   $2: limit - Maximum records to return
--   $3: offset - Records to skip for pagination
-- Returns:
--   All transaction fields plus total_count of matching active records
-- Business Logic:
--   - Only includes active transactions (deleted_at IS NULL)
--   - Filters on card_number and payment_method fields
--   - Orders by transaction_time (newest first)
--   - Provides pagination metadata
--   - Used in transaction management interfaces
-- name: GetActiveTransactions :many
SELECT
    transaction_id,
    transaction_no,
    card_number,
    amount,
    payment_method,
    merchant_id,
    transaction_time,
    status,
    created_at,
    updated_at,
    deleted_at,
    COUNT(*) OVER () AS total_count
FROM transactions
WHERE
    deleted_at IS NULL
    AND (
        $1::TEXT IS NULL
        OR card_number ILIKE '%' || $1 || '%'
        OR payment_method ILIKE '%' || $1 || '%'
    )
ORDER BY transaction_time DESC
LIMIT $2
OFFSET
    $3;

-- GetTrashedTransactions: Retrieves paginated soft-deleted transactions
-- Purpose: List all deleted transactions for recovery or audit purposes
-- Parameters:
--   $1: search_term - Optional text to filter deleted transactions
--   $2: limit - Maximum records to return
--   $3: offset - Records to skip for pagination
-- Returns:
--   All transaction fields plus total_count of matching deleted records
-- Business Logic:
--   - Only includes soft-deleted transactions (deleted_at IS NOT NULL)
--   - Same filtering capabilities as active transactions
--   - Maintains newest-first ordering
--   - Used in admin interfaces for transaction recovery
-- name: GetTrashedTransactions :many
SELECT
    transaction_id,
    transaction_no,
    card_number,
    amount,
    payment_method,
    merchant_id,
    transaction_time,
    status,
    created_at,
    updated_at,
    deleted_at,
    COUNT(*) OVER () AS total_count
FROM transactions
WHERE
    deleted_at IS NOT NULL
    AND (
        $1::TEXT IS NULL
        OR card_number ILIKE '%' || $1 || '%'
        OR payment_method ILIKE '%' || $1 || '%'
    )
ORDER BY transaction_time DESC
LIMIT $2
OFFSET
    $3;

-- GetTransactionByID: Retrieves a single transaction by its ID
-- Purpose: Get detailed information about a specific transaction
-- Parameters:
--   $1: transaction_id - The ID of the transaction to retrieve
-- Returns:
--   All fields for the specified transaction or NULL if not found/deleted
-- Business Logic:
--   - Only returns active transactions (deleted_at IS NULL)
--   - Useful for transaction details viewing and verification
-- name: GetTransactionByID :one
SELECT
    transaction_id,
    transaction_no,
    card_number,
    amount,
    payment_method,
    merchant_id,
    transaction_time,
    status,
    created_at,
    updated_at
FROM transactions
WHERE
    transaction_id = $1
    AND deleted_at IS NULL;

-- GetTransactionsByCardNumber: Retrieves paginated transactions for a specific card
-- Purpose: List all transactions associated with a particular card
-- Parameters:
--   $1: card_number - The card number to filter transactions
--   $2: search_term - Optional text to filter by payment method or status
--   $3: limit - Maximum number of records to return
--   $4: offset - Number of records to skip for pagination
-- Returns:
--   All transaction fields plus total_count of matching records
-- Business Logic:
--   - Only includes active transactions (deleted_at IS NULL)
--   - Strict card number matching combined with optional search filters
--   - Orders by transaction_time (newest first)
--   - Provides pagination support with total_count
--   - Useful for cardholder transaction history
-- name: GetTransactionsByCardNumber :many
SELECT
    transaction_id,
    transaction_no,
    card_number,
    amount,
    payment_method,
    merchant_id,
    transaction_time,
    status,
    created_at,
    updated_at,
    COUNT(*) OVER () AS total_count
FROM transactions
WHERE
    deleted_at IS NULL
    AND card_number = $1
    AND (
        $2::TEXT IS NULL
        OR payment_method ILIKE '%' || $2 || '%'
        OR status ILIKE '%' || $2 || '%'
    )
ORDER BY transaction_time DESC
LIMIT $3
OFFSET
    $4;

-- GetTransactionsByMerchantID: Retrieves transactions for a specific merchant
-- Purpose: List all transactions associated with a merchant
-- Parameters:
--   $1: merchant_id - The ID of the merchant to filter transactions
-- Returns:
--   All transaction fields for the merchant's transactions
-- Business Logic:
--   - Only includes active transactions (deleted_at IS NULL)
--   - Orders by transaction_time (newest first)
--   - No pagination (assumes manageable number of records per merchant)
--   - Useful for merchant transaction reports
-- name: GetTransactionsByMerchantID :many
SELECT
    transaction_id,
    transaction_no,
    card_number,
    amount,
    payment_method,
    merchant_id,
    transaction_time,
    status,
    created_at,
    updated_at
FROM transactions
WHERE
    merchant_id = $1
    AND deleted_at IS NULL
ORDER BY transaction_time DESC;

-- GetTrashedTransactionByID: Retrieves a single soft-deleted transaction by ID
-- Purpose: View details of a deleted transaction for recovery or audit
-- Parameters:
--   $1: transaction_id - The ID of the transaction to retrieve
-- Returns:
--   All fields for the specified trashed transaction or NULL if not found/active
-- Business Logic:
--   - Only returns soft-deleted transactions (deleted_at IS NOT NULL)
--   - Used in admin interfaces for transaction recovery
-- name: GetTrashedTransactionByID :one
SELECT *
FROM transactions
WHERE
    transaction_id = $1
    AND deleted_at IS NOT NULL;

-- GetMonthTransactionStatusSuccess: Retrieves monthly success metrics for transactions
-- Purpose: Analyze successful transaction trends across comparison periods
-- Parameters:
--   $1: period1_start - Start date of first comparison period
--   $2: period1_end - End date of first comparison period
--   $3: period2_start - Start date of second comparison period
--   $4: period2_end - End date of second comparison period
-- Returns:
--   year: Year as text (e.g., '2023')
--   month: 3-letter month abbreviation (e.g., 'Jan')
--   total_success: Count of successful transactions
--   total_amount: Sum of successful transaction amounts
-- Business Logic:
--   - Only includes successful transactions (status = 'success')
--   - Covers two customizable time periods for comparison
--   - Zero-fills months with no activity
--   - Formats output for consistent visualization (year as text, month as 'Mon')
--   - Orders by year and month (newest first)
--   - Useful for identifying seasonal transaction patterns and revenue trends
-- name: GetMonthTransactionStatusSuccess :many
WITH
    monthly_data AS (
        SELECT
            EXTRACT(
                YEAR
                FROM t.transaction_time
            )::integer AS year,
            EXTRACT(
                MONTH
                FROM t.transaction_time
            )::integer AS month,
            COUNT(*) AS total_success,
            COALESCE(SUM(t.amount), 0)::integer AS total_amount
        FROM transactions t
        WHERE
            t.deleted_at IS NULL
            AND t.status = 'success'
            AND (
                (
                    t.transaction_time >= $1::timestamp
                    AND t.transaction_time <= $2::timestamp
                )
                OR (
                    t.transaction_time >= $3::timestamp
                    AND t.transaction_time <= $4::timestamp
                )
            )
        GROUP BY
            EXTRACT(
                YEAR
                FROM t.transaction_time
            ),
            EXTRACT(
                MONTH
                FROM t.transaction_time
            )
    ),
    formatted_data AS (
        SELECT
            year::text,
            TO_CHAR(
                TO_DATE(month::text, 'MM'),
                'Mon'
            ) AS month,
            total_success,
            total_amount
        FROM monthly_data
        UNION ALL
        SELECT
            EXTRACT(
                YEAR
                FROM $1::timestamp
            )::text AS year,
            TO_CHAR($1::timestamp, 'Mon') AS month,
            0 AS total_success,
            0 AS total_amount
        WHERE
            NOT EXISTS (
                SELECT 1
                FROM monthly_data
                WHERE
                    year = EXTRACT(
                        YEAR
                        FROM $1::timestamp
                    )::integer
                    AND month = EXTRACT(
                        MONTH
                        FROM $1::timestamp
                    )::integer
            )
        UNION ALL
        SELECT
            EXTRACT(
                YEAR
                FROM $3::timestamp
            )::text AS year,
            TO_CHAR($3::timestamp, 'Mon') AS month,
            0 AS total_success,
            0 AS total_amount
        WHERE
            NOT EXISTS (
                SELECT 1
                FROM monthly_data
                WHERE
                    year = EXTRACT(
                        YEAR
                        FROM $3::timestamp
                    )::integer
                    AND month = EXTRACT(
                        MONTH
                        FROM $3::timestamp
                    )::integer
            )
    )
SELECT *
FROM formatted_data
ORDER BY year DESC, TO_DATE(month, 'Mon') DESC;

-- GetYearlyTransactionStatusSuccess: Retrieves yearly success metrics for transactions
-- Purpose: Compare annual successful transaction performance
-- Parameters:
--   $1: current_year - The target year (includes this year and previous)
-- Returns:
--   year: Year as text (e.g., '2023')
--   total_success: Count of successful transactions
--   total_amount: Sum of successful transaction amounts
-- Business Logic:
--   - Only includes successful transactions (status = 'success')
--   - Compares current year with previous year
--   - Zero-fills years with no activity
--   - Orders by year (newest first)
--   - Useful for year-over-year growth analysis and financial reporting
--   - Helps identify annual transaction volume and revenue trends
-- name: GetYearlyTransactionStatusSuccess :many
WITH
    yearly_data AS (
        SELECT
            EXTRACT(
                YEAR
                FROM t.transaction_time
            )::integer AS year,
            COUNT(*) AS total_success,
            COALESCE(SUM(t.amount), 0)::integer AS total_amount
        FROM transactions t
        WHERE
            t.deleted_at IS NULL
            AND t.status = 'success'
            AND (
                EXTRACT(
                    YEAR
                    FROM t.transaction_time
                ) = $1::integer
                OR EXTRACT(
                    YEAR
                    FROM t.transaction_time
                ) = $1::integer - 1
            )
        GROUP BY
            EXTRACT(
                YEAR
                FROM t.transaction_time
            )
    ),
    formatted_data AS (
        SELECT
            year::text,
            total_success::integer,
            total_amount::integer
        FROM yearly_data
        UNION ALL
        SELECT
            $1::text AS year,
            0::integer AS total_success,
            0::integer AS total_amount
        WHERE
            NOT EXISTS (
                SELECT 1
                FROM yearly_data
                WHERE
                    year = $1::integer
            )
        UNION ALL
        SELECT ($1::integer - 1)::text AS year,
            0::integer AS total_success,
            0::integer AS total_amount
        WHERE
            NOT EXISTS (
                SELECT 1
                FROM yearly_data
                WHERE
                    year = $1::integer - 1
            )
    )
SELECT *
FROM formatted_data
ORDER BY year DESC;

-- GetMonthTransactionStatusFailed: Retrieves monthly failed metrics for transactions
-- Purpose: Analyze failedful transaction trends across comparison periods
-- Parameters:
--   $1: period1_start - Start date of first comparison period
--   $2: period1_end - End date of first comparison period
--   $3: period2_start - Start date of second comparison period
--   $4: period2_end - End date of second comparison period
-- Returns:
--   year: Year as text (e.g., '2023')
--   month: 3-letter month abbreviation (e.g., 'Jan')
--   total_failed: Count of failedful transactions
--   total_amount: Sum of failedful transaction amounts
-- Business Logic:
--   - Only includes failedful transactions (status = 'failed')
--   - Covers two customizable time periods for comparison
--   - Zero-fills months with no activity
--   - Formats output for consistent visualization (year as text, month as 'Mon')
--   - Orders by year and month (newest first)
--   - Useful for identifying seasonal transaction patterns and revenue trends
-- name: GetMonthTransactionStatusFailed :many
WITH
    monthly_data AS (
        SELECT
            EXTRACT(
                YEAR
                FROM t.transaction_time
            )::integer AS year,
            EXTRACT(
                MONTH
                FROM t.transaction_time
            )::integer AS month,
            COUNT(*) AS total_failed,
            COALESCE(SUM(t.amount), 0)::integer AS total_amount
        FROM transactions t
        WHERE
            t.deleted_at IS NULL
            AND t.status = 'failed'
            AND (
                (
                    t.transaction_time >= $1::timestamp
                    AND t.transaction_time <= $2::timestamp
                )
                OR (
                    t.transaction_time >= $3::timestamp
                    AND t.transaction_time <= $4::timestamp
                )
            )
        GROUP BY
            EXTRACT(
                YEAR
                FROM t.transaction_time
            ),
            EXTRACT(
                MONTH
                FROM t.transaction_time
            )
    ),
    formatted_data AS (
        SELECT
            year::text,
            TO_CHAR(
                TO_DATE(month::text, 'MM'),
                'Mon'
            ) AS month,
            total_failed,
            total_amount
        FROM monthly_data
        UNION ALL
        SELECT
            EXTRACT(
                YEAR
                FROM $1::timestamp
            )::text AS year,
            TO_CHAR($1::timestamp, 'Mon') AS month,
            0 AS total_failed,
            0 AS total_amount
        WHERE
            NOT EXISTS (
                SELECT 1
                FROM monthly_data
                WHERE
                    year = EXTRACT(
                        YEAR
                        FROM $1::timestamp
                    )::integer
                    AND month = EXTRACT(
                        MONTH
                        FROM $1::timestamp
                    )::integer
            )
        UNION ALL
        SELECT
            EXTRACT(
                YEAR
                FROM $3::timestamp
            )::text AS year,
            TO_CHAR($3::timestamp, 'Mon') AS month,
            0 AS total_failed,
            0 AS total_amount
        WHERE
            NOT EXISTS (
                SELECT 1
                FROM monthly_data
                WHERE
                    year = EXTRACT(
                        YEAR
                        FROM $3::timestamp
                    )::integer
                    AND month = EXTRACT(
                        MONTH
                        FROM $3::timestamp
                    )::integer
            )
    )
SELECT *
FROM formatted_data
ORDER BY year DESC, TO_DATE(month, 'Mon') DESC;

-- GetYearlyTransactionStatusFailed: Retrieves yearly failed metrics for transactions
-- Purpose: Compare annual failedful transaction performance
-- Parameters:
--   $1: current_year - The target year (includes this year and previous)
-- Returns:
--   year: Year as text (e.g., '2023')
--   total_failed: Count of failedful transactions
--   total_amount: Sum of failedful transaction amounts
-- Business Logic:
--   - Only includes failedful transactions (status = 'failed')
--   - Compares current year with previous year
--   - Zero-fills years with no activity
--   - Orders by year (newest first)
--   - Useful for year-over-year growth analysis and financial reporting
--   - Helps identify annual transaction volume and revenue trends
-- name: GetYearlyTransactionStatusFailed :many
WITH
    yearly_data AS (
        SELECT
            EXTRACT(
                YEAR
                FROM t.transaction_time
            )::integer AS year,
            COUNT(*) AS total_failed,
            COALESCE(SUM(t.amount), 0)::integer AS total_amount
        FROM transactions t
        WHERE
            t.deleted_at IS NULL
            AND t.status = 'failed'
            AND (
                EXTRACT(
                    YEAR
                    FROM t.transaction_time
                ) = $1::integer
                OR EXTRACT(
                    YEAR
                    FROM t.transaction_time
                ) = $1::integer - 1
            )
        GROUP BY
            EXTRACT(
                YEAR
                FROM t.transaction_time
            )
    ),
    formatted_data AS (
        SELECT
            year::text,
            total_failed::integer,
            total_amount::integer
        FROM yearly_data
        UNION ALL
        SELECT
            $1::text AS year,
            0::integer AS total_failed,
            0::integer AS total_amount
        WHERE
            NOT EXISTS (
                SELECT 1
                FROM yearly_data
                WHERE
                    year = $1::integer
            )
        UNION ALL
        SELECT ($1::integer - 1)::text AS year,
            0::integer AS total_failed,
            0::integer AS total_amount
        WHERE
            NOT EXISTS (
                SELECT 1
                FROM yearly_data
                WHERE
                    year = $1::integer - 1
            )
    )
SELECT *
FROM formatted_data
ORDER BY year DESC;

-- GetMonthTransactionStatusSuccessCardNumber: Retrieves monthly success metrics for transactions
-- Purpose: Analyze successful transaction trends across comparison periods
-- Parameters:
--   $1: card_number  - filter by card_number
--   $2: period1_start - Start date of first comparison period
--   $3: period1_end - End date of first comparison period
--   $4: period2_start - Start date of second comparison period
--   $5: period2_end - End date of second comparison period
-- Returns:
--   year: Year as text (e.g., '2023')
--   month: 3-letter month abbreviation (e.g., 'Jan')
--   total_success: Count of successful transactions
--   total_amount: Sum of successful transaction amounts
-- Business Logic:
--   - Only includes successful transactions (status = 'success')
--   - Covers two customizable time periods for comparison
--   - Zero-fills months with no activity
--   - Formats output for consistent visualization (year as text, month as 'Mon')
--   - Orders by year and month (newest first)
--   - Useful for identifying seasonal transaction patterns and revenue trends
-- name: GetMonthTransactionStatusSuccessCardNumber :many
WITH
    monthly_data AS (
        SELECT
            EXTRACT(
                YEAR
                FROM t.transaction_time
            )::integer AS year,
            EXTRACT(
                MONTH
                FROM t.transaction_time
            )::integer AS month,
            COUNT(*) AS total_success,
            COALESCE(SUM(t.amount), 0)::integer AS total_amount
        FROM transactions t
        WHERE
            t.deleted_at IS NULL
            AND t.status = 'success'
            AND t.card_number = $1
            AND (
                (
                    t.transaction_time >= $2::timestamp
                    AND t.transaction_time <= $3::timestamp
                )
                OR (
                    t.transaction_time >= $4::timestamp
                    AND t.transaction_time <= $5::timestamp
                )
            )
        GROUP BY
            EXTRACT(
                YEAR
                FROM t.transaction_time
            ),
            EXTRACT(
                MONTH
                FROM t.transaction_time
            )
    ),
    formatted_data AS (
        SELECT
            year::text,
            TO_CHAR(
                TO_DATE(month::text, 'MM'),
                'Mon'
            ) AS month,
            total_success,
            total_amount
        FROM monthly_data
        UNION ALL
        SELECT
            EXTRACT(
                YEAR
                FROM $2::timestamp
            )::text AS year,
            TO_CHAR($2::timestamp, 'Mon') AS month,
            0 AS total_success,
            0 AS total_amount
        WHERE
            NOT EXISTS (
                SELECT 1
                FROM monthly_data
                WHERE
                    year = EXTRACT(
                        YEAR
                        FROM $2::timestamp
                    )::integer
                    AND month = EXTRACT(
                        MONTH
                        FROM $2::timestamp
                    )::integer
            )
        UNION ALL
        SELECT
            EXTRACT(
                YEAR
                FROM $3::timestamp
            )::text AS year,
            TO_CHAR($3::timestamp, 'Mon') AS month,
            0 AS total_success,
            0 AS total_amount
        WHERE
            NOT EXISTS (
                SELECT 1
                FROM monthly_data
                WHERE
                    year = EXTRACT(
                        YEAR
                        FROM $3::timestamp
                    )::integer
                    AND month = EXTRACT(
                        MONTH
                        FROM $3::timestamp
                    )::integer
            )
    )
SELECT *
FROM formatted_data
ORDER BY year DESC, TO_DATE(month, 'Mon') DESC;

-- GetYearlyTransactionStatusSuccessCardNumber: Retrieves yearly success metrics for transactions
-- Purpose: Compare annual successful transaction performance
-- Parameters:
--   $1: card_number  - filter by card_number
--   $2: current_year - The target year (includes this year and previous)
-- Returns:
--   year: Year as text (e.g., '2023')
--   total_success: Count of successful transactions
--   total_amount: Sum of successful transaction amounts
-- Business Logic:
--   - Only includes successful transactions (status = 'success')
--   - Compares current year with previous year
--   - Zero-fills years with no activity
--   - Orders by year (newest first)
--   - Useful for year-over-year growth analysis and financial reporting
--   - Helps identify annual transaction volume and revenue trends
-- name: GetYearlyTransactionStatusSuccessCardNumber :many
WITH
    yearly_data AS (
        SELECT
            EXTRACT(
                YEAR
                FROM t.transaction_time
            )::integer AS year,
            COUNT(*) AS total_success,
            COALESCE(SUM(t.amount), 0)::integer AS total_amount
        FROM transactions t
        WHERE
            t.deleted_at IS NULL
            AND t.status = 'success'
            AND t.card_number = $1
            AND (
                EXTRACT(
                    YEAR
                    FROM t.transaction_time
                ) = $2::integer
                OR EXTRACT(
                    YEAR
                    FROM t.transaction_time
                ) = $2::integer - 1
            )
        GROUP BY
            EXTRACT(
                YEAR
                FROM t.transaction_time
            )
    ),
    formatted_data AS (
        SELECT
            year::text,
            total_success::integer,
            total_amount::integer
        FROM yearly_data
        UNION ALL
        SELECT
            $2::text AS year,
            0::integer AS total_success,
            0::integer AS total_amount
        WHERE
            NOT EXISTS (
                SELECT 1
                FROM yearly_data
                WHERE
                    year = $2::integer
            )
        UNION ALL
        SELECT ($2::integer - 1)::text AS year,
            0::integer AS total_success,
            0::integer AS total_amount
        WHERE
            NOT EXISTS (
                SELECT 1
                FROM yearly_data
                WHERE
                    year = $2::integer - 1
            )
    )
SELECT *
FROM formatted_data
ORDER BY year DESC;

-- GetMonthTransactionStatusFailed: Retrieves monthly failed metrics for transactions
-- Purpose: Analyze failedful transaction trends across comparison periods
-- Parameters:
--   $1: card_number  - filter by card_number
--   $2: period1_start - Start date of first comparison period
--   $3: period1_end - End date of first comparison period
--   $4: period2_start - Start date of second comparison period
--   $5: period2_end - End date of second comparison period
-- Returns:
--   year: Year as text (e.g., '2023')
--   month: 3-letter month abbreviation (e.g., 'Jan')
--   total_failed: Count of failedful transactions
--   total_amount: Sum of failedful transaction amounts
-- Business Logic:
--   - Only includes failedful transactions (status = 'failed')
--   - Covers two customizable time periods for comparison
--   - Zero-fills months with no activity
--   - Formats output for consistent visualization (year as text, month as 'Mon')
--   - Orders by year and month (newest first)
--   - Useful for identifying seasonal transaction patterns and revenue trends
-- name: GetMonthTransactionStatusFailedCardNumber :many
WITH
    monthly_data AS (
        SELECT
            EXTRACT(
                YEAR
                FROM t.transaction_time
            )::integer AS year,
            EXTRACT(
                MONTH
                FROM t.transaction_time
            )::integer AS month,
            COUNT(*) AS total_failed,
            COALESCE(SUM(t.amount), 0)::integer AS total_amount
        FROM transactions t
        WHERE
            t.deleted_at IS NULL
            AND t.status = 'failed'
            AND t.card_number = $1
            AND (
                (
                    t.transaction_time >= $2::timestamp
                    AND t.transaction_time <= $3::timestamp
                )
                OR (
                    t.transaction_time >= $4::timestamp
                    AND t.transaction_time <= $5::timestamp
                )
            )
        GROUP BY
            EXTRACT(
                YEAR
                FROM t.transaction_time
            ),
            EXTRACT(
                MONTH
                FROM t.transaction_time
            )
    ),
    formatted_data AS (
        SELECT
            year::text,
            TO_CHAR(
                TO_DATE(month::text, 'MM'),
                'Mon'
            ) AS month,
            total_failed,
            total_amount
        FROM monthly_data
        UNION ALL
        SELECT
            EXTRACT(
                YEAR
                FROM $2::timestamp
            )::text AS year,
            TO_CHAR($2::timestamp, 'Mon') AS month,
            0 AS total_failed,
            0 AS total_amount
        WHERE
            NOT EXISTS (
                SELECT 1
                FROM monthly_data
                WHERE
                    year = EXTRACT(
                        YEAR
                        FROM $2::timestamp
                    )::integer
                    AND month = EXTRACT(
                        MONTH
                        FROM $2::timestamp
                    )::integer
            )
        UNION ALL
        SELECT
            EXTRACT(
                YEAR
                FROM $3::timestamp
            )::text AS year,
            TO_CHAR($3::timestamp, 'Mon') AS month,
            0 AS total_failed,
            0 AS total_amount
        WHERE
            NOT EXISTS (
                SELECT 1
                FROM monthly_data
                WHERE
                    year = EXTRACT(
                        YEAR
                        FROM $3::timestamp
                    )::integer
                    AND month = EXTRACT(
                        MONTH
                        FROM $3::timestamp
                    )::integer
            )
    )
SELECT *
FROM formatted_data
ORDER BY year DESC, TO_DATE(month, 'Mon') DESC;

-- GetYearlyTransactionStatusFailed: Retrieves yearly failed metrics for transactions
-- Purpose: Compare annual failedful transaction performance
-- Parameters:
--   $1: card_number  - filter by card_number
--   $2: current_year - The target year (includes this year and previous)
-- Returns:
--   year: Year as text (e.g., '2023')
--   total_failed: Count of failedful transactions
--   total_amount: Sum of failedful transaction amounts
-- Business Logic:
--   - Only includes failedful transactions (status = 'failed')
--   - Compares current year with previous year
--   - Zero-fills years with no activity
--   - Orders by year (newest first)
--   - Useful for year-over-year growth analysis and financial reporting
--   - Helps identify annual transaction volume and revenue trends
-- name: GetYearlyTransactionStatusFailedCardNumber :many
WITH
    yearly_data AS (
        SELECT
            EXTRACT(
                YEAR
                FROM t.transaction_time
            )::integer AS year,
            COUNT(*) AS total_failed,
            COALESCE(SUM(t.amount), 0)::integer AS total_amount
        FROM transactions t
        WHERE
            t.deleted_at IS NULL
            AND t.status = 'failed'
            AND t.card_number = $1
            AND (
                EXTRACT(
                    YEAR
                    FROM t.transaction_time
                ) = $2::integer
                OR EXTRACT(
                    YEAR
                    FROM t.transaction_time
                ) = $2::integer - 1
            )
        GROUP BY
            EXTRACT(
                YEAR
                FROM t.transaction_time
            )
    ),
    formatted_data AS (
        SELECT
            year::text,
            total_failed::integer,
            total_amount::integer
        FROM yearly_data
        UNION ALL
        SELECT
            $2::text AS year,
            0::integer AS total_failed,
            0::integer AS total_amount
        WHERE
            NOT EXISTS (
                SELECT 1
                FROM yearly_data
                WHERE
                    year = $2::integer
            )
        UNION ALL
        SELECT ($2::integer - 1)::text AS year,
            0::integer AS total_failed,
            0::integer AS total_amount
        WHERE
            NOT EXISTS (
                SELECT 1
                FROM yearly_data
                WHERE
                    year = $2::integer - 1
            )
    )
SELECT *
FROM formatted_data
ORDER BY year DESC;

-- GetMonthlyPaymentMethods: Retrieves a monthly summary of transaction transactions categorized by payment method
-- Purpose:
--   Useful for visualizing how each payment method is used over time within a given year
-- Parameters:
--   $1: reference_date - Any date within the target year (used to generate monthly range)
-- Returns:
--   - month (e.g., 'Jan', 'Feb')
--   - payment_method (e.g., 'e-wallet', 'bank_transfer')
--   - total_transactions: Number of transactions for the method that month
--   - total_amount: Total amount of transactions for the method that month
-- Business Logic:
--   - Includes all combinations of months and available payment methods (even if 0 data)
--   - Excludes soft-deleted transactions (deleted_at IS NULL)
--   - Uses CROSS JOIN to ensure all months and methods are represented
-- name: GetMonthlyPaymentMethods :many
WITH
    months AS (
        SELECT generate_series(
                date_trunc('year', $1::timestamp), date_trunc('year', $1::timestamp) + interval '1 year' - interval '1 day', interval '1 month'
            ) AS month
    ),
    payment_methods AS (
        SELECT DISTINCT
            payment_method
        FROM transactions
        WHERE
            deleted_at IS NULL
    )
SELECT
    TO_CHAR(m.month, 'Mon') AS month,
    pm.payment_method,
    COALESCE(COUNT(t.transaction_id), 0)::int AS total_transactions,
    COALESCE(SUM(t.amount), 0)::int AS total_amount
FROM
    months m
    CROSS JOIN payment_methods pm
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
    AND t.payment_method = pm.payment_method
    AND t.deleted_at IS NULL
GROUP BY
    m.month,
    pm.payment_method
ORDER BY m.month, pm.payment_method;

-- GetYearlyPaymentMethods: Retrieves yearly summary of transaction transactions grouped by payment method over a 5-year span
-- Purpose:
--   Analyze long-term trends of transaction method usage across years
-- Parameters:
--   $1: current_year - The most recent year to include (covers current_year - 4 to current_year)
-- Returns:
--   - year: Year of transaction (e.g., 2020, 2021)
--   - payment_method
--   - total_transactions: Count of transactions per method per year
--   - total_amount: Sum of amounts per method per year
-- Business Logic:
--   - Filters data within a 5-year window
--   - Excludes soft-deleted transactions (deleted_at IS NULL)
-- name: GetYearlyPaymentMethods :many
SELECT
    EXTRACT(
        YEAR
        FROM t.created_at
    ) AS year,
    t.payment_method,
    COUNT(t.transaction_id) AS total_transactions,
    SUM(t.amount) AS total_amount
FROM transactions t
WHERE
    t.deleted_at IS NULL
    AND EXTRACT(
        YEAR
        FROM t.created_at
    ) >= $1 - 4
    AND EXTRACT(
        YEAR
        FROM t.created_at
    ) <= $1
GROUP BY
    EXTRACT(
        YEAR
        FROM t.created_at
    ),
    t.payment_method
ORDER BY year;

-- GetMonthlyAmounts: Retrieves total transaction amount per month for a specific year
-- Purpose:
--   Visualize monthly trends in transaction volume for charting/dashboards
-- Parameters:
--   $1: reference_date - Any date within the target year
-- Returns:
--   - month: 3-letter month abbreviation
--   - total_amount: Sum of transaction amounts in each month
-- Business Logic:
--   - Uses LEFT JOIN to ensure all months are included, even with 0 transactions
--   - Filters out soft-deleted data (deleted_at IS NULL)
-- name: GetMonthlyAmounts :many
WITH
    months AS (
        SELECT generate_series(
                date_trunc('year', $1::timestamp), date_trunc('year', $1::timestamp) + interval '1 year' - interval '1 day', interval '1 month'
            ) AS month
    )
SELECT TO_CHAR(m.month, 'Mon') AS month, COALESCE(SUM(t.amount), 0)::int AS total_amount
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
GROUP BY
    m.month
ORDER BY m.month;

-- GetYearlyAmounts: Retrieves total transaction amount per year over a 5-year span
-- Purpose:
--   Analyze annual growth or decline in transaction volume for trend analysis
-- Parameters:
--   $1: current_year - The most recent year to include (covers current_year - 4 to current_year)
-- Returns:
--   - year: Year of the transaction
--   - total_amount: Total transaction amount for the year
-- Business Logic:
--   - Excludes soft-deleted transactions (deleted_at IS NULL)
-- name: GetYearlyAmounts :many
SELECT EXTRACT(
        YEAR
        FROM t.created_at
    ) AS year, SUM(t.amount) AS total_amount
FROM transactions t
WHERE
    t.deleted_at IS NULL
    AND EXTRACT(
        YEAR
        FROM t.created_at
    ) >= $1 - 4
    AND EXTRACT(
        YEAR
        FROM t.created_at
    ) <= $1
GROUP BY
    EXTRACT(
        YEAR
        FROM t.created_at
    )
ORDER BY year;

-- GetTransactionByCardNumber: Retrieves paginated transactions for a specific card with optional filtering
-- Purpose: View transaction history for a particular card with search capability
-- Parameters:
--   $1: card_number - The card number to filter transactions (exact match)
--   $2: search_term - Optional text to filter by payment method (NULL for no filter)
--   $3: limit - Maximum number of records to return per page
--   $4: offset - Number of records to skip for pagination
-- Returns:
--   All transaction fields plus total_count of matching records
-- Business Logic:
--   - Only returns active transactions (non-deleted records)
--   - Strict matching on card_number combined with optional payment method search
--   - Case-insensitive partial matching on payment_method when search term provided
--   - Orders results by transaction_time (newest transactions first)
--   - Includes pagination metadata via total_count
--   - Useful for cardholder transaction history views and statements
-- name: GetTransactionByCardNumber :many
SELECT *, COUNT(*) OVER () AS total_count
FROM transactions
WHERE
    deleted_at IS NULL
    AND card_number = $1
    AND (
        $2::TEXT IS NULL
        OR payment_method ILIKE '%' || $2 || '%'
    )
ORDER BY transaction_time DESC
LIMIT $3
OFFSET
    $4;

-- GetMonthlyPaymentMethodsByCardNumber: Retrieves a monthly summary of transaction transactions categorized by payment method
-- Purpose:
--   Useful for visualizing how each payment method is used over time within a given year
-- Parameters:
--   $1: card_number  - filter by card_number
--   $2: reference_date - Any date within the target year (used to generate monthly range)
-- Returns:
--   - month (e.g., 'Jan', 'Feb')
--   - payment_method (e.g., 'e-wallet', 'bank_transfer')
--   - total_transactions: Number of transactions for the method that month
--   - total_amount: Total amount of transactions for the method that month
-- Business Logic:
--   - Includes all combinations of months and available payment methods (even if 0 data)
--   - Excludes soft-deleted transactions (deleted_at IS NULL)
--   - Uses CROSS JOIN to ensure all months and methods are represent
-- name: GetMonthlyPaymentMethodsByCardNumber :many
WITH
    months AS (
        SELECT generate_series(
                date_trunc('year', $2::timestamp), date_trunc('year', $2::timestamp) + interval '1 year' - interval '1 day', interval '1 month'
            ) AS month
    ),
    payment_methods AS (
        SELECT DISTINCT
            payment_method
        FROM transactions
        WHERE
            deleted_at IS NULL
    )
SELECT
    TO_CHAR(m.month, 'Mon') AS month,
    pm.payment_method,
    COALESCE(COUNT(t.transaction_id), 0)::int AS total_transactions,
    COALESCE(SUM(t.amount), 0)::int AS total_amount
FROM
    months m
    CROSS JOIN payment_methods pm
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
    AND t.payment_method = pm.payment_method
    AND t.card_number = $1
    AND t.deleted_at IS NULL
GROUP BY
    m.month,
    pm.payment_method
ORDER BY m.month, pm.payment_method;

-- GetYearlyPaymentMethodsByCardNumber: Retrieves yearly summary of transaction transactions grouped by payment method over a 5-year span
-- Purpose:
--   Analyze long-term trends of transaction method usage across years
-- Parameters:
--   $1: card_number  - filter by card_number
--   $2: current_year - The most recent year to include (covers current_year - 4 to current_year)
-- Returns:
--   - year: Year of transaction (e.g., 2020, 2021)
--   - payment_method
--   - total_transactions: Count of transactions per method per year
--   - total_amount: Sum of amounts per method per year
-- Business Logic:
--   - Filters data within a 5-year window
--   - Excludes soft-deleted transactions (deleted_at IS NULL)
-- name: GetYearlyPaymentMethodsByCardNumber :many
SELECT
    EXTRACT(
        YEAR
        FROM t.created_at
    ) AS year,
    t.payment_method,
    COUNT(t.transaction_id) AS total_transactions,
    SUM(t.amount) AS total_amount
FROM transactions t
WHERE
    t.deleted_at IS NULL
    AND t.card_number = $1
    AND EXTRACT(
        YEAR
        FROM t.created_at
    ) >= $2 - 4
    AND EXTRACT(
        YEAR
        FROM t.created_at
    ) <= $2
GROUP BY
    EXTRACT(
        YEAR
        FROM t.created_at
    ),
    t.payment_method
ORDER BY year;

-- GetMonthlyAmountsByCardNumber: Retrieves total transaction amount per month for a specific year
-- Purpose:
--   Visualize monthly trends in transaction volume for charting/dashboards
-- Parameters:
--   $1: card_number  - filter by card_number
--   $2: reference_date - Any date within the target year
-- Returns:
--   - month: 3-letter month abbreviation
--   - total_amount: Sum of transaction amounts in each month
-- Business Logic:
--   - Uses LEFT JOIN to ensure all months are included, even with 0 transactions
--   - Filters out soft-deleted data (deleted_at IS NULL)
-- name: GetMonthlyAmountsByCardNumber :many
WITH
    months AS (
        SELECT generate_series(
                date_trunc('year', $2::timestamp), date_trunc('year', $2::timestamp) + interval '1 year' - interval '1 day', interval '1 month'
            ) AS month
    )
SELECT TO_CHAR(m.month, 'Mon') AS month, COALESCE(SUM(t.amount), 0)::int AS total_amount
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
    AND t.card_number = $1
    AND t.deleted_at IS NULL
GROUP BY
    m.month
ORDER BY m.month;

-- GetYearlyAmountsByCardNumber:  Retrieves total transaction amount per year over a 5-year span
-- Purpose:
--   Analyze annual growth or decline in transaction volume for trend analysis
-- Parameters:
--   $1: card_number  - filter by card_number
--   $2: current_year - The most recent year to include (covers current_year - 4 to current_year)
-- Returns:
--   - year: Year of the transaction
--   - total_amount: Total transaction amount for the year
-- Business Logic:
--   - Excludes soft-deleted transactions (deleted_at IS NULL)
-- name: GetYearlyAmountsByCardNumber :many
SELECT EXTRACT(
        YEAR
        FROM t.created_at
    ) AS year, SUM(t.amount) AS total_amount
FROM transactions t
WHERE
    t.deleted_at IS NULL
    AND t.card_number = $1
    AND EXTRACT(
        YEAR
        FROM t.created_at
    ) >= $2 - 4
    AND EXTRACT(
        YEAR
        FROM t.created_at
    ) <= $2
GROUP BY
    EXTRACT(
        YEAR
        FROM t.created_at
    )
ORDER BY year;

-- CreateTransaction: Creates a new transaction record
-- Purpose: Record a financial transaction in the system
-- Parameters:
--   $1: card_number - The card used for the transaction
--   $2: amount - The transaction amount
--   $3: payment_method - Payment method used (e.g., 'credit', 'debit')
--   $4: merchant_id - ID of the merchant where transaction occurred
--   $5: transaction_time - Timestamp of when transaction occurred
-- Returns:
--   The newly created transaction record with all fields
-- Business Logic:
--   - Sets creation and update timestamps automatically
--   - Used for recording purchases, payments, and other financial activities
-- name: CreateTransaction :one
INSERT INTO
    transactions (
        card_number,
        amount,
        payment_method,
        merchant_id,
        transaction_time,
        created_at,
        updated_at
    )
VALUES (
        $1,
        $2,
        $3,
        $4,
        $5,
        current_timestamp,
        current_timestamp
    )
RETURNING
    transaction_id,
    transaction_no,
    card_number,
    amount,
    payment_method,
    merchant_id,
    transaction_time,
    status,
    created_at,
    updated_at;

-- UpdateTransaction: Modifies an existing transaction's details
-- Purpose: Update transaction information
-- Parameters:
--   $1: transaction_id - ID of transaction to update
--   $2: card_number - Updated card number
--   $3: amount - Updated transaction amount
--   $4: payment_method - Updated payment method
--   $5: merchant_id - Updated merchant ID
--   $6: transaction_time - Updated transaction timestamp
-- Business Logic:
--   - Only updates active transactions (non-deleted)
--   - Automatically updates the modification timestamp
--   - Used for correcting transaction details
-- name: UpdateTransaction :one
UPDATE transactions
SET
    card_number = $2,
    amount = $3,
    payment_method = $4,
    merchant_id = $5,
    transaction_time = $6,
    updated_at = current_timestamp
WHERE
    transaction_id = $1
    AND deleted_at IS NULL
RETURNING
    transaction_id,
    transaction_no,
    card_number,
    amount,
    payment_method,
    merchant_id,
    transaction_time,
    status,
    created_at,
    updated_at;

-- UpdateTransactionStatus: Changes a transaction's status
-- Purpose: Update transaction processing status
-- Parameters:
--   $1: transaction_id - ID of transaction to update
--   $2: status - New status (e.g., 'success', 'failed', 'pending')
-- Business Logic:
--   - Only updates active transactions
--   - Used to reflect transaction processing outcomes
--   - Important for reconciliation and reporting
-- name: UpdateTransactionStatus :one
UPDATE transactions
SET
    status = $2,
    updated_at = current_timestamp
WHERE
    transaction_id = $1
    AND deleted_at IS NULL
RETURNING
    transaction_id,
    transaction_no,
    card_number,
    amount,
    payment_method,
    merchant_id,
    transaction_time,
    status,
    created_at,
    updated_at;

-- TrashTransaction: Soft-deletes a transaction record
-- Purpose: Remove transaction from active use without permanent deletion
-- Parameters:
--   $1: transaction_id - ID of transaction to trash
-- Business Logic:
--   - Sets deleted_at timestamp
--   - Preserves data for audit/recovery purposes
--   - Only affects currently active records
-- name: TrashTransaction :one
UPDATE transactions
SET
    deleted_at = current_timestamp
WHERE
    transaction_id = $1
    AND deleted_at IS NULL
RETURNING
    transaction_id,
    transaction_no,
    card_number,
    amount,
    payment_method,
    merchant_id,
    transaction_time,
    status,
    created_at,
    updated_at,
    deleted_at;

-- RestoreTransaction: Recovers a soft-deleted transaction
-- Purpose: Reactivate a previously trashed transaction
-- Parameters:
--   $1: transaction_id - ID of transaction to restore
-- Business Logic:
--   - Clears the deleted_at timestamp
--   - Only works on currently trashed records
--   - Used for data recovery purposes
-- name: RestoreTransaction :one
UPDATE transactions
SET
    deleted_at = NULL
WHERE
    transaction_id = $1
    AND deleted_at IS NOT NULL
RETURNING
    transaction_id,
    transaction_no,
    card_number,
    amount,
    payment_method,
    merchant_id,
    transaction_time,
    status,
    created_at,
    updated_at,
    deleted_at;

-- DeleteTransactionPermanently: Hard-deletes a trashed transaction
-- Purpose: Permanently remove a transaction from the system
-- Parameters:
--   $1: transaction_id - ID of transaction to delete
-- Business Logic:
--   - Physical deletion from database
--   - Only works on already trashed records
--   - Irreversible operation
--   - Used for data cleanup after retention period
-- name: DeleteTransactionPermanently :exec
DELETE FROM transactions
WHERE
    transaction_id = $1
    AND deleted_at IS NOT NULL;

-- RestoreAllTransactions: Recovers all trashed transactions
-- Purpose: Mass restoration of deleted transactions
-- Business Logic:
--   - Clears deleted_at for all trashed records
--   - Useful for system recovery scenarios
--   - Should be used cautiously in production
-- name: RestoreAllTransactions :exec
UPDATE transactions
SET
    deleted_at = NULL
WHERE
    deleted_at IS NOT NULL;

-- DeleteAllPermanentTransactions: Permanently removes all trashed transactions
-- Purpose: Clean up all soft-deleted transaction records
-- Business Logic:
--   - Irreversible bulk deletion
--   - Only affects records marked as deleted
--   - Frees database space from old records
--   - Typically used during maintenance periods
-- name: DeleteAllPermanentTransactions :exec
DELETE FROM transactions WHERE deleted_at IS NOT NULL;