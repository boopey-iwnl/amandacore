#include <MovementPhysics/MovementPhysicsSystemComponent.h>
#include <MovementPhysics/LocalPlayerControllerComponent.h>
#include <MovementPhysics/ThirdPersonCameraComponent.h>

#include <AzCore/Memory/SystemAllocator.h>
#include <AzCore/Module/Module.h>

namespace MovementPhysics
{
    class MovementPhysicsModule final
        : public AZ::Module
    {
    public:
        AZ_RTTI(MovementPhysicsModule, "{4C245893-2BD9-4E6C-9952-4CFD3B57B1C0}", AZ::Module);
        AZ_CLASS_ALLOCATOR(MovementPhysicsModule, AZ::SystemAllocator);

        MovementPhysicsModule()
        {
            m_descriptors.insert(m_descriptors.end(), {
                MovementPhysicsSystemComponent::CreateDescriptor(),
                LocalPlayerControllerComponent::CreateDescriptor(),
                ThirdPersonCameraComponent::CreateDescriptor()
            });
        }

        AZ::ComponentTypeList GetRequiredSystemComponents() const override
        {
            return AZ::ComponentTypeList{ azrtti_typeid<MovementPhysicsSystemComponent>() };
        }
    };
}

AZ_DECLARE_MODULE_CLASS(Gem_MovementPhysics, MovementPhysics::MovementPhysicsModule)
