package api

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/Fishy97/mediarr/backend/internal/audit"
	"github.com/Fishy97/mediarr/backend/internal/database"
	"github.com/Fishy97/mediarr/backend/internal/filescan"
	"github.com/Fishy97/mediarr/backend/internal/integrations"
	"github.com/Fishy97/mediarr/backend/internal/metadata"
	"github.com/Fishy97/mediarr/backend/internal/recommendations"
)

type Deps struct {
	ConfigDir       string
	FrontendDir     string
	Libraries       []filescan.Library
	Recommendations []recommendations.Recommendation
	Providers       []metadata.Provider
	Integrations    []integrations.Target
	Audit           *audit.Logger
	Scanner         filescan.Scanner
	Engine          recommendations.Engine
	Store           *database.Store
}

type Server struct {
	mux             *http.ServeMux
	configDir       string
	frontendDir     string
	mu              sync.RWMutex
	libraries       []filescan.Library
	scans           []filescan.Result
	recommendations []recommendations.Recommendation
	providers       []metadata.Provider
	integrations    []integrations.Target
	audit           *audit.Logger
	scanner         filescan.Scanner
	engine          recommendations.Engine
	store           *database.Store
}

func NewServer(deps Deps) *Server {
	server := &Server{
		mux:             http.NewServeMux(),
		configDir:       deps.ConfigDir,
		frontendDir:     deps.FrontendDir,
		libraries:       deps.Libraries,
		recommendations: deps.Recommendations,
		providers:       deps.Providers,
		integrations:    deps.Integrations,
		audit:           deps.Audit,
		scanner:         deps.Scanner,
		engine:          deps.Engine,
		store:           deps.Store,
	}
	if server.providers == nil {
		server.providers = metadata.Defaults()
	}
	if server.integrations == nil {
		server.integrations = integrations.Defaults()
	}
	server.routes()
	return server
}

func (server *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	server.mux.ServeHTTP(w, r)
}

func (server *Server) routes() {
	server.mux.HandleFunc("/api/v1/health", server.health)
	server.mux.HandleFunc("/api/v1/libraries", server.librariesHandler)
	server.mux.HandleFunc("/api/v1/catalog", server.catalogHandler)
	server.mux.HandleFunc("/api/v1/scans", server.scansHandler)
	server.mux.HandleFunc("/api/v1/recommendations", server.recommendationsHandler)
	server.mux.HandleFunc("/api/v1/providers", server.providersHandler)
	server.mux.HandleFunc("/api/v1/integrations", server.integrationsHandler)
	server.mux.HandleFunc("/api/v1/backups", server.backupsHandler)
	server.mux.HandleFunc("/api/v1/audit", server.auditHandler)
	server.mux.HandleFunc("/api/v1/media/files/", methodNotAllowed)
	server.mux.HandleFunc("/", server.frontend)
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
		results, recs, err := server.scanAll()
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

func (server *Server) scanAll() ([]filescan.Result, []recommendations.Recommendation, error) {
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
		for _, item := range result.Items {
			files = append(files, recommendations.MediaFile{
				ID:           item.ID,
				CanonicalKey: item.Parsed.CanonicalKey,
				Path:         item.Path,
				SizeBytes:    item.SizeBytes,
				Quality:      item.Parsed.Quality,
			})
		}
	}
	recs := server.engine.Generate(files)
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

func (server *Server) providersHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w, r)
		return
	}
	health := make([]metadata.Health, 0, len(server.providers))
	for _, provider := range server.providers {
		health = append(health, provider.Health(r.Context()))
	}
	writeJSON(w, http.StatusOK, envelope{Data: health})
}

func (server *Server) integrationsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w, r)
		return
	}
	writeJSON(w, http.StatusOK, envelope{Data: server.integrations})
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
