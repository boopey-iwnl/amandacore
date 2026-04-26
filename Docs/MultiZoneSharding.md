# Multi-Zone Sharding Skeleton

AmandaCore's first sharding layer is an in-process runtime ownership skeleton. It proves routing, zone isolation, capacity hints, queue backpressure, deterministic zone-to-shard assignment, and load snapshots without introducing cross-process RPC.

## Ownership Model

| Concept | Role |
| --- | --- |
| `ShardID` | Stable local shard identifier. |
| `ShardRole` | Runtime role: `LocalWorld`, `ZoneOwner`, `InstanceOwner`, or `GatewayPlaceholder`. |
| `ShardRuntime` | In-process owner for one or more zones. |
| `ShardRegistry` | Registered local shard runtimes. |
| `ZoneShardBinding` | One zone to one shard owner. |
| `ShardRouter` | Resolves character commands to the owning zone runtime. |
| `ZoneRuntime` | Per-zone command queue, session set, tick samples, and queue metrics. |
| `ZoneShardAssignment` | Deterministic Dawnwake loadsim zone-to-shard ownership record. |

A `ZoneRuntime` is owned by exactly one shard runtime. Multiple zones can be assigned to one shard.

## Assignment Policies

| Policy | Behavior |
| --- | --- |
| `static` | Assigns zones round-robin by package order. |
| `least-loaded` | Assigns to the shard with the fewest active zone bindings. |
| `hash-zone` | Deterministically hashes zone ID to shard index. |

Capacity hints are local guardrails:

- `max_zones`
- `max_sessions`
- `max_entities`
- `max_commands_per_tick`

The current skeleton records and reports capacity pressure but does not rebalance live zones.

## Dawnwake Loadsim Scenario

Run from the Go module root:

```powershell
cd Services
go run ./cmd/loadsim --clients 25 --duration 30s --cmd-rate 4 --scenario dawnwake-multizone-sharding-basic --transition-loops 3 --shards 2 --seed 42 --content ..\Content\Packs\dawnwake_isles\package.json
```

The scenario:

- loads and validates `dawnwake_isles`
- activates package zones into the world runtime
- builds deterministic shard assignments
- starts simulated players at the continent default entry
- probes enabled transition gates repeatedly
- rejects missing destination zones, missing destination entry points, disabled gates, and missing shard assignments
- reports per-zone population, per-shard population, transition counts, queue pressure, and tick timing percentiles

## Isolation Rules

Commands route through `ShardRouter.Submit`. The router rejects commands when:

- the character has no owning zone: `EntityNotInZone`
- the submitted zone does not match the character owner zone: `CharacterZoneMismatch`
- the zone is not assigned to a shard: `ShardNotOwner`
- a command is submitted directly to the wrong `ZoneRuntime`: `WrongZone`
- combat or loot targets another zone: `CrossZoneInteractionUnsupported`
- a transition is not represented by a valid zone gate: `TransitionRequired`
- the zone queue is full: `QueueFull`

Combat and loot cannot occur across zones in this milestone. Quest progress can remain global to a character, but triggering events in this skeleton must originate from the owning zone runtime. Visibility is scoped to current zone ownership; future boundary streaming hints can be layered on top of this model.

## Report Fields

The Dawnwake loadsim report includes:

- package and continent identifiers
- validation errors
- activated zones and catalogs
- transition gates loaded
- players attached
- transition loops and deterministic seed
- shard count and zone shard assignments
- final zone population and final shard population
- transition counts by transition and source zone
- requested, completed, and rejected transition totals
- visibility and streaming hint counters
- spawned NPC and quest provider counts
- average, p50, p95, p99, and max tick duration
- max simulated queue depth
- rejected command count
- errors

## Backpressure

Each zone has a configurable queue capacity. When a queue is full, `ZoneRuntime.Enqueue` rejects commands with `QueueFull`, increments queue metrics, emits backpressure events, and keeps the simulation alive.

Loadsim reports:

- accepted commands
- rejected commands
- rejection reasons
- max queue depth
- average queue depth
- per-zone queue metrics
- per-shard load snapshots

## Clean-Room Reference Boundary

This implementation uses original AmandaCore code and data. TrinityCore/AzerothCore were used only as high-level architectural reference. No source code, SQL, packet layouts, opcodes, command names, schemas, content IDs, scripts, scripting APIs, assets, formulas, loot tables, quest tables, item IDs, creature IDs, spell IDs, quest text, reward tables, map formats, zone tables, spawn schemas, coordinates, worldserver configs, database structures, command systems, remote admin systems, playerbot systems, performance scripts, or benchmark tools were copied or adapted.

## Known Limitations

- No networked cross-process shard RPC exists yet.
- No live zone migration or load-based rebalance exists yet.
- Entity counts are local skeleton counters, not full production ECS counts.
- Transition probes validate package topology and destination ownership, but do not perform production player handoff.
- Queue depth is a simulated command pressure value, not a live gateway queue.

## Next Milestone

The next recommended milestone is production zone handoff and shard coordinator design, followed by production persistence and database migration hardening.
