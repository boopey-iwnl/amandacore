# AmandaCore Art Asset Pipeline

AmandaCore source art assets live under `Content/Art`. Generated O3DE cache output must stay out of source control.

## Source Folders

- `Content/Art/Textures/Terrain` - grass, dirt, road, shore, farm, and ground cover textures.
- `Content/Art/Textures/Rock` - cliff, quarry, cave, and ruin stone textures.
- `Content/Art/Textures/Wood` - planks, dock boards, trim, and structural wood textures.
- `Content/Art/Textures/Stone` - cut stone and masonry textures.
- `Content/Art/Textures/Water` - water and foam placeholders.
- `Content/Art/Textures/Foliage` - bark, leaves, grass cards, shrubs, moss, and crop placeholders.
- `Content/Art/Textures/Architecture` - plaster, roofing, shingles, cobbles, and building surfaces.
- `Content/Art/Textures/Props` - barrels, crates, sacks, signs, and other prop surfaces.
- `Content/Art/Textures/FX` - lava, rune, glow, foam, and other effect textures.
- `Content/Art/Icons` - UI-ready ability, item, inventory, currency, and menu icons.
- `Content/Art/Materials` - O3DE source `.material` files referencing imported texture assets.
- `Content/Art/Manifests` - release-scoped asset manifests and provenance metadata.

## Import Rules

Use a curated import for each release. Do not commit the full source download folder, duplicates, screenshots, reference maps, generated cache output, logs, release zips, or unrelated downloads.

Imported filenames must be lowercase snake_case and should describe project usage rather than source filenames. Manifest entries record the source file as a relative path plus SHA-256 hash; absolute local download paths are intentionally omitted.

Terrain and building textures should be power-of-two PNGs for O3DE processing. Release 0.2.0 uses 1024x1024 runtime source PNGs for broad material coverage. UI icons should be 128x128 PNGs with `Content/Art/Icons/UI/icon_missing.png` as the deliberate fallback.

## Runtime Wiring

O3DE source material files in `Content/Art/Materials` use `StandardPBR.materialtype` and reference imported texture PNGs. The current playable O3DE scene still uses procedural AuxGeom presentation for the Stonewake validation arena, so Release 0.2.0 also maps material IDs to visible terrain, road, building, foliage, landmark, and water proxy surfaces in the client.

Gameplay responses expose icon IDs through `iconKind`. The client maps those IDs to visible action bar, spellbook, inventory, and fallback icon presentation. Missing or unknown IDs must resolve to `icon_missing` behavior, never a blank slot.
