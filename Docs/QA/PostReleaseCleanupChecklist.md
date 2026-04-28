# Post-Release Cleanup Checklist

## Release Traceability

- Published release points to a tag on `main`.
- Tag points to the source commit used to build the artifact.
- Release notes, package name, package manifest, and SHA agree.
- Downloaded public asset was tested after publication.

## Branch Audit

- `main` contains the release source.
- `develop` has been updated from `main` after release if needed.
- Temporary milestone/task branches are listed.
- Each candidate cleanup branch has no unique needed commits.
- Branch deletion is explicitly approved before deletion.

## Artifact Audit

- Previous verified package and hash are retained until the new public asset is verified.
- No local package zips, diagnostics, logs, screenshots, runtime tickets, DB files, or temp outputs are staged.
- Release package output folders are outside the repository or ignored.

## Follow-Up Issues

- Known release gaps are documented.
- Failed or skipped O3DE/package/soak checks have explicit follow-up.
- Any rollback or hotfix need is assigned before normal cleanup.

## Prohibited Cleanup

- Do not delete branches, tags, releases, release assets, packages, or diagnostics without explicit approval.
- Do not force-push or retag.
- Do not delete the only verified release package.
