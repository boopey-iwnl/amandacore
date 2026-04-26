#pragma once

#include <AzCore/Component/Component.h>
#include <AzCore/Component/TickBus.h>
#include <AzCore/std/containers/unordered_map.h>
#include <ZoneStreaming/ZoneStreamingDebugInterface.h>

namespace ZoneStreaming
{
    class ZoneStreamingSystemComponent final
        : public AZ::Component
        , public AZ::TickBus::Handler
        , public IZoneStreamingDebugRequests
    {
    public:
        AZ_COMPONENT(ZoneStreamingSystemComponent, "{54B217C3-527A-40F0-B0D0-8FA7A3232A3B}");

        static void Reflect(AZ::ReflectContext* context);
        static void GetProvidedServices(AZ::ComponentDescriptor::DependencyArrayType& provided);
        static void GetIncompatibleServices(AZ::ComponentDescriptor::DependencyArrayType& incompatible);
        static void GetRequiredServices(AZ::ComponentDescriptor::DependencyArrayType& required);
        static void GetDependentServices(AZ::ComponentDescriptor::DependencyArrayType& dependent);

        void Activate() override;
        void Deactivate() override;

        void OnTick(float deltaTime, AZ::ScriptTimePoint time) override;

        void ApplyPlaceholderSceneCommand(const PlaceholderSceneCommand& command) override;
        void ResetDebugScene() override;
        const DebugZoneVolume* GetZoneVolume() const override;
        const DebugCellVolume* GetCellVolume(const AZStd::string& cellId) const override;
        const DebugCellVolume* GetHighlightedCell() const override;
        const DebugTransitionAffordance* GetTransitionAffordance() const override;
        size_t GetVisibleCellCount() const override;

    private:
        void DrawDebugVolumes();

        DebugZoneVolume m_zoneVolume;
        AZStd::unordered_map<AZStd::string, DebugCellVolume> m_cellVolumes;
        AZStd::string m_highlightedCellId;
        DebugTransitionAffordance m_transitionAffordance;
        bool m_hasZoneVolume = false;
        bool m_hasTransitionAffordance = false;
        bool m_loggedDebugSceneActive = false;
    };
}
