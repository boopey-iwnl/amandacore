#pragma once

#include "AmandaCoreShared/CoreTypes.h"

#include <array>
#include <map>
#include <string>
#include <vector>

namespace amandacore
{
    struct StatCurvePoint
    {
        std::uint32_t level = 1;
        float value = 0.0F;
    };

    struct StatCurve
    {
        std::string id;
        StatId stat = StatId::Health;
        std::vector<StatCurvePoint> points;
    };

    struct CharacterArchetype
    {
        std::string id;
        std::string displayName;
        std::string factionTag;
        std::uint32_t startingLevel = 1;
        std::vector<std::string> starterSpellIds;
        std::map<StatId, float> baseStats;
    };

    struct EffectDefinition
    {
        std::string id;
        std::string effectType;
        float magnitude = 0.0F;
        float durationSeconds = 0.0F;
        DamageSchool school = DamageSchool::Physical;
    };

    struct SpellDefinition
    {
        std::string id;
        std::string displayName;
        float castTimeSeconds = 0.0F;
        float cooldownSeconds = 0.0F;
        float rangeMeters = 0.0F;
        StatId resourceStat = StatId::Resource;
        std::int32_t resourceCost = 0;
        std::vector<std::string> effectIds;
    };

    struct ItemDefinition
    {
        std::string id;
        std::string displayName;
        std::string slot;
        std::uint32_t itemLevel = 1;
        std::map<StatId, float> statModifiers;
        std::uint32_t sellValue = 0;
    };

    struct LootEntry
    {
        std::string itemId;
        float dropChance = 0.0F;
        std::uint32_t minCount = 1;
        std::uint32_t maxCount = 1;
    };

    struct LootTable
    {
        std::string id;
        std::vector<LootEntry> entries;
    };

    struct ObjectiveDefinition
    {
        std::string id;
        ObjectiveType type = ObjectiveType::Kill;
        std::string targetId;
        std::uint32_t requiredCount = 1;
    };

    struct QuestDefinition
    {
        std::string id;
        std::string title;
        std::uint32_t minLevel = 1;
        std::vector<std::string> prerequisiteQuestIds;
        std::vector<ObjectiveDefinition> objectives;
        std::vector<std::string> rewardItemIds;
        std::uint32_t rewardCurrency = 0;
    };

    struct ThreatProfile
    {
        float acquisitionRangeMeters = 15.0F;
        float leashRangeMeters = 35.0F;
        float assistRadiusMeters = 10.0F;
    };

    struct NpcBrain
    {
        std::string id;
        ThreatProfile threat;
        float patrolRadiusMeters = 0.0F;
        float respawnSeconds = 20.0F;
        std::vector<std::string> abilityIds;
    };

    struct UnitDefinition
    {
        std::string id;
        std::string displayName;
        std::uint32_t level = 1;
        std::map<StatId, float> baseStats;
        float moveSpeedMetersPerSecond = 4.0F;
        float meleeSwingSeconds = 2.0F;
        float collisionRadiusMeters = 0.5F;
        std::string brainId;
        std::string lootTableId;
        std::vector<std::string> spellIds;
    };

    struct SpawnPoint
    {
        EntityId localId = 0;
        std::string unitId;
        std::array<float, 3> position {};
        float facingRadians = 0.0F;
        std::uint32_t respawnVarianceSeconds = 0;
    };

    struct SpawnGroup
    {
        std::string id;
        std::vector<SpawnPoint> spawns;
    };

    struct EncounterDefinition
    {
        std::string id;
        std::string displayName;
        std::vector<std::string> spawnGroupIds;
        std::string successQuestId;
        bool createsMicroInstanceHook = false;
    };

    struct WorldCell
    {
        std::string id;
        std::vector<std::string> spawnGroupIds;
        std::vector<std::string> questIds;
        std::vector<std::string> encounterIds;
        std::array<float, 2> minBounds {};
        std::array<float, 2> maxBounds {};
    };

    struct ZoneManifest
    {
        std::string id;
        std::string displayName;
        std::vector<std::string> cellIds;
        std::string hubCellId;
        std::string microInstanceCellId;
        std::vector<std::string> vendorNpcIds;
    };
}
