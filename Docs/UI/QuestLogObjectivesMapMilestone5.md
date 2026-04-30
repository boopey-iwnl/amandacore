# UI Milestone 5 - Quest Log, Objectives, Gossip, and Map

Milestone 5 polishes AmandaCore's first-party questing shell without adding addon support or rewriting the backend quest system. The implementation uses the existing server-authoritative world quest routes and the current world session payload.

## Current Audit

- Quest accept, progress, complete, reward, and tracking are owned by the Go world service through `POST /v1/world/quest/accept`, `POST /v1/world/quest/complete`, and `POST /v1/world/quest/track`.
- The world session response already includes `quest`, `quests`, `trackedQuestIds`, `zoneMap`, `navigationAreas`, and `mapMarkers`.
- Quest summaries already carry title, category, status bucket, objective type/text, objective graph, progress counters, giver and turn-in NPC IDs, level band, reward XP, reward currency, reward items, party hints, objective area, and tracked state.
- The O3DE UiClient already had Quest Log, Objective Tracker, quest gossip, minimap, and Zone Map surfaces. M5 upgrades those surfaces instead of introducing a parallel UI framework.
- NPC quest markers are generated server-side from friendly NPC services and current quest progress. Tracked objective markers come from quest navigation areas or authored quest marker coordinates.
- Dialogue catalog content is loaded and validated by the content package boundary, but the current quest gossip panel still primarily presents quest state rather than a full branching dialogue tree.

## Implemented Behavior

- Quest Log uses a two-pane first-party panel: bucketed quest list on the left, selected quest details on the right.
- Quest details show state, category, level band when present, group/party hints, objective counters, objective graph nodes, objective area hints, reward currency, and reward items.
- Objective Tracker shows tracked/current quests compactly below the navigator, highlights ready-to-turn-in state, supports collapse/expand, and can open the selected quest in the Quest Log.
- Quest gossip selects the relevant quest for the targeted NPC, shows available/active/ready states, previews real rewards, accepts through the authoritative accept route, and completes/turns in through the authoritative complete route.
- Zone Map clicks on quest markers or quest areas select the quest and open the Quest Log. Track requests are sent only for supported quest states.
- The authored Dawnwake follow-up adds repo-local map art for the Dawnwake world map and copied zone maps. Stonewake Vale is calibrated for v1 runtime overlays; other zone art is display-reference until those zones are authored.
- Map and minimap markers remain procedural first-party shapes and colors. No external marker art or copied MMO UI assets are introduced.
- The playable Stonewake Vale layout now uses the same v1 anchors as the map overlay: Hearthwatch Yard, ValeFurrow Farms, Brookside Crossing, Stonehewn Quarry, Tiderown Ruins, Lightkeeper's Point, Whispering Cave, and the main road loop.

## Limitations And Follow-Up

- Full branching dialogue UI remains a later content-driven dialogue milestone. M5 documents the contract and keeps quest gossip compatible with current content.
- Quest abandon is not exposed because there is no current authoritative abandon endpoint in the world service.
- Overhead 3D quest markers are limited to existing first-party friendly NPC nameplate roles and map/minimap indicators.
- Stonewake calibration is a first pass. Linear bounds-based placement is suitable for v1 map overlays and manual navigation, but later terrain authoring should replace proxy geometry with authored meshes/heightfields.
- Non-Stonewake zone PNGs are copied and manifest-tracked but remain display-only until their runtime zone bounds, anchors, and service/quest data are calibrated.

## Safety Notes

- No addon API, Lua loader, AddOns folder, user-installed UI module, plugin runtime, or arbitrary UI script execution is added.
- The local texture source folder remains a read-only source pool only. The authored map follow-up copies curated PNGs into `Content/Art/UI/Maps/**` and must not create runtime, package, manifest, material, code, or docs-as-config dependencies on that machine-local path.
- Quest text and map labels come from AmandaCore repo content only.
