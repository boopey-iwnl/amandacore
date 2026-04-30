# In-Client Login Flow Contract

## Screen Stack

The pre-world UI stack is:

1. Login
2. Realm select
3. Character select
4. Character creation
5. Connecting
6. In-world HUD

Launcher-provided `--join-ticket` and `--world-endpoint` skip the pre-world stack and enter the existing world bootstrap path.

## Login

The login screen presents account and password fields, a login button, status text, error text, and settings access. Password input is masked and is never logged or persisted by the client.

Successful login stores the access token only in runtime client state and loads the realm list. Failed login shows a non-technical error and keeps the player on the login screen.

## Realm Select

Realm select lists backend realms. A single realm is still rendered as a selectable row. Selection loads the character list for that realm.

Back returns to login and clears the runtime session state.

## Character Select

Character select lists characters owned by the authenticated account for the selected realm. The selected details panel shows name, archetype, level, and zone when available.

Enter World requests a join ticket once, then transitions to the connecting screen and the existing world connect path. Create New opens character creation. Delete remains disabled until a safe backend delete flow exists.

## Input Rules

Esc/back returns to the previous pre-world screen when appropriate. Settings closes before screen navigation. Gameplay HUD and movement input are inactive until the world session connects.

## Error Rules

Errors are clear and player-facing. Expired sessions send the player back through login. Passwords and tokens are not logged.
