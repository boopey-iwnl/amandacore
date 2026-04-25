#include <GameCore/GameCoreSystemComponent.h>

#include <AzCore/Memory/SystemAllocator.h>
#include <AzCore/Module/Module.h>

namespace GameCore
{
    class GameCoreModule final
        : public AZ::Module
    {
    public:
        AZ_RTTI(GameCoreModule, "{54D8E438-6C59-447D-A994-EF0BAA48B2E4}", AZ::Module);
        AZ_CLASS_ALLOCATOR(GameCoreModule, AZ::SystemAllocator);

        GameCoreModule()
        {
            m_descriptors.insert(m_descriptors.end(), {
                GameCoreSystemComponent::CreateDescriptor()
            });
        }

        AZ::ComponentTypeList GetRequiredSystemComponents() const override
        {
            return AZ::ComponentTypeList{ azrtti_typeid<GameCoreSystemComponent>() };
        }
    };
}

AZ_DECLARE_MODULE_CLASS(Gem_GameCore, GameCore::GameCoreModule)
