package main

import (
	"log"
	"net/http"

	"amandacore/services/internal/config"
	"amandacore/services/internal/httpapi"
	"amandacore/services/internal/store"
	"amandacore/services/internal/worlds"
)

func main() {
	cfg := config.Load("world-service", "8085")
	fileStore, err := store.NewFileStore(cfg.StorePath, cfg.BuildID, cfg.WorldEndpoint)
	if err != nil {
		log.Fatal(err)
	}

	if err := fileStore.EnsureAdminSeed(cfg.AdminSeedUsername, cfg.AdminSeedPassword); err != nil {
		log.Fatal(err)
	}

	mux := http.NewServeMux()
	worlds.RegisterRoutes(mux, fileStore)
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("ok"))
	})

	log.Printf("%s listening on :%s", cfg.ServiceName, cfg.Port)
	log.Fatal(http.ListenAndServe(":"+cfg.Port, httpapi.WithCORS(mux)))
}
