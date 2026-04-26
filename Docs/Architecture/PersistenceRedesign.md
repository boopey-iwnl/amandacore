# AmandaCore Persistence Redesign

## Why Relational Persistence Is Being Introduced

AmandaCore's local file-backed store is useful for development, but the next backend milestones need stronger storage guarantees: durable transactions, uniqueness constraints, replayable migration history, cleaner repository boundaries, and a path toward multi-process service reliability.

Milestone 2 introduces the relational foundation while preserving the current playable Alpha flow. The existing file store remains the default service path.

## What Milestone 2 Implements

Milestone 2 adds a testable SQLite-backed package at `Services/internal/store/sqlstore` with:

- embedded ordered SQL migrations
- checksum-backed migration metadata in `ac_schema_migrations`
- transaction-wrapped migration apply
- migration status and applied-record listing
- repository skeletons for accounts, sessions, realms, characters, inventory, quest progress, learned abilities, action bars, world join tickets, and audit events
- deterministic seed helpers for local/test realms and accounts
- round-trip tests for the implemented repositories

The Go service module now depends on `modernc.org/sqlite` so tests do not require a native SQLite driver or external database server.

## What Remains On File Store

The active service commands still instantiate `store.NewFileStore` through current configuration. These runtime flows remain file-backed:

- launcher login and account APIs
- realm listing and build manifest serving
- character create/select/state/progression APIs
- world join and current HTTP world actions
- admin, support, moderation, auction, mail, social, guild, party, housing, and load-simulation paths

The new SQL store is a foundation and test adapter until a later milestone explicitly converts service callers behind repository interfaces.

## Repository Boundary

`Services/internal/store/repositories.go` defines behavior-oriented interfaces for:

- accounts
- sessions
- realms
- characters
- character progression
- inventory
- quest progress
- learned abilities
- action bars
- session recovery
- world join tickets
- world sessions
- audit events
- migration history
- unit-of-work transactions

These interfaces are AmandaCore-owned boundaries. They intentionally describe current service behavior instead of copying another MMO server's modules, table layout, or packet model.

## Initial Schema Summary

The relational schema uses AmandaCore `ac_*` table names:

- identity: `ac_accounts`, `ac_account_credentials`, `ac_sessions`, `ac_password_resets`
- realm and character: `ac_realms`, `ac_characters`, `ac_character_stats`
- gameplay state: `ac_character_inventory`, `ac_character_quests`, `ac_learned_abilities`, `ac_action_bar_slots`, `ac_currency_ledger`
- world session state: `ac_world_join_tickets`, `ac_world_sessions`, `ac_character_position_snapshots`
- events and audit: `ac_domain_events`, `ac_audit_events`
- migration metadata: `ac_schema_migrations`

Some character subdocuments remain JSON columns temporarily where the service aggregate is still broader than the Milestone 2 cutover scope. Later milestones should normalize only when a caller and transaction boundary are ready.

## How Migrations Work

Migrations live under `Services/internal/store/sqlstore/migrations` and are embedded into the Go binary with `go:embed`.

The migrator:

- sorts migration files by filename
- derives the ID and label from the filename
- records a SHA-256 checksum of the SQL file contents
- applies each pending migration inside a transaction
- inserts a metadata row only after the SQL succeeds
- treats re-running applied migrations as a no-op
- fails clearly if an already-applied checksum changes

Migration files are immutable once merged. Add a new migration for schema evolution.

## SQLite Local And Test Guidance

The SQL store currently supports local/test SQLite files. Tests open temporary databases through `sqlstore.Open(path)`, apply all embedded migrations automatically, and clean up with the test temp directory.

No SQLite database files should be committed. Use temp directories, ignored local paths, or throwaway files when manually experimenting.

## Later Milestone 3 Cutover Plan

Milestone 3 should convert service callers in narrow slices:

1. Keep the current file store as the default fallback.
2. Add an explicit backend selector only when startup, validation, and rollback behavior are ready.
3. Move character create/select/update behind `CharacterRepository`.
4. Move inventory, quest, learned ability, and action-bar mutations behind dedicated repositories.
5. Add concurrency tests for item loss/duplication and action-bar/quest mutation retries.
6. Add a file-state import or reset runbook before any production cutover.

## Rollback And Corruption Considerations

For Milestone 2, rollback is simple: stop using the SQL test adapter and continue using the file-backed service path. No runtime service data is migrated automatically.

For later cutovers, rollback requires:

- database backup before migration
- migration status capture
- idempotent import jobs
- clear handling for partially imported characters
- validation that launcher/login/realm/character/world flows use one selected backend consistently

## Clean-Room Note

This design, schema, table naming, migration metadata, repository interface set, and tests are original AmandaCore work. Public MMO-server architecture informed only the behavioral goals of contract discipline, relational persistence, deterministic replay readiness, authority boundaries, and reliability engineering. No TrinityCore/AzerothCore code, SQL, table layout, packet definition, opcode, command vocabulary, script, content ID, or documentation text is copied or adapted.
