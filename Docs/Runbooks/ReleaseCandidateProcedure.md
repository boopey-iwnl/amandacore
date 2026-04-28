# Release Candidate Procedure

## Pre-Release Branch State

- Normal work starts from `develop`.
- Release stabilization may use `release/<version>` only after approval.
- Do not work directly on `main`.
- Do not tag, publish, delete branches, force-push, or rewrite history without explicit approval.

## Develop Validation

From the repository root:

```powershell
git status --short --branch
git diff --check
Push-Location Services; go test ./... -count=1 -timeout 15m; Pop-Location
powershell -NoProfile -ExecutionPolicy Bypass -File .\Infra\qa\Scan-ForbiddenArtifacts.ps1
powershell -NoProfile -ExecutionPolicy Bypass -File .\Infra\dev\build-local.ps1
powershell -NoProfile -ExecutionPolicy Bypass -File .\Infra\qa\Validate-ReleaseCandidate.ps1 -SkipO3DE
```

Run O3DE validation when client/runtime assets changed:

```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File .\Infra\dev\build-o3de-client.ps1
powershell -NoProfile -ExecutionPolicy Bypass -File .\Infra\dev\verify-o3de-client.ps1
```

## Merge Develop To Main

Only after approval:

1. Ensure `develop` validation passed.
2. Open a PR from `develop` or the approved release branch to `main`.
3. Require review and required checks.
4. Merge without rewriting history.
5. Confirm `main` points to the intended source commit.

## Package Build Procedure

Build a local non-public release candidate:

```powershell
$label = "alpha-0.16-rc1"
powershell -NoProfile -ExecutionPolicy Bypass -File .\Infra\dev\package-alpha.ps1 `
  -Channel $label `
  -BuildLabel $label `
  -ReleaseNotesPath .\Docs\QA\ReleaseNotes.md
```

The package manifest must include source branch, full commit SHA, build label, timestamp, release notes path, version manifest, runtime summary, and asset digest.

## Package Extraction Test

Run package assertion and smoke against a clean extracted archive:

```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File .\Infra\qa\Assert-ReleasePackage.ps1 -ArchivePath <package.zip> -ReleaseNotesPath .\Docs\QA\ReleaseNotes.md -RunSmoke
```

## Draft GitHub Prerelease Procedure

After explicit approval, create a draft prerelease from a tag on `main`. Upload the package and hash. Do not publish yet.

## Download-Draft-Asset Test

Download the draft asset through GitHub, extract it into a clean location, and rerun package assertion/smoke. Manual gameplay validation must use this downloaded package, not the local build folder.

## Final Human Test Gate

Verify launcher, login, realm list, character create/select, join world, visible world load, movement, reconnect, quest giver/progress/reward, trainer, inventory, action bars, combat, loot, high-res icons, and Game.log.

## Publish Command

Publishing requires explicit human approval after downloaded-asset validation. Do not publish from a local-only package.

## Post-Release Branch Cleanup Audit

After release, list temporary branches and compare them to `main` and `develop`. Delete only after proving useful work is merged and branch deletion is explicitly approved.

## Rollback/Withdraw Procedure

If the draft fails, delete no tags and publish nothing. If a published prerelease must be withdrawn, record the reason, preserve the failed asset for traceability if safe, and publish a corrected build from a new source commit.

## No-Force-Push/No-Retag Policy

Force-push, retag, or history rewrite requires explicit approval naming the exact branch/tag and reason. Default action is a new commit, new package, and new tag.
