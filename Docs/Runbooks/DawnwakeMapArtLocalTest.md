# Dawnwake Map Art Local Test

Use this runbook to verify the authored Dawnwake map art follow-up after automated validation passes.

## Setup

1. Check out `codex/ui-m5-authored-dawnwake-maps`.
2. Confirm `git status --short` is clean before manual smoke.
3. Start the local stack with `Infra\dev\start-local.ps1 -StartLauncher`.
4. Log in, select a realm and character, and join Stonewake Vale.

## Map UI Smoke

- Open the World Map.
- Confirm the World tab displays `Content/Art/UI/Maps/World/dawnwake_isles_world.png`.
- Confirm the Zone tab displays `Content/Art/UI/Maps/Zones/stonewake_vale.png`.
- Confirm both authored maps are legible in the in-game window. Use Fit to reset the view, mouse-wheel or the Zoom control to inspect the art, and right-drag pan to inspect off-center labels and markers.
- Confirm the Reference Maps tab can display the other copied Dawnwake zone maps as display references.
- Confirm player, quest, tracked quest, service, trainer, vendor, travel, bind, and landmark markers overlay the calibrated World and Stonewake views plausibly.
- Confirm marker labels and circles remain aligned while zoomed and panned.
- Click calibrated quest markers and quest areas, then confirm supported Quest Log selection/tracking behavior still works.
- Confirm missing art or uncalibrated reference maps show a visible fallback or display-only state without a crash.

## Playable Stonewake Smoke

- Start at Hearthwatch Yard and confirm the yard, trainer, vendor, bind, and route service positions cluster around the same map anchor.
- Walk the main road loop through ValeFurrow Farms, Brookside Crossing, Stonehewn Quarry, Lightkeeper's Point, Whispering Cave, and Tiderown Ruins.
- Confirm terrain bands, roads, farm fields, quarry stones, lantern/point presentation, cave-side hostile pocket, ruins pocket, and visible bounds match the authored map direction closely enough for v1.
- Confirm quest hubs, NPC positions, mob pockets, gathering nodes, travel/bind markers, and objective tracker hints correspond to the map overlay.
- Confirm Armsmaster Corin Vale, Scout Rowan Bell, Roadwarden Ilya Brant, the Watch Lantern, and Quartermaster Lyra are not embedded in visual building blockouts or props and can be right-clicked from reachable ground.
- Verify movement, camera, HUD, action bars, inventory, chat, Quest Log, Objective Tracker, and Map still work.

## Safety Checks

- Run repo scans for the local source texture folder pattern and the absolute local source texture path.
- Confirm no runtime, manifest, package script, release artifact, generated runtime config, or docs-as-config file references the local source folder.
- Confirm copied map files live under `Content/Art/UI/Maps/**`.
- Confirm no AddOns folder, Lua addon loader, plugin runtime, user-installed UI module, or arbitrary UI script execution was added.

## Result

Record one of:

- `READY FOR PR`
- `NEEDS FIXES`
- `NEEDS HUMAN REVIEW`
