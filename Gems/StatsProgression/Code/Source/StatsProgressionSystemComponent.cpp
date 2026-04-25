#include <StatsProgression/StatsProgressionSystemComponent.h>

#include <AzCore/Serialization/SerializeContext.h>

namespace StatsProgression
{
    void StatsProgressionSystemComponent::Reflect(AZ::ReflectContext* context)
    {
        if (auto* serializeContext = azrtti_cast<AZ::SerializeContext*>(context))
        {
            serializeContext->Class<StatsProgressionSystemComponent, AZ::Component>()
                ->Version(0);
        }
    }

    void StatsProgressionSystemComponent::GetProvidedServices(AZ::ComponentDescriptor::DependencyArrayType& provided)
    {
        provided.push_back(AZ_CRC_CE("StatsProgressionService"));
    }

    void StatsProgressionSystemComponent::GetIncompatibleServices(AZ::ComponentDescriptor::DependencyArrayType& incompatible)
    {
        incompatible.push_back(AZ_CRC_CE("StatsProgressionService"));
    }

    void StatsProgressionSystemComponent::GetRequiredServices(AZ::ComponentDescriptor::DependencyArrayType& required)
    {
        required.push_back(AZ_CRC_CE("GameCoreService"));
    }

    void StatsProgressionSystemComponent::GetDependentServices(AZ::ComponentDescriptor::DependencyArrayType&)
    {
    }

    void StatsProgressionSystemComponent::Activate()
    {
    }

    void StatsProgressionSystemComponent::Deactivate()
    {
    }
}
