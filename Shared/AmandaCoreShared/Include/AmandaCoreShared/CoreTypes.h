#pragma once

#include <cstdint>
#include <string>

namespace amandacore
{
    using EntityId = std::uint64_t;
    using WorldTick = std::uint64_t;

    enum class StatId
    {
        Health,
        Resource,
        Strength,
        Agility,
        Intellect,
        Spirit,
        Stamina,
        Armor,
        CritChance
    };

    enum class DamageSchool
    {
        Physical,
        Arcane,
        Fire,
        Frost,
        Nature,
        Shadow,
        Holy
    };

    enum class MovementMode
    {
        Ground,
        Swim,
        Air
    };

    enum class CombatOutcome
    {
        Hit,
        Crit,
        Miss,
        Dodge,
        Block
    };

    enum class ObjectiveType
    {
        Kill,
        Collect,
        Interact,
        Talk
    };

    enum class AiState
    {
        Idle,
        Suspicious,
        Engaged,
        Returning,
        Dead
    };
}
