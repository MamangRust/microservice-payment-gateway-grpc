-- sqlc-only schema overlay (NOT applied by goose).
--
-- The auth service's resetTokenRepository wraps the user service's
-- reset_token queries against the shared database. This overlay exists only so
-- sqlc can type-check those queries in auth's schema package.
CREATE TABLE "reset_tokens" (
    "id" SERIAL PRIMARY KEY,
    "user_id" INT NOT NULL UNIQUE REFERENCES users(user_id) ON DELETE CASCADE,
    "token" TEXT NOT NULL UNIQUE,
    "expiry_date" TIMESTAMP NOT NULL
);
