# In-Client Login and Character Creation Runbook

## Setup

1. Check out `codex/ui-m2-login-character-creation`.
2. Start the local stack with `Infra/dev/start-local.ps1 -StartLauncher`.
3. Use the launcher Play button to start the O3DE game client.

## Launcher Patcher Path

1. Open the launcher.
2. Confirm it shows patch/build status, endpoint status, resolved client executable, and Play.
3. Confirm it does not show player username/password login, realm selection, character creation, or Join World controls.
4. Click Play.
5. Confirm the launcher starts the O3DE game client without username, password, bearer token, refresh token, selected realm, selected character, account ID, character ID, or join ticket arguments.

## In-Client Path

1. Confirm the game client login screen appears.
2. Log in with a local account.
3. Test `Keep me logged in`, restart the client, and confirm refresh-token restore proceeds to realm select.
4. Test logout/forget and confirm the remembered login is cleared.
5. Select a realm.
6. Select an existing character or open Create New.
7. In creation, adjust preview options, rotate/zoom/reset/randomize the preview, and enter a valid name.
8. Create the character.
9. Confirm the new character appears in character select.
10. Enter world.
11. Confirm world load, movement, camera, action bars, inventory, and chat.

## Direct-World Development Path

The legacy `--join-ticket` and `--world-endpoint` handoff remains available only for development and backward-compatible direct-world launch. It is not the normal launcher Play flow.

## Known Limitations

Appearance customization is preview-only. The backend does not persist appearance fields in this milestone.

Character deletion remains disabled because there is no safe backend delete contract for this flow.

## Safety Checks

Before commit and push, verify:

- no secrets, logs, diagnostics, screenshots, zips, local DBs, runtime tickets, generated packages, Cache/build output, or temp files are staged
- no runtime/package reference to the local texture source folder
- no credentials, tokens, credential blobs, or join tickets are passed through launcher process arguments or logged
- no AddOns folder, addon API, Lua addon loading, plugin runtime, user-installed UI modules, or arbitrary UI script execution
- no copied external game assets, private-server code, addon code, or protected UI trade dress
