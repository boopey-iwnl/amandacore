#include "AmandaCoreShared/ContentValidation.h"

namespace amandacore
{
    namespace
    {
        void AppendIssue(
            std::vector<ValidationIssue>& issues,
            const std::string& path,
            const std::string& message)
        {
            issues.push_back(ValidationIssue { path, message });
        }
    }

    std::vector<ValidationIssue> ValidateQuestDefinition(const QuestDefinition& quest)
    {
        std::vector<ValidationIssue> issues;

        if (quest.id.empty())
        {
            AppendIssue(issues, "QuestDefinition.id", "Quest id must not be empty.");
        }

        if (quest.title.empty())
        {
            AppendIssue(issues, "QuestDefinition.title", "Quest title must not be empty.");
        }

        if (quest.objectives.empty())
        {
            AppendIssue(issues, "QuestDefinition.objectives", "Quest must define at least one objective.");
        }

        for (std::size_t index = 0; index < quest.objectives.size(); ++index)
        {
            const auto& objective = quest.objectives[index];
            if (objective.id.empty())
            {
                AppendIssue(issues, "QuestDefinition.objectives[" + std::to_string(index) + "].id", "Objective id must not be empty.");
            }

            if (objective.targetId.empty())
            {
                AppendIssue(issues, "QuestDefinition.objectives[" + std::to_string(index) + "].targetId", "Objective targetId must not be empty.");
            }

            if (objective.requiredCount == 0)
            {
                AppendIssue(issues, "QuestDefinition.objectives[" + std::to_string(index) + "].requiredCount", "Objective requiredCount must be greater than zero.");
            }
        }

        return issues;
    }

    std::vector<ValidationIssue> ValidateLootTable(const LootTable& lootTable)
    {
        std::vector<ValidationIssue> issues;
        if (lootTable.id.empty())
        {
            AppendIssue(issues, "LootTable.id", "Loot table id must not be empty.");
        }

        for (std::size_t index = 0; index < lootTable.entries.size(); ++index)
        {
            const auto& entry = lootTable.entries[index];
            const std::string prefix = "LootTable.entries[" + std::to_string(index) + "]";
            if (entry.itemId.empty())
            {
                AppendIssue(issues, prefix + ".itemId", "Loot entry itemId must not be empty.");
            }

            if (entry.dropChance < 0.0F || entry.dropChance > 1.0F)
            {
                AppendIssue(issues, prefix + ".dropChance", "Loot entry dropChance must be within [0, 1].");
            }

            if (entry.maxCount < entry.minCount)
            {
                AppendIssue(issues, prefix + ".maxCount", "Loot entry maxCount must be greater than or equal to minCount.");
            }
        }

        return issues;
    }

    std::vector<ValidationIssue> ValidateSpawnGroup(
        const SpawnGroup& spawnGroup,
        const std::set<std::string>& knownUnits)
    {
        std::vector<ValidationIssue> issues;
        if (spawnGroup.id.empty())
        {
            AppendIssue(issues, "SpawnGroup.id", "Spawn group id must not be empty.");
        }

        if (spawnGroup.spawns.empty())
        {
            AppendIssue(issues, "SpawnGroup.spawns", "Spawn group must contain at least one spawn.");
        }

        for (std::size_t index = 0; index < spawnGroup.spawns.size(); ++index)
        {
            const auto& spawn = spawnGroup.spawns[index];
            if (!knownUnits.empty() && !knownUnits.contains(spawn.unitId))
            {
                AppendIssue(
                    issues,
                    "SpawnGroup.spawns[" + std::to_string(index) + "].unitId",
                    "Spawn references an unknown unit id.");
            }
        }

        return issues;
    }

    std::vector<ValidationIssue> ValidateZoneManifest(
        const ZoneManifest& zone,
        const std::set<std::string>& knownCells,
        const std::set<std::string>& knownVendors)
    {
        std::vector<ValidationIssue> issues;

        if (zone.id.empty())
        {
            AppendIssue(issues, "ZoneManifest.id", "Zone id must not be empty.");
        }

        if (zone.cellIds.empty())
        {
            AppendIssue(issues, "ZoneManifest.cellIds", "Zone must contain at least one world cell.");
        }

        if (!zone.hubCellId.empty() && !knownCells.empty() && !knownCells.contains(zone.hubCellId))
        {
            AppendIssue(issues, "ZoneManifest.hubCellId", "Hub cell id is not present in the known cell set.");
        }

        if (!zone.microInstanceCellId.empty() && !knownCells.empty() && !knownCells.contains(zone.microInstanceCellId))
        {
            AppendIssue(issues, "ZoneManifest.microInstanceCellId", "Micro-instance cell id is not present in the known cell set.");
        }

        for (std::size_t index = 0; index < zone.vendorNpcIds.size(); ++index)
        {
            const auto& vendorId = zone.vendorNpcIds[index];
            if (!knownVendors.empty() && !knownVendors.contains(vendorId))
            {
                AppendIssue(
                    issues,
                    "ZoneManifest.vendorNpcIds[" + std::to_string(index) + "]",
                    "Zone vendor id is not present in the known vendor set.");
            }
        }

        return issues;
    }
}
