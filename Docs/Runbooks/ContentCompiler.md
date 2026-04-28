# Content Compiler Runbook

## Purpose

Use the content compiler to validate AmandaCore content packages before runtime activation. The compiler is deterministic and clean-room; it does not load external MMO schemas or data.

## Validate The Default Dev Package

```powershell
Push-Location Services
go run ./cmd/content-compiler --package ..\Content\Packs\dev_foundation\package.json --check
Pop-Location
```

Expected result:

```text
content-compiler check passed: dev_foundation 0.1.0 (<sha256>)
```

## Emit A Compiled Report

Use a temporary output path. Do not commit generated local output unless a future milestone makes compiled artifacts part of the repo contract.

```powershell
Push-Location Services
go run ./cmd/content-compiler --package ..\Content\Packs\dev_foundation\package.json --out $env:TEMP\dev_foundation.compiled.json
Pop-Location
```

## Check A Committed Or Generated Output

```powershell
Push-Location Services
go run ./cmd/content-compiler --package ..\Content\Packs\dev_foundation\package.json --out $env:TEMP\dev_foundation.compiled.json --check
Pop-Location
```

## Add A Quest

1. Add a quest entry to a quest catalog listed by `quest_catalogs`.
2. Use an original AmandaCore `quest_id`.
3. Reference NPC, item, or quest-provider IDs that exist in the same package.
4. Run the compiler in `--check` mode.

## Add An NPC

1. Add the NPC archetype to an NPC catalog listed by `npc_catalogs`.
2. Reference only package abilities that exist in `ability_catalogs`.
3. Reference the NPC from a zone spawn group.
4. Run the compiler.

## Add A Loot Table

1. Add the table to a loot catalog listed by `loot_catalogs`.
2. Reference only item IDs from `item_catalogs`.
3. Reference the loot table from a spawn group or hook source as needed.
4. Run the compiler.

## Add A Trainer Or Vendor

1. Add a vendor catalog path to `vendor_catalogs`, or a trainer catalog path to `trainer_catalogs`.
2. Vendors must reference package item IDs.
3. Trainers must reference package ability IDs.
4. Use package-local NPC or provider IDs for `npc_id` when available.
5. Run the compiler.

## Add A Hook Binding

1. Add a hook catalog path to `hook_catalogs`.
2. Use one of the supported hook names documented in `Docs/Architecture/ContentCompilerRuntimeBoundary.md`.
3. Use declarative actions only.
4. Run the compiler. Invalid hook names fail before runtime.

## Troubleshooting

- `MissingFile`: a manifest path points to a file that cannot be found.
- `MalformedJson`: JSON could not be decoded.
- `DuplicateID`: two entries in the same package use the same stable ID.
- `BrokenReference`: a content entry references an ID not loaded in the package.
- `InvalidEnum`: a hook, objective kind, ability effect, item kind, or other enum is unsupported.
- `ObjectiveGraphCycle`: a quest objective graph has a dependency cycle.

## Clean-Room Rules

Do not add copied external MMO quest text, NPC names, item IDs, spell IDs, scripts, SQL schema names, packet names, comments, module structure, or content data. Use AmandaCore-original IDs, names, text, formulas, and schemas.
