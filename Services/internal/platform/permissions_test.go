package platform

import "testing"

func TestAdminPermissionMatrix(t *testing.T) {
	tests := []struct {
		name    string
		roles   []Role
		allowed []Permission
		denied  []Permission
	}{
		{
			name:  "player",
			roles: []Role{RolePlayer},
			denied: []Permission{
				PermissionViewAccount,
				PermissionViewCharacter,
				PermissionGrantItem,
				PermissionGrantCurrency,
				PermissionManageRoles,
			},
		},
		{
			name:  "support",
			roles: []Role{RolePlayer, RoleSupport},
			allowed: []Permission{
				PermissionViewAccount,
				PermissionViewCharacter,
				PermissionViewInventory,
				PermissionViewEconomy,
				PermissionManageSupport,
			},
			denied: []Permission{
				PermissionGrantItem,
				PermissionGrantCurrency,
				PermissionTeleportCharacter,
				PermissionSuspendAccount,
				PermissionManageRoles,
			},
		},
		{
			name:  "gm",
			roles: []Role{RolePlayer, RoleGM},
			allowed: []Permission{
				PermissionViewAccount,
				PermissionViewCharacter,
				PermissionRepairCharacter,
				PermissionTeleportCharacter,
				PermissionModifyQuestState,
				PermissionModerateChat,
			},
			denied: []Permission{
				PermissionGrantItem,
				PermissionGrantCurrency,
				PermissionSuspendAccount,
				PermissionManageRoles,
			},
		},
		{
			name:  "admin",
			roles: []Role{RolePlayer, RoleAdmin},
			allowed: []Permission{
				PermissionViewAccount,
				PermissionViewCharacter,
				PermissionGrantItem,
				PermissionGrantCurrency,
				PermissionSuspendAccount,
				PermissionViewAuditLog,
				PermissionManageRoles,
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			for _, permission := range test.allowed {
				if !HasPermission(test.roles, permission) {
					t.Fatalf("expected %s to allow %s", test.roles, permission)
				}
			}
			for _, permission := range test.denied {
				if HasPermission(test.roles, permission) {
					t.Fatalf("expected %s to deny %s", test.roles, permission)
				}
			}
		})
	}
}
