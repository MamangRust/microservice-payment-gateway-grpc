-- GetMerchants: Retrieves paginated list of all non-deleted merchants with search capability
-- Purpose: Display all active (non-trashed) merchants in admin interface
-- Parameters:
--   $1: search_term - Optional text to filter by name, api_key, or status (NULL for no filter)
--   $2: limit - Maximum number of records to return
--   $3: offset - Number of records to skip for pagination
-- Returns:
--   All merchant fields plus total_count of matching records
-- Business Logic:
--   - Excludes soft-deleted merchants (deleted_at IS NULL)
--   - Supports partial text search on name, api_key, and status (case-insensitive)
--   - Returns results ordered by merchant_id
--   - Provides total_count for pagination calculations
-- name: GetMerchants :many
SELECT
    merchant_id,
    merchant_no,
    name,
    api_key,
    user_id,
    status,
    created_at,
    updated_at,
    COUNT(*) OVER () AS total_count
FROM merchants
WHERE
    deleted_at IS NULL
    AND (
        $1::TEXT IS NULL
        OR name ILIKE '%' || $1 || '%'
        OR api_key ILIKE '%' || $1 || '%'
        OR status ILIKE '%' || $1 || '%'
    )
ORDER BY merchant_id
LIMIT $2
OFFSET
    $3;

-- GetActiveMerchants: Retrieves paginated list of active merchants with search capability
-- Purpose: List currently active merchants (same as GetMerchants)
-- Parameters:
--   $1: search_term - Optional text to filter by name, api_key, or status (NULL for no filter)
--   $2: limit - Maximum number of records to return
--   $3: offset - Number of records to skip for pagination
-- Returns:
--   All merchant fields plus total_count of matching records
-- Business Logic:
--   - Excludes soft-deleted merchants (deleted_at IS NULL)
--   - Supports case-insensitive partial matching on name, api_key, and status
--   - Returns results ordered by merchant_id
--   - Provides total_count for pagination calculations
-- name: GetActiveMerchants :many
SELECT
    merchant_id,
    merchant_no,
    name,
    api_key,
    user_id,
    status,
    created_at,
    updated_at,
    deleted_at,
    COUNT(*) OVER () AS total_count
FROM merchants
WHERE
    deleted_at IS NULL
    AND (
        $1::TEXT IS NULL
        OR name ILIKE '%' || $1 || '%'
        OR api_key ILIKE '%' || $1 || '%'
        OR status ILIKE '%' || $1 || '%'
    )
ORDER BY merchant_id
LIMIT $2
OFFSET
    $3;

-- GetTrashedMerchants: Retrieves paginated list of soft-deleted merchants with search capability
-- Purpose: View trashed merchants for potential restoration or permanent deletion
-- Parameters:
--   $1: search_term - Optional text to filter by name, api_key, or status (NULL for no filter)
--   $2: limit - Maximum number of records to return
--   $3: offset - Number of records to skip for pagination
-- Returns:
--   All merchant fields plus total_count of matching records
-- Business Logic:
--   - Only includes soft-deleted merchants (deleted_at IS NOT NULL)
--   - Supports partial text search (case-insensitive) on name, api_key, and status
--   - Returns results ordered by merchant_id
--   - Provides total_count for pagination calculations
-- name: GetTrashedMerchants :many
SELECT
    merchant_id,
    merchant_no,
    name,
    api_key,
    user_id,
    status,
    created_at,
    updated_at,
    deleted_at,
    COUNT(*) OVER () AS total_count
FROM merchants
WHERE
    deleted_at IS NOT NULL
    AND (
        $1::TEXT IS NULL
        OR name ILIKE '%' || $1 || '%'
        OR api_key ILIKE '%' || $1 || '%'
        OR status ILIKE '%' || $1 || '%'
    )
ORDER BY merchant_id
LIMIT $2
OFFSET
    $3;

-- GetMerchantByID: Retrieves a merchant by its unique ID
-- Purpose: Fetch details of a single merchant if not soft-deleted
-- Parameters:
--   $1: merchant_id - Unique identifier of the merchant
-- Returns:
--   Complete merchant record
-- Business Logic:
--   - Excludes soft-deleted merchants (deleted_at IS NULL)
-- name: GetMerchantByID :one
SELECT
    merchant_id,
    merchant_no,
    name,
    api_key,
    user_id,
    status,
    created_at,
    updated_at
FROM merchants
WHERE
    merchant_id = $1
    AND deleted_at IS NULL;

-- GetMerchantByApiKey: Retrieves a merchant by its API key
-- Purpose: Authenticate or lookup a merchant using its API key
-- Parameters:
--   $1: api_key - API key of the merchant
-- Returns:
--   Complete merchant record
-- Business Logic:
--   - Excludes soft-deleted merchants (deleted_at IS NULL)
-- name: GetMerchantByApiKey :one
SELECT
    merchant_id,
    merchant_no,
    name,
    api_key,
    user_id,
    status,
    created_at,
    updated_at
FROM merchants
WHERE
    api_key = $1
    AND deleted_at IS NULL;

-- GetMerchantByName: Retrieves a merchant by its name
-- Purpose: Find merchant data based on exact name match
-- Parameters:
--   $1: name - Exact name of the merchant
-- Returns:
--   Complete merchant record
-- Business Logic:
--   - Excludes soft-deleted merchants (deleted_at IS NULL)
-- name: GetMerchantByName :one
SELECT
    merchant_id,
    merchant_no,
    name,
    api_key,
    user_id,
    status,
    created_at,
    updated_at
FROM merchants
WHERE
    name = $1
    AND deleted_at IS NULL;

-- GetMerchantsByUserID: Retrieves all merchants associated with a user
-- Purpose: List all merchants that belong to a specific user
-- Parameters:
--   $1: user_id - ID of the user who owns the merchants
-- Returns:
--   List of merchant records
-- Business Logic:
--   - Excludes soft-deleted merchants (deleted_at IS NULL)
-- name: GetMerchantsByUserID :many
SELECT
    merchant_id,
    merchant_no,
    name,
    api_key,
    user_id,
    status,
    created_at,
    updated_at
FROM merchants
WHERE
    user_id = $1
    AND deleted_at IS NULL;

-- GetTrashedMerchantByID: Retrieves a soft-deleted merchant by ID
-- Purpose: Access trashed merchant record for potential restoration or inspection
-- Parameters:
--   $1: merchant_id - Unique identifier of the merchant
-- Returns:
--   Trashed merchant record
-- Business Logic:
--   - Includes only merchants that have been soft-deleted (deleted_at IS NOT NULL)
-- name: GetTrashedMerchantByID :one
SELECT *
FROM merchants
WHERE
    merchant_id = $1
    AND deleted_at IS NOT NULL;

-- GetMonthlyPaymentMethodsMerchant: Retrieves monthly transaction totals per payment method
-- Purpose: Analyze monthly transaction distribution across payment methods for a given year
-- Parameters:
--   $1: reference_date - Any date within the target year
-- Returns:
--   - Month name (e.g., Jan, Feb)
--   - Payment method
--   - Total transaction amount (0 if no activity)
-- Business Logic:
--   - Generates complete 12-month series from the reference year
--   - Cross joins with distinct active payment methods
--   - Filters only active (non-deleted) transactions and merchants
--   - Uses LEFT JOIN to ensure all month-method combinations are included
--   - Uses COALESCE to display 0 for months with no transactions
--   - Ordered by month and payment method
-- name: GetMonthlyPaymentMethodsMerchant :many
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
SELECT TO_CHAR(m.month, 'Mon') AS month, pm.payment_method, COALESCE(SUM(t.amount), 0)::int AS total_amount
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
    LEFT JOIN merchants mch ON t.merchant_id = mch.merchant_id
    AND mch.deleted_at IS NULL
GROUP BY
    m.month,
    pm.payment_method
ORDER BY m.month, pm.payment_method;

-- GetYearlyPaymentMethodMerchant: Retrieves yearly transaction totals per payment method (last 5 years)
-- Purpose: Show transaction trends across payment methods over the past 5 years
-- Parameters:
--   $1: current_year - The latest year to include in the 5-year window
-- Returns:
--   - Year (e.g., 2021, 2022)
--   - Payment method
--   - Total transaction amount
-- Business Logic:
--   - Aggregates yearly totals for each payment method
--   - Includes only active (non-deleted) transactions and merchants
--   - Covers a 5-year range: (current_year - 4) to current_year
--   - Ordered by year
-- name: GetYearlyPaymentMethodMerchant :many
WITH
    last_five_years AS (
        SELECT EXTRACT(
                YEAR
                FROM t.transaction_time
            ) AS year, t.payment_method, SUM(t.amount) AS total_amount
        FROM transactions t
            JOIN merchants m ON t.merchant_id = m.merchant_id
        WHERE
            t.deleted_at IS NULL
            AND m.deleted_at IS NULL
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
            ),
            t.payment_method
    )
SELECT year, payment_method, total_amount
FROM last_five_years
ORDER BY year;

-- GetMonthlyAmountMerchant: Retrieves total transaction amount per month for a given year
-- Purpose: Generate monthly income report regardless of payment method
-- Parameters:
--   $1: reference_date - Any date within the target year
-- Returns:
--   - Month name (e.g., Jan, Feb)
--   - Total transaction amount (0 if no activity)
-- Business Logic:
--   - Generates complete 12-month series
--   - Includes only active (non-deleted) transactions and merchants
--   - Uses LEFT JOIN to ensure each month is represented
--   - Uses COALESCE to return 0 if a month has no data
--   - Ordered chronologically by month
-- name: GetMonthlyAmountMerchant :many
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
    LEFT JOIN merchants mch ON t.merchant_id = mch.merchant_id
    AND mch.deleted_at IS NULL
GROUP BY
    m.month
ORDER BY m.month;

-- GetYearlyAmountMerchant: Retrieves total transaction amount per year for the last 5 years
-- Purpose: Show overall yearly revenue trends across all payment methods
-- Parameters:
--   $1: current_year - The latest year to include in the 5-year window
-- Returns:
--   - Year (e.g., 2021, 2022)
--   - Total transaction amount
-- Business Logic:
--   - Aggregates yearly transaction amounts
--   - Filters only active (non-deleted) transactions and merchants
--   - Includes data for the last 5 calendar years up to the current year
--   - Ordered chronologically by year
-- name: GetYearlyAmountMerchant :many
WITH
    last_five_years AS (
        SELECT EXTRACT(
                YEAR
                FROM t.transaction_time
            ) AS year, SUM(t.amount) AS total_amount
        FROM transactions t
            JOIN merchants m ON t.merchant_id = m.merchant_id
        WHERE
            t.deleted_at IS NULL
            AND m.deleted_at IS NULL
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
SELECT year, total_amount
FROM last_five_years
ORDER BY year;

-- GetMonthlyTotalAmountMerchant: Retrieves total transaction amounts for the current and previous month
-- Purpose: Provide monthly transaction summary including zero values if no transactions exist
-- Parameters:
--   $1: reference_date - Any date within the target (current) month
-- Returns:
--   - Year (as text)
--   - Month (abbreviated name, e.g., Jan, Feb)
--   - Total transaction amount for each month
-- Business Logic:
--   - Aggregates total transaction amounts for the target month and the month before
--   - Filters only active (non-deleted) transactions and merchants
--   - Includes 0 as total_amount if there's no transaction data for either month
--   - Uses UNION ALL to combine real data with "missing month" placeholders
--   - Results are sorted by year and month (most recent first)
-- name: GetMonthlyTotalAmountMerchant :many
WITH
    monthly_data AS (
        SELECT EXTRACT(
                YEAR
                FROM t.transaction_time
            )::text AS year, TO_CHAR(t.transaction_time, 'Mon') AS month, COALESCE(SUM(t.amount), 0)::integer AS total_amount
        FROM transactions t
            INNER JOIN merchants m ON t.merchant_id = m.merchant_id
        WHERE
            t.deleted_at IS NULL
            AND m.deleted_at IS NULL
            AND (
                t.transaction_time >= date_trunc('month', $1::timestamp) - interval '1 month'
                AND t.transaction_time < date_trunc('month', $1::timestamp) + interval '1 month'
            )
        GROUP BY
            EXTRACT(
                YEAR
                FROM t.transaction_time
            ),
            TO_CHAR(t.transaction_time, 'Mon')
    ),
    missing_months AS (
        SELECT EXTRACT(
                YEAR
                FROM $1::timestamp
            )::text AS year, TO_CHAR($1::timestamp, 'Mon') AS month, 0::integer AS total_amount
        WHERE
            NOT EXISTS (
                SELECT 1
                FROM monthly_data
                WHERE
                    year = EXTRACT(
                        YEAR
                        FROM $1::timestamp
                    )::text
                    AND month = TO_CHAR($1::timestamp, 'Mon')
            )
        UNION ALL
        SELECT EXTRACT(
                YEAR
                FROM date_trunc('month', $1::timestamp) - interval '1 month'
            )::text AS year, TO_CHAR(
                date_trunc('month', $1::timestamp) - interval '1 month', 'Mon'
            ) AS month, 0::integer AS total_amount
        WHERE
            NOT EXISTS (
                SELECT 1
                FROM monthly_data
                WHERE
                    year = EXTRACT(
                        YEAR
                        FROM date_trunc('month', $1::timestamp) - interval '1 month'
                    )::text
                    AND month = TO_CHAR(
                        date_trunc('month', $1::timestamp) - interval '1 month',
                        'Mon'
                    )
            )
    )
SELECT year, month, total_amount
FROM (
        SELECT year, month, total_amount
        FROM monthly_data
        UNION ALL
        SELECT year, month, total_amount
        FROM missing_months
    ) combined
ORDER BY year DESC, TO_DATE(month, 'Mon') DESC;

-- GetYearlyTotalAmountMerchant: Retrieves total transaction amounts for the current and previous year
-- Purpose: Provide yearly transaction summary with fallback to 0 if no transactions exist
-- Parameters:
--   $1: current_year - The latest year to include in the summary
-- Returns:
--   - Year (as text)
--   - Total transaction amount per year
-- Business Logic:
--   - Aggregates total amounts for both the current year and the previous year
--   - Filters only active (non-deleted) transactions and merchants
--   - Ensures both years appear in the result, even if no data exists (returns 0 in such case)
--   - Uses UNION ALL to combine actual data with 0-filled placeholders
--   - Results are ordered in descending order by year
-- name: GetYearlyTotalAmountMerchant :many
WITH
    yearly_data AS (
        SELECT EXTRACT(
                YEAR
                FROM t.transaction_time
            )::integer AS year, COALESCE(SUM(t.amount), 0)::integer AS total_amount
        FROM transactions t
            INNER JOIN merchants m ON t.merchant_id = m.merchant_id
        WHERE
            t.deleted_at IS NULL
            AND m.deleted_at IS NULL
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
        SELECT year::text, total_amount::integer
        FROM yearly_data
        UNION ALL
        SELECT $1::text AS year, 0::integer AS total_amount
        WHERE
            NOT EXISTS (
                SELECT 1
                FROM yearly_data
                WHERE
                    year = $1::integer
            )
        UNION ALL
        SELECT ($1::integer - 1)::text AS year, 0::integer AS total_amount
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

-- FindAllTransactions: Retrieves a paginated list of active transactions with optional search
-- Purpose: Display transaction list with merchant info, filtered by card number or payment method
-- Parameters:
--   $1: search_query (TEXT, nullable) - Optional search string to match card_number or payment_method
--   $2: limit (INTEGER) - Maximum number of records to return (pagination)
--   $3: offset (INTEGER) - Number of records to skip (pagination)
-- Returns:
--   - transaction_id
--   - card_number
--   - amount
--   - payment_method
--   - merchant_id
--   - merchant_name (from join with merchants table)
--   - transaction_time
--   - created_at, updated_at, deleted_at (for audit purposes)
--   - total_count: Total number of records matching the filter (useful for pagination metadata)
-- Business Logic:
--   - Joins `transactions` with `merchants` to retrieve merchant name
--   - Filters out soft-deleted transactions (where deleted_at IS NOT NULL)
--   - Applies case-insensitive partial match on card_number or payment_method if search query is provided
--   - Uses `COUNT(*) OVER()` to include total matching count for pagination without a separate query
--   - Results are ordered by `transaction_time` descending
-- name: FindAllTransactions :many
SELECT
    t.transaction_id,
    t.card_number,
    t.amount,
    t.payment_method,
    t.merchant_id,
    m.name AS merchant_name,
    t.transaction_time,
    t.created_at,
    t.updated_at,
    t.deleted_at,
    COUNT(*) OVER () AS total_count
FROM transactions t
    JOIN merchants m ON t.merchant_id = m.merchant_id
WHERE
    t.deleted_at IS NULL
    AND (
        $1::TEXT IS NULL
        OR t.card_number ILIKE '%' || $1 || '%'
        OR t.payment_method ILIKE '%' || $1 || '%'
    )
ORDER BY t.transaction_time DESC
LIMIT $2
OFFSET
    $3;

-- GetMonthlyPaymentMethodByMerchants: Retrieves total transaction amount per payment method per month for a specific merchant
-- Purpose: Analyze monthly transaction totals by payment method for a specific merchant and year
-- Parameters:
--   $1: reference_date - Any date within the target year
--   $2: merchant_id - The merchant to filter transactions
-- Returns:
--   - Month name (e.g., Jan, Feb)
--   - Payment method
--   - Total transaction amount for each combination
-- Business Logic:
--   - Generates a complete 12-month series for the given year
--   - Filters only active (non-deleted) transactions and merchants
--   - Filters by specific merchant_id
--   - Uses LEFT JOIN and CROSS JOIN to ensure all months and payment methods are included
--   - Uses COALESCE to return 0 for combinations with no data
--   - Results are ordered chronologically by month
-- name: GetMonthlyPaymentMethodByMerchants :many
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
SELECT TO_CHAR(m.month, 'Mon') AS month, pm.payment_method, COALESCE(SUM(t.amount), 0)::int AS total_amount
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
    AND t.merchant_id = $2
    LEFT JOIN merchants mch ON t.merchant_id = mch.merchant_id
    AND mch.deleted_at IS NULL
GROUP BY
    m.month,
    pm.payment_method
ORDER BY m.month, pm.payment_method;

-- GetYearlyPaymentMethodByMerchants: Retrieves total transaction amount per payment method over the last 5 years for a specific merchant
-- Purpose: Analyze yearly transaction totals grouped by payment method for a merchant
-- Parameters:
--   $1: current_year - The latest year to include in the 5-year window
--   $2: merchant_id - The merchant to filter transactions
-- Returns:
--   - Year (e.g., 2021, 2022)
--   - Payment method
--   - Total transaction amount
-- Business Logic:
--   - Filters only active (non-deleted) transactions and merchants
--   - Filters by specific merchant_id
--   - Includes data for the last 5 calendar years up to the current year
--   - Groups by calendar year and payment method
--   - Results are ordered chronologically by year
-- name: GetYearlyPaymentMethodByMerchants :many
WITH
    last_five_years AS (
        SELECT EXTRACT(
                YEAR
                FROM t.transaction_time
            ) AS year, t.payment_method, SUM(t.amount) AS total_amount
        FROM transactions t
            JOIN merchants m ON t.merchant_id = m.merchant_id
        WHERE
            t.deleted_at IS NULL
            AND m.deleted_at IS NULL
            AND t.merchant_id = $1
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
            ),
            t.payment_method
    )
SELECT year, payment_method, total_amount
FROM last_five_years
ORDER BY year;

-- GetMonthlyAmountByMerchants: Retrieves total transaction amount per month for a specific merchant and year
-- Purpose: Generate monthly income report for a specific merchant regardless of payment method
-- Parameters:
--   $1: reference_date - Any date within the target year
--   $2: merchant_id - The merchant to filter transactions
-- Returns:
--   - Month name (e.g., Jan, Feb)
--   - Total transaction amount (0 if no activity)
-- Business Logic:
--   - Generates complete 12-month series for the given year
--   - Filters only active (non-deleted) transactions and merchants
--   - Filters by specific merchant_id
--   - Uses LEFT JOIN to ensure each month is represented
--   - Uses COALESCE to return 0 if a month has no transaction data
--   - Results are ordered chronologically by month
-- name: GetMonthlyAmountByMerchants :many
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
    AND t.merchant_id = $2
    LEFT JOIN merchants mch ON t.merchant_id = mch.merchant_id
    AND mch.deleted_at IS NULL
GROUP BY
    m.month
ORDER BY m.month;

-- GetYearlyAmountByMerchants: Retrieves total transaction amount per year for the last 5 years for a specific merchant
-- Purpose: Show overall yearly revenue trends for a merchant across all payment methods
-- Parameters:
--   $1: merchant_id - The merchant to filter transactions
--   $2: current_year - The latest year to include in the 5-year window
-- Returns:
--   - Year (e.g., 2021, 2022)
--   - Total transaction amount
-- Business Logic:
--   - Filters only active (non-deleted) transactions and merchants
--   - Filters by specific merchant_id
--   - Includes data for the last 5 calendar years up to the current year
--   - Groups by calendar year
--   - Results are ordered chronologically by year
-- name: GetYearlyAmountByMerchants :many
WITH
    last_five_years AS (
        SELECT EXTRACT(
                YEAR
                FROM t.transaction_time
            ) AS year, SUM(t.amount) AS total_amount
        FROM transactions t
            JOIN merchants m ON t.merchant_id = m.merchant_id
        WHERE
            t.deleted_at IS NULL
            AND m.deleted_at IS NULL
            AND t.merchant_id = $1
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
SELECT year, total_amount
FROM last_five_years
ORDER BY year;

-- GetMonthlyTotalAmountByMerchant: Retrieves total transaction amounts for the current and previous month
-- Purpose: Provide monthly transaction summary including zero values if no transactions exist
-- Parameters:
--   $1: reference_date - Any date within the target (current) month
--   $2: merchant_id - The merchant to filter transactions
-- Returns:
--   - Year (as text)
--   - Month (abbreviated name, e.g., Jan, Feb)
--   - Total transaction amount for each month
-- Business Logic:
--   - Aggregates total transaction amounts for the target month and the month before
--   - Filters only active (non-deleted) transactions and merchants
--   - Includes 0 as total_amount if there's no transaction data for either month
--   - Uses UNION ALL to combine real data with "missing month" placeholders
--   - Results are sorted by year and month (most recent first)
-- name: GetMonthlyTotalAmountByMerchant :many
WITH
    monthly_data AS (
        SELECT EXTRACT(
                YEAR
                FROM t.transaction_time
            )::integer AS year, EXTRACT(
                MONTH
                FROM t.transaction_time
            )::integer AS month, COALESCE(SUM(t.amount), 0)::integer AS total_amount
        FROM transactions t
            INNER JOIN merchants m ON t.merchant_id = m.merchant_id
        WHERE
            t.deleted_at IS NULL
            AND m.deleted_at IS NULL
            AND EXTRACT(
                YEAR
                FROM t.transaction_time
            ) = EXTRACT(
                YEAR
                FROM $1::timestamp
            )
            AND t.merchant_id = $2::integer
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
        SELECT year::text, TO_CHAR(
                TO_DATE(month::text, 'MM'), 'Mon'
            ) AS month, total_amount
        FROM monthly_data
        UNION ALL
        SELECT EXTRACT(
                YEAR
                FROM gs.month
            )::text AS year, TO_CHAR(gs.month, 'Mon') AS month, 0::integer AS total_amount
        FROM generate_series(
                date_trunc('year', $1::timestamp), date_trunc('year', $1::timestamp) + interval '11 month', interval '1 month'
            ) AS gs (month)
        WHERE
            NOT EXISTS (
                SELECT 1
                FROM monthly_data md
                WHERE
                    md.year = EXTRACT(
                        YEAR
                        FROM gs.month
                    )::integer
                    AND md.month = EXTRACT(
                        MONTH
                        FROM gs.month
                    )::integer
            )
    )
SELECT *
FROM formatted_data
ORDER BY year DESC, TO_DATE(month, 'Mon') DESC;

-- GetYearlyTotalAmountByMerchant: Retrieves total transaction amounts for the current and previous year
-- Purpose: Provide yearly transaction summary with fallback to 0 if no transactions exist
-- Parameters:
--   $1: current_year - The latest year to include in the summary
--   $2: merchant_id - The merchant to filter transactions
-- Returns:
--   - Year (as text)
--   - Total transaction amount per year
-- Business Logic:
--   - Aggregates total amounts for both the current year and the previous year
--   - Filters only active (non-deleted) transactions and merchants
--   - Ensures both years appear in the result, even if no data exists (returns 0 in such case)
--   - Uses UNION ALL to combine actual data with 0-filled placeholders
--   - Results are ordered in descending order by year
-- name: GetYearlyTotalAmountByMerchant :many
WITH
    yearly_data AS (
        SELECT EXTRACT(
                YEAR
                FROM t.transaction_time
            )::integer AS year, COALESCE(SUM(t.amount), 0)::integer AS total_amount
        FROM transactions t
            INNER JOIN merchants m ON t.merchant_id = m.merchant_id
        WHERE
            t.deleted_at IS NULL
            AND m.deleted_at IS NULL
            AND EXTRACT(
                YEAR
                FROM t.transaction_time
            ) >= $1::integer - 4
            AND EXTRACT(
                YEAR
                FROM t.transaction_time
            ) <= $1::integer
            AND t.merchant_id = $2::integer
        GROUP BY
            EXTRACT(
                YEAR
                FROM t.transaction_time
            )
    ),
    formatted_data AS (
        SELECT year::text, total_amount
        FROM yearly_data
        UNION ALL
        SELECT y::text AS year, 0::integer AS total_amount
        FROM generate_series($1::integer - 4, $1::integer) AS y
        WHERE
            NOT EXISTS (
                SELECT 1
                FROM yearly_data yd
                WHERE
                    yd.year = y
            )
    )
SELECT *
FROM formatted_data
ORDER BY year DESC;

-- FindAllTransactions: Retrieves a paginated list of active transactions with optional search
-- Purpose: Display transaction list with merchant info, filtered by card number or payment method
-- Parameters:
--   $1: merchant_id - The merchant to filter transactions
--   $2: search_query (TEXT, nullable) - Optional search string to match card_number or payment_method
--   $3: limit (INTEGER) - Maximum number of records to return (pagination)
--   $4: offset (INTEGER) - Number of records to skip (pagination)
-- Returns:
--   - transaction_id
--   - card_number
--   - amount
--   - payment_method
--   - merchant_id
--   - merchant_name (from join with merchants table)
--   - transaction_time
--   - created_at, updated_at, deleted_at (for audit purposes)
--   - total_count: Total number of records matching the filter (useful for pagination metadata)
-- Business Logic:
--   - Joins `transactions` with `merchants` to retrieve merchant name
--   - Filters out soft-deleted transactions (where deleted_at IS NOT NULL)
--   - Applies case-insensitive partial match on card_number or payment_method if search query is provided
--   - Uses `COUNT(*) OVER()` to include total matching count for pagination without a separate query
--   - Results are ordered by `transaction_time` descending
-- name: FindAllTransactionsByMerchant :many
SELECT
    t.transaction_id,
    t.card_number,
    t.amount,
    t.payment_method,
    t.merchant_id,
    m.name AS merchant_name,
    t.transaction_time,
    t.created_at,
    t.updated_at,
    t.deleted_at,
    COUNT(*) OVER () AS total_count
FROM transactions t
    JOIN merchants m ON t.merchant_id = m.merchant_id
WHERE
    t.deleted_at IS NULL
    AND t.merchant_id = $1
    AND (
        $2::TEXT IS NULL
        OR t.card_number ILIKE '%' || $2 || '%'
        OR t.payment_method ILIKE '%' || $2 || '%'
    )
ORDER BY t.transaction_time DESC
LIMIT $3
OFFSET
    $4;

-- GetMonthlyPaymentMethodByApikey: Retrieves total transaction amount per payment method per month for a specific merchant
-- Purpose: Analyze monthly transaction totals by payment method for a specific merchant and year
-- Parameters:
--   $1: reference_date - Any date within the target year
--   $2: api-key - The merchant to filter transactions
-- Returns:
--   - Month name (e.g., Jan, Feb)
--   - Payment method
--   - Total transaction amount for each combination
-- Business Logic:
--   - Generates a complete 12-month series for the given year
--   - Filters only active (non-deleted) transactions and merchants
--   - Filters by specific merchant_id
--   - Uses LEFT JOIN and CROSS JOIN to ensure all months and payment methods are included
--   - Uses COALESCE to return 0 for combinations with no data
--   - Results are ordered chronologically by month
-- name: GetMonthlyPaymentMethodByApikey :many
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
SELECT TO_CHAR(m.month, 'Mon') AS month, pm.payment_method, COALESCE(SUM(t.amount), 0)::int AS total_amount
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
    LEFT JOIN merchants mch ON t.merchant_id = mch.merchant_id
    AND mch.deleted_at IS NULL
    AND mch.api_key = $2
GROUP BY
    m.month,
    pm.payment_method
ORDER BY m.month, pm.payment_method;

-- GetYearlyPaymentMethodByApikey: Retrieves total transaction amount per payment method over the last 5 years for a specific merchant
-- Purpose: Analyze yearly transaction totals grouped by payment method for a merchant
-- Parameters:
--   $1: api-key - The merchant to filter transactions
--   $2: current_year - The latest year to include in the 5-year window
-- Returns:
--   - Year (e.g., 2021, 2022)
--   - Payment method
--   - Total transaction amount
-- Business Logic:
--   - Filters only active (non-deleted) transactions and merchants
--   - Filters by specific merchant_id
--   - Includes data for the last 5 calendar years up to the current year
--   - Groups by calendar year and payment method
--   - Results are ordered chronologically by year
-- name: GetYearlyPaymentMethodByApikey :many
WITH
    last_five_years AS (
        SELECT EXTRACT(
                YEAR
                FROM t.transaction_time
            ) AS year, t.payment_method, SUM(t.amount) AS total_amount
        FROM transactions t
            JOIN merchants m ON t.merchant_id = m.merchant_id
        WHERE
            t.deleted_at IS NULL
            AND m.deleted_at IS NULL
            AND m.api_key = $1
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
            ),
            t.payment_method
    )
SELECT year, payment_method, total_amount
FROM last_five_years
ORDER BY year;

-- GetMonthlyAmountByApikey: Retrieves total transaction amount per month for a specific merchant and year
-- Purpose: Generate monthly income report for a specific merchant regardless of payment method
-- Parameters:
--   $1: reference_date - Any date within the target year
--   $2: api-key - The merchant to filter transactions
-- Returns:
--   - Month name (e.g., Jan, Feb)
--   - Total transaction amount (0 if no activity)
-- Business Logic:
--   - Generates complete 12-month series for the given year
--   - Filters only active (non-deleted) transactions and merchants
--   - Filters by specific merchant_id
--   - Uses LEFT JOIN to ensure each month is represented
--   - Uses COALESCE to return 0 if a month has no transaction data
--   - Results are ordered chronologically by month
-- name: GetMonthlyAmountByApikey :many
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
    LEFT JOIN merchants mch ON t.merchant_id = mch.merchant_id
    AND mch.deleted_at IS NULL
    AND mch.api_key = $2
GROUP BY
    m.month
ORDER BY m.month;

-- GetYearlyAmountByMerchants: Retrieves total transaction amount per year for the last 5 years for a specific merchant
-- Purpose: Show overall yearly revenue trends for a merchant across all payment methods
-- Parameters:
--   $1: api-key - The merchant to filter transactions
--   $2: current_year - The latest year to include in the 5-year window
-- Returns:
--   - Year (e.g., 2021, 2022)
--   - Total transaction amount
-- Business Logic:
--   - Filters only active (non-deleted) transactions and merchants
--   - Filters by specific merchant_id
--   - Includes data for the last 5 calendar years up to the current year
--   - Groups by calendar year
--   - Results are ordered chronologically by year
-- name: GetYearlyAmountByApikey :many
WITH
    last_five_years AS (
        SELECT EXTRACT(
                YEAR
                FROM t.transaction_time
            ) AS year, SUM(t.amount) AS total_amount
        FROM transactions t
            JOIN merchants m ON t.merchant_id = m.merchant_id
        WHERE
            t.deleted_at IS NULL
            AND m.deleted_at IS NULL
            AND m.api_key = $1
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
SELECT year, total_amount
FROM last_five_years
ORDER BY year;

-- GetMonthlyTotalAmountByApikey: Retrieves total transaction amounts for the current and previous month
-- Purpose: Provide monthly transaction summary including zero values if no transactions exist
-- Parameters:
--   $1: reference_date - Any date within the target (current) month
--   $2: api-key - The merchant to filter transactions
-- Returns:
--   - Year (as text)
--   - Month (abbreviated name, e.g., Jan, Feb)
--   - Total transaction amount for each month
-- Business Logic:
--   - Aggregates total transaction amounts for the target month and the month before
--   - Filters only active (non-deleted) transactions and merchants
--   - Includes 0 as total_amount if there's no transaction data for either month
--   - Uses UNION ALL to combine real data with "missing month" placeholders
--   - Results are sorted by year and month (most recent first)
-- name: GetMonthlyTotalAmountByApikey :many
WITH
    monthly_data AS (
        SELECT EXTRACT(
                YEAR
                FROM t.transaction_time
            )::integer AS year, EXTRACT(
                MONTH
                FROM t.transaction_time
            )::integer AS month, COALESCE(SUM(t.amount), 0)::integer AS total_amount
        FROM transactions t
            INNER JOIN merchants m ON t.merchant_id = m.merchant_id
        WHERE
            t.deleted_at IS NULL
            AND m.deleted_at IS NULL
            AND EXTRACT(
                YEAR
                FROM t.transaction_time
            ) = EXTRACT(
                YEAR
                FROM $1::timestamp
            )
            AND m.api_key = $2
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
        SELECT year::text, TO_CHAR(
                TO_DATE(month::text, 'MM'), 'Mon'
            ) AS month, total_amount
        FROM monthly_data
        UNION ALL
        SELECT EXTRACT(
                YEAR
                FROM gs.month
            )::text AS year, TO_CHAR(gs.month, 'Mon') AS month, 0::integer AS total_amount
        FROM generate_series(
                date_trunc('year', $1::timestamp), date_trunc('year', $1::timestamp) + interval '11 month', interval '1 month'
            ) AS gs (month)
        WHERE
            NOT EXISTS (
                SELECT 1
                FROM monthly_data md
                WHERE
                    md.year = EXTRACT(
                        YEAR
                        FROM gs.month
                    )::integer
                    AND md.month = EXTRACT(
                        MONTH
                        FROM gs.month
                    )::integer
            )
    )
SELECT *
FROM formatted_data
ORDER BY year DESC, TO_DATE(month, 'Mon') DESC;

-- GetYearlyTotalAmountByApikey: Retrieves total transaction amounts for the current and previous year
-- Purpose: Provide yearly transaction summary with fallback to 0 if no transactions exist
-- Parameters:
--   $1: current_year - The latest year to include in the summary
--   $2: api-key - The merchant to filter transactions
-- Returns:
--   - Year (as text)
--   - Total transaction amount per year
-- Business Logic:
--   - Aggregates total amounts for both the current year and the previous year
--   - Filters only active (non-deleted) transactions and merchants
--   - Ensures both years appear in the result, even if no data exists (returns 0 in such case)
--   - Uses UNION ALL to combine actual data with 0-filled placeholders
--   - Results are ordered in descending order by year
-- name: GetYearlyTotalAmountByApikey :many
WITH
    yearly_data AS (
        SELECT EXTRACT(
                YEAR
                FROM t.transaction_time
            )::integer AS year, COALESCE(SUM(t.amount), 0)::integer AS total_amount
        FROM transactions t
            INNER JOIN merchants m ON t.merchant_id = m.merchant_id
        WHERE
            t.deleted_at IS NULL
            AND m.deleted_at IS NULL
            AND EXTRACT(
                YEAR
                FROM t.transaction_time
            ) >= $1::integer - 4
            AND EXTRACT(
                YEAR
                FROM t.transaction_time
            ) <= $1::integer
            AND m.api_key = $2
        GROUP BY
            EXTRACT(
                YEAR
                FROM t.transaction_time
            )
    ),
    formatted_data AS (
        SELECT year::text, total_amount
        FROM yearly_data
        UNION ALL
        SELECT y::text AS year, 0::integer AS total_amount
        FROM generate_series($1::integer - 4, $1::integer) AS y
        WHERE
            NOT EXISTS (
                SELECT 1
                FROM yearly_data yd
                WHERE
                    yd.year = y
            )
    )
SELECT *
FROM formatted_data
ORDER BY year DESC;

-- FindAllTransactions: Retrieves a paginated list of active transactions with optional search
-- Purpose: Display transaction list with merchant info, filtered by card number or payment method
-- Parameters:
--   $1: api-key - The merchant to filter transactions
--   $2: search_query (TEXT, nullable) - Optional search string to match card_number or payment_method
--   $3: limit (INTEGER) - Maximum number of records to return (pagination)
--   $4: offset (INTEGER) - Number of records to skip (pagination)
-- Returns:
--   - transaction_id
--   - card_number
--   - amount
--   - payment_method
--   - merchant_id
--   - merchant_name (from join with merchants table)
--   - transaction_time
--   - created_at, updated_at, deleted_at (for audit purposes)
--   - total_count: Total number of records matching the filter (useful for pagination metadata)
-- Business Logic:
--   - Joins `transactions` with `merchants` to retrieve merchant name
--   - Filters out soft-deleted transactions (where deleted_at IS NOT NULL)
--   - Applies case-insensitive partial match on card_number or payment_method if search query is provided
--   - Uses `COUNT(*) OVER()` to include total matching count for pagination without a separate query
--   - Results are ordered by `transaction_time` descending
-- name: FindAllTransactionsByApikey :many
SELECT
    t.transaction_id,
    t.card_number,
    t.amount,
    t.payment_method,
    t.merchant_id,
    m.name AS merchant_name,
    t.transaction_time,
    t.created_at,
    t.updated_at,
    t.deleted_at,
    COUNT(*) OVER () AS total_count
FROM transactions t
    JOIN merchants m ON t.merchant_id = m.merchant_id
WHERE
    t.deleted_at IS NULL
    AND m.api_key = $1
    AND (
        $2::TEXT IS NULL
        OR t.card_number ILIKE '%' || $2 || '%'
        OR t.payment_method ILIKE '%' || $2 || '%'
    )
ORDER BY t.transaction_time DESC
LIMIT $3
OFFSET
    $4;

-- Create Merchant
-- name: CreateMerchant :one
-- Purpose: Insert a new merchant record into the database
-- Parameters:
--   $1: name - The name of the merchant
--   $2: api_key - Unique API key for the merchant
--   $3: user_id - ID of the user associated with the merchant
--   $4: status - Current status of the merchant (e.g., active, inactive)
-- Returns:
--   - The newly created merchant record
-- Business Logic:
--   - Inserts a new merchant with the provided details.
--   - Sets the created_at and updated_at timestamps to the current time.
--   - Returns the created merchant's data using the RETURNING clause.
-- name: CreateMerchant :one
INSERT INTO
    merchants (
        name,
        api_key,
        user_id,
        status,
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
    merchant_id,
    merchant_no,
    name,
    api_key,
    user_id,
    status,
    created_at,
    updated_at;

-- Update Merchant
-- Purpose: Update an existing merchant record
-- Parameters:
--   $1: merchant_id - ID of the merchant to be updated
--   $2: name - The new name for the merchant
--   $3: user_id - New user ID associated with the merchant
--   $4: status - New status for the merchant
-- Business Logic:
--   - Updates the specified merchant's name, user_id, and status.
--   - Ensures the merchant is not marked as deleted (deleted_at is NULL).
--   - Sets the updated_at timestamp to the current time.
-- name: UpdateMerchant :one
UPDATE merchants
SET
    name = $2,
    user_id = $3,
    status = $4,
    updated_at = current_timestamp
WHERE
    merchant_id = $1
    AND deleted_at IS NULL
RETURNING
    merchant_id,
    merchant_no,
    name,
    api_key,
    user_id,
    status,
    created_at,
    updated_at;

-- Purpose: Update only the status of an existing merchant
-- Parameters:
--   $1: merchant_id - ID of the merchant to update
--   $2: status - New status to set for the merchant
-- Business Logic:
--   - Updates the status of the specified merchant.
--   - Ensures the merchant is not marked as deleted (deleted_at is NULL).
--   - Sets the updated_at timestamp to the current time.
-- name: UpdateMerchantStatus :one
UPDATE merchants
SET
    status = $2,
    updated_at = current_timestamp
WHERE
    merchant_id = $1
    AND deleted_at IS NULL
RETURNING
    merchant_id,
    merchant_no,
    name,
    api_key,
    user_id,
    status,
    created_at,
    updated_at;

-- Trash Merchant
-- Purpose: Mark a merchant as deleted (soft delete)
-- Parameters:
--   $1: merchant_id - ID of the merchant to be trashed
-- Business Logic:
--   - Sets the `deleted_at` timestamp to the current time for the specified merchant.
--   - Marks the merchant as deleted, without permanently removing it from the database.
--   - Ensures the merchant is not already marked as deleted (deleted_at is NULL).
-- name: TrashMerchant :one
UPDATE merchants
SET
    deleted_at = current_timestamp
WHERE
    merchant_id = $1
    AND deleted_at IS NULL
RETURNING
    merchant_id,
    merchant_no,
    name,
    api_key,
    user_id,
    status,
    created_at,
    updated_at,
    deleted_at;

-- Restore Trashed Merchant
-- Purpose: Restore a previously trashed (soft deleted) merchant
-- Parameters:
--   $1: merchant_id - ID of the merchant to restore
-- Business Logic:
--   - Resets the `deleted_at` field to NULL, restoring the merchant to an active state.
--   - Ensures the merchant is currently trashed (deleted_at is not NULL).
-- name: RestoreMerchant :one
UPDATE merchants
SET
    deleted_at = NULL
WHERE
    merchant_id = $1
    AND deleted_at IS NOT NULL
RETURNING
    merchant_id,
    merchant_no,
    name,
    api_key,
    user_id,
    status,
    created_at,
    updated_at,
    deleted_at;

-- Delete Merchant Permanently
-- Purpose: Permanently delete a merchant from the database
-- Parameters:
--   $1: merchant_id - ID of the merchant to be permanently deleted
-- Business Logic:
--   - Deletes the specified merchant from the database.
--   - Ensures the merchant is marked as deleted (deleted_at is not NULL).
-- name: DeleteMerchantPermanently :exec
DELETE FROM merchants
WHERE
    merchant_id = $1
    AND deleted_at IS NOT NULL;

-- Restore All Trashed Merchants
-- Purpose: Restore all merchants that are soft deleted
-- Business Logic:
--   - Resets the `deleted_at` field to NULL for all merchants that have been marked as deleted.
--   - Restores all merchants to an active state.
-- name: RestoreAllMerchants :exec
UPDATE merchants
SET
    deleted_at = NULL
WHERE
    deleted_at IS NOT NULL;

-- Delete All Trashed Merchants Permanently
-- name: DeleteAllPermanentMerchants :exec
-- Purpose: Permanently delete all merchants that are soft deleted
-- Business Logic:
--   - Permanently deletes all merchants that have been soft deleted (i.e., deleted_at is not NULL).
--   - Removes these merchants completely from the database.
-- name: DeleteAllPermanentMerchants :exec
DELETE FROM merchants WHERE deleted_at IS NOT NULL;