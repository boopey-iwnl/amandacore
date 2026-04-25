# Alpha 0.1 Feature Freeze

Alpha 0.1 is a controlled release candidate for the existing playable slice. It is not a content expansion. Testers should stay on the assigned route and file bugs against build, launch, account, world entry, Stonewake progression, combat, inventory, persistence, diagnostics, and recovery.

## Release Buckets

| System | Status | Required validation | Known risks | Owner notes |
| --- | --- | --- | --- | --- |
| login/account | release-critical | register, login, refresh/logout smoke | session or store corruption | backend |
| launcher/local ops | release-critical | GUI opens, build/start/stop, logs, diagnostics | stale binary selection | tools |
| character creation/select | release-critical | Human Warrior create/select | archetype or roster mismatch | backend/launcher |
| world join | release-critical | ticket, connect, spawn | client/server manifest drift | backend/client |
| Stonewake Vale | release-critical | assigned starter route | quest/content drift | content/QA |
| second zone | included rough | optional Brindlebrook handoff | incomplete route polish | optional route only |
| quests | release-critical | accept, progress, complete, reward once | broken chain blocks release | gameplay |
| combat | release-critical | target, attack, ability, mob death/respawn | PvP/dungeon interactions | gameplay |
| camera/movement | release-critical | WASD, grounded movement, attached camera | O3DE build drift | client |
| NPC interaction | release-critical | friendly target and right-click services | service type mismatch | gameplay |
| trainer/spellbook/action bars | release-critical | learn, spellbook, assign, use | costs or saved bar mismatch | gameplay |
| inventory/currency | release-critical | move, spend, persist | dupes after restart | backend |
| vendors/equipment/loot | release-critical | buy, sell, equip, loot | economy side effects | gameplay |
| professions | included rough | learn/gather/craft smoke | incomplete depth | optional route |
| map/minimap/quest log | release-critical | markers and quest visibility | stale UI state | client |
| chat/friends/party | included rough | optional two-client smoke | session edge cases | optional route |
| guilds | disabled-hidden | not in main tester route | management edge cases | API/admin only |
| mail/trade/auction | disabled-hidden | not in main tester route | recent economy code | hidden from tester flow |
| dungeon prototype | included rough | optional entry/exit/reset | instance recovery | optional route |
| PvP duels | disabled-hidden | disabled in local alpha startup | active duel state edge cases | keep off for RC |
| housing/storage | disabled-hidden | not in main tester route | return/save edge cases | hidden from tester flow |
| achievements/titles/collections | deferred | none | not part of Alpha 0.1 | defer |
| travel/mounts | deferred | none | not part of Alpha 0.1 | defer |
| admin/support tools | included rough | admin-gated diagnostics/support only | unsafe exposure | operators only |
| diagnostics/bug reporting | release-critical | redacted bundle and templates | secret/log leakage | QA/tools |

## Release Blockers

- latest project cannot build
- local/test stack cannot start
- launcher cannot open
- login or character create/select fails
- world join fails
- player spawns into a blank or broken world
- camera or movement is unusable
- crash in the first 10 minutes
- persistence corrupts or silently deletes character state
- first playable quest loop cannot progress
- NPC interaction or combat is impossible
- diagnostic bundle cannot be collected
- tester package cannot run
- basic two-client sanity crashes the server
- required reset/export/import recovery tools are broken

## Non-Blockers

Visual polish, balance, placeholder art/icons, rough animation, minor clipping, incomplete deferred systems, and optional-route rough edges are non-blockers if they do not prevent the required route.
