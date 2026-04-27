package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Fishy97/mediarr/backend/internal/acceptance"
	"github.com/Fishy97/mediarr/backend/internal/ai"
	"github.com/Fishy97/mediarr/backend/internal/audit"
	"github.com/Fishy97/mediarr/backend/internal/auth"
	"github.com/Fishy97/mediarr/backend/internal/campaigns"
	"github.com/Fishy97/mediarr/backend/internal/database"
	"github.com/Fishy97/mediarr/backend/internal/filescan"
	"github.com/Fishy97/mediarr/backend/internal/integrations"
	"github.com/Fishy97/mediarr/backend/internal/metadata"
	"github.com/Fishy97/mediarr/backend/internal/recommendations"
	"github.com/Fishy97/mediarr/backend/internal/stewardship"
	"github.com/Fishy97/mediarr/backend/internal/support"
)

const (
	maxAIEnrichedRecommendations = 8
	aiRecommendationTimeout      = 6 * time.Second
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
	integrationBase    integrations.Options
	integrationOptions integrations.Options
	audit              *audit.Logger
	auth               *auth.Service
	ai                 *ai.OllamaClient
	scanner            filescan.Scanner
	engine             recommendations.Engine
	store              *database.Store
	jobCancelMu        sync.Mutex
	jobCancels         map[string]context.CancelFunc
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
		integrationBase:    deps.IntegrationOptions,
		integrationOptions: deps.IntegrationOptions,
		audit:              deps.Audit,
		auth:               deps.Auth,
		ai:                 deps.AI,
		scanner:            deps.Scanner,
		engine:             deps.Engine,
		store:              deps.Store,
		jobCancels:         map[string]context.CancelFunc{},
	}
	if server.store != nil {
		_ = server.refreshProviderOptionsFromStore()
		_ = server.refreshIntegrationOptionsFromStore()
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
	server.mux.HandleFunc("/api/v1/jobs", server.jobsHandler)
	server.mux.HandleFunc("/api/v1/jobs/", server.jobHandler)
	server.mux.HandleFunc("/api/v1/catalog", server.catalogHandler)
	server.mux.HandleFunc("/api/v1/catalog/", server.catalogCorrectionHandler)
	server.mux.HandleFunc("/api/v1/scans", server.scansHandler)
	server.mux.HandleFunc("/api/v1/scans/active", server.activeScanHandler)
	server.mux.HandleFunc("/api/v1/recommendations", server.recommendationsHandler)
	server.mux.HandleFunc("/api/v1/recommendations/", server.recommendationActionHandler)
	server.mux.HandleFunc("/api/v1/providers", server.providersHandler)
	server.mux.HandleFunc("/api/v1/provider-settings", server.providerSettingsHandler)
	server.mux.HandleFunc("/api/v1/provider-settings/", server.providerSettingHandler)
	server.mux.HandleFunc("/api/v1/integration-settings", server.integrationSettingsHandler)
	server.mux.HandleFunc("/api/v1/integration-settings/", server.integrationSettingHandler)
	server.mux.HandleFunc("/api/v1/integrations", server.integrationsHandler)
	server.mux.HandleFunc("/api/v1/integrations/", server.integrationActionHandler)
	server.mux.HandleFunc("/api/v1/activity/rollups", server.activityRollupsHandler)
	server.mux.HandleFunc("/api/v1/request-sources", server.requestSourcesHandler)
	server.mux.HandleFunc("/api/v1/request-sources/", server.requestSourceHandler)
	server.mux.HandleFunc("/api/v1/request-signals", server.requestSignalsHandler)
	server.mux.HandleFunc("/api/v1/campaign-templates", server.campaignTemplatesHandler)
	server.mux.HandleFunc("/api/v1/campaign-templates/", server.campaignTemplateHandler)
	server.mux.HandleFunc("/api/v1/campaigns", server.campaignsHandler)
	server.mux.HandleFunc("/api/v1/campaigns/", server.campaignHandler)
	server.mux.HandleFunc("/api/v1/storage-ledger", server.storageLedgerHandler)
	server.mux.HandleFunc("/api/v1/notifications", server.notificationsHandler)
	server.mux.HandleFunc("/api/v1/notifications/", server.notificationHandler)
	server.mux.HandleFunc("/api/v1/protection-requests", server.protectionRequestsHandler)
	server.mux.HandleFunc("/api/v1/protection-requests/", server.protectionRequestHandler)
	server.mux.HandleFunc("/api/v1/path-mappings", server.pathMappingsHandler)
	server.mux.HandleFunc("/api/v1/path-mappings/", server.pathMappingHandler)
	server.mux.HandleFunc("/api/v1/ai/status", server.aiStatusHandler)
	server.mux.HandleFunc("/api/v1/backups/", server.backupFileHandler)
	server.mux.HandleFunc("/api/v1/backups", server.backupsHandler)
	server.mux.HandleFunc("/api/v1/backups/restore", server.backupRestoreHandler)
	server.mux.HandleFunc("/api/v1/support/bundles/", server.supportBundleFileHandler)
	server.mux.HandleFunc("/api/v1/support/bundles", server.supportBundlesHandler)
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
	server.setSessionCookie(w, r, session)
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
	server.setSessionCookie(w, r, session)
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
		writeJSON(w, http.StatusOK, envelope{Data: emptyIfNil(server.libraries)})
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

type jobDetail struct {
	database.Job
	Events []database.JobEvent `json:"events"`
}

func (server *Server) jobsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w, r)
		return
	}
	if server.store == nil {
		http.Error(w, "job store is not configured", http.StatusBadRequest)
		return
	}
	_, _ = server.store.MarkStaleJobs(time.Now().Add(-24 * time.Hour))
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	jobs, err := server.store.ListJobs(database.JobFilter{
		Kind:     r.URL.Query().Get("kind"),
		TargetID: r.URL.Query().Get("targetId"),
		Status:   r.URL.Query().Get("status"),
		Active:   r.URL.Query().Get("active") == "true",
		Limit:    limit,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, envelope{Data: jobs})
}

func (server *Server) jobHandler(w http.ResponseWriter, r *http.Request) {
	if server.store == nil {
		http.Error(w, "job store is not configured", http.StatusBadRequest)
		return
	}
	parts := strings.Split(strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/v1/jobs/"), "/"), "/")
	if len(parts) == 0 || parts[0] == "" {
		http.NotFound(w, r)
		return
	}
	id := parts[0]
	if len(parts) == 2 {
		switch parts[1] {
		case "cancel":
			server.cancelJobHandler(w, r, id)
		case "retry":
			server.retryJobHandler(w, r, id)
		default:
			http.NotFound(w, r)
		}
		return
	}
	if len(parts) != 1 {
		http.NotFound(w, r)
		return
	}
	if r.Method != http.MethodGet {
		methodNotAllowed(w, r)
		return
	}
	job, err := server.store.GetJob(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	events, err := server.store.ListJobEvents(id, 30)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, envelope{Data: jobDetail{Job: job, Events: events}})
}

func (server *Server) cancelJobHandler(w http.ResponseWriter, r *http.Request, id string) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w, r)
		return
	}
	server.cancelRunningJob(id)
	job, err := server.store.UpdateJob(id, database.JobUpdate{
		Status:    "canceled",
		Phase:     "canceled",
		Message:   "Canceled by admin",
		Error:     "job canceled by admin",
		Completed: true,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	_, _ = server.store.AddJobEvent(database.JobEventInput{
		JobID:   id,
		Level:   "warning",
		Phase:   job.Phase,
		Message: job.Message,
	})
	server.record("job.canceled", "Background job canceled", map[string]any{"id": id, "kind": job.Kind, "targetId": job.TargetID})
	writeJSON(w, http.StatusOK, envelope{Data: job})
}

func (server *Server) retryJobHandler(w http.ResponseWriter, r *http.Request, id string) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w, r)
		return
	}
	previous, err := server.store.GetJob(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	if previous.Status == "queued" || previous.Status == "running" {
		http.Error(w, "active jobs cannot be retried", http.StatusConflict)
		return
	}
	job, err := server.store.CreateJob(database.JobInput{
		Kind:     previous.Kind,
		TargetID: previous.TargetID,
		Phase:    "queued",
		Message:  "Retry queued",
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := server.startJob(job); err != nil {
		_, _ = server.store.UpdateJob(job.ID, database.JobUpdate{Status: "failed", Phase: "failed", Message: "Unable to retry job", Error: err.Error(), Completed: true})
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	server.record("job.retried", "Background job retry queued", map[string]any{"id": previous.ID, "retryId": job.ID, "kind": job.Kind, "targetId": job.TargetID})
	writeJSON(w, http.StatusAccepted, envelope{Data: job})
}

func (server *Server) scansHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		server.mu.RLock()
		defer server.mu.RUnlock()
		writeJSON(w, http.StatusOK, envelope{Data: emptyIfNil(server.scans)})
	case http.MethodPost:
		if server.store != nil {
			job, err := server.store.CreateJob(database.JobInput{
				Kind:     "filesystem_scan",
				TargetID: "all",
				Phase:    "queued",
				Message:  "Filesystem scan queued",
			})
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			if err := server.startJob(job); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			writeJSON(w, http.StatusAccepted, envelope{Data: job})
			return
		}
		results, recs, err := server.scanAll(r.Context(), nil)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		writeJSON(w, http.StatusAccepted, envelope{Data: map[string]any{"scans": emptyIfNil(results), "recommendations": emptyIfNil(recs)}})
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
		if err := server.regenerateRecommendationsFromCatalog(r.Context()); err != nil {
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
		if err := server.regenerateRecommendationsFromCatalog(r.Context()); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		server.record("catalog.correction_cleared", "Catalog metadata correction cleared", map[string]any{"mediaFileId": parts[0]})
		writeJSON(w, http.StatusOK, envelope{Data: map[string]bool{"ok": true}})
	default:
		methodNotAllowed(w, r)
	}
}

func (server *Server) activeScanHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w, r)
		return
	}
	if server.store == nil {
		http.Error(w, "job store is not configured", http.StatusBadRequest)
		return
	}
	jobs, err := server.store.ListJobs(database.JobFilter{Kind: "filesystem_scan", TargetID: "all", Active: true, Limit: 1})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, envelope{Data: jobs})
}

func (server *Server) startJob(job database.Job) error {
	switch job.Kind {
	case "filesystem_scan":
		go server.runScanJob(job.ID)
	case "jellyfin_sync", "plex_sync", "emby_sync":
		go server.runIntegrationSyncJob(job.ID, job.TargetID)
	case "tautulli_sync":
		go server.runTautulliSyncJob(job.ID)
	default:
		return errors.New("job kind cannot be started")
	}
	return nil
}

func (server *Server) registerJobCancel(jobID string, cancel context.CancelFunc) {
	server.jobCancelMu.Lock()
	defer server.jobCancelMu.Unlock()
	server.jobCancels[jobID] = cancel
}

func (server *Server) unregisterJobCancel(jobID string) {
	server.jobCancelMu.Lock()
	defer server.jobCancelMu.Unlock()
	delete(server.jobCancels, jobID)
}

func (server *Server) cancelRunningJob(jobID string) {
	server.jobCancelMu.Lock()
	cancel := server.jobCancels[jobID]
	server.jobCancelMu.Unlock()
	if cancel != nil {
		cancel()
	}
}

func (server *Server) runScanJob(jobID string) {
	reporter := jobReporter{store: server.store, jobID: jobID}
	ctx, cancel := context.WithTimeout(context.Background(), 24*time.Hour)
	defer cancel()
	server.registerJobCancel(jobID, cancel)
	defer server.unregisterJobCancel(jobID)
	reporter.update(database.JobUpdate{Status: "running", Phase: "starting", Message: "Starting filesystem scan"}, true)
	if _, _, err := server.scanAll(ctx, &reporter); err != nil {
		if errors.Is(err, context.Canceled) {
			reporter.update(database.JobUpdate{Status: "canceled", Phase: "canceled", Message: "Filesystem scan canceled", Error: "job canceled by admin", Completed: true}, true)
			server.record("scan.canceled", "Library scan canceled", map[string]any{"jobId": jobID})
			return
		}
		reporter.update(database.JobUpdate{Status: "failed", Phase: "failed", Message: "Filesystem scan failed", Error: err.Error(), Completed: true}, true)
		server.record("scan.failed", "Library scan failed", map[string]any{"jobId": jobID, "error": err.Error()})
		return
	}
	reporter.update(database.JobUpdate{Status: "completed", Phase: "complete", Message: "Filesystem scan completed", Completed: true}, true)
}

func (server *Server) scanAll(ctx context.Context, reporter *jobReporter) ([]filescan.Result, []recommendations.Recommendation, error) {
	server.mu.RLock()
	libraries := append([]filescan.Library(nil), server.libraries...)
	server.mu.RUnlock()

	var results []filescan.Result
	var files []recommendations.MediaFile
	for libraryIndex, library := range libraries {
		if err := ctx.Err(); err != nil {
			return nil, nil, err
		}
		if reporter != nil {
			reporter.update(database.JobUpdate{
				Phase:        "discovering",
				Message:      "Discovering files in " + displayLibraryName(library),
				CurrentLabel: displayLibraryName(library),
				Processed:    intPtr(libraryIndex),
				Total:        intPtr(len(libraries)),
			}, true)
		}
		scanner := server.scanner
		if reporter != nil {
			scanner.Progress = func(progress filescan.Progress) {
				reporter.update(database.JobUpdate{
					Phase:        progress.Phase,
					Message:      displayLibraryName(library) + ": " + progress.Message,
					CurrentLabel: progress.CurrentLabel,
					Processed:    intPtr(progress.Processed),
					Total:        intPtr(progress.Total),
				}, progress.Phase != "processing" || progress.Processed == 0 || progress.Processed == progress.Total)
			}
		}
		result, err := scanner.Scan(library)
		if err != nil {
			return nil, nil, err
		}
		if err := ctx.Err(); err != nil {
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
		if reporter != nil {
			reporter.update(database.JobUpdate{Phase: "catalog", Message: "Loading catalog for recommendations"}, true)
		}
		items, err := server.store.ListCatalog()
		if err != nil {
			return nil, nil, err
		}
		files = recommendationFilesFromCatalog(items)
	}
	if reporter != nil {
		reporter.update(database.JobUpdate{Phase: "recommendations", Message: "Generating cleanup recommendations"}, true)
	}
	if err := ctx.Err(); err != nil {
		return nil, nil, err
	}
	recs, err := server.generateRecommendations(ctx, files)
	if err != nil {
		return nil, nil, err
	}
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
	writeJSON(w, http.StatusOK, envelope{Data: emptyIfNil(server.recommendations)})
}

func (server *Server) recommendationActionHandler(w http.ResponseWriter, r *http.Request) {
	if server.store == nil {
		http.Error(w, "recommendation store is not configured", http.StatusBadRequest)
		return
	}
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/v1/recommendations/"), "/")
	if len(parts) != 2 || parts[0] == "" {
		http.NotFound(w, r)
		return
	}
	if parts[1] == "evidence" {
		if r.Method != http.MethodGet {
			methodNotAllowed(w, r)
			return
		}
		rec, err := server.store.GetRecommendation(parts[0])
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		writeJSON(w, http.StatusOK, envelope{Data: recommendations.BuildEvidence(rec)})
		return
	}
	if r.Method != http.MethodPost {
		methodNotAllowed(w, r)
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
	case "protect":
		if err := server.store.SetRecommendationState(parts[0], recommendations.StateProtected); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		server.record("recommendation.protected", "Recommendation protected", map[string]any{"id": parts[0]})
	case "accept-manual":
		if err := server.store.SetRecommendationState(parts[0], recommendations.StateAcceptedForManualAction); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		server.record("recommendation.accepted_for_manual_action", "Recommendation accepted for manual action", map[string]any{"id": parts[0]})
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

func (server *Server) integrationSettingsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w, r)
		return
	}
	if server.store == nil {
		http.Error(w, "integration settings store is not configured", http.StatusBadRequest)
		return
	}
	settings, err := server.store.ListIntegrationSettings()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, envelope{Data: settings})
}

func (server *Server) integrationSettingHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		methodNotAllowed(w, r)
		return
	}
	if server.store == nil {
		http.Error(w, "integration settings store is not configured", http.StatusBadRequest)
		return
	}
	integration := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/v1/integration-settings/"), "/")
	if integration == "" {
		http.NotFound(w, r)
		return
	}
	var request database.IntegrationSettingInput
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	request.Integration = integration
	setting, err := server.store.UpsertIntegrationSetting(request)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := server.refreshIntegrationOptionsFromStore(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	server.mu.Lock()
	server.integrations = integrations.DefaultsWithOptions(server.integrationOptions)
	server.mu.Unlock()
	server.record("integration.updated", "Media-server integration settings updated", map[string]any{"integration": setting.Integration, "apiKeyConfigured": setting.APIKeyConfigured})
	if setting.AutoSyncEnabled && setting.APIKeyConfigured && setting.BaseURL != "" {
		if job, queued, err := server.queueIntegrationSync(setting.Integration, "Initial media-server auto-sync queued"); err == nil && queued {
			server.record("integration.auto_sync_queued", "Initial media-server auto-sync queued", map[string]any{"targetId": setting.Integration, "jobId": job.ID})
		}
	}
	writeJSON(w, http.StatusOK, envelope{Data: setting})
}

func (server *Server) integrationsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w, r)
		return
	}
	if err := server.refreshIntegrationOptionsFromStore(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	server.mu.Lock()
	server.integrations = integrations.DefaultsWithOptions(server.integrationOptions)
	integrationRows := append([]integrations.Target(nil), server.integrations...)
	server.mu.Unlock()
	writeJSON(w, http.StatusOK, envelope{Data: integrationRows})
}

func (server *Server) integrationActionHandler(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/v1/integrations/"), "/")
	if len(parts) != 2 || parts[0] == "" {
		http.NotFound(w, r)
		return
	}
	switch parts[1] {
	case "refresh":
		if err := server.refreshIntegrationOptionsFromStore(); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if r.Method != http.MethodPost {
			methodNotAllowed(w, r)
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
	case "sync":
		server.integrationSyncHandler(w, r, parts[0])
	case "items":
		server.integrationItemsHandler(w, r, parts[0])
	case "diagnostics":
		server.integrationDiagnosticsHandler(w, r, parts[0])
	default:
		http.NotFound(w, r)
	}
}

func (server *Server) integrationSyncHandler(w http.ResponseWriter, r *http.Request, targetID string) {
	if server.store == nil {
		http.Error(w, "integration store is not configured", http.StatusBadRequest)
		return
	}
	switch r.Method {
	case http.MethodGet:
		if targetID == "tautulli" {
			job, err := server.store.LatestTautulliSyncJob()
			if err != nil {
				http.Error(w, err.Error(), http.StatusNotFound)
				return
			}
			writeJSON(w, http.StatusOK, envelope{Data: job})
			return
		}
		if jobs, err := server.store.ListJobs(database.JobFilter{Kind: integrationJobKind(targetID), TargetID: targetID, Active: true, Limit: 1}); err == nil && len(jobs) > 0 {
			writeJSON(w, http.StatusOK, envelope{Data: mediaSyncJobFromJob(jobs[0])})
			return
		}
		job, err := server.store.LatestMediaSyncJob(targetID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		writeJSON(w, http.StatusOK, envelope{Data: job})
	case http.MethodPost:
		job, _, err := server.queueIntegrationSync(targetID, "Media-server sync queued")
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		writeJSON(w, http.StatusAccepted, envelope{Data: mediaSyncJobFromJob(job)})
	default:
		methodNotAllowed(w, r)
	}
}

func (server *Server) QueueDueAutoSyncs() ([]database.Job, error) {
	return server.queueDueAutoSyncs(time.Now().UTC())
}

func (server *Server) StartAutoSync(ctx context.Context, checkEvery time.Duration) {
	if server.store == nil {
		return
	}
	if checkEvery <= 0 {
		checkEvery = time.Minute
	}
	go func() {
		_, _ = server.QueueDueAutoSyncs()
		ticker := time.NewTicker(checkEvery)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				_, _ = server.QueueDueAutoSyncs()
			}
		}
	}()
}

func (server *Server) queueDueAutoSyncs(now time.Time) ([]database.Job, error) {
	if server.store == nil {
		return nil, errors.New("integration store is not configured")
	}
	if err := server.refreshIntegrationOptionsFromStore(); err != nil {
		return nil, err
	}
	settings, err := server.store.ListIntegrationSettingSecrets()
	if err != nil {
		return nil, err
	}
	byIntegration := map[string]database.IntegrationSetting{}
	for _, setting := range settings {
		byIntegration[setting.Integration] = setting
	}
	targets := []struct {
		id       string
		baseURL  string
		apiKey   string
		setting  database.IntegrationSetting
		hasStore bool
	}{
		{id: "jellyfin", baseURL: server.integrationOptions.JellyfinURL, apiKey: server.integrationOptions.JellyfinKey},
		{id: "plex", baseURL: server.integrationOptions.PlexURL, apiKey: server.integrationOptions.PlexToken},
		{id: "emby", baseURL: server.integrationOptions.EmbyURL, apiKey: server.integrationOptions.EmbyKey},
	}
	queued := []database.Job{}
	for _, target := range targets {
		target.setting, target.hasStore = byIntegration[target.id]
		if strings.TrimSpace(target.baseURL) == "" || strings.TrimSpace(target.apiKey) == "" {
			continue
		}
		autoEnabled := true
		intervalMinutes := 360
		if target.hasStore {
			autoEnabled = target.setting.AutoSyncEnabled
			intervalMinutes = target.setting.AutoSyncIntervalMinutes
		}
		if !autoEnabled {
			continue
		}
		if active, err := server.store.ListJobs(database.JobFilter{Kind: integrationJobKind(target.id), TargetID: target.id, Active: true, Limit: 1}); err != nil {
			return nil, err
		} else if len(active) > 0 {
			continue
		}
		if latest, err := server.store.LatestMediaSyncJob(target.id); err == nil && latest.CompletedAt.Add(time.Duration(intervalMinutes)*time.Minute).After(now) {
			continue
		}
		job, didQueue, err := server.queueIntegrationSync(target.id, "Automatic media-server sync queued")
		if err != nil {
			return nil, err
		}
		if didQueue {
			server.record("integration.auto_sync_queued", "Automatic media-server sync queued", map[string]any{"targetId": target.id, "jobId": job.ID})
			queued = append(queued, job)
		}
	}
	return queued, nil
}

func (server *Server) queueIntegrationSync(targetID string, message string) (database.Job, bool, error) {
	if server.store == nil {
		return database.Job{}, false, errors.New("integration store is not configured")
	}
	jobKind := integrationJobKind(targetID)
	if jobKind == "" {
		return database.Job{}, false, errors.New("integration sync is not supported for this target")
	}
	if jobs, err := server.store.ListJobs(database.JobFilter{Kind: jobKind, TargetID: targetID, Active: true, Limit: 1}); err == nil && len(jobs) > 0 {
		return jobs[0], false, nil
	} else if err != nil {
		return database.Job{}, false, err
	}
	if strings.TrimSpace(message) == "" {
		message = "Media-server sync queued"
	}
	job, err := server.store.CreateJob(database.JobInput{
		Kind:     jobKind,
		TargetID: targetID,
		Phase:    "queued",
		Message:  message,
	})
	if err != nil {
		return database.Job{}, false, err
	}
	if err := server.startJob(job); err != nil {
		return database.Job{}, false, err
	}
	return job, true, nil
}

func (server *Server) runIntegrationSyncJob(jobID string, targetID string) {
	reporter := jobReporter{store: server.store, jobID: jobID}
	ctx, cancel := context.WithTimeout(context.Background(), 24*time.Hour)
	defer cancel()
	server.registerJobCancel(jobID, cancel)
	defer server.unregisterJobCancel(jobID)
	reporter.update(database.JobUpdate{Status: "running", Phase: "connecting", Message: "Starting " + targetID + " sync"}, true)

	if err := server.refreshIntegrationOptionsFromStore(); err != nil {
		reporter.update(database.JobUpdate{Status: "failed", Phase: "failed", Message: "Unable to load integration settings", Error: err.Error(), Completed: true}, true)
		return
	}
	mappings, err := server.store.ListPathMappings()
	if err != nil {
		reporter.update(database.JobUpdate{Status: "failed", Phase: "failed", Message: "Unable to load path mappings", Error: err.Error(), Completed: true}, true)
		return
	}
	options := server.integrationOptions
	if targetID == "plex" {
		if latest, err := server.store.LatestMediaSyncJob(targetID); err == nil {
			options.PlexHistoryCursor = latest.Cursor
		}
		if priorRollups, err := server.store.ListMediaActivityRollups(database.MediaActivityRollupFilter{ServerID: targetID}); err == nil {
			options.PriorRollups = priorRollups
		}
	}
	options.Progress = func(progress integrations.Progress) {
		reporter.update(database.JobUpdate{
			Phase:           progress.Phase,
			Message:         progress.Message,
			CurrentLabel:    progress.CurrentLabel,
			Processed:       intPtr(progress.Processed),
			Total:           intPtr(progress.Total),
			ItemsImported:   intPtr(progress.ItemsImported),
			RollupsImported: intPtr(progress.RollupsImported),
			UnmappedItems:   intPtr(progress.UnmappedItems),
		}, progress.Phase != "items" || progress.Processed == 0 || progress.Processed == progress.Total)
	}
	snapshot, err := syncIntegrationSnapshot(ctx, options, targetID, mappings)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			reporter.update(database.JobUpdate{Status: "canceled", Phase: "canceled", Message: "Media-server sync canceled", Error: "job canceled by admin", Completed: true}, true)
			server.record("integration.sync_canceled", "Media-server inventory and activity sync canceled", map[string]any{"targetId": targetID})
			return
		}
		reporter.update(database.JobUpdate{Status: "failed", Phase: "failed", Message: "Media-server sync failed", Error: err.Error(), Completed: true}, true)
		server.record("integration.sync_failed", "Media-server inventory and activity sync failed", map[string]any{"targetId": targetID, "error": err.Error()})
		return
	}
	if err := server.store.ReplaceMediaServerSnapshot(snapshot); err != nil {
		reporter.update(database.JobUpdate{Status: "failed", Phase: "failed", Message: "Unable to persist media-server snapshot", Error: err.Error(), Completed: true}, true)
		return
	}
	reporter.update(database.JobUpdate{
		Phase:        "recommendations",
		Message:      "Generating evidence-based recommendations",
		CurrentLabel: "Deterministic review rules",
		Processed:    intPtr(0),
		Total:        intPtr(0),
	}, true)
	if err := server.regenerateRecommendationsFromCatalog(ctx); err != nil {
		reporter.update(database.JobUpdate{Status: "failed", Phase: "failed", Message: "Unable to regenerate recommendations", Error: err.Error(), Completed: true}, true)
		return
	}
	reporter.update(database.JobUpdate{
		Status:          "completed",
		Phase:           "complete",
		Message:         "Media-server sync completed",
		ItemsImported:   intPtr(snapshot.Job.ItemsImported),
		RollupsImported: intPtr(snapshot.Job.RollupsImported),
		UnmappedItems:   intPtr(snapshot.Job.UnmappedItems),
		Completed:       true,
	}, true)
	server.record("integration.sync_completed", "Media-server inventory and activity sync completed", map[string]any{"targetId": targetID, "items": snapshot.Job.ItemsImported, "rollups": snapshot.Job.RollupsImported})
}

func (server *Server) runTautulliSyncJob(jobID string) {
	reporter := jobReporter{store: server.store, jobID: jobID}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()
	server.registerJobCancel(jobID, cancel)
	defer server.unregisterJobCancel(jobID)
	started := time.Now().UTC()
	reporter.update(database.JobUpdate{Status: "running", Phase: "history", Message: "Reading Tautulli watch history"}, true)

	if err := server.refreshIntegrationOptionsFromStore(); err != nil {
		reporter.update(database.JobUpdate{Status: "failed", Phase: "failed", Message: "Unable to load Tautulli settings", Error: err.Error(), Completed: true}, true)
		_ = server.store.RecordTautulliSyncJob(stewardship.TautulliSyncJob{ID: jobID, Status: "failed", Error: err.Error(), StartedAt: started, CompletedAt: time.Now().UTC()})
		return
	}
	plays, err := stewardship.FetchTautulliHistory(ctx, server.integrationOptions.TautulliURL, server.integrationOptions.TautulliKey)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			reporter.update(database.JobUpdate{Status: "canceled", Phase: "canceled", Message: "Tautulli sync canceled", Error: "job canceled by admin", Completed: true}, true)
			_ = server.store.RecordTautulliSyncJob(stewardship.TautulliSyncJob{ID: jobID, Status: "canceled", Error: "job canceled by admin", StartedAt: started, CompletedAt: time.Now().UTC()})
			return
		}
		reporter.update(database.JobUpdate{Status: "failed", Phase: "failed", Message: "Tautulli sync failed", Error: err.Error(), Completed: true}, true)
		_ = server.store.RecordTautulliSyncJob(stewardship.TautulliSyncJob{ID: jobID, Status: "failed", Error: err.Error(), StartedAt: started, CompletedAt: time.Now().UTC()})
		return
	}
	reporter.update(database.JobUpdate{Phase: "enrichment", Message: "Applying Tautulli activity to Plex inventory", ItemsImported: intPtr(len(plays))}, true)
	rollups, err := server.store.ListMediaActivityRollups(database.MediaActivityRollupFilter{ServerID: "plex"})
	if err != nil {
		reporter.update(database.JobUpdate{Status: "failed", Phase: "failed", Message: "Unable to load Plex activity", Error: err.Error(), Completed: true}, true)
		_ = server.store.RecordTautulliSyncJob(stewardship.TautulliSyncJob{ID: jobID, Status: "failed", Error: err.Error(), StartedAt: started, CompletedAt: time.Now().UTC()})
		return
	}
	items, err := server.store.ListMediaServerItems(database.MediaServerItemFilter{ServerID: "plex"})
	if err != nil {
		reporter.update(database.JobUpdate{Status: "failed", Phase: "failed", Message: "Unable to load Plex inventory", Error: err.Error(), Completed: true}, true)
		_ = server.store.RecordTautulliSyncJob(stewardship.TautulliSyncJob{ID: jobID, Status: "failed", Error: err.Error(), StartedAt: started, CompletedAt: time.Now().UTC()})
		return
	}
	if len(items) == 0 {
		err := errors.New("plex inventory must be synced before Tautulli enrichment")
		reporter.update(database.JobUpdate{Status: "failed", Phase: "failed", Message: "Plex inventory is missing", Error: err.Error(), Completed: true}, true)
		_ = server.store.RecordTautulliSyncJob(stewardship.TautulliSyncJob{ID: jobID, Status: "failed", Error: err.Error(), StartedAt: started, CompletedAt: time.Now().UTC()})
		return
	}
	enriched := stewardship.ApplyTautulliHistory(stewardshipRollupsFromDatabase(rollups, items), plays, 80)
	if err := server.store.ReplaceMediaActivityRollups("plex", databaseRollupsFromStewardship(enriched)); err != nil {
		reporter.update(database.JobUpdate{Status: "failed", Phase: "failed", Message: "Unable to persist Tautulli activity", Error: err.Error(), Completed: true}, true)
		_ = server.store.RecordTautulliSyncJob(stewardship.TautulliSyncJob{ID: jobID, Status: "failed", Error: err.Error(), StartedAt: started, CompletedAt: time.Now().UTC()})
		return
	}
	if err := server.regenerateRecommendationsFromCatalog(ctx); err != nil {
		reporter.update(database.JobUpdate{Status: "failed", Phase: "failed", Message: "Unable to regenerate recommendations", Error: err.Error(), Completed: true}, true)
		_ = server.store.RecordTautulliSyncJob(stewardship.TautulliSyncJob{ID: jobID, Status: "failed", Error: err.Error(), StartedAt: started, CompletedAt: time.Now().UTC()})
		return
	}
	cursor := tautulliCursor(plays)
	completed := time.Now().UTC()
	_ = server.store.RecordTautulliSyncJob(stewardship.TautulliSyncJob{ID: jobID, Status: "completed", ItemsImported: len(plays), Cursor: cursor, StartedAt: started, CompletedAt: completed})
	_, _ = server.store.CreateNotification(stewardship.Notification{
		Level:     "success",
		Title:     "Tautulli activity imported",
		Body:      strconv.Itoa(len(plays)) + " watch-history rows were applied to Plex activity.",
		EventType: "tautulli.sync",
		Fields:    map[string]string{"itemsImported": strconv.Itoa(len(plays)), "cursor": cursor},
	})
	reporter.update(database.JobUpdate{
		Status:        "completed",
		Phase:         "complete",
		Message:       "Tautulli activity sync completed",
		ItemsImported: intPtr(len(plays)),
		Completed:     true,
	}, true)
	server.record("tautulli.sync_completed", "Tautulli activity sync completed", map[string]any{"items": len(plays)})
}

func integrationJobKind(targetID string) string {
	switch strings.ToLower(strings.TrimSpace(targetID)) {
	case "jellyfin":
		return "jellyfin_sync"
	case "plex":
		return "plex_sync"
	case "emby":
		return "emby_sync"
	case "tautulli":
		return "tautulli_sync"
	default:
		return ""
	}
}

func mediaSyncJobFromJob(job database.Job) database.MediaSyncJob {
	return database.MediaSyncJob{
		ID:              job.ID,
		ServerID:        job.TargetID,
		Status:          job.Status,
		Phase:           job.Phase,
		Message:         job.Message,
		CurrentLabel:    job.CurrentLabel,
		Processed:       job.Processed,
		Total:           job.Total,
		ItemsImported:   job.ItemsImported,
		RollupsImported: job.RollupsImported,
		UnmappedItems:   job.UnmappedItems,
		Error:           job.Error,
		StartedAt:       job.StartedAt,
		CompletedAt:     job.CompletedAt,
	}
}

func stewardshipRollupsFromDatabase(rollups []database.MediaActivityRollup, items []database.MediaServerItem) []stewardship.ActivityRollup {
	byItemID := map[string]database.MediaActivityRollup{}
	for _, rollup := range rollups {
		if strings.TrimSpace(rollup.ItemExternalID) != "" {
			byItemID[rollup.ItemExternalID] = rollup
		}
	}
	output := make([]stewardship.ActivityRollup, 0, len(items))
	for _, item := range items {
		externalID := strings.TrimSpace(item.ExternalID)
		if externalID == "" {
			continue
		}
		rollup := byItemID[externalID]
		output = append(output, stewardship.ActivityRollup{
			ServerID:       firstNonEmpty(rollup.ServerID, item.ServerID, "plex"),
			ItemExternalID: externalID,
			PlayCount:      rollup.PlayCount,
			UniqueUsers:    rollup.UniqueUsers,
			WatchedUsers:   rollup.WatchedUsers,
			FavoriteCount:  rollup.FavoriteCount,
			LastPlayedAt:   rollup.LastPlayedAt,
			UpdatedAt:      rollup.UpdatedAt,
			EvidenceSource: "plex",
		})
	}
	return output
}

func databaseRollupsFromStewardship(rollups []stewardship.ActivityRollup) []database.MediaActivityRollup {
	output := make([]database.MediaActivityRollup, 0, len(rollups))
	for _, rollup := range rollups {
		output = append(output, database.MediaActivityRollup{
			ServerID:       firstNonEmpty(rollup.ServerID, "plex"),
			ItemExternalID: rollup.ItemExternalID,
			PlayCount:      rollup.PlayCount,
			UniqueUsers:    rollup.UniqueUsers,
			WatchedUsers:   rollup.WatchedUsers,
			FavoriteCount:  rollup.FavoriteCount,
			LastPlayedAt:   rollup.LastPlayedAt,
			UpdatedAt:      rollup.UpdatedAt,
		})
	}
	return output
}

func tautulliCursor(plays []stewardship.TautulliPlay) string {
	var latest int64
	for _, play := range plays {
		if unix := play.PlayedAt.Unix(); unix > latest {
			latest = unix
		}
	}
	if latest <= 0 {
		return ""
	}
	return strconv.FormatInt(latest, 10)
}

func (server *Server) integrationItemsHandler(w http.ResponseWriter, r *http.Request, targetID string) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w, r)
		return
	}
	if server.store == nil {
		http.Error(w, "integration store is not configured", http.StatusBadRequest)
		return
	}
	items, err := server.store.ListMediaServerItems(database.MediaServerItemFilter{
		ServerID:     targetID,
		UnmappedOnly: r.URL.Query().Get("unmapped") == "true",
		Limit:        parsePositiveInt(r.URL.Query().Get("limit")),
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, envelope{Data: items})
}

func (server *Server) integrationDiagnosticsHandler(w http.ResponseWriter, r *http.Request, targetID string) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w, r)
		return
	}
	if server.store == nil {
		http.Error(w, "integration store is not configured", http.StatusBadRequest)
		return
	}
	targetID = strings.TrimSpace(targetID)
	snapshot, err := server.store.GetMediaServerSnapshot(targetID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.NotFound(w, r)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	recs, err := server.store.ListRecommendations()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	filtered := make([]recommendations.Recommendation, 0, len(recs))
	for _, rec := range recs {
		if rec.ServerID == targetID {
			filtered = append(filtered, rec)
		}
	}
	report := acceptance.BuildReport(snapshot, filtered, nil, acceptance.ReportOptions{
		TargetID:                 targetID,
		GeneratedAt:              time.Now().UTC(),
		RequireLocalVerification: true,
	})
	writeJSON(w, http.StatusOK, envelope{Data: report})
}

func syncIntegrationSnapshot(ctx context.Context, options integrations.Options, targetID string, mappings []database.PathMapping) (database.MediaServerSnapshot, error) {
	switch strings.ToLower(strings.TrimSpace(targetID)) {
	case "jellyfin":
		return integrations.SyncJellyfin(ctx, options, mappings)
	case "plex":
		return integrations.SyncPlex(ctx, options, mappings)
	case "emby":
		return integrations.SyncEmby(ctx, options, mappings)
	default:
		return database.MediaServerSnapshot{}, errors.New("integration sync is not supported for this target")
	}
}

func (server *Server) activityRollupsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w, r)
		return
	}
	if server.store == nil {
		http.Error(w, "activity store is not configured", http.StatusBadRequest)
		return
	}
	rollups, err := server.store.ListMediaActivityRollups(database.MediaActivityRollupFilter{
		ServerID: r.URL.Query().Get("serverId"),
		Limit:    parsePositiveInt(r.URL.Query().Get("limit")),
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, envelope{Data: rollups})
}

func (server *Server) requestSourcesHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w, r)
		return
	}
	if server.store == nil {
		http.Error(w, "request source store is not configured", http.StatusBadRequest)
		return
	}
	sources, err := server.store.ListRequestSources()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, envelope{Data: emptyIfNil(sources)})
}

func (server *Server) requestSourceHandler(w http.ResponseWriter, r *http.Request) {
	if server.store == nil {
		http.Error(w, "request source store is not configured", http.StatusBadRequest)
		return
	}
	parts := strings.Split(strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/v1/request-sources/"), "/"), "/")
	if len(parts) == 0 || parts[0] == "" {
		http.NotFound(w, r)
		return
	}
	if len(parts) == 2 && parts[1] == "sync" {
		server.requestSourceSyncHandler(w, r, parts[0])
		return
	}
	if len(parts) != 1 {
		http.NotFound(w, r)
		return
	}
	if r.Method != http.MethodPut {
		methodNotAllowed(w, r)
		return
	}
	var request stewardship.RequestSourceInput
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	request.Kind = firstNonEmpty(request.Kind, parts[0])
	if request.Kind != parts[0] {
		http.Error(w, "request source kind does not match route", http.StatusBadRequest)
		return
	}
	source, err := server.store.UpsertRequestSource(request)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	server.record("request_source.updated", "Request source settings updated", map[string]any{"sourceId": source.ID, "apiKeyConfigured": source.APIKeyConfigured})
	writeJSON(w, http.StatusOK, envelope{Data: source})
}

func (server *Server) requestSourceSyncHandler(w http.ResponseWriter, r *http.Request, sourceID string) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w, r)
		return
	}
	source, err := server.requestSourceSecret(sourceID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	if !source.Enabled {
		http.Error(w, "request source is disabled", http.StatusBadRequest)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Minute)
	defer cancel()
	var signals []stewardship.RequestSignal
	switch source.Kind {
	case "seerr":
		signals, err = stewardship.FetchSeerrRequests(ctx, source.BaseURL, source.APIKey)
	default:
		err = errors.New("request source sync is not supported")
	}
	if err != nil {
		_, _ = server.store.CreateNotification(stewardship.Notification{Level: "error", Title: "Request-source sync failed", Body: err.Error(), EventType: "request_source.sync", Fields: map[string]string{"sourceId": source.ID}})
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := server.store.ReplaceRequestSignals(source.ID, signals); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	_, _ = server.store.CreateNotification(stewardship.Notification{Level: "success", Title: "Request-source sync completed", Body: strconv.Itoa(len(signals)) + " request signals imported.", EventType: "request_source.sync", Fields: map[string]string{"sourceId": source.ID, "imported": strconv.Itoa(len(signals))}})
	server.record("request_source.sync_completed", "Request source sync completed", map[string]any{"sourceId": source.ID, "imported": len(signals)})
	writeJSON(w, http.StatusOK, envelope{Data: map[string]any{"sourceId": source.ID, "imported": len(signals)}})
}

func (server *Server) requestSourceSecret(sourceID string) (stewardship.RequestSource, error) {
	sources, err := server.store.ListRequestSourceSecrets()
	if err != nil {
		return stewardship.RequestSource{}, err
	}
	for _, source := range sources {
		if source.ID == strings.TrimSpace(sourceID) {
			return source, nil
		}
	}
	return stewardship.RequestSource{}, sql.ErrNoRows
}

func (server *Server) requestSignalsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w, r)
		return
	}
	if server.store == nil {
		http.Error(w, "request signal store is not configured", http.StatusBadRequest)
		return
	}
	signals, err := server.store.ListRequestSignals(r.URL.Query().Get("sourceId"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, envelope{Data: emptyIfNil(signals)})
}

func (server *Server) campaignTemplatesHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w, r)
		return
	}
	writeJSON(w, http.StatusOK, envelope{Data: stewardship.BuiltInCampaignTemplates()})
}

func (server *Server) campaignTemplateHandler(w http.ResponseWriter, r *http.Request) {
	if server.store == nil {
		http.Error(w, "campaign store is not configured", http.StatusBadRequest)
		return
	}
	parts := strings.Split(strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/v1/campaign-templates/"), "/"), "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] != "create" {
		http.NotFound(w, r)
		return
	}
	if r.Method != http.MethodPost {
		methodNotAllowed(w, r)
		return
	}
	ref, err := stewardship.CampaignFromTemplate(parts[0])
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	campaign, err := server.store.UpsertCampaign(campaignFromTemplate(parts[0], ref))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	server.record("campaign.template_created", "Stewardship campaign created from template", map[string]any{"templateId": parts[0], "campaignId": campaign.ID})
	writeJSON(w, http.StatusCreated, envelope{Data: campaign})
}

func (server *Server) campaignsHandler(w http.ResponseWriter, r *http.Request) {
	if server.store == nil {
		http.Error(w, "campaign store is not configured", http.StatusBadRequest)
		return
	}
	switch r.Method {
	case http.MethodGet:
		campaignList, err := server.store.ListCampaigns()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, envelope{Data: emptyIfNil(campaignList)})
	case http.MethodPost:
		var request campaigns.Campaign
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		campaign, err := server.store.UpsertCampaign(request)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		server.record("campaign.created", "Stewardship campaign saved", map[string]any{"id": campaign.ID, "name": campaign.Name})
		writeJSON(w, http.StatusCreated, envelope{Data: campaign})
	default:
		methodNotAllowed(w, r)
	}
}

func (server *Server) campaignHandler(w http.ResponseWriter, r *http.Request) {
	if server.store == nil {
		http.Error(w, "campaign store is not configured", http.StatusBadRequest)
		return
	}
	parts := strings.Split(strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/v1/campaigns/"), "/"), "/")
	if len(parts) == 0 || parts[0] == "" {
		http.NotFound(w, r)
		return
	}
	id := parts[0]
	if len(parts) == 2 {
		switch parts[1] {
		case "simulate":
			server.simulateCampaignHandler(w, r, id)
		case "what-if":
			server.campaignWhatIfHandler(w, r, id)
		case "run":
			server.runCampaignHandler(w, r, id)
		case "runs":
			server.campaignRunsHandler(w, r, id)
		case "publish-preview":
			server.campaignPublicationHandler(w, r, id, true)
		case "publish":
			server.campaignPublicationHandler(w, r, id, false)
		default:
			http.NotFound(w, r)
		}
		return
	}
	if len(parts) != 1 {
		http.NotFound(w, r)
		return
	}
	switch r.Method {
	case http.MethodGet:
		campaign, err := server.store.GetCampaign(id)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		writeJSON(w, http.StatusOK, envelope{Data: campaign})
	case http.MethodPut:
		var request campaigns.Campaign
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		request.ID = id
		campaign, err := server.store.UpsertCampaign(request)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		server.record("campaign.updated", "Stewardship campaign updated", map[string]any{"id": campaign.ID, "name": campaign.Name})
		writeJSON(w, http.StatusOK, envelope{Data: campaign})
	case http.MethodDelete:
		if err := server.store.DeleteCampaign(id); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		server.record("campaign.deleted", "Stewardship campaign deleted", map[string]any{"id": id})
		writeJSON(w, http.StatusOK, envelope{Data: map[string]bool{"ok": true}})
	default:
		methodNotAllowed(w, r)
	}
}

func (server *Server) simulateCampaignHandler(w http.ResponseWriter, r *http.Request, id string) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w, r)
		return
	}
	_, result, err := server.simulateCampaign(id, time.Now().UTC())
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, sql.ErrNoRows) {
			status = http.StatusNotFound
		}
		http.Error(w, err.Error(), status)
		return
	}
	writeJSON(w, http.StatusOK, envelope{Data: result})
}

func (server *Server) campaignWhatIfHandler(w http.ResponseWriter, r *http.Request, id string) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w, r)
		return
	}
	_, result, err := server.simulateCampaign(id, time.Now().UTC())
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, sql.ErrNoRows) {
			status = http.StatusNotFound
		}
		http.Error(w, err.Error(), status)
		return
	}
	whatIf, err := server.whatIfFromCampaignResult(id, result)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, envelope{Data: whatIf})
}

func (server *Server) campaignPublicationHandler(w http.ResponseWriter, r *http.Request, id string, dryRun bool) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w, r)
		return
	}
	var request stewardship.PublicationInput
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if !dryRun && !request.ConfirmPublish {
		http.Error(w, "collection publish requires confirmPublish=true", http.StatusBadRequest)
		return
	}
	request.CampaignID = id
	_, result, err := server.simulateCampaign(id, time.Now().UTC())
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, sql.ErrNoRows) {
			status = http.StatusNotFound
		}
		http.Error(w, err.Error(), status)
		return
	}
	request.Items = publicationCandidatesFromCampaign(result)
	plan := stewardship.PlanLeavingSoonPublication(request)
	if !dryRun {
		plan.DryRun = false
		plan.Status = "ready"
	}
	saved, err := server.store.RecordCollectionPublication(plan)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if dryRun {
		writeJSON(w, http.StatusOK, envelope{Data: saved})
		return
	}
	published, err := server.publishCollection(saved)
	if err != nil {
		failed, _ := server.store.MarkCollectionPublicationPublished(saved.ID, "failed", err.Error())
		http.Error(w, failed.Error, http.StatusBadRequest)
		return
	}
	writeJSON(w, http.StatusAccepted, envelope{Data: published})
}

func (server *Server) runCampaignHandler(w http.ResponseWriter, r *http.Request, id string) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w, r)
		return
	}
	started := time.Now().UTC()
	campaign, result, err := server.simulateCampaign(id, started)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, sql.ErrNoRows) {
			status = http.StatusNotFound
		}
		http.Error(w, err.Error(), status)
		return
	}
	run := campaigns.Run{
		ID:                    randomCampaignRunID(started),
		CampaignID:            campaign.ID,
		Status:                "completed",
		Matched:               result.Matched,
		Suppressed:            result.Suppressed,
		EstimatedSavingsBytes: result.TotalEstimatedSavingsBytes,
		VerifiedSavingsBytes:  result.TotalVerifiedSavingsBytes,
		StartedAt:             started,
		CompletedAt:           time.Now().UTC(),
	}
	recs := campaignRecommendations(campaign, run.ID, result)
	if err := server.store.RecordCampaignRun(run); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := server.store.ReplaceCampaignRecommendations(campaign.ID, recs); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	server.mu.Lock()
	server.recommendations = recs
	server.mu.Unlock()
	server.record("campaign.run", "Stewardship campaign run completed", map[string]any{"id": campaign.ID, "matched": result.Matched, "suppressed": result.Suppressed})
	writeJSON(w, http.StatusOK, envelope{Data: map[string]any{"run": run, "result": result}})
}

func (server *Server) campaignRunsHandler(w http.ResponseWriter, r *http.Request, id string) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w, r)
		return
	}
	runs, err := server.store.ListCampaignRuns(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, envelope{Data: emptyIfNil(runs)})
}

func (server *Server) simulateCampaign(id string, now time.Time) (campaigns.Campaign, campaigns.Result, error) {
	campaign, err := server.store.GetCampaign(id)
	if err != nil {
		return campaigns.Campaign{}, campaigns.Result{}, err
	}
	activityMedia, err := server.store.ListActivityRecommendationMedia()
	if err != nil {
		return campaigns.Campaign{}, campaigns.Result{}, err
	}
	candidates := campaigns.CandidatesFromActivity(activityMedia, now)
	return campaign, campaigns.Simulate(campaign, candidates, now), nil
}

func campaignRecommendations(campaign campaigns.Campaign, runID string, result campaigns.Result) []recommendations.Recommendation {
	recs := make([]recommendations.Recommendation, 0, result.Matched)
	for _, item := range result.Items {
		if item.Suppressed {
			continue
		}
		recs = append(recs, campaigns.RecommendationForMatch(campaign, runID, item))
	}
	sort.SliceStable(recs, func(i, j int) bool {
		if recs[i].SpaceSavedBytes == recs[j].SpaceSavedBytes {
			return recs[i].ID < recs[j].ID
		}
		return recs[i].SpaceSavedBytes > recs[j].SpaceSavedBytes
	})
	return recs
}

func campaignFromTemplate(templateID string, ref stewardship.TemplateCampaignRef) campaigns.Campaign {
	rules := make([]campaigns.Rule, 0, len(ref.Rules))
	for _, rule := range ref.Rules {
		rules = append(rules, campaigns.Rule{
			Field:    campaigns.Field(rule.Field),
			Operator: campaigns.Operator(rule.Operator),
			Value:    rule.Value,
			Values:   append([]string(nil), rule.Values...),
		})
	}
	id := strings.TrimSpace(ref.ID)
	if id == "" {
		id = "campaign_template_" + strings.ReplaceAll(strings.ToLower(strings.TrimSpace(templateID)), "-", "_")
	}
	return campaigns.Campaign{
		ID:                  id,
		Name:                ref.Name,
		Description:         ref.Description,
		Enabled:             ref.Enabled,
		TargetKinds:         append([]string(nil), ref.TargetKinds...),
		TargetLibraryNames:  append([]string(nil), ref.TargetLibraryNames...),
		Rules:               rules,
		RequireAllRules:     ref.RequireAllRules,
		MinimumConfidence:   ref.MinimumConfidence,
		MinimumStorageBytes: ref.MinimumStorageBytes,
	}
}

func (server *Server) whatIfFromCampaignResult(campaignID string, result campaigns.Result) (stewardship.WhatIfSimulation, error) {
	signals, err := server.store.ListRequestSignals("")
	if err != nil {
		return stewardship.WhatIfSimulation{}, err
	}
	protectionRequests, err := server.store.ListProtectionRequests("")
	if err != nil {
		return stewardship.WhatIfSimulation{}, err
	}
	requestTitles := map[string]bool{}
	for _, signal := range signals {
		if signal.Status == "declined" {
			continue
		}
		requestTitles[strings.ToLower(strings.TrimSpace(signal.Title))] = true
	}
	protectedTitles := map[string]bool{}
	for _, request := range protectionRequests {
		if request.Status != "pending" && request.Status != "approved" {
			continue
		}
		protectedTitles[strings.ToLower(strings.TrimSpace(request.Title))] = true
	}
	simulation := stewardship.CampaignSimulation{
		CampaignID:      campaignID,
		Matched:         result.Matched,
		Suppressed:      result.Suppressed,
		EstimatedBytes:  result.TotalEstimatedSavingsBytes,
		VerifiedBytes:   result.TotalVerifiedSavingsBytes,
		BlockedUnmapped: 0,
	}
	for _, item := range result.Items {
		if item.Suppressed {
			continue
		}
		title := strings.ToLower(strings.TrimSpace(item.Candidate.Title))
		if title != "" && requestTitles[title] {
			simulation.RequestConflicts++
		}
		if title != "" && protectedTitles[title] {
			simulation.ProtectionConflicts++
		}
		if item.Candidate.Verification == "" || item.Candidate.Verification == "unmapped" {
			simulation.BlockedUnmapped++
		}
	}
	return stewardship.BuildWhatIfSimulation([]stewardship.CampaignSimulation{simulation}), nil
}

func publicationCandidatesFromCampaign(result campaigns.Result) []stewardship.PublicationCandidate {
	candidates := make([]stewardship.PublicationCandidate, 0, result.Matched)
	for _, item := range result.Items {
		if item.Suppressed {
			continue
		}
		candidate := item.Candidate
		candidates = append(candidates, stewardship.PublicationCandidate{
			ExternalItemID: candidate.ExternalItemID,
			Title:          candidate.Title,
			Verification:   candidate.Verification,
			EstimatedBytes: candidate.EstimatedSavingsBytes,
		})
	}
	return candidates
}

func (server *Server) publishCollection(plan stewardship.PublicationPlan) (stewardship.PublicationPlan, error) {
	if plan.PublishableItems == 0 {
		return stewardship.PublicationPlan{}, errors.New("publication has no verified items to publish")
	}
	if err := server.refreshIntegrationOptionsFromStore(); err != nil {
		return stewardship.PublicationPlan{}, err
	}
	itemIDs := make([]string, 0, plan.PublishableItems)
	for _, item := range plan.Items {
		if item.Publishable && strings.TrimSpace(item.ExternalItemID) != "" {
			itemIDs = append(itemIDs, strings.TrimSpace(item.ExternalItemID))
		}
	}
	switch strings.ToLower(strings.TrimSpace(plan.ServerID)) {
	case "jellyfin":
		if err := server.publishJellyfinCollection(plan.CollectionTitle, itemIDs); err != nil {
			return stewardship.PublicationPlan{}, err
		}
	default:
		return stewardship.PublicationPlan{}, errors.New("collection publishing is currently supported for Jellyfin only")
	}
	return server.store.MarkCollectionPublicationPublished(plan.ID, "published", "")
}

func (server *Server) publishJellyfinCollection(title string, itemIDs []string) error {
	baseURL := strings.TrimRight(strings.TrimSpace(server.integrationOptions.JellyfinURL), "/")
	token := strings.TrimSpace(server.integrationOptions.JellyfinKey)
	if baseURL == "" || token == "" {
		return errors.New("jellyfin is not configured")
	}
	values := url.Values{}
	values.Set("Name", strings.TrimSpace(title))
	values.Set("Ids", strings.Join(itemIDs, ","))
	req, err := http.NewRequest(http.MethodPost, baseURL+"/Collections?"+values.Encode(), nil)
	if err != nil {
		return err
	}
	req.Header.Set("X-Emby-Token", token)
	res, err := (&http.Client{Timeout: 20 * time.Second}).Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode > 299 {
		return errors.New("jellyfin collection publish failed with status " + res.Status)
	}
	return nil
}

func (server *Server) storageLedgerHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w, r)
		return
	}
	if server.store == nil {
		http.Error(w, "storage ledger store is not configured", http.StatusBadRequest)
		return
	}
	ledger, err := server.store.StorageLedger()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, envelope{Data: ledger})
}

func (server *Server) notificationsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w, r)
		return
	}
	if server.store == nil {
		http.Error(w, "notification store is not configured", http.StatusBadRequest)
		return
	}
	notifications, err := server.store.ListNotifications(r.URL.Query().Get("includeRead") == "true", parsePositiveInt(r.URL.Query().Get("limit")))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, envelope{Data: emptyIfNil(notifications)})
}

func (server *Server) notificationHandler(w http.ResponseWriter, r *http.Request) {
	if server.store == nil {
		http.Error(w, "notification store is not configured", http.StatusBadRequest)
		return
	}
	parts := strings.Split(strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/v1/notifications/"), "/"), "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] != "read" {
		http.NotFound(w, r)
		return
	}
	if r.Method != http.MethodPost {
		methodNotAllowed(w, r)
		return
	}
	notification, err := server.store.MarkNotificationRead(parts[0])
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	writeJSON(w, http.StatusOK, envelope{Data: notification})
}

func (server *Server) protectionRequestsHandler(w http.ResponseWriter, r *http.Request) {
	if server.store == nil {
		http.Error(w, "protection request store is not configured", http.StatusBadRequest)
		return
	}
	switch r.Method {
	case http.MethodGet:
		requests, err := server.store.ListProtectionRequests(r.URL.Query().Get("status"))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, envelope{Data: emptyIfNil(requests)})
	case http.MethodPost:
		var request stewardship.ProtectionRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		created, err := server.store.CreateProtectionRequest(request)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		server.record("protection.requested", "Protection request created", map[string]any{"id": created.ID, "title": created.Title})
		writeJSON(w, http.StatusCreated, envelope{Data: created})
	default:
		methodNotAllowed(w, r)
	}
}

func (server *Server) protectionRequestHandler(w http.ResponseWriter, r *http.Request) {
	if server.store == nil {
		http.Error(w, "protection request store is not configured", http.StatusBadRequest)
		return
	}
	parts := strings.Split(strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/v1/protection-requests/"), "/"), "/")
	if len(parts) != 2 || parts[0] == "" {
		http.NotFound(w, r)
		return
	}
	if r.Method != http.MethodPost {
		methodNotAllowed(w, r)
		return
	}
	var decision struct {
		DecisionBy string `json:"decisionBy"`
		Note       string `json:"note"`
	}
	if r.Body != nil {
		_ = json.NewDecoder(r.Body).Decode(&decision)
	}
	approve := false
	switch parts[1] {
	case "approve":
		approve = true
	case "decline":
		approve = false
	default:
		http.NotFound(w, r)
		return
	}
	request, err := server.store.DecideProtectionRequest(parts[0], approve, decision.DecisionBy, decision.Note)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	server.record("protection.decided", "Protection request decided", map[string]any{"id": request.ID, "status": request.Status})
	writeJSON(w, http.StatusOK, envelope{Data: request})
}

func (server *Server) pathMappingsHandler(w http.ResponseWriter, r *http.Request) {
	if server.store == nil {
		http.Error(w, "path mapping store is not configured", http.StatusBadRequest)
		return
	}
	switch r.Method {
	case http.MethodGet:
		mappings, err := server.store.ListPathMappings()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, envelope{Data: mappings})
	case http.MethodPost:
		var request database.PathMapping
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		mapping, err := server.store.UpsertPathMapping(request)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		server.record("path_mapping.upserted", "Integration path mapping upserted", map[string]any{"id": mapping.ID, "serverId": mapping.ServerID})
		writeJSON(w, http.StatusCreated, envelope{Data: mapping})
	default:
		methodNotAllowed(w, r)
	}
}

func (server *Server) pathMappingHandler(w http.ResponseWriter, r *http.Request) {
	if server.store == nil {
		http.Error(w, "path mapping store is not configured", http.StatusBadRequest)
		return
	}
	parts := strings.Split(strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/v1/path-mappings/"), "/"), "/")
	if len(parts) == 0 || parts[0] == "" {
		http.NotFound(w, r)
		return
	}
	if parts[0] == "unmapped" {
		if r.Method != http.MethodGet {
			methodNotAllowed(w, r)
			return
		}
		items, err := server.store.ListMediaServerItems(database.MediaServerItemFilter{
			ServerID:     r.URL.Query().Get("serverId"),
			UnmappedOnly: true,
			Limit:        parsePositiveInt(r.URL.Query().Get("limit")),
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, envelope{Data: items})
		return
	}
	id := parts[0]
	if len(parts) == 2 && parts[1] == "verify" {
		if r.Method != http.MethodPost {
			methodNotAllowed(w, r)
			return
		}
		result, err := server.store.VerifyPathMapping(id)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		server.record("path_mapping.verified", "Integration path mapping verified", map[string]any{"id": id, "matchedFiles": result.MatchedFiles, "verifiedFiles": result.VerifiedFiles, "missingFiles": result.MissingFiles})
		writeJSON(w, http.StatusOK, envelope{Data: result})
		return
	}
	if len(parts) != 1 {
		http.NotFound(w, r)
		return
	}
	switch r.Method {
	case http.MethodPut:
		var request database.PathMapping
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		request.ID = id
		mapping, err := server.store.UpsertPathMapping(request)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		server.record("path_mapping.upserted", "Integration path mapping upserted", map[string]any{"id": mapping.ID, "serverId": mapping.ServerID})
		writeJSON(w, http.StatusOK, envelope{Data: mapping})
	case http.MethodDelete:
		if err := server.store.DeletePathMapping(id); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		server.record("path_mapping.deleted", "Integration path mapping deleted", map[string]any{"id": id})
		writeJSON(w, http.StatusOK, envelope{Data: map[string]bool{"ok": true}})
	default:
		methodNotAllowed(w, r)
	}
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
	switch r.Method {
	case http.MethodGet:
		if server.configDir == "" {
			http.Error(w, "config directory not configured", http.StatusBadRequest)
			return
		}
		backups, err := database.ListBackups(filepath.Join(server.configDir, "backups"))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, envelope{Data: backups})
	case http.MethodPost:
		if server.configDir == "" {
			http.Error(w, "config directory not configured", http.StatusBadRequest)
			return
		}
		path, err := database.CreateBackup(server.configDir, filepath.Join(server.configDir, "backups"))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		info, err := database.BackupInfoForPath(path)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		server.record("backup.created", "Backup created", map[string]any{"path": path})
		writeJSON(w, http.StatusCreated, envelope{Data: info})
	default:
		methodNotAllowed(w, r)
	}
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
		Name           string `json:"name"`
		Path           string `json:"path"`
		DryRun         bool   `json:"dryRun"`
		ConfirmRestore bool   `json:"confirmRestore"`
	}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	locator := firstNonEmpty(request.Name, request.Path)
	if locator == "" {
		http.Error(w, "backup path is required", http.StatusBadRequest)
		return
	}
	backupPath, err := database.ResolveBackupPath(filepath.Join(server.configDir, "backups"), locator)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if request.DryRun {
		entries, err := database.InspectBackup(backupPath)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		writeJSON(w, http.StatusOK, envelope{Data: map[string]any{"entries": entries}})
		return
	}
	if !request.ConfirmRestore {
		http.Error(w, "restore confirmation is required", http.StatusBadRequest)
		return
	}
	result, err := database.RestoreBackup(server.configDir, backupPath, filepath.Join(server.configDir, "backups"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	server.record("backup.restored", "Backup restored", map[string]any{"path": backupPath, "preRestoreBackup": result.PreRestoreBackup})
	writeJSON(w, http.StatusOK, envelope{Data: result})
}

func (server *Server) backupFileHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/api/v1/backups/restore" {
		server.backupRestoreHandler(w, r)
		return
	}
	if r.Method != http.MethodGet {
		methodNotAllowed(w, r)
		return
	}
	if server.configDir == "" {
		http.Error(w, "config directory not configured", http.StatusBadRequest)
		return
	}
	escaped := strings.TrimPrefix(r.URL.EscapedPath(), "/api/v1/backups/")
	name, err := url.PathUnescape(escaped)
	if err != nil || strings.TrimSpace(name) == "" {
		http.Error(w, "invalid backup name", http.StatusBadRequest)
		return
	}
	path, err := database.ResolveBackupPath(filepath.Join(server.configDir, "backups"), name)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			http.Error(w, "backup not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	filename := filepath.Base(path)
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": filename}))
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	http.ServeFile(w, r, path)
}

func (server *Server) supportBundlesHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		if server.configDir == "" {
			http.Error(w, "config directory not configured", http.StatusBadRequest)
			return
		}
		bundles, err := support.ListBundles(filepath.Join(server.configDir, "support"))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, envelope{Data: bundles})
	case http.MethodPost:
		if server.configDir == "" {
			http.Error(w, "config directory not configured", http.StatusBadRequest)
			return
		}
		if server.store == nil {
			http.Error(w, "database store not configured", http.StatusBadRequest)
			return
		}
		result, err := support.CreateBundle(support.Config{
			Store:     server.store,
			OutputDir: filepath.Join(server.configDir, "support"),
			Service:   "mediarr",
			Version:   "1.6.0",
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		server.record("support_bundle.created", "Support bundle created", map[string]any{"path": result.Path, "files": len(result.Files), "sizeBytes": result.SizeBytes})
		writeJSON(w, http.StatusCreated, envelope{Data: result})
	default:
		methodNotAllowed(w, r)
	}
}

func (server *Server) supportBundleFileHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w, r)
		return
	}
	if server.configDir == "" {
		http.Error(w, "config directory not configured", http.StatusBadRequest)
		return
	}
	escaped := strings.TrimPrefix(r.URL.EscapedPath(), "/api/v1/support/bundles/")
	name, err := url.PathUnescape(escaped)
	if err != nil || strings.TrimSpace(name) == "" {
		http.Error(w, "invalid support bundle name", http.StatusBadRequest)
		return
	}
	path, err := support.ResolveBundlePath(filepath.Join(server.configDir, "support"), name)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			http.Error(w, "support bundle not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	filename := filepath.Base(path)
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": filename}))
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	http.ServeFile(w, r, path)
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

func (server *Server) refreshIntegrationOptionsFromStore() error {
	if server.store == nil {
		return nil
	}
	settings, err := server.store.ListIntegrationSettingSecrets()
	if err != nil {
		return err
	}
	server.integrationOptions = mergeIntegrationSettings(server.integrationBase, settings)
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

func mergeIntegrationSettings(options integrations.Options, settings []database.IntegrationSetting) integrations.Options {
	for _, setting := range settings {
		switch setting.Integration {
		case "jellyfin":
			if setting.BaseURL != "" {
				options.JellyfinURL = setting.BaseURL
			}
			if setting.APIKey != "" {
				options.JellyfinKey = setting.APIKey
			}
		case "plex":
			if setting.BaseURL != "" {
				options.PlexURL = setting.BaseURL
			}
			if setting.APIKey != "" {
				options.PlexToken = setting.APIKey
			}
		case "emby":
			if setting.BaseURL != "" {
				options.EmbyURL = setting.BaseURL
			}
			if setting.APIKey != "" {
				options.EmbyKey = setting.APIKey
			}
		case "tautulli":
			if setting.BaseURL != "" {
				options.TautulliURL = setting.BaseURL
			}
			if setting.APIKey != "" {
				options.TautulliKey = setting.APIKey
			}
		}
	}
	return options
}

func (server *Server) regenerateRecommendationsFromCatalog(ctx context.Context) error {
	if server.store == nil {
		return nil
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	items, err := server.store.ListCatalog()
	if err != nil {
		return err
	}
	recs, err := server.generateRecommendations(ctx, recommendationFilesFromCatalog(items))
	if err != nil {
		return err
	}
	if err := server.store.ReplaceRecommendations(recs); err != nil {
		return err
	}
	server.mu.Lock()
	server.recommendations = recs
	server.mu.Unlock()
	return nil
}

func (server *Server) generateRecommendations(ctx context.Context, files []recommendations.MediaFile) ([]recommendations.Recommendation, error) {
	recs := server.engine.Generate(files)
	if server.store != nil {
		activityMedia, err := server.store.ListActivityRecommendationMedia()
		if err != nil {
			return nil, err
		}
		recs = append(recs, server.engine.GenerateActivity(activityMedia, time.Now().UTC())...)
	}
	sort.Slice(recs, func(i, j int) bool {
		if recs[i].SpaceSavedBytes == recs[j].SpaceSavedBytes {
			return recs[i].ID < recs[j].ID
		}
		return recs[i].SpaceSavedBytes > recs[j].SpaceSavedBytes
	})
	return server.enrichRecommendationsWithAI(ctx, recs), nil
}

func (server *Server) enrichRecommendationsWithAI(ctx context.Context, recs []recommendations.Recommendation) []recommendations.Recommendation {
	if server.ai == nil || len(recs) == 0 {
		return recs
	}
	if len(recs) > maxAIEnrichedRecommendations {
		return recs
	}
	health := server.ai.Health(ctx)
	if !health.ModelAvailable {
		return recs
	}
	enriched := append([]recommendations.Recommendation(nil), recs...)
	enrichedCount := 0
	for index := range enriched {
		if enriched[index].Destructive || !strings.HasPrefix(enriched[index].Source, "rule:") {
			continue
		}
		if enrichedCount >= maxAIEnrichedRecommendations {
			continue
		}
		itemCtx, cancel := context.WithTimeout(ctx, aiRecommendationTimeout)
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
		enrichedCount++
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

func (server *Server) setSessionCookie(w http.ResponseWriter, r *http.Request, session auth.Session) {
	http.SetCookie(w, &http.Cookie{
		Name:     auth.SessionCookieName,
		Value:    session.Token,
		Path:     "/",
		Expires:  session.ExpiresAt,
		MaxAge:   int(time.Until(session.ExpiresAt).Seconds()),
		Secure:   requestIsHTTPS(r),
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
}

func requestIsHTTPS(r *http.Request) bool {
	if r == nil {
		return false
	}
	if r.TLS != nil {
		return true
	}
	forwarded := strings.ToLower(r.Header.Get("X-Forwarded-Proto"))
	for _, value := range strings.Split(forwarded, ",") {
		if strings.TrimSpace(value) == "https" {
			return true
		}
	}
	return strings.EqualFold(strings.TrimSpace(r.Header.Get("X-Forwarded-Ssl")), "on")
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

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func randomCampaignRunID(started time.Time) string {
	if started.IsZero() {
		started = time.Now().UTC()
	}
	return "campaign_run_" + strconv.FormatInt(started.UnixNano(), 36)
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

type jobReporter struct {
	store *database.Store
	jobID string
}

func (reporter jobReporter) update(update database.JobUpdate, event bool) database.Job {
	if reporter.store == nil || reporter.jobID == "" {
		return database.Job{}
	}
	current, err := reporter.store.GetJob(reporter.jobID)
	if err == nil && isTerminalJobStatus(current.Status) && strings.TrimSpace(update.Status) != current.Status {
		return current
	}
	job, err := reporter.store.UpdateJob(reporter.jobID, update)
	if err != nil {
		return database.Job{}
	}
	if event {
		level := "info"
		if job.Status == "failed" {
			level = "error"
		}
		_, _ = reporter.store.AddJobEvent(database.JobEventInput{
			JobID:        reporter.jobID,
			Level:        level,
			Phase:        job.Phase,
			Message:      job.Message,
			CurrentLabel: job.CurrentLabel,
			Processed:    job.Processed,
			Total:        job.Total,
		})
	}
	return job
}

func isTerminalJobStatus(status string) bool {
	switch status {
	case "completed", "failed", "canceled", "stale":
		return true
	default:
		return false
	}
}

func intPtr(value int) *int {
	return &value
}

func parsePositiveInt(value string) int {
	parsed, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil || parsed <= 0 {
		return 0
	}
	return parsed
}

func displayLibraryName(library filescan.Library) string {
	if strings.TrimSpace(library.Name) != "" {
		return strings.TrimSpace(library.Name)
	}
	if strings.TrimSpace(library.ID) != "" {
		return strings.TrimSpace(library.ID)
	}
	return "library"
}

func emptyIfNil[T any](items []T) []T {
	if items == nil {
		return []T{}
	}
	return items
}
