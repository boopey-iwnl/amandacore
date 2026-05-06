# Objective Tracker Contract

The Objective Tracker is the compact right-side first-party quest summary below the navigator/minimap.

## Data Source

- The tracker reads `quests`, `quest`, and `trackedQuestIds` from the world session payload.
- It prefers quests with `tracked=true`.
- If no tracked quest exists, it may show a compact empty state or a current accepted quest fallback when the payload supports it.

## Behavior

- Display quest title, state, objective progress, objective graph nodes where available, and server-authored route or area hints.
- Show ready-to-turn-in/completed-objective state distinctly.
- Click on a quest row may select that quest and open the Quest Log.
- Collapse/expand is local UI state only.
- Do not show fake distance, waypoint, or marker data. Use only server-provided objective areas, route hints, map markers, or known live positions.

## Layout

- Default position is under the navigator and left of optional right action bars.
- The tracker participates in built-in UI edit-mode anchoring.
- It must not block movement/camera input after collapse or close and must not overlap bags/HUD severely at default layout sizes.

No addon integration is allowed.
