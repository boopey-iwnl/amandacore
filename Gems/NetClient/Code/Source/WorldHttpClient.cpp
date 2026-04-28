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

        AZStd::string FormatJsonScalar(const rapidjson::Value& value)
        {
            if (value.IsString())
            {
                return value.GetString();
            }
            if (value.IsBool())
            {
                return value.GetBool() ? "true" : "false";
            }
            if (value.IsInt())
            {
                return AZStd::string::format("%d", value.GetInt());
            }
            if (value.IsInt64())
            {
                return AZStd::string::format("%lld", static_cast<long long>(value.GetInt64()));
            }
            if (value.IsUint64())
            {
                return AZStd::string::format("%llu", static_cast<unsigned long long>(value.GetUint64()));
            }
            if (value.IsNumber())
            {
                return AZStd::string::format("%.1f", value.GetDouble());
            }
            return {};
        }

        void AppendSummaryField(AZStd::string& summary, const rapidjson::Value& fields, const char* fieldName, const char* label = nullptr)
        {
            if (!fields.IsObject() || !fields.HasMember(fieldName))
            {
                return;
            }

            const AZStd::string value = FormatJsonScalar(fields[fieldName]);
            if (value.empty())
            {
                return;
            }

            if (!summary.empty())
            {
                summary += " ";
            }
            summary += label ? label : fieldName;
            summary += "=";
            summary += value;
        }

        AZStd::string BuildWorldEventSummary(const rapidjson::Value& eventValue)
        {
            const rapidjson::Value* fields = nullptr;
            if (eventValue.HasMember("fields") && eventValue["fields"].IsObject())
            {
                fields = &eventValue["fields"];
            }
            else if (eventValue.HasMember("payload") && eventValue["payload"].IsObject())
            {
                fields = &eventValue["payload"];
            }
            if (!fields)
            {
                return {};
            }

            AZStd::string summary;
            AppendSummaryField(summary, *fields, "abilityId", "ability");
            AppendSummaryField(summary, *fields, "auraId", "aura");
            AppendSummaryField(summary, *fields, "action");
            AppendSummaryField(summary, *fields, "targetId", "target");
            AppendSummaryField(summary, *fields, "targetEntityId", "target");
            AppendSummaryField(summary, *fields, "sourceEntityId", "source");
            AppendSummaryField(summary, *fields, "entityId", "entity");
            AppendSummaryField(summary, *fields, "damage");
            AppendSummaryField(summary, *fields, "amount");
            AppendSummaryField(summary, *fields, "health");
            AppendSummaryField(summary, *fields, "maxHealth", "max");
            AppendSummaryField(summary, *fields, "alive");
            AppendSummaryField(summary, *fields, "archetypeId", "credit");
            AppendSummaryField(summary, *fields, "count");
            AppendSummaryField(summary, *fields, "killCount", "kills");
            AppendSummaryField(summary, *fields, "reason");
            return summary;
        }

        void ParseAuraArray(const rapidjson::Value& source, AZStd::vector<AuraState>& outAuras)
        {
            outAuras.clear();
            if (!source.IsArray())
            {
                return;
            }

            for (const rapidjson::Value& auraValue : source.GetArray())
            {
                if (!auraValue.IsObject())
                {
                    continue;
                }

                AuraState aura;
                ReadString(auraValue, "auraId", aura.m_auraId);
                ReadString(auraValue, "displayName", aura.m_displayName);
                ReadString(auraValue, "kind", aura.m_kind);
                ReadString(auraValue, "sourceEntityId", aura.m_sourceEntityId);
                ReadString(auraValue, "targetEntityId", aura.m_targetEntityId);
                ReadInt(auraValue, "stackCount", aura.m_stackCount);
                ReadInt64(auraValue, "appliedAtMs", aura.m_appliedAtMs);
                ReadInt64(auraValue, "expiresAtMs", aura.m_expiresAtMs);
                ReadInt64(auraValue, "nextTickAtMs", aura.m_nextTickAtMs);
                outAuras.push_back(AZStd::move(aura));
            }
        }

        void ParseKillCreditArray(const rapidjson::Value& source, AZStd::vector<KillCreditState>& outCredits)
        {
            outCredits.clear();
            if (!source.IsArray())
            {
                return;
            }

            for (const rapidjson::Value& creditValue : source.GetArray())
            {
                if (!creditValue.IsObject())
                {
                    continue;
                }

                KillCreditState credit;
                ReadString(creditValue, "archetypeId", credit.m_archetypeId);
                ReadString(creditValue, "reason", credit.m_reason);
                ReadInt(creditValue, "count", credit.m_count);
                ReadInt64(creditValue, "updatedAt", credit.m_updatedAt);
                outCredits.push_back(AZStd::move(credit));
            }
        }

        void ParseWorldEventArray(const rapidjson::Value& source, AZStd::vector<WorldEventEntry>& outEvents, bool preferDiffType)
        {
            outEvents.clear();
            if (!source.IsArray())
            {
                return;
            }

            for (const rapidjson::Value& eventValue : source.GetArray())
            {
                if (!eventValue.IsObject())
                {
                    continue;
                }

                WorldEventEntry entry;
                ReadInt64(eventValue, "sequence", entry.m_sequence);
                ReadInt64(eventValue, "occurredAtMs", entry.m_occurredAtMs);
                ReadString(eventValue, "characterId", entry.m_characterId);
                ReadString(eventValue, "entityId", entry.m_entityId);
                ReadString(eventValue, "zoneId", entry.m_zoneId);

                if (preferDiffType)
                {
                    ReadString(eventValue, "diffType", entry.m_type);
                }
                if (entry.m_type.empty())
                {
                    ReadString(eventValue, "type", entry.m_type);
                }
                if (entry.m_type.empty())
                {
                    ReadString(eventValue, "eventName", entry.m_type);
                }
                if (entry.m_type.empty())
                {
                    ReadString(eventValue, "name", entry.m_type);
                }
                entry.m_summary = BuildWorldEventSummary(eventValue);
                outEvents.push_back(AZStd::move(entry));
            }
        }

        void ParseReplicationCursor(const rapidjson::Value& source, ReplicationCursorState& outCursor)
        {
            if (!source.IsObject())
            {
                return;
            }
            ReadString(source, "shardId", outCursor.m_shardId);
            ReadString(source, "zoneId", outCursor.m_zoneId);
            ReadInt64(source, "stateVersion", outCursor.m_stateVersion);
            ReadInt64(source, "sequence", outCursor.m_sequence);
            ReadInt64(source, "tick", outCursor.m_tick);
        }

        void ParseReplicationChangedArray(const rapidjson::Value& source, AZStd::vector<ReplicationChangedFieldState>& outChanged)
        {
            outChanged.clear();
            if (!source.IsArray())
            {
                return;
            }
            for (const rapidjson::Value& changeValue : source.GetArray())
            {
                if (!changeValue.IsObject())
                {
                    continue;
                }
                ReplicationChangedFieldState change;
                ReadString(changeValue, "domain", change.m_domain);
                ReadString(changeValue, "entityId", change.m_entityId);
                ReadInt64(changeValue, "version", change.m_version);
                if (changeValue.HasMember("fields") && changeValue["fields"].IsArray())
                {
                    for (const rapidjson::Value& fieldValue : changeValue["fields"].GetArray())
                    {
                        if (fieldValue.IsString())
                        {
                            change.m_fields.push_back(fieldValue.GetString());
                        }
                    }
                }
                outChanged.push_back(AZStd::move(change));
            }
        }

        void ParseReplicationMetadata(const rapidjson::Value& document, ReplicationMetadataState& outReplication)
        {
            outReplication = ReplicationMetadataState{};
            ReadInt64(document, "snapshotVersion", outReplication.m_snapshotVersion);
            ReadInt64(document, "deltaVersion", outReplication.m_deltaVersion);
            ReadString(document, "cursor", outReplication.m_cursor);
            ReadBool(document, "fullSnapshot", outReplication.m_fullSnapshot);
            ReadBool(document, "resyncRequired", outReplication.m_resyncRequired);
            if (document.HasMember("changed"))
            {
                ParseReplicationChangedArray(document["changed"], outReplication.m_changed);
            }

            if (!document.HasMember("replication") || !document["replication"].IsObject())
            {
                return;
            }

            const rapidjson::Value& replication = document["replication"];
            ReadString(replication, "protocolVersion", outReplication.m_protocolVersion);
            ReadString(replication, "kind", outReplication.m_kind);
            ReadString(replication, "cursor", outReplication.m_cursor);
            ReadString(replication, "reason", outReplication.m_reason);
            ReadInt64(replication, "snapshotVersion", outReplication.m_snapshotVersion);
            ReadInt64(replication, "deltaVersion", outReplication.m_deltaVersion);
            ReadBool(replication, "fullSnapshot", outReplication.m_fullSnapshot);
            ReadBool(replication, "resyncRequired", outReplication.m_resyncRequired);
            if (replication.HasMember("cursorState"))
            {
                ParseReplicationCursor(replication["cursorState"], outReplication.m_cursorState);
            }
            if (replication.HasMember("changed"))
            {
                ParseReplicationChangedArray(replication["changed"], outReplication.m_changed);
            }
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
            ReadInt(quest, "partyNearbyCount", outQuest.m_partyNearbyCount);
            ReadInt(quest, "partyEligibleCount", outQuest.m_partyEligibleCount);
            ReadDouble(quest, "partyCreditRadius", outQuest.m_partyCreditRadius);
            ReadString(quest, "partyStatusText", outQuest.m_partyStatusText);
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
                        ReadString(slotValue, "itemType", slot.m_itemType);
                        ReadString(slotValue, "itemSubtype", slot.m_itemSubtype);
                        ReadString(slotValue, "quality", slot.m_quality);
                        ReadString(slotValue, "iconKind", slot.m_iconKind);
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
                    ReadString(spellValue, "iconKind", entry.m_iconKind);
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
                    ReadString(actionValue, "iconKind", slot.m_iconKind);
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
                        ReadString(offerValue, "iconKind", offer.m_iconKind);
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
            outResponse.m_pvp = PvPState{};
            if (document.HasMember("pvp") && document["pvp"].IsObject())
            {
                const rapidjson::Value& pvp = document["pvp"];
                ReadBool(pvp, "duelsEnabled", outResponse.m_pvp.m_duelsEnabled);
                ReadBool(pvp, "incomingDuel", outResponse.m_pvp.m_incomingDuel);
                ReadBool(pvp, "outgoingDuel", outResponse.m_pvp.m_outgoingDuel);
                ReadString(pvp, "duelId", outResponse.m_pvp.m_duelId);
                ReadString(pvp, "duelState", outResponse.m_pvp.m_duelState);
                ReadString(pvp, "opponentCharacterId", outResponse.m_pvp.m_opponentCharacterId);
                ReadString(pvp, "opponentDisplayName", outResponse.m_pvp.m_opponentDisplayName);
                ReadInt64(pvp, "countdownEndsAt", outResponse.m_pvp.m_countdownEndsAt);
                ReadInt64(pvp, "startedAt", outResponse.m_pvp.m_startedAt);
                ReadInt64(pvp, "timeoutAt", outResponse.m_pvp.m_timeoutAt);
                if (pvp.HasMember("boundary") && pvp["boundary"].IsObject())
                {
                    const rapidjson::Value& boundary = pvp["boundary"];
                    ReadDouble(boundary, "centerX", outResponse.m_pvp.m_boundaryCenterX);
                    ReadDouble(boundary, "centerY", outResponse.m_pvp.m_boundaryCenterY);
                    ReadDouble(boundary, "maxDistance", outResponse.m_pvp.m_boundaryMaxDistance);
                }
                if (pvp.HasMember("stats") && pvp["stats"].IsObject())
                {
                    const rapidjson::Value& stats = pvp["stats"];
                    ReadString(stats, "characterId", outResponse.m_pvp.m_stats.m_characterId);
                    ReadInt(stats, "duelsWon", outResponse.m_pvp.m_stats.m_duelsWon);
                    ReadInt(stats, "duelsLost", outResponse.m_pvp.m_stats.m_duelsLost);
                    ReadInt(stats, "honorPoints", outResponse.m_pvp.m_stats.m_honorPoints);
                    ReadInt64(stats, "lastDuelWonAt", outResponse.m_pvp.m_stats.m_lastDuelWonAt);
                    ReadInt64(stats, "updatedAt", outResponse.m_pvp.m_stats.m_updatedAt);
                }
                if (pvp.HasMember("safeZone") && pvp["safeZone"].IsObject())
                {
                    const rapidjson::Value& safeZone = pvp["safeZone"];
                    ReadBool(safeZone, "noDuel", outResponse.m_pvp.m_safeZone.m_noDuel);
                    ReadBool(safeZone, "noHostileAction", outResponse.m_pvp.m_safeZone.m_noHostileAction);
                    ReadBool(safeZone, "sanctuary", outResponse.m_pvp.m_safeZone.m_sanctuary);
                }
                if (pvp.HasMember("lastResult") && pvp["lastResult"].IsObject())
                {
                    const rapidjson::Value& result = pvp["lastResult"];
                    ReadString(result, "duelId", outResponse.m_pvp.m_lastResult.m_duelId);
                    ReadString(result, "result", outResponse.m_pvp.m_lastResult.m_result);
                    ReadString(result, "reason", outResponse.m_pvp.m_lastResult.m_reason);
                    ReadString(result, "opponentCharacterId", outResponse.m_pvp.m_lastResult.m_opponentCharacterId);
                    ReadString(result, "opponentDisplayName", outResponse.m_pvp.m_lastResult.m_opponentDisplayName);
                    ReadString(result, "winnerCharacterId", outResponse.m_pvp.m_lastResult.m_winnerCharacterId);
                    ReadInt64(result, "endedAt", outResponse.m_pvp.m_lastResult.m_endedAt);
                }
            }
            ReadString(document, "currentTargetId", outResponse.m_currentTargetId);
            ReadBool(document, "autoAttackActive", outResponse.m_autoAttackActive);
            ReadInt64(document, "globalCooldownEndsAt", outResponse.m_globalCooldownEndsAt);
            ReadInt64(document, "castEndsAt", outResponse.m_castEndsAt);
            ReadString(document, "castingAbilityId", outResponse.m_castingAbilityId);
            if (document.HasMember("auras"))
            {
                ParseAuraArray(document["auras"], outResponse.m_auras);
            }
            if (document.HasMember("killCredits"))
            {
                ParseKillCreditArray(document["killCredits"], outResponse.m_killCredits);
            }
            if (document.HasMember("domainEvents"))
            {
                ParseWorldEventArray(document["domainEvents"], outResponse.m_domainEvents, false);
            }
            if (document.HasMember("stateDiffs"))
            {
                ParseWorldEventArray(document["stateDiffs"], outResponse.m_stateDiffs, true);
            }
            ParseReplicationMetadata(document, outResponse.m_replication);
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
                    ReadBool(entityValue, "duelOpponent", entity.m_duelOpponent);
                    ReadString(entityValue, "aiState", entity.m_aiState);
                    ReadString(entityValue, "pvpState", entity.m_pvpState);
                    if (entityValue.HasMember("auras"))
                    {
                        ParseAuraArray(entityValue["auras"], entity.m_auras);
                    }
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
                        ReadBool(memberValue, "sameZone", member.m_sameZone);
                        ReadDouble(memberValue, "distanceToPlayer", member.m_distanceToPlayer);
                        ReadBool(memberValue, "groupCreditEligible", member.m_groupCreditEligible);
                        ReadString(memberValue, "groupCreditStatus", member.m_groupCreditStatus);
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

        void ParseAuctionListing(const rapidjson::Value& listingValue, AuctionListingState& outListing)
        {
            ReadString(listingValue, "auctionId", outListing.m_auctionId);
            ReadString(listingValue, "sellerCharacterId", outListing.m_sellerCharacterId);
            ReadString(listingValue, "sellerDisplayName", outListing.m_sellerDisplayName);
            ReadString(listingValue, "buyerCharacterId", outListing.m_buyerCharacterId);
            ReadString(listingValue, "itemId", outListing.m_itemId);
            ReadString(listingValue, "itemDisplayName", outListing.m_itemDisplayName);
            ReadString(listingValue, "itemQuality", outListing.m_itemQuality);
            ReadString(listingValue, "itemType", outListing.m_itemType);
            ReadString(listingValue, "itemSubtype", outListing.m_itemSubtype);
            ReadString(listingValue, "state", outListing.m_state);
            ReadInt(listingValue, "stackCount", outListing.m_stackCount);
            ReadInt(listingValue, "buyoutCopper", outListing.m_buyoutCopper);
            ReadInt(listingValue, "depositCopper", outListing.m_depositCopper);
            ReadInt(listingValue, "cutCopper", outListing.m_cutCopper);
            ReadInt(listingValue, "cutPercent", outListing.m_cutPercent);
            ReadInt(listingValue, "version", outListing.m_version);
            ReadInt64(listingValue, "createdAt", outListing.m_createdAt);
            ReadInt64(listingValue, "expiresAt", outListing.m_expiresAt);
            ReadInt64(listingValue, "soldAt", outListing.m_soldAt);
            ReadInt64(listingValue, "canceledAt", outListing.m_canceledAt);
            ReadInt64(listingValue, "timeRemainingSeconds", outListing.m_timeRemainingSeconds);
        }

        void ParseAuctionSellSlot(const rapidjson::Value& slotValue, AuctionSellSlotState& outSlot)
        {
            ReadInt(slotValue, "slotIndex", outSlot.m_slotIndex);
            ReadString(slotValue, "itemId", outSlot.m_itemId);
            ReadString(slotValue, "displayName", outSlot.m_displayName);
            ReadInt(slotValue, "stackCount", outSlot.m_stackCount);
            ReadString(slotValue, "itemType", outSlot.m_itemType);
            ReadString(slotValue, "itemSubtype", outSlot.m_itemSubtype);
            ReadInt(slotValue, "sellPriceCopper", outSlot.m_sellPriceCopper);
            ReadInt(slotValue, "depositCopper", outSlot.m_depositCopper);
            ReadBool(slotValue, "tradeable", outSlot.m_tradeable);
            ReadString(slotValue, "blockedReason", outSlot.m_blockedReason);
        }

        void ParseMailEnvelope(const rapidjson::Value& mailValue, MailEnvelopeState& outMail)
        {
            ReadString(mailValue, "mailId", outMail.m_mailId);
            ReadString(mailValue, "auctionId", outMail.m_auctionId);
            ReadString(mailValue, "senderDisplayName", outMail.m_senderDisplayName);
            ReadString(mailValue, "recipientCharacterId", outMail.m_recipientCharacterId);
            ReadString(mailValue, "subject", outMail.m_subject);
            ReadString(mailValue, "body", outMail.m_body);
            ReadInt(mailValue, "currencyCopper", outMail.m_currencyCopper);
            ReadInt64(mailValue, "createdAt", outMail.m_createdAt);
            ReadInt64(mailValue, "deliveredAt", outMail.m_deliveredAt);
            if (mailValue.HasMember("itemAttachments") && mailValue["itemAttachments"].IsArray())
            {
                for (const rapidjson::Value& attachmentValue : mailValue["itemAttachments"].GetArray())
                {
                    if (!attachmentValue.IsObject())
                    {
                        continue;
                    }
                    MailItemAttachmentState attachment;
                    ReadString(attachmentValue, "itemId", attachment.m_itemId);
                    ReadString(attachmentValue, "displayName", attachment.m_displayName);
                    ReadInt(attachmentValue, "stackCount", attachment.m_stackCount);
                    outMail.m_itemAttachments.push_back(AZStd::move(attachment));
                }
            }
        }

        bool ParseAuctionStateJson(const AZStd::string& payload, AuctionStateResponse& outResponse, AZStd::string& outError)
        {
            outResponse = AuctionStateResponse{};

            rapidjson::Document document;
            document.Parse(payload.c_str());
            if (document.HasParseError() || !document.IsObject())
            {
                outError = "Auction state response was not valid JSON.";
                return false;
            }

            ReadInt64(document, "serverTime", outResponse.m_serverTime);
            ReadInt(document, "currencyCopper", outResponse.m_currencyCopper);
            if (document.HasMember("listings") && document["listings"].IsArray())
            {
                for (const rapidjson::Value& listingValue : document["listings"].GetArray())
                {
                    if (!listingValue.IsObject())
                    {
                        continue;
                    }
                    AuctionListingState listing;
                    ParseAuctionListing(listingValue, listing);
                    outResponse.m_listings.push_back(AZStd::move(listing));
                }
            }
            if (document.HasMember("myAuctions") && document["myAuctions"].IsArray())
            {
                for (const rapidjson::Value& listingValue : document["myAuctions"].GetArray())
                {
                    if (!listingValue.IsObject())
                    {
                        continue;
                    }
                    AuctionListingState listing;
                    ParseAuctionListing(listingValue, listing);
                    outResponse.m_myAuctions.push_back(AZStd::move(listing));
                }
            }
            if (document.HasMember("sellSlots") && document["sellSlots"].IsArray())
            {
                for (const rapidjson::Value& slotValue : document["sellSlots"].GetArray())
                {
                    if (!slotValue.IsObject())
                    {
                        continue;
                    }
                    AuctionSellSlotState slot;
                    ParseAuctionSellSlot(slotValue, slot);
                    outResponse.m_sellSlots.push_back(AZStd::move(slot));
                }
            }
            if (document.HasMember("mail") && document["mail"].IsArray())
            {
                for (const rapidjson::Value& mailValue : document["mail"].GetArray())
                {
                    if (!mailValue.IsObject())
                    {
                        continue;
                    }
                    MailEnvelopeState mail;
                    ParseMailEnvelope(mailValue, mail);
                    outResponse.m_mail.push_back(AZStd::move(mail));
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
        const AZStd::string& cursor,
        WorldSessionResponse& outResponse,
        AZStd::string& outError)
    {
        AZStd::string responseBody;
        AZ::u32 statusCode = 0;
        AZStd::string pathText = AZStd::string::format("/v1/world/state?worldSessionToken=%s", JsonEscape(worldSessionToken).c_str());
        if (!cursor.empty())
        {
            pathText += AZStd::string::format("&since=%s", JsonEscape(cursor).c_str());
        }
        const AZStd::wstring path = ToWideString(pathText);
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

    bool AuctionStateRequest(
        const AZStd::string& worldEndpoint,
        const AZStd::string& worldSessionToken,
        const AZStd::string& search,
        const AZStd::string& itemType,
        const AZStd::string& sort,
        AuctionStateResponse& outResponse,
        AZStd::string& outError)
    {
        AZStd::string responseBody;
        AZ::u32 statusCode = 0;
        AZStd::string path = AZStd::string::format(
            "/v1/world/auction/listings?worldSessionToken=%s",
            JsonEscape(worldSessionToken).c_str());
        if (!search.empty())
        {
            path += AZStd::string::format("&search=%s", JsonEscape(search).c_str());
        }
        if (!itemType.empty())
        {
            path += AZStd::string::format("&itemType=%s", JsonEscape(itemType).c_str());
        }
        if (!sort.empty())
        {
            path += AZStd::string::format("&sort=%s", JsonEscape(sort).c_str());
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

        return ParseAuctionStateJson(responseBody, outResponse, outError);
    }

    bool AuctionPostRequest(
        const AZStd::string& worldEndpoint,
        const wchar_t* path,
        const AZStd::string& requestBody,
        AuctionStateResponse& outResponse,
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

        return ParseAuctionStateJson(responseBody, outResponse, outError);
    }

    bool ListAuctionItemRequest(
        const AZStd::string& worldEndpoint,
        const AZStd::string& worldSessionToken,
        int slotIndex,
        int stackCount,
        int buyoutCopper,
        AZ::s64 durationSeconds,
        AuctionStateResponse& outResponse,
        AZStd::string& outError)
    {
        const AZStd::string requestBody = AZStd::string::format(
            "{\"worldSessionToken\":\"%s\",\"slotIndex\":%d,\"stackCount\":%d,\"buyoutCopper\":%d,\"durationSeconds\":%lld}",
            JsonEscape(worldSessionToken).c_str(),
            slotIndex,
            stackCount,
            buyoutCopper,
            static_cast<long long>(durationSeconds));
        return AuctionPostRequest(worldEndpoint, L"/v1/world/auction/list", requestBody, outResponse, outError);
    }

    bool AuctionIdActionRequest(
        const AZStd::string& worldEndpoint,
        const wchar_t* path,
        const AZStd::string& worldSessionToken,
        const AZStd::string& auctionId,
        AuctionStateResponse& outResponse,
        AZStd::string& outError)
    {
        const AZStd::string requestBody = AZStd::string::format(
            "{\"worldSessionToken\":\"%s\",\"auctionId\":\"%s\"}",
            JsonEscape(worldSessionToken).c_str(),
            JsonEscape(auctionId).c_str());
        return AuctionPostRequest(worldEndpoint, path, requestBody, outResponse, outError);
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

    bool EnterDungeonRequest(
        const AZStd::string& worldEndpoint,
        const AZStd::string& worldSessionToken,
        const AZStd::string& dungeonId,
        WorldSessionResponse& outResponse,
        AZStd::string& outError)
    {
        AZStd::string responseBody;
        AZ::u32 statusCode = 0;
        const AZStd::string requestBody = AZStd::string::format(
            "{\"worldSessionToken\":\"%s\",\"dungeonId\":\"%s\"}",
            worldSessionToken.c_str(),
            dungeonId.c_str());
        if (!PerformRequest(worldEndpoint, L"POST", L"/v1/world/dungeon/enter", requestBody, responseBody, statusCode, outError))
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

    bool ExitDungeonRequest(
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
        if (!PerformRequest(worldEndpoint, L"POST", L"/v1/world/dungeon/exit", requestBody, responseBody, statusCode, outError))
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

    bool DuelPostRequest(
        const AZStd::string& worldEndpoint,
        const wchar_t* path,
        const AZStd::string& requestBody,
        WorldSessionResponse& outResponse,
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

        return ParseWorldSessionJson(responseBody, outResponse, outError);
    }

    bool RequestDuelRequest(
        const AZStd::string& worldEndpoint,
        const AZStd::string& worldSessionToken,
        const AZStd::string& targetCharacterId,
        const AZStd::string& targetName,
        WorldSessionResponse& outResponse,
        AZStd::string& outError)
    {
        const AZStd::string requestBody = AZStd::string::format(
            "{\"worldSessionToken\":\"%s\",\"targetCharacterId\":\"%s\",\"targetName\":\"%s\"}",
            JsonEscape(worldSessionToken).c_str(),
            JsonEscape(targetCharacterId).c_str(),
            JsonEscape(targetName).c_str());
        return DuelPostRequest(worldEndpoint, L"/v1/world/duel/request", requestBody, outResponse, outError);
    }

    bool DuelActionRequest(
        const AZStd::string& worldEndpoint,
        const wchar_t* path,
        const AZStd::string& worldSessionToken,
        const AZStd::string& duelId,
        WorldSessionResponse& outResponse,
        AZStd::string& outError)
    {
        const AZStd::string requestBody = AZStd::string::format(
            "{\"worldSessionToken\":\"%s\",\"duelId\":\"%s\"}",
            JsonEscape(worldSessionToken).c_str(),
            JsonEscape(duelId).c_str());
        return DuelPostRequest(worldEndpoint, path, requestBody, outResponse, outError);
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
