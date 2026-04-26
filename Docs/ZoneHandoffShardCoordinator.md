# Zone Handoff And Shard Coordinator

## Scope

This milestone adds AmandaCore's first server-authoritative zone handoff and shard coordinator skeleton. It is production-shaped, but still deliberately in-process: one world server owns the coordinator, shard workers are state records, queues are in-memory counters, and the handoff journal is an in-memory runtime journal.

The implementation proves these contracts before distributed workers are introduced:

- deterministic zone-to-shard assignment
- character zone ownership
- handoff request, accept, reject, retry, and complete states
- per-zone command queue capacity and backpressure
- shard worker lifecycle state
- reconnect correction from coordinator ownership
- event and state-diff surfaces for future clients
- loadsim coverage for a rejected handoff followed by retry

## Clean-room reference boundary

This implementation uses original AmandaCore code and data.

TrinityCore/AzerothCore were used only as high-level architectural reference.

No source code, SQL, packet layouts, opcodes, command names, schemas, content IDs, scripts, scripting APIs, assets, formulas, loot tables, quest tables, item IDs, creature IDs, spell IDs, quest text, reward tables, map formats, zone tables, spawn schemas, coordinates, or database structures were copied or adapted.

## Runtime Components

The coordinator lives in `Services/internal/worlds/zone_handoff.go`.

Core concepts:

- `ShardID`: stable identifier for an in-process shard worker.
- `ZoneShardAssignment`: deterministic mapping from zone ID to shard ID.
- `ShardCoordinator`: owns assignments, worker states, queue counters, character ownership, handoff records, and the journal.
- `CharacterZoneOwnership`: current authoritative zone, shard, and position for a character.
- `ZoneCommandQueue`: per-zone queue capacity, current depth, and max observed depth.
- `ZoneHandoffGateDefinition`: server-side transition gate from one zone to another.
- `ZoneHandoffJournalEntry`: append-only runtime handoff state record.

The first gates connect Stonewake Vale and Brindlebrook Roadlands through AmandaCore-owned transition IDs. The Northspur route remains disabled as a future-zone rejection path.

## Handoff Flow

The current flow is:

1. Client or loadsim submits a transition intent.
2. World runtime validates session attachment, alive state, travel state, source zone, gate range, destination zone, destination position, destination shard state, and destination queue capacity.
3. Coordinator appends `requested` and `accepted` journal entries.
4. World runtime clears mount/combat state and moves the character to the destination arrival point.
5. Existing character position persistence is updated when a `FileStore` is attached.
6. Coordinator updates character ownership, releases queue depth, and appends `completed`.
7. World response includes `zoneHandoff` state for future UI/client routing.

Rejected handoffs append `requested` and `rejected` journal entries and leave the character in the source zone. Retryable rejects emit `zone.handoff.retry_scheduled`.

## Rejection Reasons

Stable rejection reasons include:

- `SessionInvalid`
- `CharacterDead`
- `InvalidState`
- `TransitionMissing`
- `TransitionDisabled`
- `WrongSourceZone`
- `OutOfRange`
- `DestinationZoneMissing`
- `DestinationEntryMissing`
- `SourceShardMissing`
- `DestinationShardMissing`
- `DestinationShardUnavailable`
- `DuplicatePendingHandoff`
- `QueueFull`
- `PersistenceFailed`

## Events And Diffs

New event names:

- `shard.worker.state_changed`
- `shard.coordinator.rejected`
- `zone.handoff.requested`
- `zone.handoff.accepted`
- `zone.handoff.completed`
- `zone.handoff.rejected`
- `zone.handoff.retry_scheduled`
- `zone.handoff.reconnect_corrected`
- `zone.queue.backpressure`
- `loadsim.zone_handoff.started`
- `loadsim.zone_handoff.completed`

The handoff path emits `ZoneHandoffDelta` state diffs so the client can later render zone transition progress, rejection reasons, retry hints, and updated ownership.

## HTTP Intent

The server exposes a canonical intent endpoint:

```text
POST /v1/world/zone/handoff
```

Request body:

```json
{
  "worldSessionToken": "world_...",
  "transitionId": "to_brindlebrook"
}
```

The response is the standard world response, including the updated `zoneId`, `position`, `zoneMap`, and `zoneHandoff` payload.

## Loadsim

Run from the repository root:

```powershell
go run ./Services/cmd/loadsim --clients 25 --duration 30s --cmd-rate 4 --scenario zone-handoff-basic --transition-loops 3 --shards 2 --queue-capacity 64
```

The scenario:

- creates temporary accounts and characters
- registers character ownership with the coordinator
- alternates Stonewake -> Brindlebrook and Brindlebrook -> Stonewake handoffs
- intentionally marks one destination shard unavailable for the first client
- verifies that rejection is retryable
- restores the shard and retries successfully
- reports handoff counts, journal entries, zone population, shard population, queue pressure, and errors

## Current Limitations

- Shard workers are in-process state records, not goroutines, processes, or remote services.
- The handoff journal is in-memory. Character zone and position persist through the existing `FileStore`, but journal durability is future work.
- Queue depth is an authoritative counter around handoff execution, not a full command scheduler yet.
- Transition gates are still authored in Go runtime definitions. Content-authored handoff gates should move into the future package loader.
- Reconnect correction uses coordinator ownership while the server process is alive. Full restart recovery needs durable ownership/journal storage.
- Cross-realm, cross-process, and cross-region handoffs are intentionally out of scope.

## Next Milestone

The next recommended milestone is zone content package loader integration for handoff data:

- AmandaCore-owned content manifest format for zones and transition gates
- content-authored handoff gate definitions
- validation for destination zones, arrival points, and disabled future routes
- shard assignment policy loaded from environment or content metadata
- durable handoff journal interface backed by the production persistence layer
- loadsim coverage for content-authored transitions
