# Social Economy Local Test Runbook

## Status

Milestone 7 social and economy persistence is SQL-backed and test-only. Current AmandaCore local services still use the file-backed store by default.

## Run Focused Tests

From the repository root:

```powershell
Push-Location Services
go test ./internal/store/sqlstore -run "TestSocial|TestConcurrentParty|TestEconomy|TestConcurrentAuction" -count=1
Pop-Location
```

Run all SQL-store persistence tests:

```powershell
Push-Location Services
go test ./internal/store/sqlstore -count=1
Pop-Location
```

Run the full Go backend suite:

```powershell
Push-Location Services
go test ./... -count=1 -timeout 15m
Pop-Location
```

## What The Tests Cover

Social tests cover:

- friend add, duplicate rejection, list, and remove
- ignore add, list, and remove
- chat append and recent-message listing by scope
- party invite accept and idempotent retry
- guild create, invite accept, and idempotent retry
- concurrent party and guild accepts without duplicated membership

Economy tests cover:

- currency ledger idempotency
- vendor buy idempotency
- vendor buy rollback when inventory is full
- auction listing item/deposit mutation
- auction buyout item/currency settlement
- duplicate buyout rejection
- cancel-after-buyout rejection
- mail create/list/claim behavior
- duplicate mail claim prevention
- concurrent auction buyout and mail claim apply-once behavior

## Manual SQLite Experiments

Use throwaway database files only:

```powershell
$db = Join-Path $env:TEMP "amandacore-m7-social-economy.sqlite"
Remove-Item $db -ErrorAction SilentlyContinue
Push-Location Services
go test ./internal/store/sqlstore -run TestEconomyRepositoryTransactionsAndIdempotency -count=1
Pop-Location
```

Do not commit `.sqlite`, `.db`, WAL, SHM, logs, diagnostics, screenshots, or local state.

## Transaction Expectations

Every retry-sensitive SQL mutation should either:

- commit all related rows once
- replay the prior response for the same mutation key
- reject duplicate or stale state with a clear error
- roll back without partial item, currency, auction, mail, party, or guild changes

## Runtime Compatibility

No O3DE, fallback client, launcher, or HTTP route changes are required for this milestone. The existing playable file-backed flow should continue to work.

## Cutover Notes

Before wiring these repositories into runtime services:

- add backend selection and startup validation
- define import/reset behavior for file-store social and economy data
- decide whether auction/mail settlement should be direct, mailbox-delivered, or hybrid
- add human tests for chat, friends, party, guild, vendor, auction, mail, and reconnect flows
- confirm transfer audit rows do not leak tokens, passwords, runtime tickets, or local machine paths

## Clean-Room Boundary

Use AmandaCore-owned table names, DTOs, mutation keys, event names, and transaction semantics. Do not copy external MMO emulator schema layouts, packet models, IDs, command names, comments, or module structures.
