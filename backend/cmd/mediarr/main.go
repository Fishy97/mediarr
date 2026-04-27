package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/Fishy97/mediarr/backend/internal/ai"
	"github.com/Fishy97/mediarr/backend/internal/api"
	"github.com/Fishy97/mediarr/backend/internal/audit"
	"github.com/Fishy97/mediarr/backend/internal/auth"
	"github.com/Fishy97/mediarr/backend/internal/config"
	"github.com/Fishy97/mediarr/backend/internal/database"
	"github.com/Fishy97/mediarr/backend/internal/filescan"
	"github.com/Fishy97/mediarr/backend/internal/integrations"
	"github.com/Fishy97/mediarr/backend/internal/metadata"
	"github.com/Fishy97/mediarr/backend/internal/recommendations"
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

	authService := auth.Service{Store: store}
	aiClient := ai.OllamaClient{BaseURL: cfg.OllamaURL, Model: cfg.AIModel}
	providerOptions := metadata.Options{
		TMDbToken:           cfg.TMDbToken,
		TheTVDBAPIKey:       cfg.TheTVDBAPIKey,
		OpenSubtitlesAPIKey: cfg.OpenSubtitlesKey,
	}
	integrationOptions := integrations.Options{
		JellyfinURL: cfg.JellyfinURL,
		JellyfinKey: cfg.JellyfinAPIKey,
		PlexURL:     cfg.PlexURL,
		PlexToken:   cfg.PlexToken,
		EmbyURL:     cfg.EmbyURL,
		EmbyKey:     cfg.EmbyAPIKey,
		TautulliURL: cfg.TautulliURL,
		TautulliKey: cfg.TautulliAPIKey,
	}

	server := api.NewServer(api.Deps{
		ConfigDir:          cfg.ConfigDir,
		FrontendDir:        cfg.FrontendDir,
		Libraries:          libraries,
		Audit:              auditLog,
		Auth:               &authService,
		AI:                 &aiClient,
		ProviderOptions:    providerOptions,
		IntegrationOptions: integrationOptions,
		Scanner:            filescan.Scanner{Probe: true},
		Engine:             recommendations.Engine{OversizedThresholdBytes: cfg.OversizedBytes},
		Store:              store,
	})
	handler := auth.Middleware{AdminToken: cfg.AdminToken, Service: &authService}.Wrap(server)
	server.StartAutoSync(context.Background(), time.Minute)

	log.Printf("Mediarr listening on %s", cfg.Addr)
	if err := http.ListenAndServe(cfg.Addr, handler); err != nil {
		log.Fatal(err)
	}
}
