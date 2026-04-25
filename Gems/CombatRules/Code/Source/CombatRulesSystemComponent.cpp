#include <CombatRules/CombatRulesSystemComponent.h>

#include <AzCore/Serialization/SerializeContext.h>

namespace CombatRules
{
    void CombatRulesSystemComponent::Reflect(AZ::ReflectContext* context)
    {
        if (auto* serializeContext = azrtti_cast<AZ::SerializeContext*>(context))
        {
            serializeContext->Class<CombatRulesSystemComponent, AZ::Component>()
                ->Version(0);
        }
    }

    void CombatRulesSystemComponent::GetProvidedServices(AZ::ComponentDescriptor::DependencyArrayType& provided)
    {
        provided.push_back(AZ_CRC_CE("CombatRulesService"));
    }

    void CombatRulesSystemComponent::GetIncompatibleServices(AZ::ComponentDescriptor::DependencyArrayType& incompatible)
    {
        incompatible.push_back(AZ_CRC_CE("CombatRulesService"));
    }

    void CombatRulesSystemComponent::GetRequiredServices(AZ::ComponentDescriptor::DependencyArrayType& required)
    {
        required.push_back(AZ_CRC_CE("GameCoreService"));
    }

    void CombatRulesSystemComponent::GetDependentServices(AZ::ComponentDescriptor::DependencyArrayType&)
    {
    }

    void CombatRulesSystemComponent::Activate()
    {
    }

    void CombatRulesSystemComponent::Deactivate()
    {
    }
}
