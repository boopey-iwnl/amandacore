CREATE TABLE IF NOT EXISTS ac_world_join_tickets (
    ticket_id TEXT PRIMARY KEY,
    session_id TEXT NOT NULL,
    account_id TEXT NOT NULL,
    character_id TEXT NOT NULL,
    realm_id TEXT NOT NULL,
    world_endpoint TEXT NOT NULL,
    expires_at INTEGER NOT NULL,
    consumed_at INTEGER NOT NULL DEFAULT 0,
    FOREIGN KEY (session_id) REFERENCES ac_sessions(id) ON DELETE CASCADE,
    FOREIGN KEY (account_id) REFERENCES ac_accounts(id) ON DELETE CASCADE,
    FOREIGN KEY (character_id) REFERENCES ac_characters(id) ON DELETE CASCADE,
    FOREIGN KEY (realm_id) REFERENCES ac_realms(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_ac_world_join_tickets_character ON ac_world_join_tickets(character_id);

CREATE TABLE IF NOT EXISTS ac_world_sessions (
    world_session_token TEXT PRIMARY KEY,
    account_id TEXT NOT NULL,
    character_id TEXT NOT NULL,
    realm_id TEXT NOT NULL,
    zone_id TEXT NOT NULL,
    connected INTEGER NOT NULL DEFAULT 1,
    position_x REAL NOT NULL,
    position_y REAL NOT NULL,
    position_z REAL NOT NULL,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,
    FOREIGN KEY (account_id) REFERENCES ac_accounts(id) ON DELETE CASCADE,
    FOREIGN KEY (character_id) REFERENCES ac_characters(id) ON DELETE CASCADE,
    FOREIGN KEY (realm_id) REFERENCES ac_realms(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS ac_character_position_snapshots (
    snapshot_id TEXT PRIMARY KEY,
    character_id TEXT NOT NULL,
    zone_id TEXT NOT NULL,
    position_x REAL NOT NULL,
    position_y REAL NOT NULL,
    position_z REAL NOT NULL,
    captured_at INTEGER NOT NULL,
    reason TEXT NOT NULL DEFAULT '',
    FOREIGN KEY (character_id) REFERENCES ac_characters(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_ac_character_position_snapshots_character ON ac_character_position_snapshots(character_id, captured_at);
