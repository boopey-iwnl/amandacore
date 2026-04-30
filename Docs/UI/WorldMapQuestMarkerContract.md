# World Map And Quest Marker Contract

The World Map is a built-in map shell backed by repo-local AmandaCore map art plus server-authored zone and marker data.

## Data Source

- `zoneMap`: zone ID, display name, bounds, roads, and landmarks.
- `navigationAreas`: objective and route areas, quest IDs, target entity or mob hints.
- `mapMarkers`: NPC services, available quests, turn-in points, tracked objectives, bind points, travel points, and other server-provided markers.
- Player marker uses the authoritative world session position.
- `Content/Art/Manifests/MapArtManifest.json`: repo-relative map image paths, dimensions, source filenames, SHA-256 hashes, and calibration accuracy.
- `Content/GameData/Maps/dawnwake_map_calibration.json`: map/world bounds, image pixel bounds, north/up convention, and v1 Stonewake anchor calibration.

## Behavior

- The World tab displays `Content/Art/UI/Maps/World/dawnwake_isles_world.png`.
- The Zone tab displays `Content/Art/UI/Maps/Zones/stonewake_vale.png` for Stonewake Vale.
- The Reference Maps tab can display other copied Dawnwake zone art as display references until those runtime zones are calibrated.
- Calibrated map views display current zone name, authored roads, landmarks, navigation areas, player marker, and server-provided markers.
- Quest marker clicks select the related quest and open the Quest Log. Track calls are sent only when the quest state supports tracking.
- Entity markers may target the related NPC where the payload provides an entity ID.
- Missing image or missing calibration must fall back visibly without crashing.
- Marker precision is `calibrated_v1` for Stonewake Vale and the Stonewake inset on the Dawnwake world map. Display-reference maps do not claim runtime marker precision.

## Marker Style

- Markers are procedural first-party shapes and colors.
- Quest available, quest objective, quest turn-in, trainer, vendor, travel, bind, and player states must be visually distinct.
- Do not copy external MMO marker assets, fonts, UI code, or non-approved map art.
- Do not reference local source-asset folders from runtime content or packages.
- The local source map folder is source-only. Runtime code, manifests, package scripts, release artifacts, and docs-as-config must reference only repo-local paths.

No addon integration is allowed.
