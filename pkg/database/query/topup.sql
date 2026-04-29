-- GetTopups: Retrieves paginated list of active topups with search capability
-- Purpose: Provide admin or user access to topup history with search support
-- Parameters:
--   $1: search_term - Optional filter to match card_number, topup_no, topup_method, or status (NULL for no filter)
--   $2: limit - Max number of records to return
--   $3: offset - Records to skip for pagination
-- Returns:
--   All topup fields and total_count of matching records
-- Business Logic:
--   - Filters out soft-deleted topups (deleted_at IS NULL)
--   - Supports partial, case-insensitive search across multiple fields
--   - Results sorted by topup_time (most recent first)
--   - total_count is used for frontend pagination
-- name: GetTopups :many
SELECT
    topup_id,
    topup_no,
    card_number,
    topup_amount,
    topup_method,
    topup_time,
    status,
    created_at,
    updated_at,
    COUNT(*) OVER () AS total_count
FROM topups
WHERE
    deleted_at IS NULL
    AND (
        $1::TEXT IS NULL
        OR card_number ILIKE '%' || $1 || '%'
        OR topup_no::TEXT ILIKE '%' || $1 || '%'
        OR topup_method ILIKE '%' || $1 || '%'
        OR status ILIKE '%' || $1 || '%'
    )
ORDER BY topup_time DESC
LIMIT $2
OFFSET
    $3;

-- GetActiveTopups: Retrieves paginated list of active (non-deleted) topups with search
-- Purpose: Display only active topups for admin or user dashboards
-- Parameters:
--   $1: search_term - Optional text to filter by card_number, topup_no, or topup_method
--   $2: limit - Max records to return
--   $3: offset - Number of rows to skip
-- Returns:
--   All active topup fields and total_count
-- Business Logic:
--   - Filters out soft-deleted topups (deleted_at IS NULL)
--   - Supports partial, case-insensitive search across multiple fields
--   - Results sorted by topup_time (most recent first)
--   - total_count is used for frontend pagination
-- name: GetActiveTopups :many
SELECT
    topup_id,
    topup_no,
    card_number,
    topup_amount,
    topup_method,
    topup_time,
    status,
    created_at,
    updated_at,
    deleted_at,
    COUNT(*) OVER () AS total_count
FROM topups
WHERE
    deleted_at IS NULL
    AND (
        $1::TEXT IS NULL
        OR card_number ILIKE '%' || $1 || '%'
        OR topup_no::TEXT ILIKE '%' || $1 || '%'
        OR topup_method ILIKE '%' || $1 || '%'
    )
ORDER BY topup_time DESC
LIMIT $2
OFFSET
    $3;

-- GetTrashedTopups: Retrieves trashed (soft-deleted) topups with pagination and search
-- Purpose: Allow recovery or permanent deletion of topups
-- Parameters:
--   $1: search_term - Optional filter to match card_number, topup_no, or topup_method
--   $2: limit - Max records to return
--   $3: offset - Rows to skip
-- Returns:
--   Trashed topup records with total_count
-- Business Logic:
--   - Only includes topups where deleted_at IS NOT NULL
--   - Supports flexible search
--   - Results sorted by topup_time descending
-- name: GetTrashedTopups :many
SELECT
    topup_id,
    topup_no,
    card_number,
    topup_amount,
    topup_method,
    topup_time,
    status,
    created_at,
    updated_at,
    deleted_at,
    COUNT(*) OVER () AS total_count
FROM topups
WHERE
    deleted_at IS NOT NULL
    AND (
        $1::TEXT IS NULL
        OR card_number ILIKE '%' || $1 || '%'
        OR topup_no::TEXT ILIKE '%' || $1 || '%'
        OR topup_method ILIKE '%' || $1 || '%'
    )
ORDER BY topup_time DESC
LIMIT $2
OFFSET
    $3;

-- GetTopupByID: Retrieves a specific topup by ID
-- Purpose: Used to display details of a single topup transaction
-- Parameters:
--   $1: topup_id - Unique identifier of the topup
-- Returns:
--   Topup record matching the ID (if not soft-deleted)
-- Business Logic:
--   - Only returns record if it is active (deleted_at IS NULL)
-- name: GetTopupByID :one
SELECT
    topup_id,
    topup_no,
    card_number,
    topup_amount,
    topup_method,
    topup_time,
    status,
    created_at,
    updated_at
FROM topups
WHERE
    topup_id = $1
    AND deleted_at IS NULL;

-- GetMonthTopupStatusSuccess: Retrieves monthly success metrics for topups
-- Purpose: Analyze successful topup trends across comparison periods
-- Parameters:
--   $1: period1_start - Start date of first comparison period
--   $2: period1_end - End date of first comparison period
--   $3: period2_start - Start date of second comparison period
--   $4: period2_end - End date of second comparison period
-- Returns:
--   year: Year as text (e.g., '2023')
--   month: 3-letter month abbreviation (e.g., 'Jan')
--   total_success: Count of successful topups
--   total_amount: Sum of successful topup amounts
-- Business Logic:
--   - Only includes successful topups (status = 'success')
--   - Covers two customizable time periods for comparison
--   - Zero-fills months with no activity
--   - Formats output for consistent visualization
--   - Orders by year and month (newest first)
--   - Useful for identifying seasonal topup patterns
-- name: GetMonthTopupStatusSuccess :many
WITH
    monthly_data AS (
        SELECT
            EXTRACT(
                YEAR
                FROM t.topup_time
            )::integer AS year,
            EXTRACT(
                MONTH
                FROM t.topup_time
            )::integer AS month,
            COUNT(*) AS total_success,
            COALESCE(SUM(t.topup_amount), 0)::integer AS total_amount
        FROM topups t
        WHERE
            t.deleted_at IS NULL
            AND t.status = 'success'
            AND (
                (
                    t.topup_time >= $1::timestamp
                    AND t.topup_time <= $2::timestamp
                )
                OR (
                    t.topup_time >= $3::timestamp
                    AND t.topup_time <= $4::timestamp
                )
            )
        GROUP BY
            EXTRACT(
                YEAR
                FROM t.topup_time
            ),
            EXTRACT(
                MONTH
                FROM t.topup_time
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

-- GetYearlyTopupStatusSuccess: Retrieves yearly success metrics for topups
-- Purpose: Compare annual successful topup performance
-- Parameters:
--   $1: current_year - The target year (includes this year and previous)
-- Returns:
--   year: Year as text (e.g., '2023')
--   total_success: Count of successful topups
--   total_amount: Sum of successful topup amounts
-- Business Logic:
--   - Only includes successful topups (status = 'success')
--   - Compares current year with previous year
--   - Zero-fills years with no activity
--   - Orders by year (newest first)
--   - Useful for year-over-year growth analysis
-- name: GetYearlyTopupStatusSuccess :many
WITH
    yearly_data AS (
        SELECT
            EXTRACT(
                YEAR
                FROM t.topup_time
            )::integer AS year,
            COUNT(*) AS total_success,
            COALESCE(SUM(t.topup_amount), 0)::integer AS total_amount
        FROM topups t
        WHERE
            t.deleted_at IS NULL
            AND t.status = 'success'
            AND (
                EXTRACT(
                    YEAR
                    FROM t.topup_time
                ) = $1::integer
                OR EXTRACT(
                    YEAR
                    FROM t.topup_time
                ) = $1::integer - 1
            )
        GROUP BY
            EXTRACT(
                YEAR
                FROM t.topup_time
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

-- GetMonthTopupStatusFailed: Retrieves monthly failed metrics for topups
-- Purpose: Analyze failedful topup trends across comparison periods
-- Parameters:
--   $1: period1_start - Start date of first comparison period
--   $2: period1_end - End date of first comparison period
--   $3: period2_start - Start date of second comparison period
--   $4: period2_end - End date of second comparison period
-- Returns:
--   year: Year as text (e.g., '2023')
--   month: 3-letter month abbreviation (e.g., 'Jan')
--   total_failed: Count of failedful topups
--   total_amount: Sum of failedful topup amounts
-- Business Logic:
--   - Only includes failedful topups (status = 'failed')
--   - Covers two customizable time periods for comparison
--   - Zero-fills months with no activity
--   - Formats output for consistent visualization
--   - Orders by year and month (newest first)
--   - Useful for identifying seasonal topup patterns
-- name: GetMonthTopupStatusFailed :many
WITH
    monthly_data AS (
        SELECT
            EXTRACT(
                YEAR
                FROM t.topup_time
            )::integer AS year,
            EXTRACT(
                MONTH
                FROM t.topup_time
            )::integer AS month,
            COUNT(*) AS total_failed,
            COALESCE(SUM(t.topup_amount), 0)::integer AS total_amount
        FROM topups t
        WHERE
            t.deleted_at IS NULL
            AND t.status = 'failed'
            AND (
                (
                    t.topup_time >= $1::timestamp
                    AND t.topup_time <= $2::timestamp
                )
                OR (
                    t.topup_time >= $3::timestamp
                    AND t.topup_time <= $4::timestamp
                )
            )
        GROUP BY
            EXTRACT(
                YEAR
                FROM t.topup_time
            ),
            EXTRACT(
                MONTH
                FROM t.topup_time
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

-- GetYearlyTopupStatusFailed: Retrieves yearly failed metrics for topups
-- Purpose: Compare annual failedful topup performance
-- Parameters:
--   $1: current_year - The target year (includes this year and previous)
-- Returns:
--   year: Year as text (e.g., '2023')
--   total_failed: Count of failedful topups
--   total_amount: Sum of failedful topup amounts
-- Business Logic:
--   - Only includes failedful topups (status = 'failed')
--   - Compares current year with previous year
--   - Zero-fills years with no activity
--   - Orders by year (newest first)
--   - Useful for year-over-year growth analysis
-- name: GetYearlyTopupStatusFailed :many
WITH
    yearly_data AS (
        SELECT
            EXTRACT(
                YEAR
                FROM t.topup_time
            )::integer AS year,
            COUNT(*) AS total_failed,
            COALESCE(SUM(t.topup_amount), 0)::integer AS total_amount
        FROM topups t
        WHERE
            t.deleted_at IS NULL
            AND t.status = 'failed'
            AND (
                EXTRACT(
                    YEAR
                    FROM t.topup_time
                ) = $1::integer
                OR EXTRACT(
                    YEAR
                    FROM t.topup_time
                ) = $1::integer - 1
            )
        GROUP BY
            EXTRACT(
                YEAR
                FROM t.topup_time
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

-- GetMonthTopupStatusSuccessCardNumber: Retrieves monthly success metrics for topups
-- Purpose: Analyze successful topup trends across comparison periods
-- Parameters:
--   $1: card_number       - Optional filter by card_number (NULL to ignore filter)
--   $2: period1_start - Start date of first comparison period
--   $3: period1_end - End date of first comparison period
--   $4: period2_start - Start date of second comparison period
--   $5: period2_end - End date of second comparison period
-- Returns:
--   year: Year as text (e.g., '2023')
--   month: 3-letter month abbreviation (e.g., 'Jan')
--   total_success: Count of successful topups
--   total_amount: Sum of successful topup amounts
-- Business Logic:
--   - Only includes successful topups (status = 'success')
--   - Covers two customizable time periods for comparison
--   - Zero-fills months with no activity
--   - Formats output for consistent visualization
--   - Orders by year and month (newest first)
--   - Useful for identifying seasonal topup patterns
-- name: GetMonthTopupStatusSuccessCardNumber :many
WITH
    monthly_data AS (
        SELECT
            EXTRACT(
                YEAR
                FROM t.topup_time
            )::integer AS year,
            EXTRACT(
                MONTH
                FROM t.topup_time
            )::integer AS month,
            COUNT(*) AS total_success,
            COALESCE(SUM(t.topup_amount), 0)::integer AS total_amount
        FROM topups t
        WHERE
            t.deleted_at IS NULL
            AND t.status = 'success'
            AND t.card_number = $1
            AND (
                (
                    t.topup_time >= $2::timestamp
                    AND t.topup_time <= $3::timestamp
                )
                OR (
                    t.topup_time >= $4::timestamp
                    AND t.topup_time <= $5::timestamp
                )
            )
        GROUP BY
            EXTRACT(
                YEAR
                FROM t.topup_time
            ),
            EXTRACT(
                MONTH
                FROM t.topup_time
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

-- GetYearlyTopupStatusSuccess: Retrieves yearly success metrics for topups
-- Purpose: Compare annual successful topup performance
-- Parameters:
--   $1: card_number       - Optional filter by card_number (NULL to ignore filter)
--   $2: current_year - The target year (includes this year and previous)
-- Returns:
--   year: Year as text (e.g., '2023')
--   total_success: Count of successful topups
--   total_amount: Sum of successful topup amounts
-- Business Logic:
--   - Only includes successful topups (status = 'success')
--   - Compares current year with previous year
--   - Zero-fills years with no activity
--   - Orders by year (newest first)
--   - Useful for year-over-year growth analysis
-- name: GetYearlyTopupStatusSuccessCardNumber :many
WITH
    yearly_data AS (
        SELECT
            EXTRACT(
                YEAR
                FROM t.topup_time
            )::integer AS year,
            COUNT(*) AS total_success,
            COALESCE(SUM(t.topup_amount), 0)::integer AS total_amount
        FROM topups t
        WHERE
            t.deleted_at IS NULL
            AND t.status = 'success'
            AND t.card_number = $1
            AND (
                EXTRACT(
                    YEAR
                    FROM t.topup_time
                ) = $2::integer
                OR EXTRACT(
                    YEAR
                    FROM t.topup_time
                ) = $2::integer - 1
            )
        GROUP BY
            EXTRACT(
                YEAR
                FROM t.topup_time
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

-- GetMonthTopupStatusFailedCardNumber: Retrieves monthly failed metrics for topups
-- Purpose: Analyze failedful topup trends across comparison periods
-- Parameters:
--   $1: card_number       - Optional filter by card_number (NULL to ignore filter)
--   $2: period1_start - Start date of first comparison period
--   $3: period1_end - End date of first comparison period
--   $4: period2_start - Start date of second comparison period
--   $5: period2_end - End date of second comparison period
-- Returns:
--   year: Year as text (e.g., '2023')
--   month: 3-letter month abbreviation (e.g., 'Jan')
--   total_failed: Count of failedful topups
--   total_amount: Sum of failedful topup amounts
-- Business Logic:
--   - Only includes failedful topups (status = 'failed')
--   - Covers two customizable time periods for comparison
--   - Zero-fills months with no activity
--   - Formats output for consistent visualization
--   - Orders by year and month (newest first)
--   - Useful for identifying seasonal topup patterns
-- name: GetMonthTopupStatusFailedCardNumber :many
WITH
    monthly_data AS (
        SELECT
            EXTRACT(
                YEAR
                FROM t.topup_time
            )::integer AS year,
            EXTRACT(
                MONTH
                FROM t.topup_time
            )::integer AS month,
            COUNT(*) AS total_failed,
            COALESCE(SUM(t.topup_amount), 0)::integer AS total_amount
        FROM topups t
        WHERE
            t.deleted_at IS NULL
            AND t.status = 'failed'
            AND t.card_number = $1
            AND (
                (
                    t.topup_time >= $2::timestamp
                    AND t.topup_time <= $3::timestamp
                )
                OR (
                    t.topup_time >= $4::timestamp
                    AND t.topup_time <= $5::timestamp
                )
            )
        GROUP BY
            EXTRACT(
                YEAR
                FROM t.topup_time
            ),
            EXTRACT(
                MONTH
                FROM t.topup_time
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
                FROM $4::timestamp
            )::text AS year,
            TO_CHAR($4::timestamp, 'Mon') AS month,
            0 AS total_failed,
            0 AS total_amount
        WHERE
            NOT EXISTS (
                SELECT 1
                FROM monthly_data
                WHERE
                    year = EXTRACT(
                        YEAR
                        FROM $4::timestamp
                    )::integer
                    AND month = EXTRACT(
                        MONTH
                        FROM $4::timestamp
                    )::integer
            )
    )
SELECT *
FROM formatted_data
ORDER BY year DESC, TO_DATE(month, 'Mon') DESC;

-- GetYearlyTopupStatusFailedCardNumber: Retrieves yearly failed metrics for topups
-- Purpose: Compare annual failedful topup performance
-- Parameters:
--   $1: card_number  - filter by card_number
--   $2: current_year - The target year (includes this year and previous)
-- Returns:
--   year: Year as text (e.g., '2023')
--   total_failed: Count of failedful topups
--   total_amount: Sum of failedful topup amounts
-- Business Logic:
--   - Only includes failedful topups (status = 'failed')
--   - Compares current year with previous year
--   - Zero-fills years with no activity
--   - Orders by year (newest first)
--   - Useful for year-over-year growth analysis
-- name: GetYearlyTopupStatusFailedCardNumber :many
WITH
    yearly_data AS (
        SELECT
            EXTRACT(
                YEAR
                FROM t.topup_time
            )::integer AS year,
            COUNT(*) AS total_failed,
            COALESCE(SUM(t.topup_amount), 0)::integer AS total_amount
        FROM topups t
        WHERE
            t.deleted_at IS NULL
            AND t.status = 'failed'
            AND t.card_number = $1
            AND (
                EXTRACT(
                    YEAR
                    FROM t.topup_time
                ) = $2::integer
                OR EXTRACT(
                    YEAR
                    FROM t.topup_time
                ) = $2::integer - 1
            )
        GROUP BY
            EXTRACT(
                YEAR
                FROM t.topup_time
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

-- GetMonthlyTopupMethods: Retrieves monthly breakdown of topup usage by method
-- Purpose: Track topup method distribution and amounts over each month of the selected year
-- Parameters:
--   $1: reference_date - Any date within the target year (used to define monthly range)
-- Returns:
--   month: 3-letter month abbreviation (e.g., 'Jan')
--   topup_method: Method used for topup (e.g., 'bank_transfer', 'e-wallet')
--   total_topups: Count of topups using the method in that month
--   total_amount: Sum of topup amounts using the method in that month
-- Business Logic:
--   - Ensures every topup method is shown for every month (even with 0 data)
--   - Filters out soft-deleted records (deleted_at IS NULL)
--   - Uses CROSS JOIN to combine months with all available methods
--   - Useful for visualizing adoption trends of each topup method monthly
-- name: GetMonthlyTopupMethods :many
WITH
    months AS (
        SELECT generate_series(
                date_trunc('year', $1::timestamp), date_trunc('year', $1::timestamp) + interval '1 year' - interval '1 day', interval '1 month'
            ) AS month
    ),
    topup_methods AS (
        SELECT DISTINCT
            topup_method
        FROM topups
        WHERE
            deleted_at IS NULL
    )
SELECT
    TO_CHAR(m.month, 'Mon') AS month,
    tm.topup_method,
    COALESCE(COUNT(t.topup_id), 0)::int AS total_topups,
    COALESCE(SUM(t.topup_amount), 0)::int AS total_amount
FROM
    months m
    CROSS JOIN topup_methods tm
    LEFT JOIN topups t ON EXTRACT(
        MONTH
        FROM t.topup_time
    ) = EXTRACT(
        MONTH
        FROM m.month
    )
    AND EXTRACT(
        YEAR
        FROM t.created_at
    ) = EXTRACT(
        YEAR
        FROM m.month
    )
    AND t.topup_method = tm.topup_method
    AND t.deleted_at IS NULL
GROUP BY
    m.month,
    tm.topup_method
ORDER BY m.month, tm.topup_method;

-- GetYearlyTopupMethods: Retrieves yearly breakdown of topup usage by method
-- Purpose: Analyze how different topup methods perform over the past 5 years
-- Parameters:
--   $1: current_year - The final year to include (e.g., 2024), includes 5-year span (current_year - 4)
-- Returns:
--   year: Year extracted from topup_time
--   topup_method: Method used for topup
--   total_topups: Number of topups using that method in the year
--   total_amount: Total topup amount for the method in the year
-- Business Logic:
--   - Filters to topups within a 5-year window up to the given year
--   - Filters out soft-deleted data (deleted_at IS NULL)
--   - Useful for detecting long-term trends across payment methods
-- name: GetYearlyTopupMethods :many
SELECT
    EXTRACT(
        YEAR
        FROM t.created_at
    ) AS year,
    t.topup_method,
    COUNT(t.topup_id) AS total_topups,
    SUM(t.topup_amount) AS total_amount
FROM topups t
WHERE
    t.deleted_at IS NULL
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
        FROM t.created_at
    ),
    t.topup_method
ORDER BY year;

-- GetMonthlyTopupAmounts: Retrieves total topup amounts per month for the selected year
-- Purpose: Visualize total topup volume across months in a given year
-- Parameters:
--   $1: reference_date - Any date within the target year
-- Returns:
--   month: 3-letter month abbreviation
--   total_amount: Sum of all topup amounts per month
-- Business Logic:
--   - Filters soft-deleted entries (deleted_at IS NULL)
--   - Uses LEFT JOIN to ensure months with no topups are still included with amount = 0
--   - Useful for monthly topup charts or dashboards
-- name: GetMonthlyTopupAmounts :many
WITH
    months AS (
        SELECT generate_series(
                date_trunc('year', $1::timestamp), date_trunc('year', $1::timestamp) + interval '1 year' - interval '1 day', interval '1 month'
            ) AS month
    )
SELECT TO_CHAR(m.month, 'Mon') AS month, COALESCE(SUM(t.topup_amount), 0)::int AS total_amount
FROM months m
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
GROUP BY
    m.month
ORDER BY m.month;

-- GetYearlyTopupAmounts: Retrieves yearly total of topup amounts
-- Purpose: Analyze yearly growth or decline in topup volume
-- Parameters:
--   $1: current_year - The latest year to include (e.g., 2024), includes 5-year span (current_year - 4)
-- Returns:
--   year: Year extracted from topup_time
--   total_amount: Sum of all topup amounts in the year
-- Business Logic:
--   - Includes topup data from current year and 4 years prior
--   - Excludes soft-deleted records (deleted_at IS NULL)
--   - Ideal for trend lines or comparative bar charts by year
-- name: GetYearlyTopupAmounts :many
SELECT EXTRACT(
        YEAR
        FROM t.topup_time
    ) AS year, SUM(t.topup_amount) AS total_amount
FROM topups t
WHERE
    t.deleted_at IS NULL
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
ORDER BY year;

-- GetMonthlyTopupMethodsByCardNumber: Retrieves monthly breakdown of topup usage by method
-- Purpose: Track topup method distribution and amounts over each month of the selected year
-- Parameters:
--   $1: card_number  - filter by card_number
--   $2: reference_date - Any date within the target year (used to define monthly range)
-- Returns:
--   month: 3-letter month abbreviation (e.g., 'Jan')
--   topup_method: Method used for topup (e.g., 'bank_transfer', 'e-wallet')
--   total_topups: Count of topups using the method in that month
--   total_amount: Sum of topup amounts using the method in that month
-- Business Logic:
--   - Ensures every topup method is shown for every month (even with 0 data)
--   - Filters out soft-deleted records (deleted_at IS NULL)
--   - Uses CROSS JOIN to combine months with all available methods
--   - Useful for visualizing adoption trends of each topup method monthly
-- name: GetMonthlyTopupMethodsByCardNumber :many
WITH
    months AS (
        SELECT generate_series(
                date_trunc('year', $2::timestamp), date_trunc('year', $2::timestamp) + interval '1 year' - interval '1 day', interval '1 month'
            ) AS month
    ),
    topup_methods AS (
        SELECT DISTINCT
            topup_method
        FROM topups
        WHERE
            deleted_at IS NULL
    )
SELECT
    TO_CHAR(m.month, 'Mon') AS month,
    tm.topup_method,
    COALESCE(COUNT(t.topup_id), 0)::int AS total_topups,
    COALESCE(SUM(t.topup_amount), 0)::int AS total_amount
FROM
    months m
    CROSS JOIN topup_methods tm
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
    AND t.topup_method = tm.topup_method
    AND t.card_number = $1
    AND t.deleted_at IS NULL
GROUP BY
    m.month,
    tm.topup_method
ORDER BY m.month, tm.topup_method;

-- GetYearlyTopupMethodsByCardNumber: Retrieves yearly breakdown of topup usage by method
-- Purpose: Analyze how different topup methods perform over the past 5 years
-- Parameters:
--   $1: card_number  - filter by card_number
--   $2: current_year - The final year to include (e.g., 2024), includes 5-year span (current_year - 4)
-- Returns:
--   year: Year extracted from topup_time
--   topup_method: Method used for topup
--   total_topups: Number of topups using that method in the year
--   total_amount: Total topup amount for the method in the year
-- Business Logic:
--   - Filters to topups within a 5-year window up to the given year
--   - Filters out soft-deleted data (deleted_at IS NULL)
--   - Useful for detecting long-term trends across payment methods
-- name: GetYearlyTopupMethodsByCardNumber :many
SELECT
    EXTRACT(
        YEAR
        FROM t.topup_time
    ) AS year,
    t.topup_method,
    COUNT(t.topup_id) AS total_topups,
    SUM(t.topup_amount) AS total_amount
FROM topups t
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
        FROM t.topup_time
    ),
    t.topup_method
ORDER BY year;

-- GetMonthlyTopupAmountsByCardNumber: Retrieves total topup amounts per month for the selected year
-- Purpose: Visualize total topup volume across months in a given year
-- Parameters:
--   $1: card_number  - filter by card_number
--   $2: reference_date - Any date within the target year
-- Returns:
--   month: 3-letter month abbreviation
--   total_amount: Sum of all topup amounts per month
-- Business Logic:
--   - Filters soft-deleted entries (deleted_at IS NULL)
--   - Uses LEFT JOIN to ensure months with no topups are still included with amount = 0
--   - Useful for monthly topup charts or dashboards
-- name: GetMonthlyTopupAmountsByCardNumber :many
WITH
    months AS (
        SELECT generate_series(
                date_trunc('year', $2::timestamp), date_trunc('year', $2::timestamp) + interval '1 year' - interval '1 day', interval '1 month'
            ) AS month
    )
SELECT TO_CHAR(m.month, 'Mon') AS month, COALESCE(SUM(t.topup_amount), 0)::int AS total_amount
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
    AND t.card_number = $1
    AND t.deleted_at IS NULL
GROUP BY
    m.month
ORDER BY m.month;

-- GetYearlyTopupAmountsByCardNumber: Retrieves yearly total of topup amounts
-- Purpose: Analyze yearly growth or decline in topup volume
-- Parameters:
--   $1: card_number  - filter by card_number
--   $2: current_year - The latest year to include (e.g., 2024), includes 5-year span (current_year - 4)
-- Returns:
--   year: Year extracted from topup_time
--   total_amount: Sum of all topup amounts in the year
-- Business Logic:
--   - Includes topup data from current year and 4 years prior
--   - Excludes soft-deleted records (deleted_at IS NULL)
--   - Ideal for trend lines or comparative bar charts by year
-- name: GetYearlyTopupAmountsByCardNumber :many
SELECT EXTRACT(
        YEAR
        FROM t.topup_time
    ) AS year, SUM(t.topup_amount) AS total_amount
FROM topups t
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
        FROM t.topup_time
    )
ORDER BY year;

-- CreateTopup: Inserts a new topup transaction into the topups table
-- Purpose: Used when a user performs a topup action
-- Parameters:
--   $1: card_number - The card number receiving the topup
--   $2: topup_amount - Amount of the topup
--   $3: topup_method - Payment method used (e.g., 'e-wallet', 'bank_transfer')
--   $4: topup_time - Timestamp of the actual topup transaction
-- Returns:
--   Full topup record including auto-generated fields
-- Business Logic:
--   - Automatically sets created_at and updated_at to current timestamp
-- name: CreateTopup :one
INSERT INTO
    topups (
        card_number,
        topup_amount,
        topup_method,
        topup_time,
        created_at,
        updated_at
    )
VALUES (
        $1,
        $2,
        $3,
        $4,
        current_timestamp,
        current_timestamp
    )
RETURNING
    topup_id,
    topup_no,
    card_number,
    topup_amount,
    topup_method,
    topup_time,
    status,
    created_at,
    updated_at;

-- UpdateTopup: Updates an existing topup transaction
-- Purpose: Modify existing topup information by ID
-- Parameters:
--   $1: topup_id - ID of the topup to update
--   $2: card_number - Updated card number
--   $3: topup_amount - Updated amount
--   $4: topup_method - Updated payment method
--   $5: topup_time - Updated transaction time
-- Business Logic:
--   - Skips soft-deleted records (deleted_at IS NULL)
--   - Updates updated_at automatically
-- name: UpdateTopup :one
UPDATE topups
SET
    card_number = $2,
    topup_amount = $3,
    topup_method = $4,
    topup_time = $5,
    updated_at = current_timestamp
WHERE
    topup_id = $1
    AND deleted_at IS NULL
RETURNING
    topup_id,
    topup_no,
    card_number,
    topup_amount,
    topup_method,
    topup_time,
    status,
    created_at,
    updated_at;

-- UpdateTopupAmount: Updates only the topup_amount field for a specific topup
-- Purpose: Allow adjustment of topup amount without affecting other fields
-- Parameters:
--   $1: topup_id - ID of the target topup
--   $2: new topup amount
-- Business Logic:
--   - Ignores deleted entries
--   - Automatically updates the updated_at timestamp
-- name: UpdateTopupAmount :one
UPDATE topups
SET
    topup_amount = $2,
    updated_at = current_timestamp
WHERE
    topup_id = $1
    AND deleted_at IS NULL
RETURNING
    topup_id,
    topup_no,
    card_number,
    topup_amount,
    topup_method,
    topup_time,
    status,
    created_at,
    updated_at;

-- UpdateTopupStatus: Updates the status of a specific topup
-- Purpose: Mark topup as 'success', 'failed', etc.
-- Parameters:
--   $1: topup_id - ID of the topup
--   $2: new status value (e.g., 'success', 'failed')
-- Business Logic:
--   - Applies only to active (non-deleted) records
--   - updated_at is refreshed
-- name: UpdateTopupStatus :one
UPDATE topups
SET
    status = $2,
    updated_at = current_timestamp
WHERE
    topup_id = $1
    AND deleted_at IS NULL
RETURNING
    topup_id,
    topup_no,
    card_number,
    topup_amount,
    topup_method,
    topup_time,
    status,
    created_at,
    updated_at;

-- GetTopupsByCardNumber: Retrieves paginated topups based on card number and optional search keyword
-- Purpose: View all topups for a specific card, with filtering and pagination
-- Parameters:
--   $1: card_number - Exact card number match
--   $2: keyword - Optional keyword (nullable), filters topup_no, method, status
--   $3: limit - Number of records to return
--   $4: offset - Offset for pagination
-- Returns:
--   All matching topup records with total_count using window function
-- Business Logic:
--   - Skips soft-deleted records
--   - Ordered by topup_time descending
-- name: GetTopupsByCardNumber :many
SELECT
    topup_id,
    topup_no,
    card_number,
    topup_amount,
    topup_method,
    topup_time,
    status,
    created_at,
    updated_at,
    COUNT(*) OVER () AS total_count
FROM topups
WHERE
    deleted_at IS NULL
    AND card_number = $1
    AND (
        $2::TEXT IS NULL
        OR topup_no::TEXT ILIKE '%' || $2 || '%'
        OR topup_method ILIKE '%' || $2 || '%'
        OR status ILIKE '%' || $2 || '%'
    )
ORDER BY topup_time DESC
LIMIT $3
OFFSET
    $4;

-- GetTrashedTopupByID: Retrieves a topup that has been soft-deleted
-- Purpose: Preview or manage trashed entries (e.g., for restore)
-- Parameters:
--   $1: topup_id - ID of the soft-deleted topup
-- Returns:
--   Full topup record if found and deleted_at IS NOT NULL
-- name: GetTrashedTopupByID :one
SELECT *
FROM topups
WHERE
    topup_id = $1
    AND deleted_at IS NOT NULL;

-- TrashTopup: Soft deletes a topup by setting deleted_at
-- Purpose: Moves topup to trash without losing data
-- Parameters:
--   $1: topup_id - ID of the topup to soft-delete
-- Business Logic:
--   - Only active (non-deleted) records can be trashed
--   - Allows restore in future
-- name: TrashTopup :one
UPDATE topups
SET
    deleted_at = current_timestamp
WHERE
    topup_id = $1
    AND deleted_at IS NULL
RETURNING
    topup_id,
    topup_no,
    card_number,
    topup_amount,
    topup_method,
    topup_time,
    status,
    created_at,
    updated_at,
    deleted_at;

-- RestoreTopup: Restores a soft-deleted topup by nullifying deleted_at
-- Purpose: Reactivate a previously trashed topup
-- Parameters:
--   $1: topup_id - ID of the topup to restore
-- Business Logic:
--   - Only applies to records where deleted_at IS NOT NULL
-- name: RestoreTopup :one
UPDATE topups
SET
    deleted_at = NULL
WHERE
    topup_id = $1
    AND deleted_at IS NOT NULL
RETURNING
    topup_id,
    topup_no,
    card_number,
    topup_amount,
    topup_method,
    topup_time,
    status,
    created_at,
    updated_at,
    deleted_at;

-- DeleteTopupPermanently: Permanently deletes a topup record from the database
-- Purpose: Irrecoverably removes topup data
-- Parameters:
--   $1: topup_id - ID of the topup to delete
-- Business Logic:
--   - No soft-delete; data is permanently erased
--   - Use with caution
-- name: DeleteTopupPermanently :exec
DELETE FROM topups WHERE topup_id = $1;

-- RestoreAllTopups: Restores all soft-deleted topups in bulk
-- Purpose: Batch recovery of trashed topup data
-- Business Logic:
--   - Sets deleted_at to NULL for all where it was not null
-- name: RestoreAllTopups :exec
UPDATE topups
SET
    deleted_at = NULL
WHERE
    deleted_at IS NOT NULL;

-- DeleteAllPermanentTopups: Permanently deletes all soft-deleted topups
-- Purpose: Bulk cleanup of trashed topup records
-- Business Logic:
--   - Cannot be undone; this is a hard delete
--   - Use for permanent data purging
-- name: DeleteAllPermanentTopups :exec
DELETE FROM topups WHERE deleted_at IS NOT NULL;