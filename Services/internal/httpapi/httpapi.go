package httpapi

import (
	"encoding/json"
	"net/http"
	"strings"

	"amandacore/services/internal/platform"
	"amandacore/services/internal/store"
)

func WriteJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func Error(w http.ResponseWriter, status int, code string, message string) {
	WriteJSON(w, status, map[string]string{
		"error":   code,
		"message": message,
	})
}

func DecodeJSON(r *http.Request, target any) error {
	defer r.Body.Close()
	return json.NewDecoder(r.Body).Decode(target)
}

func ReadBearerToken(r *http.Request) string {
	header := r.Header.Get("Authorization")
	if header == "" {
		return ""
	}

	const prefix = "Bearer "
	if !strings.HasPrefix(header, prefix) {
		return ""
	}

	return strings.TrimSpace(strings.TrimPrefix(header, prefix))
}

func RequireSession(store *store.FileStore, next func(http.ResponseWriter, *http.Request, *platform.Session)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := ReadBearerToken(r)
		if token == "" {
			Error(w, http.StatusUnauthorized, "missing_token", "A bearer token is required.")
			return
		}

		session, err := store.ValidateAccessToken(token)
		if err != nil {
			Error(w, http.StatusUnauthorized, "invalid_token", err.Error())
			return
		}

		next(w, r, session)
	}
}

func RequireRole(store *store.FileStore, required platform.Role, next func(http.ResponseWriter, *http.Request, *platform.Session)) http.HandlerFunc {
	return RequireSession(store, func(w http.ResponseWriter, r *http.Request, session *platform.Session) {
		account, err := store.GetAccountByID(session.AccountID)
		if err != nil {
			Error(w, http.StatusUnauthorized, "invalid_account", err.Error())
			return
		}

		for _, role := range account.Roles {
			if role == required || role == platform.RoleAdministrator {
				next(w, r, session)
				return
			}
		}

		Error(w, http.StatusForbidden, "missing_role", "The current session does not have the required role.")
	})
}

func RequirePermission(store *store.FileStore, required platform.Permission, next func(http.ResponseWriter, *http.Request, *platform.Session, *platform.Account)) http.HandlerFunc {
	return RequireSession(store, func(w http.ResponseWriter, r *http.Request, session *platform.Session) {
		account, err := store.GetAccountByID(session.AccountID)
		if err != nil {
			Error(w, http.StatusUnauthorized, "invalid_account", err.Error())
			return
		}

		if !platform.HasPermission(account.Roles, required) {
			Error(w, http.StatusForbidden, "missing_permission", "The current session does not have the required permission.")
			return
		}

		next(w, r, session, account)
	})
}

func WithCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}
