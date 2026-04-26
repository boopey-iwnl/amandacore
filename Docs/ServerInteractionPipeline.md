# Authoritative Server Interaction Pipeline

This milestone adds AmandaCore's original server-side interaction pipeline for simulated clients and future real clients. It proves that world input can pass through a session gateway, canonical command queue, shard-owned runtime, deterministic tick, authoritative movement validation, state diff output, persistence handoff, and metrics/load simulation without relying on the O3DE client.

## Session Gateway

`Services/internal/worlds/SessionGateway` owns live world-session state after a single-use join ticket is consumed by the existing world service.

It tracks:

- session id
- account id
- character id
- realm id
- zone id
- authoritative position
- lifecycle state

Supported lifecycle states are attached, active, disconnect requested, disconnected, reconnect pending, reconnected, and expired. Commands are rejected unless the session is attached, active or reconnected, and bound to the same character as the command.

Duplicate active sessions for a character are deterministic: a new attach replaces the old active session and marks the old session disconnected with a replacement reason.

## Canonical Command Queue

`Services/internal/simcore` defines AmandaCore-native command envelopes and command payloads:

- move intent
- stop move intent
- face direction intent
- interact intent
- use ability intent
- select target intent
- heartbeat intent
- disconnect intent
- reconnect intent
- admin operation

Each envelope carries a command id, session id, account id, character id, realm id, zone id, client sequence, server receive time, intended tick, enqueue tick, payload, and validation result.

Commands are ordered deterministically inside a tick by:

1. lower intended tick
2. lower server receive time
3. lower session id
4. lower character id
5. lower command id

The queue is bounded. Full queues reject commands with an explicit queue-full reason and update queue/backpressure metrics.

## Simulation Tick

`WorldRuntime` is a fixed-step server-authoritative runtime. The default tick interval is 50 ms. A tick:

1. drains queued commands
2. sorts them deterministically
3. validates each command against the session gateway
4. applies accepted commands to authoritative runtime entity state
5. emits domain events and state diffs
6. marks dirty character state for persistence
7. updates runtime metrics

Tick processing is intentionally protocol-agnostic. Network adapters should translate client input into AmandaCore commands before reaching this runtime.

## Authoritative Movement

Movement commands are server-authoritative. The runtime validates movement against server-owned rules:

- finite numeric deltas only
- max movement distance per command
- bounded X/Y world area
- server-controlled Z

Valid movement emits a position delta. Excessive movement is corrected, not trusted, and emits a correction delta with an authoritative position and reason code. Invalid numeric movement is rejected.

The existing launcher/world flow remains compatible. The HTTP world move endpoint now routes normal outdoor movement through the canonical queue/tick path and flushes the resulting dirty character position through the persistence handoff.

## State Diffs

`simcore.StateDiff` is the internal future-client output model. Current deltas include:

- entity state delta
- position delta
- spawn delta
- despawn delta
- correction delta
- event delta

The runtime currently emits state diffs for movement and event output. Future client networking can serialize these AmandaCore-native diffs without adopting external MMO packet terminology.

## Persistence Handoff

`PersistenceHandoff` records dirty character positions separately from tick processing. The runtime marks dirty state; the world service flushes it on movement and disconnect boundaries to preserve the existing reconnect behavior.

The handoff wraps the current file store behind `CharacterStateWriter`, leaving room for a future database-backed writer.

Persistence events include:

- `persistence.flush.requested`
- `persistence.flush.completed`
- `persistence.flush.failed`
- `persistence.snapshot.saved`

## Observability

Stable event names are centralized in `Services/internal/observability/events.go`.

New interaction events include:

- `world.session.attached`
- `world.session.detached`
- `world.session.replaced`
- `world.session.reconnect_requested`
- `world.session.reconnect_completed`
- `world.session.expired`
- `world.session.command_rejected`
- `world.command.enqueued`
- `world.command.rejected`
- `world.queue.backpressure`
- `world.tick.started`
- `world.tick.completed`
- `world.tick.slow`
- `world.tick.command_batch_processed`
- `world.movement.accepted`
- `world.movement.corrected`
- `world.movement.rejected`
- `world.state.diff_emitted`
- `world.entity.state_dirty`
- `world.entity.state_flushed`
- `loadsim.started`
- `loadsim.completed`

`GET /v1/world/metrics` now includes interaction runtime metrics and persistence handoff stats alongside existing endpoint, session, tick, and persistence metrics.

## Load Simulator

The in-process load simulator runs without O3DE or a live service stack:

```powershell
powershell -ExecutionPolicy Bypass -File .\Infra\dev\run-world-loadsim.ps1 -Clients 100 -DurationSeconds 60 -CommandsPerSecond 5
```

Equivalent Go command:

```powershell
Set-Location .\Services
go run ./cmd/loadsim --clients 100 --duration 60s --cmd-rate 5
```

The report includes:

- total sessions attached
- total commands sent
- accepted and rejected commands
- average, max, and approximate p95 tick duration
- max command queue depth
- reconnect attempts and successes
- persistence flush count
- errors

## Tests

The pipeline is covered by fast Go tests for:

- unauthenticated/unbound session rejection
- deterministic duplicate-session handling
- deterministic command ordering
- bounded queue rejection
- movement tick processing
- movement correction
- state diff output
- dirty position persistence handoff
- disconnect/reconnect position restoration
- tiny load simulation scenario

Run:

```powershell
Set-Location .\Services
go test ./...
```

## Relationship to TrinityCore/AzerothCore Reference Study

AmandaCore adapts general MMO architecture principles only: session separation, authoritative world ownership, deterministic ticks, command queues, observability, persistence discipline, and load testing.

AmandaCore does not copy source code, SQL, packet layouts, opcodes, GM commands, schemas, assets, content IDs, scripting APIs, comments, constants, or surface vocabulary from TrinityCore, AzerothCore, MaNGOS, WoW private-server projects, or proprietary game data.

This implementation is AmandaCore-original and protocol-agnostic. External client protocols must remain adapters around AmandaCore commands, events, and state diffs.
