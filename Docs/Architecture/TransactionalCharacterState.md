# Transactional Character State

## Purpose

Milestone 3 defines and tests the persistence behavior required before AmandaCore moves mutable world state into an authoritative world loop. The focus is character-owned gameplay state: position, inventory, learned abilities, action bars, quests, currency, and reconnect restoration.

This milestone does not replace the current file-backed service path. It strengthens the SQL repository foundation and keeps Alpha 0.15 runtime behavior compatible.

## Current Character-State Mutation Paths

The active world service still receives a `*store.FileStore`. Current gameplay mutations write through file-store methods:

- `UpdateCharacterState` for zone and position.
- `UpdateCharacterInventory` for inventory rearrange.
- `UpdateCharacterActionBarSlots` for assign, clear, move, and swap.
- `UpdateCharacterProgression` for quest progress, reward items, currency, learned abilities, and action bars.
- `UpdateCharacterEconomy` for vendor/equipment currency and inventory changes.
- `LoadSessionRecoveryState` for reconnect-safe character summary.

World handlers keep session-local state in memory, persist through these methods, then copy normalized persisted state back into the live session.

## Character State Domains Covered

Milestone 3 covers:

- character records and optimistic state versions in SQL
- position snapshots
- inventory slots
- learned abilities
- action-bar slots
- quest progress and reward state
- character currency represented on the character row
- idempotency records for retryable character-state mutations
- reconnect restoration for position, inventory, learned abilities, action bars, quests, tracked quests, experience, and currency

## Transaction Boundary Decisions

Each SQL mutation loads one character aggregate and its normalized state collections inside a database transaction, applies the mutation in memory, writes the character row with an expected state version, rewrites the affected collection rows, records the optional idempotency response, and commits only after all steps succeed.

Rollback happens for validation errors, inventory capacity errors, duplicate mutation conflicts, SQL errors, and optimistic version conflicts.

The current file-store path already serializes writes with the file lock and `WithTransaction` / `UpdateCharacterAtomically`. Milestone 3 extends the file-store recovery payload but does not route services to SQL yet.

## Idempotency Strategy

SQL character-state mutations accept an optional `MutationOptions.MutationKey`. When provided, the store records a row in `ac_character_state_mutations` keyed by character, operation, and mutation key. Retrying the same operation with the same key returns the stored normalized character response instead of applying the mutation again.

This protects reward and grant flows from duplicate items, currency, abilities, or quest rewards under client retry.

## Concurrency And Locking Strategy

SQLite tests run with one open connection and transaction-wrapped writes. Character rows also carry `state_version`, and transactional SQL updates use an expected-version check. A stale writer fails with `ErrCharacterStateConflict` instead of silently overwriting state.

Collection rows keep per-row version/timestamp columns for later finer-grained mutation paths, but Milestone 3 still rewrites normalized character-owned collections as one aggregate.

## Reconnect Restoration Model

`SessionRecoveryState` now includes:

- character/account/realm identity
- display name
- zone and position
- level, experience, currency
- inventory slots
- learned ability IDs
- action-bar slots
- quest progress
- tracked quest IDs
- last-seen timestamp

Both file store and SQL store can provide this shape. The active world reconnect API remains unchanged because it still builds the existing client response from world session state.

## File-Store Compatibility

The file-backed dev path remains the default. Existing service/e2e flows continue to use file-store persistence. The only file-store behavior change in this milestone is a richer internal recovery state returned by `LoadSessionRecoveryState`.

## SQL-Store Behavior

The SQL store now implements transactional methods for:

- moving and swapping inventory slots
- granting inventory item stacks
- accepting and updating quest progress
- completing quests with item/currency/experience rewards
- granting learned abilities
- assigning, moving, swapping, and clearing action-bar slots
- recording and listing position snapshots
- restoring reconnect state

These methods are tested directly through `Services/internal/store/sqlstore`.

## Test Strategy

Milestone 3 tests cover:

- inventory move to empty slot
- inventory swap between occupied slots
- invalid inventory move rollback
- item reward rollback when inventory is full
- idempotent quest reward retry
- quest accept/progress/reward round trip
- action-bar assign, move, swap, and clear
- learned ability grant and duplicate handling
- position snapshot persistence
- reconnect recovery for gameplay state
- concurrent inventory grants without duplicated or lost state
- file-store recovery payload compatibility

Existing e2e tests continue to cover the active file-backed service path for account-to-world, reconnect, inventory, action bar, quest, combat, trainer, vendor, gathering, crafting, social, guild, housing, and travel flows. Milestone 5 adds loop-level replay and duplicate-mutation tests for combat, loot, quest rewards, item grants, and currency deltas while leaving the default runtime store path file-backed.

Milestone 7 adds separate SQL repository tests for social and economy transactionality. Those tests cover party/guild membership, chat, currency ledger, vendor, auction, and mail attachment claim behavior, but do not change the active file-backed runtime path.

## Non-Goals

- No full authoritative world shard loop.
- No SQL production cutover.
- No service-wide `AMANDACORE_STORE_BACKEND` selector.
- No runtime auction, guild, mail, or full economy cutover.
- No external MMO schemas, table layouts, packet layouts, opcodes, IDs, command names, or module structures.

## Risks For Milestone 4 World Loop

- The world loop still mutates session-local state before persistence.
- SQL methods are ready for authoritative command handlers, and current Stonewake gameplay HTTP actions now enter the single command queue before invoking runtime persistence.
- Quest and inventory semantics still depend on world package content definitions.
- Multi-character transactions, party-shared quest credit, and future party loot ownership still need explicit transactional cutover work.
- Runtime backend selection and import/rollback remain deferred.
