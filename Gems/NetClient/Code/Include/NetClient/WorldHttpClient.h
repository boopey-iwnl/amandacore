#pragma once

#include <AzCore/Interface/Interface.h>
#include <AzCore/std/containers/vector.h>
#include <AzCore/std/string/string.h>

namespace NetClient
{
    struct WorldPosition
    {
        double m_x = 0.0;
        double m_y = 0.0;
        double m_z = 0.0;
    };

    struct NpcServiceState
    {
        AZStd::string m_type;
        AZStd::string m_serviceId;
        AZStd::string m_label;
    };

    struct VisibleEntity
    {
        AZStd::string m_id;
        AZStd::string m_displayName;
        AZStd::string m_kind;
        double m_x = 0.0;
        double m_y = 0.0;
        double m_z = 0.0;
        double m_health = 0.0;
        double m_maxHealth = 0.0;
        bool m_alive = false;
        bool m_targetable = false;
        AZStd::string m_aiState;
        AZStd::vector<NpcServiceState> m_services;
    };

    struct QuestState
    {
        AZStd::string m_id;
        AZStd::string m_title;
        AZStd::string m_objectiveType;
        AZStd::string m_objectiveText;
        AZStd::string m_state;
        AZStd::string m_giverNpcId;
        AZStd::string m_turnInNpcId;
        int m_currentCount = 0;
        int m_targetCount = 0;
        int m_rewardXp = 0;
        int m_rewardCurrencyTotalCopper = 0;
        int m_rewardCurrencySilver = 0;
        int m_rewardCurrencyGold = 0;
        int m_rewardCurrencyCopper = 0;
    };

    struct CurrencyState
    {
        int m_totalCopper = 0;
        int m_copper = 0;
        int m_silver = 0;
        int m_gold = 0;
    };

    struct InventorySlotState
    {
        int m_slotIndex = 0;
        AZStd::string m_itemId;
        AZStd::string m_displayName;
        int m_stackCount = 0;
    };

    struct InventoryState
    {
        int m_slotCount = 0;
        AZStd::vector<InventorySlotState> m_slots;
    };

    struct SpellbookEntryState
    {
        AZStd::string m_id;
        AZStd::string m_displayName;
        AZStd::string m_description;
        AZStd::string m_requirementText;
        int m_requiredLevel = 1;
        bool m_learned = false;
    };

    struct ActionBarSlotState
    {
        int m_slotIndex = 0;
        AZStd::string m_hotkey;
        AZStd::string m_abilityId;
        AZStd::string m_displayName;
        AZStd::string m_buttonLabel;
        bool m_requiresTarget = false;
        bool m_learned = false;
    };

    struct TrainerOfferState
    {
        AZStd::string m_abilityId;
        AZStd::string m_displayName;
        AZStd::string m_description;
        AZStd::string m_requirementText;
        int m_requiredLevel = 1;
        int m_costCopper = 0;
        bool m_learned = false;
        bool m_canLearn = false;
    };

    struct TrainerState
    {
        AZStd::string m_id;
        AZStd::string m_displayName;
        AZStd::string m_classId;
        AZStd::string m_interactionHint;
        bool m_inRange = false;
        AZStd::vector<TrainerOfferState> m_offers;
    };

    struct WorldSessionResponse
    {
        AZStd::string m_worldSessionToken;
        AZStd::string m_characterId;
        AZStd::string m_realmId;
        AZStd::string m_zoneId;
        AZStd::string m_displayName;
        int m_level = 0;
        WorldPosition m_position;
        double m_health = 0.0;
        double m_maxHealth = 0.0;
        double m_resource = 0.0;
        double m_maxResource = 0.0;
        int m_experience = 0;
        CurrencyState m_currency;
        InventoryState m_inventory;
        AZStd::vector<AZStd::string> m_learnedAbilityIds;
        AZStd::vector<SpellbookEntryState> m_spellbookEntries;
        AZStd::vector<ActionBarSlotState> m_actionBarSlots;
        TrainerState m_trainer;
        bool m_alive = false;
        QuestState m_quest;
        AZStd::string m_currentTargetId;
        bool m_autoAttackActive = false;
        AZ::s64 m_globalCooldownEndsAt = 0;
        AZ::s64 m_castEndsAt = 0;
        AZStd::string m_castingAbilityId;
        AZStd::vector<VisibleEntity> m_entities;
    };

    struct WorldBootstrapResponse
    {
        AZStd::string m_zoneId;
        AZStd::string m_cellId;
        AZStd::string m_motd;
        AZStd::string m_revision;
    };

    class IWorldHttpClient
    {
    public:
        AZ_RTTI(IWorldHttpClient, "{66740C9B-96C2-48E3-A2B3-0FD4689EB711}");

        virtual ~IWorldHttpClient() = default;

        static IWorldHttpClient* Get()
        {
            return AZ::Interface<IWorldHttpClient>::Get();
        }

        static void Register(IWorldHttpClient* instance)
        {
            AZ::Interface<IWorldHttpClient>::Register(instance);
        }

        static void Unregister(IWorldHttpClient* instance)
        {
            AZ::Interface<IWorldHttpClient>::Unregister(instance);
        }

        virtual bool Connect(
            const AZStd::string& worldEndpoint,
            const AZStd::string& ticketId,
            WorldSessionResponse& outResponse,
            AZStd::string& outError) = 0;

        virtual bool Bootstrap(
            const AZStd::string& worldEndpoint,
            WorldBootstrapResponse& outResponse,
            AZStd::string& outError) = 0;

        virtual bool Move(
            const AZStd::string& worldEndpoint,
            const AZStd::string& worldSessionToken,
            double deltaX,
            double deltaY,
            WorldSessionResponse& outResponse,
            AZStd::string& outError) = 0;

        virtual bool Disconnect(
            const AZStd::string& worldEndpoint,
            const AZStd::string& worldSessionToken,
            AZStd::string& outError) = 0;

        virtual bool State(
            const AZStd::string& worldEndpoint,
            const AZStd::string& worldSessionToken,
            WorldSessionResponse& outResponse,
            AZStd::string& outError) = 0;

        virtual bool SetTarget(
            const AZStd::string& worldEndpoint,
            const AZStd::string& worldSessionToken,
            const AZStd::string& targetId,
            WorldSessionResponse& outResponse,
            AZStd::string& outError) = 0;

        virtual bool AcceptQuest(
            const AZStd::string& worldEndpoint,
            const AZStd::string& worldSessionToken,
            const AZStd::string& questId,
            WorldSessionResponse& outResponse,
            AZStd::string& outError) = 0;

        virtual bool SetAutoAttack(
            const AZStd::string& worldEndpoint,
            const AZStd::string& worldSessionToken,
            bool enabled,
            WorldSessionResponse& outResponse,
            AZStd::string& outError) = 0;

        virtual bool ActivateAbility(
            const AZStd::string& worldEndpoint,
            const AZStd::string& worldSessionToken,
            const AZStd::string& abilityId,
            WorldSessionResponse& outResponse,
            AZStd::string& outError) = 0;

        virtual bool LearnTrainerAbility(
            const AZStd::string& worldEndpoint,
            const AZStd::string& worldSessionToken,
            const AZStd::string& trainerId,
            const AZStd::string& abilityId,
            WorldSessionResponse& outResponse,
            AZStd::string& outError) = 0;

        virtual bool AssignActionBarSlot(
            const AZStd::string& worldEndpoint,
            const AZStd::string& worldSessionToken,
            int slotIndex,
            const AZStd::string& abilityId,
            WorldSessionResponse& outResponse,
            AZStd::string& outError) = 0;

        virtual bool MoveActionBarSlot(
            const AZStd::string& worldEndpoint,
            const AZStd::string& worldSessionToken,
            int fromSlotIndex,
            int toSlotIndex,
            WorldSessionResponse& outResponse,
            AZStd::string& outError) = 0;

        virtual bool ClearActionBarSlot(
            const AZStd::string& worldEndpoint,
            const AZStd::string& worldSessionToken,
            int slotIndex,
            WorldSessionResponse& outResponse,
            AZStd::string& outError) = 0;

        virtual bool MoveInventorySlot(
            const AZStd::string& worldEndpoint,
            const AZStd::string& worldSessionToken,
            int fromSlotIndex,
            int toSlotIndex,
            WorldSessionResponse& outResponse,
            AZStd::string& outError) = 0;

        virtual bool Reconnect(
            const AZStd::string& worldEndpoint,
            const AZStd::string& worldSessionToken,
            WorldSessionResponse& outResponse,
            AZStd::string& outError) = 0;
    };
} // namespace NetClient
