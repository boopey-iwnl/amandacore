#include <UiClient/UiClientSystemComponent.h>

#include <AzCore/Console/IConsole.h>
#include <AzCore/Math/MathUtils.h>
#include <AzCore/Math/Vector2.h>
#include <AzCore/Math/Vector3.h>
#include <AzCore/Serialization/SerializeContext.h>
#include <AzCore/std/chrono/chrono.h>
#include <AzCore/std/string/string.h>
#include <AzFramework/API/ApplicationAPI.h>
#include <AzFramework/Input/Channels/InputChannel.h>
#include <AzFramework/Input/Devices/Keyboard/InputDeviceKeyboard.h>
#include <GameCore/GameCoreInterface.h>
#include <GameCore/MobProxyGeometry.h>
#include <ImGuiBus.h>
#include <imgui/imgui.h>
#include <imgui/imgui_internal.h>
#include <algorithm>
#include <cstdio>
#include <cstdlib>
#include <cstring>
#ifndef NOMINMAX
#define NOMINMAX
#endif
#include <Windows.h>

namespace UiClient
{
    namespace
    {
        constexpr float CommandPointX = 13.0f;
        constexpr float CommandPointY = 10.0f;
        constexpr float CommandPointRadius = 5.0f;
        constexpr float EncounterAnchorX = 232.0f;
        constexpr float EncounterAnchorY = 118.0f;
        constexpr float FriendlyNameplateDrawDistance = 36.0f;
        constexpr float MeleeRange = 5.5f;
        constexpr float SpellRange = 24.0f;
        constexpr size_t MaxEventLogEntries = 9;
        constexpr const char* AutoAttackAbilityId = "auto_attack";
        constexpr const char* TrainerNpcKind = "trainer_npc";
        constexpr const char* QuestGiverNpcKind = "quest_giver_npc";
        constexpr const char* QuestGiverNpcId = "npc_commander_elian_rook";
        constexpr const char* SpellbookAbilityPayloadType = "AmandaCoreSpellbookAbility";
        constexpr const char* ActionBarSlotPayloadType = "AmandaCoreActionBarSlot";
        constexpr const char* InventorySlotPayloadType = "AmandaCoreInventorySlot";
        constexpr int ActionBarSlotCount = 48;

        struct SpellbookAbilityDragPayload
        {
            char m_abilityId[64]{};
        };

        struct ActionBarSlotDragPayload
        {
            int m_sourceSlotIndex = -1;
            char m_abilityId[64]{};
        };

        struct InventorySlotDragPayload
        {
            int m_sourceSlotIndex = -1;
        };

        AZStd::string SlotActionId(int slotIndex)
        {
            return AZStd::string::format("slot:%d", slotIndex);
        }

        bool TryParseSlotActionId(const AZStd::string& actionId, int& outSlotIndex)
        {
            if (actionId.rfind("slot:", 0) != 0)
            {
                return false;
            }

            outSlotIndex = atoi(actionId.c_str() + 5);
            return outSlotIndex >= 0 && outSlotIndex < ActionBarSlotCount;
        }

        AZStd::string DisplayKeyName(const AZStd::string& keyName)
        {
            if (keyName.empty())
            {
                return "Unbound";
            }

            constexpr const char* alphaPrefix = "keyboard_key_alphanumeric_";
            constexpr const char* functionPrefix = "keyboard_key_function_F";
            constexpr const char* editPrefix = "keyboard_key_edit_";
            constexpr const char* navPrefix = "keyboard_key_navigation_";
            constexpr const char* numPrefix = "keyboard_key_numpad_";
            constexpr const char* punctuationPrefix = "keyboard_key_punctuation_";

            auto stripPrefix = [&keyName](const char* prefix) -> AZStd::string
            {
                const size_t prefixLength = strlen(prefix);
                if (strncmp(keyName.c_str(), prefix, prefixLength) == 0)
                {
                    return keyName.substr(prefixLength);
                }
                return {};
            };

            AZStd::string display = stripPrefix(alphaPrefix);
            if (!display.empty())
            {
                return display;
            }

            display = stripPrefix(functionPrefix);
            if (!display.empty())
            {
                return AZStd::string::format("F%d", atoi(display.c_str()));
            }

            display = stripPrefix(editPrefix);
            if (!display.empty())
            {
                if (display == "space")
                {
                    return "Space";
                }
                if (display == "tab")
                {
                    return "Tab";
                }
                if (display == "enter")
                {
                    return "Enter";
                }
                if (display == "backspace")
                {
                    return "Backspace";
                }
                return display;
            }

            if (keyName == "keyboard_key_escape")
            {
                return "Esc";
            }

            display = stripPrefix(navPrefix);
            if (!display.empty())
            {
                return display;
            }

            display = stripPrefix(numPrefix);
            if (!display.empty())
            {
                return AZStd::string::format("Num %s", display.c_str());
            }

            display = stripPrefix(punctuationPrefix);
            if (!display.empty())
            {
                return display;
            }

            return keyName;
        }

        bool IsBindableKeyboardChannel(const AzFramework::InputChannelId& channelId)
        {
            const AZStd::string keyName = channelId.GetName();
            if (keyName == AzFramework::InputDeviceKeyboard::Key::ModifierShiftL.GetName() ||
                keyName == AzFramework::InputDeviceKeyboard::Key::ModifierShiftR.GetName())
            {
                return false;
            }

            return keyName.rfind("keyboard_key_", 0) == 0;
        }

        ImU32 ColorU32(int red, int green, int blue, int alpha = 255)
        {
            return IM_COL32(red, green, blue, alpha);
        }

        ImVec2 AddVec2(const ImVec2& left, const ImVec2& right)
        {
            return ImVec2(left.x + right.x, left.y + right.y);
        }

        float Distance2D(float leftX, float leftY, float rightX, float rightY)
        {
            const float deltaX = rightX - leftX;
            const float deltaY = rightY - leftY;
            return AZStd::sqrt((deltaX * deltaX) + (deltaY * deltaY));
        }

        float Clamp01(float value)
        {
            return AZ::GetClamp(value, 0.0f, 1.0f);
        }

        bool ProjectWorldPointToScreen(
            const GameCore::ClientCameraState& cameraState,
            const AZ::Vector3& worldPoint,
            const ImVec2& displaySize,
            ImVec2& outScreenPosition)
        {
            if (!cameraState.m_ready || displaySize.x <= 1.0f || displaySize.y <= 1.0f)
            {
                return false;
            }

            const AZ::Transform inverseView = cameraState.m_worldTransform.GetInverse();
            const AZ::Vector3 cameraLocal = inverseView.TransformPoint(worldPoint);
            if (cameraLocal.GetY() <= 0.05f)
            {
                return false;
            }

            float aspectRatio = displaySize.x / displaySize.y;
            if (aspectRatio <= 0.01f)
            {
                aspectRatio = 16.0f / 9.0f;
            }

            const float tanHalfFov = AZStd::tan(AZ::DegToRad(cameraState.m_verticalFovDegrees) * 0.5f);
            if (tanHalfFov <= 0.0f)
            {
                return false;
            }

            const float ndcX = cameraLocal.GetX() / (cameraLocal.GetY() * tanHalfFov * aspectRatio);
            const float ndcY = cameraLocal.GetZ() / (cameraLocal.GetY() * tanHalfFov);
            if (AZ::GetAbs(ndcX) > 1.05f || AZ::GetAbs(ndcY) > 1.05f)
            {
                return false;
            }

            outScreenPosition.x = ((ndcX + 1.0f) * 0.5f) * displaySize.x;
            outScreenPosition.y = ((1.0f - ndcY) * 0.5f) * displaySize.y;
            return true;
        }

        AZ::s64 NowMs()
        {
            return AZStd::chrono::duration_cast<AZStd::chrono::milliseconds>(
                       AZStd::chrono::system_clock::now().time_since_epoch())
                .count();
        }

        AZStd::string GetMobDebugSuffix(const AZStd::string& entityId)
        {
            const size_t separatorIndex = entityId.find_last_of('_');
            if (separatorIndex == AZStd::string::npos || separatorIndex + 1 >= entityId.size())
            {
                return {};
            }

            return AZStd::string::format("[%s]", entityId.substr(separatorIndex + 1).c_str());
        }

        AZStd::string GetMobDisplayLabel(const NetClient::VisibleEntity& entity)
        {
            const AZStd::string suffix = GetMobDebugSuffix(entity.m_id);
            const bool isElite = entity.m_elite || entity.m_classification == "elite";
            const AZStd::string baseLabel = isElite
                ? AZStd::string::format("%s (Elite)", entity.m_displayName.c_str())
                : entity.m_displayName;
            if (suffix.empty())
            {
                return baseLabel;
            }

            return AZStd::string::format("%s %s", baseLabel.c_str(), suffix.c_str());
        }

        AZStd::string GetQuestDisplayTitle(const NetClient::QuestState& quest)
        {
            if (!quest.m_groupRecommended)
            {
                return quest.m_title;
            }
            if (quest.m_recommendedPlayers > 0)
            {
                return AZStd::string::format("[Group %d] %s", quest.m_recommendedPlayers, quest.m_title.c_str());
            }
            return AZStd::string::format("[Group] %s", quest.m_title.c_str());
        }

        AZStd::string FormatCurrency(const NetClient::CurrencyState& currency)
        {
            return AZStd::string::format("%dg %ds %dc", currency.m_gold, currency.m_silver, currency.m_copper);
        }

        AZStd::string GetUiSettingsPath()
        {
            char* localAppData = nullptr;
            size_t localAppDataLength = 0;
            _dupenv_s(&localAppData, &localAppDataLength, "LOCALAPPDATA");
            const AZStd::string basePath = localAppData && localAppData[0] != '\0'
                ? AZStd::string(localAppData)
                : AZStd::string(".");
            if (localAppData)
            {
                free(localAppData);
            }

            const AZStd::string settingsDirectory = basePath + "\\AmandaCore";
            CreateDirectoryA(settingsDirectory.c_str(), nullptr);
            return settingsDirectory + "\\ui-settings.ini";
        }

        AZStd::string FormatDistanceState(float distanceToTarget)
        {
            if (distanceToTarget <= MeleeRange)
            {
                return AZStd::string::format("Range %.1fm  |  melee + spell", distanceToTarget);
            }
            if (distanceToTarget <= SpellRange)
            {
                return AZStd::string::format("Range %.1fm  |  spell only", distanceToTarget);
            }

            return AZStd::string::format("Range %.1fm  |  out of range", distanceToTarget);
        }

        const NetClient::VisibleEntity* FindTargetEntity(const GameCore::ClientWorldState& worldState)
        {
            if (worldState.m_session.m_currentTargetId.empty())
            {
                return nullptr;
            }

            for (const auto& entity : worldState.m_session.m_entities)
            {
                if (entity.m_id == worldState.m_session.m_currentTargetId)
                {
                    return &entity;
                }
            }
            return nullptr;
        }

        const NetClient::VisibleEntity* FindTrainerEntity(const GameCore::ClientWorldState& worldState)
        {
            for (const auto& entity : worldState.m_session.m_entities)
            {
                if (entity.m_kind == TrainerNpcKind)
                {
                    return &entity;
                }
            }
            return nullptr;
        }

        bool IsFriendlyNpc(const NetClient::VisibleEntity& entity)
        {
            return entity.m_kind == TrainerNpcKind || entity.m_kind == QuestGiverNpcKind || !entity.m_services.empty();
        }

        bool EntityHasService(const NetClient::VisibleEntity& entity, const char* serviceType)
        {
            for (const auto& service : entity.m_services)
            {
                if (service.m_type == serviceType)
                {
                    return true;
                }
            }
            return false;
        }

        bool IsTrainerNpc(const NetClient::VisibleEntity& entity)
        {
            return entity.m_kind == TrainerNpcKind || EntityHasService(entity, "trainer");
        }

        bool IsQuestGiverNpc(const NetClient::VisibleEntity& entity)
        {
            return entity.m_kind == QuestGiverNpcKind || entity.m_id == QuestGiverNpcId || EntityHasService(entity, "quest");
        }

        bool ShouldOpenQuestForEntity(const GameCore::ClientWorldState& worldState, const NetClient::VisibleEntity& entity)
        {
            if (!IsQuestGiverNpc(entity))
            {
                return false;
            }

            const auto& quest = worldState.m_session.m_quest;
            if (quest.m_id.empty())
            {
                return false;
            }

            const bool isQuestGiver = quest.m_giverNpcId == entity.m_id;
            const bool isQuestTurnIn = quest.m_turnInNpcId == entity.m_id;
            if (quest.m_state == "not_started")
            {
                return isQuestGiver;
            }
            if (quest.m_state == "completed")
            {
                return isQuestTurnIn;
            }
            if (quest.m_state == "active")
            {
                if (quest.m_currentCount >= quest.m_targetCount)
                {
                    return isQuestTurnIn;
                }

                // Talk/trainer/explore/use quests can complete from their destination service NPC.
                return isQuestTurnIn &&
                    (quest.m_objectiveType == "talk" ||
                     quest.m_objectiveType == "trainer" ||
                     quest.m_objectiveType == "explore" ||
                     quest.m_objectiveType == "use_location");
            }

            return false;
        }

        AZStd::string GetInventorySlotLabel(const NetClient::InventorySlotState& slot)
        {
            if (slot.m_itemId.empty() || slot.m_stackCount <= 0)
            {
                return {};
            }

            size_t separatorIndex = slot.m_displayName.find_last_of(' ');
            AZStd::string shortName = separatorIndex == AZStd::string::npos
                ? slot.m_displayName
                : slot.m_displayName.substr(separatorIndex + 1);
            if (shortName.empty())
            {
                shortName = slot.m_displayName;
            }

            return AZStd::string::format("%s\nx%d", shortName.c_str(), slot.m_stackCount);
        }

        int CountOccupiedSlots(const NetClient::InventoryState& inventory)
        {
            int occupiedSlots = 0;
            for (const auto& slot : inventory.m_slots)
            {
                if (!slot.m_itemId.empty() && slot.m_stackCount > 0)
                {
                    ++occupiedSlots;
                }
            }
            return occupiedSlots;
        }

        const NetClient::ActionBarSlotState* FindActionBarSlot(
            const NetClient::WorldSessionResponse& session,
            int slotIndex)
        {
            for (const auto& slot : session.m_actionBarSlots)
            {
                if (slot.m_slotIndex == slotIndex)
                {
                    return &slot;
                }
            }

            return nullptr;
        }

        AZStd::string BuildActionBarHelpText(
            const NetClient::WorldSessionResponse& session,
            const AZStd::array<AZStd::string, ActionBarSlotCount>& actionSlotBindings,
            const AZStd::string& spellbookBinding,
            const AZStd::string& bagBinding,
            const AZStd::string& settingsBinding,
            const AZStd::string& interactBinding,
            const AZStd::string& targetHostileBinding)
        {
            AZStd::string helpText = AZStd::string::format(
                "%s target hostile  |  LMB target  |  RMB friendly NPC interact  |  %s interact  |  %s spellbook  |  %s bag  |  %s menu  |  Hold SHIFT to edit bars",
                DisplayKeyName(targetHostileBinding).c_str(),
                DisplayKeyName(interactBinding).c_str(),
                DisplayKeyName(spellbookBinding).c_str(),
                DisplayKeyName(bagBinding).c_str(),
                DisplayKeyName(settingsBinding).c_str());
            for (const auto& slot : session.m_actionBarSlots)
            {
                if (slot.m_displayName.empty() || slot.m_slotIndex < 0 || slot.m_slotIndex >= ActionBarSlotCount)
                {
                    continue;
                }

                const AZStd::string hotkey = DisplayKeyName(actionSlotBindings[slot.m_slotIndex]);
                if (hotkey.empty() || hotkey == "Unbound")
                {
                    continue;
                }

                helpText += AZStd::string::format("  |  %s %s", hotkey.c_str(), slot.m_displayName.c_str());
            }

            return helpText;
        }

        AZStd::string FormatTrainerCost(int totalCopper)
        {
            const NetClient::CurrencyState currency{
                totalCopper,
                totalCopper % 100,
                (totalCopper % 10000) / 100,
                totalCopper / 10000,
            };
            return FormatCurrency(currency);
        }

        AZStd::string FormatAbilityFacts(
            const AZStd::string& resourceName,
            double resourceCost,
            double resourceGeneration,
            AZ::s64 cooldownMs,
            double rangeMeters)
        {
            AZStd::string facts;
            const AZStd::string resolvedResourceName = resourceName.empty() ? "Grit" : resourceName;
            if (resourceCost > 0.0)
            {
                facts += AZStd::string::format("Cost %.0f %s", resourceCost, resolvedResourceName.c_str());
            }
            if (resourceGeneration > 0.0)
            {
                if (!facts.empty())
                {
                    facts += "  |  ";
                }
                facts += AZStd::string::format("Generates %.0f %s", resourceGeneration, resolvedResourceName.c_str());
            }
            if (cooldownMs > 0)
            {
                if (!facts.empty())
                {
                    facts += "  |  ";
                }
                facts += AZStd::string::format("Cooldown %.1fs", static_cast<double>(cooldownMs) / 1000.0);
            }
            if (rangeMeters > 0.0)
            {
                if (!facts.empty())
                {
                    facts += "  |  ";
                }
                facts += AZStd::string::format("Range %.1fm", rangeMeters);
            }
            return facts;
        }

        AZStd::string FormatAbilityFacts(const NetClient::SpellbookEntryState& entry)
        {
            return FormatAbilityFacts(
                entry.m_resourceName,
                entry.m_resourceCost,
                entry.m_resourceGeneration,
                entry.m_cooldownMs,
                entry.m_rangeMeters);
        }

        AZStd::string FormatAbilityFacts(const NetClient::ActionBarSlotState& slot)
        {
            return FormatAbilityFacts(
                slot.m_resourceName,
                slot.m_resourceCost,
                slot.m_resourceGeneration,
                slot.m_cooldownMs,
                slot.m_rangeMeters);
        }

        AZStd::string FormatAbilityFacts(const NetClient::TrainerOfferState& offer)
        {
            return FormatAbilityFacts(
                offer.m_resourceName,
                offer.m_resourceCost,
                offer.m_resourceGeneration,
                offer.m_cooldownMs,
                offer.m_rangeMeters);
        }

        void SuppressStockImGuiWindow(const char* windowName)
        {
            if (ImGuiWindow* window = ImGui::FindWindowByName(windowName))
            {
                ImGui::SetWindowCollapsed(window, true, ImGuiCond_Always);
                window->Hidden = true;
                window->HiddenFramesCanSkipItems = 2;
                window->HiddenFramesCannotSkipItems = 0;
                window->HiddenFramesForRenderOnly = 2;
                window->Collapsed = true;
                window->Active = false;
            }
        }

        void SuppressStockImGuiChrome()
        {
            ImGui::ImGuiEntityOutlinerRequestBus::Broadcast(&ImGui::ImGuiEntityOutlinerRequestBus::Events::SetEnabled, false);
            ImGui::ImGuiAssetExplorerRequestBus::Broadcast(&ImGui::ImGuiAssetExplorerRequestBus::Events::SetEnabled, false);
            ImGui::ImGuiCameraMonitorRequestBus::Broadcast(&ImGui::ImGuiCameraMonitorRequestBus::Events::SetEnabled, false);

            static constexpr const char* HiddenWindows[] = {
                "##MainMenuBar",
                "Debug##Default",
                "Console",
                "Display Info",
                "Input Monitor",
                "Asset Explorer",
                "Camera Monitor",
                "Entity Outliner",
                "Pass Viewer",
                "Gpu Profiler",
                "Transient Attachment Profiler",
                "Render Pipelines",
                "Video Options",
            };

            for (const char* windowName : HiddenWindows)
            {
                SuppressStockImGuiWindow(windowName);
            }
        }

        void DrawPanelChrome(ImDrawList* drawList, const ImVec2& minBounds, const ImVec2& maxBounds, const char* title)
        {
            drawList->AddRectFilled(minBounds, maxBounds, ColorU32(10, 14, 19, 228), 16.0f);
            drawList->AddRect(minBounds, maxBounds, ColorU32(153, 119, 56, 235), 16.0f, 0, 2.0f);
            drawList->AddRectFilled(
                ImVec2(minBounds.x + 2.0f, minBounds.y + 2.0f),
                ImVec2(maxBounds.x - 2.0f, minBounds.y + 30.0f),
                ColorU32(36, 51, 62, 214),
                14.0f);
            drawList->AddLine(
                ImVec2(minBounds.x + 12.0f, minBounds.y + 30.0f),
                ImVec2(maxBounds.x - 12.0f, minBounds.y + 30.0f),
                ColorU32(73, 122, 124, 255),
                1.5f);
            if (title && title[0] != '\0')
            {
                drawList->AddText(ImVec2(minBounds.x + 14.0f, minBounds.y + 8.0f), ColorU32(238, 224, 192), title);
            }
        }

        bool BeginHudPanel(const char* identifier, const char* title, const ImVec2& position, const ImVec2& size)
        {
            ImGui::SetNextWindowPos(position, ImGuiCond_Always);
            ImGui::SetNextWindowSize(size, ImGuiCond_Always);
            ImGui::PushStyleColor(ImGuiCol_WindowBg, ImVec4(0.0f, 0.0f, 0.0f, 0.0f));
            ImGui::PushStyleColor(ImGuiCol_Border, ImVec4(0.0f, 0.0f, 0.0f, 0.0f));
            ImGui::PushStyleVar(ImGuiStyleVar_WindowPadding, ImVec2(14.0f, 14.0f));
            ImGui::PushStyleVar(ImGuiStyleVar_WindowRounding, 16.0f);
            const ImGuiWindowFlags flags = ImGuiWindowFlags_NoCollapse |
                ImGuiWindowFlags_NoResize |
                ImGuiWindowFlags_NoMove |
                ImGuiWindowFlags_NoScrollbar |
                ImGuiWindowFlags_NoTitleBar |
                ImGuiWindowFlags_NoSavedSettings;
            const bool visible = ImGui::Begin(identifier, nullptr, flags);
            ImGui::PopStyleVar(2);
            ImGui::PopStyleColor(2);
            const ImVec2 windowPosition = ImGui::GetWindowPos();
            const ImVec2 windowSize = ImGui::GetWindowSize();

            DrawPanelChrome(
                ImGui::GetWindowDrawList(),
                windowPosition,
                AddVec2(windowPosition, windowSize),
                title);
            ImGui::SetCursorScreenPos(ImVec2(windowPosition.x + 14.0f, windowPosition.y + 40.0f));
            return visible;
        }

        void DrawMeter(
            const char* label,
            float currentValue,
            float maxValue,
            ImU32 fillColor,
            const ImVec2& size)
        {
            ImGui::TextUnformatted(label);
            const ImVec2 barPosition = ImGui::GetCursorScreenPos();
            ImGui::InvisibleButton(label, size);
            ImDrawList* drawList = ImGui::GetWindowDrawList();
            drawList->AddRectFilled(barPosition, AddVec2(barPosition, size), ColorU32(22, 30, 38, 255), 7.0f);
            drawList->AddRect(barPosition, AddVec2(barPosition, size), ColorU32(131, 108, 64, 255), 7.0f, 0, 1.0f);

            const float ratio = maxValue > 0.0f ? Clamp01(currentValue / maxValue) : 0.0f;
            const ImVec2 fillExtent(barPosition.x + (size.x * ratio), barPosition.y + size.y);
            drawList->AddRectFilled(barPosition, fillExtent, fillColor, 7.0f);

            const AZStd::string text = AZStd::string::format("%.0f / %.0f", currentValue, maxValue);
            const ImVec2 textSize = ImGui::CalcTextSize(text.c_str());
            drawList->AddText(
                ImVec2(
                    barPosition.x + ((size.x - textSize.x) * 0.5f),
                    barPosition.y + ((size.y - textSize.y) * 0.5f)),
                ColorU32(245, 243, 234),
                text.c_str());
        }

        void DrawPortraitBadge(const AZStd::string& glyph, const ImVec2& center, ImU32 fillColor)
        {
            ImDrawList* drawList = ImGui::GetWindowDrawList();
            drawList->AddCircleFilled(center, 28.0f, fillColor, 40);
            drawList->AddCircle(center, 28.0f, ColorU32(210, 182, 118), 40, 2.0f);
            const ImVec2 glyphSize = ImGui::CalcTextSize(glyph.c_str());
            drawList->AddText(
                ImVec2(center.x - (glyphSize.x * 0.5f), center.y - (glyphSize.y * 0.5f)),
                ColorU32(250, 246, 238),
                glyph.c_str());
        }

        void DrawPlayerFrame(
            const GameCore::ClientWorldState& worldState,
            float distanceToCommandPoint,
            bool nearCommandPoint)
        {
            const ImVec2 origin = ImGui::GetCursorScreenPos();
            DrawPortraitBadge(
                worldState.m_session.m_displayName.empty()
                    ? "P"
                    : AZStd::string::format("%c", worldState.m_session.m_displayName.front()),
                ImVec2(origin.x + 34.0f, origin.y + 42.0f),
                ColorU32(36, 124, 136));

            ImGui::SetCursorScreenPos(ImVec2(origin.x + 74.0f, origin.y));
            ImGui::Text("%s  |  Level %d", worldState.m_session.m_displayName.c_str(), worldState.m_session.m_level);
            ImGui::TextUnformatted(nearCommandPoint ? "Near friendly NPC services" : "Stonewake Vale patrol");
            DrawMeter("Vitality", static_cast<float>(worldState.m_session.m_health), static_cast<float>(worldState.m_session.m_maxHealth), ColorU32(173, 52, 44), ImVec2(160.0f, 18.0f));
            DrawMeter(
                worldState.m_session.m_resourceName.empty() ? "Grit" : worldState.m_session.m_resourceName.c_str(),
                static_cast<float>(worldState.m_session.m_resource),
                static_cast<float>(worldState.m_session.m_maxResource),
                ColorU32(54, 117, 181),
                ImVec2(160.0f, 16.0f));
            ImGui::Text("Friendly NPCs %.1fm", distanceToCommandPoint);
        }

        void DrawTargetFrame(
            GameCore::IGameCoreRequests* gameCore,
            const NetClient::VisibleEntity* targetEntity,
            const GameCore::ClientWorldState& worldState,
            float playerX,
            float playerY)
        {
            (void)gameCore;
            (void)worldState;
            const ImVec2 origin = ImGui::GetCursorScreenPos();
            if (!targetEntity)
            {
                DrawPortraitBadge("?", ImVec2(origin.x + 34.0f, origin.y + 42.0f), ColorU32(72, 74, 78));
                ImGui::SetCursorScreenPos(ImVec2(origin.x + 74.0f, origin.y + 10.0f));
                ImGui::TextUnformatted("No target selected");
                ImGui::TextUnformatted("Left-click NPCs or hostiles");
                ImGui::TextUnformatted("Tab cycles hostile targets");
                return;
            }

            const float distanceToTarget = Distance2D(
                playerX,
                playerY,
                static_cast<float>(targetEntity->m_x),
                static_cast<float>(targetEntity->m_y));
            if (IsFriendlyNpc(*targetEntity))
            {
                const bool isTrainer = IsTrainerNpc(*targetEntity);
                DrawPortraitBadge(isTrainer ? "T" : "Q", ImVec2(origin.x + 34.0f, origin.y + 42.0f), isTrainer ? ColorU32(44, 103, 167) : ColorU32(42, 136, 84));
                ImGui::SetCursorScreenPos(ImVec2(origin.x + 74.0f, origin.y));
                ImGui::TextUnformatted(targetEntity->m_displayName.c_str());
                ImGui::TextUnformatted(isTrainer ? "Friendly Warrior Trainer" : "Friendly Quest Giver");
                ImGui::Text("Range %.1fm  |  %s", distanceToTarget, distanceToTarget <= CommandPointRadius ? "interactable" : "move closer");
                if (distanceToTarget <= CommandPointRadius)
                {
                    ImGui::TextUnformatted(isTrainer ? "Right-click the NPC model to train." : "Right-click the NPC model for field orders.");
                }
                else
                {
                    ImGui::TextUnformatted("Move closer, then right-click the NPC model.");
                }
                return;
            }

            const AZStd::string targetLabel = GetMobDisplayLabel(*targetEntity);
            DrawPortraitBadge("!", ImVec2(origin.x + 34.0f, origin.y + 42.0f), ColorU32(146, 72, 44));
            ImGui::SetCursorScreenPos(ImVec2(origin.x + 74.0f, origin.y));
            ImGui::TextUnformatted(targetLabel.c_str());
            ImGui::Text("%s  |  %s  |  %s", "Hostile", targetEntity->m_elite ? "elite" : "normal", targetEntity->m_alive ? "engageable" : "down");
            DrawMeter("Integrity", static_cast<float>(targetEntity->m_health), static_cast<float>(targetEntity->m_maxHealth), ColorU32(198, 93, 34), ImVec2(160.0f, 18.0f));
            ImGui::TextUnformatted(FormatDistanceState(distanceToTarget).c_str());
            ImGui::Text("Behavior: %s", targetEntity->m_aiState.empty() ? "idle" : targetEntity->m_aiState.c_str());
        }

        void DrawMinimap(
            const GameCore::ClientWorldState& worldState,
            float playerX,
            float playerY)
        {
            ImGui::TextUnformatted("Stonewake Vale");
            ImGui::TextUnformatted("Hearthwatch Yard");
            const ImVec2 mapRegion = ImVec2(208.0f, 188.0f);
            const ImVec2 regionMin = ImGui::GetCursorScreenPos();
            ImGui::InvisibleButton("##stonewake_vale_minimap", mapRegion);
            ImDrawList* drawList = ImGui::GetWindowDrawList();
            const ImVec2 center(regionMin.x + (mapRegion.x * 0.5f), regionMin.y + (mapRegion.y * 0.52f));
            const float radius = 84.0f;
            drawList->AddCircleFilled(center, radius, ColorU32(16, 25, 28, 240), 48);
            drawList->AddCircle(center, radius, ColorU32(182, 142, 69), 48, 2.0f);
            drawList->AddCircle(center, radius * 0.55f, ColorU32(58, 87, 90, 180), 40, 1.0f);
            drawList->AddText(ImVec2(center.x - 6.0f, center.y - radius - 14.0f), ColorU32(224, 214, 189), "N");

            const float mapScale = radius / 140.0f;
            auto plotWorldPoint = [&](float worldX, float worldY, ImU32 color, float pointRadius)
            {
                const float deltaX = (worldX - playerX) * mapScale;
                const float deltaY = (worldY - playerY) * mapScale;
                const ImVec2 point(center.x + deltaX, center.y - deltaY);
                const float pointDeltaX = point.x - center.x;
                const float pointDeltaY = point.y - center.y;
                if ((pointDeltaX * pointDeltaX) + (pointDeltaY * pointDeltaY) <= (radius - 8.0f) * (radius - 8.0f))
                {
                    drawList->AddCircleFilled(point, pointRadius, color, 24);
                }
            };

            for (const auto& marker : worldState.m_session.m_mapMarkers)
            {
                ImU32 color = ColorU32(226, 183, 74);
                float pointRadius = 4.0f;
                if (marker.m_kind == "quest_turn_in")
                {
                    color = ColorU32(76, 218, 141);
                    pointRadius = 5.0f;
                }
                else if (marker.m_kind == "quest_available")
                {
                    color = ColorU32(214, 194, 84);
                    pointRadius = 5.0f;
                }
                else if (marker.m_kind == "tracked_objective" || marker.m_kind == "quest_objective")
                {
                    color = ColorU32(228, 111, 54);
                    pointRadius = 5.0f;
                }
                else if (marker.m_kind == "trainer")
                {
                    color = ColorU32(88, 165, 235);
                    pointRadius = 5.0f;
                }
                else if (marker.m_kind == "vendor")
                {
                    color = ColorU32(183, 135, 223);
                    pointRadius = 5.0f;
                }
                plotWorldPoint(static_cast<float>(marker.m_x), static_cast<float>(marker.m_y), color, pointRadius);
            }
            plotWorldPoint(EncounterAnchorX, EncounterAnchorY, ColorU32(228, 111, 54), 5.0f);
            for (const auto& entity : worldState.m_session.m_entities)
            {
                if (entity.m_kind == TrainerNpcKind && entity.m_alive)
                {
                    plotWorldPoint(static_cast<float>(entity.m_x), static_cast<float>(entity.m_y), ColorU32(88, 165, 235), 5.0f);
                    continue;
                }
                if (entity.m_kind == QuestGiverNpcKind && entity.m_alive)
                {
                    plotWorldPoint(static_cast<float>(entity.m_x), static_cast<float>(entity.m_y), ColorU32(76, 218, 141), 5.0f);
                    continue;
                }
                if (entity.m_kind != "hostile_mob" || !entity.m_alive)
                {
                    continue;
                }
                plotWorldPoint(static_cast<float>(entity.m_x), static_cast<float>(entity.m_y), ColorU32(232, 191, 84), 4.0f);
            }

            drawList->AddTriangleFilled(
                ImVec2(center.x, center.y - 10.0f),
                ImVec2(center.x - 7.0f, center.y + 8.0f),
                ImVec2(center.x + 7.0f, center.y + 8.0f),
                ColorU32(226, 240, 243));
            ImGui::TextUnformatted("Markers: quest giver, trainer, route, hostiles");
        }

        ImU32 MapMarkerColor(const AZStd::string& kind)
        {
            if (kind == "quest_turn_in")
            {
                return ColorU32(76, 218, 141);
            }
            if (kind == "quest_available")
            {
                return ColorU32(214, 194, 84);
            }
            if (kind == "tracked_objective" || kind == "quest_objective")
            {
                return ColorU32(228, 111, 54);
            }
            if (kind == "trainer")
            {
                return ColorU32(88, 165, 235);
            }
            if (kind == "vendor")
            {
                return ColorU32(183, 135, 223);
            }
            return ColorU32(210, 210, 198);
        }

        void DrawZoneMapWindow(
            const GameCore::ClientWorldState& worldState,
            float playerX,
            float playerY)
        {
            const auto& zoneMap = worldState.m_session.m_zoneMap;
            ImGui::Text("%s", zoneMap.m_displayName.empty() ? "Stonewake Vale" : zoneMap.m_displayName.c_str());
            ImGui::TextUnformatted("Authored navigation blockout");

            const ImVec2 canvasSize(590.0f, 360.0f);
            const ImVec2 canvasMin = ImGui::GetCursorScreenPos();
            ImGui::InvisibleButton("##stonewake_zone_map_canvas", canvasSize);
            ImDrawList* drawList = ImGui::GetWindowDrawList();
            const ImVec2 canvasMax(canvasMin.x + canvasSize.x, canvasMin.y + canvasSize.y);
            drawList->AddRectFilled(canvasMin, canvasMax, ColorU32(18, 25, 29, 245), 6.0f);
            drawList->AddRect(canvasMin, canvasMax, ColorU32(126, 132, 117), 6.0f, 0, 1.5f);

            const double minX = zoneMap.m_minX;
            const double minY = zoneMap.m_minY;
            const double width = (zoneMap.m_maxX - zoneMap.m_minX) > 1.0 ? (zoneMap.m_maxX - zoneMap.m_minX) : 1.0;
            const double height = (zoneMap.m_maxY - zoneMap.m_minY) > 1.0 ? (zoneMap.m_maxY - zoneMap.m_minY) : 1.0;
            auto mapToScreen = [&](double x, double y) -> ImVec2
            {
                const float normalizedX = static_cast<float>((x - minX) / width);
                const float normalizedY = static_cast<float>((y - minY) / height);
                return ImVec2(
                    canvasMin.x + 18.0f + (normalizedX * (canvasSize.x - 36.0f)),
                    canvasMax.y - 18.0f - (normalizedY * (canvasSize.y - 36.0f)));
            };

            for (const auto& road : zoneMap.m_roads)
            {
                for (size_t pointIndex = 1; pointIndex < road.m_points.size(); ++pointIndex)
                {
                    const ImVec2 previous = mapToScreen(road.m_points[pointIndex - 1].m_x, road.m_points[pointIndex - 1].m_y);
                    const ImVec2 current = mapToScreen(road.m_points[pointIndex].m_x, road.m_points[pointIndex].m_y);
                    drawList->AddLine(previous, current, ColorU32(136, 112, 70), 4.0f);
                    drawList->AddLine(previous, current, ColorU32(196, 169, 103), 1.5f);
                }
            }

            for (const auto& area : worldState.m_session.m_navigationAreas)
            {
                const ImVec2 center = mapToScreen(area.m_centerX, area.m_centerY);
                const float radius = static_cast<float>((area.m_radius / width) * (canvasSize.x - 36.0f));
                const ImU32 areaColor = area.m_kind == "hostile_objective" ? ColorU32(135, 74, 52, 65) : ColorU32(72, 121, 112, 55);
                drawList->AddCircleFilled(center, AZ::GetClamp(radius, 8.0f, 42.0f), areaColor, 32);
                drawList->AddCircle(center, AZ::GetClamp(radius, 8.0f, 42.0f), ColorU32(178, 166, 124, 130), 32, 1.0f);
            }

            for (const auto& landmark : zoneMap.m_landmarks)
            {
                const ImVec2 point = mapToScreen(landmark.m_x, landmark.m_y);
                drawList->AddCircleFilled(point, 3.5f, ColorU32(199, 211, 194), 16);
                drawList->AddText(ImVec2(point.x + 5.0f, point.y - 7.0f), ColorU32(223, 219, 199), landmark.m_displayName.c_str());
            }

            for (const auto& marker : worldState.m_session.m_mapMarkers)
            {
                const ImVec2 point = mapToScreen(marker.m_x, marker.m_y);
                drawList->AddCircleFilled(point, 6.0f, MapMarkerColor(marker.m_kind), 20);
                drawList->AddCircle(point, 7.5f, ColorU32(21, 25, 27), 20, 1.5f);
                if (!marker.m_displayName.empty())
                {
                    drawList->AddText(ImVec2(point.x + 8.0f, point.y - 8.0f), ColorU32(240, 232, 206), marker.m_displayName.c_str());
                }
            }

            const ImVec2 playerPoint = mapToScreen(playerX, playerY);
            drawList->AddTriangleFilled(
                ImVec2(playerPoint.x, playerPoint.y - 9.0f),
                ImVec2(playerPoint.x - 7.0f, playerPoint.y + 7.0f),
                ImVec2(playerPoint.x + 7.0f, playerPoint.y + 7.0f),
                ColorU32(226, 240, 243));

            ImGui::TextUnformatted("Legend: gold available, green turn-in, rust objective, blue trainer, violet vendor");
        }

        void DrawFriendlyNpcNameplates(
            const GameCore::ClientWorldState& worldState,
            const GameCore::ClientCameraState& cameraState,
            const ImVec2& displaySize)
        {
            ImDrawList* drawList = ImGui::GetForegroundDrawList();
            for (const auto& entity : worldState.m_session.m_entities)
            {
                if (!entity.m_alive || !entity.m_targetable || !IsFriendlyNpc(entity))
                {
                    continue;
                }

                const float distanceToPlayer = Distance2D(
                    static_cast<float>(worldState.m_session.m_position.m_x),
                    static_cast<float>(worldState.m_session.m_position.m_y),
                    static_cast<float>(entity.m_x),
                    static_cast<float>(entity.m_y));
                const bool isSelected = entity.m_id == worldState.m_session.m_currentTargetId;
                if (!isSelected && distanceToPlayer > FriendlyNameplateDrawDistance)
                {
                    continue;
                }

                ImVec2 screenPosition{};
                if (!ProjectWorldPointToScreen(
                        cameraState,
                        AZ::Vector3(
                            static_cast<float>(entity.m_x),
                            static_cast<float>(entity.m_y),
                            static_cast<float>(entity.m_z) + GameCore::MobProxyGeometry::HeadHeight + 1.15f),
                        displaySize,
                        screenPosition))
                {
                    continue;
                }

                const bool isTrainer = IsTrainerNpc(entity);
                const AZStd::string label = AZStd::string::format(
                    "%s  %s",
                    isTrainer ? "Trainer" : "Quest",
                    entity.m_displayName.c_str());
                const ImVec2 textSize = ImGui::CalcTextSize(label.c_str());
                const ImVec2 panelMin(screenPosition.x - (textSize.x * 0.5f) - 10.0f, screenPosition.y - 12.0f);
                const ImVec2 panelMax(screenPosition.x + (textSize.x * 0.5f) + 10.0f, screenPosition.y + textSize.y + 6.0f);
                const ImU32 fillColor = isTrainer
                    ? ColorU32(20, 55, 98, isSelected ? 242 : 220)
                    : ColorU32(20, 79, 46, isSelected ? 242 : 220);
                const ImU32 borderColor = isSelected
                    ? ColorU32(248, 224, 128)
                    : (isTrainer ? ColorU32(101, 174, 238) : ColorU32(98, 224, 150));

                drawList->AddRectFilled(panelMin, panelMax, fillColor, 10.0f);
                drawList->AddRect(panelMin, panelMax, borderColor, 10.0f, 0, 2.0f);
                drawList->AddText(ImVec2(panelMin.x + 10.0f, panelMin.y + 4.0f), ColorU32(244, 235, 214), label.c_str());
            }
        }

        void DrawQuestTracker(
            const GameCore::ClientWorldState& worldState,
            bool nearCommandPoint,
            float distanceToCommandPoint,
            float distanceToEncounter)
        {
            const NetClient::QuestState* trackerQuest = &worldState.m_session.m_quest;
            for (const auto& quest : worldState.m_session.m_quests)
            {
                if (quest.m_tracked)
                {
                    trackerQuest = &quest;
                    break;
                }
            }

            ImGui::TextUnformatted("Stonewake Orders");
            ImGui::Separator();
            const AZStd::string trackerTitle = GetQuestDisplayTitle(*trackerQuest);
            ImGui::TextUnformatted(trackerTitle.c_str());
            if (trackerQuest->m_groupRecommended)
            {
                ImGui::PushStyleColor(ImGuiCol_Text, ImVec4(0.93f, 0.70f, 0.33f, 1.0f));
                if (trackerQuest->m_recommendedPlayers > 0)
                {
                    ImGui::Text("Recommended group: %d players", trackerQuest->m_recommendedPlayers);
                }
                else
                {
                    ImGui::TextUnformatted("Recommended group");
                }
                ImGui::PopStyleColor();
            }

            if (trackerQuest->m_state == "not_started")
            {
                ImGui::TextWrapped("%s", trackerQuest->m_objectiveText.c_str());
                ImGui::Spacing();
                if (nearCommandPoint)
                {
                    ImGui::TextWrapped("Find the highlighted Stonewake quest giver, left-click to target, then right-click the NPC model to review orders.");
                }
                else
                {
                    ImGui::PushStyleColor(ImGuiCol_Text, ImVec4(0.93f, 0.70f, 0.33f, 1.0f));
                    ImGui::TextWrapped("Find Commander Elian Rook in Hearthwatch Yard. Distance %.1fm.", distanceToCommandPoint);
                    ImGui::PopStyleColor();
                }
            }
            else if (trackerQuest->m_state == "active")
            {
                ImGui::Text(
                    "%s: %d / %d",
                    trackerQuest->m_objectiveText.c_str(),
                    trackerQuest->m_currentCount,
                    trackerQuest->m_targetCount);
                const AZStd::string routeHint = trackerQuest->m_routeHintText.empty()
                    ? AZStd::string::format("Follow the Stonewake route markers. Next landmark distance %.1fm.", distanceToEncounter)
                    : trackerQuest->m_routeHintText;
                ImGui::TextWrapped("%s", routeHint.c_str());
                ImGui::Text(
                    "Reward: %d XP and %dg %ds %dc",
                    trackerQuest->m_rewardXp,
                    trackerQuest->m_rewardCurrencyGold,
                    trackerQuest->m_rewardCurrencySilver,
                    trackerQuest->m_rewardCurrencyCopper);
            }
            else if (trackerQuest->m_state == "completed")
            {
                ImGui::TextWrapped("Objective complete. Return to the listed turn-in NPC and right-click to claim the reward.");
                ImGui::Text(
                    "Reward: %d XP and %dg %ds %dc",
                    trackerQuest->m_rewardXp,
                    trackerQuest->m_rewardCurrencyGold,
                    trackerQuest->m_rewardCurrencySilver,
                    trackerQuest->m_rewardCurrencyCopper);
            }
            else if (trackerQuest->m_state == "reward_granted")
            {
                ImGui::TextWrapped("Completed and persisted.");
                ImGui::Text(
                    "Rewards: +%d XP  |  +%dg %ds %dc",
                    trackerQuest->m_rewardXp,
                    trackerQuest->m_rewardCurrencyGold,
                    trackerQuest->m_rewardCurrencySilver,
                    trackerQuest->m_rewardCurrencyCopper);
                ImGui::TextUnformatted("Reconnects and restarts preserve this state.");
            }

            ImGui::Spacing();
            ImGui::Separator();
            ImGui::TextUnformatted("Warrior Trainer");
            if (!worldState.m_session.m_trainer.m_displayName.empty())
            {
                ImGui::TextUnformatted(worldState.m_session.m_trainer.m_displayName.c_str());
            }
            if (nearCommandPoint)
            {
                ImGui::PushStyleColor(ImGuiCol_Text, ImVec4(0.74f, 0.88f, 0.66f, 1.0f));
                ImGui::TextWrapped("Find the blue-gold trainer NPC, left-click to target, then right-click the NPC model to train.");
                ImGui::PopStyleColor();
            }
            else
            {
                ImGui::PushStyleColor(ImGuiCol_Text, ImVec4(0.93f, 0.70f, 0.33f, 1.0f));
                ImGui::TextWrapped("Find the blue-gold trainer NPC near the service markers. Distance %.1fm.", distanceToCommandPoint);
                ImGui::PopStyleColor();
            }

            ImGui::Spacing();
            ImGui::Separator();
            ImGui::TextUnformatted("Objectives");
            for (const auto& entity : worldState.m_session.m_entities)
            {
                if (entity.m_kind != "hostile_mob")
                {
                    continue;
                }
                ImGui::Text(
                    "%s  %s  %.0f/%.0f",
                    entity.m_id == worldState.m_session.m_currentTargetId ? ">" : " ",
                    GetMobDisplayLabel(entity).c_str(),
                    entity.m_health,
                    entity.m_maxHealth);
            }

            ImGui::Spacing();
            ImGui::Separator();
            ImGui::TextUnformatted("Controls");
            ImGui::TextWrapped("WASD move  |  RMB orbit  |  Tab hostiles  |  LMB target");
            ImGui::TextWrapped("Right-click friendly NPC model to interact  |  B bag  |  P spellbook  |  ESC menu");
            ImGui::TextWrapped("Hold SHIFT to drag/click spellbook abilities onto bars, move buttons, or clear slots.");
            ImGui::TextWrapped("F auto-attack  |  1 strike  |  2 brace  |  X disconnect  |  R reconnect");
        }

        void DrawEventLog(const AZStd::deque<AZStd::string>& eventLog)
        {
            ImGui::TextUnformatted("Field Log");
            ImGui::Separator();
            if (eventLog.empty())
            {
                ImGui::TextUnformatted("World events will appear here.");
            }
            else
            {
                for (const auto& entry : eventLog)
                {
                    ImGui::BulletText("%s", entry.c_str());
                }
            }
            ImGui::Spacing();
            ImGui::TextUnformatted("System feed only");
        }

        void DrawActionSlots(
            GameCore::IGameCoreRequests* gameCore,
            const GameCore::ClientWorldState& worldState,
            bool hasHostileTarget,
            const AZStd::array<AZStd::string, ActionBarSlotCount>& actionSlotBindings,
            int firstSlot,
            int slotCount,
            bool vertical,
            bool editMode,
            AZStd::string& pendingActionAssignmentAbilityId,
            int& pendingActionMoveSlot)
        {
            ImDrawList* drawList = ImGui::GetWindowDrawList();
            const ImVec2 slotSize(52.0f, 52.0f);
            for (int localSlotIndex = 0; localSlotIndex < slotCount; ++localSlotIndex)
            {
                if (!vertical && localSlotIndex > 0)
                {
                    ImGui::SameLine();
                }

                const int slotIndex = firstSlot + localSlotIndex;
                ImGui::PushID(slotIndex);
                const NetClient::ActionBarSlotState* slotState = FindActionBarSlot(worldState.m_session, slotIndex);
                const bool hasAbility = slotState && slotState->m_learned && !slotState->m_abilityId.empty();
                const AZ::s64 nowMs = NowMs();
                AZ::s64 abilityCooldownRemainingMs = 0;
                if (slotState)
                {
                    abilityCooldownRemainingMs = slotState->m_cooldownRemainingMs;
                    if (slotState->m_cooldownEndsAt > nowMs)
                    {
                        abilityCooldownRemainingMs = AZ::GetMax(abilityCooldownRemainingMs, slotState->m_cooldownEndsAt - nowMs);
                    }
                }
                const AZ::s64 gcdRemainingMs =
                    slotState && slotState->m_triggersGlobalCooldown && worldState.m_session.m_globalCooldownEndsAt > nowMs
                    ? worldState.m_session.m_globalCooldownEndsAt - nowMs
                    : 0;
                const AZ::s64 blockedRemainingMs = AZ::GetMax(abilityCooldownRemainingMs, gcdRemainingMs);
                const bool blockedByCooldown = blockedRemainingMs > 0;
                const bool blockedByResource = hasAbility &&
                    slotState->m_resourceCost > 0.0 &&
                    worldState.m_session.m_resource + 0.01 < slotState->m_resourceCost;
                const bool clickable = hasAbility &&
                    (!slotState->m_requiresTarget || hasHostileTarget) &&
                    !blockedByCooldown &&
                    !blockedByResource;
                if (editMode)
                {
                    ImGui::PushStyleColor(ImGuiCol_Button, ImVec4(0.22f, 0.20f, 0.10f, 1.0f));
                    ImGui::PushStyleColor(ImGuiCol_ButtonHovered, ImVec4(0.32f, 0.28f, 0.13f, 1.0f));
                    ImGui::PushStyleColor(ImGuiCol_ButtonActive, ImVec4(0.40f, 0.34f, 0.16f, 1.0f));
                }
                else if (clickable)
                {
                    ImGui::PushStyleColor(ImGuiCol_Button, ImVec4(0.12f, 0.18f, 0.24f, 1.0f));
                    ImGui::PushStyleColor(ImGuiCol_ButtonHovered, ImVec4(0.18f, 0.27f, 0.33f, 1.0f));
                    ImGui::PushStyleColor(ImGuiCol_ButtonActive, ImVec4(0.22f, 0.31f, 0.39f, 1.0f));
                }
                else
                {
                    ImGui::PushStyleColor(ImGuiCol_Button, ImVec4(0.08f, 0.10f, 0.13f, 1.0f));
                    ImGui::PushStyleColor(ImGuiCol_ButtonHovered, ImVec4(0.08f, 0.10f, 0.13f, 1.0f));
                    ImGui::PushStyleColor(ImGuiCol_ButtonActive, ImVec4(0.08f, 0.10f, 0.13f, 1.0f));
                }

                const char* buttonLabel = (slotState && !slotState->m_buttonLabel.empty()) ? slotState->m_buttonLabel.c_str() : "";
                const bool pressed = ImGui::Button(buttonLabel, slotSize);
                ImGui::PopStyleColor(3);
                const ImVec2 slotMin = ImGui::GetItemRectMin();
                const ImVec2 slotMax = ImGui::GetItemRectMax();
                drawList->AddRect(slotMin, slotMax, ColorU32(153, 119, 56), 8.0f, 0, 2.0f);
                const AZStd::string keyLabel = slotIndex >= 0 && slotIndex < ActionBarSlotCount
                    ? DisplayKeyName(actionSlotBindings[slotIndex])
                    : AZStd::string{};
                if (!keyLabel.empty() && keyLabel != "Unbound")
                {
                    drawList->AddText(ImVec2(slotMin.x + 6.0f, slotMin.y + 4.0f), ColorU32(235, 211, 154), keyLabel.c_str());
                }
                if (hasAbility && blockedByCooldown)
                {
                    const float cooldownRatio = slotState->m_cooldownMs > 0
                        ? Clamp01(static_cast<float>(blockedRemainingMs) / static_cast<float>(slotState->m_cooldownMs))
                        : 1.0f;
                    const float overlayHeight = slotSize.y * cooldownRatio;
                    drawList->AddRectFilled(
                        ImVec2(slotMin.x, slotMax.y - overlayHeight),
                        slotMax,
                        ColorU32(8, 11, 15, 185),
                        8.0f);
                    const AZStd::string cooldownText = AZStd::string::format("%.1f", static_cast<double>(blockedRemainingMs) / 1000.0);
                    const ImVec2 cooldownTextSize = ImGui::CalcTextSize(cooldownText.c_str());
                    drawList->AddText(
                        ImVec2(
                            slotMin.x + ((slotSize.x - cooldownTextSize.x) * 0.5f),
                            slotMin.y + ((slotSize.y - cooldownTextSize.y) * 0.5f)),
                        ColorU32(246, 236, 204),
                        cooldownText.c_str());
                }
                else if (hasAbility && blockedByResource)
                {
                    drawList->AddRectFilled(slotMin, slotMax, ColorU32(15, 18, 24, 150), 8.0f);
                    drawList->AddText(ImVec2(slotMin.x + 34.0f, slotMax.y - 18.0f), ColorU32(112, 154, 214), "G");
                }

                if (!editMode && pressed && clickable)
                {
                    if (slotState->m_abilityId == AutoAttackAbilityId)
                    {
                        gameCore->SetAutoAttack(!worldState.m_session.m_autoAttackActive);
                    }
                    else
                    {
                        gameCore->ActivateAbility(slotState->m_abilityId);
                    }
                }
                else if (editMode && pressed)
                {
                    if (!pendingActionAssignmentAbilityId.empty())
                    {
                        if (gameCore->AssignActionBarSlot(slotIndex, pendingActionAssignmentAbilityId))
                        {
                            AZ_Printf(
                                "amandacore",
                                "client.action_bar_assignment_requested slot=%d abilityId=%s source=shift_click",
                                slotIndex,
                                pendingActionAssignmentAbilityId.c_str());
                            pendingActionAssignmentAbilityId.clear();
                        }
                    }
                    else if (pendingActionMoveSlot >= 0)
                    {
                        if (pendingActionMoveSlot != slotIndex && gameCore->MoveActionBarSlot(pendingActionMoveSlot, slotIndex))
                        {
                            AZ_Printf(
                                "amandacore",
                                "client.action_bar_move_requested fromSlot=%d toSlot=%d source=shift_click",
                                pendingActionMoveSlot,
                                slotIndex);
                        }
                        pendingActionMoveSlot = -1;
                    }
                    else if (hasAbility)
                    {
                        pendingActionMoveSlot = slotIndex;
                        AZ_Printf("amandacore", "client.action_bar_move_armed slot=%d", slotIndex);
                    }
                }

                if (editMode && hasAbility && ImGui::BeginDragDropSource(ImGuiDragDropFlags_SourceAllowNullID))
                {
                    ActionBarSlotDragPayload payload{};
                    payload.m_sourceSlotIndex = slotIndex;
                    std::snprintf(payload.m_abilityId, sizeof(payload.m_abilityId), "%s", slotState->m_abilityId.c_str());
                    ImGui::SetDragDropPayload(ActionBarSlotPayloadType, &payload, sizeof(payload));
                    ImGui::TextUnformatted(slotState->m_displayName.c_str());
                    ImGui::Text("Move from slot %d", slotIndex + 1);
                    ImGui::EndDragDropSource();
                }

                if (editMode && ImGui::BeginDragDropTarget())
                {
                    if (const ImGuiPayload* payload = ImGui::AcceptDragDropPayload(SpellbookAbilityPayloadType))
                    {
                        const auto* drag = static_cast<const SpellbookAbilityDragPayload*>(payload->Data);
                        if (drag && drag->m_abilityId[0] != '\0' && gameCore->AssignActionBarSlot(slotIndex, drag->m_abilityId))
                        {
                            AZ_Printf(
                                "amandacore",
                                "client.action_bar_assignment_requested slot=%d abilityId=%s",
                                slotIndex,
                                drag->m_abilityId);
                            pendingActionAssignmentAbilityId.clear();
                        }
                    }
                    if (const ImGuiPayload* payload = ImGui::AcceptDragDropPayload(ActionBarSlotPayloadType))
                    {
                        const auto* drag = static_cast<const ActionBarSlotDragPayload*>(payload->Data);
                        if (drag && drag->m_sourceSlotIndex >= 0 && drag->m_sourceSlotIndex != slotIndex &&
                            gameCore->MoveActionBarSlot(drag->m_sourceSlotIndex, slotIndex))
                        {
                            AZ_Printf(
                                "amandacore",
                                "client.action_bar_move_requested fromSlot=%d toSlot=%d abilityId=%s",
                                drag->m_sourceSlotIndex,
                                slotIndex,
                                drag->m_abilityId);
                            pendingActionMoveSlot = -1;
                        }
                    }
                    ImGui::EndDragDropTarget();
                }

                if (editMode && ImGui::IsItemClicked(ImGuiMouseButton_Right))
                {
                    if (hasAbility && gameCore->ClearActionBarSlot(slotIndex))
                    {
                        AZ_Printf("amandacore", "client.action_bar_clear_requested slot=%d", slotIndex);
                    }
                    if (pendingActionMoveSlot == slotIndex)
                    {
                        pendingActionMoveSlot = -1;
                    }
                    if (!pendingActionAssignmentAbilityId.empty())
                    {
                        pendingActionAssignmentAbilityId.clear();
                    }
                }

                if (slotState && !slotState->m_displayName.empty() && ImGui::IsItemHovered())
                {
                    AZStd::string tooltip = slotState->m_displayName;
                    const AZStd::string facts = FormatAbilityFacts(*slotState);
                    if (!facts.empty())
                    {
                        tooltip += "\n";
                        tooltip += facts;
                    }
                    if (!slotState->m_tooltipText.empty())
                    {
                        tooltip += "\n";
                        tooltip += slotState->m_tooltipText;
                    }
                    if (blockedByCooldown)
                    {
                        tooltip += AZStd::string::format("\nReady in %.1fs.", static_cast<double>(blockedRemainingMs) / 1000.0);
                    }
                    if (blockedByResource)
                    {
                        tooltip += "\nNot enough Grit.";
                    }
                    if (editMode)
                    {
                        tooltip += "\nSHIFT-click to arm move, drag to move, or SHIFT-right-click to clear.";
                    }
                    else
                    {
                        tooltip += "\nBars are locked. Hold SHIFT to move or clear abilities.";
                    }
                    ImGui::SetTooltip("%s", tooltip.c_str());
                }
                else if (editMode && ImGui::IsItemHovered())
                {
                    ImGui::SetTooltip("Drop a learned spellbook ability here, SHIFT-click after selecting a spell, or drag another action slot here.");
                }
                ImGui::PopID();
            }
        }

        void DrawActionBar(
            GameCore::IGameCoreRequests* gameCore,
            const GameCore::ClientWorldState& worldState,
            bool hasHostileTarget,
            const AZStd::array<AZStd::string, ActionBarSlotCount>& actionSlotBindings,
            const AZStd::string& spellbookBinding,
            const AZStd::string& bagBinding,
            const AZStd::string& settingsBinding,
            const AZStd::string& interactBinding,
            const AZStd::string& targetHostileBinding,
            bool editMode,
            AZStd::string& pendingActionAssignmentAbilityId,
            int& pendingActionMoveSlot)
        {
            ImGui::TextUnformatted("Action Deck");
            const float xpRatio = Clamp01(static_cast<float>(worldState.m_session.m_experience) / 100.0f);
            const ImVec2 xpBarPosition = ImGui::GetCursorScreenPos();
            const ImVec2 xpBarSize(690.0f, 10.0f);
            ImGui::InvisibleButton("##xp_bar", xpBarSize);
            ImDrawList* drawList = ImGui::GetWindowDrawList();
            drawList->AddRectFilled(xpBarPosition, AddVec2(xpBarPosition, xpBarSize), ColorU32(25, 29, 38, 255), 6.0f);
            drawList->AddRect(xpBarPosition, AddVec2(xpBarPosition, xpBarSize), ColorU32(110, 96, 61, 255), 6.0f, 0, 1.0f);
            drawList->AddRectFilled(
                xpBarPosition,
                ImVec2(xpBarPosition.x + (xpBarSize.x * xpRatio), xpBarPosition.y + xpBarSize.y),
                ColorU32(76, 78, 196),
                6.0f);
            drawList->AddText(ImVec2(xpBarPosition.x + 292.0f, xpBarPosition.y - 2.0f), ColorU32(230, 232, 241), AZStd::string::format("XP %d", worldState.m_session.m_experience).c_str());

            DrawActionSlots(
                gameCore,
                worldState,
                hasHostileTarget,
                actionSlotBindings,
                0,
                12,
                false,
                editMode,
                pendingActionAssignmentAbilityId,
                pendingActionMoveSlot);

            ImGui::Spacing();
            ImGui::TextWrapped(
                "%s",
                BuildActionBarHelpText(
                    worldState.m_session,
                    actionSlotBindings,
                    spellbookBinding,
                    bagBinding,
                    settingsBinding,
                    interactBinding,
                    targetHostileBinding)
                    .c_str());
            if (editMode)
            {
                ImGui::TextWrapped("Bar editing enabled: drag learned abilities from the spellbook, drag action buttons between slots, or right-click a slotted ability to clear it.");
                if (!pendingActionAssignmentAbilityId.empty())
                {
                    ImGui::Text("Selected spell: %s. Click any action slot to place it.", pendingActionAssignmentAbilityId.c_str());
                }
                else if (pendingActionMoveSlot >= 0)
                {
                    ImGui::Text("Moving slot %d. Click another action slot to place it.", pendingActionMoveSlot + 1);
                }
            }
            else
            {
                ImGui::TextWrapped("Bars are locked by default. Hold SHIFT to drag or clear abilities.");
            }
        }

        void DrawAuxiliaryActionBar(
            GameCore::IGameCoreRequests* gameCore,
            const GameCore::ClientWorldState& worldState,
            bool hasHostileTarget,
            const AZStd::array<AZStd::string, ActionBarSlotCount>& actionSlotBindings,
            int firstSlot,
            bool vertical,
            bool editMode,
            AZStd::string& pendingActionAssignmentAbilityId,
            int& pendingActionMoveSlot)
        {
            DrawActionSlots(
                gameCore,
                worldState,
                hasHostileTarget,
                actionSlotBindings,
                firstSlot,
                12,
                vertical,
                editMode,
                pendingActionAssignmentAbilityId,
                pendingActionMoveSlot);
        }

        void DrawSpellbook(
            const GameCore::ClientWorldState& worldState,
            bool editMode,
            AZStd::string& pendingActionAssignmentAbilityId)
        {
            ImGui::TextUnformatted("Warrior Codex");
            ImGui::SameLine();
            ImGui::TextDisabled("  |  class abilities and training previews");
            ImGui::Separator();
            ImGui::TextWrapped("Learned abilities are assigned from here. Hold SHIFT, then drag a learned ability onto any action bar slot.");
            if (editMode && !pendingActionAssignmentAbilityId.empty())
            {
                ImGui::Text("Selected for action bar: %s", pendingActionAssignmentAbilityId.c_str());
            }
            else if (!editMode)
            {
                ImGui::TextWrapped("Bars are locked. Hold SHIFT to assign abilities.");
            }
            ImGui::Spacing();
            ImGui::BeginChild(
                "##spellbook_scroll",
                ImVec2(0.0f, 0.0f),
                false,
                ImGuiWindowFlags_AlwaysVerticalScrollbar);
            ImGui::Columns(2, "##spellbook_pages", false);
            ImGui::TextUnformatted("Learned");
            ImGui::Separator();
            for (const auto& entry : worldState.m_session.m_spellbookEntries)
            {
                if (!entry.m_learned)
                {
                    continue;
                }

                ImGui::PushID(entry.m_id.c_str());
                const ImVec2 cardStart = ImGui::GetCursorScreenPos();
                ImGui::InvisibleButton("##spell_card_icon", ImVec2(42.0f, 42.0f));
                ImGui::GetWindowDrawList()->AddRectFilled(cardStart, AddVec2(cardStart, ImVec2(42.0f, 42.0f)), ColorU32(36, 72, 88), 7.0f);
                ImGui::GetWindowDrawList()->AddRect(cardStart, AddVec2(cardStart, ImVec2(42.0f, 42.0f)), ColorU32(187, 143, 65), 7.0f, 0, 2.0f);
                ImGui::GetWindowDrawList()->AddText(
                    ImVec2(cardStart.x + 14.0f, cardStart.y + 12.0f),
                    ColorU32(240, 230, 198),
                    entry.m_displayName.empty() ? "?" : AZStd::string::format("%c", entry.m_displayName.front()).c_str());
                ImGui::SameLine();
                ImGui::BeginGroup();
                ImGui::PushStyleColor(ImGuiCol_Text, ImVec4(0.86f, 0.90f, 0.76f, 1.0f));
                const bool selected = pendingActionAssignmentAbilityId == entry.m_id;
                if (ImGui::Selectable(entry.m_displayName.c_str(), selected, ImGuiSelectableFlags_SpanAvailWidth) && editMode)
                {
                    pendingActionAssignmentAbilityId = entry.m_id;
                    AZ_Printf("amandacore", "client.spellbook_assignment_armed abilityId=%s", entry.m_id.c_str());
                }
                ImGui::PopStyleColor();
                ImGui::TextWrapped("%s", entry.m_description.c_str());
                const AZStd::string learnedFacts = FormatAbilityFacts(entry);
                if (!learnedFacts.empty())
                {
                    ImGui::TextWrapped("%s", learnedFacts.c_str());
                }
                if (!entry.m_tooltipText.empty())
                {
                    ImGui::TextDisabled("%s", entry.m_tooltipText.c_str());
                }
                ImGui::TextDisabled("Known  |  level %d", entry.m_requiredLevel);
                ImGui::EndGroup();
                if (editMode && ImGui::BeginDragDropSource(ImGuiDragDropFlags_SourceAllowNullID))
                {
                    SpellbookAbilityDragPayload payload{};
                    std::snprintf(payload.m_abilityId, sizeof(payload.m_abilityId), "%s", entry.m_id.c_str());
                    ImGui::SetDragDropPayload(SpellbookAbilityPayloadType, &payload, sizeof(payload));
                    ImGui::TextUnformatted(entry.m_displayName.c_str());
                    ImGui::TextUnformatted("Assign to action bar");
                    ImGui::EndDragDropSource();
                }
                if (ImGui::IsItemHovered())
                {
                    ImGui::SetTooltip(editMode ? "Drag to an action slot, or click to select then click a slot." : "Hold SHIFT to assign this ability.");
                }
                ImGui::Separator();
                ImGui::PopID();
            }

            ImGui::NextColumn();
            ImGui::TextUnformatted("Training Preview");
            ImGui::Separator();
            for (const auto& entry : worldState.m_session.m_spellbookEntries)
            {
                if (entry.m_learned)
                {
                    continue;
                }

                ImGui::PushID(entry.m_id.c_str());
                const ImVec2 cardStart = ImGui::GetCursorScreenPos();
                ImGui::InvisibleButton("##locked_spell_icon", ImVec2(42.0f, 42.0f));
                ImGui::GetWindowDrawList()->AddRectFilled(cardStart, AddVec2(cardStart, ImVec2(42.0f, 42.0f)), ColorU32(54, 47, 39), 7.0f);
                ImGui::GetWindowDrawList()->AddRect(cardStart, AddVec2(cardStart, ImVec2(42.0f, 42.0f)), ColorU32(121, 91, 54), 7.0f, 0, 2.0f);
                ImGui::GetWindowDrawList()->AddText(ImVec2(cardStart.x + 16.0f, cardStart.y + 12.0f), ColorU32(181, 151, 92), "?");
                ImGui::SameLine();
                ImGui::BeginGroup();
                ImGui::PushStyleColor(ImGuiCol_Text, ImVec4(0.80f, 0.66f, 0.38f, 1.0f));
                ImGui::Text("%s", entry.m_displayName.c_str());
                ImGui::PopStyleColor();
                ImGui::TextWrapped("%s", entry.m_description.c_str());
                const AZStd::string previewFacts = FormatAbilityFacts(entry);
                if (!previewFacts.empty())
                {
                    ImGui::TextWrapped("%s", previewFacts.c_str());
                }
                if (!entry.m_tooltipText.empty())
                {
                    ImGui::TextDisabled("%s", entry.m_tooltipText.c_str());
                }
                ImGui::TextWrapped("%s", entry.m_requirementText.c_str());
                ImGui::EndGroup();
                ImGui::Separator();
                ImGui::PopID();
            }
            ImGui::Columns(1);
            ImGui::EndChild();
        }

        void DrawTrainerWindow(GameCore::IGameCoreRequests* gameCore, const GameCore::ClientWorldState& worldState)
        {
            const auto& trainer = worldState.m_session.m_trainer;
            if (trainer.m_displayName.empty())
            {
                ImGui::TextUnformatted("No trainer is available in this slice yet.");
                return;
            }

            ImGui::TextUnformatted(trainer.m_displayName.c_str());
            ImGui::Separator();
            if (!trainer.m_interactionHint.empty())
            {
                ImGui::TextWrapped("%s", trainer.m_interactionHint.c_str());
            }
            ImGui::TextUnformatted(trainer.m_inRange ? "Status: ready to train" : "Status: move closer to the trainer");
            ImGui::Spacing();

            ImGui::BeginChild(
                "##trainer_offer_scroll",
                ImVec2(0.0f, 0.0f),
                false,
                ImGuiWindowFlags_AlwaysVerticalScrollbar);
            for (const auto& offer : trainer.m_offers)
            {
                ImGui::PushID(offer.m_abilityId.c_str());
                ImGui::Text("%s", offer.m_displayName.c_str());
                ImGui::TextWrapped("%s", offer.m_description.c_str());
                const AZStd::string offerFacts = FormatAbilityFacts(offer);
                if (!offerFacts.empty())
                {
                    ImGui::TextWrapped("%s", offerFacts.c_str());
                }
                if (!offer.m_tooltipText.empty())
                {
                    ImGui::TextDisabled("%s", offer.m_tooltipText.c_str());
                }
                ImGui::Text("Cost: %s  |  Level %d", FormatTrainerCost(offer.m_costCopper).c_str(), offer.m_requiredLevel);

                if (offer.m_learned)
                {
                    ImGui::PushStyleColor(ImGuiCol_Text, ImVec4(0.74f, 0.88f, 0.66f, 1.0f));
                    ImGui::TextUnformatted("Already learned");
                    ImGui::PopStyleColor();
                }
                else if (offer.m_canLearn)
                {
                    if (ImGui::Button("Learn", ImVec2(92.0f, 0.0f)))
                    {
                        gameCore->LearnTrainerAbility(trainer.m_id, offer.m_abilityId);
                    }
                }
                else
                {
                    ImGui::PushStyleColor(ImGuiCol_Text, ImVec4(0.88f, 0.72f, 0.44f, 1.0f));
                    ImGui::TextWrapped("%s", offer.m_requirementText.c_str());
                    ImGui::PopStyleColor();
                }

                if (!offer.m_learned && offer.m_canLearn)
                {
                    ImGui::TextWrapped("%s", offer.m_requirementText.c_str());
                }
                ImGui::Separator();
                ImGui::PopID();
            }
            ImGui::EndChild();
        }

        void DrawQuestGossipWindow(GameCore::IGameCoreRequests* gameCore, const GameCore::ClientWorldState& worldState)
        {
            const auto* targetEntity = FindTargetEntity(worldState);
            ImGui::TextUnformatted(targetEntity ? targetEntity->m_displayName.c_str() : "Quest Giver");
            ImGui::Separator();
            ImGui::TextUnformatted(worldState.m_session.m_quest.m_title.c_str());
            ImGui::TextWrapped("%s", worldState.m_session.m_quest.m_objectiveText.c_str());
            ImGui::Spacing();

            if (worldState.m_session.m_quest.m_state == "not_started")
            {
                ImGui::TextWrapped("This Stonewake order is ready. Accept it here, then follow the objective text in your tracker.");
                if (ImGui::Button("Accept Quest", ImVec2(190.0f, 32.0f)))
                {
                    gameCore->AcceptQuest(worldState.m_session.m_quest.m_id);
                }
            }
            else if (worldState.m_session.m_quest.m_state == "active")
            {
                ImGui::Text(
                    "Progress: %d / %d",
                    worldState.m_session.m_quest.m_currentCount,
                    worldState.m_session.m_quest.m_targetCount);
                ImGui::TextWrapped("If this NPC or object completes the objective, continue here. Otherwise complete the field objective first.");
                if (ImGui::Button("Continue Quest", ImVec2(190.0f, 32.0f)))
                {
                    gameCore->AcceptQuest(worldState.m_session.m_quest.m_id);
                }
            }
            else if (worldState.m_session.m_quest.m_state == "completed")
            {
                ImGui::TextWrapped("The objective is complete. Claim the reward from the turn-in NPC.");
                if (ImGui::Button("Complete Quest", ImVec2(190.0f, 32.0f)))
                {
                    gameCore->AcceptQuest(worldState.m_session.m_quest.m_id);
                }
            }
            else if (worldState.m_session.m_quest.m_state == "reward_granted")
            {
                ImGui::TextWrapped("Field orders are complete and persisted.");
                ImGui::Text(
                    "Reward: %d XP and %dg %ds %dc",
                    worldState.m_session.m_quest.m_rewardXp,
                    worldState.m_session.m_quest.m_rewardCurrencyGold,
                    worldState.m_session.m_quest.m_rewardCurrencySilver,
                    worldState.m_session.m_quest.m_rewardCurrencyCopper);
            }
            else
            {
                ImGui::TextWrapped("No field orders are available from this NPC right now.");
            }
        }

        void DrawUtilityFooter(const GameCore::ClientWorldState& worldState, bool bagOpen)
        {
            ImGui::TextUnformatted("Pack and Purse");
            ImGui::Separator();
            ImGui::Text("Currency  %s", FormatCurrency(worldState.m_session.m_currency).c_str());
            ImGui::Text("Inventory %d / %d", CountOccupiedSlots(worldState.m_session.m_inventory), worldState.m_session.m_inventory.m_slotCount);
            ImGui::TextUnformatted(bagOpen ? "Bag open [B]" : "Bag closed [B]");
        }

        void DrawInventoryWindow(GameCore::IGameCoreRequests* gameCore, const GameCore::ClientWorldState& worldState, int& pendingInventoryMoveSlot)
        {
            const int slotCount = worldState.m_session.m_inventory.m_slotCount > 0
                ? worldState.m_session.m_inventory.m_slotCount
                : static_cast<int>(worldState.m_session.m_inventory.m_slots.size());

            ImGui::Text("Frontier Pack  |  %d / %d occupied", CountOccupiedSlots(worldState.m_session.m_inventory), slotCount);
            ImGui::Text("Currency  %s", FormatCurrency(worldState.m_session.m_currency).c_str());
            ImGui::TextUnformatted("Drag items between slots to rearrange your pack. Click an item, then click another slot as a fallback.");
            if (pendingInventoryMoveSlot >= 0)
            {
                ImGui::Text("Moving item from slot %02d. Click a destination slot.", pendingInventoryMoveSlot + 1);
            }
            ImGui::Separator();

            const ImVec2 slotSize(66.0f, 66.0f);
            for (int slotIndex = 0; slotIndex < slotCount; ++slotIndex)
            {
                if (slotIndex > 0 && (slotIndex % 4) != 0)
                {
                    ImGui::SameLine();
                }

                const NetClient::InventorySlotState* slotState = nullptr;
                if (slotIndex < static_cast<int>(worldState.m_session.m_inventory.m_slots.size()))
                {
                    slotState = &worldState.m_session.m_inventory.m_slots[slotIndex];
                }

                ImGui::PushID(slotIndex);
                ImGui::PushStyleColor(ImGuiCol_Button, ImVec4(0.10f, 0.14f, 0.18f, 1.0f));
                ImGui::PushStyleColor(ImGuiCol_ButtonHovered, ImVec4(0.14f, 0.20f, 0.25f, 1.0f));
                ImGui::PushStyleColor(ImGuiCol_ButtonActive, ImVec4(0.16f, 0.23f, 0.29f, 1.0f));
                const AZStd::string slotLabel = slotState ? GetInventorySlotLabel(*slotState) : AZStd::string{};
                const bool pressed = ImGui::Button(slotLabel.c_str(), slotSize);
                ImGui::PopStyleColor(3);
                const ImVec2 slotMin = ImGui::GetItemRectMin();
                const ImVec2 slotMax = ImGui::GetItemRectMax();
                ImGui::GetWindowDrawList()->AddRect(slotMin, slotMax, ColorU32(149, 118, 59), 8.0f, 0, 2.0f);
                ImGui::GetWindowDrawList()->AddText(
                    ImVec2(slotMin.x + 6.0f, slotMin.y + 6.0f),
                    ColorU32(219, 201, 168),
                    AZStd::string::format("%02d", slotIndex + 1).c_str());
                if (slotState && !slotState->m_itemId.empty() && slotState->m_stackCount > 0 &&
                    ImGui::BeginDragDropSource(ImGuiDragDropFlags_SourceAllowNullID))
                {
                    InventorySlotDragPayload payload{};
                    payload.m_sourceSlotIndex = slotIndex;
                    ImGui::SetDragDropPayload(InventorySlotPayloadType, &payload, sizeof(payload));
                    ImGui::Text("%s x%d", slotState->m_displayName.c_str(), slotState->m_stackCount);
                    ImGui::Text("Move from slot %d", slotIndex + 1);
                    ImGui::EndDragDropSource();
                }
                if (ImGui::BeginDragDropTarget())
                {
                    if (const ImGuiPayload* payload = ImGui::AcceptDragDropPayload(InventorySlotPayloadType))
                    {
                        const auto* drag = static_cast<const InventorySlotDragPayload*>(payload->Data);
                        if (drag && drag->m_sourceSlotIndex >= 0 && drag->m_sourceSlotIndex != slotIndex &&
                            gameCore && gameCore->MoveInventorySlot(drag->m_sourceSlotIndex, slotIndex))
                        {
                            AZ_Printf(
                                "amandacore",
                                "client.inventory_move_requested fromSlot=%d toSlot=%d",
                                drag->m_sourceSlotIndex,
                                slotIndex);
                            pendingInventoryMoveSlot = -1;
                        }
                    }
                    ImGui::EndDragDropTarget();
                }
                if (pressed)
                {
                    if (pendingInventoryMoveSlot >= 0)
                    {
                        if (pendingInventoryMoveSlot != slotIndex &&
                            gameCore && gameCore->MoveInventorySlot(pendingInventoryMoveSlot, slotIndex))
                        {
                            AZ_Printf(
                                "amandacore",
                                "client.inventory_move_requested fromSlot=%d toSlot=%d source=click",
                                pendingInventoryMoveSlot,
                                slotIndex);
                        }
                        pendingInventoryMoveSlot = -1;
                    }
                    else if (slotState && !slotState->m_itemId.empty() && slotState->m_stackCount > 0)
                    {
                        pendingInventoryMoveSlot = slotIndex;
                        AZ_Printf("amandacore", "client.inventory_move_armed fromSlot=%d", slotIndex);
                    }
                }
                if (ImGui::IsItemClicked(ImGuiMouseButton_Right))
                {
                    pendingInventoryMoveSlot = -1;
                }
                if (slotState && !slotState->m_itemId.empty() && slotState->m_stackCount > 0 && ImGui::IsItemHovered())
                {
                    ImGui::SetTooltip("%s x%d\nDrag or click, then click a destination slot.", slotState->m_displayName.c_str(), slotState->m_stackCount);
                }
                ImGui::PopID();
            }
        }

        bool DrawKeyBindingRow(
            const char* label,
            const char* actionId,
            AZStd::string& binding,
            AZStd::string& pendingKeybindActionId)
        {
            bool changed = false;
            ImGui::PushID(actionId);
            ImGui::TextUnformatted(label);
            ImGui::SameLine(250.0f);
            const AZStd::string buttonLabel = pendingKeybindActionId == actionId
                ? "Press a key..."
                : DisplayKeyName(binding);
            if (ImGui::Button(buttonLabel.c_str(), ImVec2(132.0f, 0.0f)))
            {
                pendingKeybindActionId = actionId;
            }
            ImGui::SameLine();
            if (ImGui::Button("Unbind", ImVec2(74.0f, 0.0f)))
            {
                binding.clear();
                if (pendingKeybindActionId == actionId)
                {
                    pendingKeybindActionId.clear();
                }
                changed = true;
            }
            ImGui::PopID();
            return changed;
        }

        bool DrawSettingsWindow(
            GameCore::IGameCoreRequests* gameCore,
            bool& upperBarVisible,
            bool& rightBarOneVisible,
            bool& rightBarTwoVisible,
            AZStd::array<AZStd::string, ActionBarSlotCount>& actionSlotBindings,
            AZStd::string& spellbookBinding,
            AZStd::string& bagBinding,
            AZStd::string& characterBinding,
            AZStd::string& questLogBinding,
            AZStd::string& mapBinding,
            AZStd::string& settingsBinding,
            AZStd::string& interactBinding,
            AZStd::string& targetHostileBinding,
            AZStd::string& pendingKeybindActionId)
        {
            bool changed = false;
            if (ImGui::BeginTabBar("##settings_tabs"))
            {
                if (ImGui::BeginTabItem("Interface"))
                {
                    ImGui::TextUnformatted("Action Bars");
                    changed |= ImGui::Checkbox("Upper horizontal action bar", &upperBarVisible);
                    changed |= ImGui::Checkbox("Right action bar 1", &rightBarOneVisible);
                    changed |= ImGui::Checkbox("Right action bar 2", &rightBarTwoVisible);
                    ImGui::Spacing();
                    ImGui::TextWrapped("These bars use the same live action-slot payload as the main bar.");
                    ImGui::TextWrapped("Bars stay locked during normal play. Hold SHIFT to drag learned abilities from the spellbook, move existing buttons, or clear slots.");
                    ImGui::Separator();
                    if (ImGui::TreeNodeEx("Keybindings", ImGuiTreeNodeFlags_DefaultOpen))
                    {
                        ImGui::TextWrapped("Click a binding, then press a replacement key. Rebinding a key removes it from any conflicting action.");
                        if (!pendingKeybindActionId.empty())
                        {
                            ImGui::PushStyleColor(ImGuiCol_Text, ImVec4(0.92f, 0.76f, 0.45f, 1.0f));
                            ImGui::Text("Waiting for key: %s", pendingKeybindActionId.c_str());
                            ImGui::PopStyleColor();
                        }
                        changed |= DrawKeyBindingRow("Spellbook", "spellbook", spellbookBinding, pendingKeybindActionId);
                        changed |= DrawKeyBindingRow("Bag", "bag", bagBinding, pendingKeybindActionId);
                        changed |= DrawKeyBindingRow("Character", "character", characterBinding, pendingKeybindActionId);
                        changed |= DrawKeyBindingRow("Quest Log", "questLog", questLogBinding, pendingKeybindActionId);
                        changed |= DrawKeyBindingRow("Map", "map", mapBinding, pendingKeybindActionId);
                        changed |= DrawKeyBindingRow("Settings/Menu", "settings", settingsBinding, pendingKeybindActionId);
                        changed |= DrawKeyBindingRow("Interact", "interact", interactBinding, pendingKeybindActionId);
                        changed |= DrawKeyBindingRow("Target Hostile", "targetHostile", targetHostileBinding, pendingKeybindActionId);
                        ImGui::Separator();
                        if (ImGui::TreeNodeEx("Action Bar Slots", ImGuiTreeNodeFlags_DefaultOpen))
                        {
                            ImGui::BeginChild("##action_keybind_scroll", ImVec2(0.0f, 170.0f), false, ImGuiWindowFlags_AlwaysVerticalScrollbar);
                            for (int slotIndex = 0; slotIndex < ActionBarSlotCount; ++slotIndex)
                            {
                                const char* barName = slotIndex < 12
                                    ? "Main"
                                    : slotIndex < 24
                                        ? "Upper"
                                        : slotIndex < 36
                                            ? "Right 1"
                                            : "Right 2";
                                const AZStd::string label = AZStd::string::format("%s slot %02d", barName, (slotIndex % 12) + 1);
                                const AZStd::string actionId = SlotActionId(slotIndex);
                                changed |= DrawKeyBindingRow(label.c_str(), actionId.c_str(), actionSlotBindings[slotIndex], pendingKeybindActionId);
                            }
                            ImGui::EndChild();
                            ImGui::TreePop();
                        }
                        ImGui::TreePop();
                    }
                    ImGui::EndTabItem();
                }
                if (ImGui::BeginTabItem("Video"))
                {
                    ImGui::TextUnformatted("Video Settings");
                    ImGui::Separator();
                    ImGui::PushStyleColor(ImGuiCol_Text, ImVec4(0.92f, 0.76f, 0.45f, 1.0f));
                    ImGui::TextUnformatted("Unavailable in this slice");
                    ImGui::PopStyleColor();
                    ImGui::TextWrapped("No safe engine-backed video toggles are exposed in this milestone yet.");
                    ImGui::TextWrapped("Resolution, fullscreen, and graphics presets are intentionally deferred instead of shown as fake controls.");
                    ImGui::EndTabItem();
                }
                if (ImGui::BeginTabItem("Sound"))
                {
                    ImGui::TextUnformatted("Sound Settings");
                    ImGui::Separator();
                    ImGui::PushStyleColor(ImGuiCol_Text, ImVec4(0.92f, 0.76f, 0.45f, 1.0f));
                    ImGui::TextUnformatted("Unavailable in this slice");
                    ImGui::PopStyleColor();
                    ImGui::TextWrapped("No live sound bus settings are wired in this slice yet.");
                    ImGui::TextWrapped("Volume controls are deferred until the audio path is active.");
                    ImGui::EndTabItem();
                }
                ImGui::EndTabBar();
            }
            ImGui::Spacing();
            ImGui::Separator();
            if (ImGui::Button("Logout to Character Screen", ImVec2(230.0f, 30.0f)))
            {
                if (gameCore)
                {
                    gameCore->DisconnectWorld();
                }
                AZ_Printf("amandacore", "client.logout_to_character_screen_requested");
                AzFramework::ApplicationRequests::Bus::Broadcast(
                    &AzFramework::ApplicationRequests::Bus::Events::ExitMainLoop);
            }
            ImGui::SameLine();
            if (ImGui::Button("Exit Game", ImVec2(120.0f, 30.0f)))
            {
                AZ_Printf("amandacore", "client.exit_game_requested");
                AzFramework::ApplicationRequests::Bus::Broadcast(
                    &AzFramework::ApplicationRequests::Bus::Events::ExitMainLoop);
            }
            ImGui::TextWrapped("Logout disconnects this world session and closes the game client so the launcher character flow can remain the owning shell for now.");
            return changed;
        }

        void DrawCharacterSheetWindow(const GameCore::ClientWorldState& worldState)
        {
            ImGui::TextUnformatted(worldState.m_session.m_displayName.c_str());
            ImGui::Separator();
            ImGui::Text("Race: Human");
            ImGui::Text("Class: Warrior");
            ImGui::Text("Level: %d", worldState.m_session.m_level);
            ImGui::Text("Currency: %s", FormatCurrency(worldState.m_session.m_currency).c_str());
            ImGui::Separator();
            ImGui::Text("Strength: %d", worldState.m_session.m_stats.m_strength);
            ImGui::Text("Stamina: %d", worldState.m_session.m_stats.m_stamina);
            ImGui::Text("Armor: %d", worldState.m_session.m_stats.m_armor);
            ImGui::Text("Attack Power: %.1f", worldState.m_session.m_stats.m_attackPower);
            ImGui::Text("Armor Reduction: %.1f%%", worldState.m_session.m_stats.m_armorReductionPct * 100.0);
            ImGui::Separator();
            ImGui::Text(
                "Position: %.1f, %.1f, %.1f",
                worldState.m_session.m_position.m_x,
                worldState.m_session.m_position.m_y,
                worldState.m_session.m_position.m_z);
        }

        void DrawTalentWindow(GameCore::IGameCoreRequests* gameCore, const GameCore::ClientWorldState& worldState)
        {
            const auto& talents = worldState.m_session.m_talents;
            ImGui::Text("Warrior Talents  |  Points %d / %d", talents.m_pointsAvailable, talents.m_pointsGranted);
            ImGui::Separator();
            if (!talents.m_unlocked)
            {
                ImGui::Text("Unlocks at level %d", talents.m_unlockLevel);
                return;
            }

            ImGui::BeginChild(
                "##talent_scroll",
                ImVec2(0.0f, 0.0f),
                false,
                ImGuiWindowFlags_AlwaysVerticalScrollbar);
            for (const auto& category : talents.m_categories)
            {
                bool wroteCategory = false;
                for (const auto& talent : talents.m_talents)
                {
                    if (talent.m_category != category)
                    {
                        continue;
                    }
                    if (!wroteCategory)
                    {
                        ImGui::TextUnformatted(category.c_str());
                        ImGui::Separator();
                        wroteCategory = true;
                    }

                    ImGui::PushID(talent.m_id.c_str());
                    ImGui::Text("%s  %d/%d", talent.m_displayName.c_str(), talent.m_rank, talent.m_maxRank);
                    ImGui::TextWrapped("%s", talent.m_description.c_str());
                    if (!talent.m_requirementText.empty())
                    {
                        ImGui::TextDisabled("%s", talent.m_requirementText.c_str());
                    }
                    if (talent.m_canSelect)
                    {
                        if (ImGui::Button("Select", ImVec2(92.0f, 0.0f)) && gameCore)
                        {
                            gameCore->SelectTalent(talent.m_id);
                        }
                    }
                    else if (talent.m_rank >= talent.m_maxRank)
                    {
                        ImGui::TextDisabled("Max rank");
                    }
                    else if (talents.m_pointsAvailable <= 0)
                    {
                        ImGui::TextDisabled("No points available");
                    }
                    ImGui::Spacing();
                    ImGui::Separator();
                    ImGui::PopID();
                }
                if (wroteCategory)
                {
                    ImGui::Spacing();
                }
            }
            ImGui::EndChild();
        }

        const char* QuestBucketLabel(const AZStd::string& statusBucket)
        {
            if (statusBucket == "active")
            {
                return "Active";
            }
            if (statusBucket == "ready_to_turn_in")
            {
                return "Ready to Turn In";
            }
            if (statusBucket == "completed")
            {
                return "Completed";
            }
            return "Available";
        }

        void DrawQuestLogWindow(GameCore::IGameCoreRequests* gameCore, const GameCore::ClientWorldState& worldState)
        {
            AZStd::vector<NetClient::QuestState> fallbackQuests;
            const AZStd::vector<NetClient::QuestState>* quests = &worldState.m_session.m_quests;
            if (quests->empty() && !worldState.m_session.m_quest.m_id.empty())
            {
                fallbackQuests.push_back(worldState.m_session.m_quest);
                quests = &fallbackQuests;
            }

            ImGui::TextUnformatted("Stonewake Vale");
            ImGui::Separator();
            ImGui::BeginChild("##quest_log_scroll", ImVec2(0.0f, 0.0f), false, ImGuiWindowFlags_AlwaysVerticalScrollbar);
            const char* buckets[] = {"active", "ready_to_turn_in", "available", "completed"};
            for (const char* bucket : buckets)
            {
                bool wroteHeader = false;
                for (const auto& quest : *quests)
                {
                    if (quest.m_id.empty() || quest.m_statusBucket != bucket)
                    {
                        continue;
                    }
                    if (!wroteHeader)
                    {
                        ImGui::TextUnformatted(QuestBucketLabel(bucket));
                        ImGui::Separator();
                        wroteHeader = true;
                    }

                    ImGui::PushID(quest.m_id.c_str());
                    const AZStd::string questTitle = GetQuestDisplayTitle(quest);
                    ImGui::TextUnformatted(questTitle.c_str());
                    ImGui::SameLine();
                    ImGui::TextDisabled("%s", quest.m_category.empty() ? "Stonewake Vale" : quest.m_category.c_str());
                    if (quest.m_groupRecommended)
                    {
                        ImGui::SameLine();
                        ImGui::TextDisabled("Recommended Group");
                    }
                    ImGui::TextWrapped("%s", quest.m_objectiveText.c_str());
                    ImGui::Text(
                        "Progress: %d / %d  |  Reward: %d XP and %dg %ds %dc",
                        quest.m_currentCount,
                        quest.m_targetCount,
                        quest.m_rewardXp,
                        quest.m_rewardCurrencyGold,
                        quest.m_rewardCurrencySilver,
                        quest.m_rewardCurrencyCopper);
                    if (!quest.m_objectiveAreaName.empty())
                    {
                        ImGui::TextWrapped("Area: %s. %s", quest.m_objectiveAreaName.c_str(), quest.m_routeHintText.c_str());
                    }
                    const bool trackable = quest.m_statusBucket == "active" || quest.m_statusBucket == "ready_to_turn_in";
                    if (trackable && gameCore)
                    {
                        const char* buttonLabel = quest.m_tracked ? "Untrack" : "Track";
                        if (ImGui::Button(buttonLabel, ImVec2(96.0f, 0.0f)))
                        {
                            gameCore->TrackQuest(quest.m_id, !quest.m_tracked);
                        }
                    }
                    ImGui::Spacing();
                    ImGui::Separator();
                    ImGui::PopID();
                }
                if (wroteHeader)
                {
                    ImGui::Spacing();
                }
            }
            ImGui::EndChild();
        }

        ImVec4 ChatChannelColor(const AZStd::string& channel)
        {
            if (channel == "system")
            {
                return ImVec4(0.95f, 0.75f, 0.42f, 1.0f);
            }
            if (channel == "whisper")
            {
                return ImVec4(0.86f, 0.48f, 0.92f, 1.0f);
            }
            if (channel == "party")
            {
                return ImVec4(0.42f, 0.68f, 1.0f, 1.0f);
            }
            if (channel == "guild")
            {
                return ImVec4(0.38f, 0.86f, 0.48f, 1.0f);
            }
            return ImVec4(0.92f, 0.92f, 0.86f, 1.0f);
        }

        const char* ChatChannelLabel(const AZStd::string& channel)
        {
            if (channel == "system")
            {
                return "System";
            }
            if (channel == "whisper")
            {
                return "Whisper";
            }
            if (channel == "party")
            {
                return "Party";
            }
            if (channel == "guild")
            {
                return "Guild";
            }
            return "Say";
        }

        bool DrawChatWindow(
            const GameCore::ClientWorldState& worldState,
            AZStd::string& selectedChannel,
            char* inputBuffer,
            size_t inputBufferSize,
            char* whisperTargetBuffer,
            size_t whisperTargetBufferSize,
            AZStd::string& outSubmittedInput)
        {
            ImGui::BeginChild("##chat_scrollback", ImVec2(0.0f, 158.0f), false, ImGuiWindowFlags_AlwaysVerticalScrollbar);
            for (const auto& message : worldState.m_social.m_chatMessages)
            {
                ImGui::PushStyleColor(ImGuiCol_Text, ChatChannelColor(message.m_channel));
                ImGui::Text("[%s]", ChatChannelLabel(message.m_channel));
                ImGui::PopStyleColor();
                ImGui::SameLine();
                if (message.m_channel == "system")
                {
                    ImGui::TextWrapped("%s", message.m_messageText.c_str());
                }
                else
                {
                    ImGui::TextWrapped("%s: %s", message.m_senderDisplayName.c_str(), message.m_messageText.c_str());
                }
            }
            if (ImGui::GetScrollY() >= ImGui::GetScrollMaxY() - 4.0f)
            {
                ImGui::SetScrollHereY(1.0f);
            }
            ImGui::EndChild();

            if (ImGui::Button("Say", ImVec2(58.0f, 24.0f)))
            {
                selectedChannel = "say";
            }
            ImGui::SameLine();
            if (ImGui::Button("Whisper", ImVec2(82.0f, 24.0f)))
            {
                selectedChannel = "whisper";
            }
            ImGui::SameLine();
            if (ImGui::Button("Party", ImVec2(66.0f, 24.0f)))
            {
                selectedChannel = "party";
            }
            ImGui::SameLine();
            if (ImGui::Button("Guild", ImVec2(66.0f, 24.0f)))
            {
                selectedChannel = "guild";
            }
            ImGui::SameLine();
            ImGui::Text("Channel: %s", ChatChannelLabel(selectedChannel));

            if (selectedChannel == "whisper")
            {
                ImGui::SetNextItemWidth(138.0f);
                ImGui::InputText("Target", whisperTargetBuffer, whisperTargetBufferSize);
                ImGui::SameLine();
            }

            ImGui::SetNextItemWidth(-1.0f);
            const bool submitted = ImGui::InputText(
                "##chat_input",
                inputBuffer,
                inputBufferSize,
                ImGuiInputTextFlags_EnterReturnsTrue);
            if (submitted)
            {
                outSubmittedInput = inputBuffer;
                inputBuffer[0] = '\0';
            }
            return submitted;
        }

        void DrawPartyFrames(const GameCore::ClientWorldState& worldState)
        {
            if (!worldState.m_social.m_hasParty)
            {
                ImGui::TextUnformatted("No party");
                return;
            }

            for (const auto& member : worldState.m_social.m_party.m_members)
            {
                ImGui::Separator();
                ImGui::Text(
                    "%s%s  |  Level %d %s",
                    member.m_leader ? "* " : "",
                    member.m_displayName.c_str(),
                    member.m_level,
                    member.m_classId.c_str());
                ImGui::Text("%s  |  %s", member.m_zoneId.c_str(), member.m_online ? "online" : "offline");
                if (member.m_online)
                {
                    DrawMeter("Health", static_cast<float>(member.m_health), static_cast<float>(member.m_maxHealth), ColorU32(173, 52, 44), ImVec2(204.0f, 14.0f));
                    DrawMeter("Grit", static_cast<float>(member.m_resource), static_cast<float>(member.m_maxResource), ColorU32(54, 117, 181), ImVec2(204.0f, 14.0f));
                }
            }
        }

        void DrawPartyInvitePrompt(GameCore::IGameCoreRequests* gameCore, const GameCore::ClientWorldState& worldState)
        {
            if (!gameCore || worldState.m_social.m_partyInvites.empty())
            {
                return;
            }

            const auto& invite = worldState.m_social.m_partyInvites.front();
            ImGui::Text("%s invited you to a party.", invite.m_inviterDisplayName.c_str());
            if (ImGui::Button("Accept", ImVec2(104.0f, 28.0f)))
            {
                gameCore->AcceptPartyInvite(invite.m_inviteId);
            }
            ImGui::SameLine();
            if (ImGui::Button("Decline", ImVec2(104.0f, 28.0f)))
            {
                gameCore->DeclinePartyInvite(invite.m_inviteId);
            }
        }

        void DrawSocialWindow(
            GameCore::IGameCoreRequests* gameCore,
            const GameCore::ClientWorldState& worldState,
            char* nameBuffer,
            size_t nameBufferSize)
        {
            if (ImGui::BeginTabBar("##social_tabs"))
            {
                if (ImGui::BeginTabItem("Friends"))
                {
                    ImGui::SetNextItemWidth(210.0f);
                    ImGui::InputText("Name", nameBuffer, nameBufferSize);
                    ImGui::SameLine();
                    if (ImGui::Button("Add", ImVec2(62.0f, 0.0f)) && gameCore)
                    {
                        gameCore->AddFriend(nameBuffer);
                    }
                    ImGui::SameLine();
                    if (ImGui::Button("Remove", ImVec2(86.0f, 0.0f)) && gameCore)
                    {
                        gameCore->RemoveFriend(nameBuffer);
                    }

                    ImGui::Separator();
                    ImGui::BeginChild("##friends_list", ImVec2(0.0f, 280.0f), false, ImGuiWindowFlags_AlwaysVerticalScrollbar);
                    for (const auto& friendState : worldState.m_social.m_friends)
                    {
                        ImGui::Text(
                            "%s  |  L%d %s  |  %s",
                            friendState.m_displayName.c_str(),
                            friendState.m_level,
                            friendState.m_classId.c_str(),
                            friendState.m_online ? "online" : "offline");
                        if (friendState.m_online)
                        {
                            ImGui::SameLine();
                            ImGui::Text(" | %s", friendState.m_zoneId.c_str());
                            ImGui::SameLine();
                            ImGui::PushID(friendState.m_characterId.c_str());
                            if (ImGui::Button("Invite", ImVec2(72.0f, 0.0f)) && gameCore)
                            {
                                gameCore->InviteParty(friendState.m_displayName, friendState.m_characterId);
                            }
                            ImGui::PopID();
                        }
                    }
                    ImGui::EndChild();
                    ImGui::EndTabItem();
                }

                if (ImGui::BeginTabItem("Party"))
                {
                    ImGui::SetNextItemWidth(210.0f);
                    ImGui::InputText("Invite Name", nameBuffer, nameBufferSize);
                    ImGui::SameLine();
                    if (ImGui::Button("Invite", ImVec2(82.0f, 0.0f)) && gameCore)
                    {
                        gameCore->InviteParty(nameBuffer, {});
                    }
                    ImGui::SameLine();
                    if (ImGui::Button("Leave", ImVec2(72.0f, 0.0f)) && gameCore)
                    {
                        gameCore->LeaveParty();
                    }
                    ImGui::Separator();
                    DrawPartyFrames(worldState);
                    ImGui::EndTabItem();
                }
                ImGui::EndTabBar();
            }
        }

        void DrawMicroMenuBar(
            bool& characterSheetOpen,
            bool& talentsOpen,
            bool& questLogOpen,
            bool& mapOpen,
            bool& spellbookOpen,
            bool& bagOpen,
            bool& settingsOpen,
            bool& socialOpen)
        {
            struct MenuButtonState
            {
                const char* m_label;
                bool* m_toggle;
            };

            MenuButtonState buttons[] = {
                {"Character", &characterSheetOpen},
                {"Talents", &talentsOpen},
                {"Quest Log", &questLogOpen},
                {"Map", &mapOpen},
                {"Spellbook", &spellbookOpen},
                {"Bag", &bagOpen},
                {"Social", &socialOpen},
                {"Settings", &settingsOpen},
            };

            for (size_t index = 0; index < AZ_ARRAY_SIZE(buttons); ++index)
            {
                if (ImGui::Button(buttons[index].m_label, ImVec2(178.0f, 28.0f)))
                {
                    *buttons[index].m_toggle = !*buttons[index].m_toggle;
                }
            }
        }
    } // namespace

    UiClientSystemComponent::UiClientSystemComponent()
        : AzFramework::InputChannelEventListener(AzFramework::InputChannelEventListener::GetPriorityFirst())
    {
    }

    void UiClientSystemComponent::Reflect(AZ::ReflectContext* context)
    {
        if (auto* serializeContext = azrtti_cast<AZ::SerializeContext*>(context))
        {
            serializeContext->Class<UiClientSystemComponent, AZ::Component>()->Version(0);
        }
    }

    void UiClientSystemComponent::GetProvidedServices(AZ::ComponentDescriptor::DependencyArrayType& provided)
    {
        provided.push_back(AZ_CRC_CE("UiClientService"));
    }

    void UiClientSystemComponent::GetIncompatibleServices(AZ::ComponentDescriptor::DependencyArrayType& incompatible)
    {
        incompatible.push_back(AZ_CRC_CE("UiClientService"));
    }

    void UiClientSystemComponent::GetRequiredServices(AZ::ComponentDescriptor::DependencyArrayType& required)
    {
        required.push_back(AZ_CRC_CE("GameCoreService"));
        required.push_back(AZ_CRC_CE("NetClientService"));
    }

    void UiClientSystemComponent::GetDependentServices(AZ::ComponentDescriptor::DependencyArrayType&)
    {
    }

    void UiClientSystemComponent::Activate()
    {
        AzFramework::InputChannelEventListener::Connect();
        ImGui::ImGuiUpdateListenerBus::Handler::BusConnect();
        if (auto* console = AZ::Interface<AZ::IConsole>::Get())
        {
            console->PerformCommand("imgui_EnableImGui 1");
            console->PerformCommand("bg_showDebugConsole false");
            console->PerformCommand("r_DisplayInfo 0");
        }
        ImGui::ImGuiManagerBus::Broadcast(&ImGui::IImGuiManager::SetDisplayState, ImGui::DisplayState::Visible);
        ImGui::ImGuiManagerBus::Broadcast(&ImGui::IImGuiManager::SetEnableDiscreteInputMode, false);
        AZ_Printf("amandacore", "client.ui_client_activated");
        m_lastQuestCount = -1;
        m_lastExperience = -1;
        m_lastCurrencyCopper = -1;
        m_lastHudTargetId.clear();
        m_lastWorldSessionToken.clear();
        m_lastErrorMessage.clear();
        m_questToastExpiresAt = 0;
        m_lastHandledInteractionSequence = 0;
        m_questGossipOpen = false;
        m_lastNearCommandPoint = false;
        m_lastWorldConnected = false;
        m_loggedActionBarVisible = false;
        m_eventLog.clear();
        m_shiftHeld = false;
        m_pendingActionAssignmentAbilityId.clear();
        m_pendingActionMoveSlot = -1;
        m_pendingInventoryMoveSlot = -1;
        LoadDefaultKeybindings();
        LoadUiSettings();
    }

    void UiClientSystemComponent::Deactivate()
    {
        SaveUiSettings();
        ImGui::ImGuiUpdateListenerBus::Handler::BusDisconnect();
        AzFramework::InputChannelEventListener::Disconnect();
    }

    void UiClientSystemComponent::LoadDefaultKeybindings()
    {
        for (auto& binding : m_actionSlotBindings)
        {
            binding.clear();
        }

        m_actionSlotBindings[0] = AzFramework::InputDeviceKeyboard::Key::AlphanumericF.GetName();
        m_actionSlotBindings[1] = AzFramework::InputDeviceKeyboard::Key::Alphanumeric1.GetName();
        m_actionSlotBindings[2] = AzFramework::InputDeviceKeyboard::Key::Alphanumeric2.GetName();
        m_actionSlotBindings[3] = AzFramework::InputDeviceKeyboard::Key::Alphanumeric3.GetName();
        m_actionSlotBindings[4] = AzFramework::InputDeviceKeyboard::Key::Alphanumeric4.GetName();
        m_actionSlotBindings[5] = AzFramework::InputDeviceKeyboard::Key::Alphanumeric5.GetName();
        m_actionSlotBindings[6] = AzFramework::InputDeviceKeyboard::Key::Alphanumeric6.GetName();
        m_actionSlotBindings[7] = AzFramework::InputDeviceKeyboard::Key::Alphanumeric7.GetName();
        m_actionSlotBindings[8] = AzFramework::InputDeviceKeyboard::Key::Alphanumeric8.GetName();
        m_actionSlotBindings[9] = AzFramework::InputDeviceKeyboard::Key::Alphanumeric9.GetName();
        m_actionSlotBindings[10] = AzFramework::InputDeviceKeyboard::Key::Alphanumeric0.GetName();

        m_spellbookBinding = AzFramework::InputDeviceKeyboard::Key::AlphanumericP.GetName();
        m_bagBinding = AzFramework::InputDeviceKeyboard::Key::AlphanumericB.GetName();
        m_characterBinding = AzFramework::InputDeviceKeyboard::Key::AlphanumericC.GetName();
        m_questLogBinding = AzFramework::InputDeviceKeyboard::Key::AlphanumericL.GetName();
        m_mapBinding = AzFramework::InputDeviceKeyboard::Key::AlphanumericM.GetName();
        m_settingsBinding = AzFramework::InputDeviceKeyboard::Key::Escape.GetName();
        m_interactBinding = AzFramework::InputDeviceKeyboard::Key::AlphanumericE.GetName();
        m_targetHostileBinding = AzFramework::InputDeviceKeyboard::Key::EditTab.GetName();
    }

    void UiClientSystemComponent::LoadUiSettings()
    {
        const AZStd::string settingsPath = GetUiSettingsPath();
        FILE* settingsFile = nullptr;
        if (fopen_s(&settingsFile, settingsPath.c_str(), "r") != 0 || !settingsFile)
        {
            AZ_Printf("amandacore", "client.ui_settings_loaded path=%s defaults=true", settingsPath.c_str());
            return;
        }

        char line[256] = {};
        while (fgets(line, sizeof(line), settingsFile))
        {
            auto readBool = [&line](const char* key, bool& value)
            {
                const size_t keyLength = strlen(key);
                if (strncmp(line, key, keyLength) != 0 || line[keyLength] != '=')
                {
                    return;
                }
                value = line[keyLength + 1] == '1';
            };
            auto readString = [&line](const char* key, AZStd::string& value)
            {
                const size_t keyLength = strlen(key);
                if (strncmp(line, key, keyLength) != 0 || line[keyLength] != '=')
                {
                    return;
                }
                value = line + keyLength + 1;
                while (!value.empty() && (value.back() == '\n' || value.back() == '\r'))
                {
                    value.pop_back();
                }
            };
            readBool("upperActionBar", m_extraUpperActionBarVisible);
            readBool("rightActionBarOne", m_rightActionBarOneVisible);
            readBool("rightActionBarTwo", m_rightActionBarTwoVisible);
            readString("bind.spellbook", m_spellbookBinding);
            readString("bind.bag", m_bagBinding);
            readString("bind.character", m_characterBinding);
            readString("bind.questLog", m_questLogBinding);
            readString("bind.map", m_mapBinding);
            readString("bind.settings", m_settingsBinding);
            readString("bind.interact", m_interactBinding);
            readString("bind.targetHostile", m_targetHostileBinding);
            for (int slotIndex = 0; slotIndex < ActionBarSlotCount; ++slotIndex)
            {
                const AZStd::string key = AZStd::string::format("bind.slot.%d", slotIndex);
                readString(key.c_str(), m_actionSlotBindings[slotIndex]);
            }
        }
        fclose(settingsFile);

        AZ_Printf(
            "amandacore",
            "client.ui_settings_loaded path=%s upper=%s right1=%s right2=%s",
            settingsPath.c_str(),
            m_extraUpperActionBarVisible ? "true" : "false",
            m_rightActionBarOneVisible ? "true" : "false",
            m_rightActionBarTwoVisible ? "true" : "false");
    }

    void UiClientSystemComponent::SaveUiSettings() const
    {
        const AZStd::string settingsPath = GetUiSettingsPath();
        FILE* settingsFile = nullptr;
        if (fopen_s(&settingsFile, settingsPath.c_str(), "w") != 0 || !settingsFile)
        {
            AZ_Warning("amandacore", false, "Unable to save UI settings to %s", settingsPath.c_str());
            return;
        }

        fprintf(settingsFile, "upperActionBar=%d\n", m_extraUpperActionBarVisible ? 1 : 0);
        fprintf(settingsFile, "rightActionBarOne=%d\n", m_rightActionBarOneVisible ? 1 : 0);
        fprintf(settingsFile, "rightActionBarTwo=%d\n", m_rightActionBarTwoVisible ? 1 : 0);
        fprintf(settingsFile, "bind.spellbook=%s\n", m_spellbookBinding.c_str());
        fprintf(settingsFile, "bind.bag=%s\n", m_bagBinding.c_str());
        fprintf(settingsFile, "bind.character=%s\n", m_characterBinding.c_str());
        fprintf(settingsFile, "bind.questLog=%s\n", m_questLogBinding.c_str());
        fprintf(settingsFile, "bind.map=%s\n", m_mapBinding.c_str());
        fprintf(settingsFile, "bind.settings=%s\n", m_settingsBinding.c_str());
        fprintf(settingsFile, "bind.interact=%s\n", m_interactBinding.c_str());
        fprintf(settingsFile, "bind.targetHostile=%s\n", m_targetHostileBinding.c_str());
        for (int slotIndex = 0; slotIndex < ActionBarSlotCount; ++slotIndex)
        {
            fprintf(settingsFile, "bind.slot.%d=%s\n", slotIndex, m_actionSlotBindings[slotIndex].c_str());
        }
        fclose(settingsFile);
        AZ_Printf("amandacore", "client.ui_settings_saved path=%s", settingsPath.c_str());
    }

    void UiClientSystemComponent::ApplyKeyBinding(const AZStd::string& actionId, const AZStd::string& keyName)
    {
        auto clearConflicts = [&keyName](AZStd::string& binding)
        {
            if (!keyName.empty() && binding == keyName)
            {
                binding.clear();
            }
        };

        clearConflicts(m_spellbookBinding);
        clearConflicts(m_bagBinding);
        clearConflicts(m_characterBinding);
        clearConflicts(m_questLogBinding);
        clearConflicts(m_mapBinding);
        clearConflicts(m_settingsBinding);
        clearConflicts(m_interactBinding);
        clearConflicts(m_targetHostileBinding);
        for (auto& binding : m_actionSlotBindings)
        {
            clearConflicts(binding);
        }

        int slotIndex = -1;
        if (TryParseSlotActionId(actionId, slotIndex))
        {
            m_actionSlotBindings[slotIndex] = keyName;
        }
        else if (actionId == "spellbook")
        {
            m_spellbookBinding = keyName;
        }
        else if (actionId == "bag")
        {
            m_bagBinding = keyName;
        }
        else if (actionId == "character")
        {
            m_characterBinding = keyName;
        }
        else if (actionId == "questLog")
        {
            m_questLogBinding = keyName;
        }
        else if (actionId == "map")
        {
            m_mapBinding = keyName;
        }
        else if (actionId == "settings")
        {
            m_settingsBinding = keyName;
        }
        else if (actionId == "interact")
        {
            m_interactBinding = keyName;
        }
        else if (actionId == "targetHostile")
        {
            m_targetHostileBinding = keyName;
        }

        AZ_Printf(
            "amandacore",
            "client.keybind_applied action=%s key=%s",
            actionId.c_str(),
            DisplayKeyName(keyName).c_str());
    }

    bool UiClientSystemComponent::ActivateActionSlot(GameCore::IGameCoreRequests* gameCore, int slotIndex)
    {
        if (!gameCore)
        {
            return false;
        }

        const auto& worldState = gameCore->GetClientWorldState();
        const NetClient::ActionBarSlotState* slotState = FindActionBarSlot(worldState.m_session, slotIndex);
        if (!slotState || !slotState->m_learned || slotState->m_abilityId.empty())
        {
            return true;
        }

        if (slotState->m_requiresTarget)
        {
            const auto* targetEntity = FindTargetEntity(worldState);
            if (!targetEntity || targetEntity->m_kind != "hostile_mob")
            {
                AddHudEvent("That action needs a hostile target.");
                return true;
            }
        }

        if (slotState->m_abilityId == AutoAttackAbilityId)
        {
            gameCore->SetAutoAttack(!worldState.m_session.m_autoAttackActive);
        }
        else
        {
            gameCore->ActivateAbility(slotState->m_abilityId);
        }
        return true;
    }

    bool UiClientSystemComponent::TargetNextHostile(GameCore::IGameCoreRequests* gameCore)
    {
        if (!gameCore)
        {
            return false;
        }

        const auto& session = gameCore->GetClientWorldState().m_session;
        AZStd::vector<AZStd::string> hostileIds;
        for (const auto& entity : session.m_entities)
        {
            if (entity.m_kind == "hostile_mob" && entity.m_alive)
            {
                hostileIds.push_back(entity.m_id);
            }
        }
        if (hostileIds.empty())
        {
            return true;
        }

        size_t nextIndex = 0;
        for (size_t index = 0; index < hostileIds.size(); ++index)
        {
            if (hostileIds[index] == session.m_currentTargetId)
            {
                nextIndex = (index + 1) % hostileIds.size();
                break;
            }
        }
        gameCore->SetTarget(hostileIds[nextIndex]);
        return true;
    }

    bool UiClientSystemComponent::TryHandleBoundAction(GameCore::IGameCoreRequests* gameCore, const AZStd::string& keyName)
    {
        if (keyName.empty())
        {
            return false;
        }

        if (keyName == m_spellbookBinding)
        {
            m_spellbookOpen = !m_spellbookOpen;
            AZ_Printf("amandacore", "client.spellbook_visible open=%s", m_spellbookOpen ? "true" : "false");
            return true;
        }
        if (keyName == m_bagBinding)
        {
            m_bagOpen = !m_bagOpen;
            AZ_Printf("amandacore", "client.inventory_visible open=%s", m_bagOpen ? "true" : "false");
            return true;
        }
        if (keyName == m_characterBinding)
        {
            m_characterSheetOpen = !m_characterSheetOpen;
            return true;
        }
        if (keyName == m_questLogBinding)
        {
            m_questLogOpen = !m_questLogOpen;
            return true;
        }
        if (keyName == m_mapBinding)
        {
            m_mapOpen = !m_mapOpen;
            return true;
        }
        if (keyName == m_settingsBinding)
        {
            m_settingsOpen = !m_settingsOpen;
            AZ_Printf("amandacore", "client.settings_visible open=%s", m_settingsOpen ? "true" : "false");
            return true;
        }
        if (keyName == m_interactBinding)
        {
            return InteractWithCurrentTarget(gameCore);
        }
        if (keyName == m_targetHostileBinding)
        {
            return TargetNextHostile(gameCore);
        }

        if (!m_shiftHeld)
        {
            for (int slotIndex = 0; slotIndex < ActionBarSlotCount; ++slotIndex)
            {
                if (keyName == m_actionSlotBindings[slotIndex])
                {
                    return ActivateActionSlot(gameCore, slotIndex);
                }
            }
        }

        return false;
    }

    bool UiClientSystemComponent::InteractWithCurrentTarget(GameCore::IGameCoreRequests* gameCore)
    {
        if (!gameCore)
        {
            return false;
        }

        const auto& worldState = gameCore->GetClientWorldState();
        const auto* targetEntity = FindTargetEntity(worldState);
        if (!targetEntity || !IsFriendlyNpc(*targetEntity))
        {
            return false;
        }

        const float distanceToTarget = Distance2D(
            static_cast<float>(worldState.m_session.m_position.m_x),
            static_cast<float>(worldState.m_session.m_position.m_y),
            static_cast<float>(targetEntity->m_x),
            static_cast<float>(targetEntity->m_y));
        if (distanceToTarget > CommandPointRadius)
        {
            AddHudEvent(AZStd::string::format("Move closer to %s", targetEntity->m_displayName.c_str()));
            return false;
        }

        return OpenInteractionForEntity(gameCore, *targetEntity, "interact_key");
    }

    bool UiClientSystemComponent::OpenInteractionForEntity(
        GameCore::IGameCoreRequests* gameCore,
        const NetClient::VisibleEntity& entity,
        const char* source)
    {
        if (!gameCore || !IsFriendlyNpc(entity))
        {
            return false;
        }

        const auto& worldState = gameCore->GetClientWorldState();
        const float distanceToTarget = Distance2D(
            static_cast<float>(worldState.m_session.m_position.m_x),
            static_cast<float>(worldState.m_session.m_position.m_y),
            static_cast<float>(entity.m_x),
            static_cast<float>(entity.m_y));
        if (distanceToTarget > CommandPointRadius)
        {
            AddHudEvent(AZStd::string::format("Move closer to %s", entity.m_displayName.c_str()));
            return false;
        }

        if (ShouldOpenQuestForEntity(worldState, entity))
        {
            m_questGossipOpen = true;
            m_trainerOpen = false;
            AZ_Printf(
                "amandacore",
                "client.quest_gossip_visible open=true source=%s targetId=%s",
                source,
                entity.m_id.c_str());
            AddHudEvent(AZStd::string::format("Speaking with %s", entity.m_displayName.c_str()));
            return true;
        }

        if (IsTrainerNpc(entity))
        {
            m_trainerOpen = true;
            m_questGossipOpen = false;
            AZ_Printf(
                "amandacore",
                "client.trainer_visible open=true source=%s targetId=%s",
                source,
                entity.m_id.c_str());
            AddHudEvent(AZStd::string::format("Training with %s", entity.m_displayName.c_str()));
            return true;
        }

        if (IsQuestGiverNpc(entity) && !IsTrainerNpc(entity) && worldState.m_session.m_quest.m_giverNpcId.empty())
        {
            m_questGossipOpen = true;
            m_trainerOpen = false;
            AZ_Printf(
                "amandacore",
                "client.quest_gossip_visible open=true source=%s targetId=%s",
                source,
                entity.m_id.c_str());
            AddHudEvent(AZStd::string::format("Speaking with %s", entity.m_displayName.c_str()));
            return true;
        }

        AddHudEvent(AZStd::string::format("%s has nothing new right now", entity.m_displayName.c_str()));
        return false;
    }

    bool UiClientSystemComponent::SubmitChatInput(GameCore::IGameCoreRequests* gameCore, const AZStd::string& input)
    {
        if (!gameCore)
        {
            return false;
        }

        auto trim = [](const AZStd::string& value) -> AZStd::string
        {
            const size_t start = value.find_first_not_of(" \t\r\n");
            if (start == AZStd::string::npos)
            {
                return {};
            }
            const size_t end = value.find_last_not_of(" \t\r\n");
            return value.substr(start, end - start + 1);
        };

        AZStd::string text = trim(input);
        if (text.empty())
        {
            return false;
        }

        auto splitFirst = [&trim](const AZStd::string& value) -> AZStd::pair<AZStd::string, AZStd::string>
        {
            const size_t space = value.find(' ');
            if (space == AZStd::string::npos)
            {
                return {trim(value), {}};
            }
            return {trim(value.substr(0, space)), trim(value.substr(space + 1))};
        };

        if (text[0] != '/')
        {
            const AZStd::string target = m_chatChannel == "whisper" ? AZStd::string(m_chatWhisperTargetBuffer) : AZStd::string();
            return gameCore->SubmitChatMessage(m_chatChannel, target, text);
        }

        const auto commandAndRest = splitFirst(text.substr(1));
        const AZStd::string command = commandAndRest.first;
        const AZStd::string rest = commandAndRest.second;
        if (command == "say" || command == "s")
        {
            return gameCore->SubmitChatMessage("say", {}, rest);
        }
        if (command == "party" || command == "p")
        {
            return gameCore->SubmitChatMessage("party", {}, rest);
        }
        if (command == "whisper" || command == "w")
        {
            const auto targetAndMessage = splitFirst(rest);
            return gameCore->SubmitChatMessage("whisper", targetAndMessage.first, targetAndMessage.second);
        }
        if (command == "friend")
        {
            const auto actionAndName = splitFirst(rest);
            if (actionAndName.first == "add")
            {
                return gameCore->AddFriend(actionAndName.second);
            }
            if (actionAndName.first == "remove")
            {
                return gameCore->RemoveFriend(actionAndName.second);
            }
        }
        if (command == "invite")
        {
            return gameCore->InviteParty(rest, {});
        }
        if (command == "leave")
        {
            return gameCore->LeaveParty();
        }

        AddHudEvent(AZStd::string::format("Unknown command: /%s", command.c_str()));
        return false;
    }

    bool UiClientSystemComponent::OnInputChannelEventFiltered(const AzFramework::InputChannel& inputChannel)
    {
        const auto& channelId = inputChannel.GetInputChannelId();
        if (channelId == AzFramework::InputDeviceKeyboard::Key::ModifierShiftL ||
            channelId == AzFramework::InputDeviceKeyboard::Key::ModifierShiftR)
        {
            m_shiftHeld = inputChannel.IsActive();
            if (!m_shiftHeld)
            {
                m_pendingActionAssignmentAbilityId.clear();
                m_pendingActionMoveSlot = -1;
            }
            return false;
        }

        if (!inputChannel.IsStateBegan())
        {
            return false;
        }

        auto* gameCore = GameCore::IGameCoreRequests::Get();
        const AZStd::string keyName = channelId.GetName();

        if (!m_pendingKeybindActionId.empty())
        {
            if (IsBindableKeyboardChannel(channelId))
            {
                ApplyKeyBinding(m_pendingKeybindActionId, keyName);
                m_pendingKeybindActionId.clear();
                SaveUiSettings();
                AddHudEvent(AZStd::string::format("Keybinding updated: %s", DisplayKeyName(keyName).c_str()));
            }
            return true;
        }

        if (TryHandleBoundAction(gameCore, keyName))
        {
            return true;
        }

        return false;
    }

    void UiClientSystemComponent::OnImGuiUpdate()
    {
        auto* gameCore = GameCore::IGameCoreRequests::Get();
        if (!gameCore)
        {
            return;
        }

        const auto& worldState = gameCore->GetClientWorldState();
        const AZ::s64 nowMs = NowMs();

        UpdateQuestToast(
            worldState.m_session.m_quest.m_state,
            worldState.m_session.m_quest.m_currentCount,
            worldState.m_session.m_quest.m_targetCount,
            worldState.m_session.m_experience,
            worldState.m_session.m_quest.m_rewardXp,
            worldState.m_session.m_currency.m_totalCopper,
            worldState.m_session.m_quest.m_rewardCurrencyGold,
            worldState.m_session.m_quest.m_rewardCurrencySilver,
            worldState.m_session.m_quest.m_rewardCurrencyCopper);

        const ImVec2 displaySize = ImGui::GetIO().DisplaySize;
        if (displaySize.x <= 1.0f || displaySize.y <= 1.0f)
        {
            return;
        }
        SuppressStockImGuiChrome();
        if (!m_loggedPlayableZoneReady)
        {
            AZ_Printf(
                "amandacore",
                "client.hud_update_started display=(%.0f,%.0f)",
                displaySize.x,
                displaySize.y);
        }

        if (!worldState.m_worldConnected)
        {
            m_trainerOpen = false;
            m_talentsOpen = false;
            if (BeginHudPanel(
                    "##world_connect_panel",
                    "World Link",
                    ImVec2((displaySize.x * 0.5f) - 220.0f, (displaySize.y * 0.5f) - 70.0f),
                    ImVec2(440.0f, 140.0f)))
            {
                ImGui::TextUnformatted(worldState.m_connectAttempted ? "Connecting to Stonewake Vale..." : "Awaiting world bootstrap...");
                if (!worldState.m_errorMessage.empty())
                {
                    ImGui::PushStyleColor(ImGuiCol_Text, ImVec4(0.94f, 0.66f, 0.40f, 1.0f));
                    ImGui::TextWrapped("%s", worldState.m_errorMessage.c_str());
                    ImGui::PopStyleColor();
                }
            }
            ImGui::End();
            m_lastWorldConnected = false;
            m_loggedActionBarVisible = false;
            m_loggedPlayableZoneReady = false;
            return;
        }

        const float playerX = static_cast<float>(worldState.m_session.m_position.m_x);
        const float playerY = static_cast<float>(worldState.m_session.m_position.m_y);
        const auto* trainerEntity = FindTrainerEntity(worldState);
        const float trainerX = trainerEntity ? static_cast<float>(trainerEntity->m_x) : CommandPointX;
        const float trainerY = trainerEntity ? static_cast<float>(trainerEntity->m_y) : CommandPointY;
        const float distanceToCommandPoint = Distance2D(playerX, playerY, trainerX, trainerY);
        const float distanceToEncounter = Distance2D(playerX, playerY, EncounterAnchorX, EncounterAnchorY);
        const bool nearCommandPoint = distanceToCommandPoint <= CommandPointRadius;
        const auto* targetEntity = FindTargetEntity(worldState);
        const bool hasHostileTarget = targetEntity && targetEntity->m_kind == "hostile_mob";
        const auto& cameraState = gameCore->GetCameraState();
        const bool actionEditMode = m_shiftHeld || ImGui::GetIO().KeyShift;

        if (worldState.m_pendingInteractionSequence != m_lastHandledInteractionSequence)
        {
            m_lastHandledInteractionSequence = worldState.m_pendingInteractionSequence;
            const NetClient::VisibleEntity* interactionEntity = nullptr;
            for (const auto& entity : worldState.m_session.m_entities)
            {
                if (entity.m_id == worldState.m_pendingInteractionEntityId)
                {
                    interactionEntity = &entity;
                    break;
                }
            }
            if (interactionEntity)
            {
                OpenInteractionForEntity(gameCore, *interactionEntity, "right_click");
            }
        }

        if (!m_lastWorldConnected)
        {
            AddHudEvent("Entered Stonewake Vale");
        }
        if (worldState.m_session.m_worldSessionToken != m_lastWorldSessionToken)
        {
            AddHudEvent(m_lastWorldSessionToken.empty() ? "World session linked" : "World session refreshed");
            m_lastWorldSessionToken = worldState.m_session.m_worldSessionToken;
        }
        if (nearCommandPoint != m_lastNearCommandPoint)
        {
            AZ_Printf(
                "amandacore",
                "client.command_point_%s distance=%.1f",
                nearCommandPoint ? "entered" : "departed",
                distanceToCommandPoint);
            AddHudEvent(nearCommandPoint ? "Near friendly NPC services" : "Left friendly NPC services");
            m_lastNearCommandPoint = nearCommandPoint;
        }

        if (targetEntity)
        {
            if (m_lastHudTargetId != targetEntity->m_id)
            {
                AZ_Printf(
                    "amandacore",
                    "client.target_hud_applied targetId=%s displayName=%s health=%.0f/%.0f",
                    targetEntity->m_id.c_str(),
                    GetMobDisplayLabel(*targetEntity).c_str(),
                    targetEntity->m_health,
                    targetEntity->m_maxHealth);
                AddHudEvent(AZStd::string::format("Target locked: %s", GetMobDisplayLabel(*targetEntity).c_str()));
                m_lastHudTargetId = targetEntity->m_id;
            }
        }
        else if (!m_lastHudTargetId.empty())
        {
            AZ_Printf("amandacore", "client.target_hud_cleared");
            AddHudEvent("Target cleared");
            m_lastHudTargetId.clear();
        }

        if (worldState.m_session.m_quest.m_state != m_lastQuestState)
        {
            if (worldState.m_session.m_quest.m_state == "active")
            {
                AddHudEvent("Field orders accepted");
            }
            else if (worldState.m_session.m_quest.m_state == "reward_granted")
            {
                AddHudEvent(
                    AZStd::string::format(
                        "Quest complete: +%d XP, +%s",
                        worldState.m_session.m_quest.m_rewardXp,
                        FormatCurrency(worldState.m_session.m_currency).c_str()));
            }
        }
        else if (worldState.m_session.m_quest.m_state == "active" &&
            worldState.m_session.m_quest.m_currentCount != m_lastQuestCount)
        {
            AddHudEvent(
                AZStd::string::format(
                    "Quest progress %d / %d",
                    worldState.m_session.m_quest.m_currentCount,
                    worldState.m_session.m_quest.m_targetCount));
        }

        if (!worldState.m_errorMessage.empty() && worldState.m_errorMessage != m_lastErrorMessage)
        {
            AddHudEvent(AZStd::string::format("Notice: %s", worldState.m_errorMessage.c_str()));
        }
        m_lastErrorMessage = worldState.m_errorMessage;
        m_lastWorldConnected = true;

        int visibleHostileCount = 0;
        for (const auto& entity : worldState.m_session.m_entities)
        {
            if (entity.m_kind == "hostile_mob" && entity.m_alive)
            {
                ++visibleHostileCount;
            }
        }
        if (!m_loggedPlayableZoneReady &&
            gameCore->GetCameraState().m_ready &&
            worldState.m_session.m_alive &&
            visibleHostileCount >= 3)
        {
            AZ_Printf("amandacore", "client.playable_zone_ready visible=true mobs=3 hud=true grounded=true");
            m_loggedPlayableZoneReady = true;
        }

        const ImVec2 playerFramePos(18.0f, 18.0f);
        const ImVec2 playerFrameSize(250.0f, 132.0f);
        const ImVec2 targetFramePos(280.0f, 18.0f);
        const ImVec2 targetFrameSize(250.0f, 132.0f);
        const ImVec2 utilityPos(18.0f, displaySize.y - 340.0f);
        const ImVec2 utilitySize(250.0f, 146.0f);
        const ImVec2 trackerPos(displaySize.x - 368.0f, 18.0f);
        const ImVec2 trackerSize(350.0f, 330.0f);
        const ImVec2 actionBarSize(744.0f, 206.0f);
        const ImVec2 actionBarPos(
            (displaySize.x - actionBarSize.x) * 0.5f,
            displaySize.y - actionBarSize.y - 18.0f);
        const ImVec2 microMenuSize(198.0f, 270.0f);
        const ImVec2 microMenuPos(actionBarPos.x + actionBarSize.x + 8.0f, actionBarPos.y);
        const ImVec2 minimapSize(250.0f, 260.0f);
        const ImVec2 minimapPos(displaySize.x > 1180.0f ? displaySize.x - 640.0f : 548.0f, 18.0f);
        const ImVec2 spellbookSize(680.0f, 560.0f);
        const ImVec2 spellbookPos(displaySize.x - spellbookSize.x - 18.0f, displaySize.y - spellbookSize.y - 188.0f);
        const ImVec2 trainerSize(460.0f, 440.0f);
        const ImVec2 trainerPos(spellbookPos.x - trainerSize.x - 18.0f, displaySize.y - trainerSize.y - 188.0f);
        const ImVec2 inventorySize(330.0f, 430.0f);
        const ImVec2 inventoryPos(18.0f, utilityPos.y - inventorySize.y - 18.0f);
        const ImVec2 settingsSize(680.0f, 520.0f);
        const ImVec2 settingsPos((displaySize.x - settingsSize.x) * 0.5f, (displaySize.y - settingsSize.y) * 0.5f);
        const ImVec2 characterSize(360.0f, 310.0f);
        const ImVec2 characterPos(280.0f, displaySize.y - characterSize.y - 210.0f);
        const ImVec2 talentsSize(440.0f, 430.0f);
        const ImVec2 talentsPos(
            AZ::GetMin(characterPos.x + characterSize.x + 18.0f, displaySize.x - talentsSize.x - 18.0f),
            displaySize.y - talentsSize.y - 210.0f);
        const ImVec2 questLogSize(400.0f, 280.0f);
        const ImVec2 questLogPos(280.0f, displaySize.y - questLogSize.y - 180.0f);
        const ImVec2 mapSize(640.0f, 470.0f);
        const ImVec2 mapPos((displaySize.x - mapSize.x) * 0.5f, (displaySize.y - mapSize.y) * 0.5f);
        const ImVec2 upperActionBarSize(744.0f, 92.0f);
        const ImVec2 upperActionBarPos(actionBarPos.x, actionBarPos.y - upperActionBarSize.y - 4.0f);
        const ImVec2 rightActionBarSize(86.0f, 740.0f);
        const ImVec2 rightActionBarOnePos(displaySize.x - rightActionBarSize.x - 18.0f, 370.0f);
        const ImVec2 rightActionBarTwoPos(rightActionBarOnePos.x - rightActionBarSize.x - 3.0f, 370.0f);
        const ImVec2 partyFramesSize(250.0f, 250.0f);
        const ImVec2 partyFramesPos(18.0f, 158.0f);
        const ImVec2 chatSize(460.0f, 250.0f);
        const ImVec2 chatPos(18.0f, AZ::GetMax(158.0f, utilityPos.y - chatSize.y - 18.0f));
        const ImVec2 socialSize(430.0f, 430.0f);
        const ImVec2 socialPos(displaySize.x - socialSize.x - 18.0f, displaySize.y - socialSize.y - 188.0f);
        const ImVec2 invitePromptSize(360.0f, 92.0f);
        const ImVec2 invitePromptPos((displaySize.x - invitePromptSize.x) * 0.5f, 170.0f);

        if (m_trainerOpen &&
            (worldState.m_session.m_trainer.m_id.empty() ||
                !worldState.m_session.m_trainer.m_inRange ||
                worldState.m_session.m_currentTargetId != worldState.m_session.m_trainer.m_id))
        {
            m_trainerOpen = false;
        }
        if (m_questGossipOpen && (!targetEntity || !IsQuestGiverNpc(*targetEntity)))
        {
            m_questGossipOpen = false;
        }

        if (BeginHudPanel("##player_frame", "Player", playerFramePos, playerFrameSize))
        {
            DrawPlayerFrame(worldState, distanceToCommandPoint, nearCommandPoint);
        }
        ImGui::End();

        if (BeginHudPanel("##target_frame", "Target", targetFramePos, targetFrameSize))
        {
            DrawTargetFrame(gameCore, targetEntity, worldState, playerX, playerY);
        }
        ImGui::End();

        if (worldState.m_social.m_hasParty && BeginHudPanel("##party_frames", "Party", partyFramesPos, partyFramesSize))
        {
            DrawPartyFrames(worldState);
        }
        if (worldState.m_social.m_hasParty)
        {
            ImGui::End();
        }

        if (BeginHudPanel("##chat_window", "Chat", chatPos, chatSize))
        {
            AZStd::string submittedInput;
            if (DrawChatWindow(
                    worldState,
                    m_chatChannel,
                    m_chatInputBuffer,
                    AZ_ARRAY_SIZE(m_chatInputBuffer),
                    m_chatWhisperTargetBuffer,
                    AZ_ARRAY_SIZE(m_chatWhisperTargetBuffer),
                    submittedInput))
            {
                SubmitChatInput(gameCore, submittedInput);
            }
        }
        ImGui::End();

        if (!worldState.m_social.m_partyInvites.empty() && BeginHudPanel("##party_invite_prompt", "Party Invite", invitePromptPos, invitePromptSize))
        {
            DrawPartyInvitePrompt(gameCore, worldState);
        }
        if (!worldState.m_social.m_partyInvites.empty())
        {
            ImGui::End();
        }

        if (BeginHudPanel("##utility_footer", "Inventory", utilityPos, utilitySize))
        {
            DrawUtilityFooter(worldState, m_bagOpen);
        }
        ImGui::End();

        if (BeginHudPanel("##quest_tracker", "Objectives", trackerPos, trackerSize))
        {
            DrawQuestTracker(worldState, nearCommandPoint, distanceToCommandPoint, distanceToEncounter);
        }
        ImGui::End();

        if (BeginHudPanel("##stonewake_minimap", "Navigator", minimapPos, minimapSize))
        {
            DrawMinimap(worldState, playerX, playerY);
        }
        ImGui::End();

        if (BeginHudPanel("##action_bar", "Action Bar", actionBarPos, actionBarSize))
        {
            DrawActionBar(
                gameCore,
                worldState,
                hasHostileTarget,
                m_actionSlotBindings,
                m_spellbookBinding,
                m_bagBinding,
                m_settingsBinding,
                m_interactBinding,
                m_targetHostileBinding,
                actionEditMode,
                m_pendingActionAssignmentAbilityId,
                m_pendingActionMoveSlot);
            if (!m_loggedActionBarVisible)
            {
                AZ_Printf("amandacore", "client.action_bar_visible slots=%zu extraUpper=%s right1=%s right2=%s",
                    worldState.m_session.m_actionBarSlots.size(),
                    m_extraUpperActionBarVisible ? "true" : "false",
                    m_rightActionBarOneVisible ? "true" : "false",
                    m_rightActionBarTwoVisible ? "true" : "false");
                m_loggedActionBarVisible = true;
            }
        }
        ImGui::End();

        if (BeginHudPanel("##micro_menu_bar", "Menu", microMenuPos, microMenuSize))
        {
            DrawMicroMenuBar(
                m_characterSheetOpen,
                m_talentsOpen,
                m_questLogOpen,
                m_mapOpen,
                m_spellbookOpen,
                m_bagOpen,
                m_settingsOpen,
                m_socialOpen);
        }
        ImGui::End();

        if (m_extraUpperActionBarVisible && BeginHudPanel("##upper_action_bar", "", upperActionBarPos, upperActionBarSize))
        {
            DrawAuxiliaryActionBar(
                gameCore,
                worldState,
                hasHostileTarget,
                m_actionSlotBindings,
                12,
                false,
                actionEditMode,
                m_pendingActionAssignmentAbilityId,
                m_pendingActionMoveSlot);
        }
        if (m_extraUpperActionBarVisible)
        {
            ImGui::End();
        }

        if (m_rightActionBarTwoVisible && BeginHudPanel("##right_action_bar_two", "R2", rightActionBarTwoPos, rightActionBarSize))
        {
            DrawAuxiliaryActionBar(
                gameCore,
                worldState,
                hasHostileTarget,
                m_actionSlotBindings,
                36,
                true,
                actionEditMode,
                m_pendingActionAssignmentAbilityId,
                m_pendingActionMoveSlot);
        }
        if (m_rightActionBarTwoVisible)
        {
            ImGui::End();
        }

        if (m_rightActionBarOneVisible && BeginHudPanel("##right_action_bar_one", "R1", rightActionBarOnePos, rightActionBarSize))
        {
            DrawAuxiliaryActionBar(
                gameCore,
                worldState,
                hasHostileTarget,
                m_actionSlotBindings,
                24,
                true,
                actionEditMode,
                m_pendingActionAssignmentAbilityId,
                m_pendingActionMoveSlot);
        }
        if (m_rightActionBarOneVisible)
        {
            ImGui::End();
        }

        if (m_bagOpen && BeginHudPanel("##inventory_pack", "Pack", inventoryPos, inventorySize))
        {
            DrawInventoryWindow(gameCore, worldState, m_pendingInventoryMoveSlot);
        }
        if (m_bagOpen)
        {
            ImGui::End();
        }

        if (m_socialOpen && BeginHudPanel("##social_window", "Social", socialPos, socialSize))
        {
            DrawSocialWindow(gameCore, worldState, m_socialNameBuffer, AZ_ARRAY_SIZE(m_socialNameBuffer));
        }
        if (m_socialOpen)
        {
            ImGui::End();
        }

        if (m_settingsOpen && BeginHudPanel("##settings_menu", "Settings", settingsPos, settingsSize))
        {
            if (DrawSettingsWindow(
                    gameCore,
                    m_extraUpperActionBarVisible,
                    m_rightActionBarOneVisible,
                    m_rightActionBarTwoVisible,
                    m_actionSlotBindings,
                    m_spellbookBinding,
                    m_bagBinding,
                    m_characterBinding,
                    m_questLogBinding,
                    m_mapBinding,
                    m_settingsBinding,
                    m_interactBinding,
                    m_targetHostileBinding,
                    m_pendingKeybindActionId))
            {
                SaveUiSettings();
                m_loggedActionBarVisible = false;
            }
        }
        if (m_settingsOpen)
        {
            ImGui::End();
        }

        if (m_spellbookOpen && BeginHudPanel("##spellbook", "Spellbook", spellbookPos, spellbookSize))
        {
            DrawSpellbook(worldState, actionEditMode, m_pendingActionAssignmentAbilityId);
        }
        if (m_spellbookOpen)
        {
            ImGui::End();
        }

        if (m_characterSheetOpen && BeginHudPanel("##character_sheet", "Character", characterPos, characterSize))
        {
            DrawCharacterSheetWindow(worldState);
        }
        if (m_characterSheetOpen)
        {
            ImGui::End();
        }

        if (m_talentsOpen && BeginHudPanel("##talents", "Talents", talentsPos, talentsSize))
        {
            DrawTalentWindow(gameCore, worldState);
        }
        if (m_talentsOpen)
        {
            ImGui::End();
        }

        if (m_questLogOpen && BeginHudPanel("##quest_log", "Quest Log", questLogPos, questLogSize))
        {
            DrawQuestLogWindow(gameCore, worldState);
        }
        if (m_questLogOpen)
        {
            ImGui::End();
        }

        if (m_mapOpen && BeginHudPanel("##zone_map", "Zone Map", mapPos, mapSize))
        {
            DrawZoneMapWindow(worldState, playerX, playerY);
        }
        if (m_mapOpen)
        {
            ImGui::End();
        }

        if (m_trainerOpen && BeginHudPanel("##trainer", "Trainer", trainerPos, trainerSize))
        {
            DrawTrainerWindow(gameCore, worldState);
        }
        if (m_trainerOpen)
        {
            ImGui::End();
        }

        if (m_questGossipOpen && BeginHudPanel("##quest_gossip", "Quest", trainerPos, trainerSize))
        {
            DrawQuestGossipWindow(gameCore, worldState);
        }
        if (m_questGossipOpen)
        {
            ImGui::End();
        }

        DrawFriendlyNpcNameplates(worldState, cameraState, displaySize);

        if (!m_questToast.empty() && nowMs < m_questToastExpiresAt)
        {
            if (BeginHudPanel(
                    "##quest_toast",
                    "Update",
                    ImVec2((displaySize.x * 0.5f) - 220.0f, actionBarPos.y - 92.0f),
                    ImVec2(440.0f, 78.0f)))
            {
                ImGui::TextWrapped("%s", m_questToast.c_str());
            }
            ImGui::End();
        }
    }

    void UiClientSystemComponent::UpdateQuestToast(
        const AZStd::string& questState,
        int currentCount,
        int targetCount,
        int experience,
        int rewardXp,
        int totalCopper,
        int rewardGold,
        int rewardSilver,
        int rewardCopper)
    {
        if (m_lastExperience < 0)
        {
            m_lastQuestState = questState;
            m_lastQuestCount = currentCount;
            m_lastExperience = experience;
            m_lastCurrencyCopper = totalCopper;
            return;
        }

        if (questState != m_lastQuestState)
        {
            if (questState == "active")
            {
                m_questToast = "Field orders accepted";
                m_questToastExpiresAt = NowMs() + 4000;
            }
            else if (questState == "reward_granted")
            {
                m_questToast = AZStd::string::format(
                    "Quest complete: +%d XP, +%dg %ds %dc",
                    rewardXp,
                    rewardGold,
                    rewardSilver,
                    rewardCopper);
                m_questToastExpiresAt = NowMs() + 5000;
            }
        }
        else if (currentCount != m_lastQuestCount && questState == "active")
        {
            m_questToast = AZStd::string::format("Quest progress: %d / %d", currentCount, targetCount);
            m_questToastExpiresAt = NowMs() + 3500;
        }
        else if ((experience != m_lastExperience || totalCopper != m_lastCurrencyCopper) && rewardXp > 0)
        {
            m_questToast = AZStd::string::format(
                "Totals updated: %d XP, %dg %ds %dc",
                experience,
                totalCopper / 10000,
                (totalCopper % 10000) / 100,
                totalCopper % 100);
            m_questToastExpiresAt = NowMs() + 3000;
        }

        m_lastQuestState = questState;
        m_lastQuestCount = currentCount;
        m_lastExperience = experience;
        m_lastCurrencyCopper = totalCopper;
    }

    void UiClientSystemComponent::AddHudEvent(const AZStd::string& message)
    {
        if (message.empty())
        {
            return;
        }

        if (!m_eventLog.empty() && m_eventLog.back() == message)
        {
            return;
        }

        if (m_eventLog.size() >= MaxEventLogEntries)
        {
            m_eventLog.pop_front();
        }
        m_eventLog.push_back(message);
    }
} // namespace UiClient
