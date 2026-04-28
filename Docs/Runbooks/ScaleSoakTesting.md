# Scale Soak Testing

## Purpose

Scale soak testing verifies AmandaCore release candidates under repeatable load without making heavy soak a normal validation cost.

## Small HTTP Scenario

Start the local stack first, then run:

```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File .\Infra\qa\Run-ScaleSoak.ps1 -Mode http -Users 2 -Duration 30s
```

The HTTP mode uses the load-test client to register, log in, list realms, create/select characters, issue join tickets, connect to the world, move, reconnect, combat where available, and collect latency/error summaries.

## Offline Runtime Scenario

Use this when services are not running but world-loop pressure needs a fast gate:

```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File .\Infra\qa\Run-ScaleSoak.ps1 -Mode runtime -Users 5 -Duration 30s
```

Runtime mode uses the in-process loadsim and reports command counts, accepted/rejected commands, reconnect attempts, queue depth, and tick metrics.

## Opt-In Soak Mode

Longer runs must be explicit:

```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File .\Infra\qa\Run-ScaleSoak.ps1 `
  -Mode http `
  -Users 25 `
  -Duration 30m `
  -MaxErrorRate 0.01 `
  -MaxP95Ms 500
```

Do not make long soak mandatory for normal PR validation.

## Metrics

The JSON summary includes:

- total requests or commands
- success/failure counts
- error rate
- p95 latency when available
- reconnect success rate
- duplicate mutation rejection count when available
- world-loop queue depth
- replication resync count when available
- persistence transaction failure count when available
- service health failure count

## Fail Thresholds

Default thresholds are intentionally conservative for alpha:

- error rate at or below 2 percent
- HTTP p95 latency at or below 750 ms
- no service health failures

Tighten thresholds only after stable baseline data exists.

## Output

Summaries are written under the system temp folder by default:

```text
%TEMP%\AmandaCore\scale-soak\scale-soak-summary.json
```

Do not commit soak output, event logs, diagnostics, or local state.

## Clean-Room Boundary

Scenarios and metrics are AmandaCore-owned and are not copied from external MMO emulator tooling.
