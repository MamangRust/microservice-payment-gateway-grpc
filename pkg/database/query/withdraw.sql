-- GetWithdraws: Retrieves paginated withdrawal records with search capability
-- Purpose: List all withdrawals for management UI with filtering options
-- Parameters:
--   $1: search_term - Optional text to filter withdrawals by various fields (NULL for no filter)
--   $2: limit - Maximum number of records to return per page
--   $3: offset - Number of records to skip for pagination
-- Returns:
--   All withdrawal fields plus total_count of matching records
-- Business Logic:
--   - Excludes soft-deleted withdrawals (deleted_at IS NULL)
--   - Supports partial text matching on multiple fields (case-insensitive):
--     * card_number
--     * withdraw_amount (converted to text)
--     * withdraw_time (converted to text)
--     * status
--   - Orders by withdraw_time (newest withdrawals first)
--   - Provides total_count for pagination calculations
--   - Useful for withdrawal monitoring and auditing
-- name: GetWithdraws :many
SELECT
    withdraw_id,
    withdraw_no,
    card_number,
    withdraw_amount,
    withdraw_time,
    status,
    created_at,
    updated_at,
    COUNT(*) OVER () AS total_count
FROM withdraws
WHERE
    deleted_at IS NULL
    AND (
        $1::TEXT IS NULL
        OR card_number ILIKE '%' || $1 || '%'
        OR withdraw_amount::TEXT ILIKE '%' || $1 || '%'
        OR withdraw_time::TEXT ILIKE '%' || $1 || '%'
        OR status ILIKE '%' || $1 || '%'
    )
ORDER BY withdraw_time DESC
LIMIT $2
OFFSET
    $3;

-- GetActiveWithdraws: Retrieves paginated active withdrawals with search
-- Purpose: List all non-deleted withdrawals with filtering options
-- Parameters:
--   $1: search_term - Optional text to filter withdrawals
--   $2: limit - Maximum records to return per page
--   $3: offset - Records to skip for pagination
-- Returns:
--   All withdrawal fields plus total_count of matching active records
-- Business Logic:
--   - Only includes active withdrawals (deleted_at IS NULL)
--   - Same comprehensive filtering as GetWithdraws
--   - Orders by withdraw_time (newest first)
--   - Provides pagination metadata
--   - Used in withdrawal management interfaces
-- name: GetActiveWithdraws :many
SELECT
    withdraw_id,
    withdraw_no,
    card_number,
    withdraw_amount,
    withdraw_time,
    status,
    created_at,
    updated_at,
    deleted_at,
    COUNT(*) OVER () AS total_count
FROM withdraws
WHERE
    deleted_at IS NULL
    AND (
        $1::TEXT IS NULL
        OR card_number ILIKE '%' || $1 || '%'
        OR withdraw_amount::TEXT ILIKE '%' || $1 || '%'
        OR withdraw_time::TEXT ILIKE '%' || $1 || '%'
        OR status ILIKE '%' || $1 || '%'
    )
ORDER BY withdraw_time DESC
LIMIT $2
OFFSET
    $3;

-- GetTrashedWithdraws: Retrieves paginated soft-deleted withdrawals
-- Purpose: List all deleted withdrawals for recovery or audit purposes
-- Parameters:
--   $1: search_term - Optional text to filter deleted withdrawals
--   $2: limit - Maximum records to return per page
--   $3: offset - Records to skip for pagination
-- Returns:
--   All withdrawal fields plus total_count of matching deleted records
-- Business Logic:
--   - Only includes soft-deleted withdrawals (deleted_at IS NOT NULL)
--   - Same filtering capabilities as active withdrawals
--   - Maintains newest-first ordering
--   - Used in admin interfaces for withdrawal recovery
-- name: GetTrashedWithdraws :many
SELECT
    withdraw_id,
    withdraw_no,
    card_number,
    withdraw_amount,
    withdraw_time,
    status,
    created_at,
    updated_at,
    deleted_at,
    COUNT(*) OVER () AS total_count
FROM withdraws
WHERE
    deleted_at IS NOT NULL
    AND (
        $1::TEXT IS NULL
        OR card_number ILIKE '%' || $1 || '%'
        OR withdraw_amount::TEXT ILIKE '%' || $1 || '%'
        OR withdraw_time::TEXT ILIKE '%' || $1 || '%'
        OR status ILIKE '%' || $1 || '%'
    )
ORDER BY withdraw_time DESC
LIMIT $2
OFFSET
    $3;

-- GetWithdrawByID: Retrieves a single withdrawal by its ID
-- Purpose: Get detailed information about a specific withdrawal
-- Parameters:
--   $1: withdraw_id - The ID of the withdrawal to retrieve
-- Returns:
--   All fields for the specified withdrawal or NULL if not found/deleted
-- Business Logic:
--   - Only returns active withdrawals (deleted_at IS NULL)
--   - Useful for withdrawal details viewing and verification
-- name: GetWithdrawByID :one
SELECT
    withdraw_id,
    withdraw_no,
    card_number,
    withdraw_amount,
    withdraw_time,
    status,
    created_at,
    updated_at
FROM withdraws
WHERE
    withdraw_id = $1
    AND deleted_at IS NULL;

-- GetWithdrawsByCardNumber: Retrieves paginated withdrawals for a specific card with search
-- Purpose: List all withdrawals associated with a particular card
-- Parameters:
--   $1: card_number - The card number to filter withdrawals
--   $2: search_term - Optional text to filter by amount, time, or status
--   $3: limit - Maximum number of records to return per page
--   $4: offset - Number of records to skip for pagination
-- Returns:
--   All withdrawal fields plus total_count of matching records
-- Business Logic:
--   - Only includes active withdrawals (deleted_at IS NULL)
--   - Strict card number matching combined with optional search filters:
--     * withdraw_amount (converted to text for searching)
--     * withdraw_time (formatted as string for searching)
--     * status
--   - Orders by withdraw_time (newest first)
--   - Provides pagination support with total_count
--   - Useful for cardholder withdrawal history
-- name: GetWithdrawsByCardNumber :many
SELECT
    withdraw_id,
    withdraw_no,
    card_number,
    withdraw_amount,
    withdraw_time,
    status,
    created_at,
    updated_at,
    COUNT(*) OVER () AS total_count
FROM withdraws
WHERE
    deleted_at IS NULL
    AND card_number = $1
    AND (
        $2::TEXT IS NULL
        OR CAST(withdraw_amount AS TEXT) ILIKE '%' || $2 || '%'
        OR TO_CHAR(
            withdraw_time,
            'YYYY-MM-DD HH24:MI:SS'
        ) ILIKE '%' || $2 || '%'
        OR status ILIKE '%' || $2 || '%'
    )
ORDER BY withdraw_time DESC
LIMIT $3
OFFSET
    $4;

-- GetTrashedWithdrawByID: Retrieves a single soft-deleted withdrawal by ID
-- Purpose: View details of a deleted withdrawal for recovery or audit
-- Parameters:
--   $1: withdraw_id - The ID of the withdrawal to retrieve
-- Returns:
--   All fields for the specified trashed withdrawal or NULL if not found/active
-- Business Logic:
--   - Only returns soft-deleted withdrawals (deleted_at IS NOT NULL)
--   - Used in admin interfaces for withdrawal recovery
-- name: GetTrashedWithdrawByID :one
SELECT *
FROM withdraws
WHERE
    withdraw_id = $1
    AND deleted_at IS NOT NULL;

-- FindAllWithdrawsByCardNumber: Retrieves all withdrawals for a specific card
-- Purpose: Get complete withdrawal history for a card
-- Parameters:
--   $1: card_number - The card number to filter withdrawals
-- Returns:
--   Selected withdrawal fields for all matching records
-- Business Logic:
--   - Only includes active withdrawals (deleted_at IS NULL)
--   - Returns all withdrawals without pagination
--   - Orders by withdraw_time (newest first)
--   - Useful for complete withdrawal history exports
-- name: FindAllWithdrawsByCardNumber :many
SELECT w.withdraw_id, w.card_number, w.withdraw_amount, w.withdraw_time, w.created_at, w.updated_at, w.deleted_at
FROM withdraws w
WHERE
    w.card_number = $1
    AND w.deleted_at IS NULL
ORDER BY w.withdraw_time DESC;

-- GetMonthWithdrawStatusSuccess: Retrieves monthly success metrics for withdrawals
-- Purpose: Analyze successful withdrawal trends across comparison periods
-- Parameters:
--   $1: period1_start - Start date of first comparison period (timestamp)
--   $2: period1_end - End date of first comparison period (timestamp)
--   $3: period2_start - Start date of second comparison period (timestamp)
--   $4: period2_end - End date of second comparison period (timestamp)
-- Returns:
--   year: Year as text (e.g., '2023')
--   month: 3-letter month abbreviation (e.g., 'Jan')
--   total_success: Count of successful withdrawals
--   total_amount: Sum of successful withdrawal amounts
-- Business Logic:
--   - Only includes successful withdrawals (status = 'success')
--   - Covers two customizable time periods for comparison
--   - Zero-fills months with no withdrawal activity
--   - Formats output for consistent visualization (year as text, month as 'Mon')
--   - Orders by year and month (newest first)
--   - Useful for identifying seasonal cash withdrawal patterns
-- name: GetMonthWithdrawStatusSuccess :many
WITH
    monthly_data AS (
        SELECT
            EXTRACT(
                YEAR
                FROM t.withdraw_time
            )::integer AS year,
            EXTRACT(
                MONTH
                FROM t.withdraw_time
            )::integer AS month,
            COUNT(*) AS total_success,
            COALESCE(SUM(t.withdraw_amount), 0)::integer AS total_amount
        FROM withdraws t
        WHERE
            t.deleted_at IS NULL
            AND t.status = 'success'
            AND (
                (
                    t.withdraw_time >= $1::timestamp
                    AND t.withdraw_time <= $2::timestamp
                )
                OR (
                    t.withdraw_time >= $3::timestamp
                    AND t.withdraw_time <= $4::timestamp
                )
            )
        GROUP BY
            EXTRACT(
                YEAR
                FROM t.withdraw_time
            ),
            EXTRACT(
                MONTH
                FROM t.withdraw_time
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

-- GetYearlyWithdrawStatusSuccess: Retrieves yearly success metrics for withdrawals
-- Purpose: Compare annual successful withdrawal performance year-over-year
-- Parameters:
--   $1: current_year - The target year (includes this year and previous)
-- Returns:
--   year: Year as text (e.g., '2023')
--   total_success: Count of successful withdrawals
--   total_amount: Sum of successful withdrawal amounts
-- Business Logic:
--   - Only includes successful withdrawals (status = 'success')
--   - Compares current year with previous year
--   - Zero-fills years with no withdrawal activity
--   - Orders by year (newest first)
--   - Useful for year-over-year cash flow analysis
--   - Helps identify annual withdrawal patterns and liquidity trends
-- name: GetYearlyWithdrawStatusSuccess :many
WITH
    yearly_data AS (
        SELECT
            EXTRACT(
                YEAR
                FROM t.withdraw_time
            )::integer AS year,
            COUNT(*) AS total_success,
            COALESCE(SUM(t.withdraw_amount), 0)::integer AS total_amount
        FROM withdraws t
        WHERE
            t.deleted_at IS NULL
            AND t.status = 'success'
            AND (
                EXTRACT(
                    YEAR
                    FROM t.withdraw_time
                ) = $1::integer
                OR EXTRACT(
                    YEAR
                    FROM t.withdraw_time
                ) = $1::integer - 1
            )
        GROUP BY
            EXTRACT(
                YEAR
                FROM t.withdraw_time
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

-- GetMonthWithdrawStatusFailed: Retrieves monthly failed metrics for withdrawals
-- Purpose: Analyze failed withdrawal trends across comparison periods
-- Parameters:
--   $1: period1_start - Start date of first comparison period (timestamp)
--   $2: period1_end - End date of first comparison period (timestamp)
--   $3: period2_start - Start date of second comparison period (timestamp)
--   $4: period2_end - End date of second comparison period (timestamp)
-- Returns:
--   year: Year as text (e.g., '2023')
--   month: 3-letter month abbreviation (e.g., 'Jan')
--   total_failed: Count of failed withdrawals
--   total_amount: Sum of failed withdrawal amounts
-- Business Logic:
--   - Only includes failed withdrawals (status = 'failed')
--   - Covers two customizable time periods for comparison
--   - Zero-fills months with no withdrawal activity
--   - Formats output for consistent visualization (year as text, month as 'Mon')
--   - Orders by year and month (newest first)
--   - Useful for identifying seasonal cash withdrawal patterns
-- name: GetMonthWithdrawStatusFailed :many
WITH
    monthly_data AS (
        SELECT
            EXTRACT(
                YEAR
                FROM t.withdraw_time
            )::integer AS year,
            EXTRACT(
                MONTH
                FROM t.withdraw_time
            )::integer AS month,
            COUNT(*) AS total_failed,
            COALESCE(SUM(t.withdraw_amount), 0)::integer AS total_amount
        FROM withdraws t
        WHERE
            t.deleted_at IS NULL
            AND t.status = 'failed'
            AND (
                (
                    t.withdraw_time >= $1::timestamp
                    AND t.withdraw_time <= $2::timestamp
                )
                OR (
                    t.withdraw_time >= $3::timestamp
                    AND t.withdraw_time <= $4::timestamp
                )
            )
        GROUP BY
            EXTRACT(
                YEAR
                FROM t.withdraw_time
            ),
            EXTRACT(
                MONTH
                FROM t.withdraw_time
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

-- GetYearlyWithdrawStatusFailed: Retrieves yearly failed metrics for withdrawals
-- Purpose: Compare annual failedful withdrawal performance year-over-year
-- Parameters:
--   $1: current_year - The target year (includes this year and previous)
-- Returns:
--   year: Year as text (e.g., '2023')
--   total_failed: Count of failedful withdrawals
--   total_amount: Sum of failedful withdrawal amounts
-- Business Logic:
--   - Only includes failedful withdrawals (status = 'failed')
--   - Compares current year with previous year
--   - Zero-fills years with no withdrawal activity
--   - Orders by year (newest first)
--   - Useful for year-over-year cash flow analysis
--   - Helps identify annual withdrawal patterns and liquidity trends
-- name: GetYearlyWithdrawStatusFailed :many
WITH
    yearly_data AS (
        SELECT
            EXTRACT(
                YEAR
                FROM t.withdraw_time
            )::integer AS year,
            COUNT(*) AS total_failed,
            COALESCE(SUM(t.withdraw_amount), 0)::integer AS total_amount
        FROM withdraws t
        WHERE
            t.deleted_at IS NULL
            AND t.status = 'failed'
            AND (
                EXTRACT(
                    YEAR
                    FROM t.withdraw_time
                ) = $1::integer
                OR EXTRACT(
                    YEAR
                    FROM t.withdraw_time
                ) = $1::integer - 1
            )
        GROUP BY
            EXTRACT(
                YEAR
                FROM t.withdraw_time
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

-- name: GetMonthWithdrawStatusSuccessCardNumber :many
WITH
    monthly_data AS (
        SELECT
            EXTRACT(
                YEAR
                FROM t.withdraw_time
            )::integer AS year,
            EXTRACT(
                MONTH
                FROM t.withdraw_time
            )::integer AS month,
            COUNT(*) AS total_success,
            COALESCE(SUM(t.withdraw_amount), 0)::integer AS total_amount
        FROM withdraws t
        WHERE
            t.deleted_at IS NULL
            AND t.status = 'success'
            AND t.card_number = $1
            AND (
                (
                    t.withdraw_time >= $2::timestamp
                    AND t.withdraw_time <= $3::timestamp
                )
                OR (
                    t.withdraw_time >= $4::timestamp
                    AND t.withdraw_time <= $5::timestamp
                )
            )
        GROUP BY
            EXTRACT(
                YEAR
                FROM t.withdraw_time
            ),
            EXTRACT(
                MONTH
                FROM t.withdraw_time
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

-- name: GetYearlyWithdrawStatusSuccessCardNumber :many
WITH
    yearly_data AS (
        SELECT
            EXTRACT(
                YEAR
                FROM t.withdraw_time
            )::integer AS year,
            COUNT(*) AS total_success,
            COALESCE(SUM(t.withdraw_amount), 0)::integer AS total_amount
        FROM withdraws t
        WHERE
            t.deleted_at IS NULL
            AND t.status = 'success'
            AND t.card_number = $1
            AND (
                EXTRACT(
                    YEAR
                    FROM t.withdraw_time
                ) = $2::integer
                OR EXTRACT(
                    YEAR
                    FROM t.withdraw_time
                ) = $2::integer - 1
            )
        GROUP BY
            EXTRACT(
                YEAR
                FROM t.withdraw_time
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

-- GetMonthWithdrawStatusFailedCardNumber: Retrieves monthly failed metrics for withdrawals
-- Purpose: Analyze failed withdrawal trends across comparison periods
-- Parameters:
--   $1: card_number  - filter by card_number
--   $2: period1_start - Start date of first comparison period (timestamp)
--   $3: period1_end - End date of first comparison period (timestamp)
--   $4: period2_start - Start date of second comparison period (timestamp)
--   $5: period2_end - End date of second comparison period (timestamp)
-- Returns:
--   year: Year as text (e.g., '2023')
--   month: 3-letter month abbreviation (e.g., 'Jan')
--   total_failed: Count of failed withdrawals
--   total_amount: Sum of failed withdrawal amounts
-- Business Logic:
--   - Only includes failed withdrawals (status = 'failed')
--   - Covers two customizable time periods for comparison
--   - Zero-fills months with no withdrawal activity
--   - Formats output for consistent visualization (year as text, month as 'Mon')
--   - Orders by year and month (newest first)
--   - Useful for identifying seasonal cash withdrawal patterns
-- name: GetMonthWithdrawStatusFailedCardNumber :many
WITH
    monthly_data AS (
        SELECT
            EXTRACT(
                YEAR
                FROM t.withdraw_time
            )::integer AS year,
            EXTRACT(
                MONTH
                FROM t.withdraw_time
            )::integer AS month,
            COUNT(*) AS total_failed,
            COALESCE(SUM(t.withdraw_amount), 0)::integer AS total_amount
        FROM withdraws t
        WHERE
            t.deleted_at IS NULL
            AND t.status = 'failed'
            AND t.card_number = $1
            AND (
                (
                    t.withdraw_time >= $2::timestamp
                    AND t.withdraw_time <= $3::timestamp
                )
                OR (
                    t.withdraw_time >= $4::timestamp
                    AND t.withdraw_time <= $5::timestamp
                )
            )
        GROUP BY
            EXTRACT(
                YEAR
                FROM t.withdraw_time
            ),
            EXTRACT(
                MONTH
                FROM t.withdraw_time
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

-- GetYearlyWithdrawStatusFailedCardNumber: Retrieves yearly failed metrics for withdrawals
-- Purpose: Compare annual failedful withdrawal performance year-over-year
-- Parameters:
--   $1: card_number  - filter by card_number
--   $2: current_year - The target year (includes this year and previous)
-- Returns:
--   year: Year as text (e.g., '2023')
--   total_failed: Count of failedful withdrawals
--   total_amount: Sum of failedful withdrawal amounts
-- Business Logic:
--   - Only includes failedful withdrawals (status = 'failed')
--   - Compares current year with previous year
--   - Zero-fills years with no withdrawal activity
--   - Orders by year (newest first)
--   - Useful for year-over-year cash flow analysis
--   - Helps identify annual withdrawal patterns and liquidity trends
-- name: GetYearlyWithdrawStatusFailedCardNumber :many
WITH
    yearly_data AS (
        SELECT
            EXTRACT(
                YEAR
                FROM t.withdraw_time
            )::integer AS year,
            COUNT(*) AS total_failed,
            COALESCE(SUM(t.withdraw_amount), 0)::integer AS total_amount
        FROM withdraws t
        WHERE
            t.deleted_at IS NULL
            AND t.status = 'failed'
            AND t.card_number = $1
            AND (
                EXTRACT(
                    YEAR
                    FROM t.withdraw_time
                ) = $2::integer
                OR EXTRACT(
                    YEAR
                    FROM t.withdraw_time
                ) = $2::integer - 1
            )
        GROUP BY
            EXTRACT(
                YEAR
                FROM t.withdraw_time
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

-- GetMonthlyWithdraws: Retrieves monthly withdrawal totals for a given year
-- Purpose: Analyze monthly cash withdrawal patterns and trends
-- Parameters:
--   $1: reference_date - Any date within the target year (used to define the year range)
-- Returns:
--   month: 3-letter month abbreviation (e.g., 'Jan')
--   total_withdraw_amount: Sum of withdrawal amounts for that month (0 if no withdrawals)
-- Business Logic:
--   - Generates complete monthly series for the specified year
--   - Includes all withdrawals regardless of card or status
--   - Filters out soft-deleted records (deleted_at IS NULL)
--   - Zero-fills months with no withdrawal activity
--   - Orders results chronologically by month
--   - Useful for cash flow analysis and ATM/branch planning
-- name: GetMonthlyWithdraws :many
WITH
    months AS (
        SELECT generate_series(
                date_trunc('year', $1::timestamp), date_trunc('year', $1::timestamp) + interval '1 year' - interval '1 day', interval '1 month'
            ) AS month
    )
SELECT TO_CHAR(m.month, 'Mon') AS month, COALESCE(SUM(w.withdraw_amount), 0)::int AS total_withdraw_amount
FROM months m
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
GROUP BY
    m.month
ORDER BY m.month;

-- GetYearlyWithdraws: Retrieves yearly withdrawal totals for a 5-year period
-- Purpose: Analyze long-term withdrawal trends
-- Parameters:
--   $1: current_year - The final year to include (includes this year and previous 4 years)
-- Returns:
--   year: 4-digit year
--   total_withdraw_amount: Sum of withdrawal amounts for that year
-- Business Logic:
--   - Covers a 5-year rolling window (current_year-4 to current_year)
--   - Only includes active withdrawal records
--   - Groups by calendar year
--   - Orders results chronologically
--   - Useful for identifying annual cash usage patterns
-- name: GetYearlyWithdraws :many
SELECT EXTRACT(
        YEAR
        FROM w.withdraw_time
    ) AS year, SUM(w.withdraw_amount) AS total_withdraw_amount
FROM withdraws w
WHERE
    w.deleted_at IS NULL
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
ORDER BY year;

-- GetMonthlyWithdrawsByCardNumber: Retrieves monthly withdrawals for a specific card
-- Purpose: Track monthly cash usage patterns for individual cardholders
-- Parameters:
--   $1: card_number - The card number to filter withdrawals
--   $2: reference_date - Any date within the target year
-- Returns:
--   month: 3-letter month abbreviation
--   total_withdraw_amount: Sum of withdrawals for that card by month
-- Business Logic:
--   - Generates complete monthly series for the year
--   - Filters withdrawals by specific card number
--   - Zero-fills months with no activity for that card
--   - Orders chronologically
--   - Useful for individual spending pattern analysis
-- name: GetMonthlyWithdrawsByCardNumber :many
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
    AND w.card_number = $1
    AND w.deleted_at IS NULL
GROUP BY
    m.month
ORDER BY m.month;

-- GetYearlyWithdrawsByCardNumber: Retrieves yearly withdrawals for a specific card
-- Purpose: Analyze long-term cash usage for individual cardholders
-- Parameters:
--   $1: card_number - The card number to filter withdrawals
--   $2: current_year - The final year to include (5-year window)
-- Returns:
--   year: 4-digit year
--   total_withdraw_amount: Sum of withdrawals for that card by year
-- Business Logic:
--   - Covers 5-year period (current_year-4 to current_year)
--   - Filters by specific card number
--   - Only includes active withdrawals
--   - Groups by calendar year
--   - Orders chronologically
--   - Useful for customer spending habit analysis
-- name: GetYearlyWithdrawsByCardNumber :many
SELECT EXTRACT(
        YEAR
        FROM w.created_at
    ) AS year, SUM(w.withdraw_amount) AS total_withdraw_amount
FROM withdraws w
WHERE
    w.deleted_at IS NULL
    AND w.card_number = $1
    AND EXTRACT(
        YEAR
        FROM w.created_at
    ) >= $2 - 4
    AND EXTRACT(
        YEAR
        FROM w.created_at
    ) <= $2
GROUP BY
    EXTRACT(
        YEAR
        FROM w.created_at
    )
ORDER BY year;

-- CreateWithdraw: Records a new cash withdrawal
-- Purpose: Create a withdrawal transaction in the system
-- Parameters:
--   $1: card_number - The card used for withdrawal
--   $2: withdraw_amount - The amount withdrawn
--   $3: withdraw_time - When the withdrawal occurred
--   $4: created_at - Timestamp of record creation
-- Returns:
--   The newly created withdrawal record with all fields
-- Business Logic:
--   - Sets creation and update timestamps automatically
--   - Used for recording ATM/branch cash withdrawals
--   - Typically triggered after successful cash dispense
-- name: CreateWithdraw :one
INSERT INTO
    withdraws (
        card_number,
        withdraw_amount,
        withdraw_time,
        created_at,
        updated_at
    )
VALUES (
        $1,
        $2,
        $3,
        $4,
        current_timestamp
    )
RETURNING
    withdraw_id,
    withdraw_no,
    card_number,
    withdraw_amount,
    withdraw_time,
    status,
    created_at,
    updated_at;

-- UpdateWithdraw: Modifies withdrawal details
-- Purpose: Update withdrawal information
-- Parameters:
--   $1: withdraw_id - ID of withdrawal to update
--   $2: card_number - Updated card number
--   $3: withdraw_amount - Updated withdrawal amount
--   $4: withdraw_time - Updated withdrawal timestamp
-- Business Logic:
--   - Only updates active withdrawals (non-deleted)
--   - Updates modification timestamp automatically
--   - Used for correcting withdrawal records
--   - Requires original withdrawal record exists
-- name: UpdateWithdraw :one
UPDATE withdraws
SET
    card_number = $2,
    withdraw_amount = $3,
    withdraw_time = $4,
    updated_at = current_timestamp
WHERE
    withdraw_id = $1
    AND deleted_at IS NULL
RETURNING
    withdraw_id,
    withdraw_no,
    card_number,
    withdraw_amount,
    withdraw_time,
    status,
    created_at,
    updated_at;

-- UpdateWithdrawStatus: Changes withdrawal status
-- Purpose: Update processing status of a withdrawal
-- Parameters:
--   $1: withdraw_id - ID of withdrawal to update
--   $2: status - New status (e.g., 'completed', 'failed', 'pending')
-- Business Logic:
--   - Only updates active withdrawals
--   - Updates modification timestamp
--   - Used to reflect withdrawal processing outcomes
--   - Important for reconciliation purposes
-- name: UpdateWithdrawStatus :one
UPDATE withdraws
SET
    status = $2,
    updated_at = current_timestamp
WHERE
    withdraw_id = $1
    AND deleted_at IS NULL
RETURNING
    withdraw_id,
    withdraw_no,
    card_number,
    withdraw_amount,
    withdraw_time,
    status,
    created_at,
    updated_at;

-- TrashWithdraw: Soft-deletes a withdrawal record
-- Purpose: Remove withdrawal from active view without permanent deletion
-- Parameters:
--   $1: withdraw_id - ID of withdrawal to trash
-- Business Logic:
--   - Sets deleted_at timestamp to current time
--   - Only affects active withdrawals
--   - Preserves data for audit/recovery purposes
--   - Withdrawal remains in database but hidden
-- name: TrashWithdraw :one
UPDATE withdraws
SET
    deleted_at = current_timestamp
WHERE
    withdraw_id = $1
    AND deleted_at IS NULL
RETURNING
    withdraw_id,
    withdraw_no,
    card_number,
    withdraw_amount,
    withdraw_time,
    status,
    created_at,
    updated_at,
    deleted_at;

-- RestoreWithdraw: Recovers a soft-deleted withdrawal
-- Purpose: Reactivate a previously deleted withdrawal
-- Parameters:
--   $1: withdraw_id - ID of withdrawal to restore
-- Business Logic:
--   - Clears the deleted_at timestamp (sets to NULL)
--   - Only works on currently trashed withdrawals
--   - Used for data recovery purposes
-- name: RestoreWithdraw :one
UPDATE withdraws
SET
    deleted_at = NULL
WHERE
    withdraw_id = $1
    AND deleted_at IS NOT NULL
RETURNING
    withdraw_id,
    withdraw_no,
    card_number,
    withdraw_amount,
    withdraw_time,
    status,
    created_at,
    updated_at,
    deleted_at;

-- DeleteWithdrawPermanently: Hard-deletes a withdrawal
-- Purpose: Permanently remove a withdrawal from the system
-- Parameters:
--   $1: withdraw_id - ID of withdrawal to delete
-- Business Logic:
--   - Physical deletion from database
--   - Only works on already trashed withdrawals
--   - Irreversible operation
--   - Used after retention period expires
-- name: DeleteWithdrawPermanently :exec
DELETE FROM withdraws
WHERE
    withdraw_id = $1
    AND deleted_at IS NOT NULL;

-- RestoreAllWithdraws: Recovers all trashed withdrawals
-- Purpose: Mass restoration of deleted withdrawals
-- Business Logic:
--   - Clears deleted_at for all trashed withdrawals
--   - Useful for system recovery scenarios
--   - Should be used cautiously in production
--   - Admin-level operation
-- name: RestoreAllWithdraws :exec
UPDATE withdraws
SET
    deleted_at = NULL
WHERE
    deleted_at IS NOT NULL;

-- DeleteAllPermanentWithdraws: Permanently removes all trashed withdrawals
-- Purpose: Clean up all soft-deleted withdrawal records
-- Business Logic:
--   - Irreversible bulk deletion
--   - Only affects records marked as deleted
--   - Frees database space from old records
--   - Typically used during maintenance periods
--   - Requires admin privileges
-- name: DeleteAllPermanentWithdraws :exec
DELETE FROM withdraws WHERE deleted_at IS NOT NULL;