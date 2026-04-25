package api

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/Fishy97/mediarr/backend/internal/ai"
	"github.com/Fishy97/mediarr/backend/internal/audit"
	"github.com/Fishy97/mediarr/backend/internal/auth"
	"github.com/Fishy97/mediarr/backend/internal/database"
	"github.com/Fishy97/mediarr/backend/internal/filescan"
	"github.com/Fishy97/mediarr/backend/internal/integrations"
	"github.com/Fishy97/mediarr/backend/internal/metadata"
	"github.com/Fishy97/mediarr/backend/internal/recommendations"
)

type Deps struct {
	ConfigDir          string
	FrontendDir        string
	Libraries          []filescan.Library
	Recommendations    []recommendations.Recommendation
	Providers          []metadata.Provider
	ProviderOptions    metadata.Options
	Integrations       []integrations.Target
	IntegrationOptions integrations.Options
	Audit              *audit.Logger
	Auth               *auth.Service
	AI                 *ai.OllamaClient
	Scanner            filescan.Scanner
	Engine             recommendations.Engine
	Store              *database.Store
}

type Server struct {
	mux                *http.ServeMux
	configDir          string
	frontendDir        string
	mu                 sync.RWMutex
	libraries          []filescan.Library
	scans              []filescan.Result
	recommendations    []recommendations.Recommendation
	providers          []metadata.Provider
	providerBase       metadata.Options
	providerOptions    metadata.Options
	integrations       []integrations.Target
	integrationOptions integrations.Options
	audit              *audit.Logger
	auth               *auth.Service
	ai                 *ai.OllamaClient
	scanner            filescan.Scanner
	engine             recommendations.Engine
	store              *database.Store
}

func NewServer(deps Deps) *Server {
	server := &Server{
		mux:                http.NewServeMux(),
		configDir:          deps.ConfigDir,
		frontendDir:        deps.FrontendDir,
		libraries:          deps.Libraries,
		recommendations:    deps.Recommendations,
		providers:          deps.Providers,
		providerBase:       deps.ProviderOptions,
		providerOptions:    deps.ProviderOptions,
		integrations:       deps.Integrations,
		integrationOptions: deps.IntegrationOptions,
		audit:              deps.Audit,
		auth:               deps.Auth,
		ai:                 deps.AI,
		scanner:            deps.Scanner,
		engine:             deps.Engine,
		store:              deps.Store,
	}
	if server.store != nil {
		_ = server.refreshProviderOptionsFromStore()
	}
	if server.providers == nil {
		server.providers = metadata.DefaultsWithOptions(server.providerOptions)
	}
	if server.integrations == nil {
		server.integrations = integrations.DefaultsWithOptions(server.integrationOptions)
	}
	server.routes()
	return server
}

func (server *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	server.mux.ServeHTTP(w, r)
}

func (server *Server) routes() {
	server.mux.HandleFunc("/api/v1/health", server.health)
	server.mux.HandleFunc("/api/v1/setup/status", server.setupStatusHandler)
	server.mux.HandleFunc("/api/v1/setup/admin", server.setupAdminHandler)
	server.mux.HandleFunc("/api/v1/auth/login", server.loginHandler)
	server.mux.HandleFunc("/api/v1/auth/logout", server.logoutHandler)
	server.mux.HandleFunc("/api/v1/auth/me", server.meHandler)
	server.mux.HandleFunc("/api/v1/libraries", server.librariesHandler)
	server.mux.HandleFunc("/api/v1/catalog", server.catalogHandler)
	server.mux.HandleFunc("/api/v1/catalog/", server.catalogCorrectionHandler)
	server.mux.HandleFunc("/api/v1/scans", server.scansHandler)
	server.mux.HandleFunc("/api/v1/recommendations", server.recommendationsHandler)
	server.mux.HandleFunc("/api/v1/recommendations/", server.recommendationActionHandler)
	server.mux.HandleFunc("/api/v1/providers", server.providersHandler)
	server.mux.HandleFunc("/api/v1/provider-settings", server.providerSettingsHandler)
	server.mux.HandleFunc("/api/v1/provider-settings/", server.providerSettingHandler)
	server.mux.HandleFunc("/api/v1/integrations", server.integrationsHandler)
	server.mux.HandleFunc("/api/v1/integrations/", server.integrationActionHandler)
	server.mux.HandleFunc("/api/v1/ai/status", server.aiStatusHandler)
	server.mux.HandleFunc("/api/v1/backups", server.backupsHandler)
	server.mux.HandleFunc("/api/v1/backups/restore", server.backupRestoreHandler)
	server.mux.HandleFunc("/api/v1/audit", server.auditHandler)
	server.mux.HandleFunc("/api/v1/media/files/", methodNotAllowed)
	server.mux.HandleFunc("/", server.frontend)
}

func (server *Server) setupStatusHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w, r)
		return
	}
	if server.auth == nil {
		writeJSON(w, http.StatusOK, envelope{Data: map[string]bool{"setupRequired": false}})
		return
	}
	required, err := server.auth.SetupRequired()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, envelope{Data: map[string]bool{"setupRequired": required}})
}

func (server *Server) setupAdminHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w, r)
		return
	}
	if server.auth == nil {
		http.Error(w, "authentication is not configured", http.StatusBadRequest)
		return
	}
	var request credentialsRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	user, session, err := server.auth.CreateAdmin(request.Email, request.Password)
	if err != nil {
		status := http.StatusBadRequest
		if err == auth.ErrSetupComplete {
			status = http.StatusConflict
		}
		http.Error(w, err.Error(), status)
		return
	}
	server.setSessionCookie(w, session)
	server.record("auth.admin_created", "Initial admin account created", map[string]any{"email": user.Email})
	writeJSON(w, http.StatusCreated, envelope{Data: authResponse{User: user, Token: session.Token, ExpiresAt: session.ExpiresAt}})
}

func (server *Server) loginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w, r)
		return
	}
	if server.auth == nil {
		http.Error(w, "authentication is not configured", http.StatusBadRequest)
		return
	}
	var request credentialsRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	user, session, err := server.auth.Login(request.Email, request.Password)
	if err != nil {
		http.Error(w, "invalid credentials", http.StatusUnauthorized)
		return
	}
	server.setSessionCookie(w, session)
	server.record("auth.login", "Admin logged in", map[string]any{"email": user.Email})
	writeJSON(w, http.StatusOK, envelope{Data: authResponse{User: user, Token: session.Token, ExpiresAt: session.ExpiresAt}})
}

func (server *Server) logoutHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w, r)
		return
	}
	if server.auth != nil {
		_ = server.auth.Logout(requestToken(r))
	}
	http.SetCookie(w, &http.Cookie{
		Name:     auth.SessionCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
	writeJSON(w, http.StatusOK, envelope{Data: map[string]bool{"ok": true}})
}

func (server *Server) meHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w, r)
		return
	}
	if server.auth == nil {
		http.Error(w, "authentication is not configured", http.StatusBadRequest)
		return
	}
	user, err := server.auth.UserForToken(requestToken(r))
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	writeJSON(w, http.StatusOK, envelope{Data: user})
}

func (server *Server) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"status":    "ok",
		"service":   "mediarr",
		"timestamp": time.Now().UTC(),
	})
}

func (server *Server) librariesHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		server.mu.RLock()
		defer server.mu.RUnlock()
		writeJSON(w, http.StatusOK, envelope{Data: server.libraries})
	case http.MethodPost:
		var library filescan.Library
		if err := json.NewDecoder(r.Body).Decode(&library); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if strings.TrimSpace(library.ID) == "" || strings.TrimSpace(library.Root) == "" {
			http.Error(w, "library id and root are required", http.StatusBadRequest)
			return
		}
		server.mu.Lock()
		server.libraries = append(server.libraries, library)
		server.mu.Unlock()
		server.record("library.created", "Library configured", map[string]any{"id": library.ID, "root": library.Root})
		writeJSON(w, http.StatusCreated, envelope{Data: library})
	default:
		methodNotAllowed(w, r)
	}
}

func (server *Server) scansHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		server.mu.RLock()
		defer server.mu.RUnlock()
		writeJSON(w, http.StatusOK, envelope{Data: server.scans})
	case http.MethodPost:
		results, recs, err := server.scanAll(r.Context())
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		writeJSON(w, http.StatusAccepted, envelope{Data: map[string]any{"scans": results, "recommendations": recs}})
	default:
		methodNotAllowed(w, r)
	}
}

func (server *Server) catalogHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w, r)
		return
	}
	if server.store != nil {
		items, err := server.store.ListCatalog()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, envelope{Data: items})
		return
	}
	server.mu.RLock()
	defer server.mu.RUnlock()
	var items []filescan.Item
	for _, scan := range server.scans {
		items = append(items, scan.Items...)
	}
	writeJSON(w, http.StatusOK, envelope{Data: items})
}

func (server *Server) catalogCorrectionHandler(w http.ResponseWriter, r *http.Request) {
	if server.store == nil {
		http.Error(w, "catalog store is not configured", http.StatusBadRequest)
		return
	}
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/v1/catalog/"), "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] != "correction" {
		http.NotFound(w, r)
		return
	}
	switch r.Method {
	case http.MethodPut:
		var request database.CatalogCorrectionInput
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		correction, err := server.store.UpsertCatalogCorrection(parts[0], request)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if err := server.regenerateRecommendationsFromCatalog(); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		server.record("catalog.corrected", "Catalog metadata correction applied", map[string]any{"mediaFileId": parts[0], "provider": correction.Provider, "providerId": correction.ProviderID})
		writeJSON(w, http.StatusOK, envelope{Data: correction})
	case http.MethodDelete:
		if err := server.store.ClearCatalogCorrection(parts[0]); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if err := server.regenerateRecommendationsFromCatalog(); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		server.record("catalog.correction_cleared", "Catalog metadata correction cleared", map[string]any{"mediaFileId": parts[0]})
		writeJSON(w, http.StatusOK, envelope{Data: map[string]bool{"ok": true}})
	default:
		methodNotAllowed(w, r)
	}
}

func (server *Server) scanAll(ctx context.Context) ([]filescan.Result, []recommendations.Recommendation, error) {
	server.mu.RLock()
	libraries := append([]filescan.Library(nil), server.libraries...)
	server.mu.RUnlock()

	var results []filescan.Result
	var files []recommendations.MediaFile
	for _, library := range libraries {
		result, err := server.scanner.Scan(library)
		if err != nil {
			return nil, nil, err
		}
		results = append(results, result)
		if server.store != nil {
			if err := server.store.SaveScan(result); err != nil {
				return nil, nil, err
			}
		}
		if server.store == nil {
			for _, item := range result.Items {
				files = append(files, recommendations.MediaFile{
					ID:             item.ID,
					CanonicalKey:   item.Parsed.CanonicalKey,
					Path:           item.Path,
					SizeBytes:      item.SizeBytes,
					Quality:        item.Parsed.Quality,
					HasSubtitles:   len(item.Subtitles) > 0,
					WantsSubtitles: wantsSubtitles(string(item.Parsed.Kind)),
				})
			}
		}
	}
	if server.store != nil {
		items, err := server.store.ListCatalog()
		if err != nil {
			return nil, nil, err
		}
		files = recommendationFilesFromCatalog(items)
	}
	recs := server.enrichRecommendationsWithAI(ctx, server.engine.Generate(files))
	if server.store != nil {
		if err := server.store.ReplaceRecommendations(recs); err != nil {
			return nil, nil, err
		}
	}

	server.mu.Lock()
	server.scans = append(server.scans, results...)
	server.recommendations = recs
	server.mu.Unlock()
	server.record("scan.completed", "Library scan completed", map[string]any{"libraries": len(libraries), "recommendations": len(recs)})
	return results, recs, nil
}

func (server *Server) recommendationsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w, r)
		return
	}
	if server.store != nil {
		recs, err := server.store.ListRecommendations()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, envelope{Data: recs})
		return
	}
	server.mu.RLock()
	defer server.mu.RUnlock()
	writeJSON(w, http.StatusOK, envelope{Data: server.recommendations})
}

func (server *Server) recommendationActionHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w, r)
		return
	}
	if server.store == nil {
		http.Error(w, "recommendation store is not configured", http.StatusBadRequest)
		return
	}
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/v1/recommendations/"), "/")
	if len(parts) != 2 || parts[0] == "" {
		http.NotFound(w, r)
		return
	}
	switch parts[1] {
	case "ignore":
		if err := server.store.IgnoreRecommendation(parts[0]); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		server.record("recommendation.ignored", "Recommendation ignored", map[string]any{"id": parts[0]})
	case "restore":
		if err := server.store.RestoreRecommendation(parts[0]); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		server.record("recommendation.restored", "Recommendation restored", map[string]any{"id": parts[0]})
	default:
		http.NotFound(w, r)
		return
	}
	writeJSON(w, http.StatusOK, envelope{Data: map[string]bool{"ok": true}})
}

func (server *Server) providersHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w, r)
		return
	}
	server.mu.RLock()
	providers := append([]metadata.Provider(nil), server.providers...)
	server.mu.RUnlock()
	health := make([]metadata.Health, 0, len(providers))
	for _, provider := range providers {
		health = append(health, provider.Health(r.Context()))
	}
	writeJSON(w, http.StatusOK, envelope{Data: health})
}

func (server *Server) providerSettingsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w, r)
		return
	}
	if server.store == nil {
		http.Error(w, "provider settings store is not configured", http.StatusBadRequest)
		return
	}
	settings, err := server.store.ListProviderSettings()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, envelope{Data: settings})
}

func (server *Server) providerSettingHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		methodNotAllowed(w, r)
		return
	}
	if server.store == nil {
		http.Error(w, "provider settings store is not configured", http.StatusBadRequest)
		return
	}
	provider := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/v1/provider-settings/"), "/")
	if provider == "" {
		http.NotFound(w, r)
		return
	}
	var request database.ProviderSettingInput
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	request.Provider = provider
	setting, err := server.store.UpsertProviderSetting(request)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := server.refreshProviderOptionsFromStore(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	server.mu.Lock()
	server.providers = metadata.DefaultsWithOptions(server.providerOptions)
	server.mu.Unlock()
	server.record("provider.updated", "Metadata provider settings updated", map[string]any{"provider": setting.Provider, "apiKeyConfigured": setting.APIKeyConfigured})
	writeJSON(w, http.StatusOK, envelope{Data: setting})
}

func (server *Server) integrationsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w, r)
		return
	}
	server.mu.Lock()
	server.integrations = integrations.DefaultsWithOptions(server.integrationOptions)
	integrationRows := append([]integrations.Target(nil), server.integrations...)
	server.mu.Unlock()
	writeJSON(w, http.StatusOK, envelope{Data: integrationRows})
}

func (server *Server) integrationActionHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w, r)
		return
	}
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/v1/integrations/"), "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] != "refresh" {
		http.NotFound(w, r)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 12*time.Second)
	defer cancel()
	result, err := integrations.Refresh(ctx, server.integrationOptions, parts[0])
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	server.record("integration.refresh_requested", "Media-server library refresh requested", map[string]any{"targetId": result.TargetID, "status": result.Status})
	writeJSON(w, http.StatusAccepted, envelope{Data: result})
}

func (server *Server) aiStatusHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w, r)
		return
	}
	if server.ai == nil {
		writeJSON(w, http.StatusOK, envelope{Data: ai.Health{Status: "not_configured", CheckedAt: time.Now().UTC()}})
		return
	}
	writeJSON(w, http.StatusOK, envelope{Data: server.ai.Health(r.Context())})
}

func (server *Server) backupsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w, r)
		return
	}
	if server.configDir == "" {
		http.Error(w, "config directory not configured", http.StatusBadRequest)
		return
	}
	path, err := database.CreateBackup(server.configDir, filepath.Join(server.configDir, "backups"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	server.record("backup.created", "Backup created", map[string]any{"path": path})
	writeJSON(w, http.StatusCreated, envelope{Data: map[string]string{"path": path}})
}

func (server *Server) backupRestoreHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w, r)
		return
	}
	if server.configDir == "" {
		http.Error(w, "config directory not configured", http.StatusBadRequest)
		return
	}
	var request struct {
		Path   string `json:"path"`
		DryRun bool   `json:"dryRun"`
	}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if request.Path == "" {
		http.Error(w, "backup path is required", http.StatusBadRequest)
		return
	}
	if request.DryRun {
		entries, err := database.InspectBackup(request.Path)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		writeJSON(w, http.StatusOK, envelope{Data: map[string]any{"entries": entries}})
		return
	}
	result, err := database.RestoreBackup(server.configDir, request.Path, filepath.Join(server.configDir, "backups"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	server.record("backup.restored", "Backup restored", map[string]any{"path": request.Path, "preRestoreBackup": result.PreRestoreBackup})
	writeJSON(w, http.StatusOK, envelope{Data: result})
}

func (server *Server) auditHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w, r)
		return
	}
	path := filepath.Join(server.configDir, "audit", "events.jsonl")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			writeJSON(w, http.StatusOK, envelope{Data: []string{}})
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) == 1 && lines[0] == "" {
		lines = []string{}
	}
	writeJSON(w, http.StatusOK, envelope{Data: lines})
}

func (server *Server) frontend(w http.ResponseWriter, r *http.Request) {
	if server.frontendDir == "" {
		http.NotFound(w, r)
		return
	}
	path := filepath.Join(server.frontendDir, filepath.Clean(r.URL.Path))
	if stat, err := os.Stat(path); err == nil && !stat.IsDir() {
		http.ServeFile(w, r, path)
		return
	}
	index := filepath.Join(server.frontendDir, "index.html")
	if _, err := os.Stat(index); err != nil {
		http.NotFound(w, r)
		return
	}
	http.ServeFile(w, r, index)
}

func (server *Server) record(kind string, message string, fields map[string]any) {
	if server.audit == nil {
		return
	}
	_ = server.audit.Record(audit.Event{Type: kind, Message: message, Fields: fields})
}

func (server *Server) refreshProviderOptionsFromStore() error {
	if server.store == nil {
		return nil
	}
	settings, err := server.store.ListProviderSettingSecrets()
	if err != nil {
		return err
	}
	server.providerOptions = mergeProviderSettings(server.providerBase, settings)
	return nil
}

func mergeProviderSettings(options metadata.Options, settings []database.ProviderSetting) metadata.Options {
	for _, setting := range settings {
		switch setting.Provider {
		case "tmdb":
			options.TMDbToken = setting.APIKey
			if setting.BaseURL != "" {
				options.TMDbBaseURL = setting.BaseURL
			}
		case "thetvdb":
			options.TheTVDBAPIKey = setting.APIKey
			if setting.BaseURL != "" {
				options.TheTVDBBaseURL = setting.BaseURL
			}
		case "opensubtitles":
			options.OpenSubtitlesAPIKey = setting.APIKey
			if setting.BaseURL != "" {
				options.OpenSubtitlesURL = setting.BaseURL
			}
		}
	}
	return options
}

func (server *Server) regenerateRecommendationsFromCatalog() error {
	if server.store == nil {
		return nil
	}
	items, err := server.store.ListCatalog()
	if err != nil {
		return err
	}
	recs := server.enrichRecommendationsWithAI(context.Background(), server.engine.Generate(recommendationFilesFromCatalog(items)))
	if err := server.store.ReplaceRecommendations(recs); err != nil {
		return err
	}
	server.mu.Lock()
	server.recommendations = recs
	server.mu.Unlock()
	return nil
}

func (server *Server) enrichRecommendationsWithAI(ctx context.Context, recs []recommendations.Recommendation) []recommendations.Recommendation {
	if server.ai == nil || len(recs) == 0 {
		return recs
	}
	health := server.ai.Health(ctx)
	if !health.ModelAvailable {
		return recs
	}
	enriched := append([]recommendations.Recommendation(nil), recs...)
	for index := range enriched {
		if enriched[index].Destructive || !strings.HasPrefix(enriched[index].Source, "rule:") {
			continue
		}
		itemCtx, cancel := context.WithTimeout(ctx, 12*time.Second)
		suggestion, err := server.ai.SuggestRationale(itemCtx, ai.SuggestionInput{
			Title:         enriched[index].Title,
			Explanation:   enriched[index].Explanation,
			AffectedPaths: enriched[index].AffectedPaths,
		})
		cancel()
		if err != nil {
			continue
		}
		enriched[index].AIRationale = suggestion.Rationale
		enriched[index].AITags = suggestion.Tags
		enriched[index].AIConfidence = suggestion.Confidence
		enriched[index].AISource = "ollama:" + health.Model
	}
	return enriched
}

func recommendationFilesFromCatalog(items []database.CatalogItem) []recommendations.MediaFile {
	files := make([]recommendations.MediaFile, 0, len(items))
	for _, item := range items {
		files = append(files, recommendations.MediaFile{
			ID:             item.ID,
			CanonicalKey:   item.CanonicalKey,
			Path:           item.Path,
			SizeBytes:      item.SizeBytes,
			Quality:        item.Quality,
			HasSubtitles:   len(item.Subtitles) > 0,
			WantsSubtitles: wantsSubtitles(string(item.Kind)),
		})
	}
	return files
}

func wantsSubtitles(kind string) bool {
	switch kind {
	case "movie", "series", "anime":
		return true
	default:
		return false
	}
}

func (server *Server) setSessionCookie(w http.ResponseWriter, session auth.Session) {
	http.SetCookie(w, &http.Cookie{
		Name:     auth.SessionCookieName,
		Value:    session.Token,
		Path:     "/",
		Expires:  session.ExpiresAt,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
}

func requestToken(r *http.Request) string {
	header := strings.TrimSpace(r.Header.Get("Authorization"))
	if strings.HasPrefix(strings.ToLower(header), "bearer ") {
		return strings.TrimSpace(header[7:])
	}
	if cookie, err := r.Cookie(auth.SessionCookieName); err == nil {
		return cookie.Value
	}
	return ""
}

type credentialsRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type authResponse struct {
	User      database.User `json:"user"`
	Token     string        `json:"token"`
	ExpiresAt time.Time     `json:"expiresAt"`
}

type envelope struct {
	Data any `json:"data"`
}

func methodNotAllowed(w http.ResponseWriter, _ *http.Request) {
	http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
