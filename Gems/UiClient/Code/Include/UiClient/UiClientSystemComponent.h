#pragma once

#include <AzCore/Component/Component.h>
#include <AzCore/std/containers/array.h>
#include <AzCore/std/containers/deque.h>
#include <AzCore/std/string/string.h>
#include <AzFramework/Input/Events/InputChannelEventListener.h>
#include <GameCore/GameCoreInterface.h>
#include <ImGuiBus.h>

namespace UiClient
{
    class UiClientSystemComponent final
        : public AZ::Component
        , public AzFramework::InputChannelEventListener
        , public ImGui::ImGuiUpdateListenerBus::Handler
    {
    public:
        AZ_COMPONENT(UiClientSystemComponent, "{A991C93B-A09D-4508-903F-590B6A29A827}");

        UiClientSystemComponent();

        static void Reflect(AZ::ReflectContext* context);
        static void GetProvidedServices(AZ::ComponentDescriptor::DependencyArrayType& provided);
        static void GetIncompatibleServices(AZ::ComponentDescriptor::DependencyArrayType& incompatible);
        static void GetRequiredServices(AZ::ComponentDescriptor::DependencyArrayType& required);
        static void GetDependentServices(AZ::ComponentDescriptor::DependencyArrayType& dependent);

        void Activate() override;
        void Deactivate() override;
        bool OnInputChannelEventFiltered(const AzFramework::InputChannel& inputChannel) override;
        void OnImGuiUpdate() override;

    private:
        void UpdateQuestToast(
            const AZStd::string& questState,
            int currentCount,
            int targetCount,
            int experience,
            int rewardXp,
            int totalCopper,
            int rewardGold,
            int rewardSilver,
            int rewardCopper);
        void AddHudEvent(const AZStd::string& message);
        void LoadUiSettings();
        void SaveUiSettings() const;
        void LoadDefaultKeybindings();
        void ApplyKeyBinding(const AZStd::string& actionId, const AZStd::string& keyName);
        bool TryHandleBoundAction(GameCore::IGameCoreRequests* gameCore, const AZStd::string& keyName);
        bool ActivateActionSlot(GameCore::IGameCoreRequests* gameCore, int slotIndex);
        bool TargetNextHostile(GameCore::IGameCoreRequests* gameCore);
        bool InteractWithCurrentTarget(GameCore::IGameCoreRequests* gameCore);
        bool OpenInteractionForEntity(
            GameCore::IGameCoreRequests* gameCore,
            const NetClient::VisibleEntity& entity,
            const char* source);
        bool CloseNpcInteraction(const char* reason);
        bool CloseOpenGameplayPanel(const char* reason);
        bool SubmitChatInput(GameCore::IGameCoreRequests* gameCore, const AZStd::string& input);

        AZStd::string m_lastQuestState;
        int m_lastQuestCount = -1;
        int m_lastExperience = -1;
        int m_lastCurrencyCopper = -1;
        AZStd::string m_lastHudTargetId;
        AZStd::string m_lastWorldSessionToken;
        AZStd::string m_lastErrorMessage;
        AZStd::string m_activeInteractionEntityId;
        AZStd::string m_activeInteractionKind;
        AZStd::string m_questToast;
        AZStd::deque<AZStd::string> m_eventLog;
        AZ::u64 m_lastHandledInteractionSequence = 0;
        AZ::s64 m_questToastExpiresAt = 0;
        bool m_spellbookOpen = false;
        bool m_questGossipOpen = false;
        bool m_trainerOpen = false;
        bool m_bagOpen = false;
        bool m_settingsOpen = false;
        bool m_socialOpen = false;
        bool m_auctionOpen = false;
        bool m_characterSheetOpen = false;
        bool m_questLogOpen = false;
        bool m_mapOpen = false;
        bool m_talentsOpen = false;
        int m_settingsTab = 0;
        bool m_extraUpperActionBarVisible = true;
        bool m_rightActionBarOneVisible = true;
        bool m_rightActionBarTwoVisible = false;
        bool m_shiftHeld = false;
        bool m_lastWorldConnected = false;
        bool m_lastNearCommandPoint = false;
        bool m_loggedActionBarVisible = false;
        bool m_loggedPlayableZoneReady = false;
        AZStd::string m_pendingActionAssignmentAbilityId;
        AZStd::string m_pendingKeybindActionId;
        AZStd::array<AZStd::string, 48> m_actionSlotBindings;
        AZStd::string m_spellbookBinding;
        AZStd::string m_bagBinding;
        AZStd::string m_characterBinding;
        AZStd::string m_questLogBinding;
        AZStd::string m_mapBinding;
        AZStd::string m_settingsBinding;
        AZStd::string m_interactBinding;
        AZStd::string m_targetHostileBinding;
        AZStd::string m_chatChannel = "say";
        bool m_chatFocusRequested = false;
        char m_chatInputBuffer[257]{};
        char m_chatWhisperTargetBuffer[65]{};
        char m_socialNameBuffer[65]{};
        char m_guildNameBuffer[65]{};
        char m_guildMotdBuffer[161]{};
        char m_auctionSearchBuffer[65]{};
        char m_auctionBuyoutBuffer[33]{};
        int m_auctionTab = 0;
        int m_pendingAuctionSellSlot = -1;
        int m_pendingAuctionBuyoutIndex = -1;
        int m_auctionStackCount = 1;
        int m_pendingActionMoveSlot = -1;
        int m_pendingInventoryMoveSlot = -1;
    };
}
