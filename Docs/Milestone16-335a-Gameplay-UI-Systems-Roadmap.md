# Milestone 16 Next Steps: 3.3.5a-Era Gameplay, UI, and Systems Roadmap

This document recommends the next implementation direction after the multi-client stability foundation. World of Warcraft 3.3.5a is used as a gameplay, UI, and systems reference point only.

Clean-room rule: copy no names, quests, zones, icons, art, sounds, animation timing, UI frames, text, encounter designs, or trade dress. AmandaCore should adapt durable MMO patterns into original systems, original visuals, and original Stonewake-era content.

## Product Target

Milestone 16 should make the current MMO slice feel more like a readable, hotkey-driven, server-authoritative MMORPG without adding large new content categories. The target is not feature breadth. The target is stable moment-to-moment play: clear targets, reliable action feedback, understandable NPC state, compact UI, and systems that can support later group play.

3.3.5a-era design is useful because it emphasized:

- Hotkey-first combat with mouse support.
- Dense but predictable HUD information.
- Clear target, cast, cooldown, and combat feedback.
- Server-authoritative outcomes with client-side responsiveness.
- Practical quest, vendor, trainer, inventory, and social workflows.
- Low-friction solo play that still prepared the game for parties.

## Recommended Milestone Sequence

### Milestone 16: Core MMORPG Feel Pass

Focus on the play loop that happens every few seconds.

Recommended scope:

- Target frame and target-of-target data if available.
- Player unit frame with health, resource, level, and combat state.
- Compact action bar polish for keybind labels, cooldown sweep, disabled state, and range/resource feedback.
- Global cooldown visualization and server-confirmed action result feedback.
- Cast bar or channel bar only if any current actions require timing.
- Combat log foundation with concise event categories.
- Nameplate readability for hostile, friendly, dead, elite, quest-relevant, and interactable entities.
- Loot, vendor, trainer, and quest dialogs using consistent interaction panel patterns.
- Error feedback for invalid target, out of range, missing resource, cooldown active, dead target, and movement restrictions.

Acceptance criteria:

- A new player can identify self state, target state, action availability, and combat outcome without reading logs.
- Solo login, movement, camera, quests, vendors, professions, equipment, social panels, and reconnect still work.
- No new gameplay content is required.

### Milestone 17: Combat Rules Hardening

Make combat predictable before adding classes or encounter complexity.

Recommended scope:

- Server-side global cooldown and per-action cooldown enforcement.
- Auto-attack lifecycle: start, stop, target death, out-of-range, leash, disconnect, and reconnect.
- Melee range validation and facing/line checks only if already supported cheaply.
- Threat table foundation for mobs, with deterministic target selection.
- Evade/leash behavior for mobs that lose valid targets.
- Death, respawn, corpse return, and durability hooks only at foundation level.
- Combat log event schema for damage, miss, death, threat target changes, resource changes, and XP/loot outcomes.
- Unit tests for concurrent combat actions from multiple sessions.

Avoid:

- Adding many new abilities.
- Rebalancing the whole Warrior class.
- Designing dungeon or raid encounters.

### Milestone 18: Quest and NPC Workflow Maturity

Polish existing questing and interaction workflows using compact, reliable MMO conventions.

Recommended scope:

- Quest log grouping by active, complete, available, and completed history.
- Objective tracker cap, manual track/untrack, and reconnect persistence.
- Gossip-style NPC interaction data model using original labels and layouts.
- Quest reward choice support if needed by existing content.
- Item-use quest objective hook if it fits current data cleanly.
- Map/minimap markers driven from the same quest/NPC state source.
- Trainer and vendor discoverability through map and NPC marker metadata.

Avoid:

- Large zone expansion.
- Copying quest text patterns or marker trade dress.
- Advanced pathfinding or automated routing.

### Milestone 19: Inventory, Economy, and Profession Usability

Make the already-started item systems feel dependable and inspectable.

Recommended scope:

- Bag slot clarity, item count, item quality, sell price, stack size, and tooltip consistency.
- Vendor buy/sell feedback, insufficient currency messaging, and buyback only if cheap.
- Equipment comparison tooltip foundation.
- Durability and repair hooks only if combat death and vendor roles need them.
- Profession recipe list, required materials, craft result preview, and error states.
- Bank/storage planning document before implementation.

Avoid:

- Auction house.
- Mail.
- Player trading unless it is explicitly scoped later.
- Economy simulation beyond current vendor/currency needs.

### Milestone 20: Social and Party Readiness

Prepare for party play without building dungeons yet.

Recommended scope:

- Party invite, accept, decline, leave, leader, and disconnect behavior.
- Party frames with health, level, zone, online state, and target/combat indicators.
- Chat channel polish for say, party, system, combat, and errors.
- Friends and ignore list persistence if current foundation supports it.
- Group loot rule design document, not full implementation unless required.
- Shared mob tagging and quest credit rules for party members in the same zone.

Avoid:

- Guilds.
- Matchmaking.
- Cross-zone group finder.
- Dungeon queue.

### Milestone 21: Dungeon-Ready Technical Foundation

Build the plumbing needed for later instanced content, but do not ship dungeons in this milestone.

Recommended scope:

- Instance/session ownership design.
- Party transfer and reconnect rules.
- Mob reset boundaries and wipe cleanup design.
- Loot ownership and lockout design notes.
- Server metrics for party and instance lifecycle.
- Soak tests for a party moving and fighting together.

Avoid:

- Actual dungeons, raids, bosses, or scripted encounters.

## UI Direction

The UI should be dense, readable, and hotkey-oriented. It should feel like a tool for repeated play, not a landing page or cinematic overlay.

Recommended UI principles:

- Keep primary HUD information visible without large decorative frames.
- Use compact panels for action bars, bags, character info, quest log, map, social, and vendor/trainer workflows.
- Prefer consistent status colors and labels over ornate visuals.
- Show cooldowns, resources, range, and invalid-action reasons at the point of use.
- Make keyboard shortcuts visible on buttons where they matter.
- Let server state correct the UI, but preserve responsive local button feedback.
- Keep chat, combat log, tracker, and minimap from competing for the same screen space.

Clean-room constraints:

- Do not recreate the exact action bar, minimap frame, unit frame, bag frame, fonts, icons, textures, sounds, or layout proportions of the reference game.
- Do not use reference faction/race/class names, spell names, quest names, zone names, or NPC names.
- Do not copy specific quest flows, enemy rosters, talent trees, or itemization tables.

## Systems Direction

Prioritize systems that make the existing slice repeatable and debuggable:

- Server-authoritative state for combat, inventory, quests, currency, social, and sessions.
- Client prediction only for harmless presentation, never authoritative outcomes.
- Deterministic validation for movement, action use, quest credit, loot, and vendor transactions.
- Structured logs for major player-visible state transitions.
- Repeatable tests for concurrency and reconnect behavior before adding breadth.
- Data-driven definitions for actions, NPC services, quests, vendors, recipes, and map markers.

## Recommended First Implementation Slice

Ship a focused "Core MMORPG Feel Pass" before new content.

Implement in this order:

1. Add a compact target frame and nameplate state pass using existing world payload fields.
2. Add action button cooldown, disabled, out-of-range, and missing-resource feedback using server-confirmed action state where available.
3. Add a small combat log panel or event stream view fed by existing combat responses.
4. Normalize invalid-action error messages across movement, target, combat, vendor, profession, and quest endpoints.
5. Add tests that assert duplicate sessions, reconnects, and concurrent combat do not leave stale target/action state.
6. Add a manual validation checklist for solo play and two-client play.

This slice improves the current game feel while staying inside existing systems and preserving the Milestone 15 stability focus.

## Near-Term Backlog

- Target frame and nameplate readability.
- Action bar cooldown and range/resource feedback.
- Combat log schema and UI.
- Server-side global cooldown enforcement tests.
- Auto-attack start/stop/reconnect tests.
- Quest tracker persistence and map marker consistency tests.
- Vendor/trainer/profession shared interaction panel cleanup.
- Party frame technical design.
- Shared quest credit rules design.
- Dungeon-ready architecture notes, without dungeon content.

## Risks

- The UI can become derivative if visual styling follows the reference too closely. Keep layouts original and functional.
- Combat changes can destabilize solo play if server validation is broadened without tests.
- Adding class depth before action feedback is readable will hide bugs behind balance problems.
- Party features can create persistence and session edge cases before reconnect handling is mature.
- More panels can clutter the screen unless HUD density is managed deliberately.
- Performance findings from Milestone 15 should guide any high-frequency UI polling or combat event stream work.

## Approval Needed Before Implementation

Before implementing this roadmap, confirm the first slice and strict scope:

- First slice: Core MMORPG Feel Pass.
- No new races, classes, dungeons, raids, PvP, guilds, auction house, mail, matchmaking, or large content expansion.
- Keep all reference-game use clean-room and systems-level only.
- Preserve local solo play and multi-client soak-test workflows.
