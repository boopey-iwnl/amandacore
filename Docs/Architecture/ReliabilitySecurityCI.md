# Reliability, Security, and CI Hardening

## Purpose

Milestone 9 hardens AmandaCore before production cutover work. The goal is to catch regressions, unsafe configuration, malformed inputs, duplicate mutations, forbidden files, package mistakes, and runtime instability through automated checks and documented gates.

## Current CI/Test Surface

The repository now defines GitHub Actions gates for Go service tests, contract/doc consistency checks, content compiler validation, .NET client builds, forbidden artifact scans, and package-smoke readiness. Full O3DE verification remains a local release gate because the hosted runner does not include the AmandaCore O3DE workspace/toolchain.

## Current Local Validation Scripts

Local validation still uses `Infra/dev/build-local.ps1`, `Infra/dev/build-o3de-client.ps1`, `Infra/dev/verify-o3de-client.ps1`, `Infra/dev/start-local.ps1`, and loadsim scripts. Milestone 9 adds `Infra/qa/Scan-ForbiddenArtifacts.ps1` for tracked-file scans before staging, CI, packaging, and release work.

## Current Packaging/Release Validation

`Infra/dev/package-alpha.ps1` and `Infra/qa/Smoke-Test.ps1` remain the release package path. Package gates must verify required launcher, Local Ops, client, runtime, content, icon, and manifest assets while excluding secrets, logs, local state, caches, diagnostics, nested archives, and local machine paths.

Milestone 10 adds `Infra/qa/Assert-ReleasePackage.ps1` and `Infra/qa/Validate-ReleaseCandidate.ps1` so package assertions and release-candidate validation can run before any tag or public release is created.

## Current Config and Secret Handling

Service startup now has explicit validation through `config.LoadValidated`. Local development defaults remain accepted. Production mode rejects local admin tools, local `.secrets` seed reliance, weak admin seed passwords, unsupported store backends, invalid service ports, and malformed world endpoints.

## Current Auth/Session/Join-Ticket Safety

Auth endpoints now use bounded JSON parsing and local in-memory rate limiting for register, login, and password recovery attempts. Session and join-ticket semantics remain unchanged; later distributed deployment will need shared rate limiting.

## Current API Defensive Handling

`httpapi.DecodeJSON` enforces an optional `application/json` content type, bounded request body size, non-empty request bodies, and single JSON values. Existing route names and response shapes remain compatible.

## Current Observability/Audit Events

Milestone 9 adds stable security/reliability event names for login failures, rate limiting, session rejection, admin unauthorized access, config rejection, rejected HTTP requests, persistence transaction failures, idempotent retry detection, and package smoke failure. Existing structured JSON logging remains the runtime convention.

## Current Load/Soak Coverage

Existing loadsim and loadtest-client flows cover movement, reconnect pressure, combat, abilities, quests, multizone traversal, and mixed client behavior. M9 documents short local reliability smoke usage and keeps longer soak runs opt-in.

Milestone 10 wraps these paths with `Infra/qa/Run-ScaleSoak.ps1`, which keeps small defaults fast and makes longer HTTP/runtime soak explicit.

## Known Reliability Gaps

Full O3DE CI is not yet practical on hosted runners. Package smoke is manual/dispatch unless explicitly enabled. Rate limiting is process-local and not shared across multiple service instances. Runtime config validation is intentionally conservative and does not convert SQL/content/social systems to production defaults.

## Known Security Gaps

The scanner is a sanity gate, not a full secret-detection product. It prioritizes high-confidence token/private-key patterns and repository artifact mistakes. Production deployments still need managed secrets, centralized logging controls, TLS termination policy, and shared abuse controls.

## Milestone 9 Scope

This milestone adds CI workflow foundations, forbidden artifact scanning, config validation, HTTP decode hardening, auth rate limiting, fuzz/property tests, observability event names, contract/doc consistency checks, and release/PR runbooks.

## Non-Goals

No new gameplay systems, no binary protocol, no push replication, no SQL production cutover, no file-store removal, no release publishing, no tag changes, and no full production infrastructure provisioning.

## Clean-Room Notes

All CI workflows, scripts, validation rules, event names, and runbooks are AmandaCore-original operational designs. No external MMO emulator code, schemas, scripts, packet layouts, command vocabularies, or operational module structures are copied.

## Risks for Milestone 10

Milestone 10 must validate the final production defaults, package cutover, rollback path, release artifact traceability, and multi-hour soak behavior. M9 gates reduce risk but do not replace manual gameplay and package verification.
