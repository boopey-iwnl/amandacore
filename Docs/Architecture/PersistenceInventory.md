# AmandaCore Persistence Inventory

## Purpose

This inventory records the persistence behavior that exists before AmandaCore replaces the local file-backed development store with relational persistence. Milestone 2 uses this inventory to define repository boundaries, migration ownership, and cutover risks without changing the Alpha 0.15 service flow.

## Current `platform-state.json` State Shape

The local development store is `Services/internal/store.FileStore`, persisted as one JSON document. The default path is resolved by `AMANDACORE_STORE_PATH`; when unset, the service uses the platform user config directory under `amandacore/platform-state.json`.

The JSON state currently contains maps for:

- identity and access: `accounts`, `sessions`, `passwordReset`
- realm and build bootstrap: `realms`, `buildManifest`
- characters and gameplay state: `characters`, `accountProgress`
- world entry: `worldJoinTickets`
- social state: `friends`, `parties`, `guilds`, `guildInvites`
- economy and mail: `auctions`, `mail`
- moderation and operations: `auditEvents`, `supportTickets`, `mutes`
- housing: `housingEntitlements`, `housingSpaces`, `housingStorage`, `housingDecorations`
- file-store migration bookkeeping: `migrationHistory`

No real local state or secrets are documented here.

## Current Store Entry Points

The current service runtime opens one `FileStore` per process with `store.NewFileStore(cfg.StorePath, cfg.BuildID, cfg.WorldEndpoint)`. Startup applies file-store migrations and refreshes runtime build manifest fields before saving the state file.

Core entry points include:

- account registration, authentication, role, ban, suspension, and password reset methods
- session creation, refresh, validation, revocation, and account session revocation
- realm list and build manifest reads
- character list, create, lookup, state update, progression update, economy update, action-bar update, and inventory update methods
- world join ticket issue, validate, consume, and revoke methods
- social, guild, party, auction, mail, housing, support, mute, and audit methods
- `WithTransaction` and `UpdateCharacterAtomically` for locked JSON mutations with rollback on error
- `ApplyMigrations` and `MigrationHistory` for file-store migration metadata
- `LoadSessionRecoveryState` for reconnect-safe character state reads

## Services And Packages That Read Or Write Persistence

Command entry points currently construct the file store directly:

- `Services/cmd/account-service`
- `Services/cmd/admin-service`
- `Services/cmd/auth-service`
- `Services/cmd/character-service`
- `Services/cmd/realm-service`
- `Services/cmd/world-service`
- `Services/cmd/dbmigrate`
- `Services/cmd/loadsim`

HTTP packages receive `*store.FileStore` directly:

- `Services/internal/accounts`
- `Services/internal/admin`
- `Services/internal/authn`
- `Services/internal/characters`
- `Services/internal/httpapi`
- `Services/internal/realms`
- `Services/internal/worlds`

Gameplay and QA packages also depend on file-store behavior through e2e tests, load simulations, and world tests.

## Current Account And Session Persistence Behavior

Accounts are stored by account ID and include username, password hash, roles, ban/suspension state, login metadata, and timestamps. Password hashes are generated locally and are never documented with real values.

Sessions are stored by session ID with access and refresh tokens plus independent expiry timestamps. Validation checks token presence, expiry, and account moderation state. Refresh rotates tokens. Revocation removes matching sessions or account session sets.

## Current Character, Inventory, Quest, And Action-Bar Persistence Behavior

Characters are stored as full aggregates in the `characters` map. The aggregate currently contains identity, realm ownership, level, experience, currency, zone and position, inventory, equipment, professions, learned abilities, action-bar slots, talents, quests, kill credits, tracked quests, PvP stats, bind point, travel state, mount state, and last-seen timestamp.

Gameplay mutations usually update the complete character aggregate. Coupled updates use `UpdateCharacterProgression`, `UpdateCharacterEconomy`, `UpdateCharacterInventory`, `UpdateCharacterActionBarSlots`, or `UpdateCharacterAtomically`.

Milestone 2 adds repository views for inventory, quest progress, learned abilities, and action bars so later milestones can move callers away from full aggregate JSON mutation.

## Current World Join Ticket And Session Persistence Behavior

World join tickets are persisted in `worldJoinTickets`. They bind account, session, character, realm, endpoint, expiry, and consumed timestamp. The world service validates and consumes tickets before entering the current HTTP world flow.

World session recovery is currently derived from durable character state. There is no durable world-session table in the runtime path yet.

## Tests That Protect Current Persistence Behavior

Existing coverage includes:

- `Services/internal/store/file_store_test.go`
- `Services/internal/store/migrations_test.go`
- `Services/internal/store/transactions_test.go`
- `Services/internal/store/auction_store_test.go`
- `Services/internal/e2e/account_to_world_test.go`
- `Services/internal/e2e/admin_tools_test.go`
- `Services/internal/e2e/combat_slice_test.go`
- `Services/internal/e2e/quest_slice_test.go`
- `Services/internal/e2e/social_slice_test.go`
- `Services/internal/e2e/guild_slice_test.go`
- `Services/internal/e2e/housing_slice_test.go`
- world package tests that create temporary file stores for progression, reconnect, PvP, load simulation, and group content

Milestone 1 contract tests also protect the public HTTP and DTO inventory.

## Known Concurrency Risks

- The JSON store serializes mutations with a process-local mutex and file lock, but each service process opens its own store handle.
- Hot position and gameplay updates rewrite the same document-shaped state.
- Full aggregate writes can overwrite adjacent changes if future callers bypass locked transaction helpers.
- World ticket consumption is file-backed and not yet guarded by a relational uniqueness or compare-and-set operation.
- Social and economy paths still need stronger idempotency and ledger-style records in later milestones.

## Known Production Blockers

- The active runtime path is still a local JSON file.
- There is no production database adapter selected by configuration.
- SQL migrations do not yet drive service startup.
- Some domains are still represented as full JSON aggregates.
- Cross-service consistency depends on a shared local file path.
- Backups, corruption recovery, and rollback are local-dev oriented.

## Milestone 2 Scope

Milestone 2 adds:

- AmandaCore-owned repository interfaces
- a SQLite-compatible relational migration runner
- initial `ac_*` relational schema migrations
- SQL-backed repository skeletons for identity, sessions, realms, characters, gameplay subsets, world join tickets, and audit events
- deterministic test seed helpers
- migration and repository round-trip tests
- persistence redesign and local relational persistence documentation

## Non-Goals

- No service runtime is converted from file store to SQL in this milestone.
- No production database server is required.
- No Alpha 0.15 release behavior is changed.
- No world shard loop or single-writer authority implementation is added.
- No file-backed store is removed.
- No external MMO schema, ID set, packet layout, command vocabulary, or implementation detail is copied.

## Cutover Risks For Later Milestones

- Character aggregate fields need careful decomposition so reconnect restores the same state.
- Inventory, quest, learned ability, and action-bar updates need transaction-level protection before service callers switch.
- World ticket consume must become idempotent and retry-safe.
- Account/session migration needs explicit password hash and token handling rules.
- File-store historical data needs a tested one-way migration or documented reset path.
- Admin, audit, social, economy, housing, and support data require separate repository coverage before the file store can leave production paths.
