#include <NetClient/WorldHttpClient.h>

#include <AzCore/Debug/Trace.h>
#include <AzCore/std/containers/vector.h>
#include <AzCore/std/string/conversions.h>
#include <AzCore/std/string/string.h>
#include <rapidjson/document.h>

#define WIN32_LEAN_AND_MEAN
#include <Windows.h>
#include <winhttp.h>

namespace NetClient
{
    namespace
    {
        constexpr const char* LogName = "amandacore";

        AZStd::wstring ToWideString(const AZStd::string& text)
        {
            if (text.empty())
            {
                return {};
            }

            const int size = MultiByteToWideChar(CP_UTF8, 0, text.c_str(), -1, nullptr, 0);
            if (size <= 0)
            {
                return {};
            }

            AZStd::wstring result;
            result.resize_no_construct(static_cast<size_t>(size - 1));
            MultiByteToWideChar(CP_UTF8, 0, text.c_str(), -1, result.data(), size);
            return result;
        }

        AZStd::string FormatWinHttpError(const char* operation)
        {
            return AZStd::string::format("%s failed with WinHTTP error %lu.", operation, GetLastError());
        }

        struct ParsedEndpoint
        {
            AZStd::wstring m_host;
            INTERNET_PORT m_port = INTERNET_DEFAULT_HTTP_PORT;
            bool m_secure = false;
        };

        bool ParseEndpoint(const AZStd::string& endpoint, ParsedEndpoint& outEndpoint, AZStd::string& outError)
        {
            URL_COMPONENTSW components{};
            components.dwStructSize = sizeof(components);
            components.dwSchemeLength = static_cast<DWORD>(-1);
            components.dwHostNameLength = static_cast<DWORD>(-1);

            const AZStd::wstring wideEndpoint = ToWideString(endpoint);
            if (!WinHttpCrackUrl(wideEndpoint.c_str(), 0, 0, &components))
            {
                outError = FormatWinHttpError("WinHttpCrackUrl");
                return false;
            }

            outEndpoint.m_host.assign(components.lpszHostName, components.dwHostNameLength);
            outEndpoint.m_port = components.nPort;
            outEndpoint.m_secure = components.nScheme == INTERNET_SCHEME_HTTPS;
            return true;
        }

        bool PerformRequest(
            const AZStd::string& endpoint,
            const wchar_t* method,
            const AZStd::wstring& path,
            const AZStd::string& requestBody,
            AZStd::string& responseBody,
            AZ::u32& outStatusCode,
            AZStd::string& outError)
        {
            ParsedEndpoint parsedEndpoint;
            if (!ParseEndpoint(endpoint, parsedEndpoint, outError))
            {
                return false;
            }

            HINTERNET session = WinHttpOpen(
                L"amandacore-o3de/0.2",
                WINHTTP_ACCESS_TYPE_AUTOMATIC_PROXY,
                WINHTTP_NO_PROXY_NAME,
                WINHTTP_NO_PROXY_BYPASS,
                0);
            if (!session)
            {
                outError = FormatWinHttpError("WinHttpOpen");
                return false;
            }

            HINTERNET connection = WinHttpConnect(session, parsedEndpoint.m_host.c_str(), parsedEndpoint.m_port, 0);
            if (!connection)
            {
                outError = FormatWinHttpError("WinHttpConnect");
                WinHttpCloseHandle(session);
                return false;
            }

            const DWORD requestFlags = parsedEndpoint.m_secure ? WINHTTP_FLAG_SECURE : 0;
            HINTERNET request = WinHttpOpenRequest(
                connection,
                method,
                path.c_str(),
                nullptr,
                WINHTTP_NO_REFERER,
                WINHTTP_DEFAULT_ACCEPT_TYPES,
                requestFlags);
            if (!request)
            {
                outError = FormatWinHttpError("WinHttpOpenRequest");
                WinHttpCloseHandle(connection);
                WinHttpCloseHandle(session);
                return false;
            }

            const wchar_t* headers = L"Content-Type: application/json\r\n";
            const DWORD headerLength = requestBody.empty() ? 0 : static_cast<DWORD>(wcslen(headers));
            const DWORD bodyLength = static_cast<DWORD>(requestBody.size());
            if (!WinHttpSendRequest(
                    request,
                    requestBody.empty() ? WINHTTP_NO_ADDITIONAL_HEADERS : headers,
                    headerLength,
                    requestBody.empty() ? WINHTTP_NO_REQUEST_DATA : const_cast<char*>(requestBody.data()),
                    bodyLength,
                    bodyLength,
                    0))
            {
                outError = FormatWinHttpError("WinHttpSendRequest");
                WinHttpCloseHandle(request);
                WinHttpCloseHandle(connection);
                WinHttpCloseHandle(session);
                return false;
            }

            if (!WinHttpReceiveResponse(request, nullptr))
            {
                outError = FormatWinHttpError("WinHttpReceiveResponse");
                WinHttpCloseHandle(request);
                WinHttpCloseHandle(connection);
                WinHttpCloseHandle(session);
                return false;
            }

            DWORD statusCodeSize = sizeof(outStatusCode);
            if (!WinHttpQueryHeaders(
                    request,
                    WINHTTP_QUERY_STATUS_CODE | WINHTTP_QUERY_FLAG_NUMBER,
                    WINHTTP_HEADER_NAME_BY_INDEX,
                    &outStatusCode,
                    &statusCodeSize,
                    WINHTTP_NO_HEADER_INDEX))
            {
                outError = FormatWinHttpError("WinHttpQueryHeaders");
                WinHttpCloseHandle(request);
                WinHttpCloseHandle(connection);
                WinHttpCloseHandle(session);
                return false;
            }

            responseBody.clear();
            DWORD bytesAvailable = 0;
            do
            {
                if (!WinHttpQueryDataAvailable(request, &bytesAvailable))
                {
                    outError = FormatWinHttpError("WinHttpQueryDataAvailable");
                    WinHttpCloseHandle(request);
                    WinHttpCloseHandle(connection);
                    WinHttpCloseHandle(session);
                    return false;
                }

                if (bytesAvailable > 0)
                {
                    AZStd::vector<char> buffer(bytesAvailable);
                    DWORD bytesRead = 0;
                    if (!WinHttpReadData(request, buffer.data(), bytesAvailable, &bytesRead))
                    {
                        outError = FormatWinHttpError("WinHttpReadData");
                        WinHttpCloseHandle(request);
                        WinHttpCloseHandle(connection);
                        WinHttpCloseHandle(session);
                        return false;
                    }

                    responseBody.append(buffer.data(), buffer.data() + bytesRead);
                }
            } while (bytesAvailable > 0);

            WinHttpCloseHandle(request);
            WinHttpCloseHandle(connection);
            WinHttpCloseHandle(session);
            return true;
        }

        bool ReadString(const rapidjson::Value& object, const char* name, AZStd::string& outValue)
        {
            if (!object.HasMember(name) || !object[name].IsString())
            {
                return false;
            }

            outValue = object[name].GetString();
            return true;
        }

        bool ReadBool(const rapidjson::Value& object, const char* name, bool& outValue)
        {
            if (!object.HasMember(name) || !object[name].IsBool())
            {
                return false;
            }

            outValue = object[name].GetBool();
            return true;
        }

        bool ReadInt64(const rapidjson::Value& object, const char* name, AZ::s64& outValue)
        {
            if (!object.HasMember(name) || !object[name].IsInt64())
            {
                return false;
            }

            outValue = object[name].GetInt64();
            return true;
        }

        bool ReadInt(const rapidjson::Value& object, const char* name, int& outValue)
        {
            if (!object.HasMember(name) || !object[name].IsInt())
            {
                return false;
            }

            outValue = object[name].GetInt();
            return true;
        }

        bool ReadDouble(const rapidjson::Value& object, const char* name, double& outValue)
        {
            if (!object.HasMember(name) || !object[name].IsNumber())
            {
                return false;
            }

            outValue = object[name].GetDouble();
            return true;
        }

        AZStd::string JsonEscape(const AZStd::string& text)
        {
            AZStd::string escaped;
            escaped.reserve(text.size() + 8);
            for (unsigned char value : text)
            {
                switch (value)
                {
                case '\\':
                    escaped += "\\\\";
                    break;
                case '"':
                    escaped += "\\\"";
                    break;
                case '\b':
                    escaped += "\\b";
                    break;
                case '\f':
                    escaped += "\\f";
                    break;
                case '\n':
                    escaped += "\\n";
                    break;
                case '\r':
                    escaped += "\\r";
                    break;
                case '\t':
                    escaped += "\\t";
                    break;
                default:
                    if (value < 0x20)
                    {
                        escaped += AZStd::string::format("\\u%04x", static_cast<unsigned int>(value));
                    }
                    else
                    {
                        escaped.push_back(static_cast<char>(value));
                    }
                    break;
                }
            }
            return escaped;
        }

        void ParseQuestState(const rapidjson::Value& quest, QuestState& outQuest)
        {
            ReadString(quest, "id", outQuest.m_id);
            ReadString(quest, "title", outQuest.m_title);
            ReadString(quest, "category", outQuest.m_category);
            ReadString(quest, "statusBucket", outQuest.m_statusBucket);
            ReadString(quest, "objectiveType", outQuest.m_objectiveType);
            ReadString(quest, "objectiveText", outQuest.m_objectiveText);
            ReadString(quest, "state", outQuest.m_state);
            ReadString(quest, "giverNpcId", outQuest.m_giverNpcId);
            ReadString(quest, "turnInNpcId", outQuest.m_turnInNpcId);
            ReadBool(quest, "tracked", outQuest.m_tracked);
            ReadBool(quest, "partyShareable", outQuest.m_partyShareable);
            ReadBool(quest, "groupRecommended", outQuest.m_groupRecommended);
            ReadInt(quest, "currentCount", outQuest.m_currentCount);
            ReadInt(quest, "targetCount", outQuest.m_targetCount);
            ReadInt(quest, "recommendedPlayers", outQuest.m_recommendedPlayers);
            ReadDouble(quest, "partyCreditRadius", outQuest.m_partyCreditRadius);
            ReadInt(quest, "rewardXp", outQuest.m_rewardXp);
            ReadInt(quest, "rewardCurrencyCopper", outQuest.m_rewardCurrencyTotalCopper);
            if (quest.HasMember("rewardCurrency") && quest["rewardCurrency"].IsObject())
            {
                const rapidjson::Value& rewardCurrency = quest["rewardCurrency"];
                ReadInt(rewardCurrency, "gold", outQuest.m_rewardCurrencyGold);
                ReadInt(rewardCurrency, "silver", outQuest.m_rewardCurrencySilver);
                ReadInt(rewardCurrency, "copper", outQuest.m_rewardCurrencyCopper);
            }
            if (quest.HasMember("objectiveArea") && quest["objectiveArea"].IsObject())
            {
                const rapidjson::Value& objectiveArea = quest["objectiveArea"];
                ReadString(objectiveArea, "areaId", outQuest.m_objectiveAreaId);
                ReadString(objectiveArea, "displayName", outQuest.m_objectiveAreaName);
                ReadString(objectiveArea, "kind", outQuest.m_objectiveAreaKind);
                ReadString(objectiveArea, "routeHintText", outQuest.m_routeHintText);
                ReadDouble(objectiveArea, "centerX", outQuest.m_objectiveX);
                ReadDouble(objectiveArea, "centerY", outQuest.m_objectiveY);
                ReadDouble(objectiveArea, "radius", outQuest.m_objectiveRadius);
            }
        }

        bool ParseWorldSessionJson(const AZStd::string& payload, WorldSessionResponse& outResponse, AZStd::string& outError)
        {
            outResponse = WorldSessionResponse{};

            rapidjson::Document document;
            document.Parse(payload.c_str());
            if (document.HasParseError() || !document.IsObject())
            {
                outError = "World session response was not valid JSON.";
                return false;
            }

            if (!ReadString(document, "worldSessionToken", outResponse.m_worldSessionToken) ||
                !ReadString(document, "characterId", outResponse.m_characterId) ||
                !ReadString(document, "realmId", outResponse.m_realmId) ||
                !ReadString(document, "zoneId", outResponse.m_zoneId) ||
                !ReadString(document, "displayName", outResponse.m_displayName))
            {
                outError = "World session response was missing required string fields.";
                return false;
            }

            if (!document.HasMember("position") || !document["position"].IsObject())
            {
                outError = "World session response was missing position data.";
                return false;
            }

            const rapidjson::Value& position = document["position"];
            if (!position.HasMember("x") || !position["x"].IsNumber() ||
                !position.HasMember("y") || !position["y"].IsNumber() ||
                !position.HasMember("z") || !position["z"].IsNumber())
            {
                outError = "World session response position was incomplete.";
                return false;
            }

            outResponse.m_position.m_x = position["x"].GetDouble();
            outResponse.m_position.m_y = position["y"].GetDouble();
            outResponse.m_position.m_z = position["z"].GetDouble();
            if (!document.HasMember("health") || !document["health"].IsNumber() ||
                !document.HasMember("maxHealth") || !document["maxHealth"].IsNumber() ||
                !document.HasMember("resource") || !document["resource"].IsNumber() ||
                !document.HasMember("maxResource") || !document["maxResource"].IsNumber() ||
                !ReadBool(document, "alive", outResponse.m_alive))
            {
                outError = "World session response was missing combat state.";
                return false;
            }

            outResponse.m_health = document["health"].GetDouble();
            outResponse.m_maxHealth = document["maxHealth"].GetDouble();
            outResponse.m_resource = document["resource"].GetDouble();
            outResponse.m_maxResource = document["maxResource"].GetDouble();
            ReadString(document, "resourceName", outResponse.m_resourceName);
            ReadInt(document, "level", outResponse.m_level);
            ReadInt(document, "experience", outResponse.m_experience);
            ReadInt(document, "currencyCopper", outResponse.m_currency.m_totalCopper);
            if (document.HasMember("currency") && document["currency"].IsObject())
            {
                const rapidjson::Value& currency = document["currency"];
                ReadInt(currency, "gold", outResponse.m_currency.m_gold);
                ReadInt(currency, "silver", outResponse.m_currency.m_silver);
                ReadInt(currency, "copper", outResponse.m_currency.m_copper);
            }
            if (document.HasMember("inventory") && document["inventory"].IsObject())
            {
                const rapidjson::Value& inventory = document["inventory"];
                ReadInt(inventory, "slotCount", outResponse.m_inventory.m_slotCount);
                outResponse.m_inventory.m_slots.clear();
                if (inventory.HasMember("slots") && inventory["slots"].IsArray())
                {
                    for (const rapidjson::Value& slotValue : inventory["slots"].GetArray())
                    {
                        if (!slotValue.IsObject())
                        {
                            continue;
                        }

                        InventorySlotState slot;
                        ReadInt(slotValue, "slotIndex", slot.m_slotIndex);
                        ReadString(slotValue, "itemId", slot.m_itemId);
                        ReadString(slotValue, "displayName", slot.m_displayName);
                        ReadInt(slotValue, "stackCount", slot.m_stackCount);
                        outResponse.m_inventory.m_slots.push_back(AZStd::move(slot));
                    }
                }
                if (outResponse.m_inventory.m_slotCount <= 0)
                {
                    outResponse.m_inventory.m_slotCount = static_cast<int>(outResponse.m_inventory.m_slots.size());
                }
            }
            if (document.HasMember("stats") && document["stats"].IsObject())
            {
                const rapidjson::Value& stats = document["stats"];
                ReadInt(stats, "strength", outResponse.m_stats.m_strength);
                ReadInt(stats, "stamina", outResponse.m_stats.m_stamina);
                ReadInt(stats, "armor", outResponse.m_stats.m_armor);
                ReadDouble(stats, "attackPower", outResponse.m_stats.m_attackPower);
                ReadDouble(stats, "armorReductionPct", outResponse.m_stats.m_armorReductionPct);
            }
            outResponse.m_talents = TalentState{};
            if (document.HasMember("talents") && document["talents"].IsObject())
            {
                const rapidjson::Value& talents = document["talents"];
                ReadBool(talents, "unlocked", outResponse.m_talents.m_unlocked);
                ReadInt(talents, "unlockLevel", outResponse.m_talents.m_unlockLevel);
                ReadInt(talents, "pointsGranted", outResponse.m_talents.m_pointsGranted);
                ReadInt(talents, "pointsSpent", outResponse.m_talents.m_pointsSpent);
                ReadInt(talents, "pointsAvailable", outResponse.m_talents.m_pointsAvailable);
                if (talents.HasMember("categories") && talents["categories"].IsArray())
                {
                    for (const rapidjson::Value& categoryValue : talents["categories"].GetArray())
                    {
                        if (categoryValue.IsString())
                        {
                            outResponse.m_talents.m_categories.push_back(categoryValue.GetString());
                        }
                    }
                }
                if (talents.HasMember("talents") && talents["talents"].IsArray())
                {
                    for (const rapidjson::Value& talentValue : talents["talents"].GetArray())
                    {
                        if (!talentValue.IsObject())
                        {
                            continue;
                        }

                        TalentEntryState talent;
                        ReadString(talentValue, "id", talent.m_id);
                        ReadString(talentValue, "displayName", talent.m_displayName);
                        ReadString(talentValue, "category", talent.m_category);
                        ReadString(talentValue, "description", talent.m_description);
                        ReadString(talentValue, "requirementText", talent.m_requirementText);
                        ReadInt(talentValue, "rank", talent.m_rank);
                        ReadInt(talentValue, "maxRank", talent.m_maxRank);
                        ReadInt(talentValue, "minLevel", talent.m_minLevel);
                        ReadBool(talentValue, "passive", talent.m_passive);
                        ReadBool(talentValue, "canSelect", talent.m_canSelect);
                        outResponse.m_talents.m_talents.push_back(AZStd::move(talent));
                    }
                }
            }
            outResponse.m_learnedAbilityIds.clear();
            if (document.HasMember("learnedAbilityIds") && document["learnedAbilityIds"].IsArray())
            {
                for (const rapidjson::Value& abilityValue : document["learnedAbilityIds"].GetArray())
                {
                    if (!abilityValue.IsString())
                    {
                        continue;
                    }
                    outResponse.m_learnedAbilityIds.push_back(abilityValue.GetString());
                }
            }
            outResponse.m_spellbookEntries.clear();
            if (document.HasMember("spellbook") && document["spellbook"].IsArray())
            {
                for (const rapidjson::Value& spellValue : document["spellbook"].GetArray())
                {
                    if (!spellValue.IsObject())
                    {
                        continue;
                    }

                    SpellbookEntryState entry;
                    ReadString(spellValue, "id", entry.m_id);
                    ReadString(spellValue, "displayName", entry.m_displayName);
                    ReadString(spellValue, "classId", entry.m_classId);
                    ReadString(spellValue, "description", entry.m_description);
                    ReadString(spellValue, "tooltipText", entry.m_tooltipText);
                    ReadString(spellValue, "requirementText", entry.m_requirementText);
                    ReadString(spellValue, "resourceName", entry.m_resourceName);
                    ReadInt(spellValue, "requiredLevel", entry.m_requiredLevel);
                    ReadDouble(spellValue, "resourceCost", entry.m_resourceCost);
                    ReadDouble(spellValue, "resourceGeneration", entry.m_resourceGeneration);
                    ReadInt64(spellValue, "cooldownMs", entry.m_cooldownMs);
                    ReadDouble(spellValue, "rangeMeters", entry.m_rangeMeters);
                    ReadBool(spellValue, "requiresTarget", entry.m_requiresTarget);
                    ReadBool(spellValue, "triggersGCD", entry.m_triggersGlobalCooldown);
                    ReadBool(spellValue, "learned", entry.m_learned);
                    outResponse.m_spellbookEntries.push_back(AZStd::move(entry));
                }
            }
            outResponse.m_actionBarSlots.clear();
            if (document.HasMember("actionBar") && document["actionBar"].IsArray())
            {
                for (const rapidjson::Value& actionValue : document["actionBar"].GetArray())
                {
                    if (!actionValue.IsObject())
                    {
                        continue;
                    }

                    ActionBarSlotState slot;
                    ReadInt(actionValue, "slotIndex", slot.m_slotIndex);
                    ReadString(actionValue, "hotkey", slot.m_hotkey);
                    ReadString(actionValue, "abilityId", slot.m_abilityId);
                    ReadString(actionValue, "displayName", slot.m_displayName);
                    ReadString(actionValue, "buttonLabel", slot.m_buttonLabel);
                    ReadString(actionValue, "resourceName", slot.m_resourceName);
                    ReadString(actionValue, "tooltipText", slot.m_tooltipText);
                    ReadDouble(actionValue, "resourceCost", slot.m_resourceCost);
                    ReadDouble(actionValue, "resourceGeneration", slot.m_resourceGeneration);
                    ReadInt64(actionValue, "cooldownMs", slot.m_cooldownMs);
                    ReadInt64(actionValue, "cooldownEndsAt", slot.m_cooldownEndsAt);
                    ReadInt64(actionValue, "cooldownRemainingMs", slot.m_cooldownRemainingMs);
                    ReadDouble(actionValue, "rangeMeters", slot.m_rangeMeters);
                    ReadBool(actionValue, "requiresTarget", slot.m_requiresTarget);
                    ReadBool(actionValue, "triggersGCD", slot.m_triggersGlobalCooldown);
                    ReadBool(actionValue, "learned", slot.m_learned);
                    outResponse.m_actionBarSlots.push_back(AZStd::move(slot));
                }
            }
            outResponse.m_trainer = TrainerState{};
            if (document.HasMember("trainer") && document["trainer"].IsObject())
            {
                const rapidjson::Value& trainer = document["trainer"];
                ReadString(trainer, "id", outResponse.m_trainer.m_id);
                ReadString(trainer, "displayName", outResponse.m_trainer.m_displayName);
                ReadString(trainer, "classId", outResponse.m_trainer.m_classId);
                ReadString(trainer, "interactionHint", outResponse.m_trainer.m_interactionHint);
                ReadBool(trainer, "inRange", outResponse.m_trainer.m_inRange);
                outResponse.m_trainer.m_offers.clear();
                if (trainer.HasMember("offers") && trainer["offers"].IsArray())
                {
                    for (const rapidjson::Value& offerValue : trainer["offers"].GetArray())
                    {
                        if (!offerValue.IsObject())
                        {
                            continue;
                        }

                        TrainerOfferState offer;
                        ReadString(offerValue, "abilityId", offer.m_abilityId);
                        ReadString(offerValue, "displayName", offer.m_displayName);
                        ReadString(offerValue, "description", offer.m_description);
                        ReadString(offerValue, "tooltipText", offer.m_tooltipText);
                        ReadString(offerValue, "requirementText", offer.m_requirementText);
                        ReadString(offerValue, "resourceName", offer.m_resourceName);
                        ReadInt(offerValue, "requiredLevel", offer.m_requiredLevel);
                        ReadInt(offerValue, "costCopper", offer.m_costCopper);
                        ReadDouble(offerValue, "resourceCost", offer.m_resourceCost);
                        ReadDouble(offerValue, "resourceGeneration", offer.m_resourceGeneration);
                        ReadInt64(offerValue, "cooldownMs", offer.m_cooldownMs);
                        ReadDouble(offerValue, "rangeMeters", offer.m_rangeMeters);
                        ReadBool(offerValue, "learned", offer.m_learned);
                        ReadBool(offerValue, "canLearn", offer.m_canLearn);
                        outResponse.m_trainer.m_offers.push_back(AZStd::move(offer));
                    }
                }
            }
            ReadString(document, "currentTargetId", outResponse.m_currentTargetId);
            ReadBool(document, "autoAttackActive", outResponse.m_autoAttackActive);
            ReadInt64(document, "globalCooldownEndsAt", outResponse.m_globalCooldownEndsAt);
            ReadInt64(document, "castEndsAt", outResponse.m_castEndsAt);
            ReadString(document, "castingAbilityId", outResponse.m_castingAbilityId);
            if (document.HasMember("quest") && document["quest"].IsObject())
            {
                ParseQuestState(document["quest"], outResponse.m_quest);
            }
            outResponse.m_quests.clear();
            if (document.HasMember("quests") && document["quests"].IsArray())
            {
                for (const rapidjson::Value& questValue : document["quests"].GetArray())
                {
                    if (!questValue.IsObject())
                    {
                        continue;
                    }

                    QuestState quest;
                    ParseQuestState(questValue, quest);
                    outResponse.m_quests.push_back(AZStd::move(quest));
                }
            }
            outResponse.m_trackedQuestIds.clear();
            if (document.HasMember("trackedQuestIds") && document["trackedQuestIds"].IsArray())
            {
                for (const rapidjson::Value& trackedQuestValue : document["trackedQuestIds"].GetArray())
                {
                    if (trackedQuestValue.IsString())
                    {
                        outResponse.m_trackedQuestIds.push_back(trackedQuestValue.GetString());
                    }
                }
            }
            outResponse.m_zoneMap = ZoneMapState{};
            if (document.HasMember("zoneMap") && document["zoneMap"].IsObject())
            {
                const rapidjson::Value& zoneMap = document["zoneMap"];
                ReadString(zoneMap, "zoneId", outResponse.m_zoneMap.m_zoneId);
                ReadString(zoneMap, "displayName", outResponse.m_zoneMap.m_displayName);
                if (zoneMap.HasMember("bounds") && zoneMap["bounds"].IsObject())
                {
                    const rapidjson::Value& bounds = zoneMap["bounds"];
                    ReadDouble(bounds, "minX", outResponse.m_zoneMap.m_minX);
                    ReadDouble(bounds, "minY", outResponse.m_zoneMap.m_minY);
                    ReadDouble(bounds, "maxX", outResponse.m_zoneMap.m_maxX);
                    ReadDouble(bounds, "maxY", outResponse.m_zoneMap.m_maxY);
                }
                if (zoneMap.HasMember("roads") && zoneMap["roads"].IsArray())
                {
                    for (const rapidjson::Value& roadValue : zoneMap["roads"].GetArray())
                    {
                        if (!roadValue.IsObject())
                        {
                            continue;
                        }
                        MapRoadState road;
                        ReadString(roadValue, "id", road.m_id);
                        ReadString(roadValue, "displayName", road.m_displayName);
                        if (roadValue.HasMember("points") && roadValue["points"].IsArray())
                        {
                            for (const rapidjson::Value& pointValue : roadValue["points"].GetArray())
                            {
                                if (!pointValue.IsObject())
                                {
                                    continue;
                                }
                                MapPointState point;
                                ReadDouble(pointValue, "x", point.m_x);
                                ReadDouble(pointValue, "y", point.m_y);
                                road.m_points.push_back(point);
                            }
                        }
                        outResponse.m_zoneMap.m_roads.push_back(AZStd::move(road));
                    }
                }
                if (zoneMap.HasMember("landmarks") && zoneMap["landmarks"].IsArray())
                {
                    for (const rapidjson::Value& landmarkValue : zoneMap["landmarks"].GetArray())
                    {
                        if (!landmarkValue.IsObject())
                        {
                            continue;
                        }
                        MapLandmarkState landmark;
                        ReadString(landmarkValue, "id", landmark.m_id);
                        ReadString(landmarkValue, "displayName", landmark.m_displayName);
                        ReadString(landmarkValue, "kind", landmark.m_kind);
                        ReadDouble(landmarkValue, "x", landmark.m_x);
                        ReadDouble(landmarkValue, "y", landmark.m_y);
                        outResponse.m_zoneMap.m_landmarks.push_back(AZStd::move(landmark));
                    }
                }
            }
            outResponse.m_navigationAreas.clear();
            if (document.HasMember("navigationAreas") && document["navigationAreas"].IsArray())
            {
                for (const rapidjson::Value& areaValue : document["navigationAreas"].GetArray())
                {
                    if (!areaValue.IsObject())
                    {
                        continue;
                    }
                    NavigationAreaState area;
                    ReadString(areaValue, "areaId", area.m_areaId);
                    ReadString(areaValue, "displayName", area.m_displayName);
                    ReadString(areaValue, "kind", area.m_kind);
                    ReadString(areaValue, "routeHintText", area.m_routeHintText);
                    ReadString(areaValue, "targetMobType", area.m_targetMobType);
                    ReadString(areaValue, "targetEntityId", area.m_targetEntityId);
                    ReadDouble(areaValue, "centerX", area.m_centerX);
                    ReadDouble(areaValue, "centerY", area.m_centerY);
                    ReadDouble(areaValue, "radius", area.m_radius);
                    if (areaValue.HasMember("questIds") && areaValue["questIds"].IsArray())
                    {
                        for (const rapidjson::Value& questIdValue : areaValue["questIds"].GetArray())
                        {
                            if (questIdValue.IsString())
                            {
                                area.m_questIds.push_back(questIdValue.GetString());
                            }
                        }
                    }
                    outResponse.m_navigationAreas.push_back(AZStd::move(area));
                }
            }
            outResponse.m_mapMarkers.clear();
            if (document.HasMember("mapMarkers") && document["mapMarkers"].IsArray())
            {
                for (const rapidjson::Value& markerValue : document["mapMarkers"].GetArray())
                {
                    if (!markerValue.IsObject())
                    {
                        continue;
                    }
                    MapMarkerState marker;
                    ReadString(markerValue, "id", marker.m_id);
                    ReadString(markerValue, "displayName", marker.m_displayName);
                    ReadString(markerValue, "kind", marker.m_kind);
                    ReadString(markerValue, "questId", marker.m_questId);
                    ReadString(markerValue, "entityId", marker.m_entityId);
                    ReadString(markerValue, "areaId", marker.m_areaId);
                    ReadString(markerValue, "routeHintText", marker.m_routeHintText);
                    ReadDouble(markerValue, "x", marker.m_x);
                    ReadDouble(markerValue, "y", marker.m_y);
                    ReadDouble(markerValue, "radius", marker.m_radius);
                    outResponse.m_mapMarkers.push_back(AZStd::move(marker));
                }
            }
            outResponse.m_entities.clear();

            if (document.HasMember("entities") && document["entities"].IsArray())
            {
                for (const rapidjson::Value& entityValue : document["entities"].GetArray())
                {
                    if (!entityValue.IsObject())
                    {
                        continue;
                    }

                    VisibleEntity entity;
                    ReadString(entityValue, "id", entity.m_id);
                    ReadString(entityValue, "displayName", entity.m_displayName);
                    ReadString(entityValue, "kind", entity.m_kind);
                    ReadString(entityValue, "classification", entity.m_classification);
                    ReadBool(entityValue, "elite", entity.m_elite);
                    if (entityValue.HasMember("x") && entityValue["x"].IsNumber())
                    {
                        entity.m_x = entityValue["x"].GetDouble();
                    }
                    if (entityValue.HasMember("y") && entityValue["y"].IsNumber())
                    {
                        entity.m_y = entityValue["y"].GetDouble();
                    }
                    if (entityValue.HasMember("z") && entityValue["z"].IsNumber())
                    {
                        entity.m_z = entityValue["z"].GetDouble();
                    }
                    if (entityValue.HasMember("health") && entityValue["health"].IsNumber())
                    {
                        entity.m_health = entityValue["health"].GetDouble();
                    }
                    if (entityValue.HasMember("maxHealth") && entityValue["maxHealth"].IsNumber())
                    {
                        entity.m_maxHealth = entityValue["maxHealth"].GetDouble();
                    }
                    if (entityValue.HasMember("alive") && entityValue["alive"].IsBool())
                    {
                        entity.m_alive = entityValue["alive"].GetBool();
                    }
                    if (entityValue.HasMember("targetable") && entityValue["targetable"].IsBool())
                    {
                        entity.m_targetable = entityValue["targetable"].GetBool();
                    }
                    ReadString(entityValue, "aiState", entity.m_aiState);
                    entity.m_services.clear();
                    if (entityValue.HasMember("npcServices") && entityValue["npcServices"].IsArray())
                    {
                        for (const rapidjson::Value& serviceValue : entityValue["npcServices"].GetArray())
                        {
                            if (!serviceValue.IsObject())
                            {
                                continue;
                            }

                            NpcServiceState service;
                            ReadString(serviceValue, "type", service.m_type);
                            ReadString(serviceValue, "serviceId", service.m_serviceId);
                            ReadString(serviceValue, "label", service.m_label);
                            entity.m_services.push_back(AZStd::move(service));
                        }
                    }

                    outResponse.m_entities.push_back(AZStd::move(entity));
                }
            }

            return true;
        }

        bool ParseBootstrapJson(const AZStd::string& payload, WorldBootstrapResponse& outResponse, AZStd::string& outError)
        {
            rapidjson::Document document;
            document.Parse(payload.c_str());
            if (document.HasParseError() || !document.IsObject())
            {
                outError = "Bootstrap response was not valid JSON.";
                return false;
            }

            if (!ReadString(document, "zoneId", outResponse.m_zoneId) ||
                !ReadString(document, "cellId", outResponse.m_cellId) ||
                !ReadString(document, "motd", outResponse.m_motd) ||
                !ReadString(document, "revision", outResponse.m_revision))
            {
                outError = "Bootstrap response was missing required fields.";
                return false;
            }

            return true;
        }

        bool ParseSocialStateJson(const AZStd::string& payload, SocialStateResponse& outResponse, AZStd::string& outError)
        {
            outResponse = SocialStateResponse{};

            rapidjson::Document document;
            document.Parse(payload.c_str());
            if (document.HasParseError() || !document.IsObject())
            {
                outError = "Social state response was not valid JSON.";
                return false;
            }

            if (document.HasMember("chatMessages") && document["chatMessages"].IsArray())
            {
                for (const rapidjson::Value& messageValue : document["chatMessages"].GetArray())
                {
                    if (!messageValue.IsObject())
                    {
                        continue;
                    }

                    ChatMessageState message;
                    ReadString(messageValue, "messageId", message.m_messageId);
                    ReadString(messageValue, "channel", message.m_channel);
                    ReadString(messageValue, "senderCharacterId", message.m_senderCharacterId);
                    ReadString(messageValue, "senderDisplayName", message.m_senderDisplayName);
                    ReadString(messageValue, "targetCharacterId", message.m_targetCharacterId);
                    ReadString(messageValue, "partyId", message.m_partyId);
                    ReadString(messageValue, "guildId", message.m_guildId);
                    ReadString(messageValue, "zoneId", message.m_zoneId);
                    ReadString(messageValue, "messageText", message.m_messageText);
                    ReadInt64(messageValue, "timestamp", message.m_timestamp);
                    outResponse.m_chatMessages.push_back(AZStd::move(message));
                }
            }

            if (document.HasMember("friends") && document["friends"].IsArray())
            {
                for (const rapidjson::Value& friendValue : document["friends"].GetArray())
                {
                    if (!friendValue.IsObject())
                    {
                        continue;
                    }

                    FriendState friendState;
                    ReadString(friendValue, "characterId", friendState.m_characterId);
                    ReadString(friendValue, "displayName", friendState.m_displayName);
                    ReadInt(friendValue, "level", friendState.m_level);
                    ReadString(friendValue, "classId", friendState.m_classId);
                    ReadString(friendValue, "zoneId", friendState.m_zoneId);
                    ReadBool(friendValue, "online", friendState.m_online);
                    outResponse.m_friends.push_back(AZStd::move(friendState));
                }
            }

            if (document.HasMember("party") && document["party"].IsObject())
            {
                outResponse.m_hasParty = true;
                const rapidjson::Value& partyValue = document["party"];
                ReadString(partyValue, "partyId", outResponse.m_party.m_partyId);
                ReadString(partyValue, "leaderCharacterId", outResponse.m_party.m_leaderCharacterId);
                if (partyValue.HasMember("members") && partyValue["members"].IsArray())
                {
                    for (const rapidjson::Value& memberValue : partyValue["members"].GetArray())
                    {
                        if (!memberValue.IsObject())
                        {
                            continue;
                        }

                        PartyMemberState member;
                        ReadString(memberValue, "characterId", member.m_characterId);
                        ReadString(memberValue, "displayName", member.m_displayName);
                        ReadInt(memberValue, "level", member.m_level);
                        ReadString(memberValue, "classId", member.m_classId);
                        ReadString(memberValue, "zoneId", member.m_zoneId);
                        ReadBool(memberValue, "online", member.m_online);
                        ReadBool(memberValue, "leader", member.m_leader);
                        ReadDouble(memberValue, "health", member.m_health);
                        ReadDouble(memberValue, "maxHealth", member.m_maxHealth);
                        ReadDouble(memberValue, "resource", member.m_resource);
                        ReadDouble(memberValue, "maxResource", member.m_maxResource);
                        ReadBool(memberValue, "disconnected", member.m_disconnected);
                        outResponse.m_party.m_members.push_back(AZStd::move(member));
                    }
                }
            }

            if (document.HasMember("partyInvites") && document["partyInvites"].IsArray())
            {
                for (const rapidjson::Value& inviteValue : document["partyInvites"].GetArray())
                {
                    if (!inviteValue.IsObject())
                    {
                        continue;
                    }

                    PartyInviteState invite;
                    ReadString(inviteValue, "inviteId", invite.m_inviteId);
                    ReadString(inviteValue, "partyId", invite.m_partyId);
                    ReadString(inviteValue, "inviterCharacterId", invite.m_inviterCharacterId);
                    ReadString(inviteValue, "inviterDisplayName", invite.m_inviterDisplayName);
                    ReadInt64(inviteValue, "expiresAt", invite.m_expiresAt);
                    outResponse.m_partyInvites.push_back(AZStd::move(invite));
                }
            }

            if (document.HasMember("guild") && document["guild"].IsObject())
            {
                outResponse.m_hasGuild = true;
                const rapidjson::Value& guildValue = document["guild"];
                ReadString(guildValue, "guildId", outResponse.m_guild.m_guildId);
                ReadString(guildValue, "guildName", outResponse.m_guild.m_guildName);
                ReadString(guildValue, "leaderCharacterId", outResponse.m_guild.m_leaderCharacterId);
                ReadString(guildValue, "messageOfTheDay", outResponse.m_guild.m_messageOfTheDay);
                ReadString(guildValue, "currentRankId", outResponse.m_guild.m_currentRankId);
                ReadInt64(guildValue, "createdAt", outResponse.m_guild.m_createdAt);
                ReadString(guildValue, "createdByCharacterId", outResponse.m_guild.m_createdByCharacterId);
                if (guildValue.HasMember("currentPermissions") && guildValue["currentPermissions"].IsArray())
                {
                    for (const rapidjson::Value& permissionValue : guildValue["currentPermissions"].GetArray())
                    {
                        if (permissionValue.IsString())
                        {
                            outResponse.m_guild.m_currentPermissions.push_back(permissionValue.GetString());
                        }
                    }
                }
                if (guildValue.HasMember("ranks") && guildValue["ranks"].IsArray())
                {
                    for (const rapidjson::Value& rankValue : guildValue["ranks"].GetArray())
                    {
                        if (!rankValue.IsObject())
                        {
                            continue;
                        }
                        GuildRankState rank;
                        ReadString(rankValue, "rankId", rank.m_rankId);
                        ReadString(rankValue, "displayName", rank.m_displayName);
                        ReadInt(rankValue, "priority", rank.m_priority);
                        if (rankValue.HasMember("permissions") && rankValue["permissions"].IsArray())
                        {
                            for (const rapidjson::Value& permissionValue : rankValue["permissions"].GetArray())
                            {
                                if (permissionValue.IsString())
                                {
                                    rank.m_permissions.push_back(permissionValue.GetString());
                                }
                            }
                        }
                        outResponse.m_guild.m_ranks.push_back(AZStd::move(rank));
                    }
                }
                if (guildValue.HasMember("members") && guildValue["members"].IsArray())
                {
                    for (const rapidjson::Value& memberValue : guildValue["members"].GetArray())
                    {
                        if (!memberValue.IsObject())
                        {
                            continue;
                        }
                        GuildMemberState member;
                        ReadString(memberValue, "characterId", member.m_characterId);
                        ReadString(memberValue, "displayName", member.m_displayName);
                        ReadString(memberValue, "raceId", member.m_raceId);
                        ReadString(memberValue, "classId", member.m_classId);
                        ReadInt(memberValue, "level", member.m_level);
                        ReadString(memberValue, "rankId", member.m_rankId);
                        ReadString(memberValue, "rankName", member.m_rankName);
                        ReadInt64(memberValue, "joinedAt", member.m_joinedAt);
                        ReadInt64(memberValue, "lastOnlineAt", member.m_lastOnlineAt);
                        ReadBool(memberValue, "online", member.m_online);
                        ReadString(memberValue, "currentZoneId", member.m_currentZoneId);
                        outResponse.m_guild.m_members.push_back(AZStd::move(member));
                    }
                }
            }

            if (document.HasMember("guildInvites") && document["guildInvites"].IsArray())
            {
                for (const rapidjson::Value& inviteValue : document["guildInvites"].GetArray())
                {
                    if (!inviteValue.IsObject())
                    {
                        continue;
                    }

                    GuildInviteState invite;
                    ReadString(inviteValue, "inviteId", invite.m_inviteId);
                    ReadString(inviteValue, "guildId", invite.m_guildId);
                    ReadString(inviteValue, "guildName", invite.m_guildName);
                    ReadString(inviteValue, "inviterCharacterId", invite.m_inviterCharacterId);
                    ReadString(inviteValue, "inviterDisplayName", invite.m_inviterDisplayName);
                    ReadInt64(inviteValue, "expiresAt", invite.m_expiresAt);
                    outResponse.m_guildInvites.push_back(AZStd::move(invite));
                }
            }

            return true;
        }

        AZStd::string ExtractErrorMessage(const AZStd::string& payload)
        {
            if (payload.empty())
            {
                return "The server did not return an error message.";
            }

            rapidjson::Document document;
            document.Parse(payload.c_str());
            if (!document.HasParseError() && document.IsObject())
            {
                AZStd::string message;
                if (ReadString(document, "message", message) && !message.empty())
                {
                    return message;
                }
            }

            return payload;
        }
    } // namespace

    bool ConnectRequest(
        const AZStd::string& worldEndpoint,
        const AZStd::string& ticketId,
        WorldSessionResponse& outResponse,
        AZStd::string& outError)
    {
        AZStd::string responseBody;
        AZ::u32 statusCode = 0;
        const AZStd::string requestBody = AZStd::string::format("{\"ticketId\":\"%s\"}", ticketId.c_str());
        if (!PerformRequest(worldEndpoint, L"POST", L"/v1/world/connect", requestBody, responseBody, statusCode, outError))
        {
            return false;
        }

        if (statusCode < 200 || statusCode >= 300)
        {
            outError = ExtractErrorMessage(responseBody);
            return false;
        }

        return ParseWorldSessionJson(responseBody, outResponse, outError);
    }

    bool BootstrapRequest(
        const AZStd::string& worldEndpoint,
        WorldBootstrapResponse& outResponse,
        AZStd::string& outError)
    {
        AZStd::string responseBody;
        AZ::u32 statusCode = 0;
        if (!PerformRequest(worldEndpoint, L"GET", L"/v1/world/bootstrap", {}, responseBody, statusCode, outError))
        {
            return false;
        }

        if (statusCode < 200 || statusCode >= 300)
        {
            outError = ExtractErrorMessage(responseBody);
            return false;
        }

        return ParseBootstrapJson(responseBody, outResponse, outError);
    }

    bool MoveRequest(
        const AZStd::string& worldEndpoint,
        const AZStd::string& worldSessionToken,
        double deltaX,
        double deltaY,
        WorldSessionResponse& outResponse,
        AZStd::string& outError)
    {
        AZStd::string responseBody;
        AZ::u32 statusCode = 0;
        const AZStd::string requestBody = AZStd::string::format(
            "{\"worldSessionToken\":\"%s\",\"deltaX\":%.6f,\"deltaY\":%.6f}",
            worldSessionToken.c_str(),
            deltaX,
            deltaY);

        if (!PerformRequest(worldEndpoint, L"POST", L"/v1/world/move", requestBody, responseBody, statusCode, outError))
        {
            return false;
        }

        if (statusCode < 200 || statusCode >= 300)
        {
            outError = ExtractErrorMessage(responseBody);
            return false;
        }

        return ParseWorldSessionJson(responseBody, outResponse, outError);
    }

    bool DisconnectRequest(
        const AZStd::string& worldEndpoint,
        const AZStd::string& worldSessionToken,
        AZStd::string& outError)
    {
        AZStd::string responseBody;
        AZ::u32 statusCode = 0;
        const AZStd::string requestBody = AZStd::string::format(
            "{\"worldSessionToken\":\"%s\"}",
            worldSessionToken.c_str());

        if (!PerformRequest(worldEndpoint, L"POST", L"/v1/world/disconnect", requestBody, responseBody, statusCode, outError))
        {
            return false;
        }

        if (statusCode < 200 || statusCode >= 300)
        {
            outError = ExtractErrorMessage(responseBody);
            return false;
        }

        return true;
    }

    bool StateRequest(
        const AZStd::string& worldEndpoint,
        const AZStd::string& worldSessionToken,
        WorldSessionResponse& outResponse,
        AZStd::string& outError)
    {
        AZStd::string responseBody;
        AZ::u32 statusCode = 0;
        const AZStd::wstring path = ToWideString(AZStd::string::format("/v1/world/state?worldSessionToken=%s", worldSessionToken.c_str()));
        if (!PerformRequest(worldEndpoint, L"GET", path, {}, responseBody, statusCode, outError))
        {
            return false;
        }

        if (statusCode < 200 || statusCode >= 300)
        {
            outError = ExtractErrorMessage(responseBody);
            return false;
        }

        return ParseWorldSessionJson(responseBody, outResponse, outError);
    }

    bool SocialStateRequest(
        const AZStd::string& worldEndpoint,
        const AZStd::string& worldSessionToken,
        const AZStd::string& afterMessageId,
        SocialStateResponse& outResponse,
        AZStd::string& outError)
    {
        AZStd::string responseBody;
        AZ::u32 statusCode = 0;
        AZStd::string path = AZStd::string::format("/v1/world/social/state?worldSessionToken=%s", worldSessionToken.c_str());
        if (!afterMessageId.empty())
        {
            path += AZStd::string::format("&afterMessageId=%s", afterMessageId.c_str());
        }

        if (!PerformRequest(worldEndpoint, L"GET", ToWideString(path), {}, responseBody, statusCode, outError))
        {
            return false;
        }

        if (statusCode < 200 || statusCode >= 300)
        {
            outError = ExtractErrorMessage(responseBody);
            return false;
        }

        return ParseSocialStateJson(responseBody, outResponse, outError);
    }

    bool SocialPostRequest(
        const AZStd::string& worldEndpoint,
        const wchar_t* path,
        const AZStd::string& requestBody,
        SocialStateResponse& outResponse,
        AZStd::string& outError)
    {
        AZStd::string responseBody;
        AZ::u32 statusCode = 0;
        if (!PerformRequest(worldEndpoint, L"POST", path, requestBody, responseBody, statusCode, outError))
        {
            return false;
        }

        if (statusCode < 200 || statusCode >= 300)
        {
            outError = ExtractErrorMessage(responseBody);
            return false;
        }

        return ParseSocialStateJson(responseBody, outResponse, outError);
    }

    bool SendChatRequest(
        const AZStd::string& worldEndpoint,
        const AZStd::string& worldSessionToken,
        const AZStd::string& channel,
        const AZStd::string& targetName,
        const AZStd::string& messageText,
        SocialStateResponse& outResponse,
        AZStd::string& outError)
    {
        const AZStd::string requestBody = AZStd::string::format(
            "{\"worldSessionToken\":\"%s\",\"channel\":\"%s\",\"targetName\":\"%s\",\"messageText\":\"%s\"}",
            JsonEscape(worldSessionToken).c_str(),
            JsonEscape(channel).c_str(),
            JsonEscape(targetName).c_str(),
            JsonEscape(messageText).c_str());
        return SocialPostRequest(worldEndpoint, L"/v1/world/chat/send", requestBody, outResponse, outError);
    }

    bool FriendRequest(
        const AZStd::string& worldEndpoint,
        const wchar_t* path,
        const AZStd::string& worldSessionToken,
        const AZStd::string& name,
        SocialStateResponse& outResponse,
        AZStd::string& outError)
    {
        const AZStd::string requestBody = AZStd::string::format(
            "{\"worldSessionToken\":\"%s\",\"name\":\"%s\"}",
            JsonEscape(worldSessionToken).c_str(),
            JsonEscape(name).c_str());
        return SocialPostRequest(worldEndpoint, path, requestBody, outResponse, outError);
    }

    bool InvitePartyRequest(
        const AZStd::string& worldEndpoint,
        const AZStd::string& worldSessionToken,
        const AZStd::string& targetName,
        const AZStd::string& targetCharacterId,
        SocialStateResponse& outResponse,
        AZStd::string& outError)
    {
        const AZStd::string requestBody = AZStd::string::format(
            "{\"worldSessionToken\":\"%s\",\"targetName\":\"%s\",\"targetCharacterId\":\"%s\"}",
            JsonEscape(worldSessionToken).c_str(),
            JsonEscape(targetName).c_str(),
            JsonEscape(targetCharacterId).c_str());
        return SocialPostRequest(worldEndpoint, L"/v1/world/party/invite", requestBody, outResponse, outError);
    }

    bool PartyInviteActionRequest(
        const AZStd::string& worldEndpoint,
        const wchar_t* path,
        const AZStd::string& worldSessionToken,
        const AZStd::string& inviteId,
        SocialStateResponse& outResponse,
        AZStd::string& outError)
    {
        const AZStd::string requestBody = AZStd::string::format(
            "{\"worldSessionToken\":\"%s\",\"inviteId\":\"%s\"}",
            JsonEscape(worldSessionToken).c_str(),
            JsonEscape(inviteId).c_str());
        return SocialPostRequest(worldEndpoint, path, requestBody, outResponse, outError);
    }

    bool PartyActionRequest(
        const AZStd::string& worldEndpoint,
        const wchar_t* path,
        const AZStd::string& worldSessionToken,
        SocialStateResponse& outResponse,
        AZStd::string& outError)
    {
        const AZStd::string requestBody = AZStd::string::format(
            "{\"worldSessionToken\":\"%s\"}",
            JsonEscape(worldSessionToken).c_str());
        return SocialPostRequest(worldEndpoint, path, requestBody, outResponse, outError);
    }

    bool GuildCreateRequest(
        const AZStd::string& worldEndpoint,
        const AZStd::string& worldSessionToken,
        const AZStd::string& guildName,
        SocialStateResponse& outResponse,
        AZStd::string& outError)
    {
        const AZStd::string requestBody = AZStd::string::format(
            "{\"worldSessionToken\":\"%s\",\"guildName\":\"%s\"}",
            JsonEscape(worldSessionToken).c_str(),
            JsonEscape(guildName).c_str());
        return SocialPostRequest(worldEndpoint, L"/v1/world/guild/create", requestBody, outResponse, outError);
    }

    bool GuildNameActionRequest(
        const AZStd::string& worldEndpoint,
        const wchar_t* path,
        const AZStd::string& worldSessionToken,
        const AZStd::string& targetName,
        SocialStateResponse& outResponse,
        AZStd::string& outError)
    {
        const AZStd::string requestBody = AZStd::string::format(
            "{\"worldSessionToken\":\"%s\",\"targetName\":\"%s\"}",
            JsonEscape(worldSessionToken).c_str(),
            JsonEscape(targetName).c_str());
        return SocialPostRequest(worldEndpoint, path, requestBody, outResponse, outError);
    }

    bool GuildInviteActionRequest(
        const AZStd::string& worldEndpoint,
        const wchar_t* path,
        const AZStd::string& worldSessionToken,
        const AZStd::string& inviteId,
        SocialStateResponse& outResponse,
        AZStd::string& outError)
    {
        const AZStd::string requestBody = AZStd::string::format(
            "{\"worldSessionToken\":\"%s\",\"inviteId\":\"%s\"}",
            JsonEscape(worldSessionToken).c_str(),
            JsonEscape(inviteId).c_str());
        return SocialPostRequest(worldEndpoint, path, requestBody, outResponse, outError);
    }

    bool GuildMOTDRequest(
        const AZStd::string& worldEndpoint,
        const AZStd::string& worldSessionToken,
        const AZStd::string& messageOfTheDay,
        SocialStateResponse& outResponse,
        AZStd::string& outError)
    {
        const AZStd::string requestBody = AZStd::string::format(
            "{\"worldSessionToken\":\"%s\",\"messageOfTheDay\":\"%s\"}",
            JsonEscape(worldSessionToken).c_str(),
            JsonEscape(messageOfTheDay).c_str());
        return SocialPostRequest(worldEndpoint, L"/v1/world/guild/motd", requestBody, outResponse, outError);
    }

    bool SetTargetRequest(
        const AZStd::string& worldEndpoint,
        const AZStd::string& worldSessionToken,
        const AZStd::string& targetId,
        WorldSessionResponse& outResponse,
        AZStd::string& outError)
    {
        AZStd::string responseBody;
        AZ::u32 statusCode = 0;
        const AZStd::string requestBody = AZStd::string::format(
            "{\"worldSessionToken\":\"%s\",\"targetId\":\"%s\"}",
            worldSessionToken.c_str(),
            targetId.c_str());
        if (!PerformRequest(worldEndpoint, L"POST", L"/v1/world/target", requestBody, responseBody, statusCode, outError))
        {
            return false;
        }

        if (statusCode < 200 || statusCode >= 300)
        {
            outError = ExtractErrorMessage(responseBody);
            return false;
        }

        return ParseWorldSessionJson(responseBody, outResponse, outError);
    }

    bool AcceptQuestRequest(
        const AZStd::string& worldEndpoint,
        const AZStd::string& worldSessionToken,
        const AZStd::string& questId,
        WorldSessionResponse& outResponse,
        AZStd::string& outError)
    {
        AZStd::string responseBody;
        AZ::u32 statusCode = 0;
        const AZStd::string requestBody = AZStd::string::format(
            "{\"worldSessionToken\":\"%s\",\"questId\":\"%s\"}",
            worldSessionToken.c_str(),
            questId.c_str());
        if (!PerformRequest(worldEndpoint, L"POST", L"/v1/world/quest/accept", requestBody, responseBody, statusCode, outError))
        {
            return false;
        }

        if (statusCode < 200 || statusCode >= 300)
        {
            outError = ExtractErrorMessage(responseBody);
            return false;
        }

        return ParseWorldSessionJson(responseBody, outResponse, outError);
    }

    bool TrackQuestRequest(
        const AZStd::string& worldEndpoint,
        const AZStd::string& worldSessionToken,
        const AZStd::string& questId,
        bool tracked,
        WorldSessionResponse& outResponse,
        AZStd::string& outError)
    {
        AZStd::string responseBody;
        AZ::u32 statusCode = 0;
        const AZStd::string requestBody = AZStd::string::format(
            "{\"worldSessionToken\":\"%s\",\"questId\":\"%s\",\"tracked\":%s}",
            worldSessionToken.c_str(),
            questId.c_str(),
            tracked ? "true" : "false");
        if (!PerformRequest(worldEndpoint, L"POST", L"/v1/world/quest/track", requestBody, responseBody, statusCode, outError))
        {
            return false;
        }

        if (statusCode < 200 || statusCode >= 300)
        {
            outError = ExtractErrorMessage(responseBody);
            return false;
        }

        return ParseWorldSessionJson(responseBody, outResponse, outError);
    }

    bool SetAutoAttackRequest(
        const AZStd::string& worldEndpoint,
        const AZStd::string& worldSessionToken,
        bool enabled,
        WorldSessionResponse& outResponse,
        AZStd::string& outError)
    {
        AZStd::string responseBody;
        AZ::u32 statusCode = 0;
        const AZStd::string requestBody = AZStd::string::format(
            "{\"worldSessionToken\":\"%s\",\"enabled\":%s}",
            worldSessionToken.c_str(),
            enabled ? "true" : "false");
        if (!PerformRequest(worldEndpoint, L"POST", L"/v1/world/attack/auto", requestBody, responseBody, statusCode, outError))
        {
            return false;
        }

        if (statusCode < 200 || statusCode >= 300)
        {
            outError = ExtractErrorMessage(responseBody);
            return false;
        }

        return ParseWorldSessionJson(responseBody, outResponse, outError);
    }

    bool ActivateAbilityRequest(
        const AZStd::string& worldEndpoint,
        const AZStd::string& worldSessionToken,
        const AZStd::string& abilityId,
        WorldSessionResponse& outResponse,
        AZStd::string& outError)
    {
        AZStd::string responseBody;
        AZ::u32 statusCode = 0;
        const AZStd::string requestBody = AZStd::string::format(
            "{\"worldSessionToken\":\"%s\",\"abilityId\":\"%s\"}",
            worldSessionToken.c_str(),
            abilityId.c_str());
        if (!PerformRequest(worldEndpoint, L"POST", L"/v1/world/attack/ability", requestBody, responseBody, statusCode, outError))
        {
            return false;
        }

        if (statusCode < 200 || statusCode >= 300)
        {
            outError = ExtractErrorMessage(responseBody);
            return false;
        }

        return ParseWorldSessionJson(responseBody, outResponse, outError);
    }

    bool LearnTrainerAbilityRequest(
        const AZStd::string& worldEndpoint,
        const AZStd::string& worldSessionToken,
        const AZStd::string& trainerId,
        const AZStd::string& abilityId,
        WorldSessionResponse& outResponse,
        AZStd::string& outError)
    {
        AZStd::string responseBody;
        AZ::u32 statusCode = 0;
        const AZStd::string requestBody = AZStd::string::format(
            "{\"worldSessionToken\":\"%s\",\"trainerId\":\"%s\",\"abilityId\":\"%s\"}",
            worldSessionToken.c_str(),
            trainerId.c_str(),
            abilityId.c_str());
        if (!PerformRequest(worldEndpoint, L"POST", L"/v1/world/trainer/learn", requestBody, responseBody, statusCode, outError))
        {
            return false;
        }

        if (statusCode < 200 || statusCode >= 300)
        {
            outError = ExtractErrorMessage(responseBody);
            return false;
        }

        return ParseWorldSessionJson(responseBody, outResponse, outError);
    }

    bool SelectTalentRequest(
        const AZStd::string& worldEndpoint,
        const AZStd::string& worldSessionToken,
        const AZStd::string& talentId,
        WorldSessionResponse& outResponse,
        AZStd::string& outError)
    {
        AZStd::string responseBody;
        AZ::u32 statusCode = 0;
        const AZStd::string requestBody = AZStd::string::format(
            "{\"worldSessionToken\":\"%s\",\"talentId\":\"%s\"}",
            worldSessionToken.c_str(),
            talentId.c_str());
        if (!PerformRequest(worldEndpoint, L"POST", L"/v1/world/talent/select", requestBody, responseBody, statusCode, outError))
        {
            return false;
        }

        if (statusCode < 200 || statusCode >= 300)
        {
            outError = ExtractErrorMessage(responseBody);
            return false;
        }

        return ParseWorldSessionJson(responseBody, outResponse, outError);
    }

    bool AssignActionBarSlotRequest(
        const AZStd::string& worldEndpoint,
        const AZStd::string& worldSessionToken,
        int slotIndex,
        const AZStd::string& abilityId,
        WorldSessionResponse& outResponse,
        AZStd::string& outError)
    {
        AZStd::string responseBody;
        AZ::u32 statusCode = 0;
        const AZStd::string requestBody = AZStd::string::format(
            "{\"worldSessionToken\":\"%s\",\"slotIndex\":%d,\"abilityId\":\"%s\"}",
            worldSessionToken.c_str(),
            slotIndex,
            abilityId.c_str());
        if (!PerformRequest(worldEndpoint, L"POST", L"/v1/world/action-bar/assign", requestBody, responseBody, statusCode, outError))
        {
            return false;
        }

        if (statusCode < 200 || statusCode >= 300)
        {
            outError = ExtractErrorMessage(responseBody);
            return false;
        }

        return ParseWorldSessionJson(responseBody, outResponse, outError);
    }

    bool ClearActionBarSlotRequest(
        const AZStd::string& worldEndpoint,
        const AZStd::string& worldSessionToken,
        int slotIndex,
        WorldSessionResponse& outResponse,
        AZStd::string& outError)
    {
        AZStd::string responseBody;
        AZ::u32 statusCode = 0;
        const AZStd::string requestBody = AZStd::string::format(
            "{\"worldSessionToken\":\"%s\",\"slotIndex\":%d}",
            worldSessionToken.c_str(),
            slotIndex);
        if (!PerformRequest(worldEndpoint, L"POST", L"/v1/world/action-bar/clear", requestBody, responseBody, statusCode, outError))
        {
            return false;
        }

        if (statusCode < 200 || statusCode >= 300)
        {
            outError = ExtractErrorMessage(responseBody);
            return false;
        }

        return ParseWorldSessionJson(responseBody, outResponse, outError);
    }

    bool ReconnectRequest(
        const AZStd::string& worldEndpoint,
        const AZStd::string& worldSessionToken,
        WorldSessionResponse& outResponse,
        AZStd::string& outError)
    {
        AZStd::string responseBody;
        AZ::u32 statusCode = 0;
        const AZStd::string requestBody = AZStd::string::format(
            "{\"worldSessionToken\":\"%s\"}",
            worldSessionToken.c_str());

        if (!PerformRequest(worldEndpoint, L"POST", L"/v1/world/reconnect", requestBody, responseBody, statusCode, outError))
        {
            return false;
        }

        if (statusCode < 200 || statusCode >= 300)
        {
            outError = ExtractErrorMessage(responseBody);
            return false;
        }

        return ParseWorldSessionJson(responseBody, outResponse, outError);
    }

    bool MoveActionBarSlotRequest(
        const AZStd::string& worldEndpoint,
        const AZStd::string& worldSessionToken,
        int fromSlotIndex,
        int toSlotIndex,
        WorldSessionResponse& outResponse,
        AZStd::string& outError)
    {
        AZStd::string responseBody;
        AZ::u32 statusCode = 0;
        const AZStd::string requestBody = AZStd::string::format(
            "{\"worldSessionToken\":\"%s\",\"fromSlotIndex\":%d,\"toSlotIndex\":%d}",
            worldSessionToken.c_str(),
            fromSlotIndex,
            toSlotIndex);
        if (!PerformRequest(worldEndpoint, L"POST", L"/v1/world/action-bar/move", requestBody, responseBody, statusCode, outError))
        {
            return false;
        }

        if (statusCode < 200 || statusCode >= 300)
        {
            outError = ExtractErrorMessage(responseBody);
            return false;
        }

        return ParseWorldSessionJson(responseBody, outResponse, outError);
    }

    bool MoveInventorySlotRequest(
        const AZStd::string& worldEndpoint,
        const AZStd::string& worldSessionToken,
        int fromSlotIndex,
        int toSlotIndex,
        WorldSessionResponse& outResponse,
        AZStd::string& outError)
    {
        AZStd::string responseBody;
        AZ::u32 statusCode = 0;
        const AZStd::string requestBody = AZStd::string::format(
            "{\"worldSessionToken\":\"%s\",\"fromSlotIndex\":%d,\"toSlotIndex\":%d}",
            worldSessionToken.c_str(),
            fromSlotIndex,
            toSlotIndex);
        if (!PerformRequest(worldEndpoint, L"POST", L"/v1/world/inventory/move", requestBody, responseBody, statusCode, outError))
        {
            return false;
        }

        if (statusCode < 200 || statusCode >= 300)
        {
            outError = ExtractErrorMessage(responseBody);
            return false;
        }

        return ParseWorldSessionJson(responseBody, outResponse, outError);
    }
} // namespace NetClient
