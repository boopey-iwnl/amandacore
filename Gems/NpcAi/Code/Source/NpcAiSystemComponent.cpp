#include <NpcAi/NpcAiSystemComponent.h>

#include <Atom/RPI.Public/AuxGeom/AuxGeomDraw.h>
#include <Atom/RPI.Public/AuxGeom/AuxGeomFeatureProcessorInterface.h>
#include <Atom/RPI.Public/Scene.h>
#include <AzCore/Component/Entity.h>
#include <AzCore/Math/Color.h>
#include <AzCore/Math/MathUtils.h>
#include <AzCore/Serialization/SerializeContext.h>
#include <AzCore/std/containers/vector.h>
#include <AzCore/std/containers/unordered_set.h>
#include <AzFramework/Components/TransformComponent.h>
#include <AzFramework/Entity/GameEntityContextBus.h>
#include <AzCore/Component/TransformBus.h>
#include <GameCore/GameCoreInterface.h>
#include <GameCore/MobProxyGeometry.h>
#include <NpcAi/MobCombatStateComponent.h>
#include <NpcAi/MobPresentationComponent.h>
#ifndef NOMINMAX
#define NOMINMAX
#endif
#include <Windows.h>

namespace NpcAi
{
    namespace
    {
        constexpr float PlayerTargetRange = 28.0f;
        constexpr float CommandPointX = 13.0f;
        constexpr float CommandPointY = 10.0f;
        constexpr float EncounterAnchorX = 322.0f;
        constexpr float EncounterAnchorY = 174.0f;
        constexpr float RingSphereRadius = 0.18f;
        constexpr int RingSegments = 18;
        constexpr int MarkerSteps = 5;
        constexpr float TwoPi = 6.28318530717958647692f;
        constexpr const char* HostileMobKind = "hostile_mob";
        constexpr const char* TrainerNpcKind = "trainer_npc";
        constexpr const char* QuestGiverNpcKind = "quest_giver_npc";

        float Distance2D(const AZ::Vector3& left, const AZ::Vector3& right)
        {
            const float deltaX = right.GetX() - left.GetX();
            const float deltaY = right.GetY() - left.GetY();
            return AZStd::sqrt((deltaX * deltaX) + (deltaY * deltaY));
        }

        void DrawRing(AZ::RPI::AuxGeomDrawPtr auxGeom, const AZ::Vector3& center, float ringRadius, const AZ::Color& color)
        {
            for (int segmentIndex = 0; segmentIndex < RingSegments; ++segmentIndex)
            {
                const float angleRadians = (TwoPi / static_cast<float>(RingSegments)) * static_cast<float>(segmentIndex);
                const AZ::Vector3 offset(
                    AZStd::cos(angleRadians) * ringRadius,
                    AZStd::sin(angleRadians) * ringRadius,
                    0.0f);
                auxGeom->DrawSphere(center + offset, RingSphereRadius, color);
            }
        }

        void DrawMarkerColumn(AZ::RPI::AuxGeomDrawPtr auxGeom, const AZ::Vector3& basePosition, const AZ::Color& color)
        {
            for (int markerIndex = 0; markerIndex < MarkerSteps; ++markerIndex)
            {
                const float heightOffset = 0.9f + (0.35f * static_cast<float>(markerIndex));
                auxGeom->DrawSphere(basePosition + AZ::Vector3(0.0f, 0.0f, heightOffset), RingSphereRadius, color);
            }
        }

        void DrawPathNode(AZ::RPI::AuxGeomDrawPtr auxGeom, float x, float y, const AZ::Color& color)
        {
            auxGeom->DrawSphere(AZ::Vector3(x, y, 0.22f), 0.22f, color);
        }

        bool GetClientViewportAspectRatio(float& outAspectRatio)
        {
            const HWND activeWindow = ::GetForegroundWindow();
            if (!activeWindow)
            {
                return false;
            }

            RECT clientRect{};
            if (!::GetClientRect(activeWindow, &clientRect))
            {
                return false;
            }

            const float width = static_cast<float>(clientRect.right - clientRect.left);
            const float height = static_cast<float>(clientRect.bottom - clientRect.top);
            if (width <= 1.0f || height <= 1.0f)
            {
                return false;
            }

            outAspectRatio = width / height;
            return true;
        }

        bool IsPointInCameraFrustum(
            const GameCore::ClientCameraState& cameraState,
            const AZ::Vector3& worldPoint,
            float viewportAspectRatio)
        {
            if (!cameraState.m_ready)
            {
                return false;
            }

            const AZ::Transform inverseView = cameraState.m_worldTransform.GetInverse();
            const AZ::Vector3 cameraLocal = inverseView.TransformPoint(worldPoint);
            if (cameraLocal.GetY() <= 0.05f)
            {
                return false;
            }

            const float tanHalfFov = AZStd::tan(AZ::DegToRad(cameraState.m_verticalFovDegrees) * 0.5f);
            if (tanHalfFov <= 0.0f)
            {
                return false;
            }

            if (viewportAspectRatio <= 0.01f)
            {
                viewportAspectRatio = 16.0f / 9.0f;
            }

            const float ndcX = cameraLocal.GetX() / (cameraLocal.GetY() * tanHalfFov * viewportAspectRatio);
            const float ndcY = cameraLocal.GetZ() / (cameraLocal.GetY() * tanHalfFov);
            return AZ::GetAbs(ndcX) <= 0.95f && AZ::GetAbs(ndcY) <= 0.95f;
        }
    }

    void NpcAiSystemComponent::Reflect(AZ::ReflectContext* context)
    {
        if (auto* serializeContext = azrtti_cast<AZ::SerializeContext*>(context))
        {
            serializeContext->Class<NpcAiSystemComponent, AZ::Component>()
                ->Version(0);
        }
    }

    void NpcAiSystemComponent::GetProvidedServices(AZ::ComponentDescriptor::DependencyArrayType& provided)
    {
        provided.push_back(AZ_CRC_CE("NpcAiService"));
    }

    void NpcAiSystemComponent::GetIncompatibleServices(AZ::ComponentDescriptor::DependencyArrayType& incompatible)
    {
        incompatible.push_back(AZ_CRC_CE("NpcAiService"));
    }

    void NpcAiSystemComponent::GetRequiredServices(AZ::ComponentDescriptor::DependencyArrayType& required)
    {
        required.push_back(AZ_CRC_CE("GameCoreService"));
        required.push_back(AZ_CRC_CE("CombatRulesService"));
    }

    void NpcAiSystemComponent::GetDependentServices(AZ::ComponentDescriptor::DependencyArrayType&)
    {
    }

    void NpcAiSystemComponent::Activate()
    {
        m_loggedEncounterVisible = false;
        m_loggedEncounterValidationPassed = false;
        m_loggedZoneLandmarksReady = false;
        m_loggedTrainerNpcVisible = false;
        m_loggedQuestGiverNpcVisible = false;
        AZ::TickBus::Handler::BusConnect();
    }

    void NpcAiSystemComponent::Deactivate()
    {
        AZ::TickBus::Handler::BusDisconnect();
        for (auto& [mobId, proxy] : m_mobProxies)
        {
            if (proxy.m_entity)
            {
                proxy.m_entity->Deactivate();
                delete proxy.m_entity;
            }
        }
        m_mobProxies.clear();
    }

    void NpcAiSystemComponent::OnTick(float, AZ::ScriptTimePoint)
    {
        auto* gameCore = GameCore::IGameCoreRequests::Get();
        if (!gameCore)
        {
            return;
        }

        const auto& worldState = gameCore->GetClientWorldState();
        if (!worldState.m_worldConnected)
        {
            return;
        }

        const AZ::Vector3 playerPosition(
            static_cast<float>(worldState.m_session.m_position.m_x),
            static_cast<float>(worldState.m_session.m_position.m_y),
            static_cast<float>(worldState.m_session.m_position.m_z));
        const auto& cameraState = gameCore->GetCameraState();
        float viewportAspectRatio = 16.0f / 9.0f;
        GetClientViewportAspectRatio(viewportAspectRatio);
        AZStd::unordered_set<AZStd::string> seenMobIds;
        AZStd::vector<AZStd::string> visibleMobIds;
        AZStd::vector<AZ::Vector3> visibleMobPositions;
        bool targetsReachable = true;
        bool targetsInFrustum = true;
        for (const auto& entity : worldState.m_session.m_entities)
        {
            const bool isHostileMob = entity.m_kind == HostileMobKind;
            const bool isTrainerNpc = entity.m_kind == TrainerNpcKind;
            const bool isQuestGiverNpc = entity.m_kind == QuestGiverNpcKind;
            if (!isHostileMob && !isTrainerNpc && !isQuestGiverNpc)
            {
                continue;
            }

            seenMobIds.insert(entity.m_id);
            const AZ::Vector3 entityPosition(
                static_cast<float>(entity.m_x),
                static_cast<float>(entity.m_y),
                static_cast<float>(entity.m_z));
            if (isHostileMob)
            {
                visibleMobIds.push_back(entity.m_id);
                visibleMobPositions.push_back(entityPosition);
                targetsReachable = targetsReachable &&
                    entity.m_alive &&
                    entity.m_targetable &&
                    Distance2D(playerPosition, entityPosition) <= PlayerTargetRange;
                targetsInFrustum = targetsInFrustum &&
                    IsPointInCameraFrustum(
                        cameraState,
                        entityPosition + AZ::Vector3(0.0f, 0.0f, GameCore::MobProxyGeometry::BodyHeight),
                        viewportAspectRatio);
            }
            auto& proxyState = m_mobProxies[entity.m_id];
            if (!proxyState.m_entity)
            {
                AzFramework::GameEntityContextRequestBus::BroadcastResult(
                    proxyState.m_entity,
                    &AzFramework::GameEntityContextRequestBus::Events::CreateGameEntity,
                    entity.m_displayName.c_str());
                if (!proxyState.m_entity)
                {
                    AZ_Warning("amandacore", false, "Unable to create mob proxy entity for %s", entity.m_id.c_str());
                    continue;
                }

                proxyState.m_entity->CreateComponent<AzFramework::TransformComponent>();
                proxyState.m_entity->CreateComponent<MobCombatStateComponent>();
                proxyState.m_entity->CreateComponent<MobPresentationComponent>();
                proxyState.m_entity->Init();
                AzFramework::GameEntityContextRequestBus::Broadcast(
                    &AzFramework::GameEntityContextRequestBus::Events::AddGameEntity,
                    proxyState.m_entity);
                AzFramework::GameEntityContextRequestBus::Broadcast(
                    &AzFramework::GameEntityContextRequestBus::Events::ActivateGameEntity,
                    proxyState.m_entity->GetId());
                AZ_Printf(
                    "amandacore",
                    isTrainerNpc
                        ? "client.trainer_npc_proxy_spawned trainerId=%s displayName=%s"
                        : (isQuestGiverNpc
                            ? "client.quest_giver_npc_proxy_spawned questGiverId=%s displayName=%s"
                            : "client.mob_proxy_spawned mobId=%s displayName=%s"),
                    entity.m_id.c_str(),
                    entity.m_displayName.c_str());
            }

            AZ::TransformBus::Event(
                proxyState.m_entity->GetId(),
                &AZ::TransformBus::Events::SetWorldTranslation,
                AZ::Vector3(
                    static_cast<float>(entity.m_x),
                    static_cast<float>(entity.m_y),
                    static_cast<float>(entity.m_z)));

            if (auto* combatState = proxyState.m_entity->FindComponent<MobCombatStateComponent>())
            {
                if (isHostileMob && proxyState.m_lastAlive && !entity.m_alive)
                {
                    AZ_Printf("amandacore", "client.mob_death_observed mobId=%s", entity.m_id.c_str());
                }
                if (isHostileMob && !proxyState.m_lastAlive && entity.m_alive)
                {
                    AZ_Printf("amandacore", "client.mob_respawn_observed mobId=%s", entity.m_id.c_str());
                }

                combatState->SetMobId(entity.m_id);
                combatState->SetDisplayName(entity.m_displayName);
                combatState->SetHealth(entity.m_health);
                combatState->SetMaxHealth(entity.m_maxHealth);
                combatState->SetAlive(entity.m_alive);
                combatState->SetTargetable(entity.m_targetable);
                combatState->SetAiState(entity.m_aiState);
            }

            proxyState.m_lastAlive = entity.m_alive;

            if (isTrainerNpc && !m_loggedTrainerNpcVisible)
            {
                AZ_Printf(
                    "amandacore",
                    "client.trainer_npc_visible trainerId=%s displayName=%s position=(%.1f,%.1f,%.1f) interact=right_click",
                    entity.m_id.c_str(),
                    entity.m_displayName.c_str(),
                    entityPosition.GetX(),
                    entityPosition.GetY(),
                    entityPosition.GetZ());
                m_loggedTrainerNpcVisible = true;
            }
            if (isQuestGiverNpc && !m_loggedQuestGiverNpcVisible)
            {
                AZ_Printf(
                    "amandacore",
                    "client.quest_giver_npc_visible questGiverId=%s displayName=%s position=(%.1f,%.1f,%.1f) interact=right_click",
                    entity.m_id.c_str(),
                    entity.m_displayName.c_str(),
                    entityPosition.GetX(),
                    entityPosition.GetY(),
                    entityPosition.GetZ());
                m_loggedQuestGiverNpcVisible = true;
            }
        }

        if (!m_loggedEncounterVisible && visibleMobIds.size() >= 3)
        {
            AZ_Printf(
                "amandacore",
                "client.visible_encounter_ready mobCount=%zu mobIds=%s,%s,%s positions=(%.1f,%.1f,%.1f);(%.1f,%.1f,%.1f);(%.1f,%.1f,%.1f)",
                visibleMobIds.size(),
                visibleMobIds[0].c_str(),
                visibleMobIds[1].c_str(),
                visibleMobIds[2].c_str(),
                visibleMobPositions[0].GetX(),
                visibleMobPositions[0].GetY(),
                visibleMobPositions[0].GetZ(),
                visibleMobPositions[1].GetX(),
                visibleMobPositions[1].GetY(),
                visibleMobPositions[1].GetZ(),
                visibleMobPositions[2].GetX(),
                visibleMobPositions[2].GetY(),
                visibleMobPositions[2].GetZ());
            m_loggedEncounterVisible = true;
        }

        if (!m_loggedEncounterValidationPassed &&
            cameraState.m_ready &&
            visibleMobIds.size() >= 3 &&
            targetsReachable &&
            targetsInFrustum)
        {
            AZ_Printf(
                "amandacore",
                "client.encounter_validation_passed mobCount=%zu targetsReachable=true",
                visibleMobIds.size());
            m_loggedEncounterValidationPassed = true;
        }

        AZ::RPI::Scene* scene = nullptr;
        for (const auto& [_, proxy] : m_mobProxies)
        {
            if (!proxy.m_entity)
            {
                continue;
            }

            scene = AZ::RPI::Scene::GetSceneForEntityId(proxy.m_entity->GetId());
            if (scene)
            {
                break;
            }
        }
        if (scene)
        {
            auto auxGeom = AZ::RPI::AuxGeomFeatureProcessorInterface::GetDrawQueueForScene(scene);
            if (auxGeom)
            {
                if (!m_loggedZoneLandmarksReady)
                {
                    AZ_Printf(
                        "amandacore",
                        "client.zone_landmarks_ready zone=stonewake_vale commandPoint=(%.1f,%.1f) encounter=(%.1f,%.1f)",
                        CommandPointX,
                        CommandPointY,
                        EncounterAnchorX,
                        EncounterAnchorY);
                    m_loggedZoneLandmarksReady = true;
                }

                const bool questNotStarted = worldState.m_session.m_quest.m_state == "not_started";
                const bool questActive = worldState.m_session.m_quest.m_state == "active";
                const bool questComplete = worldState.m_session.m_quest.m_state == "reward_granted";
                const AZ::Color routeColor = questComplete
                    ? AZ::Color(0.30f, 0.55f, 0.30f, 1.0f)
                    : AZ::Color(0.90f, 0.65f, 0.18f, 1.0f);
                const AZ::Color encounterColor = questActive || questNotStarted
                    ? AZ::Color(0.95f, 0.32f, 0.18f, 1.0f)
                    : AZ::Color(0.35f, 0.55f, 0.35f, 1.0f);

                DrawPathNode(auxGeom, 22.0f, 14.0f, routeColor);
                DrawPathNode(auxGeom, 52.0f, 26.0f, routeColor);
                DrawPathNode(auxGeom, 84.0f, 36.0f, routeColor);
                DrawPathNode(auxGeom, 134.0f, 64.0f, routeColor);
                DrawPathNode(auxGeom, 184.0f, 96.0f, routeColor);
                DrawPathNode(auxGeom, 232.0f, 118.0f, routeColor);
                DrawPathNode(auxGeom, 322.0f, 174.0f, routeColor);
                DrawPathNode(auxGeom, 420.0f, 224.0f, routeColor);
                DrawPathNode(auxGeom, 438.0f, 246.0f, routeColor);

                DrawRing(auxGeom, AZ::Vector3(EncounterAnchorX, EncounterAnchorY, 0.08f), 7.8f, encounterColor);
                DrawMarkerColumn(auxGeom, AZ::Vector3(EncounterAnchorX, EncounterAnchorY, 0.12f), encounterColor);
            }
        }

        AZStd::vector<AZStd::string> staleMobIds;
        staleMobIds.reserve(m_mobProxies.size());
        for (const auto& [mobId, _] : m_mobProxies)
        {
            if (seenMobIds.find(mobId) == seenMobIds.end())
            {
                staleMobIds.push_back(mobId);
            }
        }
        for (const auto& mobId : staleMobIds)
        {
            DestroyProxy(mobId);
        }
    }

    void NpcAiSystemComponent::DestroyProxy(const AZStd::string& mobId)
    {
        auto proxyIt = m_mobProxies.find(mobId);
        if (proxyIt == m_mobProxies.end())
        {
            return;
        }
        if (proxyIt->second.m_entity)
        {
            AzFramework::GameEntityContextRequestBus::Broadcast(
                &AzFramework::GameEntityContextRequestBus::Events::DestroyGameEntity,
                proxyIt->second.m_entity->GetId());
        }
        m_mobProxies.erase(proxyIt);
    }
}
