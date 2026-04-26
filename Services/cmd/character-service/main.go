package main

import (
	"log"
	"net/http"

	"amandacore/services/internal/characters"
	"amandacore/services/internal/config"
	"amandacore/services/internal/httpapi"
	"amandacore/services/internal/store"
)

func main() {
	cfg := config.Load("character-service", "8084")
	fileStore, err := store.NewFileStore(cfg.StorePath, cfg.BuildID, cfg.WorldEndpoint)
	if err != nil {
		log.Fatal(err)
	}

	if err := fileStore.EnsureAdminSeed(cfg.AdminSeedUsername, cfg.AdminSeedPassword); err != nil {
		log.Fatal(err)
	}

	mux := http.NewServeMux()
	characters.RegisterRoutes(mux, fileStore)
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("ok"))
	})

	listenAddress := cfg.ListenAddress()
	log.Printf("%s listening on %s", cfg.ServiceName, listenAddress)
	log.Fatal(http.ListenAndServe(listenAddress, httpapi.WithCORS(mux)))
}
