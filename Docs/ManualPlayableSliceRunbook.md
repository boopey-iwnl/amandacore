# Manual Playable Slice Runbook

Use this runbook to drive the current Alpha 0.1 `stonewake_vale` slice through the Local Playable Slice Controls desktop shortcut, the real launcher, and the O3DE client.

## Local Controls App

- Desktop shortcut: `C:\Users\forwo\OneDrive\Desktop\Local Playable Slice Controls.lnk`
- App project: `<repo root>\Client\Tools\AmandaCore.LocalControls\AmandaCore.LocalControls.csproj`
- Double-click wrapper: `<repo root>\Infra\dev\Launch-LocalOpsGui.cmd`
- Direct launch command:

```powershell
dotnet run --project "<repo root>\Client\Tools\AmandaCore.LocalControls\AmandaCore.LocalControls.csproj"
```

The compiled app wraps the existing local scripts instead of replacing them:

- `Start local stack` runs `start-local.ps1 -BuildFirst`.
- `Stop local stack` runs `stop-local.ps1`.
- `Build local` runs `build-local.ps1`.
- `Build O3DE client` runs `build-o3de-client.ps1`.
- `Verify O3DE client` runs `verify-o3de-client.ps1`.
- `Launch AmandaCore Launcher` builds the latest playable client binaries and starts `Client\Launcher\AmandaCore.Launcher\bin\Debug\net8.0-windows\AmandaCore.Launcher.exe`.
- `Open logs/output folder` opens the first available local logs or output folder.
- `Collect diagnostics` runs `Infra\qa\Collect-Diagnostics.ps1`.
- `Open QA docs` opens `Docs\QA`.
- `Reset test state` confirms first, then runs `Infra\qa\Reset-LocalTestState.ps1 -All -ConfirmReset`.
- `Open admin portal` opens `Client\Portal\admin-portal.html`.

## Main Logs and Local State

- Service logs folder: `<repo root>\Infra\dev\logs`
- Game client log: `<repo root>\user\log\Game.log`
- User log folder: `<repo root>\user\log`
- Process manifest: `<repo root>\Infra\dev\local-processes.json`
- Local persisted state: `C:\Users\forwo\AppData\Local\amandacore\platform-state.json`
- Load-test output: `<repo root>\Infra\dev\load-tests`

## Human Test Flow

### 1. Start the stack

1. Double-click `Local Playable Slice Controls` on the desktop.
2. Click `Start local stack` to build current binaries and start the stack.
3. Wait until the command output reports `Local amandacore stack started.`; `start-local.ps1` waits for all six services to become healthy before returning.

### 2. Open the launcher

1. Click `Launch AmandaCore Launcher`.
2. Confirm the command output reports `Launcher opened`.
3. In the launcher:
   - register a new account or log in with an existing local account
   - select the local realm
   - create a character if needed
   - select the character and click `Join World`

### 3. Validate Stonewake world entry

1. Confirm the O3DE client opens into the Stonewake scene.
2. Confirm the player is visible as a third-person character/proxy.
3. Confirm the camera is attached behind and above the player.
4. Confirm the player remains grounded while moving with `WASD`.
5. Confirm the HUD is visible and readable enough to show:
   - player/target frames at top-left
   - minimap and objectives on the right
   - chat bottom-left
   - action bars bottom-center/right

### 4. Validate NPC interaction lifecycle

1. Right-click the first quest giver in Hearthwatch.
2. Accept the starter quest and confirm the quest/gossip window closes or transitions cleanly.
3. Move out of interaction range and confirm any NPC window closes.
4. Right-click a different NPC and confirm the old interaction context does not remain open.
5. Open an NPC interaction window, press `Esc`, and confirm the interaction closes before the system menu opens.

### 5. Validate hub readability, map, and combat

1. Confirm the spawn view clearly separates the quest giver, trainer, road/path, service area, training ring, and first hostile pocket.
2. Open the map and confirm Stonewake labels and markers are readable without heavy overlap.
3. Move to the first hostile/objective pocket.
4. Use:
   - `Tab` to cycle visible targets
   - `F` for auto-attack
   - `1` and `2` for abilities
5. Kill required hostile mobs and confirm quest progress increases once per valid kill.

### 6. Verify completion and rewards

1. Finish the active quest objective in-client.
2. Confirm completion is shown in the HUD/objective tracker.
3. Return to the correct turn-in NPC.
4. Confirm rewards are granted exactly once.
5. Check:
   - the in-client HUD/state
   - `<repo root>\user\log\Game.log`
   - the relevant world-service log in `<repo root>\Infra\dev\logs`

### 7. Verify reconnect behavior

1. While the client is running, trigger the current reconnect flow.
2. Confirm the player reconnects into `stonewake_vale`.
3. Confirm persisted position and quest state remain correct.
4. Confirm transient combat and interaction window state is cleared appropriately.

### 8. Verify full restart persistence

1. Close the game client.
2. In the local controls app, click `Stop local stack`.
3. Wait until the command finishes with exit code `0`.
4. Click `Start local stack`.
5. Click `Launch AmandaCore Launcher`, log back in, and `Join World`.
6. Confirm completed quest state and rewards persisted across the full restart.
7. Confirm no duplicate reward was granted after re-entry.

### 9. Run a short simulated multi-client soak

1. Keep the local stack running.
2. Run:

```powershell
powershell -ExecutionPolicy Bypass -File "<repo root>\Infra\dev\run-load-test.ps1" -Clients 2 -Scenario mixed -DurationMinutes 5
```

3. Inspect the generated `summary.md`, `summary.json`, and `events.jsonl` under `<repo root>\Infra\dev\load-tests`.
4. Confirm the world service stayed running and the summary shows endpoint timings, error counts, session counts, and desync count.

## Expected In-Client Acceptance

The slice is considered ready for manual acceptance only if a human can confirm all of the following from the real client window:

- desktop `Local Playable Slice Controls` opens the compiled local controls app
- launcher `Join World` path works
- third-person character is visible and the camera is attached to it
- grounded movement works
- Stonewake hub, trainer, road, and first hostile pocket are readable
- map/minimap labels are readable
- tab targeting works
- click targeting works
- quest acceptance closes or updates the NPC window correctly
- walking away, target switching, and `Esc` close stale interaction windows
- quest progress increments once per valid kill
- completion and rewards happen exactly once
- reconnect preserves persisted state and clears transient interaction/combat state
- restart preserves rewarded state correctly
