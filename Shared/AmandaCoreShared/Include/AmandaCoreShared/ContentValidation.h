#pragma once

#include "AmandaCoreShared/Contracts.h"

#include <set>
#include <string>
#include <vector>

namespace amandacore
{
    struct ValidationIssue
    {
        std::string path;
        std::string message;
    };

    [[nodiscard]] std::vector<ValidationIssue> ValidateQuestDefinition(const QuestDefinition& quest);
    [[nodiscard]] std::vector<ValidationIssue> ValidateLootTable(const LootTable& lootTable);
    [[nodiscard]] std::vector<ValidationIssue> ValidateSpawnGroup(
        const SpawnGroup& spawnGroup,
        const std::set<std::string>& knownUnits);
    [[nodiscard]] std::vector<ValidationIssue> ValidateZoneManifest(
        const ZoneManifest& zone,
        const std::set<std::string>& knownCells,
        const std::set<std::string>& knownVendors);
}
