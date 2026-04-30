# In-Client Login Flow Contract

## Screen Stack

The pre-world UI stack is:

1. Login
2. Remembered-login restore
3. Realm select
4. Character select
5. Character creation
6. Connecting
7. In-world HUD

The launcher starts the game client without `--join-ticket`. Launcher-provided endpoint arguments only tell the client which services to use. The legacy `--join-ticket` and `--world-endpoint` direct-world path remains development/backward-compatible only.

## Login

The login screen presents account and password fields, a `Keep me logged in` checkbox, a login button, status text, error text, settings access, and a forget-login action when a remembered session exists. Password input is masked and is never logged or persisted.

Successful login stores the access token only in runtime client state. When `Keep me logged in` is enabled, the client persists only OS-protected refresh-session material with account/service context. Failed login shows a non-technical error and keeps the player on the login screen.

On boot, if a remembered session exists, the client calls `POST /v1/auth/refresh`. Success proceeds to realm select. Failure clears the remembered session and shows login.

## Realm Select

Realm select lists backend realms. A single realm is still rendered as a selectable row. Selection loads the character list for that realm.

Back returns to login and clears the runtime session state. Logout/forget clears both runtime tokens and the remembered session.

## Character Select

Character select lists characters owned by the authenticated account for the selected realm. The selected details panel shows name, archetype, level, and zone when available.

Enter World requests a join ticket once, then transitions to the connecting screen and the existing world connect path. Create New opens character creation. Delete remains disabled until a safe backend delete flow exists.

## Input Rules

Esc/back returns to the previous pre-world screen when appropriate. Settings closes before screen navigation. Gameplay HUD and movement input are inactive until the world session connects.

## Error Rules

Errors are clear and player-facing. Expired sessions send the player back through login. Passwords, bearer tokens, refresh tokens, world-session tokens, credential blobs, and join tickets are not logged.
