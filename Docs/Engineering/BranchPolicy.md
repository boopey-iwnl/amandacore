# AmandaCore Branch Policy

AmandaCore uses two permanent branches and short-lived task, hotfix, and release branches.

Clean-room architecture references are documented in [Docs/Engineering/CleanRoomStudy-TrinityCore-AzerothCore.md](CleanRoomStudy-TrinityCore-AzerothCore.md).

## Permanent Branches

### main

Purpose:

- Stable release branch.
- Latest published or release-candidate-safe code.
- Source branch for release tags.
- Source branch for GitHub releases.

Rules:

- Do not do normal development directly on `main`.
- Do not let Codex agents commit directly to `main` for normal work.
- Merge to `main` only after validation and approval.
- Create release tags from `main`.
- `main` should always build.

### develop

Purpose:

- Active integration branch for the next release.
- Normal target for completed Codex work.
- Can be ahead of `main`.
- Should remain buildable enough for continuous testing.

Rules:

- Merge normal feature and fix branches to `develop` first.
- Sync `develop` from `main` after every release or hotfix.
- Do not leave `develop` broken.
- Cut release branches from `develop`.

No other permanent branches are allowed unless explicitly approved.

## Temporary Branches

Allowed temporary branch patterns:

- `codex/<short-task-name>` for normal Codex tasks.
- `codex/m<number>-<short-task-name>` for milestone-scoped Codex tasks.
- `hotfix/<short-task-name>` for urgent fixes against published `main`.
- `release/<version>` for release-candidate stabilization.

Examples:

- `codex/fix-chat-scrollbars`
- `codex/m27-zone-polish`
- `codex/package-level-load-fix`
- `hotfix/launcher-startup`
- `release/alpha-0.2`

Avoid vague branch names such as:

- `fix`
- `update`
- `test`
- `temp`
- `backup`
- `final`
- `final2`
- `latest`
- `new-work`

Do not use branches as archives. Use tags and GitHub releases for release history, and commits or pull requests for traceability.

## Normal Development Flow

1. Start from `develop`.
2. Create `codex/<task>` or `codex/m<number>-<task>`.
3. Implement and validate the change.
4. Push the task branch.
5. Merge to `develop` after approval.
6. Delete the task branch only after approval.

## Release Flow

1. Cut `release/<version>` from `develop`.
2. Fix release blockers only.
3. Validate and package.
4. Merge the release branch to `main` after approval.
5. Tag `main`.
6. Create the GitHub release from the tag.
7. Merge `main` back into `develop`.
8. Delete the release branch only after approval.

## Hotfix Flow

1. Cut `hotfix/<task>` from `main`.
2. Fix the urgent issue.
3. Validate the fix.
4. Merge to `main` after approval.
5. Tag and release if needed.
6. Merge `main` back into `develop`.
7. Delete the hotfix branch only after approval.

## Cleanup Rules

Temporary branches may be deleted only after all of the following are true:

- Useful work is merged.
- Validation passes.
- The branch has no unique needed commits.
- GitHub `main` or `develop` contains the work.
- Branch deletion is explicitly approved.

Do not delete branches as part of release publication unless branch deletion was separately approved.

## Recommended Required Checks

Before merge to `develop`:

- Go service tests.
- Contract/content compiler tests.
- Forbidden artifact scan.
- Local build check where practical.
- Focused package/assertion checks when packaging scripts changed.

Before merge to `main`:

- All `develop` checks.
- `Infra/qa/Validate-ReleaseCandidate.ps1 -SkipO3DE`.
- O3DE build/verify when client/runtime assets changed.
- Package assertion and smoke from a clean extracted candidate.
- Human gameplay checklist approval for release branches.

## Review Policy

Require at least one human review for release branches, hotfixes, branch-policy changes, release scripts, migration tooling, and security-sensitive changes. Codex-created milestone branches should remain draft until validation output and known gaps are documented.

## Branch Protection Recommendations

Protect `main` with required status checks, required review, no force-push, and no direct pushes except explicitly approved maintainers. Protect `develop` with required CI and no force-push. Do not change repository branch protection settings from an automation task unless separately approved.

## Future-Work Branches

Keep future-work branches only when they contain unmerged, intentionally deferred work with a named owner and documented next step. Do not keep branches as archives, backups, or release records.
