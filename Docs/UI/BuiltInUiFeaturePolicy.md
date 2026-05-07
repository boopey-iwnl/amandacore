# Built-In UI Feature Policy

AmandaCore can ship advanced UI features, but they must be built as first-party game systems.

## Policy

- UI features are implemented in AmandaCore source, reviewed with the game, tested with the game, and packaged with the game.
- UI behavior must respect server authority for combat, inventory, quests, currency, social state, and world sessions.
- UI settings may persist locally only through approved local settings paths.
- UI content and package manifests must use repo-relative assets.
- Local source art folders are read-only inputs only; runtime code and package contents must depend on project-local assets.

## Allowed Built-In Features

- frame movement and reset/default layout behavior
- keybind mode for built-in commands and action slots
- action-bar paging and visibility rules
- bag search, sort, and rearranging when backed by real behavior
- objective tracker filters
- combat text and chat filters
- threat, party, raid, nameplate, and tooltip options
- equipment manager and UI profiles when implemented as first-party systems

## Not Allowed

- third-party addon runtime
- arbitrary UI script execution
- runtime plugin loading
- user-installed UI modules
- package or runtime references to machine-local source asset folders
- fake controls that appear to work without a real implementation
