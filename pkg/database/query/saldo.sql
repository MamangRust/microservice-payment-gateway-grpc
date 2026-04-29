-- GetSaldos: Retrieves paginated list of active saldos with search capability
-- Purpose: List all active saldos for admin or user dashboard with optional filtering
-- Parameters:
--   $1: search_term - Optional text to filter saldos by card_number (NULL for no filter)
--   $2: limit - Maximum number of records to return
--   $3: offset - Number of records to skip for pagination
-- Returns:
--   All saldo fields plus total_count of matching records
-- Business Logic:
--   - Excludes soft-deleted saldos (deleted_at IS NULL)
--   - Supports partial text matching on card_number (case-insensitive)
--   - Returns saldos ordered by saldo_id
--   - Provides total_count for pagination calculations
-- name: GetSaldos :many
SELECT
    saldo_id,
    card_number,
    total_balance,
    withdraw_amount,
    withdraw_time,
    created_at,
    updated_at,
    COUNT(*) OVER () AS total_count
FROM saldos
WHERE
    deleted_at IS NULL
    AND (
        $1::TEXT IS NULL
        OR card_number ILIKE '%' || $1 || '%'
    )
ORDER BY saldo_id
LIMIT $2
OFFSET
    $3;

-- GetSaldoByID: Retrieves single active saldo by ID
-- Purpose: Fetch a specific saldo record for display or processing
-- Parameters:
--   $1: saldo_id - Unique identifier of the saldo
-- Returns:
--   Single saldo record if it is active (deleted_at IS NULL)
-- Business Logic:
--   - Ensures only active saldos are returned (soft-deleted saldos are excluded)
--   - Used for detail views or transaction lookups
-- name: GetSaldoByID :one
SELECT
    saldo_id,
    card_number,
    total_balance,
    withdraw_amount,
    withdraw_time,
    created_at,
    updated_at
FROM saldos
WHERE
    saldo_id = $1
    AND deleted_at IS NULL;

-- GetActiveSaldos: Retrieves active saldos with pagination and optional search
-- Purpose: List all non-deleted saldos with optional filtering for administrative views
-- Parameters:
--   $1: search_term - Optional filter by card_number (case-insensitive, NULL for no filter)
--   $2: limit - Number of records to retrieve
--   $3: offset - Records to skip (pagination)
-- Returns:
--   Active saldo records with total_count for pagination
-- Business Logic:
--   - Filters out trashed saldos (deleted_at IS NULL)
--   - Supports partial matching on card_number
--   - Results ordered by saldo_id
-- name: GetActiveSaldos :many
SELECT
    saldo_id,
    card_number,
    total_balance,
    withdraw_amount,
    withdraw_time,
    created_at,
    updated_at,
    deleted_at,
    COUNT(*) OVER () AS total_count
FROM saldos
WHERE
    deleted_at IS NULL
    AND (
        $1::TEXT IS NULL
        OR card_number ILIKE '%' || $1 || '%'
    )
ORDER BY saldo_id
LIMIT $2
OFFSET
    $3;

-- GetTrashedSaldos: Retrieves soft-deleted saldos with search and pagination
-- Purpose: Display trashed saldos for recovery or permanent deletion
-- Parameters:
--   $1: search_term - Optional search by card_number (NULL for no filter)
--   $2: limit - Max number of records
--   $3: offset - Number of rows to skip
-- Returns:
--   List of trashed saldos and total_count of matches
-- Business Logic:
--   - Includes only soft-deleted saldos (deleted_at IS NOT NULL)
--   - Partial match on card_number
--   - Useful for building a "Trash Bin" feature in the UI
-- name: GetTrashedSaldos :many
SELECT
    saldo_id,
    card_number,
    total_balance,
    withdraw_amount,
    withdraw_time,
    created_at,
    updated_at,
    deleted_at,
    COUNT(*) OVER () AS total_count
FROM saldos
WHERE
    deleted_at IS NOT NULL
    AND (
        $1::TEXT IS NULL
        OR card_number ILIKE '%' || $1 || '%'
    )
ORDER BY saldo_id
LIMIT $2
OFFSET
    $3;

-- GetTrashedSaldoByID: Retrieves a single soft-deleted saldo record by ID
-- Purpose: View details of a trashed saldo for recovery or audit purposes
-- Parameters:
--   $1: saldo_id - The ID of the saldo record to retrieve
-- Returns:
--   All fields for the specified trashed saldo or NULL if not found/not deleted
-- Business Logic:
--   - Only returns soft-deleted saldos (deleted_at IS NOT NULL)
--   - Useful for admin interfaces showing deleted items
--   - Can be used before restoring a deleted saldo
-- name: GetTrashedSaldoByID :one
SELECT * FROM saldos WHERE saldo_id = $1 AND deleted_at IS NOT NULL;

-- GetMonthlyTotalSaldoBalance: Retrieves monthly balance totals for comparison periods
-- Purpose: Compare monthly balance trends between two time periods
-- Parameters:
--   $1: period1_start - Start date of first comparison period
--   $2: period1_end - End date of first comparison period
--   $3: period2_start - Start date of second comparison period
--   $4: period2_end - End date of second comparison period
-- Returns:
--   year: The year as text
--   month: 3-letter month abbreviation
--   total_balance: Monthly balance total (0 if no data)
-- Business Logic:
--   - Aggregates balances for two customizable time periods
--   - Only includes active saldos (deleted_at IS NULL)
--   - Ensures both periods' months appear with zero-filling
--   - Formats output for consistent visualization
--   - Results ordered by year and month (newest first)
-- name: GetMonthlyTotalSaldoBalance :many
WITH
    monthly_data AS (
        SELECT EXTRACT(
                YEAR
                FROM s.created_at
            )::integer AS year, EXTRACT(
                MONTH
                FROM s.created_at
            )::integer AS month, COALESCE(SUM(s.total_balance), 0) AS total_balance
        FROM saldos s
        WHERE
            s.deleted_at IS NULL
            AND (
                (
                    s.created_at >= $1::timestamp
                    AND s.created_at <= $2::timestamp
                )
                OR (
                    s.created_at >= $3::timestamp
                    AND s.created_at <= $4::timestamp
                )
            )
        GROUP BY
            EXTRACT(
                YEAR
                FROM s.created_at
            ),
            EXTRACT(
                MONTH
                FROM s.created_at
            )
    ),
    formatted_data AS (
        SELECT year::text, TO_CHAR(
                TO_DATE(month::text, 'MM'), 'Mon'
            ) AS month, total_balance::integer
        FROM monthly_data
        UNION ALL
        SELECT EXTRACT(
                YEAR
                FROM $1::timestamp
            )::text AS year, TO_CHAR($1::timestamp, 'Mon') AS month, 0::integer AS total_balance
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
        SELECT EXTRACT(
                YEAR
                FROM $3::timestamp
            )::text AS year, TO_CHAR($3::timestamp, 'Mon') AS month, 0::integer AS total_balance
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

-- GetYearlyTotalSaldoBalances: Retrieves yearly balance totals for current and previous year
-- Purpose: Compare annual balance trends between current and previous year
-- Parameters:
--   $1: current_year - The year to analyze (includes this year and previous)
-- Returns:
--   year: The year as text
--   total_balance: Annual balance total (0 if no data)
-- Business Logic:
--   - Shows comparison between specified year and previous year
--   - Only includes active saldos (deleted_at IS NULL)
--   - Ensures both years appear with zero-filling if missing
--   - Results ordered by year (newest first)
--   - Useful for year-over-year financial analysis
-- name: GetYearlyTotalSaldoBalances :many
WITH
    yearly_data AS (
        SELECT EXTRACT(
                YEAR
                FROM s.created_at
            )::integer AS year, COALESCE(SUM(s.total_balance), 0)::integer AS total_balance
        FROM saldos s
        WHERE
            s.deleted_at IS NULL
            AND (
                EXTRACT(
                    YEAR
                    FROM s.created_at
                ) = $1::integer
                OR EXTRACT(
                    YEAR
                    FROM s.created_at
                ) = $1::integer - 1
            )
        GROUP BY
            EXTRACT(
                YEAR
                FROM s.created_at
            )
    ),
    formatted_data AS (
        SELECT year::text, total_balance::integer
        FROM yearly_data
        UNION ALL
        SELECT $1::text AS year, 0::integer AS total_balance
        WHERE
            NOT EXISTS (
                SELECT 1
                FROM yearly_data
                WHERE
                    year = $1::integer
            )
        UNION ALL
        SELECT ($1::integer - 1)::text AS year, 0::integer AS total_balance
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

-- GetMonthlySaldoBalances: Retrieves monthly balance totals for a given year
-- Purpose: Provide monthly balance trends for financial reporting and dashboards
-- Parameters:
--   $1: reference_date - A date used to determine the year to analyze
-- Returns:
--   month: 3-letter month abbreviation (e.g., 'Jan', 'Feb')
--   total_balance: Sum of balances for that month (0 if no data exists)
-- Business Logic:
--   - Generates a complete 12-month series for the specified year
--   - Uses LEFT JOIN to ensure all months appear in results
--   - COALESCE returns 0 for months with no balance data
--   - Only includes active saldo records (deleted_at IS NULL)
--   - Groups by month and orders chronologically
--   - Useful for cash flow analysis and financial planning
-- name: GetMonthlySaldoBalances :many
WITH
    months AS (
        SELECT generate_series(
                date_trunc('year', $1::timestamp), date_trunc('year', $1::timestamp) + interval '1 year' - interval '1 day', interval '1 month'
            ) AS month
    )
SELECT TO_CHAR(m.month, 'Mon') AS month, COALESCE(SUM(s.total_balance), 0)::int AS total_balance
FROM months m
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
GROUP BY
    m.month
ORDER BY m.month;

-- GetYearlySaldoBalances: Retrieves yearly balance totals for a 5-year period
-- Purpose: Show annual balance trends for long-term financial analysis
-- Parameters:
--   $1: reference_year - The target year (includes this year plus previous 4 years)
-- Returns:
--   year: The 4-digit year
--   total_balance: Sum of balances for that year
-- Business Logic:
--   - Covers a 5-year rolling window (reference_year-4 to reference_year)
--   - Only includes active saldo records (deleted_at IS NULL)
--   - Groups by calendar year
--   - Results ordered chronologically
--   - Useful for identifying year-over-year trends and growth patterns
-- name: GetYearlySaldoBalances :many
WITH
    last_five_years AS (
        SELECT EXTRACT(
                YEAR
                FROM s.created_at
            ) AS year, SUM(s.total_balance) AS total_balance
        FROM saldos s
        WHERE
            s.deleted_at IS NULL
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

-- GetSaldoByCardNumber: Retrieves saldo information for a specific card
-- Purpose: Get the current balance and details for a particular card
-- Parameters:
--   $1: card_number - The card number to lookup
-- Returns:
--   All saldo fields for the active record matching the card number
-- Business Logic:
--   - Only returns active saldo records (deleted_at IS NULL)
--   - Useful for checking card balances before transactions
-- name: GetSaldoByCardNumber :one
SELECT * FROM saldos WHERE card_number = $1 AND deleted_at IS NULL;

-- CreateSaldo: Creates a new saldo record
-- Purpose: Initialize a balance record for a new card
-- Parameters:
--   $1: card_number - The card number to associate with this saldo
--   $2: total_balance - The initial balance amount
-- Returns:
--   The newly created saldo record
-- Business Logic:
--   - Sets creation and update timestamps automatically
--   - Used when issuing new cards
-- name: CreateSaldo :one
INSERT INTO
    saldos (
        card_number,
        total_balance,
        created_at,
        updated_at
    )
VALUES (
        $1,
        $2,
        current_timestamp,
        current_timestamp
    )
RETURNING
    saldo_id,
    card_number,
    total_balance,
    withdraw_amount,
    withdraw_time,
    created_at,
    updated_at;

-- UpdateSaldo: Modifies saldo record details
-- Purpose: Update card number and balance for an existing saldo
-- Parameters:
--   $1: saldo_id - The ID of the saldo to update
--   $2: card_number - New card number to associate
--   $3: total_balance - New balance amount
-- Business Logic:
--   - Only updates active records (deleted_at IS NULL)
--   - Automatically updates the modification timestamp
--   - Useful for administrative corrections
-- name: UpdateSaldo :one
UPDATE saldos
SET
    card_number = $2,
    total_balance = $3,
    updated_at = current_timestamp
WHERE
    saldo_id = $1
    AND deleted_at IS NULL
RETURNING
    saldo_id,
    card_number,
    total_balance,
    withdraw_amount,
    withdraw_time,
    created_at,
    updated_at;
;
-- UpdateSaldoBalance: Updates only the balance amount for a card
-- Purpose: Adjust card balance without changing card association
-- Parameters:
--   $1: card_number - The card number to update
--   $2: total_balance - New balance amount
-- Business Logic:
--   - Card-specific update (uses card_number instead of saldo_id)
--   - Only updates active records
--   - Useful for balance adjustments and corrections
-- name: UpdateSaldoBalance :one
UPDATE saldos
SET
    total_balance = $2,
    updated_at = current_timestamp
WHERE
    card_number = $1
    AND deleted_at IS NULL
RETURNING
    saldo_id,
    card_number,
    total_balance,
    withdraw_amount,
    withdraw_time,
    created_at,
    updated_at;

-- UpdateSaldoWithdraw: Processes a withdrawal transaction
-- Purpose: Record a withdrawal and update the remaining balance
-- Parameters:
--   $1: card_number - The card used for withdrawal
--   $2: withdraw_amount - The amount being withdrawn
--   $3: withdraw_time - Timestamp of the withdrawal
-- Business Logic:
--   - Only processes if sufficient balance exists (total_balance >= $2)
--   - Updates both withdrawal amount and remaining balance
--   - Records withdrawal timestamp
--   - Only affects active records
-- name: UpdateSaldoWithdraw :one
UPDATE saldos
SET
    withdraw_amount = $2,
    total_balance = total_balance - $2,
    withdraw_time = $3,
    updated_at = current_timestamp
WHERE
    card_number = $1
    AND deleted_at IS NULL
    AND total_balance >= $2
RETURNING
    saldo_id,
    card_number,
    total_balance,
    withdraw_amount,
    withdraw_time,
    created_at,
    updated_at;

-- TrashSaldo: Soft-deletes a saldo record
-- Purpose: Remove a saldo from active use without permanent deletion
-- Parameters:
--   $1: saldo_id - The ID of the saldo to trash
-- Business Logic:
--   - Sets deleted_at timestamp
--   - Only affects currently active records
--   - Preserves data for possible recovery
-- name: TrashSaldo :one
UPDATE saldos
SET
    deleted_at = current_timestamp
WHERE
    saldo_id = $1
    AND deleted_at IS NULL
RETURNING
    saldo_id,
    card_number,
    total_balance,
    withdraw_amount,
    withdraw_time,
    created_at,
    updated_at,
    deleted_at;

-- RestoreSaldo: Recovers a soft-deleted saldo
-- Purpose: Reactivate a previously trashed saldo record
-- Parameters:
--   $1: saldo_id - The ID of the saldo to restore
-- Business Logic:
--   - Clears the deleted_at timestamp
--   - Only works on currently trashed records
-- name: RestoreSaldo :one
UPDATE saldos
SET
    deleted_at = NULL
WHERE
    saldo_id = $1
    AND deleted_at IS NOT NULL
RETURNING
    saldo_id,
    card_number,
    total_balance,
    withdraw_amount,
    withdraw_time,
    created_at,
    updated_at,
    deleted_at;

-- DeleteSaldoPermanently: Hard-deletes a trashed saldo
-- Purpose: Permanently remove a previously soft-deleted record
-- Parameters:
--   $1: saldo_id - The ID of the saldo to delete
-- Business Logic:
--   - Physical deletion from database
--   - Only works on already trashed records
--   - Irreversible operation
-- name: DeleteSaldoPermanently :exec
DELETE FROM saldos WHERE saldo_id = $1 AND deleted_at IS NOT NULL;

-- RestoreAllSaldos: Recovers all trashed saldo records
-- Purpose: Mass restoration of deleted saldos
-- Business Logic:
--   - Clears deleted_at for all trashed records
--   - Useful for system recovery scenarios
-- name: RestoreAllSaldos :exec
UPDATE saldos
SET
    deleted_at = NULL
WHERE
    deleted_at IS NOT NULL;

-- DeleteAllPermanentSaldos: Permanently removes all trashed saldos
-- Purpose: Clean up all soft-deleted records
-- Business Logic:
--   - Irreversible bulk deletion
--   - Only affects records marked as deleted
--   - Frees database space from old records
-- name: DeleteAllPermanentSaldos :exec
DELETE FROM saldos WHERE deleted_at IS NOT NULL;