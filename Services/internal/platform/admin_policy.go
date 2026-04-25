package platform

import "time"

type AdminPermission Permission

type AdminDecision string

const (
	AdminDecisionAllowed             AdminDecision = "allowed"
	AdminDecisionDenied              AdminDecision = "denied"
	AdminDecisionRequiresApproval    AdminDecision = "requires_approval"
	AdminDecisionApplied             AdminDecision = "applied"
	AdminDecisionRejectedByValidator AdminDecision = "rejected_by_validator"
)

type AdminActor struct {
	AccountID   string            `json:"accountId"`
	DisplayName string            `json:"displayName,omitempty"`
	Roles       []Role            `json:"roles"`
	Permissions []AdminPermission `json:"permissions"`
}

type AdminAction struct {
	ActionID           string          `json:"actionId"`
	Name               string          `json:"name"`
	Actor              AdminActor      `json:"actor"`
	TargetAccountID    string          `json:"targetAccountId,omitempty"`
	TargetCharacterID  string          `json:"targetCharacterId,omitempty"`
	RequiredPermission AdminPermission `json:"requiredPermission,omitempty"`
	Reason             string          `json:"reason,omitempty"`
	RequestedAt        time.Time       `json:"requestedAt"`
	Metadata           map[string]any  `json:"metadata,omitempty"`
}

type AdminAuditEvent struct {
	AuditEventID   string         `json:"auditEventId"`
	Action         AdminAction    `json:"action"`
	Decision       AdminDecision  `json:"decision"`
	DecisionReason string         `json:"decisionReason,omitempty"`
	AppliedAt      time.Time      `json:"appliedAt,omitempty"`
	BeforeSummary  map[string]any `json:"beforeSummary,omitempty"`
	AfterSummary   map[string]any `json:"afterSummary,omitempty"`
}

func NewAdminActor(account Account) AdminActor {
	permissions := PermissionsForRoles(account.Roles)
	adminPermissions := make([]AdminPermission, 0, len(permissions))
	for _, permission := range permissions {
		adminPermissions = append(adminPermissions, AdminPermission(permission))
	}

	return AdminActor{
		AccountID:   account.ID,
		DisplayName: account.Username,
		Roles:       append([]Role{}, account.Roles...),
		Permissions: adminPermissions,
	}
}

func (a AdminActor) HasPermission(required AdminPermission) bool {
	for _, permission := range a.Permissions {
		if permission == required {
			return true
		}
	}
	return false
}
