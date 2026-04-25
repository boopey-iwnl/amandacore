#include "AmandaCoreShared/ContentValidation.h"
#include "AmandaCoreShared/GameplayRules.h"

#include <cstdlib>
#include <exception>
#include <iostream>
#include <set>
#include <stdexcept>
#include <string>

namespace
{
    void Expect(const bool condition, const std::string& message)
    {
        if (!condition)
        {
            throw std::runtime_error(message);
        }
    }

    void TestMovementStep()
    {
        amandacore::MovementState state {};
        amandacore::MovementInput input {};
        input.forwardAxis = 1.0F;
        input.wantsToRun = true;
        input.facingRadians = 0.0F;
        input.jumpPressed = true;

        const amandacore::MovementConfig config {};
        const auto next = amandacore::SimulateMovement(state, input, config, 0.25F);

        Expect(next.position.x > 1.0F, "Forward movement should advance on the X axis.");
        Expect(next.position.z > 0.0F, "Jumping should move the player upward.");
        Expect(!next.grounded, "Jumping should unground the player.");
    }

    void TestCombatMitigationAndCrit()
    {
        amandacore::CombatSnapshot snapshot {};
        snapshot.attackerLevel = 10;
        snapshot.defenderLevel = 10;
        snapshot.attack.weaponMinDamage = 10.0F;
        snapshot.attack.weaponMaxDamage = 20.0F;
        snapshot.attack.attackPower = 42.0F;
        snapshot.attack.critChance = 0.25F;
        snapshot.defender.values[amandacore::StatId::Armor] = 150.0F;

        const auto critEvent = amandacore::ComputeMeleeSwing(snapshot, 1.0F, 0.1F);
        Expect(critEvent.outcome == amandacore::CombatOutcome::Crit, "Outcome roll below crit chance should crit.");
        Expect(critEvent.mitigatedDamage < critEvent.rawDamage, "Armor should mitigate physical damage.");
    }

    void TestObjectiveProgress()
    {
        amandacore::ObjectiveDefinition objective {};
        objective.id = "road_clear";
        objective.type = amandacore::ObjectiveType::Kill;
        objective.targetId = "fen_raider";
        objective.requiredCount = 3;

        const auto firstUpdate = amandacore::ApplyObjectiveEvent(objective, 0, amandacore::ObjectiveType::Kill, "fen_raider", 2);
        Expect(firstUpdate.currentCount == 2, "Objective should increment by the event amount.");
        Expect(!firstUpdate.completed, "Objective should not complete before the target count is reached.");

        const auto finalUpdate = amandacore::ApplyObjectiveEvent(objective, firstUpdate.currentCount, amandacore::ObjectiveType::Kill, "fen_raider", 2);
        Expect(finalUpdate.currentCount == 3, "Objective progress should clamp to the required count.");
        Expect(finalUpdate.completed, "Objective should complete at the required count.");
    }

    void TestThreatEvaluation()
    {
        amandacore::ThreatProfile profile {};
        profile.acquisitionRangeMeters = 18.0F;
        profile.assistRadiusMeters = 8.0F;
        profile.leashRangeMeters = 30.0F;

        Expect(
            amandacore::EvaluateThreatState(profile, 12.0F, 10.0F, true, false) == amandacore::AiState::Engaged,
            "Visible targets in acquisition range should engage.");
        Expect(
            amandacore::EvaluateThreatState(profile, 40.0F, 31.0F, true, false) == amandacore::AiState::Returning,
            "Targets that pull beyond the leash range should force return.");
    }

    void TestContentValidation()
    {
        amandacore::QuestDefinition quest {};
        quest.id = "clear_the_road";
        quest.title = "Clear the Road";
        quest.objectives.push_back(amandacore::ObjectiveDefinition {
            "kill_raiders",
            amandacore::ObjectiveType::Kill,
            "fen_raider",
            4
        });

        const auto questIssues = amandacore::ValidateQuestDefinition(quest);
        Expect(questIssues.empty(), "Well-formed quests should validate cleanly.");

        amandacore::ZoneManifest zone {};
        zone.id = "sunset_frontier";
        zone.displayName = "Sunset Frontier";
        zone.cellIds = { "west_approach", "ferry_crossing" };
        zone.hubCellId = "west_approach";
        zone.microInstanceCellId = "ferry_crossing";
        zone.vendorNpcIds = { "quartermaster_lyra" };

        const auto zoneIssues = amandacore::ValidateZoneManifest(
            zone,
            std::set<std::string> { "west_approach", "ferry_crossing" },
            std::set<std::string> { "quartermaster_lyra" });
        Expect(zoneIssues.empty(), "Known cells and vendors should validate cleanly.");
    }
}

int main()
{
    try
    {
        TestMovementStep();
        TestCombatMitigationAndCrit();
        TestObjectiveProgress();
        TestThreatEvaluation();
        TestContentValidation();
    }
    catch (const std::exception& exception)
    {
        std::cerr << "Test failure: " << exception.what() << '\n';
        return EXIT_FAILURE;
    }

    std::cout << "AmandaCoreSharedTests passed\n";
    return EXIT_SUCCESS;
}
