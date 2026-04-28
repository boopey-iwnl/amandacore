package main

import (
	"log"
	"net/http"

	"amandacore/services/internal/authn"
	"amandacore/services/internal/config"
	"amandacore/services/internal/httpapi"
	"amandacore/services/internal/servicehost"
)

func main() {
	cfg, err := config.LoadValidated("auth-service", "8081")
	if err != nil {
		log.Fatal(err)
	}
	fileStore, storageReport, err := servicehost.OpenPlatformStore(cfg)
	if err != nil {
		log.Printf("%s storage backend=%s environment=%s migrations=%s pending=%d", cfg.ServiceName, storageReport.Backend, storageReport.Environment, storageReport.MigrationState, storageReport.PendingCount)
		log.Fatal(err)
	}
	log.Printf("%s storage backend=%s environment=%s migrations=%s", cfg.ServiceName, storageReport.Backend, storageReport.Environment, storageReport.MigrationState)

	if err := fileStore.EnsureAdminSeed(cfg.AdminSeedUsername, cfg.AdminSeedPassword); err != nil {
		log.Fatal(err)
	}

	mux := http.NewServeMux()
	authn.RegisterRoutes(mux, fileStore)
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("ok"))
	})

	listenAddress := cfg.ListenAddress()
	log.Printf("%s listening on %s", cfg.ServiceName, listenAddress)
	log.Fatal(http.ListenAndServe(listenAddress, httpapi.WithCORS(mux)))
}
