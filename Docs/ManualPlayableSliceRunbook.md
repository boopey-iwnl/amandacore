# Manual Playable Slice Runbook

Use this runbook to drive the current Alpha 0.1 `stonewake_vale` slice through the Local Playable Slice Controls desktop shortcut, the real launcher, and the O3DE client.

## Local Ops GUI

- Desktop shortcut: `C:\Users\forwo\OneDrive\Desktop\Local Playable Slice Controls.lnk`
- GUI source: `C:\Users\forwo\OneDrive\Desktop\Code Project - Alpha Integration\Infra\dev\LocalOpsGui.ps1`
- Double-click wrapper: `C:\Users\forwo\OneDrive\Desktop\Code Project - Alpha Integration\Infra\dev\Launch-LocalOpsGui.cmd`
- PowerShell launch command:

```powershell
powershell -NoLogo -NoProfile -ExecutionPolicy Bypass -File "C:\Users\forwo\OneDrive\Desktop\Code Project - Alpha Integration\Infra\dev\LocalOpsGui.ps1"
```

The GUI wraps the existing local scripts instead of replacing them:

- `Build + Restart Stack` runs `start-local.ps1` with `-BuildFirst`.
- `Start Services` runs `start-local.ps1` with `-BuildFirst:$false`.
- `Stop Local Stack` runs `stop-local.ps1` and then closes any lingering launcher/client/service processes.
- `Open Launcher` builds the latest playable client binaries and starts `Client\Launcher\AmandaCore.Launcher\bin\Debug\net8.0-windows\AmandaCore.Launcher.exe`.
- `Open Logs Folder` opens:
  - `C:\Users\forwo\OneDrive\Desktop\Code Project - Alpha Integration\Infra\dev\logs`
  - `C:\Users\forwo\OneDrive\Desktop\Code Project - Alpha Integration\user\log`

## Main Logs and Local State

- Service logs folder: `C:\Users\forwo\OneDrive\Desktop\Code Project - Alpha Integration\Infra\dev\logs`
- Game client log: `C:\Users\forwo\OneDrive\Desktop\Code Project - Alpha Integration\user\log\Game.log`
- User log folder: `C:\Users\forwo\OneDrive\Desktop\Code Project - Alpha Integration\user\log`
- Process manifest: `C:\Users\forwo\OneDrive\Desktop\Code Project - Alpha Integration\Infra\dev\local-processes.json`
- Local persisted state: `C:\Users\forwo\AppData\Local\amandacore\platform-state.json`
- Load-test output: `C:\Users\forwo\OneDrive\Desktop\Code Project - Alpha Integration\Infra\dev\load-tests`

## Human Test Flow

### 1. Start the stack

1. Double-click `Local Playable Slice Controls` on the desktop.
2. Click `Build + Restart Stack` when you want the latest binaries, or `Start Services` when the build is already current.
3. Wait until the GUI shows:
   - `Stack: Running`
   - all six services as `Healthy`

### 2. Open the launcher

1. Click `Open Launcher`.
2. Confirm the GUI shows `Launcher: Running`.
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
   - `C:\Users\forwo\OneDrive\Desktop\Code Project - Alpha Integration\user\log\Game.log`
   - the relevant world-service log in `C:\Users\forwo\OneDrive\Desktop\Code Project - Alpha Integration\Infra\dev\logs`

### 7. Verify reconnect behavior

1. While the client is running, trigger the current reconnect flow.
2. Confirm the player reconnects into `stonewake_vale`.
3. Confirm persisted position and quest state remain correct.
4. Confirm transient combat and interaction window state is cleared appropriately.

### 8. Verify full restart persistence

1. Close the game client.
2. In the Local Ops GUI, click `Stop Local Stack`.
3. Wait until the GUI shows `Stack: Stopped`.
4. Click `Start Services` or `Build + Restart Stack`.
5. Click `Open Launcher`, log back in, and `Join World`.
6. Confirm completed quest state and rewards persisted across the full restart.
7. Confirm no duplicate reward was granted after re-entry.

### 9. Run a short simulated multi-client soak

1. Keep the local stack running.
2. Run:

```powershell
powershell -ExecutionPolicy Bypass -File "C:\Users\forwo\OneDrive\Desktop\Code Project - Alpha Integration\Infra\dev\run-load-test.ps1" -Clients 2 -Scenario mixed -DurationMinutes 5
```

3. Inspect the generated `summary.md`, `summary.json`, and `events.jsonl` under `C:\Users\forwo\OneDrive\Desktop\Code Project - Alpha Integration\Infra\dev\load-tests`.
4. Confirm the world service stayed running and the summary shows endpoint timings, error counts, session counts, and desync count.

## Expected In-Client Acceptance

The slice is considered ready for manual acceptance only if a human can confirm all of the following from the real client window:

- desktop `Local Playable Slice Controls` opens the integration GUI
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
