#include <UiClient/UiClientSystemComponent.h>

#include <AzCore/Console/IConsole.h>
#include <AzCore/std/containers/unordered_map.h>
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
#include <array>
#include <cstdio>
#include <cstdlib>
#include <cstring>
#include <string>
#include <vector>
#ifndef NOMINMAX
#define NOMINMAX
#endif
#include <Windows.h>
#include <wincodec.h>
#include <wrl/client.h>

#pragma comment(lib, "windowscodecs.lib")

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
        constexpr const char* SteadyStrikeAbilityId = "steady_strike";
        constexpr const char* BraceAbilityId = "brace";
        constexpr const char* DrivingBlowAbilityId = "driving_blow";
        constexpr const char* RallyingCallAbilityId = "rallying_call";
        constexpr const char* HamperingStrikeAbilityId = "hampering_strike";
        constexpr const char* GuardedFormAbilityId = "guarded_form";
        constexpr const char* OverhandCutAbilityId = "overhand_cut";
        constexpr const char* IronResolveAbilityId = "iron_resolve";
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

        struct MapLabelRect
        {
            ImVec2 m_min;
            ImVec2 m_max;
            int m_priority = 99;
        };

        constexpr int PngIconSampleDimension = 32;
        constexpr int PngIconSamplePixelCount = PngIconSampleDimension * PngIconSampleDimension;
        constexpr UINT MaxPngIconSourceDimension = 2048;

        struct PngIconSample
        {
            bool m_attempted = false;
            bool m_loaded = false;
            AZStd::string m_path;
            std::array<ImU32, PngIconSamplePixelCount> m_pixels{};
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

        bool FileExists(const AZStd::string& path)
        {
            const DWORD attributes = GetFileAttributesA(path.c_str());
            return attributes != INVALID_FILE_ATTRIBUTES && (attributes & FILE_ATTRIBUTE_DIRECTORY) == 0;
        }

        AZStd::string JoinPath(const AZStd::string& left, const char* right)
        {
            if (left.empty())
            {
                return right;
            }

            const char last = left[left.size() - 1];
            if (last == '\\' || last == '/')
            {
                return left + right;
            }
            return left + "\\" + right;
        }

        AZStd::string ParentPath(const AZStd::string& path)
        {
            const size_t separator = path.find_last_of("\\/");
            if (separator == AZStd::string::npos || separator == 0)
            {
                return {};
            }
            return path.substr(0, separator);
        }

        AZStd::string FindRepoRootForIcons()
        {
            char currentDirectory[MAX_PATH]{};
            const DWORD length = GetCurrentDirectoryA(MAX_PATH, currentDirectory);
            if (length == 0 || length >= MAX_PATH)
            {
                return {};
            }

            AZStd::string candidate(currentDirectory);
            for (int depth = 0; depth < 9 && !candidate.empty(); ++depth)
            {
                if (FileExists(JoinPath(candidate, "Content\\Art\\Icons\\UI\\icon_missing.png")))
                {
                    return candidate;
                }
                candidate = ParentPath(candidate);
            }

            return {};
        }

        AZStd::string IconPathForKind(const AZStd::string& kind)
        {
            if (kind == "ability_auto_attack")
            {
                return "Content\\Art\\Icons\\Abilities\\ability_auto_attack.png";
            }
            if (kind == "ability_steady_strike")
            {
                return "Content\\Art\\Icons\\Abilities\\ability_steady_strike.png";
            }
            if (kind == "ability_driving_blow")
            {
                return "Content\\Art\\Icons\\Abilities\\ability_driving_blow.png";
            }
            if (kind == "ability_hampering_strike")
            {
                return "Content\\Art\\Icons\\Abilities\\ability_hampering_strike.png";
            }
            if (kind == "ability_brace")
            {
                return "Content\\Art\\Icons\\Abilities\\ability_brace.png";
            }
            if (kind == "ability_rallying_call")
            {
                return "Content\\Art\\Icons\\Abilities\\ability_rallying_call.png";
            }
            if (kind == "item_road_ration")
            {
                return "Content\\Art\\Icons\\Inventory\\item_road_ration.png";
            }
            if (kind == "item_oat_bundle")
            {
                return "Content\\Art\\Icons\\Items\\item_oat_bundle.png";
            }
            if (kind == "currency_copper")
            {
                return "Content\\Art\\Icons\\Inventory\\currency_copper.png";
            }
            if (kind == "item_militia_token")
            {
                return "Content\\Art\\Icons\\Inventory\\item_militia_token.png";
            }
            if (kind == "item_padded_vest")
            {
                return "Content\\Art\\Icons\\Inventory\\item_padded_vest.png";
            }
            if (kind == "item_handwraps")
            {
                return "Content\\Art\\Icons\\Inventory\\item_handwraps.png";
            }
            if (kind == "item_scroll_supplies")
            {
                return "Content\\Art\\Icons\\Inventory\\item_scroll_supplies.png";
            }
            if (kind == "item_ore_chunk")
            {
                return "Content\\Art\\Icons\\Items\\item_ore_chunk.png";
            }
            if (kind == "item_torn_cloth")
            {
                return "Content\\Art\\Icons\\Items\\item_torn_cloth.png";
            }
            if (kind == "item_field_boots")
            {
                return "Content\\Art\\Icons\\Inventory\\item_field_boots.png";
            }
            if (kind == "item_field_dressing")
            {
                return "Content\\Art\\Icons\\Inventory\\item_field_dressing.png";
            }
            if (kind == "menu_spellbook")
            {
                return "Content\\Art\\Icons\\UI\\menu_spellbook.png";
            }
            return "Content\\Art\\Icons\\UI\\icon_missing.png";
        }

        AZStd::string ResolveIconPath(const AZStd::string& kind)
        {
            static const AZStd::string repoRoot = FindRepoRootForIcons();
            if (repoRoot.empty())
            {
                return {};
            }

            const AZStd::string requested = JoinPath(repoRoot, IconPathForKind(kind).c_str());
            if (FileExists(requested))
            {
                return requested;
            }

            const AZStd::string fallback = JoinPath(repoRoot, "Content\\Art\\Icons\\UI\\icon_missing.png");
            return FileExists(fallback) ? fallback : AZStd::string{};
        }

        bool ToWidePath(const AZStd::string& path, std::wstring& outPath)
        {
            const int required = MultiByteToWideChar(CP_UTF8, 0, path.c_str(), -1, nullptr, 0);
            if (required <= 0)
            {
                return false;
            }

            outPath.assign(static_cast<size_t>(required), L'\0');
            const int written = MultiByteToWideChar(CP_UTF8, 0, path.c_str(), -1, outPath.data(), required);
            if (written <= 0)
            {
                outPath.clear();
                return false;
            }
            if (!outPath.empty() && outPath.back() == L'\0')
            {
                outPath.pop_back();
            }
            return true;
        }

        Microsoft::WRL::ComPtr<IWICImagingFactory> CreateWicFactory()
        {
            const HRESULT initResult = CoInitializeEx(nullptr, COINIT_MULTITHREADED);
            if (FAILED(initResult) && initResult != RPC_E_CHANGED_MODE)
            {
                return {};
            }

            Microsoft::WRL::ComPtr<IWICImagingFactory> factory;
            if (FAILED(CoCreateInstance(
                    CLSID_WICImagingFactory,
                    nullptr,
                    CLSCTX_INPROC_SERVER,
                    IID_PPV_ARGS(factory.GetAddressOf()))))
            {
                return {};
            }
            return factory;
        }

        bool LoadPngIconSample(const AZStd::string& path, PngIconSample& outSample)
        {
            std::wstring widePath;
            if (!ToWidePath(path, widePath))
            {
                return false;
            }

            static Microsoft::WRL::ComPtr<IWICImagingFactory> factory = CreateWicFactory();
            if (!factory)
            {
                return false;
            }

            Microsoft::WRL::ComPtr<IWICBitmapDecoder> decoder;
            if (FAILED(factory->CreateDecoderFromFilename(
                    widePath.c_str(),
                    nullptr,
                    GENERIC_READ,
                    WICDecodeMetadataCacheOnLoad,
                    decoder.GetAddressOf())))
            {
                return false;
            }

            Microsoft::WRL::ComPtr<IWICBitmapFrameDecode> frame;
            if (FAILED(decoder->GetFrame(0, frame.GetAddressOf())))
            {
                return false;
            }

            Microsoft::WRL::ComPtr<IWICFormatConverter> converter;
            if (FAILED(factory->CreateFormatConverter(converter.GetAddressOf())))
            {
                return false;
            }
            if (FAILED(converter->Initialize(
                    frame.Get(),
                    GUID_WICPixelFormat32bppRGBA,
                    WICBitmapDitherTypeNone,
                    nullptr,
                    0.0,
                    WICBitmapPaletteTypeCustom)))
            {
                return false;
            }

            UINT width = 0;
            UINT height = 0;
            if (FAILED(converter->GetSize(&width, &height)) ||
                width == 0 ||
                height == 0 ||
                width > MaxPngIconSourceDimension ||
                height > MaxPngIconSourceDimension)
            {
                return false;
            }

            const UINT stride = width * 4;
            const UINT byteCount = stride * height;
            std::vector<unsigned char> rgba(byteCount);
            if (FAILED(converter->CopyPixels(nullptr, stride, byteCount, rgba.data())))
            {
                return false;
            }

            outSample.m_path = path;
            for (int y = 0; y < PngIconSampleDimension; ++y)
            {
                const UINT sampledY = static_cast<UINT>(((y * 2 + 1) * height) / (PngIconSampleDimension * 2));
                const UINT sourceY = sampledY < height ? sampledY : height - 1;
                for (int x = 0; x < PngIconSampleDimension; ++x)
                {
                    const UINT sampledX = static_cast<UINT>(((x * 2 + 1) * width) / (PngIconSampleDimension * 2));
                    const UINT sourceX = sampledX < width ? sampledX : width - 1;
                    const size_t sourceIndex = (static_cast<size_t>(sourceY) * stride) + (static_cast<size_t>(sourceX) * 4);
                    outSample.m_pixels[(y * PngIconSampleDimension) + x] =
                        IM_COL32(rgba[sourceIndex], rgba[sourceIndex + 1], rgba[sourceIndex + 2], rgba[sourceIndex + 3]);
                }
            }
            return true;
        }

        const PngIconSample* GetPngIconSample(const AZStd::string& kind)
        {
            static AZStd::unordered_map<AZStd::string, PngIconSample> iconCache;
            PngIconSample& sample = iconCache[kind];
            if (!sample.m_attempted)
            {
                sample.m_attempted = true;
                const AZStd::string path = ResolveIconPath(kind);
                sample.m_loaded = !path.empty() && LoadPngIconSample(path, sample);
            }
            return sample.m_loaded ? &sample : nullptr;
        }

        ImU32 MutedIconColor(ImU32 color)
        {
            const int red = (color >> IM_COL32_R_SHIFT) & 0xFF;
            const int green = (color >> IM_COL32_G_SHIFT) & 0xFF;
            const int blue = (color >> IM_COL32_B_SHIFT) & 0xFF;
            const int alpha = (color >> IM_COL32_A_SHIFT) & 0xFF;
            return IM_COL32(
                (red + 66) / 2,
                (green + 62) / 2,
                (blue + 58) / 2,
                AZ::GetMin(alpha, 165));
        }

        bool DrawPngIcon(
            ImDrawList* drawList,
            const ImVec2& minBounds,
            const ImVec2& maxBounds,
            const AZStd::string& kind,
            bool muted)
        {
            const PngIconSample* sample = GetPngIconSample(kind);
            const float width = maxBounds.x - minBounds.x;
            const float height = maxBounds.y - minBounds.y;
            if (!sample || width <= 2.0f || height <= 2.0f)
            {
                return false;
            }

            drawList->AddRectFilled(minBounds, maxBounds, ColorU32(13, 16, 20, 255), 7.0f);
            drawList->AddRectFilled(
                ImVec2(minBounds.x + 2.0f, minBounds.y + 2.0f),
                ImVec2(maxBounds.x - 2.0f, maxBounds.y - 2.0f),
                ColorU32(27, 32, 36, 220),
                5.0f);
            const float cellWidth = width / static_cast<float>(PngIconSampleDimension);
            const float cellHeight = height / static_cast<float>(PngIconSampleDimension);
            for (int y = 0; y < PngIconSampleDimension; ++y)
            {
                for (int x = 0; x < PngIconSampleDimension; ++x)
                {
                    const ImU32 pixel = sample->m_pixels[(y * PngIconSampleDimension) + x];
                    const int alpha = (pixel >> IM_COL32_A_SHIFT) & 0xFF;
                    if (alpha < 8)
                    {
                        continue;
                    }
                    const ImU32 color = muted ? MutedIconColor(pixel) : pixel;
                    const ImVec2 cellMin(minBounds.x + (cellWidth * x), minBounds.y + (cellHeight * y));
                    const ImVec2 cellMax(
                        minBounds.x + (cellWidth * (x + 1)) + 0.35f,
                        minBounds.y + (cellHeight * (y + 1)) + 0.35f);
                    drawList->AddRectFilled(cellMin, cellMax, color);
                }
            }
            if (muted)
            {
                drawList->AddRectFilled(minBounds, maxBounds, ColorU32(8, 10, 12, 78), 7.0f);
            }
            drawList->AddRect(minBounds, maxBounds, ColorU32(201, 159, 78, muted ? 120 : 220), 7.0f, 0, 1.5f);
            return true;
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

        AZStd::string PartyCreditStatusLabel(const NetClient::PartyMemberState& member)
        {
            if (member.m_groupCreditStatus == "you")
            {
                return "you";
            }
            if (member.m_groupCreditStatus == "eligible")
            {
                return "shared credit ready";
            }
            if (member.m_groupCreditStatus == "out_of_range")
            {
                return "too far for shared credit";
            }
            if (member.m_groupCreditStatus == "wrong_zone")
            {
                return "wrong zone";
            }
            if (member.m_groupCreditStatus == "not_on_group_quest")
            {
                return "not on tracked group quest";
            }
            if (member.m_groupCreditStatus == "no_group_quest")
            {
                return "no active group quest";
            }
            if (member.m_groupCreditStatus == "offline")
            {
                return "offline";
            }
            return member.m_groupCreditStatus.empty() ? AZStd::string("unknown") : member.m_groupCreditStatus;
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

        AZStd::string FormatRemainingTime(AZ::s64 endsAtMs, AZ::s64 nowMs)
        {
            if (endsAtMs <= nowMs)
            {
                return {};
            }

            const AZ::s64 remainingMs = endsAtMs - nowMs;
            if (remainingMs >= 10000)
            {
                return AZStd::string::format("%llds", static_cast<long long>((remainingMs + 999) / 1000));
            }
            return AZStd::string::format("%.1fs", static_cast<double>(remainingMs) / 1000.0);
        }

        AZStd::string GetAbilityDisplayName(const NetClient::WorldSessionResponse& session, const AZStd::string& abilityId)
        {
            if (abilityId.empty())
            {
                return {};
            }

            for (const auto& slot : session.m_actionBarSlots)
            {
                if (slot.m_abilityId == abilityId && !slot.m_displayName.empty())
                {
                    return slot.m_displayName;
                }
            }
            for (const auto& entry : session.m_spellbookEntries)
            {
                if (entry.m_id == abilityId && !entry.m_displayName.empty())
                {
                    return entry.m_displayName;
                }
            }
            if (abilityId == AutoAttackAbilityId)
            {
                return "Auto Attack";
            }
            if (abilityId == SteadyStrikeAbilityId)
            {
                return "Steady Strike";
            }
            return abilityId;
        }

        AZStd::string FormatAuraLabel(const NetClient::AuraState& aura, AZ::s64 nowMs)
        {
            AZStd::string label = aura.m_displayName.empty() ? aura.m_auraId : aura.m_displayName;
            if (label.empty())
            {
                label = "Aura";
            }
            if (aura.m_stackCount > 1)
            {
                label += AZStd::string::format(" x%d", aura.m_stackCount);
            }
            const AZStd::string remaining = FormatRemainingTime(aura.m_expiresAtMs, nowMs);
            if (!remaining.empty())
            {
                label += AZStd::string::format(" %s", remaining.c_str());
            }
            return label;
        }

        AZStd::string FormatAuraLine(const AZStd::vector<NetClient::AuraState>& auras, AZ::s64 nowMs, size_t maxAuras)
        {
            if (auras.empty())
            {
                return {};
            }

            AZStd::string line;
            const size_t auraCount = std::min(auras.size(), maxAuras);
            for (size_t index = 0; index < auraCount; ++index)
            {
                if (!line.empty())
                {
                    line += ", ";
                }
                line += FormatAuraLabel(auras[index], nowMs);
            }
            if (auras.size() > auraCount)
            {
                line += AZStd::string::format(", +%zu", auras.size() - auraCount);
            }
            return line;
        }

        AZStd::string FormatKillCreditSummary(const AZStd::vector<NetClient::KillCreditState>& credits)
        {
            if (credits.empty())
            {
                return {};
            }

            const auto& credit = credits.back();
            AZStd::string summary = AZStd::string::format("%s x%d", credit.m_archetypeId.c_str(), credit.m_count);
            if (!credit.m_reason.empty())
            {
                summary += AZStd::string::format(" (%s)", credit.m_reason.c_str());
            }
            return summary;
        }

        bool IsCombatEventType(const AZStd::string& type)
        {
            return type.rfind("combat.", 0) == 0 ||
                type.rfind("npc.", 0) == 0 ||
                type.rfind("ability.", 0) == 0 ||
                type.rfind("aura.", 0) == 0 ||
                type.rfind("cooldown.", 0) == 0 ||
                type == "entity.health_changed" ||
                type == "entity.died" ||
                type == "player.died" ||
                type == "progression.kill_credit_awarded" ||
                type == "progression.kill_credit_persisted" ||
                type == "EntityHealthDelta" ||
                type == "EntityCombatStateDelta" ||
                type == "TargetSelectionDelta" ||
                type == "AbilityResultDelta" ||
                type == "EntityDeathDelta" ||
                type == "AuraStateDelta" ||
                type == "CooldownDelta" ||
                type == "CastStateDelta" ||
                type == "ProgressionDelta";
        }

        AZ::s64 MaxEventSequence(const AZStd::vector<NetClient::WorldEventEntry>& events)
        {
            AZ::s64 sequence = 0;
            for (const auto& event : events)
            {
                sequence = AZ::GetMax(sequence, event.m_sequence);
            }
            return sequence;
        }

        bool HasVisibleCooldown(const NetClient::WorldSessionResponse& session, AZ::s64 nowMs)
        {
            if (session.m_globalCooldownEndsAt > nowMs)
            {
                return true;
            }
            for (const auto& slot : session.m_actionBarSlots)
            {
                if (slot.m_cooldownRemainingMs > 0 || slot.m_cooldownEndsAt > nowMs)
                {
                    return true;
                }
            }
            return false;
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

        AZStd::string EntityServiceId(const NetClient::VisibleEntity& entity, const char* serviceType)
        {
            for (const auto& service : entity.m_services)
            {
                if (service.m_type == serviceType)
                {
                    return service.m_serviceId;
                }
            }
            return {};
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
            (void)session;
            (void)actionSlotBindings;
            return AZStd::string::format(
                "%s target  |  %s interact  |  %s spells  |  %s bag  |  %s menu  |  SHIFT edit",
                DisplayKeyName(targetHostileBinding).c_str(),
                DisplayKeyName(interactBinding).c_str(),
                DisplayKeyName(spellbookBinding).c_str(),
                DisplayKeyName(bagBinding).c_str(),
                DisplayKeyName(settingsBinding).c_str());
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

        constexpr float HudButtonHeight = 30.0f;

        ImVec2 HudButtonSize(float width)
        {
            return ImVec2(width, HudButtonHeight);
        }

        bool BeginHudPanel(
            const char* identifier,
            const char* title,
            const ImVec2& position,
            const ImVec2& size,
            bool compact = false,
            bool allowScroll = false)
        {
            ImGui::SetNextWindowPos(position, ImGuiCond_Always);
            ImGui::SetNextWindowSize(size, ImGuiCond_Always);
            ImGui::PushStyleColor(ImGuiCol_WindowBg, ImVec4(0.0f, 0.0f, 0.0f, 0.0f));
            ImGui::PushStyleColor(ImGuiCol_Border, ImVec4(0.0f, 0.0f, 0.0f, 0.0f));
            ImGui::PushStyleVar(ImGuiStyleVar_WindowPadding, compact ? ImVec2(7.0f, 7.0f) : ImVec2(14.0f, 14.0f));
            ImGui::PushStyleVar(ImGuiStyleVar_WindowRounding, compact ? 8.0f : 16.0f);
            ImGuiWindowFlags flags = ImGuiWindowFlags_NoCollapse |
                ImGuiWindowFlags_NoResize |
                ImGuiWindowFlags_NoMove |
                ImGuiWindowFlags_NoTitleBar |
                ImGuiWindowFlags_NoSavedSettings;
            if (!allowScroll)
            {
                flags |= ImGuiWindowFlags_NoScrollbar;
            }
            const bool visible = ImGui::Begin(identifier, nullptr, flags);
            ImGui::PopStyleVar(2);
            ImGui::PopStyleColor(2);
            const ImVec2 windowPosition = ImGui::GetWindowPos();
            const ImVec2 windowSize = ImGui::GetWindowSize();

            if (compact)
            {
                ImGui::GetWindowDrawList()->AddRectFilled(
                    windowPosition,
                    AddVec2(windowPosition, windowSize),
                    ColorU32(10, 14, 19, 212),
                    8.0f);
                ImGui::GetWindowDrawList()->AddRect(
                    windowPosition,
                    AddVec2(windowPosition, windowSize),
                    ColorU32(153, 119, 56, 190),
                    8.0f,
                    0,
                    1.4f);
                ImGui::SetCursorScreenPos(ImVec2(windowPosition.x + 7.0f, windowPosition.y + 7.0f));
            }
            else
            {
                DrawPanelChrome(
                    ImGui::GetWindowDrawList(),
                    windowPosition,
                    AddVec2(windowPosition, windowSize),
                    title);
                ImGui::SetCursorScreenPos(ImVec2(windowPosition.x + 14.0f, windowPosition.y + 40.0f));
            }
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

        AZStd::string AbilityIconKind(const AZStd::string& abilityId)
        {
            if (abilityId == AutoAttackAbilityId)
            {
                return "ability_auto_attack";
            }
            if (abilityId == SteadyStrikeAbilityId)
            {
                return "ability_steady_strike";
            }
            if (abilityId == DrivingBlowAbilityId || abilityId == OverhandCutAbilityId)
            {
                return "ability_driving_blow";
            }
            if (abilityId == HamperingStrikeAbilityId)
            {
                return "ability_hampering_strike";
            }
            if (abilityId == BraceAbilityId || abilityId == GuardedFormAbilityId || abilityId == IronResolveAbilityId)
            {
                return "ability_brace";
            }
            if (abilityId == RallyingCallAbilityId)
            {
                return "ability_rallying_call";
            }
            return "icon_missing";
        }

        AZStd::string ItemIconKind(const NetClient::InventorySlotState& slot)
        {
            if (!slot.m_iconKind.empty())
            {
                return slot.m_iconKind;
            }
            if (slot.m_itemType == "weapon")
            {
                return "ability_auto_attack";
            }
            if (slot.m_itemType == "armor")
            {
                return "item_padded_vest";
            }
            if (slot.m_itemType == "consumable")
            {
                return "item_road_ration";
            }
            if (slot.m_itemType == "material")
            {
                return "item_ore_chunk";
            }
            if (slot.m_itemType == "quest")
            {
                return "item_scroll_supplies";
            }
            if (slot.m_itemType == "junk")
            {
                return "item_torn_cloth";
            }
            return "icon_missing";
        }

        AZStd::string IconFamily(const AZStd::string& kind)
        {
            if (kind == "ability_auto_attack" || kind == "ability_steady_strike" ||
                kind == "ability_driving_blow" || kind == "ability_hampering_strike" ||
                kind == "weapon" || kind == "strike")
            {
                return "strike";
            }
            if (kind == "ability_brace" || kind == "item_padded_vest" || kind == "item_handwraps" ||
                kind == "armor" || kind == "defense")
            {
                return "defense";
            }
            if (kind == "ability_rallying_call" || kind == "item_road_ration" || kind == "item_field_dressing" ||
                kind == "consumable" || kind == "utility")
            {
                return "utility";
            }
            if (kind == "item_oat_bundle" || kind == "menu_spellbook")
            {
                return "nature";
            }
            if (kind == "item_ore_chunk" || kind == "material")
            {
                return "material";
            }
            if (kind == "item_scroll_supplies" || kind == "item_militia_token" || kind == "quest")
            {
                return "quest";
            }
            if (kind == "currency_copper")
            {
                return "currency";
            }
            if (kind == "item_torn_cloth" || kind == "junk")
            {
                return "cloth";
            }
            return "missing";
        }

        void DrawProceduralIcon(
            ImDrawList* drawList,
            const ImVec2& minBounds,
            const ImVec2& maxBounds,
            const AZStd::string& kind,
            bool muted = false)
        {
            if (DrawPngIcon(drawList, minBounds, maxBounds, kind, muted))
            {
                return;
            }

            const AZStd::string family = IconFamily(kind);
            const auto baseColorForFamily = [](const AZStd::string& iconFamily, bool isMuted) -> ImU32
            {
                if (isMuted)
                {
                    return ColorU32(55, 52, 49, 255);
                }
                if (iconFamily == "strike")
                {
                    return ColorU32(94, 58, 44, 255);
                }
                if (iconFamily == "defense")
                {
                    return ColorU32(43, 73, 88, 255);
                }
                if (iconFamily == "utility")
                {
                    return ColorU32(50, 91, 68, 255);
                }
                if (iconFamily == "nature")
                {
                    return ColorU32(34, 79, 48, 255);
                }
                if (iconFamily == "material")
                {
                    return ColorU32(76, 67, 49, 255);
                }
                if (iconFamily == "quest")
                {
                    return ColorU32(91, 73, 35, 255);
                }
                if (iconFamily == "currency")
                {
                    return ColorU32(90, 70, 34, 255);
                }
                if (iconFamily == "cloth")
                {
                    return ColorU32(84, 48, 48, 255);
                }
                return ColorU32(58, 62, 72, 255);
            };
            const auto accentColorForFamily = [](const AZStd::string& iconFamily, bool isMuted) -> ImU32
            {
                if (isMuted)
                {
                    return ColorU32(128, 118, 96, 255);
                }
                if (iconFamily == "strike")
                {
                    return ColorU32(232, 187, 96, 255);
                }
                if (iconFamily == "defense")
                {
                    return ColorU32(146, 203, 221, 255);
                }
                if (iconFamily == "utility")
                {
                    return ColorU32(163, 220, 145, 255);
                }
                if (iconFamily == "nature")
                {
                    return ColorU32(168, 221, 122, 255);
                }
                if (iconFamily == "material")
                {
                    return ColorU32(214, 183, 118, 255);
                }
                if (iconFamily == "quest")
                {
                    return ColorU32(238, 213, 109, 255);
                }
                if (iconFamily == "currency")
                {
                    return ColorU32(239, 191, 86, 255);
                }
                if (iconFamily == "cloth")
                {
                    return ColorU32(221, 108, 91, 255);
                }
                return ColorU32(205, 205, 194, 255);
            };
            const ImU32 baseColor = baseColorForFamily(family, muted);
            const ImU32 accentColor = accentColorForFamily(family, muted);

            const ImVec2 center((minBounds.x + maxBounds.x) * 0.5f, (minBounds.y + maxBounds.y) * 0.5f);
            const float width = maxBounds.x - minBounds.x;
            const float height = maxBounds.y - minBounds.y;
            drawList->AddRectFilled(minBounds, maxBounds, baseColor, 7.0f);
            drawList->AddRect(minBounds, maxBounds, ColorU32(201, 159, 78, muted ? 120 : 210), 7.0f, 0, 1.5f);

            if (family == "missing")
            {
                drawList->AddLine(
                    ImVec2(minBounds.x + width * 0.28f, minBounds.y + height * 0.28f),
                    ImVec2(maxBounds.x - width * 0.28f, maxBounds.y - height * 0.28f),
                    accentColor,
                    3.0f);
                drawList->AddLine(
                    ImVec2(maxBounds.x - width * 0.28f, minBounds.y + height * 0.28f),
                    ImVec2(minBounds.x + width * 0.28f, maxBounds.y - height * 0.28f),
                    accentColor,
                    3.0f);
            }
            else if (family == "strike")
            {
                drawList->AddLine(
                    ImVec2(minBounds.x + width * 0.25f, maxBounds.y - height * 0.23f),
                    ImVec2(maxBounds.x - width * 0.22f, minBounds.y + height * 0.22f),
                    accentColor,
                    3.2f);
                drawList->AddTriangleFilled(
                    ImVec2(maxBounds.x - width * 0.17f, minBounds.y + height * 0.17f),
                    ImVec2(maxBounds.x - width * 0.33f, minBounds.y + height * 0.20f),
                    ImVec2(maxBounds.x - width * 0.20f, minBounds.y + height * 0.34f),
                    accentColor);
            }
            else if (family == "defense")
            {
                drawList->AddTriangleFilled(
                    ImVec2(center.x, minBounds.y + height * 0.18f),
                    ImVec2(maxBounds.x - width * 0.22f, minBounds.y + height * 0.34f),
                    ImVec2(center.x, maxBounds.y - height * 0.16f),
                    accentColor);
                drawList->AddTriangleFilled(
                    ImVec2(center.x, minBounds.y + height * 0.18f),
                    ImVec2(minBounds.x + width * 0.22f, minBounds.y + height * 0.34f),
                    ImVec2(center.x, maxBounds.y - height * 0.16f),
                    ColorU32(100, 151, 171, muted ? 160 : 255));
            }
            else if (family == "utility")
            {
                drawList->AddCircleFilled(center, AZ::GetMin(width, height) * 0.22f, accentColor, 24);
                drawList->AddRectFilled(
                    ImVec2(center.x - width * 0.07f, minBounds.y + height * 0.18f),
                    ImVec2(center.x + width * 0.07f, center.y),
                    ColorU32(226, 237, 202, muted ? 120 : 230),
                    3.0f);
            }
            else if (family == "nature")
            {
                drawList->AddLine(
                    ImVec2(center.x, maxBounds.y - height * 0.18f),
                    ImVec2(center.x, minBounds.y + height * 0.22f),
                    accentColor,
                    3.0f);
                drawList->AddCircleFilled(ImVec2(center.x - width * 0.16f, center.y - height * 0.04f), width * 0.13f, accentColor, 16);
                drawList->AddCircleFilled(ImVec2(center.x + width * 0.16f, center.y - height * 0.10f), width * 0.12f, ColorU32(118, 190, 89, muted ? 140 : 255), 16);
                drawList->AddCircleFilled(ImVec2(center.x - width * 0.04f, center.y - height * 0.20f), width * 0.11f, ColorU32(206, 186, 84, muted ? 130 : 255), 16);
            }
            else if (family == "material")
            {
                drawList->AddCircleFilled(ImVec2(center.x - width * 0.10f, center.y + height * 0.06f), width * 0.16f, accentColor, 16);
                drawList->AddCircleFilled(ImVec2(center.x + width * 0.13f, center.y - height * 0.08f), width * 0.13f, ColorU32(184, 160, 103, muted ? 120 : 255), 16);
            }
            else if (family == "quest")
            {
                drawList->AddRectFilled(
                    ImVec2(minBounds.x + width * 0.28f, minBounds.y + height * 0.18f),
                    ImVec2(maxBounds.x - width * 0.25f, maxBounds.y - height * 0.18f),
                    accentColor,
                    4.0f);
                drawList->AddLine(
                    ImVec2(minBounds.x + width * 0.36f, minBounds.y + height * 0.36f),
                    ImVec2(maxBounds.x - width * 0.34f, minBounds.y + height * 0.36f),
                    baseColor,
                    2.0f);
            }
            else if (family == "currency")
            {
                drawList->AddCircleFilled(ImVec2(center.x - width * 0.12f, center.y + height * 0.04f), width * 0.17f, accentColor, 24);
                drawList->AddCircle(ImVec2(center.x - width * 0.12f, center.y + height * 0.04f), width * 0.17f, ColorU32(255, 236, 149, muted ? 110 : 230), 24, 2.0f);
                drawList->AddCircleFilled(ImVec2(center.x + width * 0.12f, center.y - height * 0.06f), width * 0.15f, ColorU32(221, 151, 66, muted ? 120 : 255), 24);
            }
            else if (family == "cloth")
            {
                drawList->AddQuadFilled(
                    ImVec2(minBounds.x + width * 0.22f, minBounds.y + height * 0.24f),
                    ImVec2(maxBounds.x - width * 0.18f, minBounds.y + height * 0.30f),
                    ImVec2(maxBounds.x - width * 0.28f, maxBounds.y - height * 0.20f),
                    ImVec2(minBounds.x + width * 0.16f, maxBounds.y - height * 0.26f),
                    accentColor);
                drawList->AddLine(
                    ImVec2(minBounds.x + width * 0.25f, center.y),
                    ImVec2(maxBounds.x - width * 0.25f, center.y + height * 0.04f),
                    ColorU32(255, 186, 165, muted ? 120 : 220),
                    2.0f);
            }
            else
            {
                drawList->AddCircleFilled(center, AZ::GetMin(width, height) * 0.22f, accentColor, 20);
            }
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
            const AZ::s64 nowMs = NowMs();
            const AZStd::string castRemaining = FormatRemainingTime(worldState.m_session.m_castEndsAt, nowMs);
            if (!castRemaining.empty())
            {
                ImGui::Text("Casting %s  |  %s", GetAbilityDisplayName(worldState.m_session, worldState.m_session.m_castingAbilityId).c_str(), castRemaining.c_str());
            }
            const AZStd::string auraLine = FormatAuraLine(worldState.m_session.m_auras, nowMs, 2);
            if (!auraLine.empty())
            {
                ImGui::TextWrapped("Auras: %s", auraLine.c_str());
            }
            ImGui::Text("Friendly NPCs %.1fm", distanceToCommandPoint);
        }

        void DrawTargetFrame(
            GameCore::IGameCoreRequests* gameCore,
            const NetClient::VisibleEntity* targetEntity,
            const GameCore::ClientWorldState& worldState,
            float playerX,
            float playerY)
        {
            const ImVec2 origin = ImGui::GetCursorScreenPos();
            if (!targetEntity)
            {
                DrawPortraitBadge("?", ImVec2(origin.x + 34.0f, origin.y + 42.0f), ColorU32(72, 74, 78));
                ImGui::SetCursorScreenPos(ImVec2(origin.x + 74.0f, origin.y + 10.0f));
                ImGui::TextUnformatted("No target selected");
                ImGui::TextUnformatted("Left-click NPCs, players, or hostiles");
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

            if (targetEntity->m_kind == "player")
            {
                DrawPortraitBadge("P", ImVec2(origin.x + 34.0f, origin.y + 42.0f), targetEntity->m_duelOpponent ? ColorU32(181, 70, 62) : ColorU32(68, 103, 159));
                ImGui::SetCursorScreenPos(ImVec2(origin.x + 74.0f, origin.y));
                ImGui::TextUnformatted(targetEntity->m_displayName.c_str());
                ImGui::Text("%s  |  %s", targetEntity->m_duelOpponent ? "Duel opponent" : "Player", targetEntity->m_alive ? "available" : "down");
                DrawMeter("Vitality", static_cast<float>(targetEntity->m_health), static_cast<float>(targetEntity->m_maxHealth), ColorU32(173, 52, 44), ImVec2(160.0f, 18.0f));
                ImGui::TextUnformatted(FormatDistanceState(distanceToTarget).c_str());
                const AZStd::string auraLine = FormatAuraLine(targetEntity->m_auras, NowMs(), 2);
                if (!auraLine.empty())
                {
                    ImGui::TextWrapped("Auras: %s", auraLine.c_str());
                }
                if (worldState.m_session.m_pvp.m_safeZone.m_noDuel)
                {
                    ImGui::TextUnformatted("Safe zone: duels unavailable");
                    return;
                }
                if (!worldState.m_session.m_pvp.m_duelId.empty())
                {
                    if (worldState.m_session.m_pvp.m_duelState == "active" && targetEntity->m_duelOpponent)
                    {
                        if (ImGui::Button("Surrender", ImVec2(104.0f, 24.0f)) && gameCore)
                        {
                            gameCore->SurrenderDuel(worldState.m_session.m_pvp.m_duelId);
                        }
                    }
                    else if (worldState.m_session.m_pvp.m_outgoingDuel)
                    {
                        if (ImGui::Button("Cancel Duel", ImVec2(112.0f, 24.0f)) && gameCore)
                        {
                            gameCore->CancelDuel(worldState.m_session.m_pvp.m_duelId);
                        }
                    }
                    return;
                }
                if (ImGui::Button("Request Duel", ImVec2(116.0f, 24.0f)) && gameCore)
                {
                    gameCore->RequestDuel(targetEntity->m_id, {});
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
            const AZStd::string auraLine = FormatAuraLine(targetEntity->m_auras, NowMs(), 2);
            if (!auraLine.empty())
            {
                ImGui::TextWrapped("Auras: %s", auraLine.c_str());
            }
            if (!targetEntity->m_alive)
            {
                ImGui::TextUnformatted("Defeated. Awaiting authoritative respawn.");
            }
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
            auto tryWorldPoint = [&](float worldX, float worldY, ImVec2& outPoint) -> bool
            {
                const float deltaX = (worldX - playerX) * mapScale;
                const float deltaY = (worldY - playerY) * mapScale;
                const ImVec2 point(center.x + deltaX, center.y - deltaY);
                const float pointDeltaX = point.x - center.x;
                const float pointDeltaY = point.y - center.y;
                if ((pointDeltaX * pointDeltaX) + (pointDeltaY * pointDeltaY) <= (radius - 8.0f) * (radius - 8.0f))
                {
                    outPoint = point;
                    return true;
                }
                return false;
            };
            auto plotWorldPoint = [&](float worldX, float worldY, ImU32 color, float pointRadius)
            {
                ImVec2 point;
                if (tryWorldPoint(worldX, worldY, point))
                {
                    drawList->AddCircleFilled(point, pointRadius, color, 24);
                }
            };

            for (const auto& road : worldState.m_session.m_zoneMap.m_roads)
            {
                for (size_t pointIndex = 1; pointIndex < road.m_points.size(); ++pointIndex)
                {
                    ImVec2 previous;
                    ImVec2 current;
                    if (tryWorldPoint(
                            static_cast<float>(road.m_points[pointIndex - 1].m_x),
                            static_cast<float>(road.m_points[pointIndex - 1].m_y),
                            previous) &&
                        tryWorldPoint(
                            static_cast<float>(road.m_points[pointIndex].m_x),
                            static_cast<float>(road.m_points[pointIndex].m_y),
                            current))
                    {
                        drawList->AddLine(previous, current, ColorU32(92, 74, 42, 230), 5.5f);
                        drawList->AddLine(previous, current, ColorU32(215, 176, 96, 240), 2.0f);
                    }
                }
            }
            for (const auto& landmark : worldState.m_session.m_zoneMap.m_landmarks)
            {
                ImVec2 point;
                if (tryWorldPoint(static_cast<float>(landmark.m_x), static_cast<float>(landmark.m_y), point))
                {
                    drawList->AddRectFilled(
                        ImVec2(point.x - 3.5f, point.y - 3.5f),
                        ImVec2(point.x + 3.5f, point.y + 3.5f),
                        ColorU32(229, 216, 166),
                        1.5f);
                }
            }
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
            for (const auto& entity : worldState.m_session.m_entities)
            {
                if (entity.m_id != worldState.m_session.m_currentTargetId || !entity.m_alive)
                {
                    continue;
                }
                if (entity.m_kind == "hostile_mob")
                {
                    plotWorldPoint(static_cast<float>(entity.m_x), static_cast<float>(entity.m_y), ColorU32(232, 191, 84), 4.0f);
                }
            }

            drawList->AddTriangleFilled(
                ImVec2(center.x, center.y - 10.0f),
                ImVec2(center.x - 7.0f, center.y + 8.0f),
                ImVec2(center.x + 7.0f, center.y + 8.0f),
                ColorU32(226, 240, 243));
            ImGui::TextUnformatted("Markers: quests, trainer, services, current target");
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

        int MapMarkerPriority(const AZStd::string& kind)
        {
            if (kind == "tracked_objective")
            {
                return 0;
            }
            if (kind == "quest_turn_in" || kind == "quest_objective" || kind == "quest_available")
            {
                return 1;
            }
            if (kind == "trainer" || kind == "vendor")
            {
                return 2;
            }
            if (kind == "travel_point" || kind == "bind_point")
            {
                return 3;
            }
            return 4;
        }

        int LandmarkPriority(const AZStd::string& kind)
        {
            if (kind == "hub" || kind == "training")
            {
                return 2;
            }
            if (kind == "handoff")
            {
                return 3;
            }
            return 4;
        }

        bool RectsOverlap(const MapLabelRect& left, const ImVec2& rightMin, const ImVec2& rightMax)
        {
            return left.m_min.x < rightMax.x &&
                left.m_max.x > rightMin.x &&
                left.m_min.y < rightMax.y &&
                left.m_max.y > rightMin.y;
        }

        bool TryDrawMapLabel(
            ImDrawList* drawList,
            AZStd::vector<MapLabelRect>& placedLabels,
            const ImVec2& anchor,
            const char* text,
            ImU32 textColor,
            int priority)
        {
            if (!text || text[0] == '\0')
            {
                return false;
            }

            const ImVec2 textSize = ImGui::CalcTextSize(text);
            const ImVec2 offsets[] = {
                ImVec2(8.0f, -8.0f),
                ImVec2(8.0f, 8.0f),
                ImVec2(-textSize.x - 8.0f, -8.0f),
                ImVec2(-textSize.x - 8.0f, 8.0f),
                ImVec2((-textSize.x * 0.5f), -24.0f),
            };

            for (const ImVec2& offset : offsets)
            {
                const ImVec2 minBounds(anchor.x + offset.x - 3.0f, anchor.y + offset.y - 2.0f);
                const ImVec2 maxBounds(minBounds.x + textSize.x + 6.0f, minBounds.y + textSize.y + 4.0f);
                bool overlapsHigherOrEqual = false;
                for (const auto& placed : placedLabels)
                {
                    if (placed.m_priority <= priority && RectsOverlap(placed, minBounds, maxBounds))
                    {
                        overlapsHigherOrEqual = true;
                        break;
                    }
                }
                if (overlapsHigherOrEqual)
                {
                    continue;
                }

                drawList->AddRectFilled(minBounds, maxBounds, ColorU32(9, 13, 16, 210), 4.0f);
                drawList->AddText(ImVec2(minBounds.x + 3.0f, minBounds.y + 2.0f), textColor, text);
                placedLabels.push_back(MapLabelRect{minBounds, maxBounds, priority});
                return true;
            }

            return false;
        }

        void DrawZoneMapWindow(
            GameCore::IGameCoreRequests* gameCore,
            const GameCore::ClientWorldState& worldState,
            float playerX,
            float playerY)
        {
            const auto& zoneMap = worldState.m_session.m_zoneMap;
            ImGui::Text("%s", zoneMap.m_displayName.empty() ? "Stonewake Vale" : zoneMap.m_displayName.c_str());
            ImGui::TextUnformatted("Authored navigation blockout");

            const ImVec2 availableRegion = ImGui::GetContentRegionAvail();
            const ImVec2 canvasSize(
                AZ::GetMax(590.0f, availableRegion.x),
                AZ::GetMax(360.0f, availableRegion.y - 26.0f));
            const ImVec2 canvasMin = ImGui::GetCursorScreenPos();
            ImGui::InvisibleButton("##stonewake_zone_map_canvas", canvasSize);
            const bool canvasClicked = ImGui::IsItemClicked(ImGuiMouseButton_Left);
            const ImVec2 clickPoint = ImGui::GetIO().MousePos;
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
                    drawList->AddLine(previous, current, ColorU32(88, 65, 36), 7.0f);
                    drawList->AddLine(previous, current, ColorU32(226, 184, 104), 3.0f);
                }
            }

            const NetClient::NavigationAreaState* clickedArea = nullptr;
            float clickedAreaDistance = 999999.0f;
            for (const auto& area : worldState.m_session.m_navigationAreas)
            {
                const ImVec2 center = mapToScreen(area.m_centerX, area.m_centerY);
                const float radius = static_cast<float>((area.m_radius / width) * (canvasSize.x - 36.0f));
                const ImU32 areaColor = area.m_kind == "hostile_objective" ? ColorU32(135, 74, 52, 65) : ColorU32(72, 121, 112, 55);
                drawList->AddCircleFilled(center, AZ::GetClamp(radius, 8.0f, 42.0f), areaColor, 32);
                drawList->AddCircle(center, AZ::GetClamp(radius, 8.0f, 42.0f), ColorU32(178, 166, 124, 130), 32, 1.0f);
                if (canvasClicked && !area.m_questIds.empty())
                {
                    const float deltaX = clickPoint.x - center.x;
                    const float deltaY = clickPoint.y - center.y;
                    const float distance = AZStd::sqrt((deltaX * deltaX) + (deltaY * deltaY));
                    const float clickRadius = AZ::GetClamp(radius, 10.0f, 48.0f);
                    if (distance <= clickRadius && distance < clickedAreaDistance)
                    {
                        clickedArea = &area;
                        clickedAreaDistance = distance;
                    }
                }
            }

            AZStd::vector<MapLabelRect> placedLabels;
            for (const auto& landmark : zoneMap.m_landmarks)
            {
                const ImVec2 point = mapToScreen(landmark.m_x, landmark.m_y);
                drawList->AddRectFilled(
                    ImVec2(point.x - 5.5f, point.y - 5.5f),
                    ImVec2(point.x + 5.5f, point.y + 5.5f),
                    ColorU32(206, 198, 154),
                    2.0f);
                drawList->AddRect(
                    ImVec2(point.x - 7.0f, point.y - 7.0f),
                    ImVec2(point.x + 7.0f, point.y + 7.0f),
                    ColorU32(35, 28, 18),
                    2.0f,
                    0,
                    1.5f);
                TryDrawMapLabel(
                    drawList,
                    placedLabels,
                    point,
                    landmark.m_displayName.c_str(),
                    ColorU32(223, 219, 199),
                    LandmarkPriority(landmark.m_kind));
            }

            AZStd::vector<const NetClient::MapMarkerState*> sortedMarkers;
            sortedMarkers.reserve(worldState.m_session.m_mapMarkers.size());
            for (const auto& marker : worldState.m_session.m_mapMarkers)
            {
                sortedMarkers.push_back(&marker);
            }
            std::sort(
                sortedMarkers.begin(),
                sortedMarkers.end(),
                [](const NetClient::MapMarkerState* left, const NetClient::MapMarkerState* right)
                {
                    return MapMarkerPriority(left->m_kind) < MapMarkerPriority(right->m_kind);
                });

            const NetClient::MapMarkerState* clickedMarker = nullptr;
            float clickedMarkerDistance = 999999.0f;
            for (const auto* marker : sortedMarkers)
            {
                if (!marker)
                {
                    continue;
                }

                const ImVec2 point = mapToScreen(marker->m_x, marker->m_y);
                drawList->AddCircleFilled(point, 6.0f, MapMarkerColor(marker->m_kind), 20);
                drawList->AddCircle(point, 7.5f, ColorU32(21, 25, 27), 20, 1.5f);
                if (canvasClicked)
                {
                    const float deltaX = clickPoint.x - point.x;
                    const float deltaY = clickPoint.y - point.y;
                    const float distance = AZStd::sqrt((deltaX * deltaX) + (deltaY * deltaY));
                    if (distance <= 18.0f && distance < clickedMarkerDistance)
                    {
                        clickedMarker = marker;
                        clickedMarkerDistance = distance;
                    }
                }
                const int priority = MapMarkerPriority(marker->m_kind);
                if (!marker->m_displayName.empty() && priority <= 3)
                {
                    TryDrawMapLabel(
                        drawList,
                        placedLabels,
                        point,
                        marker->m_displayName.c_str(),
                        ColorU32(240, 232, 206),
                        priority);
                }
            }

            const ImVec2 playerPoint = mapToScreen(playerX, playerY);
            drawList->AddTriangleFilled(
                ImVec2(playerPoint.x, playerPoint.y - 9.0f),
                ImVec2(playerPoint.x - 7.0f, playerPoint.y + 7.0f),
                ImVec2(playerPoint.x + 7.0f, playerPoint.y + 7.0f),
                ColorU32(226, 240, 243));

            if (canvasClicked && gameCore)
            {
                if (clickedMarker && !clickedMarker->m_entityId.empty())
                {
                    if (gameCore->SetTarget(clickedMarker->m_entityId))
                    {
                        AZ_Printf(
                            "amandacore",
                            "client.zone_map_marker_selected entityId=%s markerId=%s",
                            clickedMarker->m_entityId.c_str(),
                            clickedMarker->m_id.c_str());
                    }
                }
                else if (clickedMarker && !clickedMarker->m_questId.empty())
                {
                    if (gameCore->TrackQuest(clickedMarker->m_questId, true))
                    {
                        AZ_Printf(
                            "amandacore",
                            "client.zone_map_quest_tracked questId=%s markerId=%s",
                            clickedMarker->m_questId.c_str(),
                            clickedMarker->m_id.c_str());
                    }
                }
                else if (clickedArea && !clickedArea->m_questIds.empty())
                {
                    if (gameCore->TrackQuest(clickedArea->m_questIds[0], true))
                    {
                        AZ_Printf(
                            "amandacore",
                            "client.zone_map_area_tracked questId=%s areaId=%s",
                            clickedArea->m_questIds[0].c_str(),
                            clickedArea->m_areaId.c_str());
                    }
                }
            }

            ImGui::TextUnformatted("Legend: white player, rust objective, gold available, green turn-in, blue trainer, violet vendor. Click markers to target or track.");
        }

        void DrawStonewakeLandmarkNameplates(
            const GameCore::ClientWorldState& worldState,
            const GameCore::ClientCameraState& cameraState,
            const ImVec2& displaySize)
        {
            if (worldState.m_session.m_zoneMap.m_landmarks.empty())
            {
                return;
            }

            ImDrawList* drawList = ImGui::GetForegroundDrawList();
            const float playerX = static_cast<float>(worldState.m_session.m_position.m_x);
            const float playerY = static_cast<float>(worldState.m_session.m_position.m_y);
            for (const auto& landmark : worldState.m_session.m_zoneMap.m_landmarks)
            {
                if (landmark.m_displayName.empty())
                {
                    continue;
                }

                const float landmarkX = static_cast<float>(landmark.m_x);
                const float landmarkY = static_cast<float>(landmark.m_y);
                const float distance = Distance2D(playerX, playerY, landmarkX, landmarkY);
                if (distance > 96.0f && LandmarkPriority(landmark.m_kind) > 2)
                {
                    continue;
                }

                ImVec2 screenPosition;
                if (!ProjectWorldPointToScreen(
                        cameraState,
                        AZ::Vector3(landmarkX, landmarkY, 4.1f),
                        displaySize,
                        screenPosition))
                {
                    continue;
                }

                const ImVec2 textSize = ImGui::CalcTextSize(landmark.m_displayName.c_str());
                const ImVec2 panelMin(screenPosition.x - (textSize.x * 0.5f) - 9.0f, screenPosition.y - 10.0f);
                const ImVec2 panelMax(screenPosition.x + (textSize.x * 0.5f) + 9.0f, screenPosition.y + textSize.y + 5.0f);
                drawList->AddRectFilled(panelMin, panelMax, ColorU32(18, 20, 17, 206), 6.0f);
                drawList->AddRect(panelMin, panelMax, ColorU32(226, 190, 103, 210), 6.0f, 0, 1.5f);
                drawList->AddText(
                    ImVec2(panelMin.x + 9.0f, panelMin.y + 3.0f),
                    ColorU32(241, 229, 196),
                    landmark.m_displayName.c_str());
            }
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
                if (!trackerQuest->m_partyStatusText.empty())
                {
                    ImGui::TextWrapped(
                        "%s Nearby: %d, eligible: %d.",
                        trackerQuest->m_partyStatusText.c_str(),
                        trackerQuest->m_partyNearbyCount,
                        trackerQuest->m_partyEligibleCount);
                }
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
            int aliveHostiles = 0;
            int nearbyHostiles = 0;
            for (const auto& entity : worldState.m_session.m_entities)
            {
                if (entity.m_kind != "hostile_mob")
                {
                    continue;
                }
                ++aliveHostiles;
                const float distanceToPlayer = Distance2D(
                    static_cast<float>(worldState.m_session.m_position.m_x),
                    static_cast<float>(worldState.m_session.m_position.m_y),
                    static_cast<float>(entity.m_x),
                    static_cast<float>(entity.m_y));
                if (distanceToPlayer <= 55.0f)
                {
                    ++nearbyHostiles;
                }
            }
            ImGui::Text("Field threats nearby: %d  |  zone: %d", nearbyHostiles, aliveHostiles);
            ImGui::Spacing();
            ImGui::Separator();
            ImGui::TextUnformatted("Controls");
            ImGui::TextWrapped("WASD move  |  RMB orbit  |  Tab target  |  E interact  |  ESC close/menu");
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

        void DrawCombatEventEntries(
            const AZStd::vector<NetClient::WorldEventEntry>& events,
            const char* prefix,
            int& drawn,
            int maxDrawn)
        {
            for (size_t offset = 0; offset < events.size() && drawn < maxDrawn; ++offset)
            {
                const NetClient::WorldEventEntry& event = events[events.size() - offset - 1];
                if (!IsCombatEventType(event.m_type))
                {
                    continue;
                }

                AZStd::string line = AZStd::string::format("%s %s", prefix, event.m_type.c_str());
                if (!event.m_summary.empty())
                {
                    line += " | ";
                    line += event.m_summary;
                }
                ImGui::BulletText("%s", line.c_str());
                ++drawn;
            }
        }

        void DrawCombatFeed(const GameCore::ClientWorldState& worldState, const NetClient::VisibleEntity* targetEntity)
        {
            ImGui::TextUnformatted("Authoritative Combat");
            ImGui::Separator();

            if (targetEntity)
            {
                ImGui::Text("Target: %s", GetMobDisplayLabel(*targetEntity).c_str());
                ImGui::Text(
                    "Health %.0f / %.0f  |  %s",
                    targetEntity->m_health,
                    targetEntity->m_maxHealth,
                    targetEntity->m_alive ? "alive" : "dead");
            }
            else
            {
                ImGui::TextUnformatted("Target: none");
            }

            const AZ::s64 nowMs = NowMs();
            const AZStd::string gcdRemaining = FormatRemainingTime(worldState.m_session.m_globalCooldownEndsAt, nowMs);
            if (!gcdRemaining.empty())
            {
                ImGui::Text("Global cooldown: %s", gcdRemaining.c_str());
            }
            const AZStd::string playerAuras = FormatAuraLine(worldState.m_session.m_auras, nowMs, 3);
            if (!playerAuras.empty())
            {
                ImGui::TextWrapped("Player auras: %s", playerAuras.c_str());
            }

            const AZStd::string creditSummary = FormatKillCreditSummary(worldState.m_session.m_killCredits);
            if (!creditSummary.empty())
            {
                ImGui::TextWrapped("Kill credit: %s", creditSummary.c_str());
            }

            ImGui::Spacing();
            ImGui::Separator();
            ImGui::TextUnformatted("Server feed");
            int drawn = 0;
            DrawCombatEventEntries(worldState.m_session.m_domainEvents, "event", drawn, 6);
            DrawCombatEventEntries(worldState.m_session.m_stateDiffs, "diff", drawn, 6);
            if (drawn == 0)
            {
                ImGui::TextUnformatted("No combat updates yet.");
            }
        }

        bool BeginSpellbookAbilityDrag(const NetClient::SpellbookEntryState& entry)
        {
            if (!ImGui::BeginDragDropSource(ImGuiDragDropFlags_SourceAllowNullID))
            {
                return false;
            }

            SpellbookAbilityDragPayload payload{};
            std::snprintf(payload.m_abilityId, sizeof(payload.m_abilityId), "%s", entry.m_id.c_str());
            ImGui::SetDragDropPayload(SpellbookAbilityPayloadType, &payload, sizeof(payload));
            ImGui::TextUnformatted(entry.m_displayName.c_str());
            ImGui::TextUnformatted("Assign to action bar");
            ImGui::EndDragDropSource();
            return true;
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

                const bool pressed = ImGui::Button("##action_slot_button", slotSize);
                ImGui::PopStyleColor(3);
                const ImVec2 slotMin = ImGui::GetItemRectMin();
                const ImVec2 slotMax = ImGui::GetItemRectMax();
                if (hasAbility)
                {
                    const AZStd::string iconKind = slotState->m_iconKind.empty()
                        ? AbilityIconKind(slotState->m_abilityId)
                        : slotState->m_iconKind;
                    DrawProceduralIcon(
                        drawList,
                        ImVec2(slotMin.x + 4.0f, slotMin.y + 4.0f),
                        ImVec2(slotMax.x - 4.0f, slotMax.y - 4.0f),
                        iconKind);
                    const size_t shortLabelLength = slotState->m_displayName.size() < 5 ? slotState->m_displayName.size() : 5;
                    const AZStd::string buttonLabel = !slotState->m_buttonLabel.empty()
                        ? slotState->m_buttonLabel
                        : slotState->m_displayName.substr(0, shortLabelLength);
                    const ImVec2 labelSize = ImGui::CalcTextSize(buttonLabel.c_str());
                    drawList->AddRectFilled(
                        ImVec2(slotMin.x + 3.0f, slotMax.y - 18.0f),
                        ImVec2(slotMax.x - 3.0f, slotMax.y - 3.0f),
                        ColorU32(8, 11, 15, 170),
                        4.0f);
                    drawList->AddText(
                        ImVec2(slotMin.x + ((slotSize.x - labelSize.x) * 0.5f), slotMax.y - 17.0f),
                        ColorU32(246, 236, 204),
                        buttonLabel.c_str());
                }
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

                if (hasAbility && ImGui::BeginDragDropSource(ImGuiDragDropFlags_SourceAllowNullID))
                {
                    ActionBarSlotDragPayload payload{};
                    payload.m_sourceSlotIndex = slotIndex;
                    std::snprintf(payload.m_abilityId, sizeof(payload.m_abilityId), "%s", slotState->m_abilityId.c_str());
                    ImGui::SetDragDropPayload(ActionBarSlotPayloadType, &payload, sizeof(payload));
                    ImGui::TextUnformatted(slotState->m_displayName.c_str());
                    ImGui::Text("Move from slot %d", slotIndex + 1);
                    ImGui::EndDragDropSource();
                }

                if (ImGui::BeginDragDropTarget())
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

                if (hasAbility && ImGui::IsItemClicked(ImGuiMouseButton_Right))
                {
                    if (gameCore->ClearActionBarSlot(slotIndex))
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
                        tooltip += "\nSHIFT-click to arm move, drag to move, or right-click to clear.";
                    }
                    else
                    {
                        tooltip += "\nDrag to move this ability, or right-click to clear.";
                    }
                    ImGui::SetTooltip("%s", tooltip.c_str());
                }
                else if (ImGui::IsItemHovered())
                {
                    ImGui::SetTooltip("Drop a learned spellbook ability here, or drag another action slot here.");
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
            const bool clearPressed = ImGui::Button("Clear Slot", ImVec2(112.0f, 26.0f));
            if (clearPressed && pendingActionMoveSlot >= 0)
            {
                if (gameCore->ClearActionBarSlot(pendingActionMoveSlot))
                {
                    AZ_Printf("amandacore", "client.action_bar_clear_requested slot=%d source=clear_drop_target", pendingActionMoveSlot);
                }
                pendingActionMoveSlot = -1;
                pendingActionAssignmentAbilityId.clear();
            }
            if (ImGui::BeginDragDropTarget())
            {
                if (const ImGuiPayload* payload = ImGui::AcceptDragDropPayload(ActionBarSlotPayloadType))
                {
                    const auto* drag = static_cast<const ActionBarSlotDragPayload*>(payload->Data);
                    if (drag && drag->m_sourceSlotIndex >= 0 && gameCore->ClearActionBarSlot(drag->m_sourceSlotIndex))
                    {
                        AZ_Printf(
                            "amandacore",
                            "client.action_bar_clear_requested slot=%d abilityId=%s source=drag_off_bar",
                            drag->m_sourceSlotIndex,
                            drag->m_abilityId);
                        pendingActionMoveSlot = -1;
                        pendingActionAssignmentAbilityId.clear();
                    }
                }
                ImGui::EndDragDropTarget();
            }
            if (ImGui::IsItemHovered())
            {
                ImGui::SetTooltip("Drop an action slot here, or SHIFT-click a slot and press this, to clear it.");
            }
            ImGui::SameLine();
            ImGui::TextDisabled("Drag spells onto slots. Drag an action here to remove it.");

            ImGui::Text(
                "%s interact  |  %s target  |  %s bag  |  %s spells  |  %s menu  |  Hold SHIFT to edit",
                DisplayKeyName(interactBinding).c_str(),
                DisplayKeyName(targetHostileBinding).c_str(),
                DisplayKeyName(bagBinding).c_str(),
                DisplayKeyName(spellbookBinding).c_str(),
                DisplayKeyName(settingsBinding).c_str());
            if (editMode)
            {
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
                ImGui::TextUnformatted("Hold SHIFT to edit bars.");
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
            ImGui::TextUnformatted("Spellbook");
            ImGui::SameLine();
            ImGui::TextDisabled("  |  Warrior / General");
            ImGui::Separator();
            if (editMode && !pendingActionAssignmentAbilityId.empty())
            {
                ImGui::Text("Selected for action bar: %s", pendingActionAssignmentAbilityId.c_str());
            }
            else if (!editMode)
            {
                ImGui::TextWrapped("Drag abilities to action slots. Hold SHIFT for click-to-place.");
            }
            ImGui::Spacing();
            ImGui::BeginChild(
                "##spellbook_scroll",
                ImVec2(0.0f, 0.0f),
                false,
                ImGuiWindowFlags_AlwaysVerticalScrollbar);
            ImGui::Columns(2, "##spellbook_pages", false);
            ImGui::TextUnformatted("Class Abilities");
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
                BeginSpellbookAbilityDrag(entry);
                DrawProceduralIcon(
                    ImGui::GetWindowDrawList(),
                    cardStart,
                    AddVec2(cardStart, ImVec2(42.0f, 42.0f)),
                    entry.m_iconKind.empty() ? AbilityIconKind(entry.m_id) : entry.m_iconKind);
                ImGui::SameLine();
                ImGui::BeginGroup();
                ImGui::PushStyleColor(ImGuiCol_Text, ImVec4(0.86f, 0.90f, 0.76f, 1.0f));
                const bool selected = pendingActionAssignmentAbilityId == entry.m_id;
                if (ImGui::Selectable(entry.m_displayName.c_str(), selected, ImGuiSelectableFlags_SpanAvailWidth) && editMode)
                {
                    pendingActionAssignmentAbilityId = entry.m_id;
                    AZ_Printf("amandacore", "client.spellbook_assignment_armed abilityId=%s", entry.m_id.c_str());
                }
                BeginSpellbookAbilityDrag(entry);
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
                if (ImGui::IsItemHovered())
                {
                    ImGui::SetTooltip(editMode ? "Drag to an action slot, or click to select then click a slot." : "Drag to an action slot.");
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
                DrawProceduralIcon(
                    ImGui::GetWindowDrawList(),
                    cardStart,
                    AddVec2(cardStart, ImVec2(42.0f, 42.0f)),
                    entry.m_iconKind.empty() ? AbilityIconKind(entry.m_id) : entry.m_iconKind,
                    true);
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
                const ImVec2 iconStart = ImGui::GetCursorScreenPos();
                ImGui::InvisibleButton("##trainer_offer_icon", ImVec2(38.0f, 38.0f));
                DrawProceduralIcon(
                    ImGui::GetWindowDrawList(),
                    iconStart,
                    AddVec2(iconStart, ImVec2(38.0f, 38.0f)),
                    offer.m_iconKind.empty() ? AbilityIconKind(offer.m_abilityId) : offer.m_iconKind,
                    offer.m_learned || !offer.m_canLearn);
                ImGui::SameLine();
                ImGui::BeginGroup();
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
                    if (ImGui::Button("Learn", HudButtonSize(92.0f)))
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
                ImGui::EndGroup();
                ImGui::Separator();
                ImGui::PopID();
            }
            ImGui::EndChild();
        }

        const char* DrawQuestGossipWindow(GameCore::IGameCoreRequests* gameCore, const GameCore::ClientWorldState& worldState)
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
                    if (gameCore && gameCore->AcceptQuest(worldState.m_session.m_quest.m_id))
                    {
                        return "accepted";
                    }
                }
            }
            else if (worldState.m_session.m_quest.m_state == "active")
            {
                ImGui::Text(
                    "Progress: %d / %d",
                    worldState.m_session.m_quest.m_currentCount,
                    worldState.m_session.m_quest.m_targetCount);
                const bool objectiveCountReady =
                    worldState.m_session.m_quest.m_currentCount >= worldState.m_session.m_quest.m_targetCount;
                const bool serviceObjective =
                    worldState.m_session.m_quest.m_objectiveType == "talk" ||
                    worldState.m_session.m_quest.m_objectiveType == "trainer" ||
                    worldState.m_session.m_quest.m_objectiveType == "explore" ||
                    worldState.m_session.m_quest.m_objectiveType == "use_location";
                if (objectiveCountReady || serviceObjective)
                {
                    ImGui::TextWrapped("Continue only when this target is the active objective or turn-in.");
                    if (ImGui::Button("Continue Quest", ImVec2(190.0f, 32.0f)))
                    {
                        if (gameCore && gameCore->AcceptQuest(worldState.m_session.m_quest.m_id))
                        {
                            return "turn_in";
                        }
                    }
                }
                else
                {
                    ImGui::TextWrapped("Complete the field objective first.");
                }
            }
            else if (worldState.m_session.m_quest.m_state == "completed")
            {
                ImGui::TextWrapped("The objective is complete. Claim the reward from the turn-in NPC.");
                if (ImGui::Button("Complete Quest", ImVec2(190.0f, 32.0f)))
                {
                    if (gameCore && gameCore->AcceptQuest(worldState.m_session.m_quest.m_id))
                    {
                        return "turn_in";
                    }
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
            return nullptr;
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

            ImGui::BeginChild("##inventory_slots_scroll", ImVec2(0.0f, 0.0f), false, ImGuiWindowFlags_AlwaysVerticalScrollbar);
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
                const bool pressed = ImGui::Button("##inventory_slot_button", slotSize);
                ImGui::PopStyleColor(3);
                const ImVec2 slotMin = ImGui::GetItemRectMin();
                const ImVec2 slotMax = ImGui::GetItemRectMax();
                if (slotState && !slotState->m_itemId.empty() && slotState->m_stackCount > 0)
                {
                    DrawProceduralIcon(
                        ImGui::GetWindowDrawList(),
                        ImVec2(slotMin.x + 8.0f, slotMin.y + 10.0f),
                        ImVec2(slotMax.x - 8.0f, slotMax.y - 12.0f),
                        ItemIconKind(*slotState));
                    if (slotState->m_stackCount > 1)
                    {
                        const AZStd::string stackText = AZStd::string::format("%d", slotState->m_stackCount);
                        const ImVec2 stackSize = ImGui::CalcTextSize(stackText.c_str());
                        ImGui::GetWindowDrawList()->AddText(
                            ImVec2(slotMax.x - stackSize.x - 7.0f, slotMax.y - 18.0f),
                            ColorU32(246, 236, 204),
                            stackText.c_str());
                    }
                    const AZStd::string itemLabel = GetInventorySlotLabel(*slotState);
                    const size_t shortLabelLength = itemLabel.size() < 8 ? itemLabel.size() : 8;
                    ImGui::GetWindowDrawList()->AddRectFilled(
                        ImVec2(slotMin.x + 5.0f, slotMax.y - 18.0f),
                        ImVec2(slotMax.x - 5.0f, slotMax.y - 3.0f),
                        ColorU32(8, 11, 15, 150),
                        4.0f);
                    ImGui::GetWindowDrawList()->AddText(
                        ImVec2(slotMin.x + 7.0f, slotMax.y - 17.0f),
                        ColorU32(231, 220, 190),
                        itemLabel.substr(0, shortLabelLength).c_str());
                }
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
            ImGui::EndChild();
        }

        AZStd::string FormatCopperAmount(int totalCopper)
        {
            const NetClient::CurrencyState currency{
                totalCopper,
                totalCopper % 100,
                (totalCopper % 10000) / 100,
                totalCopper / 10000,
            };
            return FormatCurrency(currency);
        }

        AZStd::string FormatAuctionRemaining(AZ::s64 remainingSeconds)
        {
            if (remainingSeconds <= 0)
            {
                return "expired";
            }
            const AZ::s64 hours = remainingSeconds / 3600;
            AZ::s64 minutes = (remainingSeconds % 3600) / 60;
            if (hours > 0)
            {
                return AZStd::string::format("%lldh %lldm", static_cast<long long>(hours), static_cast<long long>(minutes));
            }
            if (minutes < 1)
            {
                minutes = 1;
            }
            return AZStd::string::format("%lldm", static_cast<long long>(minutes));
        }

        const NetClient::InventorySlotState* FindInventorySlot(
            const NetClient::InventoryState& inventory,
            int slotIndex)
        {
            for (const auto& slot : inventory.m_slots)
            {
                if (slot.m_slotIndex == slotIndex)
                {
                    return &slot;
                }
            }
            return nullptr;
        }

        const NetClient::AuctionSellSlotState* FindAuctionSellSlot(
            const NetClient::AuctionStateResponse& auctionState,
            int slotIndex)
        {
            for (const auto& slot : auctionState.m_sellSlots)
            {
                if (slot.m_slotIndex == slotIndex)
                {
                    return &slot;
                }
            }
            return nullptr;
        }

        void DrawAuctionListingRow(
            GameCore::IGameCoreRequests* gameCore,
            const NetClient::AuctionListingState& listing,
            int rowIndex,
            bool canBuy,
            int& pendingBuyoutIndex)
        {
            ImGui::PushID(listing.m_auctionId.c_str());
            ImGui::Text("%s x%d", listing.m_itemDisplayName.c_str(), listing.m_stackCount);
            ImGui::TextDisabled("%s / %s", listing.m_itemType.c_str(), listing.m_itemSubtype.c_str());
            ImGui::SameLine(210.0f);
            ImGui::Text("%s", listing.m_sellerDisplayName.c_str());
            ImGui::SameLine(360.0f);
            ImGui::Text("%s", FormatCopperAmount(listing.m_buyoutCopper).c_str());
            ImGui::SameLine(490.0f);
            ImGui::Text("%s", FormatAuctionRemaining(listing.m_timeRemainingSeconds).c_str());
            if (canBuy)
            {
                ImGui::SameLine(590.0f);
                if (ImGui::Button("Buyout", HudButtonSize(78.0f)))
                {
                    pendingBuyoutIndex = rowIndex;
                    ImGui::OpenPopup("Confirm Buyout");
                }
            }
            else if (listing.m_state == "active")
            {
                ImGui::SameLine(590.0f);
                if (ImGui::Button("Cancel", HudButtonSize(78.0f)) && gameCore)
                {
                    gameCore->CancelAuction(listing.m_auctionId);
                }
            }
            ImGui::Separator();
            ImGui::PopID();
        }

        void DrawAuctionWindow(
            GameCore::IGameCoreRequests* gameCore,
            const GameCore::ClientWorldState& worldState,
            char* searchBuffer,
            size_t searchBufferLength,
            char* buyoutBuffer,
            size_t buyoutBufferLength,
            int& selectedSellSlot,
            int& stackCount,
            int& pendingBuyoutIndex)
        {
            const char* itemTypes[] = {"", "weapon", "armor", "consumable", "material", "junk"};
            static int selectedItemType = 0;
            static int selectedSort = 0;
            const char* sortLabels[] = {"price asc", "price desc"};
            const char* sortValues[] = {"buyout_asc", "buyout_desc"};
            static int selectedDuration = 1;
            const char* durationLabels[] = {"30 min", "12 hours", "24 hours"};
            const AZ::s64 durationSeconds[] = {30 * 60, 12 * 60 * 60, 24 * 60 * 60};

            ImGui::Text("Highmere Market");
            ImGui::SameLine();
            ImGui::TextDisabled("Purse %s", FormatCurrency(worldState.m_session.m_currency).c_str());
            if (!worldState.m_errorMessage.empty())
            {
                ImGui::TextColored(ImVec4(0.95f, 0.34f, 0.28f, 1.0f), "Status");
                ImGui::SameLine();
                ImGui::TextWrapped("%s", worldState.m_errorMessage.c_str());
            }
            ImGui::Separator();

            if (ImGui::BeginTabBar("##auction_tabs"))
            {
                if (ImGui::BeginTabItem("Browse"))
                {
                    ImGui::SetNextItemWidth(220.0f);
                    ImGui::InputText("Search", searchBuffer, searchBufferLength);
                    ImGui::SameLine();
                    ImGui::SetNextItemWidth(130.0f);
                    ImGui::Combo("Type", &selectedItemType, itemTypes, AZ_ARRAY_SIZE(itemTypes));
                    ImGui::SameLine();
                    ImGui::SetNextItemWidth(120.0f);
                    ImGui::Combo("Sort", &selectedSort, sortLabels, AZ_ARRAY_SIZE(sortLabels));
                    ImGui::SameLine();
                    if (ImGui::Button("Refresh", HudButtonSize(82.0f)) && gameCore)
                    {
                        gameCore->BrowseAuctions(searchBuffer, itemTypes[selectedItemType], sortValues[selectedSort]);
                    }

                    ImGui::Separator();
                    ImGui::TextDisabled("Item");
                    ImGui::SameLine(210.0f);
                    ImGui::TextDisabled("Seller");
                    ImGui::SameLine(360.0f);
                    ImGui::TextDisabled("Buyout");
                    ImGui::SameLine(490.0f);
                    ImGui::TextDisabled("Time");
                    ImGui::Separator();
                    ImGui::BeginChild("##auction_listing_scroll", ImVec2(0.0f, 0.0f), false, ImGuiWindowFlags_AlwaysVerticalScrollbar);
                    for (int index = 0; index < static_cast<int>(worldState.m_auction.m_listings.size()); ++index)
                    {
                        DrawAuctionListingRow(gameCore, worldState.m_auction.m_listings[index], index, true, pendingBuyoutIndex);
                    }
                    ImGui::EndChild();

                    if (ImGui::BeginPopupModal("Confirm Buyout", nullptr, ImGuiWindowFlags_AlwaysAutoResize))
                    {
                        if (pendingBuyoutIndex >= 0 &&
                            pendingBuyoutIndex < static_cast<int>(worldState.m_auction.m_listings.size()))
                        {
                            const auto& listing = worldState.m_auction.m_listings[pendingBuyoutIndex];
                            ImGui::TextWrapped(
                                "Buy %s x%d for %s?",
                                listing.m_itemDisplayName.c_str(),
                                listing.m_stackCount,
                                FormatCopperAmount(listing.m_buyoutCopper).c_str());
                            if (ImGui::Button("Confirm", HudButtonSize(96.0f)) && gameCore)
                            {
                                gameCore->BuyoutAuction(listing.m_auctionId);
                                pendingBuyoutIndex = -1;
                                ImGui::CloseCurrentPopup();
                            }
                            ImGui::SameLine();
                        }
                        if (ImGui::Button("Cancel", HudButtonSize(96.0f)))
                        {
                            pendingBuyoutIndex = -1;
                            ImGui::CloseCurrentPopup();
                        }
                        ImGui::EndPopup();
                    }
                    ImGui::EndTabItem();
                }

                if (ImGui::BeginTabItem("Sell"))
                {
                    ImGui::TextUnformatted("Select an inventory slot or drag an item here.");
                    ImGui::Button("Drop Item", ImVec2(140.0f, 34.0f));
                    if (ImGui::BeginDragDropTarget())
                    {
                        if (const ImGuiPayload* payload = ImGui::AcceptDragDropPayload(InventorySlotPayloadType))
                        {
                            const auto* drag = static_cast<const InventorySlotDragPayload*>(payload->Data);
                            if (drag && drag->m_sourceSlotIndex >= 0)
                            {
                                selectedSellSlot = drag->m_sourceSlotIndex;
                            }
                        }
                        ImGui::EndDragDropTarget();
                    }
                    ImGui::SameLine();
                    if (selectedSellSlot >= 0)
                    {
                        const auto* selectedSlot = FindInventorySlot(worldState.m_session.m_inventory, selectedSellSlot);
                        const auto* sellSlot = FindAuctionSellSlot(worldState.m_auction, selectedSellSlot);
                        ImGui::Text(
                            "Slot %02d: %s x%d",
                            selectedSellSlot + 1,
                            selectedSlot ? selectedSlot->m_displayName.c_str() : "empty",
                            selectedSlot ? selectedSlot->m_stackCount : 0);
                        if (sellSlot && !sellSlot->m_tradeable)
                        {
                            ImGui::SameLine();
                            ImGui::TextColored(
                                ImVec4(0.95f, 0.34f, 0.28f, 1.0f),
                                "%s",
                                sellSlot->m_blockedReason.empty() ? "not tradeable" : sellSlot->m_blockedReason.c_str());
                        }
                    }
                    else
                    {
                        ImGui::TextUnformatted("No item selected");
                    }

                    ImGui::BeginChild("##auction_sell_inventory", ImVec2(250.0f, 220.0f), true);
                    for (const auto& slot : worldState.m_session.m_inventory.m_slots)
                    {
                        if (slot.m_itemId.empty() || slot.m_stackCount <= 0)
                        {
                            continue;
                        }
                        const auto* sellSlot = FindAuctionSellSlot(worldState.m_auction, slot.m_slotIndex);
                        ImGui::PushID(slot.m_slotIndex);
                        const bool selected = selectedSellSlot == slot.m_slotIndex;
                        const bool disabled = sellSlot && !sellSlot->m_tradeable;
                        const AZStd::string label = disabled
                            ? AZStd::string::format("%02d  %s x%d (blocked)", slot.m_slotIndex + 1, slot.m_displayName.c_str(), slot.m_stackCount)
                            : AZStd::string::format("%02d  %s x%d", slot.m_slotIndex + 1, slot.m_displayName.c_str(), slot.m_stackCount);
                        if (ImGui::Selectable(label.c_str(), selected) && !disabled)
                        {
                            selectedSellSlot = slot.m_slotIndex;
                            if (stackCount < 1)
                            {
                                stackCount = 1;
                            }
                            if (stackCount > slot.m_stackCount)
                            {
                                stackCount = slot.m_stackCount;
                            }
                        }
                        if (disabled)
                        {
                            if (ImGui::IsItemHovered())
                            {
                                ImGui::SetTooltip("%s", sellSlot->m_blockedReason.empty() ? "This item cannot be auctioned." : sellSlot->m_blockedReason.c_str());
                            }
                        }
                        ImGui::PopID();
                    }
                    ImGui::EndChild();
                    ImGui::SameLine();
                    ImGui::BeginGroup();
                    ImGui::SetNextItemWidth(120.0f);
                    ImGui::InputInt("Stack", &stackCount);
                    if (stackCount < 1)
                    {
                        stackCount = 1;
                    }
                    ImGui::SetNextItemWidth(150.0f);
                    ImGui::InputText("Buyout copper", buyoutBuffer, buyoutBufferLength, ImGuiInputTextFlags_CharsDecimal);
                    ImGui::SetNextItemWidth(150.0f);
                    ImGui::Combo("Duration", &selectedDuration, durationLabels, AZ_ARRAY_SIZE(durationLabels));
                    const auto* sellSlot = FindAuctionSellSlot(worldState.m_auction, selectedSellSlot);
                    if (sellSlot)
                    {
                        const int clampedStack = stackCount > 0
                            ? (stackCount < sellSlot->m_stackCount ? stackCount : sellSlot->m_stackCount)
                            : 1;
                        int previewDeposit = sellSlot->m_sellPriceCopper * clampedStack / 20;
                        if (previewDeposit <= 0)
                        {
                            previewDeposit = 1;
                        }
                        ImGui::TextDisabled("Deposit %s", FormatCopperAmount(previewDeposit).c_str());
                        if (!sellSlot->m_tradeable)
                        {
                            ImGui::TextColored(
                                ImVec4(0.95f, 0.34f, 0.28f, 1.0f),
                                "%s",
                                sellSlot->m_blockedReason.empty() ? "This item cannot be auctioned." : sellSlot->m_blockedReason.c_str());
                        }
                    }
                    else
                    {
                        ImGui::TextDisabled("Deposit appears after selecting an item.");
                    }
                    if (ImGui::Button("Create Listing", HudButtonSize(150.0f)) && gameCore)
                    {
                        const int buyoutCopper = atoi(buyoutBuffer);
                        if (gameCore->ListAuctionItem(selectedSellSlot, stackCount, buyoutCopper, durationSeconds[selectedDuration]))
                        {
                            selectedSellSlot = -1;
                            stackCount = 1;
                            buyoutBuffer[0] = '\0';
                        }
                    }
                    ImGui::EndGroup();
                    ImGui::EndTabItem();
                }

                if (ImGui::BeginTabItem("My Auctions"))
                {
                    ImGui::BeginChild("##auction_mine_scroll", ImVec2(0.0f, 0.0f), false, ImGuiWindowFlags_AlwaysVerticalScrollbar);
                    for (int index = 0; index < static_cast<int>(worldState.m_auction.m_myAuctions.size()); ++index)
                    {
                        DrawAuctionListingRow(gameCore, worldState.m_auction.m_myAuctions[index], index, false, pendingBuyoutIndex);
                    }
                    ImGui::EndChild();
                    ImGui::EndTabItem();
                }
                ImGui::EndTabBar();
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
            if (ImGui::Button(buttonLabel.c_str(), HudButtonSize(132.0f)))
            {
                pendingKeybindActionId = actionId;
            }
            ImGui::SameLine();
            if (ImGui::Button("Unbind", HudButtonSize(74.0f)))
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
            ImGui::Columns(2, "##character_columns", false);
            ImGui::Text("Race: Human");
            ImGui::Text("Class: Warrior");
            ImGui::Text("Level: %d", worldState.m_session.m_level);
            ImGui::Text("Currency: %s", FormatCurrency(worldState.m_session.m_currency).c_str());
            ImGui::Spacing();
            ImGui::TextUnformatted("Supported equipment");
            ImGui::BulletText("Main hand");
            ImGui::BulletText("Chest");
            ImGui::BulletText("Hands");
            ImGui::BulletText("Legs");
            ImGui::BulletText("Feet");
            ImGui::NextColumn();
            ImGui::TextUnformatted("Stats");
            ImGui::Separator();
            ImGui::Text("Strength: %d", worldState.m_session.m_stats.m_strength);
            ImGui::Text("Stamina: %d", worldState.m_session.m_stats.m_stamina);
            ImGui::Text("Armor: %d", worldState.m_session.m_stats.m_armor);
            ImGui::Text("Attack Power: %.1f", worldState.m_session.m_stats.m_attackPower);
            ImGui::Text("Armor Reduction: %.1f%%", worldState.m_session.m_stats.m_armorReductionPct * 100.0);
            ImGui::Columns(1);
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
                        if (ImGui::Button("Select", HudButtonSize(92.0f)) && gameCore)
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
                    if (!quest.m_partyStatusText.empty())
                    {
                        ImGui::TextWrapped(
                            "Party credit: %s Nearby %d, eligible %d.",
                            quest.m_partyStatusText.c_str(),
                            quest.m_partyNearbyCount,
                            quest.m_partyEligibleCount);
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
                        if (ImGui::Button(buttonLabel, HudButtonSize(96.0f)))
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
            bool& focusRequested,
            bool& inputActive,
            AZStd::string& outSubmittedInput)
        {
            AZ_UNUSED(whisperTargetBuffer);
            AZ_UNUSED(whisperTargetBufferSize);
            selectedChannel = "say";
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
            ImGui::Text("Channel: %s", ChatChannelLabel(selectedChannel));

            if (focusRequested)
            {
                ImGui::SetKeyboardFocusHere();
                focusRequested = false;
            }
            const float sendButtonWidth = 64.0f;
            const float inputWidth = AZ::GetMax(
                120.0f,
                ImGui::GetContentRegionAvail().x - sendButtonWidth - ImGui::GetStyle().ItemSpacing.x);
            ImGui::SetNextItemWidth(inputWidth);
            const bool submittedByEnter = ImGui::InputText(
                "##chat_input",
                inputBuffer,
                inputBufferSize,
                ImGuiInputTextFlags_EnterReturnsTrue);
            inputActive = ImGui::IsItemActive();
            ImGui::SameLine();
            const bool submittedByButton = ImGui::Button("Send", ImVec2(sendButtonWidth, 0.0f));
            if (submittedByEnter || submittedByButton)
            {
                outSubmittedInput = inputBuffer;
                return true;
            }
            return false;
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
                    const AZStd::string creditStatus = PartyCreditStatusLabel(member);
                    if (member.m_sameZone && member.m_distanceToPlayer > 0.0)
                    {
                        ImGui::Text(
                            "%s  |  %.0fm",
                            creditStatus.c_str(),
                            member.m_distanceToPlayer);
                    }
                    else
                    {
                        ImGui::TextUnformatted(creditStatus.c_str());
                    }
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

        void DrawDuelPrompt(GameCore::IGameCoreRequests* gameCore, const GameCore::ClientWorldState& worldState)
        {
            if (!gameCore)
            {
                return;
            }

            const auto& pvp = worldState.m_session.m_pvp;
            if (pvp.m_incomingDuel)
            {
                ImGui::Text("%s challenged you to a duel.", pvp.m_opponentDisplayName.c_str());
                if (ImGui::Button("Accept", ImVec2(104.0f, 28.0f)))
                {
                    gameCore->AcceptDuel(pvp.m_duelId);
                }
                ImGui::SameLine();
                if (ImGui::Button("Decline", ImVec2(104.0f, 28.0f)))
                {
                    gameCore->DeclineDuel(pvp.m_duelId);
                }
                return;
            }

            if (pvp.m_duelState == "countdown")
            {
                AZ::s64 remainingMs = pvp.m_countdownEndsAt - NowMs();
                if (remainingMs < 0)
                {
                    remainingMs = 0;
                }
                ImGui::Text("Duel starts in %.1fs", static_cast<double>(remainingMs) / 1000.0);
                ImGui::Text("%s", pvp.m_opponentDisplayName.c_str());
                if (ImGui::Button("Cancel", ImVec2(104.0f, 28.0f)))
                {
                    gameCore->CancelDuel(pvp.m_duelId);
                }
                return;
            }

            if (pvp.m_duelState == "active")
            {
                ImGui::Text("Dueling %s", pvp.m_opponentDisplayName.c_str());
                ImGui::Text("Wins %d  Losses %d  Honor %d", pvp.m_stats.m_duelsWon, pvp.m_stats.m_duelsLost, pvp.m_stats.m_honorPoints);
                if (ImGui::Button("Surrender", ImVec2(112.0f, 28.0f)))
                {
                    gameCore->SurrenderDuel(pvp.m_duelId);
                }
                return;
            }

            if (!pvp.m_lastResult.m_duelId.empty())
            {
                ImGui::Text(
                    "Last duel: %s vs %s",
                    pvp.m_lastResult.m_result.c_str(),
                    pvp.m_lastResult.m_opponentDisplayName.c_str());
                ImGui::Text("Wins %d  Losses %d  Honor %d", pvp.m_stats.m_duelsWon, pvp.m_stats.m_duelsLost, pvp.m_stats.m_honorPoints);
            }
        }

        bool HasGuildPermission(const NetClient::GuildState& guild, const char* permission)
        {
            return AZStd::find(
                guild.m_currentPermissions.begin(),
                guild.m_currentPermissions.end(),
                AZStd::string(permission)) != guild.m_currentPermissions.end();
        }

        void DrawGuildInvitePrompt(GameCore::IGameCoreRequests* gameCore, const GameCore::ClientWorldState& worldState)
        {
            if (!gameCore || worldState.m_social.m_guildInvites.empty())
            {
                return;
            }

            const auto& invite = worldState.m_social.m_guildInvites.front();
            ImGui::Text("%s invited you to join %s.", invite.m_inviterDisplayName.c_str(), invite.m_guildName.c_str());
            if (ImGui::Button("Accept", ImVec2(104.0f, 28.0f)))
            {
                gameCore->AcceptGuildInvite(invite.m_inviteId);
            }
            ImGui::SameLine();
            if (ImGui::Button("Decline", ImVec2(104.0f, 28.0f)))
            {
                gameCore->DeclineGuildInvite(invite.m_inviteId);
            }
        }

        void DrawSocialWindow(
            GameCore::IGameCoreRequests* gameCore,
            const GameCore::ClientWorldState& worldState,
            char* nameBuffer,
            size_t nameBufferSize,
            char* guildNameBuffer,
            size_t guildNameBufferSize,
            char* guildMotdBuffer,
            size_t guildMotdBufferSize)
        {
            if (ImGui::BeginTabBar("##social_tabs"))
            {
                if (ImGui::BeginTabItem("Friends"))
                {
                    ImGui::SetNextItemWidth(210.0f);
                    ImGui::InputText("Name", nameBuffer, nameBufferSize);
                    ImGui::SameLine();
                    if (ImGui::Button("Add", HudButtonSize(62.0f)) && gameCore)
                    {
                        gameCore->AddFriend(nameBuffer);
                    }
                    ImGui::SameLine();
                    if (ImGui::Button("Remove", HudButtonSize(86.0f)) && gameCore)
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
                            if (ImGui::Button("Invite", HudButtonSize(72.0f)) && gameCore)
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
                    if (ImGui::Button("Invite", HudButtonSize(82.0f)) && gameCore)
                    {
                        gameCore->InviteParty(nameBuffer, {});
                    }
                    ImGui::SameLine();
                    if (ImGui::Button("Leave", HudButtonSize(72.0f)) && gameCore)
                    {
                        gameCore->LeaveParty();
                    }
                    ImGui::Separator();
                    DrawPartyFrames(worldState);
                    ImGui::EndTabItem();
                }

                if (ImGui::BeginTabItem("Guild"))
                {
                    if (!worldState.m_social.m_hasGuild)
                    {
                        ImGui::SetNextItemWidth(250.0f);
                        ImGui::InputText("Guild Name", guildNameBuffer, guildNameBufferSize);
                        ImGui::SameLine();
                        if (ImGui::Button("Create", HudButtonSize(86.0f)) && gameCore)
                        {
                            gameCore->CreateGuild(guildNameBuffer);
                        }
                        ImGui::Separator();
                        ImGui::TextUnformatted("No guild");
                    }
                    else
                    {
                        const auto& guild = worldState.m_social.m_guild;
                        ImGui::Text("%s  |  %s", guild.m_guildName.c_str(), guild.m_currentRankId.c_str());
                        if (!guild.m_messageOfTheDay.empty())
                        {
                            ImGui::TextWrapped("MOTD: %s", guild.m_messageOfTheDay.c_str());
                        }
                        if (guildMotdBuffer[0] == '\0' && !guild.m_messageOfTheDay.empty())
                        {
                            strncpy_s(guildMotdBuffer, guildMotdBufferSize, guild.m_messageOfTheDay.c_str(), _TRUNCATE);
                        }

                        if (HasGuildPermission(guild, "invite_member"))
                        {
                            ImGui::SetNextItemWidth(190.0f);
                            ImGui::InputText("Invite Name", nameBuffer, nameBufferSize);
                            ImGui::SameLine();
                            if (ImGui::Button("Invite", HudButtonSize(82.0f)) && gameCore)
                            {
                                gameCore->InviteGuild(nameBuffer);
                            }
                        }
                        if (HasGuildPermission(guild, "edit_motd"))
                        {
                            ImGui::SetNextItemWidth(286.0f);
                            ImGui::InputText("Message", guildMotdBuffer, guildMotdBufferSize);
                            ImGui::SameLine();
                            if (ImGui::Button("Set", HudButtonSize(58.0f)) && gameCore)
                            {
                                gameCore->SetGuildMessageOfTheDay(guildMotdBuffer);
                            }
                        }

                        if (ImGui::Button("Leave", HudButtonSize(72.0f)) && gameCore)
                        {
                            gameCore->LeaveGuild();
                        }
                        if (HasGuildPermission(guild, "disband_guild"))
                        {
                            ImGui::SameLine();
                            if (ImGui::Button("Disband", HudButtonSize(92.0f)) && gameCore)
                            {
                                gameCore->DisbandGuild();
                            }
                        }

                        ImGui::Separator();
                        ImGui::BeginChild("##guild_roster", ImVec2(0.0f, 250.0f), false, ImGuiWindowFlags_AlwaysVerticalScrollbar);
                        for (const auto& member : guild.m_members)
                        {
                            ImGui::PushID(member.m_characterId.c_str());
                            ImGui::Text(
                                "%s  |  %s  |  L%d %s  |  %s",
                                member.m_displayName.c_str(),
                                member.m_rankName.c_str(),
                                member.m_level,
                                member.m_classId.c_str(),
                                member.m_online ? "online" : "offline");
                            if (!member.m_currentZoneId.empty())
                            {
                                ImGui::SameLine();
                                ImGui::Text(" | %s", member.m_currentZoneId.c_str());
                            }
                            if (member.m_characterId != worldState.m_session.m_characterId)
                            {
                                if (HasGuildPermission(guild, "promote_member"))
                                {
                                    if (ImGui::Button("Promote", HudButtonSize(82.0f)) && gameCore)
                                    {
                                        gameCore->PromoteGuildMember(member.m_displayName);
                                    }
                                    ImGui::SameLine();
                                }
                                if (HasGuildPermission(guild, "demote_member"))
                                {
                                    if (ImGui::Button("Demote", HudButtonSize(78.0f)) && gameCore)
                                    {
                                        gameCore->DemoteGuildMember(member.m_displayName);
                                    }
                                    ImGui::SameLine();
                                }
                                if (HasGuildPermission(guild, "remove_member"))
                                {
                                    if (ImGui::Button("Remove", HudButtonSize(78.0f)) && gameCore)
                                    {
                                        gameCore->RemoveGuildMember(member.m_displayName);
                                    }
                                }
                            }
                            ImGui::Separator();
                            ImGui::PopID();
                        }
                        ImGui::EndChild();
                    }
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
                const char* m_panelName;
                bool* m_toggle;
            };

            talentsOpen = false;
            MenuButtonState buttons[] = {
                {"Char", "character", &characterSheetOpen},
                {"Spells", "spells", &spellbookOpen},
                {"Quests", "quests", &questLogOpen},
                {"Map", "map", &mapOpen},
                {"Bag", "bag", &bagOpen},
                {"Settings", "settings", &settingsOpen},
            };

            for (size_t index = 0; index < AZ_ARRAY_SIZE(buttons); ++index)
            {
                if (index > 0)
                {
                    ImGui::SameLine();
                }
                if (ImGui::Button(buttons[index].m_label, ImVec2(index == 5 ? 74.0f : 58.0f, 28.0f)))
                {
                    *buttons[index].m_toggle = !*buttons[index].m_toggle;
                    AZ_Printf(
                        "amandacore",
                        "client.utility_menu_toggle panel=%s open=%s source=button",
                        buttons[index].m_panelName,
                        *buttons[index].m_toggle ? "true" : "false");
                }
            }
            socialOpen = false;
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
        m_activeInteractionEntityId.clear();
        m_activeInteractionKind.clear();
        m_questToastExpiresAt = 0;
        m_lastHandledInteractionSequence = 0;
        m_questGossipOpen = false;
        m_auctionOpen = false;
        m_lastNearCommandPoint = false;
        m_lastWorldConnected = false;
        m_loggedActionBarVisible = false;
        m_eventLog.clear();
        m_shiftHeld = false;
        m_pendingActionAssignmentAbilityId.clear();
        m_pendingActionMoveSlot = -1;
        m_pendingInventoryMoveSlot = -1;
        m_pendingAuctionSellSlot = -1;
        m_pendingAuctionBuyoutIndex = -1;
        m_auctionStackCount = 1;
        m_chatChannel = "say";
        m_chatFocusRequested = false;
        m_chatInputActive = false;
        m_chatInputBuffer[0] = '\0';
        m_auctionSearchBuffer[0] = '\0';
        m_auctionBuyoutBuffer[0] = '\0';
        LoadDefaultKeybindings();
        LoadUiSettings();
    }

    void UiClientSystemComponent::Deactivate()
    {
        SaveUiSettings();
        ImGui::ImGuiManagerBus::Broadcast(&ImGui::IImGuiManager::SetEnableDiscreteInputMode, false);
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
            if (CloseNpcInteraction("esc"))
            {
                return true;
            }
            if (CloseOpenGameplayPanel("esc"))
            {
                return true;
            }
            m_settingsOpen = !m_settingsOpen;
            AZ_Printf("amandacore", "client.settings_visible open=%s reason=esc", m_settingsOpen ? "true" : "false");
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

    bool UiClientSystemComponent::CloseNpcInteraction(const char* reason)
    {
        const bool questWasOpen = m_questGossipOpen;
        const bool trainerWasOpen = m_trainerOpen;
        const bool auctionWasOpen = m_auctionOpen;
        if (!questWasOpen && !trainerWasOpen && !auctionWasOpen)
        {
            m_activeInteractionEntityId.clear();
            m_activeInteractionKind.clear();
            return false;
        }

        const char* safeReason = reason && reason[0] != '\0' ? reason : "closed";
        if (questWasOpen)
        {
            AZ_Printf(
                "amandacore",
                "client.quest_window_closed reason=%s targetId=%s",
                safeReason,
                m_activeInteractionEntityId.c_str());
        }
        AZ_Printf(
            "amandacore",
            "client.npc_interaction_closed reason=%s targetId=%s kind=%s",
            safeReason,
            m_activeInteractionEntityId.c_str(),
            m_activeInteractionKind.empty() ? "unknown" : m_activeInteractionKind.c_str());

        m_questGossipOpen = false;
        m_trainerOpen = false;
        m_auctionOpen = false;
        m_activeInteractionEntityId.clear();
        m_activeInteractionKind.clear();
        return true;
    }

    bool UiClientSystemComponent::CloseOpenGameplayPanel(const char* reason)
    {
        const bool anyPanelOpen = m_spellbookOpen ||
            m_bagOpen ||
            m_socialOpen ||
            m_characterSheetOpen ||
            m_questLogOpen ||
            m_mapOpen ||
            m_talentsOpen ||
            m_settingsOpen;
        if (!anyPanelOpen)
        {
            return false;
        }

        m_spellbookOpen = false;
        m_bagOpen = false;
        m_socialOpen = false;
        m_characterSheetOpen = false;
        m_questLogOpen = false;
        m_mapOpen = false;
        m_talentsOpen = false;
        m_settingsOpen = false;
        AZ_Printf(
            "amandacore",
            "client.gameplay_panels_closed reason=%s",
            reason && reason[0] != '\0' ? reason : "closed");
        return true;
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

        if (!m_activeInteractionEntityId.empty() && m_activeInteractionEntityId != entity.m_id)
        {
            CloseNpcInteraction("target_changed");
        }

        if (EntityHasService(entity, "dungeon_entrance"))
        {
            const AZStd::string dungeonId = EntityServiceId(entity, "dungeon_entrance");
            if (dungeonId.empty())
            {
                AddHudEvent("Dungeon entrance is unavailable");
                return false;
            }
            if (gameCore->EnterDungeon(dungeonId))
            {
                CloseNpcInteraction("target_changed");
                AddHudEvent(AZStd::string::format("Entering %s", entity.m_displayName.c_str()));
                return true;
            }
            const auto& latestState = gameCore->GetClientWorldState();
            AddHudEvent(latestState.m_errorMessage.empty() ? "Unable to enter dungeon" : latestState.m_errorMessage);
            return false;
        }

        if (EntityHasService(entity, "dungeon_exit"))
        {
            if (gameCore->ExitDungeon())
            {
                CloseNpcInteraction("target_changed");
                AddHudEvent("Leaving dungeon");
                return true;
            }
            const auto& latestState = gameCore->GetClientWorldState();
            AddHudEvent(latestState.m_errorMessage.empty() ? "Unable to leave dungeon" : latestState.m_errorMessage);
            return false;
        }

        if (EntityHasService(entity, "auction"))
        {
            if (gameCore->BrowseAuctions(m_auctionSearchBuffer, {}, "buyout_asc"))
            {
                m_auctionOpen = true;
                m_questGossipOpen = false;
                m_trainerOpen = false;
                m_activeInteractionEntityId = entity.m_id;
                m_activeInteractionKind = "auction";
                AZ_Printf(
                    "amandacore",
                    "client.npc_interaction_opened source=%s targetId=%s kind=auction",
                    source,
                    entity.m_id.c_str());
                AZ_Printf(
                    "amandacore",
                    "client.auction_visible open=true source=%s targetId=%s",
                    source,
                    entity.m_id.c_str());
                AddHudEvent(AZStd::string::format("Viewing %s", entity.m_displayName.c_str()));
                return true;
            }
            const auto& latestState = gameCore->GetClientWorldState();
            AddHudEvent(latestState.m_errorMessage.empty() ? "Unable to open market" : latestState.m_errorMessage);
            return false;
        }

        if (ShouldOpenQuestForEntity(worldState, entity))
        {
            m_questGossipOpen = true;
            m_trainerOpen = false;
            m_auctionOpen = false;
            m_activeInteractionEntityId = entity.m_id;
            m_activeInteractionKind = "quest";
            AZ_Printf(
                "amandacore",
                "client.npc_interaction_opened source=%s targetId=%s kind=quest",
                source,
                entity.m_id.c_str());
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
            m_auctionOpen = false;
            m_activeInteractionEntityId = entity.m_id;
            m_activeInteractionKind = "trainer";
            AZ_Printf(
                "amandacore",
                "client.npc_interaction_opened source=%s targetId=%s kind=trainer",
                source,
                entity.m_id.c_str());
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
            m_auctionOpen = false;
            m_activeInteractionEntityId = entity.m_id;
            m_activeInteractionKind = "quest";
            AZ_Printf(
                "amandacore",
                "client.npc_interaction_opened source=%s targetId=%s kind=quest",
                source,
                entity.m_id.c_str());
            AZ_Printf(
                "amandacore",
                "client.quest_gossip_visible open=true source=%s targetId=%s",
                source,
                entity.m_id.c_str());
            AddHudEvent(AZStd::string::format("Speaking with %s", entity.m_displayName.c_str()));
            return true;
        }

        if (!entity.m_services.empty())
        {
            AddHudEvent(AZStd::string::format("%s service is unavailable in Alpha 0.1", entity.m_displayName.c_str()));
            AZ_Printf(
                "amandacore",
                "client.npc_service_unavailable targetId=%s serviceCount=%zu",
                entity.m_id.c_str(),
                entity.m_services.size());
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
        if (command == "guild" || command == "g")
        {
            return gameCore->SubmitChatMessage("guild", {}, rest);
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
        if (command == "gcreate")
        {
            return gameCore->CreateGuild(rest);
        }
        if (command == "ginvite")
        {
            return gameCore->InviteGuild(rest);
        }
        if (command == "gleave")
        {
            return gameCore->LeaveGuild();
        }
        if (command == "gkick")
        {
            return gameCore->RemoveGuildMember(rest);
        }
        if (command == "gpromote")
        {
            return gameCore->PromoteGuildMember(rest);
        }
        if (command == "gdemote")
        {
            return gameCore->DemoteGuildMember(rest);
        }

        AddHudEvent(AZStd::string::format("Unknown command: /%s", command.c_str()));
        return false;
    }

    void UiClientSystemComponent::BeginChatInput()
    {
        m_chatChannel = "say";
        m_chatInputActive = true;
        m_chatFocusRequested = true;
        ImGui::ImGuiManagerBus::Broadcast(&ImGui::IImGuiManager::SetEnableDiscreteInputMode, true);
    }

    void UiClientSystemComponent::EndChatInput(bool clearBuffer)
    {
        if (clearBuffer)
        {
            m_chatInputBuffer[0] = '\0';
        }
        m_chatInputActive = false;
        m_chatFocusRequested = false;
        ImGui::ClearActiveID();
        ImGui::ImGuiManagerBus::Broadcast(&ImGui::IImGuiManager::SetEnableDiscreteInputMode, false);
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

        if (channelId == AzFramework::InputDeviceKeyboard::Key::Escape && m_chatInputActive)
        {
            EndChatInput(false);
            AddHudEvent("Chat canceled.");
            return true;
        }

        if (channelId == AzFramework::InputDeviceKeyboard::Key::EditEnter)
        {
            if (!m_chatInputActive && !ImGui::GetIO().WantTextInput)
            {
                BeginChatInput();
                return true;
            }
            return false;
        }

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

        if (m_chatInputActive || ImGui::GetIO().WantTextInput)
        {
            return false;
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
            CloseNpcInteraction("disconnect");
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
            m_loggedActionBarCooldownRendered = false;
            m_loggedCombatHudReady = false;
            m_loggedPlayableZoneReady = false;
            m_lastTargetFrameSummary.clear();
            m_lastKillCreditSummary.clear();
            m_lastCombatDomainEventSequence = 0;
            m_lastCombatStateDiffSequence = 0;
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

        if (!m_loggedCombatHudReady)
        {
            AZ_Printf(
                "amandacore",
                "client.combat_hud_ready auras=%zu killCredits=%zu domainEvents=%zu stateDiffs=%zu",
                worldState.m_session.m_auras.size(),
                worldState.m_session.m_killCredits.size(),
                worldState.m_session.m_domainEvents.size(),
                worldState.m_session.m_stateDiffs.size());
            m_loggedCombatHudReady = true;
        }

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
            if (!m_lastWorldSessionToken.empty())
            {
                CloseNpcInteraction("disconnect");
            }
            AddHudEvent(m_lastWorldSessionToken.empty() ? "World session linked" : "World session refreshed");
            m_lastWorldSessionToken = worldState.m_session.m_worldSessionToken;
            m_lastTargetFrameSummary.clear();
            m_lastKillCreditSummary.clear();
            m_lastCombatDomainEventSequence = 0;
            m_lastCombatStateDiffSequence = 0;
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
            const AZStd::string targetFrameSummary = AZStd::string::format(
                "%s|%.0f|%.0f|%s|%zu|%s",
                targetEntity->m_id.c_str(),
                targetEntity->m_health,
                targetEntity->m_maxHealth,
                targetEntity->m_alive ? "alive" : "dead",
                targetEntity->m_auras.size(),
                targetEntity->m_aiState.c_str());
            if (targetFrameSummary != m_lastTargetFrameSummary)
            {
                AZ_Printf(
                    "amandacore",
                    "client.target_frame_updated targetId=%s health=%.0f maxHealth=%.0f alive=%s auras=%zu aiState=%s",
                    targetEntity->m_id.c_str(),
                    targetEntity->m_health,
                    targetEntity->m_maxHealth,
                    targetEntity->m_alive ? "true" : "false",
                    targetEntity->m_auras.size(),
                    targetEntity->m_aiState.c_str());
                if (!targetEntity->m_alive)
                {
                    AddHudEvent(AZStd::string::format("Target defeated: %s", GetMobDisplayLabel(*targetEntity).c_str()));
                }
                m_lastTargetFrameSummary = targetFrameSummary;
            }
        }
        else if (!m_lastHudTargetId.empty())
        {
            AZ_Printf("amandacore", "client.target_hud_cleared");
            AddHudEvent("Target cleared");
            m_lastHudTargetId.clear();
            m_lastTargetFrameSummary.clear();
        }

        const AZStd::string killCreditSummary = FormatKillCreditSummary(worldState.m_session.m_killCredits);
        if (!killCreditSummary.empty() && killCreditSummary != m_lastKillCreditSummary)
        {
            AZ_Printf("amandacore", "client.kill_credit_displayed credit=%s", killCreditSummary.c_str());
            AddHudEvent(AZStd::string::format("Kill credit: %s", killCreditSummary.c_str()));
            m_lastKillCreditSummary = killCreditSummary;
        }

        const AZ::s64 maxDomainSequence = MaxEventSequence(worldState.m_session.m_domainEvents);
        const AZ::s64 maxStateDiffSequence = MaxEventSequence(worldState.m_session.m_stateDiffs);
        if (maxDomainSequence > m_lastCombatDomainEventSequence || maxStateDiffSequence > m_lastCombatStateDiffSequence)
        {
            AZ_Printf(
                "amandacore",
                "client.combat_hud_state_applied domainSeq=%lld stateDiffSeq=%lld domainEvents=%zu stateDiffs=%zu",
                static_cast<long long>(maxDomainSequence),
                static_cast<long long>(maxStateDiffSequence),
                worldState.m_session.m_domainEvents.size(),
                worldState.m_session.m_stateDiffs.size());
            m_lastCombatDomainEventSequence = maxDomainSequence;
            m_lastCombatStateDiffSequence = maxStateDiffSequence;
        }

        if (!m_loggedActionBarCooldownRendered && HasVisibleCooldown(worldState.m_session, nowMs))
        {
            AZ_Printf("amandacore", "client.action_bar_cooldown_rendered");
            m_loggedActionBarCooldownRendered = true;
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
        const ImVec2 playerFrameSize(250.0f, 156.0f);
        const ImVec2 targetFramePos(280.0f, 18.0f);
        const ImVec2 targetFrameSize(300.0f, 168.0f);
        const ImVec2 combatFeedPos(280.0f, 196.0f);
        const ImVec2 combatFeedSize(360.0f, 170.0f);
        const ImVec2 minimapSize(250.0f, 244.0f);
        const ImVec2 minimapPos(displaySize.x - minimapSize.x - 18.0f, 18.0f);
        const ImVec2 rightActionBarSize(66.0f, AZ::GetClamp(displaySize.y - 330.0f, 360.0f, 748.0f));
        const ImVec2 rightActionBarOnePos(displaySize.x - rightActionBarSize.x - 12.0f, 312.0f);
        const ImVec2 rightActionBarTwoPos(rightActionBarOnePos.x - rightActionBarSize.x - 4.0f, 312.0f);
        const ImVec2 trackerSize(292.0f, 292.0f);
        const ImVec2 trackerPos(rightActionBarTwoPos.x - trackerSize.x - 12.0f, 286.0f);
        const ImVec2 actionBarSize(744.0f, 154.0f);
        const ImVec2 actionBarPos(
            (displaySize.x - actionBarSize.x) * 0.5f,
            displaySize.y - actionBarSize.y - 18.0f);
        const ImVec2 upperActionBarSize(744.0f, 72.0f);
        const ImVec2 upperActionBarPos(actionBarPos.x, actionBarPos.y - upperActionBarSize.y - 6.0f);
        const ImVec2 microMenuSize(410.0f, 42.0f);
        const float microMenuRightX = actionBarPos.x + actionBarSize.x + 8.0f;
        const bool microMenuFitsRight = microMenuRightX + microMenuSize.x < rightActionBarTwoPos.x - 12.0f;
        const ImVec2 microMenuPos(
            microMenuFitsRight ? microMenuRightX : actionBarPos.x + actionBarSize.x - microMenuSize.x,
            microMenuFitsRight ? actionBarPos.y + 40.0f : upperActionBarPos.y - microMenuSize.y - 6.0f);
        const ImVec2 spellbookSize(720.0f, 590.0f);
        const ImVec2 spellbookPos(
            AZ::GetMax(18.0f, rightActionBarTwoPos.x - spellbookSize.x - 18.0f),
            AZ::GetMax(18.0f, displaySize.y - spellbookSize.y - 176.0f));
        const ImVec2 trainerSize(460.0f, 440.0f);
        const ImVec2 trainerPos(
            AZ::GetMax(18.0f, spellbookPos.x - trainerSize.x - 18.0f),
            AZ::GetMax(18.0f, displaySize.y - trainerSize.y - 176.0f));
        const ImVec2 inventorySize(330.0f, 430.0f);
        const ImVec2 inventoryPos(
            AZ::GetMax(18.0f, rightActionBarTwoPos.x - inventorySize.x - 14.0f),
            AZ::GetMax(18.0f, displaySize.y - inventorySize.y - 176.0f));
        const ImVec2 settingsSize(680.0f, 520.0f);
        const ImVec2 settingsPos((displaySize.x - settingsSize.x) * 0.5f, (displaySize.y - settingsSize.y) * 0.5f);
        const ImVec2 characterSize(420.0f, 360.0f);
        const ImVec2 characterPos(280.0f, AZ::GetMax(18.0f, displaySize.y - characterSize.y - 176.0f));
        const ImVec2 talentsSize(440.0f, 430.0f);
        const ImVec2 talentsPos(
            AZ::GetMin(characterPos.x + characterSize.x + 18.0f, displaySize.x - talentsSize.x - 18.0f),
            AZ::GetMax(18.0f, displaySize.y - talentsSize.y - 176.0f));
        const ImVec2 questLogSize(460.0f, 360.0f);
        const ImVec2 questLogPos(280.0f, AZ::GetMax(18.0f, displaySize.y - questLogSize.y - 176.0f));
        const ImVec2 mapSize(700.0f, 510.0f);
        const ImVec2 mapPos((displaySize.x - mapSize.x) * 0.5f, (displaySize.y - mapSize.y) * 0.5f);
        const ImVec2 partyFramesSize(250.0f, 250.0f);
        const ImVec2 partyFramesPos(18.0f, 158.0f);
        const ImVec2 chatSize(displaySize.x < 1500.0f ? 360.0f : 440.0f, 230.0f);
        const bool chatWouldOverlapActionBar = 18.0f + chatSize.x > actionBarPos.x - 12.0f;
        const ImVec2 chatPos(18.0f, chatWouldOverlapActionBar ? actionBarPos.y - chatSize.y - 8.0f : displaySize.y - chatSize.y - 18.0f);
        const ImVec2 socialSize(430.0f, 430.0f);
        const ImVec2 socialPos(
            AZ::GetMax(18.0f, rightActionBarTwoPos.x - socialSize.x - 14.0f),
            AZ::GetMax(18.0f, displaySize.y - socialSize.y - 176.0f));
        const ImVec2 auctionSize(720.0f, 520.0f);
        const ImVec2 auctionPos((displaySize.x - auctionSize.x) * 0.5f, (displaySize.y - auctionSize.y) * 0.5f);
        const ImVec2 invitePromptSize(360.0f, 92.0f);
        const ImVec2 invitePromptPos((displaySize.x - invitePromptSize.x) * 0.5f, 170.0f);
        const ImVec2 guildInvitePromptPos((displaySize.x - invitePromptSize.x) * 0.5f, 270.0f);
        const ImVec2 duelPromptSize(380.0f, 108.0f);
        const ImVec2 duelPromptPos((displaySize.x - duelPromptSize.x) * 0.5f, 370.0f);

        const NetClient::VisibleEntity* activeInteractionEntity = nullptr;
        if (!m_activeInteractionEntityId.empty())
        {
            for (const auto& entity : worldState.m_session.m_entities)
            {
                if (entity.m_id == m_activeInteractionEntityId)
                {
                    activeInteractionEntity = &entity;
                    break;
                }
            }
            if (!activeInteractionEntity)
            {
                CloseNpcInteraction("out_of_range");
            }
        }
        if (activeInteractionEntity && worldState.m_session.m_currentTargetId != m_activeInteractionEntityId)
        {
            CloseNpcInteraction("target_changed");
            activeInteractionEntity = nullptr;
        }
        if (activeInteractionEntity)
        {
            const float distanceToInteraction = Distance2D(
                playerX,
                playerY,
                static_cast<float>(activeInteractionEntity->m_x),
                static_cast<float>(activeInteractionEntity->m_y));
            if (distanceToInteraction > CommandPointRadius)
            {
                CloseNpcInteraction("out_of_range");
                activeInteractionEntity = nullptr;
            }
        }
        if (m_questGossipOpen && activeInteractionEntity && !ShouldOpenQuestForEntity(worldState, *activeInteractionEntity))
        {
            const char* closeReason = worldState.m_session.m_quest.m_state == "active"
                ? "accepted"
                : (worldState.m_session.m_quest.m_state == "reward_granted" ? "turn_in" : "target_changed");
            CloseNpcInteraction(closeReason);
            activeInteractionEntity = nullptr;
        }
        if (m_trainerOpen &&
            (worldState.m_session.m_trainer.m_id.empty() ||
                !worldState.m_session.m_trainer.m_inRange ||
                worldState.m_session.m_currentTargetId != worldState.m_session.m_trainer.m_id))
        {
            CloseNpcInteraction("out_of_range");
        }
        if (m_questGossipOpen && (!targetEntity || !IsQuestGiverNpc(*targetEntity)))
        {
            CloseNpcInteraction("target_changed");
        }
        if (m_auctionOpen && (!targetEntity || !EntityHasService(*targetEntity, "auction")))
        {
            CloseNpcInteraction("target_changed");
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

        if (BeginHudPanel("##combat_feed", "Combat", combatFeedPos, combatFeedSize))
        {
            DrawCombatFeed(worldState, targetEntity);
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
            bool chatFieldActive = false;
            if (DrawChatWindow(
                    worldState,
                    m_chatChannel,
                    m_chatInputBuffer,
                    AZ_ARRAY_SIZE(m_chatInputBuffer),
                    m_chatWhisperTargetBuffer,
                    AZ_ARRAY_SIZE(m_chatWhisperTargetBuffer),
                    m_chatFocusRequested,
                    chatFieldActive,
                    submittedInput))
            {
                const bool hasText = submittedInput.find_first_not_of(" \t\r\n") != AZStd::string::npos;
                if (hasText && SubmitChatInput(gameCore, submittedInput))
                {
                    EndChatInput(true);
                }
                else if (hasText)
                {
                    AddHudEvent("Chat message could not be sent.");
                    BeginChatInput();
                }
                else
                {
                    EndChatInput(true);
                }
            }
            else if (chatFieldActive && !m_chatInputActive)
            {
                BeginChatInput();
            }
            else if (m_chatInputActive && !chatFieldActive && !m_chatFocusRequested && !ImGui::IsAnyItemActive())
            {
                EndChatInput(false);
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

        if (!worldState.m_social.m_guildInvites.empty() && BeginHudPanel("##guild_invite_prompt", "Guild Invite", guildInvitePromptPos, invitePromptSize))
        {
            DrawGuildInvitePrompt(gameCore, worldState);
        }
        if (!worldState.m_social.m_guildInvites.empty())
        {
            ImGui::End();
        }

        const bool duelPanelVisible = worldState.m_session.m_pvp.m_incomingDuel ||
            worldState.m_session.m_pvp.m_duelState == "countdown" ||
            worldState.m_session.m_pvp.m_duelState == "active" ||
            !worldState.m_session.m_pvp.m_lastResult.m_duelId.empty();
        if (duelPanelVisible && BeginHudPanel("##duel_prompt", "Duel", duelPromptPos, duelPromptSize))
        {
            DrawDuelPrompt(gameCore, worldState);
        }
        if (duelPanelVisible)
        {
            ImGui::End();
        }

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

        if (BeginHudPanel("##micro_menu_bar", "", microMenuPos, microMenuSize, true))
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

        if (m_extraUpperActionBarVisible && BeginHudPanel("##upper_action_bar", "", upperActionBarPos, upperActionBarSize, true))
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

        if (m_rightActionBarTwoVisible && BeginHudPanel("##right_action_bar_two", "", rightActionBarTwoPos, rightActionBarSize, true))
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

        if (m_rightActionBarOneVisible && BeginHudPanel("##right_action_bar_one", "", rightActionBarOnePos, rightActionBarSize, true))
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

        if (m_bagOpen && BeginHudPanel("##inventory_pack", "Pack", inventoryPos, inventorySize, false, true))
        {
            DrawInventoryWindow(gameCore, worldState, m_pendingInventoryMoveSlot);
        }
        if (m_bagOpen)
        {
            ImGui::End();
        }

        if (m_socialOpen && BeginHudPanel("##social_window", "Social", socialPos, socialSize, false, true))
        {
            DrawSocialWindow(
                gameCore,
                worldState,
                m_socialNameBuffer,
                AZ_ARRAY_SIZE(m_socialNameBuffer),
                m_guildNameBuffer,
                AZ_ARRAY_SIZE(m_guildNameBuffer),
                m_guildMotdBuffer,
                AZ_ARRAY_SIZE(m_guildMotdBuffer));
        }
        if (m_socialOpen)
        {
            ImGui::End();
        }

        if (m_settingsOpen && BeginHudPanel("##settings_menu", "Settings", settingsPos, settingsSize, false, true))
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

        if (m_spellbookOpen && BeginHudPanel("##spellbook", "Spellbook", spellbookPos, spellbookSize, false, true))
        {
            DrawSpellbook(worldState, actionEditMode, m_pendingActionAssignmentAbilityId);
        }
        if (m_spellbookOpen)
        {
            ImGui::End();
        }

        if (m_characterSheetOpen && BeginHudPanel("##character_sheet", "Character", characterPos, characterSize, false, true))
        {
            DrawCharacterSheetWindow(worldState);
        }
        if (m_characterSheetOpen)
        {
            ImGui::End();
        }

        if (m_talentsOpen && BeginHudPanel("##talents", "Talents", talentsPos, talentsSize, false, true))
        {
            DrawTalentWindow(gameCore, worldState);
        }
        if (m_talentsOpen)
        {
            ImGui::End();
        }

        if (m_questLogOpen && BeginHudPanel("##quest_log", "Quest Log", questLogPos, questLogSize, false, true))
        {
            DrawQuestLogWindow(gameCore, worldState);
        }
        if (m_questLogOpen)
        {
            ImGui::End();
        }

        if (m_mapOpen && BeginHudPanel("##zone_map", "Zone Map", mapPos, mapSize, false, true))
        {
            DrawZoneMapWindow(gameCore, worldState, playerX, playerY);
        }
        if (m_mapOpen)
        {
            ImGui::End();
        }

        if (m_auctionOpen && BeginHudPanel("##auction_house", "Market", auctionPos, auctionSize, false, true))
        {
            DrawAuctionWindow(
                gameCore,
                worldState,
                m_auctionSearchBuffer,
                AZ_ARRAY_SIZE(m_auctionSearchBuffer),
                m_auctionBuyoutBuffer,
                AZ_ARRAY_SIZE(m_auctionBuyoutBuffer),
                m_pendingAuctionSellSlot,
                m_auctionStackCount,
                m_pendingAuctionBuyoutIndex);
        }
        if (m_auctionOpen)
        {
            ImGui::End();
        }

        if (m_trainerOpen && BeginHudPanel("##trainer", "Trainer", trainerPos, trainerSize, false, true))
        {
            DrawTrainerWindow(gameCore, worldState);
        }
        if (m_trainerOpen)
        {
            ImGui::End();
        }

        if (m_questGossipOpen && BeginHudPanel("##quest_gossip", "Quest", trainerPos, trainerSize, false, true))
        {
            if (const char* closeReason = DrawQuestGossipWindow(gameCore, worldState))
            {
                CloseNpcInteraction(closeReason);
            }
        }
        if (m_questGossipOpen)
        {
            ImGui::End();
        }

        DrawFriendlyNpcNameplates(worldState, cameraState, displaySize);
        DrawStonewakeLandmarkNameplates(worldState, cameraState, displaySize);

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
