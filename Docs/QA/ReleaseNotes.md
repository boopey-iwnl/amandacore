# Alpha 0.1 Release Candidate Notes

Build ID: generated per package in `Infra\dev\version-manifest.json`
Package manifest: `release-package-manifest.json`

## Test Scope

This build is for Alpha 0.1 release-candidate validation of the existing playable slice. It does not add gameplay content and should be treated as feature frozen.

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

## Out of Scope

- New races, classes, zones, raids, PvP modes, guild features, auction house, mail, housing, achievements, travel, or mounts
- Public launch operations
- External bug tracking integration
- Production telemetry
- Cloud operations or load testing

## Disabled Or Hidden

- PvP duels are disabled in local Alpha 0.1 startup.
- Guild, mail, trade, auction, housing/storage, achievements/titles/collections, travel, and mounts are not part of the main tester flow.
- Admin/support tools are for build owners only and must stay admin-gated.

## Reporting

For every `FAIL` or `BLOCKED` checklist item, attach a diagnostic bundle and complete `bug-report-template.md`.

## Local QA Tools

- `Infra\qa\Collect-Diagnostics.ps1` creates a redacted diagnostic bundle.
- `Infra\qa\Smoke-Test.ps1` validates docs, manifests, QA scripts, and optional service health.
- `Infra\qa\Seed-TestAccount.ps1` creates or reuses a local tester account and Human Warrior through local APIs.
- `Infra\qa\Reset-LocalTestState.ps1` backs up and resets local state while preserving logs.
- `Infra\dev\package-alpha.ps1` creates an allowlisted Alpha 0.1 package and checksum.
- `Infra\dev\Launch-LocalOpsGui.cmd` exposes diagnostics, QA docs, and guarded state reset actions.
