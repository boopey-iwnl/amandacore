# AmandaCore Load Testing

AmandaCore load testing is a local developer simulation harness. These tiers are measurement targets for repeatable local runs, not production capacity claims.

## Scale-Test Tiers

| Tier | Simulated clients | Purpose |
| --- | ---: | --- |
| smoke | 1-5 | Fast validation for content loading, routing, and report generation. |
| small | 25 | Developer-machine sanity check for queue depth and tick health. |
| local-medium | 100 | Local pressure run for multi-zone routing and transition churn. |
| local-large | 250 | Best-effort local run; results depend on machine resources. |
| soak-placeholder | future | Long-duration scenario placeholder, not implemented as a production claim. |

The default loadsim configuration is intentionally safe for a normal developer machine: 5 clients, 10 seconds, 2 commands per client per second, and a 50 ms simulation tick.

## CLI

Run from `Services`:

```powershell
go run ./cmd/loadsim --clients 5 --duration 10s --cmd-rate 2 --scenario multizone-pressure --content ../Content/Packs/dawnwake_isles/package.json --seed 42
```

Supported flags:

| Flag | Purpose |
| --- | --- |
| `--scenario` | Scenario name. |
| `--clients` | Simulated client count. |
| `--duration` | Run duration such as `10s` or `2m`. |
| `--cmd-rate` | Commands per simulated client per second. |
| `--content` | Content package manifest path. |
| `--continent` | Continent identifier label for reports. |
| `--zone-distribution` | `even`, `transition-heavy`, `single:<zone>`, or `weighted:<zone>=<weight>,...`. |
| `--transition-rate` | Probability that an eligible command attempts a zone transition. |
| `--combat-rate` | Probability that a command becomes a combat pressure command. |
| `--ability-rate` | Reserved for ability pressure. |
| `--quest-rate` | Reserved for quest pressure. |
| `--reconnect-rate` | Probability for reconnect churn checks. |
| `--reconnect-interval` | Minimum reconnect interval per client. |
| `--seed` | Deterministic random seed. The seed is always printed in the report. |
| `--report` | Optional JSON report path. |
| `--tick-ms` | Simulation tick duration in milliseconds. |
| `--queue-capacity` | Per-zone command queue capacity. |
| `--verbose` | Emit high-volume per-client/per-command observability events. |
| `--shards` | Number of local shard runtimes. |
| `--assignment-policy` | `static`, `least-loaded`, or `hash-zone`. |

## Scenarios

| Scenario | Behavior |
| --- | --- |
| `movement-basic` | Sends movement commands through the same in-process router and zone queue. |
| `combat-basic` | Sends local combat commands and rejects cross-zone combat when generated. |
| `ability-basic` | Reserved for ability pressure; currently follows movement pressure. |
| `quest-basic` | Reserved for quest pressure; currently follows movement pressure. |
| `dawnwake-traversal-basic` | Raises transition likelihood across Dawnwake Isles gates. |
| `multizone-pressure` | Distributes clients across zones, sends movement and transition commands, and measures queues/ticks. |
| `shard-assignment-basic` | Creates multiple local shards and reports zone-to-shard ownership. |
| `reconnect-pressure` | Simulates reconnect attempts and verifies zone ownership remains available. |

## Reports

Every run prints a concise text report. When `--report` is provided, loadsim also writes JSON with:

- run ID, scenario, seed, content package, duration, client count, command rate
- activated zones, shard count, clients per zone, zones per shard
- commands sent, accepted, rejected, and rejection reasons
- transition requests, completions, rejections, and route counts
- reconnect attempts, successes, and failures
- tick count, average, max, p50, p95, and p99 duration
- max and average queue depth
- per-zone and per-shard load snapshots
- errors

Tick duration measures in-process simulation work for this harness, not real production server latency. Queue depth shows local backpressure pressure points; sustained nonzero rejected commands means the configured queue capacity or command rate exceeded the local runtime's ability to accept work.

## Clean-Room Reference Boundary

This implementation uses original AmandaCore code and tooling. TrinityCore/AzerothCore were used only as high-level architectural reference. No source code, SQL, packet layouts, opcodes, command names, schemas, content IDs, scripts, scripting APIs, assets, formulas, map formats, area tables, zone tables, spawn schemas, coordinates, worldserver configs, database structures, command systems, remote admin systems, playerbot systems, performance scripts, or benchmark tools were copied or adapted.

## Known Limitations

- This is an in-process local/dev harness, not a networked shard cluster.
- Combat, ability, and quest scenarios are pressure placeholders unless the active runtime exposes those systems through the loadsim command model.
- Tick timings can be near zero for tiny smoke runs because the local simulation is intentionally lightweight.
- Reports are generated artifacts and should not be committed unless intentionally used as small fixtures.

## Next Milestone

The next recommended milestone is production persistence and database migration layer: database-backed character/world/progression persistence, migration runner, schema versioning, repository interfaces, transactional boundaries, dirty-state flush policies, recovery/reconnect persistence tests, local dev database setup, seed data management, persistence observability, and clean-room schema documentation.
