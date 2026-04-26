#include <ZoneStreaming/ZoneStreamingSystemComponent.h>

#include <Atom/RPI.Public/AuxGeom/AuxGeomDraw.h>
#include <Atom/RPI.Public/AuxGeom/AuxGeomFeatureProcessorInterface.h>
#include <Atom/RPI.Public/Scene.h>
#include <AzCore/Debug/Trace.h>
#include <AzCore/Math/Aabb.h>
#include <AzCore/Math/Color.h>
#include <AzCore/Serialization/SerializeContext.h>
#include <AzCore/std/algorithm.h>
#include <cstdlib>
#include <fstream>
#include <rapidjson/document.h>

namespace ZoneStreaming
{
    namespace
    {
        constexpr float CommandStreamPollIntervalSeconds = 0.10f;
        constexpr const char* CommandStreamEnvironmentVariable = "AMANDACORE_STREAMING_COMMAND_FILE";

        AZ::Aabb BoundsToAabb(const DebugBounds& bounds)
        {
            AZ::Vector3 min(
                static_cast<float>(bounds.m_minX),
                static_cast<float>(bounds.m_minY),
                static_cast<float>(bounds.m_minZ));
            AZ::Vector3 max(
                static_cast<float>(bounds.m_maxX),
                static_cast<float>(bounds.m_maxY),
                static_cast<float>(bounds.m_maxZ));

            if (max.GetX() <= min.GetX())
            {
                max.SetX(min.GetX() + 0.1f);
            }
            if (max.GetY() <= min.GetY())
            {
                max.SetY(min.GetY() + 0.1f);
            }
            if (max.GetZ() <= min.GetZ())
            {
                max.SetZ(min.GetZ() + 0.5f);
            }

            return AZ::Aabb::CreateFromMinMax(min, max);
        }

        AZ::Vector3 PositionToVector3(const DebugPosition& position)
        {
            return AZ::Vector3(
                static_cast<float>(position.m_x),
                static_cast<float>(position.m_y),
                static_cast<float>(position.m_z));
        }

        const rapidjson::Value* FindMember(const rapidjson::Value& object, const char* pascalName, const char* camelName)
        {
            if (!object.IsObject())
            {
                return nullptr;
            }

            if (object.HasMember(pascalName))
            {
                return &object[pascalName];
            }

            if (camelName && object.HasMember(camelName))
            {
                return &object[camelName];
            }

            return nullptr;
        }

        bool ReadOptionalString(
            const rapidjson::Value& object,
            const char* pascalName,
            const char* camelName,
            AZStd::string& outValue,
            AZStd::string& outError)
        {
            const rapidjson::Value* value = FindMember(object, pascalName, camelName);
            if (!value || value->IsNull())
            {
                return true;
            }
            if (!value->IsString())
            {
                outError = AZStd::string::format("Expected string field '%s'.", pascalName);
                return false;
            }

            outValue = value->GetString();
            return true;
        }

        bool ReadRequiredString(
            const rapidjson::Value& object,
            const char* pascalName,
            const char* camelName,
            AZStd::string& outValue,
            AZStd::string& outError)
        {
            const rapidjson::Value* value = FindMember(object, pascalName, camelName);
            if (!value || !value->IsString())
            {
                outError = AZStd::string::format("Expected required string field '%s'.", pascalName);
                return false;
            }

            outValue = value->GetString();
            return true;
        }

        bool ReadOptionalDouble(
            const rapidjson::Value& object,
            const char* pascalName,
            const char* camelName,
            double& outValue,
            AZStd::string& outError)
        {
            const rapidjson::Value* value = FindMember(object, pascalName, camelName);
            if (!value || value->IsNull())
            {
                return true;
            }
            if (!value->IsNumber())
            {
                outError = AZStd::string::format("Expected numeric field '%s'.", pascalName);
                return false;
            }

            outValue = value->GetDouble();
            return true;
        }

        bool ReadOptionalBool(
            const rapidjson::Value& object,
            const char* pascalName,
            const char* camelName,
            bool& outValue,
            AZStd::string& outError)
        {
            const rapidjson::Value* value = FindMember(object, pascalName, camelName);
            if (!value || value->IsNull())
            {
                return true;
            }
            if (!value->IsBool())
            {
                outError = AZStd::string::format("Expected bool field '%s'.", pascalName);
                return false;
            }

            outValue = value->GetBool();
            return true;
        }

        bool ReadBounds(const rapidjson::Value& object, DebugBounds& outBounds, AZStd::string& outError)
        {
            const rapidjson::Value* bounds = FindMember(object, "Bounds", "bounds");
            if (!bounds || bounds->IsNull())
            {
                return true;
            }
            if (!bounds->IsObject())
            {
                outError = "Expected object field 'Bounds'.";
                return false;
            }

            return ReadOptionalDouble(*bounds, "MinX", "minX", outBounds.m_minX, outError) &&
                ReadOptionalDouble(*bounds, "MinY", "minY", outBounds.m_minY, outError) &&
                ReadOptionalDouble(*bounds, "MinZ", "minZ", outBounds.m_minZ, outError) &&
                ReadOptionalDouble(*bounds, "MaxX", "maxX", outBounds.m_maxX, outError) &&
                ReadOptionalDouble(*bounds, "MaxY", "maxY", outBounds.m_maxY, outError) &&
                ReadOptionalDouble(*bounds, "MaxZ", "maxZ", outBounds.m_maxZ, outError);
        }

        bool ReadPosition(const rapidjson::Value& object, DebugPosition& outPosition, AZStd::string& outError)
        {
            const rapidjson::Value* position = FindMember(object, "Position", "position");
            if (!position || position->IsNull())
            {
                return true;
            }
            if (!position->IsObject())
            {
                outError = "Expected object field 'Position'.";
                return false;
            }

            return ReadOptionalDouble(*position, "X", "x", outPosition.m_x, outError) &&
                ReadOptionalDouble(*position, "Y", "y", outPosition.m_y, outError) &&
                ReadOptionalDouble(*position, "Z", "z", outPosition.m_z, outError);
        }

        bool ReadTags(const rapidjson::Value& object, AZStd::vector<AZStd::string>& outTags, AZStd::string& outError)
        {
            const rapidjson::Value* tags = FindMember(object, "Tags", "tags");
            if (!tags || tags->IsNull())
            {
                return true;
            }
            if (!tags->IsArray())
            {
                outError = "Expected array field 'Tags'.";
                return false;
            }

            outTags.clear();
            for (const rapidjson::Value& tag : tags->GetArray())
            {
                if (!tag.IsString())
                {
                    outError = "Expected string values in 'Tags'.";
                    return false;
                }
                outTags.emplace_back(tag.GetString());
            }
            return true;
        }

        bool ParsePlaceholderSceneCommandJson(
            const AZStd::string& line,
            PlaceholderSceneCommand& outCommand,
            AZStd::string& outError)
        {
            rapidjson::Document document;
            document.Parse(line.c_str());
            if (document.HasParseError() || !document.IsObject())
            {
                outError = "Malformed placeholder scene command JSON.";
                return false;
            }

            PlaceholderSceneCommand parsed;
            if (!ReadRequiredString(document, "Command", "command", parsed.m_command, outError) ||
                !ReadOptionalString(document, "ZoneId", "zoneId", parsed.m_zoneId, outError) ||
                !ReadOptionalString(document, "MapId", "mapId", parsed.m_mapId, outError) ||
                !ReadOptionalString(document, "CellId", "cellId", parsed.m_cellId, outError) ||
                !ReadOptionalString(document, "TransitionId", "transitionId", parsed.m_transitionId, outError) ||
                !ReadOptionalString(document, "TargetZoneId", "targetZoneId", parsed.m_targetZoneId, outError) ||
                !ReadOptionalString(document, "StreamingCellId", "streamingCellId", parsed.m_streamingCellId, outError) ||
                !ReadOptionalString(document, "DisplayName", "displayName", parsed.m_displayName, outError) ||
                !ReadOptionalString(document, "Hint", "hint", parsed.m_hint, outError) ||
                !ReadBounds(document, parsed.m_bounds, outError) ||
                !ReadPosition(document, parsed.m_position, outError) ||
                !ReadOptionalDouble(document, "Radius", "radius", parsed.m_radius, outError) ||
                !ReadOptionalBool(document, "Ready", "ready", parsed.m_ready, outError) ||
                !ReadTags(document, parsed.m_tags, outError))
            {
                return false;
            }

            outCommand = AZStd::move(parsed);
            return true;
        }

        void TrimLine(AZStd::string& line)
        {
            if (line.size() >= 3 &&
                static_cast<unsigned char>(line[0]) == 0xEF &&
                static_cast<unsigned char>(line[1]) == 0xBB &&
                static_cast<unsigned char>(line[2]) == 0xBF)
            {
                line.erase(0, 3);
            }
            while (!line.empty() && (line.back() == '\r' || line.back() == '\n'))
            {
                line.pop_back();
            }
        }
    } // namespace

    void ZoneStreamingSystemComponent::Reflect(AZ::ReflectContext* context)
    {
        if (auto* serializeContext = azrtti_cast<AZ::SerializeContext*>(context))
        {
            serializeContext->Class<ZoneStreamingSystemComponent, AZ::Component>()
                ->Version(1);
        }
    }

    void ZoneStreamingSystemComponent::GetProvidedServices(AZ::ComponentDescriptor::DependencyArrayType& provided)
    {
        provided.push_back(AZ_CRC_CE("ZoneStreamingService"));
    }

    void ZoneStreamingSystemComponent::GetIncompatibleServices(AZ::ComponentDescriptor::DependencyArrayType& incompatible)
    {
        incompatible.push_back(AZ_CRC_CE("ZoneStreamingService"));
    }

    void ZoneStreamingSystemComponent::GetRequiredServices(AZ::ComponentDescriptor::DependencyArrayType& required)
    {
        required.push_back(AZ_CRC_CE("GameCoreService"));
        required.push_back(AZ_CRC_CE("NetClientService"));
    }

    void ZoneStreamingSystemComponent::GetDependentServices(AZ::ComponentDescriptor::DependencyArrayType&)
    {
    }

    void ZoneStreamingSystemComponent::Activate()
    {
        ResetDebugScene();
        ConfigureCommandStreamFromEnvironment();
        IZoneStreamingDebugRequests::Register(this);
        AZ::TickBus::Handler::BusConnect();
    }

    void ZoneStreamingSystemComponent::Deactivate()
    {
        AZ::TickBus::Handler::BusDisconnect();
        IZoneStreamingDebugRequests::Unregister(this);
        ResetDebugScene();
    }

    void ZoneStreamingSystemComponent::OnTick(float deltaTime, AZ::ScriptTimePoint)
    {
        PollCommandStream(deltaTime);
        DrawDebugVolumes();
    }

    void ZoneStreamingSystemComponent::ApplyPlaceholderSceneCommand(const PlaceholderSceneCommand& command)
    {
        if (command.m_command == PlaceholderSceneCommandNames::CreateZoneBoundsVolume)
        {
            m_zoneVolume = DebugZoneVolume{ command.m_zoneId, command.m_mapId, command.m_bounds };
            m_hasZoneVolume = true;
            m_loggedDebugSceneActive = false;
            return;
        }

        if (command.m_command == PlaceholderSceneCommandNames::CreateStreamingCellVolume)
        {
            if (!command.m_cellId.empty())
            {
                m_cellVolumes[command.m_cellId] = DebugCellVolume{
                    command.m_zoneId,
                    command.m_mapId,
                    command.m_cellId,
                    command.m_displayName,
                    command.m_bounds,
                    command.m_tags
                };
            }
            return;
        }

        if (command.m_command == PlaceholderSceneCommandNames::HideStreamingCellVolume)
        {
            m_cellVolumes.erase(command.m_cellId);
            if (m_highlightedCellId == command.m_cellId)
            {
                m_highlightedCellId.clear();
            }
            return;
        }

        if (command.m_command == PlaceholderSceneCommandNames::HighlightCurrentCell)
        {
            m_highlightedCellId = command.m_cellId;
            return;
        }

        if (command.m_command == PlaceholderSceneCommandNames::ClearCurrentCellHighlight)
        {
            m_highlightedCellId.clear();
            return;
        }

        if (command.m_command == PlaceholderSceneCommandNames::ShowTransitionAffordance)
        {
            m_transitionAffordance = DebugTransitionAffordance{
                command.m_zoneId,
                command.m_mapId,
                command.m_transitionId,
                command.m_displayName,
                command.m_targetZoneId,
                command.m_streamingCellId,
                command.m_hint,
                command.m_position,
                command.m_radius,
                command.m_ready
            };
            m_hasTransitionAffordance = true;
            return;
        }

        if (command.m_command == PlaceholderSceneCommandNames::ClearTransitionAffordance)
        {
            m_transitionAffordance = DebugTransitionAffordance{};
            m_hasTransitionAffordance = false;
        }
    }

    void ZoneStreamingSystemComponent::SetCommandStreamPath(const AZStd::string& path)
    {
        m_commandStreamPath = path;
        m_commandStreamOffset = 0;
        m_commandStreamPendingLine.clear();
        m_commandStreamPollAccumulator = CommandStreamPollIntervalSeconds;
        m_commandStreamLinesRead = 0;
        m_commandStreamCommandsApplied = 0;
        m_commandStreamParseErrors = 0;
        m_loggedCommandStreamActive = false;
        m_loggedCommandStreamMissing = false;

        if (!m_commandStreamPath.empty())
        {
            AZ_Printf(
                "amandacore",
                "zone_streaming.command_stream_configured path=%s",
                m_commandStreamPath.c_str());
        }
    }

    void ZoneStreamingSystemComponent::ResetDebugScene()
    {
        m_zoneVolume = DebugZoneVolume{};
        m_cellVolumes.clear();
        m_highlightedCellId.clear();
        m_transitionAffordance = DebugTransitionAffordance{};
        m_hasZoneVolume = false;
        m_hasTransitionAffordance = false;
        m_loggedDebugSceneActive = false;
    }

    const DebugZoneVolume* ZoneStreamingSystemComponent::GetZoneVolume() const
    {
        return m_hasZoneVolume ? &m_zoneVolume : nullptr;
    }

    const DebugCellVolume* ZoneStreamingSystemComponent::GetCellVolume(const AZStd::string& cellId) const
    {
        const auto it = m_cellVolumes.find(cellId);
        return it != m_cellVolumes.end() ? &it->second : nullptr;
    }

    const DebugCellVolume* ZoneStreamingSystemComponent::GetHighlightedCell() const
    {
        if (m_highlightedCellId.empty())
        {
            return nullptr;
        }
        return GetCellVolume(m_highlightedCellId);
    }

    const DebugTransitionAffordance* ZoneStreamingSystemComponent::GetTransitionAffordance() const
    {
        return m_hasTransitionAffordance ? &m_transitionAffordance : nullptr;
    }

    CommandStreamBridgeStatus ZoneStreamingSystemComponent::GetCommandStreamBridgeStatus() const
    {
        return CommandStreamBridgeStatus{
            m_commandStreamPath,
            !m_commandStreamPath.empty(),
            m_commandStreamLinesRead,
            m_commandStreamCommandsApplied,
            m_commandStreamParseErrors
        };
    }

    size_t ZoneStreamingSystemComponent::GetVisibleCellCount() const
    {
        return m_cellVolumes.size();
    }

    void ZoneStreamingSystemComponent::ConfigureCommandStreamFromEnvironment()
    {
        const char* streamPath = std::getenv(CommandStreamEnvironmentVariable);
        if (streamPath && streamPath[0] != '\0')
        {
            SetCommandStreamPath(streamPath);
        }
    }

    void ZoneStreamingSystemComponent::PollCommandStream(float deltaTime)
    {
        if (m_commandStreamPath.empty())
        {
            return;
        }

        m_commandStreamPollAccumulator += deltaTime;
        if (m_commandStreamPollAccumulator < CommandStreamPollIntervalSeconds)
        {
            return;
        }
        m_commandStreamPollAccumulator = 0.0f;

        std::ifstream stream(m_commandStreamPath.c_str(), std::ios::binary);
        if (!stream)
        {
            if (!m_loggedCommandStreamMissing)
            {
                m_loggedCommandStreamMissing = true;
                AZ_Warning(
                    "amandacore",
                    false,
                    "ZoneStreaming command stream is not readable yet: %s",
                    m_commandStreamPath.c_str());
            }
            return;
        }

        m_loggedCommandStreamMissing = false;

        stream.seekg(0, std::ios::end);
        const std::streamoff fileSize = stream.tellg();
        if (fileSize <= 0)
        {
            return;
        }

        if (static_cast<AZ::u64>(fileSize) < m_commandStreamOffset)
        {
            m_commandStreamOffset = 0;
            m_commandStreamPendingLine.clear();
        }

        if (static_cast<AZ::u64>(fileSize) == m_commandStreamOffset)
        {
            return;
        }

        const AZ::u64 bytesToRead = static_cast<AZ::u64>(fileSize) - m_commandStreamOffset;
        stream.seekg(static_cast<std::streamoff>(m_commandStreamOffset), std::ios::beg);

        AZStd::vector<char> buffer(static_cast<size_t>(bytesToRead));
        stream.read(buffer.data(), static_cast<std::streamsize>(buffer.size()));
        const std::streamsize bytesRead = stream.gcount();
        if (bytesRead <= 0)
        {
            return;
        }

        m_commandStreamOffset += static_cast<AZ::u64>(bytesRead);
        ProcessCommandStreamChunk(buffer.data(), static_cast<size_t>(bytesRead));

        if (!m_loggedCommandStreamActive && m_commandStreamCommandsApplied > 0)
        {
            m_loggedCommandStreamActive = true;
            AZ_Printf(
                "amandacore",
                "zone_streaming.command_stream_active path=%s commands=%zu",
                m_commandStreamPath.c_str(),
                m_commandStreamCommandsApplied);
        }
    }

    void ZoneStreamingSystemComponent::ProcessCommandStreamChunk(const char* data, size_t size)
    {
        m_commandStreamPendingLine.append(data, size);

        size_t newline = m_commandStreamPendingLine.find('\n');
        while (newline != AZStd::string::npos)
        {
            AZStd::string line = m_commandStreamPendingLine.substr(0, newline + 1);
            m_commandStreamPendingLine.erase(0, newline + 1);
            ProcessCommandStreamLine(line);
            newline = m_commandStreamPendingLine.find('\n');
        }
    }

    void ZoneStreamingSystemComponent::ProcessCommandStreamLine(const AZStd::string& line)
    {
        AZStd::string trimmed = line;
        TrimLine(trimmed);
        if (trimmed.empty())
        {
            return;
        }

        ++m_commandStreamLinesRead;

        PlaceholderSceneCommand command;
        AZStd::string error;
        if (!ParsePlaceholderSceneCommandJson(trimmed, command, error))
        {
            ++m_commandStreamParseErrors;
            if (m_commandStreamParseErrors <= 5)
            {
                AZ_Warning(
                    "amandacore",
                    false,
                    "ZoneStreaming command stream parse error at line %zu: %s",
                    m_commandStreamLinesRead,
                    error.c_str());
            }
            return;
        }

        ApplyPlaceholderSceneCommand(command);
        ++m_commandStreamCommandsApplied;
    }

    void ZoneStreamingSystemComponent::DrawDebugVolumes()
    {
        if (!m_hasZoneVolume && m_cellVolumes.empty() && !m_hasTransitionAffordance)
        {
            return;
        }

        AZ::RPI::Scene* scene = AZ::RPI::Scene::GetSceneForEntityId(GetEntityId());
        if (!scene)
        {
            return;
        }

        auto auxGeom = AZ::RPI::AuxGeomFeatureProcessorInterface::GetDrawQueueForScene(scene);
        if (!auxGeom)
        {
            return;
        }

        if (!m_loggedDebugSceneActive)
        {
            m_loggedDebugSceneActive = true;
            AZ_Printf(
                "amandacore",
                "zone_streaming.debug_visualization_active zone=%s map=%s cells=%zu transition=%s",
                m_zoneVolume.m_zoneId.c_str(),
                m_zoneVolume.m_mapId.c_str(),
                m_cellVolumes.size(),
                m_transitionAffordance.m_transitionId.c_str());
        }

        const AZ::Color zoneColor(0.18f, 0.58f, 0.92f, 0.24f);
        const AZ::Color cellColor(0.24f, 0.78f, 0.86f, 0.30f);
        const AZ::Color currentCellColor(0.95f, 0.75f, 0.28f, 0.46f);
        const AZ::Color transitionReadyColor(0.24f, 0.86f, 0.45f, 0.88f);
        const AZ::Color transitionHintColor(0.96f, 0.58f, 0.24f, 0.88f);

        if (m_hasZoneVolume)
        {
            auxGeom->DrawAabb(
                BoundsToAabb(m_zoneVolume.m_bounds),
                zoneColor,
                AZ::RPI::AuxGeomDraw::DrawStyle::Shaded);
        }

        for (const auto& [cellId, cell] : m_cellVolumes)
        {
            const bool highlighted = cellId == m_highlightedCellId;
            auxGeom->DrawAabb(
                BoundsToAabb(cell.m_bounds),
                highlighted ? currentCellColor : cellColor,
                AZ::RPI::AuxGeomDraw::DrawStyle::Shaded);
        }

        if (m_hasTransitionAffordance)
        {
            const AZ::Vector3 position = PositionToVector3(m_transitionAffordance.m_position);
            const float radius = AZStd::max(0.25f, static_cast<float>(m_transitionAffordance.m_radius));
            const AZ::Color color = m_transitionAffordance.m_ready ? transitionReadyColor : transitionHintColor;
            auxGeom->DrawSphere(position, radius, color);
            auxGeom->DrawSphere(position + AZ::Vector3(0.0f, 0.0f, radius + 0.5f), 0.35f, color);
        }
    }
}
