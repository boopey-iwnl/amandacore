# World Map And Quest Marker Contract

The World Map is a built-in schematic map shell backed by server-authored zone and marker data.

## Data Source

- `zoneMap`: zone ID, display name, bounds, roads, and landmarks.
- `navigationAreas`: objective and route areas, quest IDs, target entity or mob hints.
- `mapMarkers`: NPC services, available quests, turn-in points, tracked objectives, bind points, travel points, and other server-provided markers.
- Player marker uses the authoritative world session position.

## Behavior

- The map displays current zone name, authored roads, landmarks, navigation areas, player marker, and server-provided markers.
- Quest marker clicks select the related quest and open the Quest Log. Track calls are sent only when the quest state supports tracking.
- Entity markers may target the related NPC where the payload provides an entity ID.
- Placeholder schematic art is acceptable until AmandaCore-owned map art is integrated.

## Marker Style

- Markers are procedural first-party shapes and colors.
- Quest available, quest objective, quest turn-in, trainer, vendor, travel, bind, and player states must be visually distinct.
- Do not copy external MMO marker assets, map art, fonts, or UI code.
- Do not reference local source-asset folders from runtime content or packages.

No addon integration is allowed.
