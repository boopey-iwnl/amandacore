# Closed-Alpha Playtest Operations

This document is for the person running a small closed-alpha test pass.

## Before Sending a Build

1. Run the smoke test:

```powershell
powershell -NoLogo -NoProfile -ExecutionPolicy Bypass -File .\Infra\qa\Smoke-Test.ps1
```

2. Build or package the tester drop.
3. Confirm these files are included:

- `Docs\QA\TesterInstructions.md`
- `Docs\QA\checklists\closed-alpha-route.md`
- `Docs\QA\bug-report-template.md`
- `Docs\QA\KnownIssues.md`
- `Docs\QA\ReleaseNotes.md`
- `Docs\QA\TestFocus.md`
- `Infra\qa\Collect-Diagnostics.ps1`
- `Infra\qa\Smoke-Test.ps1`
- `Infra\qa\Seed-TestAccount.ps1`
- `Infra\qa\Reset-LocalTestState.ps1`

4. Update `KnownIssues.md`, `ReleaseNotes.md`, and `TestFocus.md` for the build.
5. Provide assigned test credentials through the approved tester channel.

## During the Test

- Keep test requests focused on the assigned route.
- Ask testers to file one bug per failure.
- Require a diagnostic bundle for every `FAIL` or `BLOCKED` checklist item.
- Treat duplicate issues as known issues only after a developer confirms the duplicate.
- Do not request new gameplay content during this milestone.

## Seed Test Account

With services running:

```powershell
powershell -NoLogo -NoProfile -ExecutionPolicy Bypass -File .\Infra\qa\Seed-TestAccount.ps1 -Username alpha_tester -Password "AlphaTest!123" -CharacterName Alphaone
```

The seed script uses local APIs. It does not write the password to disk.

## Reset Local Test State

Full reset:

```powershell
powershell -NoLogo -NoProfile -ExecutionPolicy Bypass -File .\Infra\qa\Reset-LocalTestState.ps1 -All -ConfirmReset
```

Selective reset:

```powershell
powershell -NoLogo -NoProfile -ExecutionPolicy Bypass -File .\Infra\qa\Reset-LocalTestState.ps1 -AccountUsername alpha_tester -ConfirmReset
```

Stop services first. Use `-Force` only when recovering a stuck local environment.

## End of Test Pass

Collect:

- completed checklist
- bug reports
- diagnostic bundles
- build ID
- tester notes
- known issue updates

Summarize:

- blockers
- high-confidence regressions
- repro steps that were confirmed twice
- route steps most testers could not complete
- reset or recovery instructions that failed
