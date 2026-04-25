#include <ZoneStreaming/ZoneStreamingSystemComponent.h>

#include <AzCore/Serialization/SerializeContext.h>

namespace ZoneStreaming
{
    void ZoneStreamingSystemComponent::Reflect(AZ::ReflectContext* context)
    {
        if (auto* serializeContext = azrtti_cast<AZ::SerializeContext*>(context))
        {
            serializeContext->Class<ZoneStreamingSystemComponent, AZ::Component>()
                ->Version(0);
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
    }

    void ZoneStreamingSystemComponent::Deactivate()
    {
    }
}
