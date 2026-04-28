# Replication And Client Convergence

## Purpose

Milestone 6 formalizes AmandaCore's current Stonewake client-state synchronization without changing the active transport. HTTP polling remains the compatibility path, but every loop-backed world response now carries versioned replication metadata that can be used by current clients and future push transports.

## Current Polling/Bootstrap Behavior

The current client flow is still:

1. `POST /v1/world/connect`
2. repeated `POST /v1/world/move` and gameplay commands
3. repeated `GET /v1/world/state?worldSessionToken=<token>`
4. `POST /v1/world/reconnect`

Each response returns the rich world-session JSON payload consumed by the O3DE client and fallback .NET client. That payload includes position, vitals, target/combat state, inventory, action bars, spellbook, quest state, loot containers, visible entities, domain events, state diffs, map hints, and adjacent gameplay state.

## Current Client-Side Rehydration/Shim Behavior

The O3DE client still has defensive presentation rehydration for ability and action-bar UI fields. Milestone 6 does not remove those guards because they protect current local play, but the server now exposes complete replication metadata so stale or out-of-order state can be detected before the client applies it.

## Current Authoritative World-Loop Snapshot Behavior

Milestones 4 and 5 route the Stonewake hot path through a single-writer loop. The loop already mirrors player, NPC, combat, threat, quest, loot, inventory-adjacent, and action-bar state in a compact authoritative `WorldSnapshot`. Milestone 6 stamps that state with a monotonic state version whenever the authoritative snapshot changes.

## Proposed Snapshot Contract

The replication protocol is `amandacore.replication.v1`.

Full snapshots are emitted for connect, reconnect, missing cursors, and resync cases. A full snapshot includes:

- shard and zone identity
- monotonic state version and command cursor
- changed field summary
- compact authoritative Stonewake snapshot
- compatibility metadata copied to top-level response fields

Existing world-session response fields remain the current full-state payload for HTTP clients.

## Proposed Delta Contract

Deltas are generated from retained loop frames. A delta includes:

- prior cursor
- current cursor
- delta version
- changed domains and fields
- compact authoritative state payload for current polling compatibility

Current HTTP polling still receives the complete world-session response. The delta metadata allows clients and tests to prove convergence now, and gives a future push transport a stable handoff point.

## State Version/Cursor Model

The cursor token format is:

```text
shardId:zoneId:stateVersion:sequence:tick
```

`stateVersion` advances only when authoritative player, NPC, loot, combat, quest, inventory, action-bar, or world identity state changes. No-op polls can advance command sequence/tick internally without advancing state version.

## Client Acknowledgement Model

Clients acknowledge state by sending `since=<cursor>` on `GET /v1/world/state`. The server accepts omitted cursors as a full snapshot request. A retained cursor returns delta metadata. A stale or too-new cursor returns a full resync frame or rejects malformed input with `invalid_cursor`.

## Out-Of-Order/Stale Update Handling

The O3DE client stores the latest server replication version in `WorldSessionResponse`. If a non-resync, non-full-snapshot response arrives with an older version than the applied state, the client drops it and logs a warning instead of overwriting newer state.

## Transport-Neutral Design

The replication model is independent of HTTP. HTTP responses carry the metadata today. Later WebSocket, UDP, or binary gateway work can map the same frame and cursor model to push delivery without changing world-loop command ownership.

## HTTP Polling Compatibility

No route names changed. Existing clients can ignore the new fields:

- `snapshotVersion`
- `deltaVersion`
- `cursor`
- `fullSnapshot`
- `resyncRequired`
- `changed`
- `replication`

`GET /v1/world/state?worldSessionToken=<token>&since=<cursor>` is additive.

## Future Push-Replication Handoff

Future push replication should reuse the same cursor, changed-field, snapshot, and delta contracts. Push transport can send compact deltas directly, while HTTP polling can continue returning complete compatibility payloads with metadata.

## Convergence Test Strategy

Tests cover:

- snapshot and cursor emission on connect
- movement delta from an acknowledged cursor
- no-op poll preserving state version
- stale cursor full-resync behavior
- replay determinism for generated versions
- a small multi-client convergence soak through retained delta frames
- HTTP response metadata and invalid cursor handling

## Non-Goals

- no binary protocol
- no WebSocket or UDP transport cutover
- no polling removal
- no full SQL runtime cutover
- no file-store removal
- no full multi-zone replication model
- no social/economy replication expansion beyond existing world response compatibility

## Clean-Room Notes

The snapshot, delta, cursor, acknowledgement, changed-field, and convergence model is AmandaCore-original. It does not copy TrinityCore/AzerothCore packet layouts, opcode systems, update-field layouts, schemas, IDs, command names, comments, or module structure.

## Risks For Milestone 7

- Social and economy systems still use separate request/response state models.
- Future mail, auction, party, and guild transactionality will need their own versioning and acknowledgement rules.
- HTTP polling still returns full payloads, so delta-size metrics are useful but not yet representative of a push transport.

## Milestone 8 Content Boundary Note

The content compiler does not change the replication contract. Package version and catalog data remain server-side runtime inputs unless a future endpoint deliberately exposes them. If future clients need content manifest metadata, it should be added as optional replication or bootstrap metadata with stale-version handling and contract fixtures.
