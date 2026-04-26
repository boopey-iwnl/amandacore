CREATE TABLE IF NOT EXISTS ac_schema_migrations (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    checksum TEXT NOT NULL,
    applied_at INTEGER NOT NULL,
    duration_ms INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS ac_accounts (
    id TEXT PRIMARY KEY,
    username TEXT NOT NULL,
    normalized_username TEXT NOT NULL UNIQUE,
    roles_json TEXT NOT NULL,
    banned INTEGER NOT NULL DEFAULT 0,
    suspended_until INTEGER NOT NULL DEFAULT 0,
    suspension_reason TEXT NOT NULL DEFAULT '',
    last_login_at INTEGER NOT NULL DEFAULT 0,
    last_session_id TEXT NOT NULL DEFAULT '',
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS ac_account_credentials (
    account_id TEXT PRIMARY KEY,
    password_hash TEXT NOT NULL,
    updated_at INTEGER NOT NULL,
    FOREIGN KEY (account_id) REFERENCES ac_accounts(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS ac_sessions (
    id TEXT PRIMARY KEY,
    account_id TEXT NOT NULL,
    access_token TEXT NOT NULL UNIQUE,
    refresh_token TEXT NOT NULL UNIQUE,
    access_expires_at INTEGER NOT NULL,
    refresh_expires_at INTEGER NOT NULL,
    created_at INTEGER NOT NULL,
    revoked_at INTEGER NOT NULL DEFAULT 0,
    FOREIGN KEY (account_id) REFERENCES ac_accounts(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_ac_sessions_account_id ON ac_sessions(account_id);

CREATE TABLE IF NOT EXISTS ac_password_resets (
    id TEXT PRIMARY KEY,
    account_id TEXT NOT NULL,
    expires_at INTEGER NOT NULL,
    consumed_at INTEGER NOT NULL DEFAULT 0,
    FOREIGN KEY (account_id) REFERENCES ac_accounts(id) ON DELETE CASCADE
);
