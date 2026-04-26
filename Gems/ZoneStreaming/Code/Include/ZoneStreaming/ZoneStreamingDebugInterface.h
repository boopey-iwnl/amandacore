#pragma once

#include <AzCore/Interface/Interface.h>
#include <AzCore/Math/Vector3.h>
#include <AzCore/RTTI/RTTI.h>
#include <AzCore/std/containers/vector.h>
#include <AzCore/std/string/string.h>

namespace ZoneStreaming
{
    struct DebugBounds
    {
        double m_minX = 0.0;
        double m_minY = 0.0;
        double m_minZ = 0.0;
        double m_maxX = 0.0;
        double m_maxY = 0.0;
        double m_maxZ = 0.0;
    };

    struct DebugPosition
    {
        double m_x = 0.0;
        double m_y = 0.0;
        double m_z = 0.0;
    };

    struct PlaceholderSceneCommand
    {
        AZStd::string m_command;
        AZStd::string m_zoneId;
        AZStd::string m_mapId;
        AZStd::string m_cellId;
        AZStd::string m_transitionId;
        AZStd::string m_targetZoneId;
        AZStd::string m_streamingCellId;
        AZStd::string m_displayName;
        AZStd::string m_hint;
        DebugBounds m_bounds;
        DebugPosition m_position;
        double m_radius = 0.0;
        bool m_ready = false;
        AZStd::vector<AZStd::string> m_tags;
    };

    struct DebugZoneVolume
    {
        AZStd::string m_zoneId;
        AZStd::string m_mapId;
        DebugBounds m_bounds;
    };

    struct DebugCellVolume
    {
        AZStd::string m_zoneId;
        AZStd::string m_mapId;
        AZStd::string m_cellId;
        AZStd::string m_displayName;
        DebugBounds m_bounds;
        AZStd::vector<AZStd::string> m_tags;
    };

    struct DebugTransitionAffordance
    {
        AZStd::string m_zoneId;
        AZStd::string m_mapId;
        AZStd::string m_transitionId;
        AZStd::string m_displayName;
        AZStd::string m_targetZoneId;
        AZStd::string m_streamingCellId;
        AZStd::string m_hint;
        DebugPosition m_position;
        double m_radius = 0.0;
        bool m_ready = false;
    };

    class IZoneStreamingDebugRequests
    {
    public:
        AZ_RTTI(IZoneStreamingDebugRequests, "{D78D2532-6A45-43F6-B50E-EB2378C62155}");

        virtual ~IZoneStreamingDebugRequests() = default;

        static IZoneStreamingDebugRequests* Get()
        {
            return AZ::Interface<IZoneStreamingDebugRequests>::Get();
        }

        static void Register(IZoneStreamingDebugRequests* instance)
        {
            AZ::Interface<IZoneStreamingDebugRequests>::Register(instance);
        }

        static void Unregister(IZoneStreamingDebugRequests* instance)
        {
            AZ::Interface<IZoneStreamingDebugRequests>::Unregister(instance);
        }

        virtual void ApplyPlaceholderSceneCommand(const PlaceholderSceneCommand& command) = 0;
        virtual void ResetDebugScene() = 0;
        virtual const DebugZoneVolume* GetZoneVolume() const = 0;
        virtual const DebugCellVolume* GetCellVolume(const AZStd::string& cellId) const = 0;
        virtual const DebugCellVolume* GetHighlightedCell() const = 0;
        virtual const DebugTransitionAffordance* GetTransitionAffordance() const = 0;
        virtual size_t GetVisibleCellCount() const = 0;
    };

    namespace PlaceholderSceneCommandNames
    {
        constexpr const char* CreateZoneBoundsVolume = "CreateZoneBoundsVolume";
        constexpr const char* CreateStreamingCellVolume = "CreateStreamingCellVolume";
        constexpr const char* HideStreamingCellVolume = "HideStreamingCellVolume";
        constexpr const char* HighlightCurrentCell = "HighlightCurrentCell";
        constexpr const char* ClearCurrentCellHighlight = "ClearCurrentCellHighlight";
        constexpr const char* ShowTransitionAffordance = "ShowTransitionAffordance";
        constexpr const char* ClearTransitionAffordance = "ClearTransitionAffordance";
    } // namespace PlaceholderSceneCommandNames
} // namespace ZoneStreaming
