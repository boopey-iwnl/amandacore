#include <CombatRules/PlayerTargetingComponent.h>

#include <AzCore/Component/TransformBus.h>
#include <AzCore/Math/MathUtils.h>
#include <AzCore/Math/Vector2.h>
#include <AzCore/Math/Vector3.h>
#include <AzCore/Serialization/SerializeContext.h>
#include <AzFramework/Input/Channels/InputChannel.h>
#include <AzFramework/Input/Devices/Keyboard/InputDeviceKeyboard.h>
#include <AzFramework/Input/Devices/Mouse/InputDeviceMouse.h>
#include <GameCore/GameCoreInterface.h>
#include <GameCore/MobProxyGeometry.h>
#include <cfloat>
#ifndef NOMINMAX
#define NOMINMAX
#endif
#include <Windows.h>

namespace CombatRules
{
    namespace
    {
        constexpr const char* HostileMobKind = "hostile_mob";
        constexpr const char* TrainerNpcKind = "trainer_npc";
        constexpr const char* QuestGiverNpcKind = "quest_giver_npc";
        constexpr float FriendlyNpcScreenPickRadiusPixels = 84.0f;
        constexpr float HostileScreenPickRadiusPixels = 42.0f;

        bool GetClientCursorInfo(AZ::Vector2& cursorPosition, AZ::Vector2& viewportSize)
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

            POINT cursorPoint{};
            if (!::GetCursorPos(&cursorPoint) || !::ScreenToClient(activeWindow, &cursorPoint))
            {
                return false;
            }

            viewportSize = AZ::Vector2(
                static_cast<float>(clientRect.right - clientRect.left),
                static_cast<float>(clientRect.bottom - clientRect.top));
            cursorPosition = AZ::Vector2(static_cast<float>(cursorPoint.x), static_cast<float>(cursorPoint.y));
            return viewportSize.GetX() > 1.0f && viewportSize.GetY() > 1.0f;
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

        bool ProjectWorldPointToScreen(
            const GameCore::ClientCameraState& cameraState,
            const AZ::Vector3& worldPoint,
            const AZ::Vector2& viewportSize,
            AZ::Vector2& outScreenPosition)
        {
            if (!cameraState.m_ready || viewportSize.GetX() <= 1.0f || viewportSize.GetY() <= 1.0f)
            {
                return false;
            }

            const AZ::Transform inverseView = cameraState.m_worldTransform.GetInverse();
            const AZ::Vector3 cameraLocal = inverseView.TransformPoint(worldPoint);
            if (cameraLocal.GetY() <= 0.05f)
            {
                return false;
            }

            float aspectRatio = viewportSize.GetX() / viewportSize.GetY();
            if (aspectRatio <= 0.01f)
            {
                aspectRatio = 16.0f / 9.0f;
            }

            const float tanHalfFov = AZStd::tan(AZ::DegToRad(cameraState.m_verticalFovDegrees) * 0.5f);
            if (tanHalfFov <= 0.0f)
            {
                return false;
            }

            const float ndcX = cameraLocal.GetX() / (cameraLocal.GetY() * tanHalfFov * aspectRatio);
            const float ndcY = cameraLocal.GetZ() / (cameraLocal.GetY() * tanHalfFov);
            if (AZ::GetAbs(ndcX) > 1.1f || AZ::GetAbs(ndcY) > 1.1f)
            {
                return false;
            }

            outScreenPosition.SetX(((ndcX + 1.0f) * 0.5f) * viewportSize.GetX());
            outScreenPosition.SetY(((1.0f - ndcY) * 0.5f) * viewportSize.GetY());
            return true;
        }

        AZ::Vector3 BuildCameraRayDirection(
            const GameCore::ClientCameraState& cameraState,
            const AZ::Vector2& cursorPosition,
            const AZ::Vector2& viewportSize)
        {
            const float aspectRatio = viewportSize.GetX() / viewportSize.GetY();
            const float tanHalfFov = AZStd::tan(AZ::DegToRad(cameraState.m_verticalFovDegrees) * 0.5f);
            const float normalizedX = ((cursorPosition.GetX() / viewportSize.GetX()) * 2.0f) - 1.0f;
            const float normalizedY = 1.0f - ((cursorPosition.GetY() / viewportSize.GetY()) * 2.0f);
            const AZ::Vector3 cameraLocalDirection(
                normalizedX * tanHalfFov * aspectRatio,
                1.0f,
                normalizedY * tanHalfFov);
            return cameraState.m_worldTransform.TransformVector(cameraLocalDirection.GetNormalized()).GetNormalized();
        }

        bool RayIntersectsSphere(
            const AZ::Vector3& rayOrigin,
            const AZ::Vector3& rayDirection,
            const AZ::Vector3& sphereCenter,
            float sphereRadius,
            float& outRayDistance)
        {
            const AZ::Vector3 originToCenter = rayOrigin - sphereCenter;
            const float a = rayDirection.Dot(rayDirection);
            const float b = 2.0f * originToCenter.Dot(rayDirection);
            const float c = originToCenter.Dot(originToCenter) - (sphereRadius * sphereRadius);
            const float discriminant = (b * b) - (4.0f * a * c);
            if (discriminant < 0.0f)
            {
                return false;
            }

            const float root = AZStd::sqrt(discriminant);
            const float denominator = 2.0f * a;
            const float t0 = (-b - root) / denominator;
            const float t1 = (-b + root) / denominator;

            float hitDistance = -1.0f;
            if (t0 > 0.0f)
            {
                hitDistance = t0;
            }
            else if (t1 > 0.0f)
            {
                hitDistance = t1;
            }

            if (hitDistance <= 0.0f)
            {
                return false;
            }

            outRayDistance = hitDistance;
            return true;
        }

        AZStd::vector<AZStd::string> CollectOrderedHostileTargetIds(const NetClient::WorldSessionResponse& session)
        {
            AZStd::vector<AZStd::string> targetIds;
            targetIds.reserve(session.m_entities.size());
            for (const auto& entity : session.m_entities)
            {
                if (entity.m_kind != HostileMobKind || !entity.m_alive || !entity.m_targetable)
                {
                    continue;
                }

                targetIds.push_back(entity.m_id);
            }

            return targetIds;
        }

        bool IsClickableTargetKind(const AZStd::string& kind)
        {
            return kind == HostileMobKind || kind == TrainerNpcKind || kind == QuestGiverNpcKind || kind == "player";
        }

        bool HasNpcService(const NetClient::VisibleEntity& entity)
        {
            return !entity.m_services.empty() || entity.m_kind == TrainerNpcKind || entity.m_kind == QuestGiverNpcKind;
        }

        AZStd::string JoinTargetIds(const AZStd::vector<AZStd::string>& targetIds)
        {
            AZStd::string joined;
            for (size_t index = 0; index < targetIds.size(); ++index)
            {
                if (index != 0)
                {
                    joined += ",";
                }
                joined += targetIds[index];
            }
            return joined;
        }
    }

    PlayerTargetingComponent::PlayerTargetingComponent()
        : AzFramework::InputChannelEventListener(AzFramework::InputChannelEventListener::GetPriorityFirst())
    {
    }

    void PlayerTargetingComponent::Reflect(AZ::ReflectContext* context)
    {
        if (auto* serializeContext = azrtti_cast<AZ::SerializeContext*>(context))
        {
            serializeContext->Class<PlayerTargetingComponent, AZ::Component>()
                ->Version(0);
        }
    }

    void PlayerTargetingComponent::GetProvidedServices(AZ::ComponentDescriptor::DependencyArrayType& provided)
    {
        provided.push_back(AZ_CRC_CE("PlayerTargetingService"));
    }

    void PlayerTargetingComponent::GetIncompatibleServices(AZ::ComponentDescriptor::DependencyArrayType& incompatible)
    {
        incompatible.push_back(AZ_CRC_CE("PlayerTargetingService"));
    }

    void PlayerTargetingComponent::GetRequiredServices(AZ::ComponentDescriptor::DependencyArrayType& required)
    {
        required.push_back(AZ_CRC_CE("TransformService"));
    }

    void PlayerTargetingComponent::GetDependentServices(AZ::ComponentDescriptor::DependencyArrayType&)
    {
    }

    void PlayerTargetingComponent::Activate()
    {
        AzFramework::InputChannelEventListener::Connect();
        AZ::TickBus::Handler::BusConnect();
    }

    void PlayerTargetingComponent::Deactivate()
    {
        AZ::TickBus::Handler::BusDisconnect();
        AzFramework::InputChannelEventListener::Disconnect();
    }

    void PlayerTargetingComponent::OnTick(float, AZ::ScriptTimePoint)
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

        if (!m_loggedReady)
        {
            m_loggedReady = true;
            AZ_Printf("amandacore", "client.targeting_ready entity=LocalPlayer");
        }
        if (!m_loggedHelp)
        {
            m_loggedHelp = true;
            AZ_Printf("amandacore", "client.targeting_input_help selectHostile=Tab selectNpcOrHostile=LMB interactNpc=RMB");
        }

        const AZStd::string& currentTargetId = worldState.m_session.m_currentTargetId;
        if (currentTargetId == m_lastLoggedTargetId)
        {
            return;
        }

        if (currentTargetId.empty())
        {
            AZ_Printf("amandacore", "client.target_cleared");
        }
        else
        {
            AZ_Printf("amandacore", "client.target_selected targetId=%s", currentTargetId.c_str());
        }

        m_lastLoggedTargetId = currentTargetId;
    }

    bool PlayerTargetingComponent::OnInputChannelEventFiltered(const AzFramework::InputChannel& inputChannel)
    {
        if (!inputChannel.IsStateBegan())
        {
            return false;
        }

        const auto& channelId = inputChannel.GetInputChannelId();
        if (channelId != AzFramework::InputDeviceKeyboard::Key::EditTab &&
            channelId != AzFramework::InputDeviceMouse::Button::Left &&
            channelId != AzFramework::InputDeviceMouse::Button::Right)
        {
            return false;
        }

        auto* gameCore = GameCore::IGameCoreRequests::Get();
        if (!gameCore)
        {
            return false;
        }

        AZStd::string targetId;
        if (channelId == AzFramework::InputDeviceMouse::Button::Right)
        {
            targetId = FindClickedFriendlyNpcTarget();
            if (targetId.empty())
            {
                return false;
            }

            AZ_Printf("amandacore", "client.npc_right_click_candidate targetId=%s", targetId.c_str());
            return gameCore->InteractWithEntity(targetId);
        }

        if (channelId == AzFramework::InputDeviceKeyboard::Key::EditTab)
        {
            const auto orderedTargetIds = CollectOrderedHostileTargetIds(gameCore->GetClientWorldState().m_session);
            targetId = FindNextHostileTarget();
            if (!targetId.empty())
            {
                AZ_Printf(
                    "amandacore",
                    "client.target_cycle_requested order=%s from=%s to=%s",
                    JoinTargetIds(orderedTargetIds).c_str(),
                    gameCore->GetClientWorldState().m_session.m_currentTargetId.c_str(),
                    targetId.c_str());
            }
        }
        else
        {
            targetId = FindClickedTarget(false);
            if (!targetId.empty())
            {
                AZ_Printf("amandacore", "client.click_target_candidate targetId=%s", targetId.c_str());
            }
        }

        if (targetId.empty())
        {
            return false;
        }

        return gameCore->SetTarget(targetId);
    }

    AZStd::string PlayerTargetingComponent::FindNextHostileTarget() const
    {
        auto* gameCore = GameCore::IGameCoreRequests::Get();
        if (!gameCore)
        {
            return {};
        }

        const auto& worldState = gameCore->GetClientWorldState();
        if (!worldState.m_worldConnected)
        {
            return {};
        }

        AZStd::vector<AZStd::string> targetIds = CollectOrderedHostileTargetIds(worldState.m_session);
        if (targetIds.empty())
        {
            return {};
        }
        auto currentTargetIt = AZStd::find(
            targetIds.begin(),
            targetIds.end(),
            worldState.m_session.m_currentTargetId);
        if (currentTargetIt == targetIds.end())
        {
            return targetIds.front();
        }

        ++currentTargetIt;
        if (currentTargetIt == targetIds.end())
        {
            return targetIds.front();
        }

        return *currentTargetIt;
    }

    AZStd::string PlayerTargetingComponent::FindClickedFriendlyNpcTarget() const
    {
        return FindClickedTarget(true);
    }

    AZStd::string PlayerTargetingComponent::FindClickedTarget(bool friendlyOnly) const
    {
        auto* gameCore = GameCore::IGameCoreRequests::Get();
        if (!gameCore)
        {
            return {};
        }

        const auto& worldState = gameCore->GetClientWorldState();
        const auto& cameraState = gameCore->GetCameraState();
        if (!worldState.m_worldConnected || !cameraState.m_ready)
        {
            return {};
        }

        AZ::Vector2 cursorPosition = AZ::Vector2::CreateZero();
        AZ::Vector2 viewportSize = AZ::Vector2::CreateZero();
        if (!GetClientCursorInfo(cursorPosition, viewportSize))
        {
            return {};
        }

        const AZ::Vector3 rayOrigin = cameraState.m_worldTransform.GetTranslation();
        const AZ::Vector3 rayDirection = BuildCameraRayDirection(cameraState, cursorPosition, viewportSize);
        const float viewportAspectRatio = viewportSize.GetY() > 0.0f
            ? viewportSize.GetX() / viewportSize.GetY()
            : (16.0f / 9.0f);
        float bestHitDistance = FLT_MAX;
        AZStd::string bestTargetId;
        float bestScreenDistanceSq = FLT_MAX;
        AZStd::string bestScreenTargetId;
        for (const auto& entity : worldState.m_session.m_entities)
        {
            if (!IsClickableTargetKind(entity.m_kind) || !entity.m_alive || !entity.m_targetable)
            {
                continue;
            }
            if (friendlyOnly && !HasNpcService(entity))
            {
                continue;
            }

            const bool isFriendlyNpc = HasNpcService(entity);

            const AZ::Vector3 basePosition(
                static_cast<float>(entity.m_x),
                static_cast<float>(entity.m_y),
                static_cast<float>(entity.m_z));
            if (!IsPointInCameraFrustum(
                    cameraState,
                    basePosition + AZ::Vector3(0.0f, 0.0f, GameCore::MobProxyGeometry::BodyHeight),
                    viewportAspectRatio))
            {
                continue;
            }

            float entityHitDistance = FLT_MAX;
            GameCore::MobProxyGeometry::VisitWorldProxySpheres(
                entity.m_id,
                basePosition,
                entity.m_alive,
                [&rayOrigin, &rayDirection, &entityHitDistance, isFriendlyNpc](const AZ::Vector3& sphereCenter, float sphereRadius)
                {
                    float rayDistance = 0.0f;
                    const float effectiveRadius = isFriendlyNpc ? (sphereRadius * 1.45f) : sphereRadius;
                    if (!RayIntersectsSphere(rayOrigin, rayDirection, sphereCenter, effectiveRadius, rayDistance))
                    {
                        return;
                    }

                    if (rayDistance < entityHitDistance)
                    {
                        entityHitDistance = rayDistance;
                    }
                });

            AZ::Vector2 projectedScreenPosition = AZ::Vector2::CreateZero();
            float entityScreenDistanceSq = FLT_MAX;
            static constexpr float SampleHeights[] = {
                0.0f,
                GameCore::MobProxyGeometry::BodyHeight,
                GameCore::MobProxyGeometry::HeadHeight,
            };
            for (float sampleHeight : SampleHeights)
            {
                if (!ProjectWorldPointToScreen(
                        cameraState,
                        basePosition + AZ::Vector3(0.0f, 0.0f, sampleHeight),
                        viewportSize,
                        projectedScreenPosition))
                {
                    continue;
                }

                const float deltaX = projectedScreenPosition.GetX() - cursorPosition.GetX();
                const float deltaY = projectedScreenPosition.GetY() - cursorPosition.GetY();
                const float screenDistanceSq = (deltaX * deltaX) + (deltaY * deltaY);
                if (screenDistanceSq < entityScreenDistanceSq)
                {
                    entityScreenDistanceSq = screenDistanceSq;
                }
            }

            const float maxPickRadius = isFriendlyNpc ? FriendlyNpcScreenPickRadiusPixels : HostileScreenPickRadiusPixels;
            if (entityScreenDistanceSq <= (maxPickRadius * maxPickRadius) &&
                entityScreenDistanceSq < bestScreenDistanceSq)
            {
                bestScreenDistanceSq = entityScreenDistanceSq;
                bestScreenTargetId = entity.m_id;
            }

            if (entityHitDistance >= bestHitDistance)
            {
                continue;
            }

            bestHitDistance = entityHitDistance;
            bestTargetId = entity.m_id;
        }

        if (!bestTargetId.empty())
        {
            return bestTargetId;
        }

        return bestScreenTargetId;
    }
} // namespace CombatRules
