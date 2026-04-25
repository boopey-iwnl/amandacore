#pragma once

#include "AmandaCoreShared/Contracts.h"
#include "AmandaCoreShared/Messages.h"

#include <map>
#include <string>
#include <utility>
#include <vector>

namespace amandacore
{
    struct MovementConfig
    {
        float walkSpeedMetersPerSecond = 3.5F;
        float runSpeedMetersPerSecond = 7.0F;
        float backpedalMultiplier = 0.7F;
        float strafeMultiplier = 1.0F;
        float jumpVelocityMetersPerSecond = 7.0F;
        float gravityMetersPerSecondSquared = -19.6F;
    };

    struct MovementState
    {
        Vec3 position;
        Vec3 velocity;
        bool grounded = true;
        MovementMode mode = MovementMode::Ground;
        float facingRadians = 0.0F;
    };

    struct MovementInput
    {
        float forwardAxis = 0.0F;
        float strafeAxis = 0.0F;
        bool wantsToRun = true;
        bool jumpPressed = false;
        float facingRadians = 0.0F;
    };

    struct StatBlock
    {
        std::map<StatId, float> values;

        [[nodiscard]] float Get(StatId stat, float fallback = 0.0F) const;
        void Add(StatId stat, float amount);
    };

    struct AttackProfile
    {
        float weaponMinDamage = 1.0F;
        float weaponMaxDamage = 2.0F;
        float attackPower = 0.0F;
        float critChance = 0.0F;
        DamageSchool school = DamageSchool::Physical;
    };

    struct CombatSnapshot
    {
        StatBlock attacker;
        StatBlock defender;
        AttackProfile attack;
        std::uint32_t attackerLevel = 1;
        std::uint32_t defenderLevel = 1;
        float swingSeconds = 2.0F;
    };

    struct CombatEvent
    {
        CombatOutcome outcome = CombatOutcome::Hit;
        float rawDamage = 0.0F;
        float mitigatedDamage = 0.0F;
        bool glancingBlow = false;
    };

    struct ObjectiveProgress
    {
        std::string objectiveId;
        std::uint32_t currentCount = 0;
        bool completed = false;
    };

    [[nodiscard]] MovementState SimulateMovement(
        const MovementState& current,
        const MovementInput& input,
        const MovementConfig& config,
        float deltaSeconds);

    [[nodiscard]] CombatEvent ComputeMeleeSwing(
        const CombatSnapshot& snapshot,
        float damageRoll01,
        float outcomeRoll01);

    [[nodiscard]] std::vector<std::pair<std::string, std::uint32_t>> RollLoot(
        const LootTable& table,
        const std::vector<float>& randoms);

    [[nodiscard]] ObjectiveProgress ApplyObjectiveEvent(
        const ObjectiveDefinition& objective,
        std::uint32_t currentCount,
        ObjectiveType eventType,
        const std::string& targetId,
        std::uint32_t increment);

    [[nodiscard]] AiState EvaluateThreatState(
        const ThreatProfile& profile,
        float distanceToTargetMeters,
        float distanceFromAnchorMeters,
        bool targetVisible,
        bool hasDamageTaken);
}
