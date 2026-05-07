# Integrated UI Contract

AmandaCore's integrated UI is a dense first-party MMO shell. It keeps the world visible behind most panels, supports multiple simultaneous windows, and treats server state as authoritative for gameplay decisions.

## Global Rules

- The launcher and in-client front-end flow must continue to support login, realm selection, character selection/creation, join-ticket handoff, and world entry.
- In-world HUD frames must remain usable while character, spellbook, quest, map, social, settings, help, vendor, trainer, auction, and bag panels are open.
- UI clicks, text entry, drag/drop, and modal confirmations must not leak into world targeting, movement, camera, or action activation.
- Client UI may preview and request changes, but inventory, equipment, abilities, quests, social/economy state, and world state remain server-owned.
- Missing or invalid local UI settings must fall back to safe defaults.

## Surface Inventory

- Default HUD: player, target, party, combat feed, chat, action bars, objective tracker, navigator, utility menu, tutorial hints, and notifications.
- RPG panels: character, spellbook, talents, professions, quest log, map, social, settings, help, bag, trainer, vendor, auction, mail, dialogue, and confirmation prompts.
- Front-end panels: login, realm list, character roster, character creation, settings, help, and world handoff.

## Persistence

- Local UI settings are stored only as safe client preferences.
- Keybinds are first-party command bindings, not addon commands or scripts.
- No credentials, secrets, tokens, runtime tickets, or machine-local package paths may be written to UI settings.

## Packaging

- UI assets must resolve to repo-controlled paths.
- Release packages must include required repo-side UI assets and exclude local source folders, generated packages, logs, diagnostics, caches, screenshots, local DBs, runtime state, and raw texture dumps.
- Missing icon fallback must remain available.

## Prohibited Surfaces

- Addon APIs, Lua addon loading, AddOns folders, plugin runtimes, user-installed UI modules, arbitrary UI script execution, addon settings, addon managers, addon command systems, addon profile formats, addon package formats, and addon compatibility layers are not part of AmandaCore.
