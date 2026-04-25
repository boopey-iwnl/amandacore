# Multi-Zone Sharding Skeleton

AmandaCore's first sharding layer is an in-process runtime ownership skeleton. It proves routing, zone isolation, capacity hints, queue backpressure, and load snapshots without introducing cross-process RPC.

## Ownership Model

The skeleton defines:

| Concept | Role |
| --- | --- |
| `ShardID` | Stable local shard identifier. |
| `ShardRole` | Runtime role: `LocalWorld`, `ZoneOwner`, `InstanceOwner`, or `GatewayPlaceholder`. |
| `ShardRuntime` | In-process owner for one or more zones. |
| `ShardRegistry` | Registered local shard runtimes. |
| `ZoneShardBinding` | One zone to one shard owner. |
| `ShardRouter` | Resolves character commands to the owning zone runtime. |
| `ZoneRuntime` | Per-zone command queue, session set, tick samples, and queue metrics. |

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

## Observability Events

Stable event names added for this milestone include:

- `loadsim.run_started`
- `loadsim.run_completed`
- `loadsim.run_failed`
- `loadsim.scenario_started`
- `loadsim.scenario_completed`
- `loadsim.report_written`
- `loadsim.zone_distribution_applied`
- `loadsim.client_spawned`
- `loadsim.command_sent`
- `loadsim.command_rejected`
- `loadsim.transition_requested`
- `loadsim.transition_completed`
- `loadsim.transition_rejected`
- `loadsim.reconnect_attempted`
- `loadsim.reconnect_completed`
- `shard.registry.created`
- `shard.registered`
- `shard.zone.assigned`
- `shard.zone.unassigned`
- `shard.assignment.rejected`
- `shard.route.resolved`
- `shard.route.rejected`
- `shard.load_snapshot.recorded`
- `shard.capacity.warning`
- `shard.backpressure.detected`
- `world.queue.backpressure`
- `world.command.rejected`
- `world.command.wrong_zone_rejected`
- `world.zone.isolation_violation_detected`

High-volume events are reserved for verbose runs where practical.

## Clean-Room Reference Boundary

This implementation uses original AmandaCore code and tooling. TrinityCore/AzerothCore were used only as high-level architectural reference. No source code, SQL, packet layouts, opcodes, command names, schemas, content IDs, scripts, scripting APIs, assets, formulas, map formats, area tables, zone tables, spawn schemas, coordinates, worldserver configs, database structures, command systems, remote admin systems, playerbot systems, performance scripts, or benchmark tools were copied or adapted.

## Known Limitations

- No networked cross-process shard RPC exists yet.
- No live zone migration or load-based rebalance exists yet.
- Entity counts are local skeleton counters, not full production ECS counts.
- The route model is intentionally conservative and rejects cross-zone gameplay interactions.

## Next Milestone

Production persistence and database migration layer should come next, with clean-room schemas, migrations, transactional boundaries, recovery tests, and persistence latency/failure observability.
