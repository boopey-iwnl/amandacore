# Alpha Release Checklist

## Source Control

- Release source is on `main` only after approved merge from `develop`.
- `develop` release-candidate validation passed with `Infra/qa/Validate-ReleaseCandidate.ps1`.
- Tag creation is explicitly approved.
- Release tag points to the source commit used to build the artifact.
- No feature branch is deleted before proving it is merged and explicitly approving deletion.

## Build and Package

- Local validation passes.
- O3DE client build and verification pass when applicable.
- Forbidden artifact scanner passes.
- Release package is built from the intended source commit.
- Package manifest records source commit, source branch, build label, version, channel, build time, release notes path, and asset digest.
- `Infra/qa/Assert-ReleasePackage.ps1` passes on the archive or clean extraction.
- Package smoke passes on a clean extracted package.

## Artifact Contents

- Required launcher/local ops/client/runtime/content/assets are present.
- Required high-res icons and O3DE level assets are present.
- No `.git`, `.secrets`, logs, diagnostics, screenshots, runtime tickets, local DBs, zips, cache/build output, temp files, or machine-local paths are present.

## Human Gameplay

- Launcher opens.
- Login works.
- Realm list works.
- Character create/select works.
- Join world works.
- Visible world loads.
- WASD movement and camera work.
- Reconnect restores state.
- Quest giver/progress/reward work.
- Trainer works.
- Inventory and action bars work.
- Combat, loot, and reward flow work.
- Social/economy smoke passes if exposed in the build.
- Game log is clean enough for alpha release.

## Publish

- Create a draft prerelease first when practical.
- Download the draft artifact before publishing.
- Extract into a clean location.
- Repeat package smoke and a short gameplay test.
- Record package hash in release notes.
- Publish only after final human approval.

## Cutover

- Local/dev file-store mode still works.
- Staging/production file-store mode is rejected unless explicitly overridden.
- SQLite migration status/check/apply commands pass on a test database.
- Legacy state dry-run inventory has been reviewed if carrying forward local dev state.
- Rollback path is documented for the candidate.
