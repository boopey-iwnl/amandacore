# Release Gate Checklist

## Preflight

- `develop` is clean and contains the approved milestone work.
- `main` is untouched until release approval.
- No tags or GitHub releases are created before explicit release approval.
- Branch state and source commit are recorded.

## Automated Gates

- `git diff --check`
- `Push-Location Services; go test ./... -count=1 -timeout 15m; Pop-Location`
- `Infra/dev/build-local.ps1`
- `Infra/qa/Scan-ForbiddenArtifacts.ps1`
- O3DE build and verify scripts when client/runtime assets changed.
- Package smoke check against a clean extracted package when packaging changes or release candidates are involved.

## Package Smoke

The package must include required launcher, Local Ops, client/runtime, content, icons, version manifest, and support scripts. It must not include `.git`, `.secrets`, logs, diagnostics, runtime tickets, local DB files, nested archives, build caches, or local machine paths.

## Human Playtest

Before a public release, test from the downloaded/extracted package, not only from the local workspace. Verify login, realm list, character create/select, join world, visible world load, movement, reconnect, quest, trainer, inventory, action bar, combat, loot/reward, and high-res icons.

## Publish Gate

Only publish from an approved tag on `main`. The tag must point to the source commit used to build the package. Release notes, package name, package hash, and version manifest must agree.

## Rollback Notes

Keep the previous verified release artifact and hash available until the new public download is verified. If package smoke or human playtest fails, do not publish; fix on `develop` or a release branch and rebuild from a new source commit.
