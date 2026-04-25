# AmandaCore Alpha 0.1 Release Notes

Release tag: `alpha-0.1`
Package asset: `AmandaCore-Alpha-0.1-Windows-x64.zip`
Package SHA256: verify against the GitHub release asset digest and the final release report.
Package size: verify against the GitHub release asset metadata.
Source commit: `ab1308811c5e1c7479019748a2a3ac593b42cb8a`
Build label: recorded in packaged `Infra\dev\version-manifest.json`
Package manifest: `release-package-manifest.json`

## Scope

Alpha 0.1 is a closed test build for the current playable slice. It covers local launcher startup, account and realm flow, Human Warrior character creation, Stonewake Vale world entry, the assigned quest route, combat, hostile AI, trainer/vendor/inventory/equipment flow, map/minimap support, persistence, diagnostics, and recovery tooling.

This build is feature frozen for the Alpha 0.1 test pass. Testers should use the assigned route and file only issues that block or degrade the existing playable slice, packaging, diagnostics, local reset, or tester instructions.

## Extract And Run

1. Download `AmandaCore-Alpha-0.1-Windows-x64.zip` from the GitHub release.
2. Extract it to a short writable path, for example `C:\AmandaCoreAlpha01`.
3. Open `Infra\dev\Launch-LocalOpsGui.cmd`.
4. Click `Start Services`.
5. Wait until all services show healthy.
6. Click `Open Launcher`.
7. Register or log in, load realms, create or select a Human Warrior, and join the world.
8. Follow `Docs\QA\checklists\closed-alpha-route.md`.

If Local Ops is unavailable, start from an elevated PowerShell in the extracted package root:

```powershell
powershell -NoLogo -NoProfile -ExecutionPolicy Bypass -Command "& '.\Infra\dev\start-local.ps1' -BuildFirst:`$false -StartLauncher"
```

Stop local services after testing:

```powershell
powershell -NoLogo -NoProfile -ExecutionPolicy Bypass -File .\Infra\dev\stop-local.ps1
```

## Included Test Areas

- Launcher login and realm flow
- Human Warrior character creation and selection
- Stonewake Vale quest route
- Combat and hostile AI
- Trainer, vendor, inventory, currency, and equipment flows
- Map/minimap route support
- Local persistence and restart recovery
- Diagnostic bundle collection
- Structured bug report and checklist templates
- Optional rough validation for professions, second-zone handoff, chat/friends/party, and dungeon entry only when assigned

## Rough Or Deferred Systems

- Professions, second-zone handoff, chat/friends/party, and dungeon entry are rough optional coverage only when assigned.
- Guild management, mail, direct trade, auction house, housing/storage, achievements/titles/collections, travel, mounts, and PvP duels are hidden, disabled, or deferred.
- Admin/support tools are build-owner tools and must stay admin-gated.
- Cloud operations, public launch operations, production telemetry, load testing, external bug tracking, monetization, and anti-cheat are out of scope.

## Known Issues

No approved duplicate known issues are listed for this build. Use `Docs\QA\KnownIssues.md` for the current known-issue table and scope notes.

## Diagnostics And Reports

For every `FAIL` or `BLOCKED` checklist item:

1. Click `Collect Diagnostics` in Local Ops, or run:

   ```powershell
   powershell -NoLogo -NoProfile -ExecutionPolicy Bypass -File .\Infra\qa\Collect-Diagnostics.ps1
   ```

2. Attach the diagnostic zip from `%LOCALAPPDATA%\amandacore\diagnostics`.
3. Complete `Docs\QA\bug-report-template.md`.
4. Include the completed route checklist.

Do not include real passwords, private keys, personal documents, or unrelated screenshots.

## Local QA Tools

- `Infra\qa\Collect-Diagnostics.ps1` creates a redacted diagnostic bundle.
- `Infra\qa\Smoke-Test.ps1` validates docs, manifests, QA scripts, and optional service health.
- `Infra\qa\Seed-TestAccount.ps1` creates or reuses a local tester account and Human Warrior through local APIs.
- `Infra\qa\Reset-LocalTestState.ps1` backs up and resets local state while preserving logs.
- `Infra\dev\package-alpha.ps1` creates an allowlisted Alpha 0.1 package and checksum.
- `Infra\dev\Launch-LocalOpsGui.cmd` exposes diagnostics, QA docs, and guarded state reset actions.
