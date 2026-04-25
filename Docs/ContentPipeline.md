# Content Pipeline

## Current Flow

AmandaCore content is authored as project-owned JSON and loaded through the Go content package loader.

```text
authored JSON package -> ContentPackageLoader -> RuntimeContentRegistry -> ContinentRuntime / ZoneRuntime
```

The Dawnwake Isles package is the first continent package using this flow. It contains original placeholder topology and gameplay references only.

## Authoritative Data Boundary

The server owns authoritative simulation data. O3DE can later author or visualize terrain, zone markers, trigger volumes, and presentation metadata, but those assets should be compiled into AmandaCore-owned runtime packages before the server uses them.

Future tooling should add:

- map trace import for owner-supplied Dawnwake images
- O3DE marker export
- coordinate transform validation
- zone overlap validation reports
- transition gate placement reports
- spawn and quest provider bounds validation
- compiled content package artifacts with checksums

## Placeholder Versus Map-accurate Data

Current Dawnwake coordinates are placeholder rectangles. They are intentionally tagged with `accuracy: placeholder` and `source: pending_map_trace`.

Before a map-accurate milestone, replace:

- zone bounds
- entry point positions
- transition gate bounds
- spawn point positions
- quest provider positions
- streaming hint radii or regions if final geography requires it

Do not silently mix placeholder and traced coordinates. Package metadata should say which source produced the data.

## Clean-room reference boundary

This implementation uses original AmandaCore code and data.

TrinityCore and AzerothCore were used only as high-level architectural reference.

Dawnwake Isles is AmandaCore-original world content.

No source code, SQL, packet layouts, opcodes, command names, schemas, content IDs, scripts, scripting APIs, assets, formulas, map formats, area tables, zone tables, spawn schemas, coordinates, quest tables, item tables, creature tables, spell tables, aura tables, reward schemas, or database structures were copied or adapted.
