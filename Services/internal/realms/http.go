package realms

import (
	"net/http"

	"amandacore/services/internal/httpapi"
	"amandacore/services/internal/store"
)

func RegisterRoutes(mux *http.ServeMux, fileStore *store.FileStore) {
	mux.HandleFunc("GET /v1/realms", func(w http.ResponseWriter, r *http.Request) {
		realms, err := fileStore.ListRealms()
		if err != nil {
			httpapi.Error(w, http.StatusInternalServerError, "realm_list_failed", err.Error())
			return
		}

		httpapi.WriteJSON(w, http.StatusOK, map[string]any{"realms": realms})
	})

	mux.HandleFunc("GET /v1/patch/manifest", func(w http.ResponseWriter, r *http.Request) {
		httpapi.WriteJSON(w, http.StatusOK, fileStore.GetBuildManifest())
	})
}
