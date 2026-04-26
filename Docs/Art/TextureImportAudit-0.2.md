# Texture Import Audit - Release 0.2.0

Source: user-provided local texture folder. Absolute local paths are omitted from committed manifests and docs; provenance is tracked by relative source path and SHA-256 in `Content/Art/Manifests/TextureManifest-0.2.json`.

## Summary

- Total files scanned: 328
- Total source size: 991.48 MiB
- File types: 328 PNG
- Source dimensions: 319 at 1254x1254, 8 at 1448x1086, 1 at 1055x1491
- Exact duplicate groups: 18 groups, 36 files
- Unsupported formats: none
- Suspicious filename term matches: none
- Imported image assets: 52
- Imported image asset size: 77.79 MiB
- Source material files created: 32
- Files skipped: 276

## Folder Buckets

- Terrain: `core terrain`, `Roads_paths_bridges_surfaces`, selected blend overlays
- Buildings: `Buildings_human_settlement_textures`, dock/wood surfaces
- Foliage: `Leaves and plants`, `Vegetation_textures _Trees`, grass-card overlays
- Props: selected barrels, crates, sacks, and training-object cloth/wood surfaces
- UI: selected `icons` sources for abilities, inventory, items, currency, and menu fallback
- FX: selected `Magic, VFX, and emissive`, lava, rune, foam, and water placeholders
- Reference-only: `Dawnwake_Isles_maps`

## Largest Imported Images

- `Content/Art/Textures/Foliage/mossy_grass_cover.png` - 3,149,770 bytes
- `Content/Art/Textures/Terrain/stonewake_grass_lush.png` - 3,065,292 bytes
- `Content/Art/Textures/Foliage/wheat_crop_placeholder.png` - 3,035,780 bytes
- `Content/Art/Textures/Terrain/shore_sand_pebbles.png` - 2,991,934 bytes
- `Content/Art/Textures/Architecture/hearthwatch_cobble_path.png` - 2,945,568 bytes
- `Content/Art/Textures/Terrain/rocky_ground_gray.png` - 2,858,693 bytes
- `Content/Art/Textures/Foliage/tree_bark_brown.png` - 2,776,780 bytes
- `Content/Art/Textures/Terrain/stonewake_grass_worn.png` - 2,736,928 bytes
- `Content/Art/Textures/Architecture/village_white_plaster.png` - 2,715,428 bytes
- `Content/Art/Textures/Props/sack_cloth.png` - 2,712,839 bytes

No imported file is over 20 MiB.

## Imported Coverage

Terrain coverage includes grass, worn grass, dirt path, mud, farm furrows, rocky ground, shore sand, cobble path, furrow overlay, and shore foam. Building coverage includes village wood, plaster, cut stone, thatch, shingles, dock planks, barrel wood, crate wood, and sack cloth. Foliage coverage includes bark, mossy ground cover, dense shrubs, grass cards, and wheat crop placeholders. FX coverage includes rune decals, lava cracks, and stream/shore water placeholders.

UI coverage includes visible icons for Auto Attack, Steady Strike, Brace, Driving Blow, Hampering Strike, Rallying Call, starter inventory items, currency, and a deliberate missing-icon fallback.

## Skipped Assets

Skipped files were not deleted or modified. Skipped reasons:

- Exact duplicate hashes from generated batches.
- Reference maps that are not runtime textures or UI icons for this import.
- Additional generated variants not needed for Release 0.2.0 coverage.
- Oversized full source set avoided to keep the release branch reviewable.

## Provenance Review

The audited folder names and sampled image previews indicate user-generated fantasy textures/icons. No filenames referenced World of Warcraft, Blizzard, TrinityCore, AzerothCore, screenshots, logos, minimaps, or UI captures. No obvious commercial game extraction was identified during this audit. Human review is still recommended before merge because provenance ultimately depends on the user's authority to supply these local assets.
