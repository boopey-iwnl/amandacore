#include <NetServer/NetServerSystemComponent.h>

#include <AzCore/Memory/SystemAllocator.h>
#include <AzCore/Module/Module.h>

namespace NetServer
{
    class NetServerModule final
        : public AZ::Module
    {
    public:
        AZ_RTTI(NetServerModule, "{58A96EB5-C10F-4894-B0D4-C34228541906}", AZ::Module);
        AZ_CLASS_ALLOCATOR(NetServerModule, AZ::SystemAllocator);

        NetServerModule()
        {
            m_descriptors.insert(m_descriptors.end(), {
                NetServerSystemComponent::CreateDescriptor()
            });
        }

        AZ::ComponentTypeList GetRequiredSystemComponents() const override
        {
            return AZ::ComponentTypeList{ azrtti_typeid<NetServerSystemComponent>() };
        }
    };
}

AZ_DECLARE_MODULE_CLASS(Gem_NetServer, NetServer::NetServerModule)
