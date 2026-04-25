#include <NpcAi/MobCombatStateComponent.h>

#include <AzCore/Serialization/SerializeContext.h>

namespace NpcAi
{
    void MobCombatStateComponent::Reflect(AZ::ReflectContext* context)
    {
        if (auto* serializeContext = azrtti_cast<AZ::SerializeContext*>(context))
        {
            serializeContext->Class<MobCombatStateComponent, AZ::Component>()
                ->Version(0);
        }
    }

    void MobCombatStateComponent::GetProvidedServices(AZ::ComponentDescriptor::DependencyArrayType& provided)
    {
        provided.push_back(AZ_CRC_CE("MobCombatStateService"));
    }

    void MobCombatStateComponent::GetIncompatibleServices(AZ::ComponentDescriptor::DependencyArrayType& incompatible)
    {
        incompatible.push_back(AZ_CRC_CE("MobCombatStateService"));
    }

    void MobCombatStateComponent::GetRequiredServices(AZ::ComponentDescriptor::DependencyArrayType&)
    {
    }

    void MobCombatStateComponent::GetDependentServices(AZ::ComponentDescriptor::DependencyArrayType&)
    {
    }
} // namespace NpcAi
