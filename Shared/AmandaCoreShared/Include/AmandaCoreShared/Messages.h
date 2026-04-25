#pragma once

#include "AmandaCoreShared/Contracts.h"
#include "AmandaCoreShared/PlatformTypes.h"

#include <string>
#include <vector>

namespace amandacore
{
    struct Vec3
    {
        float x = 0.0F;
        float y = 0.0F;
        float z = 0.0F;
    };

    struct MovementIntentMessage
    {
        WorldTick clientTick = 0;
        float forwardAxis = 0.0F;
        float strafeAxis = 0.0F;
        bool wantsToRun = true;
        bool jumpPressed = false;
        float facingRadians = 0.0F;
    };

    struct AbilityRequestMessage
    {
        WorldTick clientTick = 0;
        EntityId casterId = 0;
        EntityId targetId = 0;
        std::string spellId;
    };

    struct QuestInteractionMessage
    {
        EntityId npcId = 0;
        std::string questId;
        bool accept = false;
        bool complete = false;
    };

    struct ChatMessage
    {
        EntityId speakerId = 0;
        std::string channel;
        std::string body;
    };

    struct LoginBootstrap
    {
        SessionId sessionId;
        AccountId accountId;
        RealmId selectedRealmId;
        BuildId buildId;
        std::vector<CharacterSummary> characters;
    };

    struct ReplicatedTransform
    {
        EntityId entityId = 0;
        Vec3 position;
        float facingRadians = 0.0F;
        MovementMode movementMode = MovementMode::Ground;
    };

    struct ReplicatedUnitState
    {
        EntityId entityId = 0;
        std::uint32_t level = 1;
        float health = 0.0F;
        float resource = 0.0F;
        AiState aiState = AiState::Idle;
    };

    struct ServerSnapshotMessage
    {
        WorldTick serverTick = 0;
        std::vector<ReplicatedTransform> transforms;
        std::vector<ReplicatedUnitState> units;
    };
}
