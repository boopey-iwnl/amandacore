# Closed-Alpha Release Notes

Build ID: amandacore-local-0.2.0

## Test Scope

This build is for closed-alpha readiness of the existing playable slice. It does not add gameplay content.

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

## Out of Scope

- New races, classes, zones, dungeons, raids, PvP, guilds, auction house, or mail
- Public launch operations
- External bug tracking integration
- Production telemetry
- Cloud operations or load testing

## Reporting

For every `FAIL` or `BLOCKED` checklist item, attach a diagnostic bundle and complete `bug-report-template.md`.

## Local QA Tools

- `Infra\qa\Collect-Diagnostics.ps1` creates a redacted diagnostic bundle.
- `Infra\qa\Smoke-Test.ps1` validates docs, manifests, QA scripts, and optional service health.
- `Infra\qa\Seed-TestAccount.ps1` creates or reuses a local tester account and Human Warrior through local APIs.
- `Infra\qa\Reset-LocalTestState.ps1` backs up and resets local state while preserving logs.
- `Infra\dev\Launch-LocalOpsGui.cmd` exposes diagnostics, QA docs, and guarded state reset actions.
