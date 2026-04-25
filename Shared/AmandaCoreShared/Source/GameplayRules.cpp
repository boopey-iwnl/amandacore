#include "AmandaCoreShared/GameplayRules.h"

#include <algorithm>
#include <cmath>

namespace
{
    float Clamp(const float value, const float minimum, const float maximum)
    {
        return std::max(minimum, std::min(maximum, value));
    }
}

namespace amandacore
{
    float StatBlock::Get(const StatId stat, const float fallback) const
    {
        const auto found = values.find(stat);
        return found == values.end() ? fallback : found->second;
    }

    void StatBlock::Add(const StatId stat, const float amount)
    {
        values[stat] = Get(stat) + amount;
    }

    MovementState SimulateMovement(
        const MovementState& current,
        const MovementInput& input,
        const MovementConfig& config,
        const float deltaSeconds)
    {
        MovementState next = current;
        next.facingRadians = input.facingRadians;

        const float forward = Clamp(input.forwardAxis, -1.0F, 1.0F);
        const float strafe = Clamp(input.strafeAxis, -1.0F, 1.0F);
        const float baseSpeed = input.wantsToRun ? config.runSpeedMetersPerSecond : config.walkSpeedMetersPerSecond;
        const float forwardSpeed = forward >= 0.0F ? baseSpeed : baseSpeed * config.backpedalMultiplier;
        const float strafeSpeed = baseSpeed * config.strafeMultiplier;

        const float sinYaw = std::sin(next.facingRadians);
        const float cosYaw = std::cos(next.facingRadians);

        const Vec3 forwardVector { cosYaw, sinYaw, 0.0F };
        const Vec3 rightVector { -sinYaw, cosYaw, 0.0F };

        next.velocity.x = (forwardVector.x * forwardSpeed * forward) + (rightVector.x * strafeSpeed * strafe);
        next.velocity.y = (forwardVector.y * forwardSpeed * forward) + (rightVector.y * strafeSpeed * strafe);

        if (next.grounded && input.jumpPressed)
        {
            next.velocity.z = config.jumpVelocityMetersPerSecond;
            next.grounded = false;
        }
        else if (!next.grounded)
        {
            next.velocity.z += config.gravityMetersPerSecondSquared * deltaSeconds;
        }

        next.position.x += next.velocity.x * deltaSeconds;
        next.position.y += next.velocity.y * deltaSeconds;
        next.position.z += next.velocity.z * deltaSeconds;

        if (next.position.z <= 0.0F)
        {
            next.position.z = 0.0F;
            next.velocity.z = 0.0F;
            next.grounded = true;
        }

        return next;
    }

    CombatEvent ComputeMeleeSwing(
        const CombatSnapshot& snapshot,
        const float damageRoll01,
        const float outcomeRoll01)
    {
        CombatEvent event {};
        const float clampedDamageRoll = Clamp(damageRoll01, 0.0F, 1.0F);
        const float clampedOutcomeRoll = Clamp(outcomeRoll01, 0.0F, 1.0F);

        const float baseWeaponDamage = snapshot.attack.weaponMinDamage +
            (snapshot.attack.weaponMaxDamage - snapshot.attack.weaponMinDamage) * clampedDamageRoll;
        const float attackPowerContribution = (snapshot.attack.attackPower / 14.0F) * snapshot.swingSeconds;
        event.rawDamage = baseWeaponDamage + attackPowerContribution;

        const float critChance = Clamp(snapshot.attack.critChance, 0.0F, 1.0F);
        if (clampedOutcomeRoll < critChance)
        {
            event.outcome = CombatOutcome::Crit;
            event.rawDamage *= 2.0F;
        }

        if (snapshot.attack.school == DamageSchool::Physical)
        {
            const float armor = snapshot.defender.Get(StatId::Armor);
            const float mitigation = armor / (armor + 400.0F + (85.0F * static_cast<float>(snapshot.attackerLevel)));
            event.mitigatedDamage = event.rawDamage * (1.0F - Clamp(mitigation, 0.0F, 0.75F));
        }
        else
        {
            event.mitigatedDamage = event.rawDamage;
        }

        const std::int32_t levelDelta = static_cast<std::int32_t>(snapshot.defenderLevel) - static_cast<std::int32_t>(snapshot.attackerLevel);
        event.glancingBlow = levelDelta >= 2 && event.outcome == CombatOutcome::Hit;
        if (event.glancingBlow)
        {
            event.mitigatedDamage *= 0.7F;
        }

        return event;
    }

    std::vector<std::pair<std::string, std::uint32_t>> RollLoot(
        const LootTable& table,
        const std::vector<float>& randoms)
    {
        std::vector<std::pair<std::string, std::uint32_t>> drops;
        if (table.entries.empty())
        {
            return drops;
        }

        for (std::size_t index = 0; index < table.entries.size(); ++index)
        {
            const auto& entry = table.entries[index];
            const float roll = index < randoms.size() ? Clamp(randoms[index], 0.0F, 1.0F) : 0.5F;
            if (roll > entry.dropChance)
            {
                continue;
            }

            const std::uint32_t amountRange = entry.maxCount - entry.minCount;
            const std::uint32_t amount = entry.minCount + static_cast<std::uint32_t>(std::lround(amountRange * roll));
            drops.emplace_back(entry.itemId, amount);
        }

        return drops;
    }

    ObjectiveProgress ApplyObjectiveEvent(
        const ObjectiveDefinition& objective,
        const std::uint32_t currentCount,
        const ObjectiveType eventType,
        const std::string& targetId,
        const std::uint32_t increment)
    {
        ObjectiveProgress progress {};
        progress.objectiveId = objective.id;
        progress.currentCount = currentCount;

        if (objective.type != eventType || objective.targetId != targetId)
        {
            progress.completed = currentCount >= objective.requiredCount;
            return progress;
        }

        progress.currentCount = std::min(objective.requiredCount, currentCount + increment);
        progress.completed = progress.currentCount >= objective.requiredCount;
        return progress;
    }

    AiState EvaluateThreatState(
        const ThreatProfile& profile,
        const float distanceToTargetMeters,
        const float distanceFromAnchorMeters,
        const bool targetVisible,
        const bool hasDamageTaken)
    {
        if (distanceFromAnchorMeters > profile.leashRangeMeters)
        {
            return AiState::Returning;
        }

        if ((targetVisible && distanceToTargetMeters <= profile.acquisitionRangeMeters) || hasDamageTaken)
        {
            return AiState::Engaged;
        }

        if (targetVisible && distanceToTargetMeters <= profile.assistRadiusMeters)
        {
            return AiState::Suspicious;
        }

        return AiState::Idle;
    }
}
