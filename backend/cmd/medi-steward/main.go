package main

import (
	"log"
	"net/http"

	"github.com/media-steward/media-library-manager/backend/internal/api"
	"github.com/media-steward/media-library-manager/backend/internal/audit"
	"github.com/media-steward/media-library-manager/backend/internal/auth"
	"github.com/media-steward/media-library-manager/backend/internal/config"
	"github.com/media-steward/media-library-manager/backend/internal/database"
	"github.com/media-steward/media-library-manager/backend/internal/filescan"
	"github.com/media-steward/media-library-manager/backend/internal/recommendations"
)

func main() {
	cfg := config.Load()

	store, err := database.Open(cfg.ConfigDir)
	if err != nil {
		log.Fatalf("open database: %v", err)
	}
	defer store.Close()

	auditLog, err := audit.New(cfg.ConfigDir)
	if err != nil {
		log.Fatalf("open audit log: %v", err)
	}

	libraries := make([]filescan.Library, 0, len(cfg.DefaultLibraries))
	for _, library := range cfg.DefaultLibraries {
		libraries = append(libraries, filescan.Library{
			ID:   library.ID,
			Name: library.Name,
			Kind: library.Kind,
			Root: library.Root,
		})
	}

	server := api.NewServer(api.Deps{
		ConfigDir:   cfg.ConfigDir,
		FrontendDir: cfg.FrontendDir,
		Libraries:   libraries,
		Audit:       auditLog,
		Scanner:     filescan.Scanner{Probe: true},
		Engine:      recommendations.Engine{OversizedThresholdBytes: cfg.OversizedBytes},
		Store:       store,
	})
	handler := auth.Middleware{AdminToken: cfg.AdminToken}.Wrap(server)

	log.Printf("Media Steward listening on %s", cfg.Addr)
	if err := http.ListenAndServe(cfg.Addr, handler); err != nil {
		log.Fatal(err)
	}
}
