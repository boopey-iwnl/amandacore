# Reliability and Security Local Checks

## Purpose

Run these checks before PR review, merge approval, or any release-candidate packaging work that touches services, clients, contracts, content, or infrastructure scripts.

## Standard PR Checks

From the repository root:

```powershell
git status --short
git diff --check
Push-Location Services
go test ./... -count=1 -timeout 15m
Pop-Location
powershell -NoProfile -ExecutionPolicy Bypass -File .\Infra\dev\build-local.ps1
powershell -NoProfile -ExecutionPolicy Bypass -File .\Infra\qa\Scan-ForbiddenArtifacts.ps1
```

## Client/O3DE Gates

Run these when world bootstrap payloads, O3DE-facing APIs, client parsing, package scripts, O3DE assets, or launcher/client code changed:

```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File .\Infra\dev\build-o3de-client.ps1
powershell -NoProfile -ExecutionPolicy Bypass -File .\Infra\dev\verify-o3de-client.ps1
powershell -NoProfile -ExecutionPolicy Bypass -File .\Infra\dev\start-local.ps1 -StartLauncher
```

If startup smoke passes but manual gameplay was not performed, say so explicitly.

## Forbidden Files Policy

Do not stage `.secrets`, tokens, local DB files, runtime tickets, logs, screenshots, diagnostics, zips, cache/build output, temp files, local process manifests, or machine-local paths. Run the scanner before staging and again before pushing.

## Short Reliability Smoke

The bundled short reliability smoke runs the forbidden artifact scanner and a reconnect-pressure world loadsim:

```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File .\Infra\dev\run-reliability-smoke.ps1 -Clients 2 -Duration 30s
```

After starting the local stack, use a short mixed scenario:

```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File .\Infra\dev\run-load-test.ps1 -Clients 2 -Scenario mixed -DurationMinutes 1 -StepInterval 250ms
```

For world-loop pressure without the launcher stack:

```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File .\Infra\dev\run-world-loadsim.ps1 -Scenario reconnect-pressure -Clients 5 -Duration 10s -CommandRate 2
```

## Failure Handling

Stop on dirty worktrees, failed validation, unexpected generated artifacts, forbidden scanner findings, rejected pushes, or non-fast-forward branch updates. Report the exact command, failure output summary, and changed files before continuing.
