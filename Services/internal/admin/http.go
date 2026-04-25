package admin

import (
	"net/http"

	"amandacore/services/internal/httpapi"
	"amandacore/services/internal/platform"
	"amandacore/services/internal/store"
)

type banRequest struct {
	AccountID string `json:"accountId"`
	Banned    bool   `json:"banned"`
}

type roleRequest struct {
	AccountID string        `json:"accountId"`
	Role      platform.Role `json:"role"`
}

func RegisterRoutes(mux *http.ServeMux, fileStore *store.FileStore) {
	mux.Handle("GET /v1/admin/accounts", httpapi.RequireRole(fileStore, platform.RoleAdministrator, func(w http.ResponseWriter, r *http.Request, session *platform.Session) {
		accounts, err := fileStore.ListAccounts()
		if err != nil {
			httpapi.Error(w, http.StatusInternalServerError, "account_list_failed", err.Error())
			return
		}

		httpapi.WriteJSON(w, http.StatusOK, map[string]any{"accounts": accounts})
	}))

	mux.Handle("POST /v1/admin/accounts/ban", httpapi.RequireRole(fileStore, platform.RoleAdministrator, func(w http.ResponseWriter, r *http.Request, session *platform.Session) {
		var request banRequest
		if err := httpapi.DecodeJSON(r, &request); err != nil {
			httpapi.Error(w, http.StatusBadRequest, "invalid_json", err.Error())
			return
		}

		if err := fileStore.SetAccountBanned(request.AccountID, request.Banned); err != nil {
			httpapi.Error(w, http.StatusBadRequest, "ban_failed", err.Error())
			return
		}

		httpapi.WriteJSON(w, http.StatusOK, map[string]bool{"ok": true})
	}))

	mux.Handle("POST /v1/admin/accounts/role", httpapi.RequireRole(fileStore, platform.RoleAdministrator, func(w http.ResponseWriter, r *http.Request, session *platform.Session) {
		var request roleRequest
		if err := httpapi.DecodeJSON(r, &request); err != nil {
			httpapi.Error(w, http.StatusBadRequest, "invalid_json", err.Error())
			return
		}

		if err := fileStore.SetAccountRole(request.AccountID, request.Role); err != nil {
			httpapi.Error(w, http.StatusBadRequest, "role_failed", err.Error())
			return
		}

		httpapi.WriteJSON(w, http.StatusOK, map[string]bool{"ok": true})
	}))
}
