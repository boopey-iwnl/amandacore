# Milestone 8 Next Steps: 3.3.5a-Era MMO Guidance Reference

This document records recommended next steps for quest log, map, minimap, navigation, and world guidance polish using World of Warcraft 3.3.5a as a systems reference point only.

Clean-room rule: use the broad gameplay patterns that made that era readable, not its names, icons, map art, quest text, UI frames, colors, sounds, or trade dress. AmandaCore should keep original Stonewake Vale names, visuals, marker language, and interface styling.

## Product Target

Milestone 8 should make Stonewake Vale feel self-explanatory without turning the game into a GPS overlay. The player should understand which quests exist, what state each quest is in, where to go next, and which NPC or area matters, while still navigating by landmarks and roads.

The 3.3.5a-era reference point is useful because it was direct, text-forward, and reliable:

- Quest log and objective tracker were always available, predictable, and compact.
- Quest state was legible from NPC markers, tracker text, and map objective hints.
- The minimap acted as an orientation tool rather than a full navigation solution.
- Routes and hubs mattered because the world was understood through roads, named areas, and recognizable NPC services.

## Recommended Next Steps

### 1. Stabilize The Quest Data Contract

Treat quest state as a first-class session payload, not a single starter-quest special case.

Add or verify payload support for:

- Active quests
- Available quests
- Completed and reward-granted quests
- Ready-to-turn-in quests
- Tracked quest IDs
- Objective progress
- Objective area metadata
- Reward preview
- Giver and turn-in NPC IDs
- Recommended level or difficulty hint when useful

Keep this generic enough for later zones, but authored enough that Stonewake Vale can ship without final map art.

### 2. Make The Quest Log The Source Of Truth

The quest log should become the player-facing control center for quest state.

Recommended behavior:

- Group quests by status: active, ready to turn in, available, completed.
- Show objective text, progress, reward preview, and turn-in NPC.
- Show the relevant Stonewake area name and route hint for active quests.
- Let the player track or untrack quests from the log.
- Keep the layout dense and readable, closer to a practical tool than a decorative journal.

Avoid copying parchment-like frames, exact panel structure, icon shapes, or text styling from the reference game.

### 3. Cap And Prioritize The Objective Tracker

Use the tracker as a compact moment-to-moment task list.

Recommended behavior:

- Track accepted quests automatically, up to a small cap.
- Support manual track and untrack from the quest log.
- Display two to four tracked quests comfortably.
- Show progress as `current / target`.
- When complete, change the wording to a clear return instruction.
- Prefer route hint text over exact GPS-style arrows.

For the first polished version, cap tracked quests at 3. This keeps the HUD readable in the starter zone.

### 4. Use Authored Objective Areas Instead Of Pathfinding

Do not build advanced pathfinding yet. Use authored navigation areas and route hints.

Recommended metadata per area:

- `areaId`
- `displayName`
- `kind`
- `centerX`
- `centerY`
- `radius`
- `questIds`
- `routeHintText`
- Optional target mob type or target entity ID

This mirrors the practical clarity of 3.3.5a-era questing while staying cheap, deterministic, and original.

### 5. Build The Stonewake Map As A Blockout Tool

The zone map should communicate shape and intent before final art exists.

Recommended map layers:

- Zone bounds
- Hub marker
- Main roads and paths
- Trainer marker
- Vendor marker
- Quest giver and turn-in markers
- Tracked objective area markers
- Named area labels
- Handoff route marker toward future content
- Player position marker

The map should look like an original tactical blockout, not a fantasy parchment map.

### 6. Keep The Minimap As Orientation, Not Autopilot

For this milestone, a minimap/compass hybrid is enough.

Recommended behavior:

- Always show the player near the center.
- Show nearby quest/service markers.
- Show tracked objective direction or nearby objective marker.
- Show hostile areas only when useful for learning the starter zone.
- Avoid exact route lines unless they are authored road hints.
- Add zoom only if it is trivial and stable.

This captures the usefulness of the era without recreating its frame, iconography, or circular minimap presentation.

### 7. Standardize Quest And NPC Marker Semantics

Markers must be consistent everywhere: above NPCs, on map, and on navigator.

Recommended marker states:

- Available quest
- Quest in progress
- Ready to turn in
- Completed and already rewarded
- Trainer
- Vendor
- Optional unavailable quest marker, only if it does not create confusion

Use original symbols, colors, and text labels. Do not copy question-mark or exclamation-mark trade dress from another game.

### 8. Add Location Discovery Only Where It Helps Guidance

Location discovery should be simple and restrained.

Recommended first slice:

- Discover named Stonewake areas when the player enters an authored radius.
- Show a small HUD toast with the original area name.
- Persist discovered area IDs.
- Optionally dim undiscovered area labels on the map.

Do not add exploration XP, achievements, or a completion meta-system yet.

### 9. Persist UI Preferences And Tracked State

The game should remember how the player arranged core guidance.

Recommended persistence:

- Tracked quest IDs on the character.
- Map window open state only if the existing UI settings pattern supports it cleanly.
- Minimap visibility and scale only if settings are already present.
- Discovered area IDs if location discovery is included.

Reconnect and restart must preserve tracked quests.

### 10. Manual Playtest The Starter Flow Like A New Player

Run a human validation pass after every meaningful change:

- Create a human warrior.
- Accept the first Stonewake quest.
- Use tracker text to reach the trainer or objective area.
- Open the quest log and toggle tracking.
- Open the map and confirm marker accuracy.
- Use the minimap/navigator while moving.
- Complete an objective and confirm turn-in state changes.
- Restart/reconnect and confirm tracked state remains.

## Near-Term Implementation Order

1. Confirm all Milestone 8 payload fields are parsed by NetClient and surfaced by GameCore.
2. Finish quest log grouping, tracking toggles, and reward/objective display.
3. Render all Stonewake map metadata through a primitive zone map.
4. Mount the navigator/minimap in the HUD and feed it authored map markers.
5. Add overhead NPC markers for quest and service state.
6. Persist tracked quests and UI/map preferences.
7. Add automated tests for payload, tracking persistence, map metadata, and ready-to-turn-in markers.
8. Run an end-to-end starter-zone playthrough and record the manual validation result.

## Risks

- The UI can become cluttered if the tracker, map, and navigator all compete for screen space.
- Marker meaning can drift if NPC, map, and tracker states are computed separately.
- Overusing exact objective arrows can make the world feel like a checklist instead of a place.
- Copying the reference game's visual language would create clean-room risk.
- Large e2e tests currently produce noisy logs, which can hide failing assertions and slow iteration.

## Recommended First Follow-Up Slice

Ship a focused "Guidance Contract Hardening" slice:

- Add tests that assert `quests`, `trackedQuestIds`, `zoneMap`, `navigationAreas`, and `mapMarkers` exist on every world session response.
- Add tests for available, active, ready-to-turn-in, and reward-granted quest buckets.
- Add tests for tracked quest persistence through disconnect, reconnect, and store restart.
- Add one manual runbook section for validating the Stonewake quest log, map, navigator, and marker states.

This is the safest next step because it locks down the guidance data before adding more UI polish.
