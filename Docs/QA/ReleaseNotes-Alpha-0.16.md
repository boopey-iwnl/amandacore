# AmandaCore Alpha 0.16

Release tag: `alpha-0.16`
Release title: AmandaCore Alpha 0.16
Source branch: `main`, merged from validated `develop`
Source commit: `alpha-0.16` tag target; exact commit is recorded in the package manifest and final release report
Develop source commit: `84e5180e27fed0a51f68cf8bb598fc584bceebe6`
Package asset: `AmandaCore-Alpha-0.16-Windows-x64.zip`
Package SHA256: `TO_BE_FILLED_FROM_FINAL_RELEASE_REPORT`
Package manifest: `release-package-manifest.json`

## Summary

Alpha 0.16 is a public prerelease for the current integrated AmandaCore playable foundation after Milestones 1 through 10. It consolidates the contract, persistence, world-loop, combat, replication, content, reliability, and release-gate work into `main` without claiming feature completeness.

This build is intended for local Windows x64 testing through the packaged Local Ops controls, launcher, services, and O3DE runtime.

## Major Changes Since Alpha 0.15

- Contract and branch policy foundation for reviewable milestone delivery.
- Relational SQLite migration and repository foundations while keeping runtime HTTP adapters file-store-bound for this release.
- Transactional character-state persistence boundaries and reconnect recovery coverage.
- Authoritative Stonewake world-loop command path for gameplay mutations.
- Combat, threat, loot, rewards, quest progression, action bars, and inventory-adjacent flow routed through the world loop.
- Replication and client convergence metadata for snapshot, delta, cursor, stale-client, and resync behavior.
- Transactional social and economy repository foundations for friends, party/guild membership, chat, currency, vendors, auctions, and mail.
- Content compiler/runtime boundary for AmandaCore-owned content package validation.
- Reliability, security, CI, forbidden artifact scanning, rate-limit, and package-smoke hardening.
- Production cutover, migration, scale-soak, package assertion, release-candidate, and post-release cleanup runbooks.

## Current Playable Status

- Local service stack starts through Local Ops or scripts.
- Launcher opens against the local stack.
- O3DE runtime verification confirms level load, world bootstrap/connect, and player spawn.
- The active runtime remains the file-backed local store.
- SQLite migration tooling can status/apply/check explicit test databases, but SQLite HTTP runtime cutover is intentionally refused for this release candidate path.
- Writable legacy import remains intentionally disabled.

## Setup And Run

1. Download `AmandaCore-Alpha-0.16-Windows-x64.zip` from the GitHub prerelease.
2. Extract it to a short writable path, for example `C:\AmandaCoreAlpha016`.
3. Open `Infra\dev\Launch-LocalOpsGui.cmd`.
4. Use Local Ops to start the local stack.
5. Wait for services to report healthy.
6. Launch AmandaCore Launcher from Local Ops.
7. Register or log in, load realms, create or select a character, and join the world.

If Local Ops is unavailable, start from PowerShell in the extracted package root:

```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File .\Infra\dev\start-local.ps1 -StartLauncher
```

Stop local services after testing:

```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File .\Infra\dev\stop-local.ps1
```

## Known Limitations

- Alpha 0.16 is not a complete MMO and is not production hosted.
- Gameplay coverage remains a local alpha slice, not a broad content release.
- SQLite runtime is migration-verified but not enabled for HTTP service adapters.
- File-store runtime remains the default local path.
- Writable legacy state import is disabled.
- Full downloaded-package human gameplay validation should still be repeated by a human tester after publication.
- Public release package validation does not imply hosted service readiness.

## Validation Performed

- `git diff --check`
- `go test ./... -count=1 -timeout 15m` from `Services`
- `Infra\qa\Scan-ForbiddenArtifacts.ps1`
- `Infra\dev\build-local.ps1`
- `Infra\dev\build-o3de-client.ps1`
- `Infra\dev\verify-o3de-client.ps1`
- `Infra\qa\Validate-ReleaseCandidate.ps1 -SkipO3DE`
- `Infra\qa\Smoke-Test.ps1 -SelfTest`
- `Infra\qa\Run-ScaleSoak.ps1 -Mode runtime -Users 2 -Duration 1s`
- Local startup smoke with `Infra\dev\start-local.ps1 -StartLauncher`
- Clean extracted package assertion and smoke are required before tag/release publication.
- Downloaded draft release asset assertion and smoke are required before publishing the prerelease.

## Clean-Room Note

AmandaCore Alpha 0.16 uses AmandaCore-original code, schemas, protocols, content IDs, scripts, assets, docs, runbooks, package validation, and release tooling. MMO emulator projects may inform high-level architecture only; no external MMO emulator source code, SQL schemas, packet layouts, opcodes, command names, content IDs, assets, quest text, UI skins, or protected trade dress are copied or adapted.
