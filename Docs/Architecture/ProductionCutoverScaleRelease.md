# Production Cutover, Scale Soak, and Release Discipline

## Purpose

Milestone 10 makes the Milestone 1 through Milestone 9 architecture usable as a release-candidate path without adding large gameplay features. It defines the cutover guardrails, migration checks, legacy state dry run, scale soak discipline, package assertions, and release-candidate gate needed before the next hardened alpha.

## Current Runtime Architecture

Local services expose separate auth, account, realm, character, world, and admin HTTP processes. Gameplay remains driven by AmandaCore-owned service contracts, the authoritative Stonewake world loop, replication/convergence DTOs, content package loading, and file-backed platform state for the default local/dev path.

## Current File-Store Path

The default backend is `AMANDACORE_STORE_BACKEND=file`. `AMANDACORE_STORE_PATH` controls the platform-state JSON path, and local defaults use the user config directory unless scripts override it. File-store migrations apply on open and keep Alpha gameplay flows working.

## Current Relational Persistence Path

SQLite migrations and repository tests live under `Services/internal/store/sqlstore`. `AMANDACORE_STORE_BACKEND=sqlite` and `AMANDACORE_SQLITE_PATH=<path>` are now recognized by config and migration tooling. Service startup verifies SQLite migration state before refusing runtime use, because HTTP service adapters still depend on `FileStore` in this release candidate.

## Current World-Loop, Replication, and Content-Runtime Status

The Stonewake loop remains authoritative for movement, combat, quests, loot, inventory mutation, action bars, reconnect, social/economy foundations, and runtime metrics. Content compiler/runtime boundaries remain data-driven through AmandaCore content packages and schemas.

## Current Package/Release Workflow

`Infra/dev/package-alpha.ps1` creates local release-candidate packages only. The package manifest records source branch, full source commit, build label, timestamp, version manifest data, runtime path summary, release notes path, and content asset digest. `Infra/qa/Assert-ReleasePackage.ps1` and `Infra/qa/Smoke-Test.ps1` validate clean extracted packages.

## Cutover Target

The cutover target is a release candidate that can prove:

- local/dev file-store gameplay still works
- production/staging configs cannot accidentally use file-backed storage
- SQLite migration status/apply/check commands work
- legacy file-state can be inventoried before manual import
- package artifacts are traceable to source and release notes
- scale soak can be run with explicit thresholds

## Rollback Strategy

Until writable SQLite import is implemented, rollback is restore backup plus redeploy the previous verified build. For package releases, retain the previous verified artifact and SHA until the downloaded new artifact passes smoke and human gameplay checks. Do not retag or overwrite release history without explicit approval.

## Soak/Scale Criteria

Short release soak defaults are intentionally small. HTTP soak covers register, login, realm list, character create/select, join ticket, connect, movement, reconnect, combat, and state fetch through `Infra/qa/Run-ScaleSoak.ps1 -Mode http`. Offline runtime soak covers multizone, reconnect, queue depth, and command rejection metrics with `-Mode runtime`. Heavy duration/user counts are opt-in.

## Release Discipline

Release candidates are validated from `develop` with `Infra/qa/Validate-ReleaseCandidate.ps1 -SkipO3DE` plus optional O3DE, package smoke, and soak flags. Publishing still requires an approved merge to `main`, an approved tag on `main`, a draft prerelease, a downloaded asset test, and final human approval.

## Non-Goals

This milestone does not publish a release, create tags, delete branches, rewrite history, add gameplay features, enable writable SQLite service runtime, or replace manual gameplay testing.

## Clean-Room Note

All cutover procedures, migration checks, load scripts, package rules, and runbooks are AmandaCore-original. Do not copy external MMO emulator code, schemas, packet layouts, command names, operational scripts, IDs, comments, or content.

## Known Risks

The relational store is not yet the live HTTP service backend. Full import remains report-only. O3DE validation remains local because hosted runners do not provide the full workspace. HTTP scale soak requires local services to be running.
