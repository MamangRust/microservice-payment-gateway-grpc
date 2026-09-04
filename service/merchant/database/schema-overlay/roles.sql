-- sqlc-only schema overlay (NOT applied by goose).
--
-- Every payment-gateway service runs role lookups (roleRepository wraps
-- GetRole/GetRoleByName) against the shared database. The roles/user_roles
-- tables belong to the role service; this overlay exists only so sqlc can
-- type-check those queries in each service's schema package.
CREATE TABLE IF NOT EXISTS "roles" (
    "role_id" SERIAL PRIMARY KEY,
    "role_name" VARCHAR(50) UNIQUE NOT NULL,
    "created_at" timestamp DEFAULT current_timestamp,
    "updated_at" timestamp DEFAULT current_timestamp,
    "deleted_at" TIMESTAMP DEFAULT NULL
);
CREATE INDEX idx_roles_role_name ON roles (role_name);
CREATE INDEX idx_roles_created_at ON roles (created_at);
CREATE INDEX idx_roles_updated_at ON roles (updated_at);

CREATE TABLE IF NOT EXISTS "user_roles" (
    "user_role_id" SERIAL PRIMARY KEY,
    "user_id" INT NOT NULL,
    "role_id" INT NOT NULL REFERENCES "roles" ("role_id") ON DELETE CASCADE,
    "created_at" timestamp DEFAULT current_timestamp,
    "updated_at" timestamp DEFAULT current_timestamp,
    "deleted_at" TIMESTAMP DEFAULT NULL
);
CREATE INDEX idx_user_roles_user_id ON user_roles (user_id);
CREATE INDEX idx_user_roles_role_id ON user_roles (role_id);
CREATE INDEX idx_user_roles_user_id_role_id ON user_roles (user_id, role_id);
CREATE INDEX idx_user_roles_created_at ON user_roles (created_at);
CREATE INDEX idx_user_roles_updated_at ON user_roles (updated_at);
