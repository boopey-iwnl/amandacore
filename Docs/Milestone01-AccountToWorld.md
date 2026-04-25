# Milestone 01 - Account To World Hardening

## What It Proves

- A fresh player can register, log in, load the single realm, create a named character, request a join ticket, launch the client, connect to the world, spawn in `west_approach`, disconnect, reconnect, and restore position.
- The authenticated account owns the character selected for join-ticket issuance.
- Join tickets are single-use and cannot be replayed after world connect.
- Character position survives both in-process reconnect and a full local stack restart followed by a fresh login and world join.

## Files Changed

- `Services/internal/authn/http.go`
- `Services/internal/characters/http.go`
- `Services/internal/worlds/http.go`
- `Services/internal/store/file_store.go`
- `Services/internal/platform/types.go`
- `Services/internal/e2e/account_to_world_test.go`
- `Services/internal/observability/events.go`
- `Docs/RuntimeContract.md`

## Local Commands

1. Build and test:

```powershell
powershell -ExecutionPolicy Bypass -File .\Infra\dev\build-local.ps1
```

2. Start the stack and launcher:

```powershell
powershell -ExecutionPolicy Bypass -File .\Infra\dev\start-local.ps1 -StartLauncher
```

3. Run the golden path verifier against the live stack:

```powershell
powershell -ExecutionPolicy Bypass -File .\Infra\dev\verify-golden-path.ps1
```

4. Run the restart-persistence verifier against the live stack:

```powershell
powershell -ExecutionPolicy Bypass -File .\Infra\dev\verify-account-to-world-restart.ps1
```

5. Stop the stack:

```powershell
powershell -ExecutionPolicy Bypass -File .\Infra\dev\stop-local.ps1
```

## Manual Test Steps

1. Start the stack with `start-local.ps1 -StartLauncher`.
2. In the launcher, register a new user or log in with the local dev admin account.
3. Click `Load Realms` and select `Sunset Frontier Dev`.
4. Create a character with a unique name and archetype `wayfarer_warden`.
5. Select the character and click `Join World`.
6. In the world client, move with `W`, `A`, `S`, `D`.
7. Press `X` to disconnect, then `R` to reconnect. Confirm the position is restored.
8. Close the client, stop the local stack, start it again, log back in, join the same character again, and confirm the position is still restored.
9. Inspect `Infra/dev/logs/*.log` and confirm the required structured events were emitted.

## Pass / Fail

Pass:

- A reused join ticket is rejected.
- The world client spawns the selected character at the persisted position.
- Reconnect restores the same character and position.
- Restarting the stack and repeating login -> join -> connect restores the same position.
- The required structured log events are present.

Fail:

- A join ticket can be reused.
- A player can join a character owned by another account.
- Restarting the stack loses the character transform.
- Required structured log events are missing or unstructured.

## What Remains After This Step

- Replace the minimal world client with the thinnest O3DE 3D client path.
- Add server-authoritative 3D movement reconciliation.
- Add target selection, one hostile mob, combat, reward persistence, and acceptance-gate bring-up docs.
