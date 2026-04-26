# AmandaCore Persistence

AmandaCore now has a production persistence foundation around the existing durable store. The current local implementation remains file-backed for developer velocity, but the domain boundary is repository-oriented so a future Postgres implementation can replace the adapter without changing gameplay handlers.

## Persistence Domains

| Domain | Current owner | Notes |
| --- | --- | --- |
| Identity and sessions | `FileStore` account/session methods | Accounts, password hashes, access/refresh sessions, bans, suspensions, and admin seed state. |
| Character profile | `CharacterRepository` | Race, class, archetype, level, currency, current zone, and position. |
| Progression | `ProgressionRepository` | Learned abilities, action bars, quests, tracked quest IDs, talents, professions, and account progression. |
| Inventory and equipment | `InventoryRepository` | Inventory slots, equipment slots, vendor/economy updates, and housing storage handoff. |
| Session recovery | `SessionRecoveryRepository` | Reconnect-safe zone, position, profile, and last-seen state. |
| Migrations | `MigrationRepository` | Immutable migration IDs, checksums, dry-run validation, and applied history. |
| Unit of work | `UnitOfWork` | Transaction helpers for multi-field character updates and future multi-aggregate operations. |

## Transaction Boundaries

`FileStore.WithTransaction` provides a single locked mutation boundary with rollback on error. `UpdateCharacterAtomically` is the character-focused helper for gameplay flows that need coupled state changes, such as quest reward updates that touch experience, currency, inventory, action bars, or quest progress together.

Recommended atomic operations:

- character creation plus starter inventory/action bars
- quest completion plus reward grants
- loot claim plus inventory update
- reconnect recovery state updates
- zone transfer plus position save
- equipment mutation plus inventory slot mutation

## Dirty-State Flush Policy

`DirtyStateBuffer` tracks the latest pending character position/state update per character and flushes it through the repository interface. This keeps hot movement updates from forcing every tick to know storage details.

Policy fields:

- `FlushInterval`: intended cadence for a future scheduled flusher.
- `MaxPending`: maximum tracked dirty characters before the oldest pending entry is evicted.

Flush results report attempted, flushed, failed, and still-pending counts. Failed flushes leave entries pending for later retry.

## Recovery

`LoadSessionRecoveryState` returns reconnect state from durable character data:

- character/account/realm IDs
- display name
- zone and position
- level, experience, currency
- last seen timestamp

World reconnect can use this shape as a narrow persistence contract instead of depending on the full character aggregate.

## Observability

Stable persistence events added in this milestone:

- `persistence.migration_started`
- `persistence.migration_completed`
- `persistence.migration_failed`
- `persistence.transaction_started`
- `persistence.transaction_committed`
- `persistence.transaction_rolled_back`
- `persistence.flush_started`
- `persistence.flush_completed`
- `persistence.flush_failed`
- `persistence.recovery_started`
- `persistence.recovery_completed`
- `persistence.recovery_failed`

## Known Limitations

- The runtime adapter is still file-backed; Postgres is prepared by interface and migration discipline, not yet active as the service backend.
- Dirty-state buffering is implemented as a reusable store component but is not yet scheduled inside the world tick loop.
- Cross-aggregate transactions are currently modeled inside one durable store process. A future database adapter should map the same unit-of-work contract to real SQL transactions.

## Clean-Room Reference Boundary

This implementation uses original AmandaCore code, interfaces, and migration metadata. TrinityCore/AzerothCore were used only as high-level architectural reference. No source code, SQL, packet layouts, opcodes, command names, schemas, content IDs, scripts, scripting APIs, assets, formulas, map formats, area tables, zone tables, spawn schemas, coordinates, worldserver configs, database structures, command systems, remote admin systems, playerbot systems, performance scripts, or benchmark tools were copied or adapted.
