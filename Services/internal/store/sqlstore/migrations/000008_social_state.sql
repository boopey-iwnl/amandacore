CREATE TABLE IF NOT EXISTS ac_friend_links (
    owner_character_id TEXT NOT NULL,
    friend_character_id TEXT NOT NULL,
    friend_display_name TEXT NOT NULL DEFAULT '',
    created_at INTEGER NOT NULL,
    PRIMARY KEY (owner_character_id, friend_character_id),
    CHECK (owner_character_id <> friend_character_id),
    FOREIGN KEY (owner_character_id) REFERENCES ac_characters(id) ON DELETE CASCADE,
    FOREIGN KEY (friend_character_id) REFERENCES ac_characters(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_ac_friend_links_friend
    ON ac_friend_links(friend_character_id);

CREATE TABLE IF NOT EXISTS ac_ignore_links (
    owner_character_id TEXT NOT NULL,
    ignored_character_id TEXT NOT NULL,
    created_at INTEGER NOT NULL,
    reason TEXT NOT NULL DEFAULT '',
    PRIMARY KEY (owner_character_id, ignored_character_id),
    CHECK (owner_character_id <> ignored_character_id),
    FOREIGN KEY (owner_character_id) REFERENCES ac_characters(id) ON DELETE CASCADE,
    FOREIGN KEY (ignored_character_id) REFERENCES ac_characters(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS ac_parties (
    party_id TEXT PRIMARY KEY,
    leader_character_id TEXT NOT NULL,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,
    disbanded_at INTEGER NOT NULL DEFAULT 0,
    version INTEGER NOT NULL DEFAULT 1,
    FOREIGN KEY (leader_character_id) REFERENCES ac_characters(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS ac_party_members (
    party_id TEXT NOT NULL,
    character_id TEXT NOT NULL,
    role_id TEXT NOT NULL DEFAULT 'member',
    joined_at INTEGER NOT NULL,
    left_at INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (party_id, character_id),
    FOREIGN KEY (party_id) REFERENCES ac_parties(party_id) ON DELETE CASCADE,
    FOREIGN KEY (character_id) REFERENCES ac_characters(id) ON DELETE CASCADE
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_ac_party_members_active_character
    ON ac_party_members(character_id)
    WHERE left_at = 0;

CREATE INDEX IF NOT EXISTS idx_ac_party_members_party
    ON ac_party_members(party_id, left_at);

CREATE TABLE IF NOT EXISTS ac_party_invites (
    invite_id TEXT PRIMARY KEY,
    party_id TEXT NOT NULL DEFAULT '',
    inviter_character_id TEXT NOT NULL,
    target_character_id TEXT NOT NULL,
    state TEXT NOT NULL,
    created_at INTEGER NOT NULL,
    expires_at INTEGER NOT NULL,
    responded_at INTEGER NOT NULL DEFAULT 0,
    mutation_key TEXT NOT NULL DEFAULT '',
    FOREIGN KEY (inviter_character_id) REFERENCES ac_characters(id) ON DELETE CASCADE,
    FOREIGN KEY (target_character_id) REFERENCES ac_characters(id) ON DELETE CASCADE
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_ac_party_invites_pending_target
    ON ac_party_invites(inviter_character_id, target_character_id)
    WHERE state = 'pending';

CREATE TABLE IF NOT EXISTS ac_guilds (
    guild_id TEXT PRIMARY KEY,
    realm_id TEXT NOT NULL,
    guild_name TEXT NOT NULL,
    normalized_guild_name TEXT NOT NULL,
    leader_character_id TEXT NOT NULL,
    created_by_character_id TEXT NOT NULL,
    motd TEXT NOT NULL DEFAULT '',
    ranks_json TEXT NOT NULL,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,
    disbanded_at INTEGER NOT NULL DEFAULT 0,
    version INTEGER NOT NULL DEFAULT 1,
    FOREIGN KEY (realm_id) REFERENCES ac_realms(id) ON DELETE CASCADE,
    FOREIGN KEY (leader_character_id) REFERENCES ac_characters(id) ON DELETE CASCADE,
    FOREIGN KEY (created_by_character_id) REFERENCES ac_characters(id) ON DELETE CASCADE
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_ac_guilds_realm_name_active
    ON ac_guilds(realm_id, normalized_guild_name)
    WHERE disbanded_at = 0;

CREATE TABLE IF NOT EXISTS ac_guild_members (
    guild_id TEXT NOT NULL,
    character_id TEXT NOT NULL,
    display_name TEXT NOT NULL,
    race_id TEXT NOT NULL,
    class_id TEXT NOT NULL,
    level INTEGER NOT NULL,
    rank_id TEXT NOT NULL,
    joined_at INTEGER NOT NULL,
    last_online_at INTEGER NOT NULL DEFAULT 0,
    left_at INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (guild_id, character_id),
    FOREIGN KEY (guild_id) REFERENCES ac_guilds(guild_id) ON DELETE CASCADE,
    FOREIGN KEY (character_id) REFERENCES ac_characters(id) ON DELETE CASCADE
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_ac_guild_members_active_character
    ON ac_guild_members(character_id)
    WHERE left_at = 0;

CREATE TABLE IF NOT EXISTS ac_guild_invites (
    invite_id TEXT PRIMARY KEY,
    guild_id TEXT NOT NULL,
    guild_name TEXT NOT NULL,
    inviter_character_id TEXT NOT NULL,
    target_character_id TEXT NOT NULL,
    state TEXT NOT NULL DEFAULT 'pending',
    created_at INTEGER NOT NULL,
    expires_at INTEGER NOT NULL,
    responded_at INTEGER NOT NULL DEFAULT 0,
    mutation_key TEXT NOT NULL DEFAULT '',
    FOREIGN KEY (guild_id) REFERENCES ac_guilds(guild_id) ON DELETE CASCADE,
    FOREIGN KEY (inviter_character_id) REFERENCES ac_characters(id) ON DELETE CASCADE,
    FOREIGN KEY (target_character_id) REFERENCES ac_characters(id) ON DELETE CASCADE
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_ac_guild_invites_pending_target
    ON ac_guild_invites(guild_id, target_character_id)
    WHERE state = 'pending';

CREATE TABLE IF NOT EXISTS ac_chat_messages (
    message_id TEXT PRIMARY KEY,
    channel TEXT NOT NULL,
    scope_id TEXT NOT NULL DEFAULT '',
    sender_character_id TEXT NOT NULL,
    sender_display_name TEXT NOT NULL,
    target_character_id TEXT NOT NULL DEFAULT '',
    party_id TEXT NOT NULL DEFAULT '',
    guild_id TEXT NOT NULL DEFAULT '',
    zone_id TEXT NOT NULL DEFAULT '',
    message_text TEXT NOT NULL,
    sequence INTEGER NOT NULL,
    created_at INTEGER NOT NULL,
    FOREIGN KEY (sender_character_id) REFERENCES ac_characters(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_ac_chat_messages_scope_sequence
    ON ac_chat_messages(channel, scope_id, sequence);

CREATE TABLE IF NOT EXISTS ac_social_mutations (
    mutation_id TEXT PRIMARY KEY,
    actor_character_id TEXT NOT NULL,
    operation TEXT NOT NULL,
    mutation_key TEXT NOT NULL,
    response_json TEXT NOT NULL,
    created_at INTEGER NOT NULL,
    UNIQUE (actor_character_id, operation, mutation_key),
    FOREIGN KEY (actor_character_id) REFERENCES ac_characters(id) ON DELETE CASCADE
);
