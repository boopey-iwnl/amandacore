package accounts

import (
	"net/http"

	"amandacore/services/internal/httpapi"
	"amandacore/services/internal/platform"
	"amandacore/services/internal/store"
)

func RegisterRoutes(mux *http.ServeMux, fileStore *store.FileStore) {
	mux.Handle("GET /v1/account/me", httpapi.RequireSession(fileStore, func(w http.ResponseWriter, r *http.Request, session *platform.Session) {
		account, err := fileStore.GetAccountByID(session.AccountID)
		if err != nil {
			httpapi.Error(w, http.StatusNotFound, "account_missing", err.Error())
			return
		}

		httpapi.WriteJSON(w, http.StatusOK, map[string]any{
			"id":       account.ID,
			"username": account.Username,
			"roles":    account.Roles,
			"banned":   account.Banned,
		})
	}))
}
