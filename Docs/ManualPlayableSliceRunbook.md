# Manual Playable Slice Runbook

Use this runbook to drive the current single-zone `sunset_frontier / west_approach` slice through the real launcher and O3DE client without adding any new gameplay scope.

## Local Ops GUI

- GUI source: `C:\Users\forwo\OneDrive\Desktop\Code Project\Infra\dev\LocalOpsGui.ps1`
- Double-click wrapper: `C:\Users\forwo\OneDrive\Desktop\Code Project\Infra\dev\Launch-LocalOpsGui.cmd`
- PowerShell launch command:

```powershell
powershell -NoLogo -NoProfile -ExecutionPolicy Bypass -File "C:\Users\forwo\OneDrive\Desktop\Code Project\Infra\dev\LocalOpsGui.ps1"
```

The GUI wraps the existing local scripts instead of replacing them:

- `Start Local Stack` runs `C:\Users\forwo\OneDrive\Desktop\Code Project\Infra\dev\start-local.ps1` with `-BuildFirst:$false`
- `Stop Local Stack` runs `C:\Users\forwo\OneDrive\Desktop\Code Project\Infra\dev\stop-local.ps1`
- `Open Launcher` starts `C:\Users\forwo\OneDrive\Desktop\Code Project\Client\Launcher\AmandaCore.Launcher\bin\Debug\net8.0-windows\AmandaCore.Launcher.exe`
- `Open Logs Folder` opens:
  - `C:\Users\forwo\OneDrive\Desktop\Code Project\Infra\dev\logs`
  - `C:\Users\forwo\OneDrive\Desktop\Code Project\user\log`

## Main Logs and Local State

- Service logs folder: `C:\Users\forwo\OneDrive\Desktop\Code Project\Infra\dev\logs`
- Game client log: `C:\Users\forwo\OneDrive\Desktop\Code Project\user\log\Game.log`
- User log folder: `C:\Users\forwo\OneDrive\Desktop\Code Project\user\log`
- Process manifest: `C:\Users\forwo\OneDrive\Desktop\Code Project\Infra\dev\local-processes.json`
- Local persisted state: `C:\Users\forwo\AppData\Local\amandacore\platform-state.json`
- Load-test output: `C:\Users\forwo\OneDrive\Desktop\Code Project\Infra\dev\load-tests`

## Human Test Flow

### 1. Start the stack

1. Launch the Local Ops GUI.
2. Click `Start Local Stack`.
3. Wait until the GUI shows:
   - `Stack: Running`
   - `Running Services: auth-service, account-service, realm-service, character-service, world-service, admin-service`

### 2. Open the launcher

1. Click `Open Launcher`.
2. Confirm the GUI shows `Launcher: Running`.
3. In the launcher:
   - register a new account or log in with an existing local account
   - select the local realm
   - create a character if needed
   - select the character and click `Join World`

### 3. Validate world entry and third-person play

1. Confirm the O3DE client opens into a visible `TestZone01 / West Approach` scene.
2. Confirm the player is visible on-screen as a third-person character/proxy.
3. Confirm the camera is attached behind and above the player rather than behaving like a detached observer.
4. Confirm the player remains grounded on the arena floor while moving with `WASD`.
5. Confirm the HUD is visible and readable enough to show:
   - player frame
   - target frame
   - quest tracker

### 4. Accept the quest and fight the hostile mobs

1. At the command post, accept the field order quest in-client.
2. Move from spawn toward the encounter pocket.
3. Confirm exactly three hostile mobs are visible.
4. Press `Tab` and confirm the target cycles through visible mobs in a stable order.
5. Left-click a mob and confirm the selected target matches what was clicked.
6. Use:
   - `F` for auto-attack
   - `1` for the first ability
   - `2` for the second ability
7. Kill the required mobs and confirm quest progress increases once per valid kill.

### 5. Verify completion and rewards

1. Finish the quest objective in-client.
2. Confirm completion is shown in the HUD.
3. Confirm rewards are granted exactly once.
4. Check:
   - the in-client HUD/state
   - `C:\Users\forwo\OneDrive\Desktop\Code Project\user\log\Game.log`
   - the relevant world-service log in `C:\Users\forwo\OneDrive\Desktop\Code Project\Infra\dev\logs`

### 6. Verify reconnect behavior

1. While the client is running, trigger the current reconnect flow.
2. Confirm the player reconnects into the same zone.
3. Confirm persisted position and quest state remain correct.
4. Confirm transient combat state is cleared appropriately.

### 7. Verify full restart persistence

1. Close the game client.
2. In the Local Ops GUI, click `Stop Local Stack`.
3. Wait until the GUI shows `Stack: Stopped`.
4. Click `Start Local Stack` again.
5. Click `Open Launcher`, log back in, and `Join World`.
6. Confirm completed quest state and rewards persisted across the full restart.
7. Confirm no duplicate reward was granted after re-entry.

### 8. Run a short simulated multi-client soak

1. Keep the local stack running.
2. Run:

```powershell
powershell -ExecutionPolicy Bypass -File "C:\Users\forwo\OneDrive\Desktop\Code Project\Infra\dev\run-load-test.ps1" -Clients 2 -Scenario mixed -DurationMinutes 5
```

3. Inspect the generated `summary.md`, `summary.json`, and `events.jsonl` under `C:\Users\forwo\OneDrive\Desktop\Code Project\Infra\dev\load-tests`.
4. Confirm the world service stayed running and the summary shows endpoint timings, error counts, session counts, and desync count.

## Expected In-Client Acceptance

The slice is considered ready for manual acceptance only if a human can confirm all of the following from the real client window:

- launcher `Join World` path works
- third-person character is visible and the camera is attached to it
- grounded movement works
- three hostile mobs are visible
- tab targeting works
- click targeting works
- quest acceptance works
- quest progress increments once per valid kill
- completion and rewards happen exactly once
- reconnect preserves state correctly
- restart preserves rewarded state correctly
