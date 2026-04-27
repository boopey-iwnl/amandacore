ALTER TABLE ac_currency_ledger ADD COLUMN operation TEXT NOT NULL DEFAULT 'currency.adjust';
ALTER TABLE ac_currency_ledger ADD COLUMN source_kind TEXT NOT NULL DEFAULT '';
ALTER TABLE ac_currency_ledger ADD COLUMN source_id TEXT NOT NULL DEFAULT '';
ALTER TABLE ac_currency_ledger ADD COLUMN actor_character_id TEXT NOT NULL DEFAULT '';

CREATE UNIQUE INDEX IF NOT EXISTS idx_ac_currency_ledger_idempotency
    ON ac_currency_ledger(character_id, operation, mutation_key)
    WHERE mutation_key <> '';

CREATE TABLE IF NOT EXISTS ac_auction_listings (
    auction_id TEXT PRIMARY KEY,
    realm_id TEXT NOT NULL,
    seller_character_id TEXT NOT NULL,
    seller_display_name TEXT NOT NULL,
    buyer_character_id TEXT NOT NULL DEFAULT '',
    item_id TEXT NOT NULL,
    item_display_name TEXT NOT NULL,
    item_quality TEXT NOT NULL DEFAULT '',
    item_type TEXT NOT NULL DEFAULT '',
    item_subtype TEXT NOT NULL DEFAULT '',
    item_stackable INTEGER NOT NULL DEFAULT 0,
    item_max_stack INTEGER NOT NULL DEFAULT 1,
    stack_count INTEGER NOT NULL,
    buyout_copper INTEGER NOT NULL,
    bid_copper INTEGER NOT NULL DEFAULT 0,
    current_bid_copper INTEGER NOT NULL DEFAULT 0,
    current_bidder_character_id TEXT NOT NULL DEFAULT '',
    deposit_copper INTEGER NOT NULL DEFAULT 0,
    cut_copper INTEGER NOT NULL DEFAULT 0,
    cut_percent INTEGER NOT NULL DEFAULT 5,
    created_at INTEGER NOT NULL,
    expires_at INTEGER NOT NULL,
    sold_at INTEGER NOT NULL DEFAULT 0,
    canceled_at INTEGER NOT NULL DEFAULT 0,
    state TEXT NOT NULL,
    source_inventory_slot INTEGER NOT NULL,
    version INTEGER NOT NULL DEFAULT 1,
    item_delivered_mail_id TEXT NOT NULL DEFAULT '',
    proceeds_delivered_mail_id TEXT NOT NULL DEFAULT '',
    return_delivered_mail_id TEXT NOT NULL DEFAULT '',
    mutation_key TEXT NOT NULL DEFAULT '',
    FOREIGN KEY (realm_id) REFERENCES ac_realms(id) ON DELETE CASCADE,
    FOREIGN KEY (seller_character_id) REFERENCES ac_characters(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_ac_auction_listings_realm_state
    ON ac_auction_listings(realm_id, state, expires_at);

CREATE INDEX IF NOT EXISTS idx_ac_auction_listings_seller
    ON ac_auction_listings(seller_character_id, created_at);

CREATE UNIQUE INDEX IF NOT EXISTS idx_ac_auction_listings_idempotency
    ON ac_auction_listings(seller_character_id, mutation_key)
    WHERE mutation_key <> '';

CREATE TABLE IF NOT EXISTS ac_auction_transactions (
    transaction_id TEXT PRIMARY KEY,
    auction_id TEXT NOT NULL,
    transaction_type TEXT NOT NULL,
    actor_character_id TEXT NOT NULL,
    counterparty_character_id TEXT NOT NULL DEFAULT '',
    currency_copper INTEGER NOT NULL DEFAULT 0,
    item_id TEXT NOT NULL DEFAULT '',
    stack_count INTEGER NOT NULL DEFAULT 0,
    mutation_key TEXT NOT NULL DEFAULT '',
    created_at INTEGER NOT NULL,
    FOREIGN KEY (auction_id) REFERENCES ac_auction_listings(auction_id) ON DELETE CASCADE,
    FOREIGN KEY (actor_character_id) REFERENCES ac_characters(id) ON DELETE CASCADE
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_ac_auction_transactions_idempotency
    ON ac_auction_transactions(auction_id, transaction_type, mutation_key)
    WHERE mutation_key <> '';

CREATE TABLE IF NOT EXISTS ac_vendor_transactions (
    transaction_id TEXT PRIMARY KEY,
    character_id TEXT NOT NULL,
    transaction_type TEXT NOT NULL,
    item_id TEXT NOT NULL,
    stack_count INTEGER NOT NULL,
    unit_price_copper INTEGER NOT NULL,
    total_copper INTEGER NOT NULL,
    mutation_key TEXT NOT NULL DEFAULT '',
    source_id TEXT NOT NULL DEFAULT '',
    created_at INTEGER NOT NULL,
    FOREIGN KEY (character_id) REFERENCES ac_characters(id) ON DELETE CASCADE
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_ac_vendor_transactions_idempotency
    ON ac_vendor_transactions(character_id, transaction_type, mutation_key)
    WHERE mutation_key <> '';

CREATE TABLE IF NOT EXISTS ac_economy_mutations (
    mutation_id TEXT PRIMARY KEY,
    actor_character_id TEXT NOT NULL,
    operation TEXT NOT NULL,
    mutation_key TEXT NOT NULL,
    response_json TEXT NOT NULL,
    created_at INTEGER NOT NULL,
    UNIQUE (actor_character_id, operation, mutation_key),
    FOREIGN KEY (actor_character_id) REFERENCES ac_characters(id) ON DELETE CASCADE
);
