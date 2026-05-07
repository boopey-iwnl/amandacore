#pragma once

#include <AzCore/Component/Component.h>
#include <AzCore/std/containers/array.h>
#include <AzCore/std/containers/deque.h>
#include <AzCore/std/string/string.h>
#include <AzFramework/Input/Events/InputChannelEventListener.h>
#include <GameCore/GameCoreInterface.h>
#include <ImGuiBus.h>

struct ImVec2;

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
        void AddNotification(const AZStd::string& title, const AZStd::string& message);
        void AddCombatFeedbackPulse(const AZStd::string& message, AZ::s64 nowMs);
        void DrawCombatFeedbackPulses(AZ::s64 nowMs, const ImVec2& displaySize);
        void DrawNotifications(AZ::s64 nowMs, const ImVec2& displaySize);
        void DrawTutorialHints(const GameCore::ClientWorldState& worldState, const ImVec2& displaySize);
        void ResetTutorialState();
        void MarkTutorialDismissed(int tutorialIndex);
        void OpenHelpPanel(const char* source);
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
        void BeginChatInput();
        void EndChatInput(bool clearBuffer);
        void MarkGameplayPanelOpened(const char* panelId);
        bool IsGameplayPanelOpen(const char* panelId) const;
        void RemoveGameplayPanelFromStack(const char* panelId);
        void SyncGameplayPanelStack();
        bool CloseGameplayPanelById(const char* panelId, const char* reason);
        void ClearPendingUiInteractionState();
        void ResetHudLayout();
        void ResetUiOptionsToDefaults();
        void DrawPreWorldFrontend(GameCore::IGameCoreRequests* gameCore, const ImVec2& displaySize);
        void ResetCharacterCreationDraft();

        AZStd::string m_lastQuestState;
        int m_lastQuestCount = -1;
        int m_lastExperience = -1;
        int m_lastCurrencyCopper = -1;
        AZStd::string m_lastHudTargetId;
        AZStd::string m_lastTargetFrameSummary;
        AZStd::string m_lastKillCreditSummary;
        AZStd::string m_lastWorldSessionToken;
        AZStd::string m_lastErrorMessage;
        AZStd::string m_activeInteractionEntityId;
        AZStd::string m_activeInteractionKind;
        AZStd::string m_selectedQuestId;
        AZStd::string m_questToast;
        AZStd::deque<AZStd::string> m_eventLog;
        AZStd::deque<AZStd::string> m_gameplayPanelStack;
        struct NotificationToast
        {
            AZStd::string m_title;
            AZStd::string m_message;
            AZ::s64 m_expiresAt = 0;
        };
        AZStd::deque<NotificationToast> m_notifications;
        AZStd::deque<AZStd::string> m_combatPulseTexts;
        AZStd::deque<AZ::s64> m_combatPulseExpiresAt;
        AZ::u64 m_lastHandledInteractionSequence = 0;
        AZ::s64 m_lastCombatDomainEventSequence = 0;
        AZ::s64 m_lastCombatStateDiffSequence = 0;
        AZ::s64 m_lastCombatPulseAt = 0;
        AZ::s64 m_questToastExpiresAt = 0;
        bool m_spellbookOpen = false;
        bool m_questGossipOpen = false;
        bool m_trainerOpen = false;
        bool m_bagOpen = false;
        bool m_gameMenuOpen = false;
        bool m_settingsOpen = false;
        bool m_helpOpen = false;
        bool m_socialOpen = false;
        bool m_auctionOpen = false;
        bool m_vendorOpen = false;
        bool m_characterSheetOpen = false;
        bool m_questLogOpen = false;
        bool m_mapOpen = false;
        bool m_talentsOpen = false;
        bool m_professionsOpen = false;
        bool m_uiEditMode = false;
        bool m_uiLayoutDirty = false;
        bool m_objectiveTrackerCollapsed = false;
        int m_settingsTab = 0;
        float m_uiScale = 1.0f;
        float m_readabilityScale = 1.0f;
        bool m_uiFramesLocked = true;
        bool m_chatFrameVisible = true;
        bool m_objectiveTrackerVisible = true;
        bool m_minimapVisible = true;
        bool m_actionBarsVisible = true;
        bool m_combatTextVisible = true;
        bool m_nameplatesVisible = true;
        bool m_tutorialsEnabled = true;
        bool m_keybindHintsVisible = true;
        bool m_tooltipsVisible = true;
        bool m_notificationsVisible = true;
        bool m_tooltipComparisonEnabled = true;
        bool m_reduceUiMotion = false;
        AZStd::string m_layoutProfileName = "Default";
        bool m_extraUpperActionBarVisible = true;
        bool m_rightActionBarOneVisible = true;
        bool m_rightActionBarTwoVisible = false;
        bool m_shiftHeld = false;
        bool m_lastUiEditMode = false;
        bool m_lastWorldConnected = false;
        bool m_lastNearCommandPoint = false;
        bool m_loggedActionBarVisible = false;
        bool m_loggedActionBarCooldownRendered = false;
        bool m_loggedCombatHudReady = false;
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
        AZStd::string m_chatFilterChannel = "all";
        AZStd::string m_keybindNotice;
        bool m_chatFocusRequested = false;
        bool m_chatInputActive = false;
        bool m_preWorldDiscreteInputEnabled = false;
        bool m_preWorldSettingsOpen = false;
        char m_chatInputBuffer[257]{};
        char m_loginUsernameBuffer[65]{};
        char m_loginPasswordBuffer[65]{};
        char m_characterNameBuffer[65]{};
        char m_chatWhisperTargetBuffer[65]{};
        char m_socialNameBuffer[65]{};
        char m_guildNameBuffer[65]{};
        char m_guildMotdBuffer[161]{};
        char m_auctionSearchBuffer[65]{};
        char m_auctionBuyoutBuffer[33]{};
        char m_helpFilterBuffer[65]{};
        int m_auctionTab = 0;
        int m_pendingAuctionSellSlot = -1;
        int m_pendingAuctionBuyoutIndex = -1;
        int m_auctionStackCount = 1;
        int m_selectedMailIndex = -1;
        int m_helpCategoryIndex = 0;
        int m_pendingVendorSellSlot = -1;
        int m_vendorStackCount = 1;
        int m_createLineageIndex = 0;
        int m_createBodyIndex = 0;
        int m_createSkinIndex = 0;
        int m_createFaceIndex = 0;
        int m_createHairIndex = 0;
        int m_createHairColorIndex = 0;
        int m_createMarkingIndex = 0;
        int m_pendingActionMoveSlot = -1;
        int m_pendingInventoryMoveSlot = -1;
        AZStd::string m_characterPanelNotice;
        AZStd::array<bool, 10> m_tutorialDismissed{};
        float m_previewYaw = 0.0f;
        float m_previewZoom = 1.0f;
        float m_notificationDurationSeconds = 4.5f;
        float m_chatOffsetX = 0.0f;
        float m_chatOffsetY = 0.0f;
        float m_objectiveTrackerOffsetX = 0.0f;
        float m_objectiveTrackerOffsetY = 0.0f;
        float m_actionBarOffsetX = 0.0f;
        float m_actionBarOffsetY = 0.0f;
        float m_bagOffsetX = 0.0f;
        float m_bagOffsetY = 0.0f;
        float m_minimapOffsetX = 0.0f;
        float m_minimapOffsetY = 0.0f;
    };
}
