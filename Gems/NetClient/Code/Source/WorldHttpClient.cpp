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
                    ReadString(spellValue, "description", entry.m_description);
                    ReadString(spellValue, "requirementText", entry.m_requirementText);
                    ReadInt(spellValue, "requiredLevel", entry.m_requiredLevel);
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
                    ReadBool(actionValue, "requiresTarget", slot.m_requiresTarget);
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
                        ReadString(offerValue, "requirementText", offer.m_requirementText);
                        ReadInt(offerValue, "requiredLevel", offer.m_requiredLevel);
                        ReadInt(offerValue, "costCopper", offer.m_costCopper);
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
                const rapidjson::Value& quest = document["quest"];
                ReadString(quest, "id", outResponse.m_quest.m_id);
                ReadString(quest, "title", outResponse.m_quest.m_title);
                ReadString(quest, "objectiveType", outResponse.m_quest.m_objectiveType);
                ReadString(quest, "objectiveText", outResponse.m_quest.m_objectiveText);
                ReadString(quest, "state", outResponse.m_quest.m_state);
                ReadString(quest, "giverNpcId", outResponse.m_quest.m_giverNpcId);
                ReadString(quest, "turnInNpcId", outResponse.m_quest.m_turnInNpcId);
                ReadInt(quest, "currentCount", outResponse.m_quest.m_currentCount);
                ReadInt(quest, "targetCount", outResponse.m_quest.m_targetCount);
                ReadInt(quest, "rewardXp", outResponse.m_quest.m_rewardXp);
                ReadInt(quest, "rewardCurrencyCopper", outResponse.m_quest.m_rewardCurrencyTotalCopper);
                if (quest.HasMember("rewardCurrency") && quest["rewardCurrency"].IsObject())
                {
                    const rapidjson::Value& rewardCurrency = quest["rewardCurrency"];
                    ReadInt(rewardCurrency, "gold", outResponse.m_quest.m_rewardCurrencyGold);
                    ReadInt(rewardCurrency, "silver", outResponse.m_quest.m_rewardCurrencySilver);
                    ReadInt(rewardCurrency, "copper", outResponse.m_quest.m_rewardCurrencyCopper);
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
