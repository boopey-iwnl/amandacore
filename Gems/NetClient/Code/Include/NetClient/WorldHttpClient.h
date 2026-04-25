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
        AZStd::string m_category;
        AZStd::string m_statusBucket;
        AZStd::string m_objectiveType;
        AZStd::string m_objectiveText;
        AZStd::string m_state;
        AZStd::string m_giverNpcId;
        AZStd::string m_turnInNpcId;
        AZStd::string m_objectiveAreaId;
        AZStd::string m_objectiveAreaName;
        AZStd::string m_objectiveAreaKind;
        AZStd::string m_routeHintText;
        bool m_tracked = false;
        double m_objectiveX = 0.0;
        double m_objectiveY = 0.0;
        double m_objectiveRadius = 0.0;
        int m_currentCount = 0;
        int m_targetCount = 0;
        int m_rewardXp = 0;
        int m_rewardCurrencyTotalCopper = 0;
        int m_rewardCurrencySilver = 0;
        int m_rewardCurrencyGold = 0;
        int m_rewardCurrencyCopper = 0;
    };

    struct MapPointState
    {
        double m_x = 0.0;
        double m_y = 0.0;
    };

    struct MapRoadState
    {
        AZStd::string m_id;
        AZStd::string m_displayName;
        AZStd::vector<MapPointState> m_points;
    };

    struct MapLandmarkState
    {
        AZStd::string m_id;
        AZStd::string m_displayName;
        AZStd::string m_kind;
        double m_x = 0.0;
        double m_y = 0.0;
    };

    struct ZoneMapState
    {
        AZStd::string m_zoneId;
        AZStd::string m_displayName;
        double m_minX = 0.0;
        double m_minY = 0.0;
        double m_maxX = 0.0;
        double m_maxY = 0.0;
        AZStd::vector<MapRoadState> m_roads;
        AZStd::vector<MapLandmarkState> m_landmarks;
    };

    struct NavigationAreaState
    {
        AZStd::string m_areaId;
        AZStd::string m_displayName;
        AZStd::string m_kind;
        AZStd::string m_routeHintText;
        AZStd::string m_targetMobType;
        AZStd::string m_targetEntityId;
        double m_centerX = 0.0;
        double m_centerY = 0.0;
        double m_radius = 0.0;
        AZStd::vector<AZStd::string> m_questIds;
    };

    struct MapMarkerState
    {
        AZStd::string m_id;
        AZStd::string m_displayName;
        AZStd::string m_kind;
        AZStd::string m_questId;
        AZStd::string m_entityId;
        AZStd::string m_areaId;
        AZStd::string m_routeHintText;
        double m_x = 0.0;
        double m_y = 0.0;
        double m_radius = 0.0;
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
        AZStd::vector<QuestState> m_quests;
        AZStd::vector<AZStd::string> m_trackedQuestIds;
        ZoneMapState m_zoneMap;
        AZStd::vector<NavigationAreaState> m_navigationAreas;
        AZStd::vector<MapMarkerState> m_mapMarkers;
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

    struct ChatMessageState
    {
        AZStd::string m_messageId;
        AZStd::string m_channel;
        AZStd::string m_senderCharacterId;
        AZStd::string m_senderDisplayName;
        AZStd::string m_targetCharacterId;
        AZStd::string m_partyId;
        AZStd::string m_zoneId;
        AZStd::string m_messageText;
        AZ::s64 m_timestamp = 0;
    };

    struct FriendState
    {
        AZStd::string m_characterId;
        AZStd::string m_displayName;
        int m_level = 0;
        AZStd::string m_classId;
        AZStd::string m_zoneId;
        bool m_online = false;
    };

    struct PartyMemberState
    {
        AZStd::string m_characterId;
        AZStd::string m_displayName;
        int m_level = 0;
        AZStd::string m_classId;
        AZStd::string m_zoneId;
        bool m_online = false;
        bool m_leader = false;
        double m_health = 0.0;
        double m_maxHealth = 0.0;
        double m_resource = 0.0;
        double m_maxResource = 0.0;
        bool m_disconnected = false;
    };

    struct PartyState
    {
        AZStd::string m_partyId;
        AZStd::string m_leaderCharacterId;
        AZStd::vector<PartyMemberState> m_members;
    };

    struct PartyInviteState
    {
        AZStd::string m_inviteId;
        AZStd::string m_partyId;
        AZStd::string m_inviterCharacterId;
        AZStd::string m_inviterDisplayName;
        AZ::s64 m_expiresAt = 0;
    };

    struct SocialStateResponse
    {
        AZStd::vector<ChatMessageState> m_chatMessages;
        AZStd::vector<FriendState> m_friends;
        bool m_hasParty = false;
        PartyState m_party;
        AZStd::vector<PartyInviteState> m_partyInvites;
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

        virtual bool SocialState(
            const AZStd::string& worldEndpoint,
            const AZStd::string& worldSessionToken,
            const AZStd::string& afterMessageId,
            SocialStateResponse& outResponse,
            AZStd::string& outError) = 0;

        virtual bool SendChat(
            const AZStd::string& worldEndpoint,
            const AZStd::string& worldSessionToken,
            const AZStd::string& channel,
            const AZStd::string& targetName,
            const AZStd::string& messageText,
            SocialStateResponse& outResponse,
            AZStd::string& outError) = 0;

        virtual bool AddFriend(
            const AZStd::string& worldEndpoint,
            const AZStd::string& worldSessionToken,
            const AZStd::string& name,
            SocialStateResponse& outResponse,
            AZStd::string& outError) = 0;

        virtual bool RemoveFriend(
            const AZStd::string& worldEndpoint,
            const AZStd::string& worldSessionToken,
            const AZStd::string& name,
            SocialStateResponse& outResponse,
            AZStd::string& outError) = 0;

        virtual bool InviteParty(
            const AZStd::string& worldEndpoint,
            const AZStd::string& worldSessionToken,
            const AZStd::string& targetName,
            const AZStd::string& targetCharacterId,
            SocialStateResponse& outResponse,
            AZStd::string& outError) = 0;

        virtual bool AcceptPartyInvite(
            const AZStd::string& worldEndpoint,
            const AZStd::string& worldSessionToken,
            const AZStd::string& inviteId,
            SocialStateResponse& outResponse,
            AZStd::string& outError) = 0;

        virtual bool DeclinePartyInvite(
            const AZStd::string& worldEndpoint,
            const AZStd::string& worldSessionToken,
            const AZStd::string& inviteId,
            SocialStateResponse& outResponse,
            AZStd::string& outError) = 0;

        virtual bool LeaveParty(
            const AZStd::string& worldEndpoint,
            const AZStd::string& worldSessionToken,
            SocialStateResponse& outResponse,
            AZStd::string& outError) = 0;

        virtual bool DisbandParty(
            const AZStd::string& worldEndpoint,
            const AZStd::string& worldSessionToken,
            SocialStateResponse& outResponse,
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

        virtual bool TrackQuest(
            const AZStd::string& worldEndpoint,
            const AZStd::string& worldSessionToken,
            const AZStd::string& questId,
            bool tracked,
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
