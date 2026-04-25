# Milestone 15 Multi-Client Stability

Milestone 15 adds a local, repeatable foundation for multi-client stability, request timing, session lifecycle diagnostics, and short soak tests. It does not add gameplay content or production scaling infrastructure.

## Local Load Test Command

Start the local stack first:

```powershell
powershell -ExecutionPolicy Bypass -File "Infra/dev/start-local.ps1"
```

Run a quick two-client mixed scenario:

```powershell
powershell -ExecutionPolicy Bypass -File "Infra/dev/run-load-test.ps1" -Clients 2 -Scenario mixed -DurationMinutes 5
```

Run target tiers:

```powershell
powershell -ExecutionPolicy Bypass -File "Infra/dev/run-load-test.ps1" -Clients 2 -Scenario mixed -DurationMinutes 5
powershell -ExecutionPolicy Bypass -File "Infra/dev/run-load-test.ps1" -Clients 5 -Scenario mixed -DurationMinutes 5
powershell -ExecutionPolicy Bypass -File "Infra/dev/run-load-test.ps1" -Clients 10 -Scenario mixed -DurationMinutes 5
powershell -ExecutionPolicy Bypass -File "Infra/dev/run-load-test.ps1" -Clients 25 -Scenario mixed -DurationMinutes 5
```

Treat 25 clients as opt-in until 2, 5, and 10 client runs are stable.

## Supported Scenarios

- `idle`: connected clients poll state periodically.
- `move`: clients submit repeated movement requests.
- `combat`: clients move near hostile mobs, target, and enable auto-attack.
- `reconnect`: clients cycle disconnect/reconnect while polling state.
- `mixed`: assigns clients across move, combat, idle, and reconnect behavior.

## Output

Load-test output is written under:

```text
Infra/dev/load-tests/loadtest-<timestamp>-<scenario>-clients<N>/
```

Each run writes:

- `events.jsonl`: per-request timing, status, and error events.
- `summary.json`: machine-readable aggregate report.
- `summary.md`: human-readable latency/error summary.

The summary includes build manifest data, scenario, client count, duration, request counts, error counts, latency percentiles, desync count, and a final `/v1/world/metrics` snapshot.

## World Metrics Endpoint

The world service exposes a lightweight local endpoint:

```text
GET http://localhost:8085/v1/world/metrics
```

It reports:

- active, connected, and disconnected world sessions
- endpoint counts, status counts, average latency, and max latency
- world tick timing
- persistence write timing
- stale sessions dropped
- goroutine count
- Go memory snapshot

The server also emits periodic `world.metrics_snapshot` structured log events during request-driven world advancement.

## Manual Validation Checklist

1. Start the stack.
2. Launch two real clients with different accounts and characters.
3. Confirm both clients join the same zone.
4. Confirm both characters appear in the other session's world state, and in-client if remote player presentation is available.
5. Move both clients at the same time.
6. Enter combat from both clients.
7. Disconnect and reconnect one client while the other remains active.
8. Run a two-client simulated mixed soak.
9. Inspect `summary.md`, `summary.json`, `events.jsonl`, and `Infra/dev/logs/world-service.log`.
10. Confirm the world service did not crash and solo login/join/move still works.

## Baseline Targets

- Tier 1: 2 clients, required for every local validation.
- Tier 2: 5 clients, expected to be stable for short runs.
- Tier 3: 10 simulated clients, first meaningful baseline for current architecture.
- Tier 4: 25 simulated clients, run only after lower tiers pass.
- Later: 50+ clients after persistence and world-lock bottlenecks are measured and addressed.

Do not claim production capacity from these numbers. They are local architecture baselines only.
