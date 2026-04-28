CREATE TABLE IF NOT EXISTS ac_realms (
    id TEXT PRIMARY KEY,
    display_name TEXT NOT NULL,
    region TEXT NOT NULL,
    endpoint TEXT NOT NULL,
    supported_build TEXT NOT NULL,
    online_players INTEGER NOT NULL DEFAULT 0,
    online INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS ac_characters (
    id TEXT PRIMARY KEY,
    account_id TEXT NOT NULL,
    realm_id TEXT NOT NULL,
    display_name TEXT NOT NULL,
    normalized_display_name TEXT NOT NULL,
    race_id TEXT NOT NULL,
    class_id TEXT NOT NULL,
    archetype_id TEXT NOT NULL,
    level INTEGER NOT NULL,
    experience INTEGER NOT NULL,
    currency_copper INTEGER NOT NULL,
    zone_id TEXT NOT NULL,
    position_x REAL NOT NULL,
    position_y REAL NOT NULL,
    position_z REAL NOT NULL,
    equipment_json TEXT NOT NULL,
    professions_json TEXT NOT NULL,
    talents_json TEXT NOT NULL,
    kill_credits_json TEXT NOT NULL,
    tracked_quest_ids_json TEXT NOT NULL,
    pvp_stats_json TEXT NOT NULL,
    bind_point_json TEXT NOT NULL,
    travel_state_json TEXT NOT NULL,
    mount_state_json TEXT NOT NULL,
    last_seen_at INTEGER NOT NULL,
    UNIQUE (realm_id, normalized_display_name),
    FOREIGN KEY (account_id) REFERENCES ac_accounts(id) ON DELETE CASCADE,
    FOREIGN KEY (realm_id) REFERENCES ac_realms(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_ac_characters_account_realm ON ac_characters(account_id, realm_id);

CREATE TABLE IF NOT EXISTS ac_character_stats (
    character_id TEXT NOT NULL,
    stat_id TEXT NOT NULL,
    stat_value REAL NOT NULL,
    updated_at INTEGER NOT NULL,
    PRIMARY KEY (character_id, stat_id),
    FOREIGN KEY (character_id) REFERENCES ac_characters(id) ON DELETE CASCADE
);
