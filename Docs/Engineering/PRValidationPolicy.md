# PR Validation Policy

## Purpose

Every AmandaCore PR should be small enough to review and validated against the systems it touches. The default target branch is `develop`; `main` is stable release code.

## Required Baseline

- Worktree clean before final validation.
- No unrelated user changes overwritten.
- `git diff --check`.
- Relevant Go tests for backend/service changes.
- `Infra/dev/build-local.ps1` for service, launcher, or local tooling changes.
- `Infra/qa/Scan-ForbiddenArtifacts.ps1` before staging and before push.

## Conditional Gates

- Contract docs/routes changed: run contract tests.
- Content schemas/compiler/runtime changed: run content compiler tests and `go run ./cmd/content-compiler --package ..\Content\Packs\dev_foundation\package.json --check`.
- O3DE/client/world payload changed: run O3DE build/verify/start-local smoke.
- Package/release scripts changed: run package smoke when practical and document anything skipped.
- Load, reconnect, or world-loop behavior changed: run a short loadsim/reconnect scenario.

## Forbidden Staging

Do not stage secrets, local machine files, logs, screenshots, diagnostics, zips, caches, build output, temp DB files, runtime tickets, process manifests, or generated credentials.

## PR Report

Include branch, base, files changed summary, tests run, validation result, O3DE/package validation status, clean-room confirmation, intentionally omitted scope, and recommended next step.
