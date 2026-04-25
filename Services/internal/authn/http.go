package authn

import (
	"net/http"

	"amandacore/services/internal/httpapi"
	"amandacore/services/internal/observability"
	"amandacore/services/internal/platform"
	"amandacore/services/internal/store"
)

type registerRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type refreshRequest struct {
	RefreshToken string `json:"refreshToken"`
}

type changePasswordRequest struct {
	CurrentPassword string `json:"currentPassword"`
	NewPassword     string `json:"newPassword"`
}

type recoverPasswordRequest struct {
	Username string `json:"username"`
}

func RegisterRoutes(mux *http.ServeMux, fileStore *store.FileStore) {
	mux.HandleFunc("POST /v1/accounts/register", func(w http.ResponseWriter, r *http.Request) {
		var request registerRequest
		if err := httpapi.DecodeJSON(r, &request); err != nil {
			httpapi.Error(w, http.StatusBadRequest, "invalid_json", err.Error())
			return
		}

		account, err := fileStore.RegisterAccount(request.Username, request.Password)
		if err != nil {
			httpapi.Error(w, http.StatusConflict, "register_failed", err.Error())
			return
		}

		observability.LogEvent("auth-service", "account.registered", map[string]any{
			"accountId": account.ID,
			"username":  account.Username,
		})

		httpapi.WriteJSON(w, http.StatusCreated, map[string]any{
			"accountId": account.ID,
			"username":  account.Username,
		})
	})

	mux.HandleFunc("POST /v1/auth/login", func(w http.ResponseWriter, r *http.Request) {
		var request loginRequest
		if err := httpapi.DecodeJSON(r, &request); err != nil {
			httpapi.Error(w, http.StatusBadRequest, "invalid_json", err.Error())
			return
		}

		account, err := fileStore.Authenticate(request.Username, request.Password)
		if err != nil {
			httpapi.Error(w, http.StatusUnauthorized, "login_failed", err.Error())
			return
		}

		session, err := fileStore.CreateSession(account.ID)
		if err != nil {
			httpapi.Error(w, http.StatusInternalServerError, "session_failed", err.Error())
			return
		}

		observability.LogEvent("auth-service", "auth.session_issued", map[string]any{
			"accountId": account.ID,
			"sessionId": session.ID,
			"username":  account.Username,
			"roles":     account.Roles,
		})

		httpapi.WriteJSON(w, http.StatusOK, map[string]any{
			"accessToken":  session.AccessToken,
			"refreshToken": session.RefreshToken,
			"accountId":    account.ID,
			"roles":        account.Roles,
		})
	})

	mux.HandleFunc("POST /v1/auth/refresh", func(w http.ResponseWriter, r *http.Request) {
		var request refreshRequest
		if err := httpapi.DecodeJSON(r, &request); err != nil {
			httpapi.Error(w, http.StatusBadRequest, "invalid_json", err.Error())
			return
		}

		session, err := fileStore.RefreshSession(request.RefreshToken)
		if err != nil {
			httpapi.Error(w, http.StatusUnauthorized, "refresh_failed", err.Error())
			return
		}

		observability.LogEvent("auth-service", "auth.session_refreshed", map[string]any{
			"sessionId": session.ID,
			"accountId": session.AccountID,
		})

		httpapi.WriteJSON(w, http.StatusOK, map[string]any{
			"accessToken":  session.AccessToken,
			"refreshToken": session.RefreshToken,
		})
	})

	mux.Handle("POST /v1/auth/logout", httpapi.RequireSession(fileStore, func(w http.ResponseWriter, r *http.Request, session *platform.Session) {
		_ = fileStore.RevokeSession(session.AccessToken)
		httpapi.WriteJSON(w, http.StatusOK, map[string]bool{"ok": true})
	}))

	mux.Handle("POST /v1/auth/password/change", httpapi.RequireSession(fileStore, func(w http.ResponseWriter, r *http.Request, session *platform.Session) {
		var request changePasswordRequest
		if err := httpapi.DecodeJSON(r, &request); err != nil {
			httpapi.Error(w, http.StatusBadRequest, "invalid_json", err.Error())
			return
		}

		if err := fileStore.ChangePassword(session.AccountID, request.CurrentPassword, request.NewPassword); err != nil {
			httpapi.Error(w, http.StatusUnauthorized, "change_password_failed", err.Error())
			return
		}

		httpapi.WriteJSON(w, http.StatusOK, map[string]bool{"ok": true})
	}))

	mux.HandleFunc("POST /v1/auth/password/recover", func(w http.ResponseWriter, r *http.Request) {
		var request recoverPasswordRequest
		if err := httpapi.DecodeJSON(r, &request); err != nil {
			httpapi.Error(w, http.StatusBadRequest, "invalid_json", err.Error())
			return
		}

		ticket, err := fileStore.StartPasswordReset(request.Username)
		if err != nil {
			httpapi.Error(w, http.StatusNotFound, "recover_failed", err.Error())
			return
		}

		httpapi.WriteJSON(w, http.StatusAccepted, map[string]any{
			"resetTicketId": ticket.ID,
			"expiresAt":     ticket.ExpiresAt,
		})
	})
}
