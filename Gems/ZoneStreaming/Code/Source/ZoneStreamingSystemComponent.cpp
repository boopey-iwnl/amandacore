#include <ZoneStreaming/ZoneStreamingSystemComponent.h>

#include <Atom/RPI.Public/AuxGeom/AuxGeomDraw.h>
#include <Atom/RPI.Public/AuxGeom/AuxGeomFeatureProcessorInterface.h>
#include <Atom/RPI.Public/Scene.h>
#include <AzCore/Debug/Trace.h>
#include <AzCore/Math/Aabb.h>
#include <AzCore/Math/Color.h>
#include <AzCore/Serialization/SerializeContext.h>
#include <AzCore/std/algorithm.h>

namespace ZoneStreaming
{
    namespace
    {
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
        IZoneStreamingDebugRequests::Register(this);
        AZ::TickBus::Handler::BusConnect();
    }

    void ZoneStreamingSystemComponent::Deactivate()
    {
        AZ::TickBus::Handler::BusDisconnect();
        IZoneStreamingDebugRequests::Unregister(this);
        ResetDebugScene();
    }

    void ZoneStreamingSystemComponent::OnTick(float, AZ::ScriptTimePoint)
    {
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

    size_t ZoneStreamingSystemComponent::GetVisibleCellCount() const
    {
        return m_cellVolumes.size();
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
