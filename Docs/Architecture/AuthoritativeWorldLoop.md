# Authoritative Stonewake World Loop

## Purpose

Milestone 4 introduces an AmandaCore-owned single-writer execution context for the current playable Stonewake Vale slice. The goal is to serialize mutable world state behind the existing HTTP/JSON API surface so launcher, O3DE, and fallback client compatibility remain intact while the service gains deterministic command ordering, replay records, and clearer authority boundaries.

This is not a transport rewrite. HTTP polling remains the compatibility path for this milestone.

## Current World Mutation Paths

Before this milestone, most world HTTP handlers decoded a request, took `worldServer.mutex`, advanced world simulation, mutated session or NPC state directly, persisted selected character fields, and returned the existing world response shape. The main mutable paths were:

- session connect, reconnect, disconnect, and stale-session cleanup in `Services/internal/worlds/http.go` and `metrics.go`
- movement and position persistence in `/v1/world/move`
- target selection, auto-attack, ability activation, aura advancement, mob AI, and damage resolution in combat helpers
- quest accept, progress, completion, reward, tracking, and kill-credit helpers
- inventory movement/equip and action-bar assignment/move/clear helpers
- snapshot generation through `/v1/world/state` and `buildResponse`
- adjacent systems such as dungeons, housing, travel, social, guild, auction, vendor, gathering, crafting, and admin routes that also mutate state under `worldServer.mutex`

## Current Request/Response And Polling Behavior

The public contract remains request/response:

- `POST /v1/world/connect` consumes a join ticket and returns the bootstrap world state.
- `POST /v1/world/move` applies a movement delta and returns the full world state.
- `GET /v1/world/state` polls the current full world state.
- `POST /v1/world/reconnect` restores an in-process world session and returns the full world state.
- Combat, quest, inventory, and action-bar routes keep returning the same additive world response shape.

No route names, DTO names, or client-facing payload shapes were intentionally changed.

## Proposed Stonewake Single-Writer Model

`Services/internal/worlds/loop` owns the new shard-loop primitive:

- `ShardLoop` runs one worker goroutine for one shard.
- `CommandFunc` and typed command structs model AmandaCore world commands.
- `ShardState` owns a compact authoritative snapshot mirror for players and NPCs.
- external callers submit commands through a bounded queue.
- every accepted command is applied by the loop worker, producing a `CommandResult`.
- replay records are appended in applied command order.

`worldServer` starts a Stonewake loop for `stonewake_vale.primary` and routes the current Stonewake hot path through `submitStonewakeHTTPCommand` / `submitStonewakeSessionMutation`. The adapter preserves existing handler behavior by running existing domain helpers inside the loop command apply function, then syncing the resulting session/NPC state into the loop snapshot mirror.

## Command Ownership Rules

Loop-backed in Milestone 4:

- world session connect
- world session disconnect
- world session reconnect
- movement
- target selection and clear target
- auto-attack toggle
- ability activation
- quest accept, complete, and track
- inventory move and equip
- action-bar assign, move, and clear
- world state snapshot/poll

Temporarily outside the loop:

- join-ticket issuance and consumption remain persistence operations before session ownership enters the loop.
- admin routes remain direct operator actions.
- social, guild, party, auction, vendor, housing, travel, dungeon, gathering, crafting, profession, talent, and loot-specific routes still use existing locked helpers unless they enter through a loop-backed command listed above.
- multi-zone handoff coordinator behavior remains its existing queue/coordinator model.

Milestone 5 should move combat, loot, and quest reward expansion deeper into command-owned gameplay rules. Later milestones should absorb social/economy systems into their own transactional authority boundaries.

## Snapshot/Delta Strategy For Current Polling Clients

The loop emits an authoritative `WorldSnapshot` for its compact state mirror. Current clients still consume `buildResponse`, which is richer than the loop snapshot and includes inventory, quest, action bar, spellbook, social, map, streaming, housing, travel, PvP, domain-event, and state-diff fields.

For M4, `/v1/world/state` submits `RequestSnapshot` through the loop, advances the existing world simulation inside that command, syncs the compact loop snapshot, then returns the unchanged full HTTP response. This preserves polling clients while making snapshot generation loop-backed.

Push replication and binary deltas remain non-goals.

## Persistence Boundary

The loop owns the ordering of world mutations. Persistence still uses the current file-store runtime path by default:

- movement and disconnect persist character position through `UpdateCharacterState`
- action bars, inventory, quest progress, learned abilities, and rewards continue through existing store helpers
- SQL transactional repositories from Milestone 3 remain available for later cutover but are not made the runtime default here

This avoids mixing a world-authority change with a storage cutover.

## Replay Log Strategy

Every successfully applied loop command records:

- sequence
- logical tick
- command ID
- command kind
- session token
- actor ID
- recorded timestamp
- small command payload

The loop package includes an in-memory replay harness that can rebuild a compact `WorldSnapshot` from an initial snapshot plus replay records. M4 does not add an external replay file format or replay service endpoint.

## Compatibility With Current HTTP APIs

The adapter keeps the current HTTP routes and response shapes. Existing handlers still return the same status codes and error codes for the loop-backed paths, including missing world session, invalid movement, invalid target, action-bar validation, quest validation, and persistence failures.

Contract fixtures did not need changes because no public API shape changed.

## Non-Goals

- no push replication
- no WebSocket, UDP, or binary gateway
- no full continent or multi-zone world loop
- no full SQL runtime cutover
- no file-store removal
- no auction, guild, mail, party, or social transactionality
- no external MMO opcode, packet, schema, command, or module model

## Risks For Milestone 5 Combat/Loot/Quest Expansion

- Combat helpers still contain substantial existing logic and are invoked inside loop commands rather than being fully decomposed into command-native reducers.
- Mob AI and aura advancement still run when HTTP commands or polls advance the world; a future fixed tick worker may need to own those ticks.
- Loot, vendor, gathering, crafting, dungeon, and social/economy routes still mutate through existing locked helpers.
- The loop snapshot is intentionally compact while the HTTP response remains richer; future replication work must define an explicit delta contract.
- SQL transactional character-state methods are ready, but runtime world commands still write through the file-store default path.

## Clean-Room Note

The Stonewake loop, command names, replay records, state model, event names, documentation, and tests are AmandaCore-original. Public MMO-server architecture informed only the behavioral goal of single-writer world authority. No TrinityCore/AzerothCore code, SQL schema, packet layout, opcode table, command vocabulary, IDs, comments, or module structure is copied or adapted.
