# Dawnwake Isles Zone Skeletons

## Clean-room reference boundary

This implementation uses original AmandaCore code and data. TrinityCore/AzerothCore were used only as high-level architectural reference. No source code, SQL, packet layouts, opcodes, command names, schemas, content IDs, scripts, scripting APIs, assets, formulas, loot tables, quest tables, item IDs, creature IDs, spell IDs, quest text, reward tables, map formats, zone tables, spawn schemas, coordinates, or database structures were copied or adapted.

## Package

`Content/Packs/dawnwake_isles/package.json` is the first AmandaCore-owned multi-zone runtime package. It contains three placeholder zones:

- `dawnwake_landing`
- `dawnwake_tideglass_shoal`
- `dawnwake_windspur_rise`

Each zone has original bounds, entry points, spawn groups, quest providers, runtime caps, and transition metadata. The package is intentionally small; it validates the runtime loader and traversal boundary before an O3DE terrain or asset pipeline exists.

## Zone Transitions

Transitions are server-side adjacency records:

- source zone owns the transition position and radius
- target zone must exist in the same validated package
- destination entry point must exist in the target zone
- runtime movement places the session at the destination entry point

Current transition coverage:

- `dawnwake_landing.to_tideglass_shoal`
- `dawnwake_tideglass_shoal.to_landing`
- `dawnwake_tideglass_shoal.to_windspur_rise`
- `dawnwake_windspur_rise.to_tideglass_shoal`

These are future streaming hooks, not terrain streams. They prove authoritative traversal state and validation before client streaming assets are introduced.

## Runtime Activation

When the package is loaded, the world runtime:

- creates three `ZoneRuntime` records
- registers four transition points
- registers three quest providers
- spawns ten placeholder NPCs from loaded spawn groups
- adds package items to the item catalog
- projects supported content quests into the current quest runtime

Existing Stonewake and Brindlebrook hardcoded flows remain available for current tests and local play.

## Loadsim

Run from the Go module root:

```powershell
cd Services
go run ./cmd/loadsim --clients 1 --duration 30s --cmd-rate 2 --scenario dawnwake-traversal-basic --content ..\Content\Packs\dawnwake_isles\package.json
```

The scenario validates the package, activates all zones, enters `dawnwake_landing`, completes the first transition to `dawnwake_tideglass_shoal`, verifies spawned NPC content, resolves the `dw_tideglass_sparks` placeholder quest path, claims deterministic guaranteed loot, grants the placeholder reward, and prints a concise report.

## Current Limitations

- No O3DE map, terrain, prefab, asset, or world-partition data is loaded yet.
- Zone bounds and positions are placeholder server coordinates authored for this package only.
- Transition handling is radius-based and single-step.
- Combat and loot in the loadsim are deterministic validation summaries, not a full client session.
- Ability and aura package entries are validated and registered, but combat still uses the existing runtime ability path.

## Next Milestone

Connect Dawnwake package zones to O3DE-authored placeholder map exports and streamed-world hooks. The next step should add generated server content from AmandaCore-owned map metadata, zone adjacency derived from those exports, client-facing transition hints, and broader traversal loadsim coverage.
