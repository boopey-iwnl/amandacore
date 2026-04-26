# Milestone 2 - O3DE Client Stabilization

Milestone 2 proves the real O3DE client path for the movement-only vertical slice without changing the Milestone 1 backend contract.

## Acceptance target

- The launcher starts `amandacore.GameLauncher`.
- The launcher passes `--join-ticket` and `--world-endpoint`.
- `TestZone01` loads.
- The client logs `client.world_connect_started`, then `client.level_ready`, then `client.world_connected`.
- `client.player_spawned` occurs after successful world connect/bootstrap.
- WASD movement and third-person camera work.
- Disconnect/reconnect restores persisted position.
- No crashes occur on launch, connect, disconnect, or reconnect.

## Build and verification flow

```powershell
powershell -ExecutionPolicy Bypass -File .\Infra\dev\build-local.ps1
powershell -ExecutionPolicy Bypass -File .\Infra\dev\build-o3de-client.ps1
powershell -ExecutionPolicy Bypass -File .\Infra\dev\start-local.ps1 -StartLauncher
powershell -ExecutionPolicy Bypass -File .\Infra\dev\verify-o3de-client.ps1
```

## Manual runtime verification

1. Start the local stack and launcher.
2. Register or log in.
3. Load the single realm and select a character.
4. Click `Join World`.
5. Confirm `amandacore.GameLauncher` starts.
6. Confirm `TestZone01` loads and the player spawns.
7. Move with `W/A/S/D`.
8. Rotate the camera with right mouse.
9. Press `X` to disconnect and `R` to reconnect.
10. Verify the player returns to the persisted position.
11. Repeat after a full local service restart.

## Manual acceptance checklist

### 1. Join World from the launcher

Action:
- Click `Join World` after selecting a realm and character.

Expected launcher logs:
- `Issued world join ticket ...`
- `Resolved client executable path: ...amandacore.GameLauncher.exe`
- `Launch command: ... --join-ticket abcd...wxyz --world-endpoint http://127.0.0.1:8085`
- `Client process start succeeded. Pid: ...`
- `Launched configured game client executable.`

Expected server logs:
- `character.selected`
- `world.join_ticket_issued`
- `world.join_ticket_consumed`

### 2. Level load and initial spawn

Action:
- Wait for the O3DE client window to finish loading.

Expected client logs in `user/log/Game.log`:
- `client.world_connect_started`
- `client.level_ready`
- `client.world_connected reconnect=false ...`
- `client.player_spawned ...`
- `client.camera_activated entity=ThirdPersonCamera`
- `client.camera_attached entity=LocalPlayer`
- `client.input_help move=WASD camera=RMB disconnect=X reconnect=R quit=Q`

Expected server logs:
- `world.player_spawned`

### 3. First movement

Action:
- Press `W`, `A`, `S`, or `D` after the player has spawned.

Expected client logs:
- `client.first_movement_input_received`
- one or more `client.move_submitted ...`
- one or more `client.authoritative_position_applied ...`

Expected server logs:
- `world.character_saved` with `reason":"move"`

### 4. Camera orbit

Action:
- Hold right mouse and move the mouse.

Expected client logs:
- `client.camera_activated entity=ThirdPersonCamera`
- `client.camera_attached entity=LocalPlayer`

Manual visible behavior:
- the camera orbits/follows the player in third-person

### 5. Disconnect and reconnect

Action:
- Press `X` to disconnect, then `R` to reconnect.

Expected client logs:
- `client.disconnect_requested`
- `client.world_disconnected ...`
- `client.reconnect_requested`
- `client.world_connected reconnect=true ...`
- `client.reconnect_completed ...`

Expected server logs:
- `world.character_saved` with `reason":"disconnect"`
- `world.reconnected`

### 6. Quit

Action:
- Press `Q` to quit the O3DE client.

Expected client logs:
- `client.quit_requested`

Manual visible behavior:
- the O3DE client closes cleanly without crashing

## Expected evidence

- `Infra/dev/logs/o3de-assetprocessor-*.log` exists and comes from a successful `AssetProcessorBatch` run.
- The launcher log shows the resolved executable path and the exact command line with a redacted join ticket.
- The O3DE runtime log under `user/log` shows:
  - `client.world_connect_started`
  - `client.level_ready`
  - `client.world_connected`
  - `client.player_spawned`
  - `client.input_help`
  - `client.camera_activated`
  - `client.camera_attached`
  - `client.first_movement_input_received`
