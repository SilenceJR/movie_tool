package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"movie-tool/backend/internal/ai"
	"movie-tool/backend/internal/automation"
	"movie-tool/backend/internal/catalog"
	"movie-tool/backend/internal/config"
	"movie-tool/backend/internal/download"
	"movie-tool/backend/internal/integration"
	"movie-tool/backend/internal/library"
	"movie-tool/backend/internal/localization"
	"movie-tool/backend/internal/media"
	"movie-tool/backend/internal/metadata"
	"movie-tool/backend/internal/nfo"
	"movie-tool/backend/internal/organizer"
	"movie-tool/backend/internal/scanner"
	"movie-tool/backend/internal/scraper"
	"movie-tool/backend/internal/strm"
	"movie-tool/backend/internal/task"
)

type Server struct {
	cfg          config.Config
	mux          *http.ServeMux
	ai           ai.Store
	automations  automation.Store
	catalog      catalog.Store
	downloads    download.Store
	integrations integration.Store
	libraries    library.Store
	localization localization.Store
	mediaFiles   media.Store
	organizer    organizer.Store
	scraper      scraper.Store
	strm         strm.Store
	tasks        *task.Queue
	scanDB       transactionBeginner
}

type Dependencies struct {
	AI           ai.Store
	Automations  automation.Store
	Catalog      catalog.Store
	Downloads    download.Store
	Integrations integration.Store
	Libraries    library.Store
	Localization localization.Store
	MediaFiles   media.Store
	Organizer    organizer.Store
	Scraper      scraper.Store
	STRM         strm.Store
	Tasks        *task.Queue
	ScanDB       transactionBeginner
}

type transactionBeginner interface {
	BeginTx(context.Context, *sql.TxOptions) (*sql.Tx, error)
}

func NewServer(cfg config.Config) *Server {
	return NewServerWithDependencies(cfg, Dependencies{})
}

func NewServerWithDependencies(cfg config.Config, deps Dependencies) *Server {
	if deps.AI == nil {
		deps.AI = ai.NewMemoryStore()
	}
	if deps.Automations == nil {
		deps.Automations = automation.NewMemoryStore()
	}
	if deps.Catalog == nil {
		deps.Catalog = catalog.NewMemoryStore()
	}
	if deps.Downloads == nil {
		deps.Downloads = download.NewMemoryStore()
	}
	if deps.Integrations == nil {
		deps.Integrations = integration.NewMemoryStore()
	}
	if deps.Libraries == nil {
		deps.Libraries = library.NewMemoryStore()
	}
	if deps.Localization == nil {
		deps.Localization = localization.NewMemoryStore()
	}
	if deps.MediaFiles == nil {
		deps.MediaFiles = media.NewMemoryStore()
	}
	if deps.Organizer == nil {
		deps.Organizer = organizer.NewMemoryStore()
	}
	if deps.Scraper == nil {
		deps.Scraper = scraper.NewMemoryStore()
	}
	if deps.STRM == nil {
		deps.STRM = strm.NewMemoryStore()
	}
	if deps.Tasks == nil {
		deps.Tasks = task.NewQueue()
	}

	server := &Server{
		cfg:          cfg,
		mux:          http.NewServeMux(),
		ai:           deps.AI,
		automations:  deps.Automations,
		catalog:      deps.Catalog,
		downloads:    deps.Downloads,
		integrations: deps.Integrations,
		libraries:    deps.Libraries,
		localization: deps.Localization,
		mediaFiles:   deps.MediaFiles,
		organizer:    deps.Organizer,
		scraper:      deps.Scraper,
		strm:         deps.STRM,
		tasks:        deps.Tasks,
		scanDB:       deps.ScanDB,
	}
	server.routes()
	return server
}

func (s *Server) ListenAndServe() error {
	return http.ListenAndServe(s.cfg.Addr(), s.mux)
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

func (s *Server) StartAutomationTicker(ctx context.Context, interval time.Duration) {
	if interval <= 0 {
		interval = time.Minute
	}
	ticker := time.NewTicker(interval)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case now := <-ticker.C:
				_, _ = s.RunDueAutomations(ctx, now.UTC())
			}
		}
	}()
}

func (s *Server) routes() {
	s.mux.HandleFunc("GET /api/health", s.handleHealth)
	s.mux.HandleFunc("GET /api/config", s.handleConfig)
	s.mux.HandleFunc("GET /api/integrations", s.handleListIntegrations)
	s.mux.HandleFunc("POST /api/integrations", s.handleCreateIntegration)
	s.mux.HandleFunc("GET /api/integrations/", s.handleGetIntegration)
	s.mux.HandleFunc("PATCH /api/integrations/", s.handleUpdateIntegration)
	s.mux.HandleFunc("DELETE /api/integrations/", s.handleDeleteIntegration)
	s.mux.HandleFunc("POST /api/integrations/", s.handleIntegrationAction)
	s.mux.HandleFunc("GET /api/ai/providers", s.handleListAIProviders)
	s.mux.HandleFunc("POST /api/ai/providers", s.handleCreateAIProvider)
	s.mux.HandleFunc("GET /api/ai/providers/", s.handleGetAIProvider)
	s.mux.HandleFunc("PATCH /api/ai/providers/", s.handleUpdateAIProvider)
	s.mux.HandleFunc("DELETE /api/ai/providers/", s.handleDeleteAIProvider)
	s.mux.HandleFunc("POST /api/ai/providers/", s.handleAIProviderAction)
	s.mux.HandleFunc("GET /api/libraries", s.handleListLibraries)
	s.mux.HandleFunc("POST /api/libraries", s.handleCreateLibrary)
	s.mux.HandleFunc("GET /api/libraries/", s.handleGetLibrary)
	s.mux.HandleFunc("PATCH /api/libraries/", s.handleUpdateLibrary)
	s.mux.HandleFunc("POST /api/libraries/", s.handleLibraryAction)
	s.mux.HandleFunc("DELETE /api/libraries/", s.handleDeleteLibrary)
	s.mux.HandleFunc("GET /api/download-directories", s.handleListDownloadDirectories)
	s.mux.HandleFunc("POST /api/download-directories", s.handleCreateDownloadDirectory)
	s.mux.HandleFunc("GET /api/download-directories/", s.handleGetDownloadDirectory)
	s.mux.HandleFunc("PATCH /api/download-directories/", s.handleUpdateDownloadDirectory)
	s.mux.HandleFunc("POST /api/download-directories/", s.handleDownloadDirectoryAction)
	s.mux.HandleFunc("DELETE /api/download-directories/", s.handleDeleteDownloadDirectory)
	s.mux.HandleFunc("GET /api/media-files", s.handleListMediaFiles)
	s.mux.HandleFunc("GET /api/media", s.handleListMedia)
	s.mux.HandleFunc("POST /api/media", s.handleCreateMedia)
	s.mux.HandleFunc("GET /api/media/", s.handleGetMedia)
	s.mux.HandleFunc("PATCH /api/media/", s.handleUpdateMedia)
	s.mux.HandleFunc("POST /api/media/", s.handleMediaAction)
	s.mux.HandleFunc("GET /api/media-translations", s.handleListMediaTranslations)
	s.mux.HandleFunc("POST /api/media-translations", s.handleUpsertMediaTranslation)
	s.mux.HandleFunc("GET /api/people", s.handleListPeople)
	s.mux.HandleFunc("POST /api/people", s.handleCreatePerson)
	s.mux.HandleFunc("GET /api/people/", s.handleGetPerson)
	s.mux.HandleFunc("GET /api/tags", s.handleListTags)
	s.mux.HandleFunc("POST /api/tags", s.handleCreateTag)
	s.mux.HandleFunc("GET /api/tags/", s.handleGetTag)
	s.mux.HandleFunc("GET /api/collections", s.handleListCollections)
	s.mux.HandleFunc("POST /api/collections", s.handleCreateCollection)
	s.mux.HandleFunc("GET /api/collections/", s.handleGetCollection)
	s.mux.HandleFunc("POST /api/collections/", s.handleCollectionAction)
	s.mux.HandleFunc("GET /api/scrape-candidates", s.handleListScrapeCandidates)
	s.mux.HandleFunc("POST /api/scrape-candidates", s.handleCreateScrapeCandidate)
	s.mux.HandleFunc("GET /api/scrape-decisions", s.handleListScrapeDecisions)
	s.mux.HandleFunc("POST /api/scrape-decisions", s.handleCreateScrapeDecision)
	s.mux.HandleFunc("GET /api/strm/rules", s.handleListSTRMRules)
	s.mux.HandleFunc("POST /api/strm/rules", s.handleCreateSTRMRule)
	s.mux.HandleFunc("GET /api/strm/rules/", s.handleGetSTRMRule)
	s.mux.HandleFunc("PATCH /api/strm/rules/", s.handleUpdateSTRMRule)
	s.mux.HandleFunc("DELETE /api/strm/rules/", s.handleDeleteSTRMRule)
	s.mux.HandleFunc("POST /api/nfo/generate", s.handleGenerateNFO)
	s.mux.HandleFunc("POST /api/strm/generate", s.handleGenerateSTRM)
	s.mux.HandleFunc("POST /api/strm/validate", s.handleValidateSTRM)
	s.mux.HandleFunc("GET /api/tasks", s.handleListTasks)
	s.mux.HandleFunc("GET /api/tasks/", s.handleGetTask)
	s.mux.HandleFunc("POST /api/tasks/", s.handleTaskAction)
	s.mux.HandleFunc("GET /api/organizer/rules", s.handleListOrganizerRules)
	s.mux.HandleFunc("POST /api/organizer/rules", s.handleCreateOrganizerRule)
	s.mux.HandleFunc("GET /api/organizer/rules/", s.handleGetOrganizerRule)
	s.mux.HandleFunc("PATCH /api/organizer/rules/", s.handleUpdateOrganizerRule)
	s.mux.HandleFunc("DELETE /api/organizer/rules/", s.handleDeleteOrganizerRule)
	s.mux.HandleFunc("POST /api/organizer/plan", s.handleCreateOrganizerPlan)
	s.mux.HandleFunc("GET /api/organizer/plans/", s.handleGetOrganizerPlan)
	s.mux.HandleFunc("POST /api/organizer/plans/", s.handleOrganizerPlanAction)
	s.mux.HandleFunc("GET /api/organizer/actions", s.handleListOrganizerActions)
	s.mux.HandleFunc("GET /api/automations", s.handleListAutomations)
	s.mux.HandleFunc("POST /api/automations", s.handleCreateAutomation)
	s.mux.HandleFunc("POST /api/automations/run-due", s.handleRunDueAutomations)
	s.mux.HandleFunc("GET /api/automations/", s.handleGetAutomation)
	s.mux.HandleFunc("PATCH /api/automations/", s.handleUpdateAutomation)
	s.mux.HandleFunc("DELETE /api/automations/", s.handleDeleteAutomation)
	s.mux.HandleFunc("POST /api/automations/", s.handleAutomationAction)
}

func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"status":  "ok",
		"service": "movie-tool",
	})
}

func (s *Server) handleConfig(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"host":      s.cfg.Host,
		"port":      s.cfg.Port,
		"data_dir":  s.cfg.DataDir,
		"cache_dir": s.cfg.CacheDir,
		"database":  s.cfg.Database,
	})
}

func (s *Server) handleListIntegrations(w http.ResponseWriter, r *http.Request) {
	servers, err := s.integrations.List(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, servers)
}

func (s *Server) handleCreateIntegration(w http.ResponseWriter, r *http.Request) {
	var input integration.ServerInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	server, err := s.integrations.Create(r.Context(), input)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusCreated, server)
}

func (s *Server) handleGetIntegration(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r.URL.Path, "/api/integrations/")
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	server, ok, err := s.integrations.Get(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if !ok {
		writeError(w, http.StatusNotFound, fmt.Errorf("integration not found"))
		return
	}
	writeJSON(w, http.StatusOK, server)
}

func (s *Server) handleUpdateIntegration(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r.URL.Path, "/api/integrations/")
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	var input integration.ServerUpdate
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	server, ok, err := s.integrations.Update(r.Context(), id, input)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if !ok {
		writeError(w, http.StatusNotFound, fmt.Errorf("integration not found"))
		return
	}
	writeJSON(w, http.StatusOK, server)
}

func (s *Server) handleDeleteIntegration(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r.URL.Path, "/api/integrations/")
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	ok, err := s.integrations.Delete(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if !ok {
		writeError(w, http.StatusNotFound, fmt.Errorf("integration not found"))
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleIntegrationAction(w http.ResponseWriter, r *http.Request) {
	id, action, err := pathIDAction(r.URL.Path, "/api/integrations/")
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	server, ok, err := s.integrations.Get(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if !ok {
		writeError(w, http.StatusNotFound, fmt.Errorf("integration not found"))
		return
	}
	switch action {
	case "test":
		writeJSON(w, http.StatusOK, map[string]any{
			"status":      "configured",
			"server_id":   server.ID,
			"server_type": server.Type,
			"base_url":    server.BaseURL,
			"has_api_key": server.HasAPIKey,
		})
	case "refresh":
		var input integration.RefreshInput
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		plan := integration.BuildRefreshPlan(server, input)
		taskRecord := s.tasks.Enqueue(task.TypeRefreshServer, "refresh server: "+server.Name)
		writeJSON(w, http.StatusAccepted, map[string]any{
			"task": taskRecord,
			"plan": plan,
		})
	default:
		writeError(w, http.StatusBadRequest, fmt.Errorf("unsupported integration action %q", action))
	}
}

func (s *Server) handleListAIProviders(w http.ResponseWriter, r *http.Request) {
	providers, err := s.ai.ListProviders(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, providers)
}

func (s *Server) handleCreateAIProvider(w http.ResponseWriter, r *http.Request) {
	var input ai.ProviderInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	provider, err := s.ai.CreateProvider(r.Context(), input)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusCreated, provider)
}

func (s *Server) handleGetAIProvider(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r.URL.Path, "/api/ai/providers/")
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	provider, ok, err := s.ai.GetProvider(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if !ok {
		writeError(w, http.StatusNotFound, fmt.Errorf("ai provider not found"))
		return
	}
	writeJSON(w, http.StatusOK, provider)
}

func (s *Server) handleUpdateAIProvider(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r.URL.Path, "/api/ai/providers/")
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	var input ai.ProviderUpdate
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	provider, ok, err := s.ai.UpdateProvider(r.Context(), id, input)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if !ok {
		writeError(w, http.StatusNotFound, fmt.Errorf("ai provider not found"))
		return
	}
	writeJSON(w, http.StatusOK, provider)
}

func (s *Server) handleDeleteAIProvider(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r.URL.Path, "/api/ai/providers/")
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	ok, err := s.ai.DeleteProvider(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if !ok {
		writeError(w, http.StatusNotFound, fmt.Errorf("ai provider not found"))
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleAIProviderAction(w http.ResponseWriter, r *http.Request) {
	id, action, err := pathIDAction(r.URL.Path, "/api/ai/providers/")
	if err != nil || action != "test" {
		writeError(w, http.StatusBadRequest, fmt.Errorf("unsupported ai provider action"))
		return
	}
	provider, ok, err := s.ai.GetProvider(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if !ok {
		writeError(w, http.StatusNotFound, fmt.Errorf("ai provider not found"))
		return
	}
	if provider.Type != ai.ProviderOpenAICompatible && provider.Type != ai.ProviderOpenAI && provider.Type != ai.ProviderGemini && provider.Type != ai.ProviderClaude && provider.Type != ai.ProviderOllama {
		writeError(w, http.StatusBadRequest, fmt.Errorf("unsupported provider type %q", provider.Type))
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"status":      "configured",
		"provider_id": provider.ID,
		"has_api_key": provider.HasAPIKey,
		"model":       provider.DefaultModel,
	})
}

func (s *Server) handleListLibraries(w http.ResponseWriter, r *http.Request) {
	libraries, err := s.libraries.List(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, libraries)
}

func (s *Server) handleCreateLibrary(w http.ResponseWriter, r *http.Request) {
	var input library.CreateInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	created, err := s.libraries.Create(r.Context(), input)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

func (s *Server) handleGetLibrary(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r.URL.Path, "/api/libraries/")
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	found, ok, err := s.libraries.Get(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if !ok {
		writeError(w, http.StatusNotFound, fmt.Errorf("library not found"))
		return
	}
	writeJSON(w, http.StatusOK, found)
}

func (s *Server) handleUpdateLibrary(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r.URL.Path, "/api/libraries/")
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	var input library.UpdateInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	updated, ok, err := s.libraries.Update(r.Context(), id, input)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if !ok {
		writeError(w, http.StatusNotFound, fmt.Errorf("library not found"))
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

func (s *Server) handleDeleteLibrary(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r.URL.Path, "/api/libraries/")
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	deleted, err := s.libraries.Delete(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if !deleted {
		writeError(w, http.StatusNotFound, fmt.Errorf("library not found"))
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleLibraryAction(w http.ResponseWriter, r *http.Request) {
	id, action, err := pathIDAction(r.URL.Path, "/api/libraries/")
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	switch action {
	case "scan":
		s.handleScanLibrary(w, r, id)
	case "cleanup-missing":
		s.handleCleanupMissing(w, r, id)
	default:
		writeError(w, http.StatusBadRequest, fmt.Errorf("unsupported library action %q", action))
	}
}

func (s *Server) handleCleanupMissing(w http.ResponseWriter, r *http.Request, id string) {
	if _, ok, err := s.libraries.Get(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	} else if !ok {
		writeError(w, http.StatusNotFound, fmt.Errorf("library not found"))
		return
	}

	deleted, err := s.mediaFiles.DeleteMissingByLibrary(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	taskRecord := s.tasks.Enqueue(task.TypeCleanupMissing, "cleanup missing files")
	taskRecord, _ = s.tasks.Start(taskRecord.ID)
	s.tasks.Log(taskRecord.ID, task.LogLevelInfo, fmt.Sprintf("deleted %d missing files", deleted))
	taskRecord, _ = s.tasks.Succeed(taskRecord.ID, fmt.Sprintf("cleanup missing files: %d deleted", deleted))
	writeJSON(w, http.StatusOK, map[string]any{
		"task":          taskRecord,
		"deleted_count": deleted,
	})
}

func (s *Server) handleScanLibrary(w http.ResponseWriter, r *http.Request, id string) {
	found, ok, err := s.libraries.Get(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if !ok {
		writeError(w, http.StatusNotFound, fmt.Errorf("library not found"))
		return
	}

	taskRecord := s.tasks.Enqueue(task.TypeLibraryScan, "scan library: "+found.Name)
	taskRecord, _ = s.tasks.Start(taskRecord.ID)
	s.tasks.Log(taskRecord.ID, task.LogLevelInfo, "walking "+found.Path)

	files, err := scanner.NewScanner().Walk(scanner.ScanRequest{
		Library: scanner.LibraryInfo{
			ID:        found.ID,
			Name:      found.Name,
			Path:      found.Path,
			MediaType: string(found.MediaType),
		},
	})
	if err != nil {
		taskRecord, _ = s.tasks.Fail(taskRecord.ID, err)
		writeError(w, http.StatusBadRequest, err)
		return
	}
	s.tasks.Log(taskRecord.ID, task.LogLevelInfo, fmt.Sprintf("discovered %d media files", len(files)))

	imported, missingCount, err := s.importScannedFiles(r.Context(), files, found.ID)
	if err != nil {
		taskRecord, _ = s.tasks.Fail(taskRecord.ID, err)
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	s.tasks.Log(taskRecord.ID, task.LogLevelInfo, fmt.Sprintf("imported %d files, marked %d missing", len(imported), missingCount))
	taskRecord, _ = s.tasks.Succeed(taskRecord.ID, fmt.Sprintf("scan library: %s (%d files)", found.Name, len(imported)))
	writeJSON(w, http.StatusAccepted, map[string]any{
		"task":          taskRecord,
		"files":         files,
		"imported":      imported,
		"count":         len(files),
		"missing_count": missingCount,
	})
}

func (s *Server) handleListDownloadDirectories(w http.ResponseWriter, r *http.Request) {
	directories, err := s.downloads.List(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, directories)
}

func (s *Server) handleCreateDownloadDirectory(w http.ResponseWriter, r *http.Request) {
	var input download.DirectoryInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if _, ok, err := s.libraries.Get(r.Context(), input.LibraryID); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	} else if !ok {
		writeError(w, http.StatusBadRequest, fmt.Errorf("target library not found"))
		return
	}
	directory, err := s.downloads.Create(r.Context(), input)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusCreated, directory)
}

func (s *Server) handleGetDownloadDirectory(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r.URL.Path, "/api/download-directories/")
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	directory, ok, err := s.downloads.Get(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if !ok {
		writeError(w, http.StatusNotFound, fmt.Errorf("download directory not found"))
		return
	}
	writeJSON(w, http.StatusOK, directory)
}

func (s *Server) handleUpdateDownloadDirectory(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r.URL.Path, "/api/download-directories/")
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	var input download.DirectoryUpdate
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if input.LibraryID != nil {
		if _, ok, err := s.libraries.Get(r.Context(), *input.LibraryID); err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		} else if !ok {
			writeError(w, http.StatusBadRequest, fmt.Errorf("target library not found"))
			return
		}
	}
	directory, ok, err := s.downloads.Update(r.Context(), id, input)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if !ok {
		writeError(w, http.StatusNotFound, fmt.Errorf("download directory not found"))
		return
	}
	writeJSON(w, http.StatusOK, directory)
}

func (s *Server) handleDeleteDownloadDirectory(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r.URL.Path, "/api/download-directories/")
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	deleted, err := s.downloads.Delete(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if !deleted {
		writeError(w, http.StatusNotFound, fmt.Errorf("download directory not found"))
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleDownloadDirectoryAction(w http.ResponseWriter, r *http.Request) {
	id, action, err := pathIDAction(r.URL.Path, "/api/download-directories/")
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	switch action {
	case "scan":
		s.handleScanDownloadDirectory(w, r, id)
	default:
		writeError(w, http.StatusBadRequest, fmt.Errorf("unsupported download directory action %q", action))
	}
}

func (s *Server) handleScanDownloadDirectory(w http.ResponseWriter, r *http.Request, id string) {
	directory, ok, err := s.downloads.Get(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if !ok {
		writeError(w, http.StatusNotFound, fmt.Errorf("download directory not found"))
		return
	}
	if !directory.Enabled {
		writeError(w, http.StatusBadRequest, fmt.Errorf("download directory is disabled"))
		return
	}
	targetLibrary, ok, err := s.libraries.Get(r.Context(), directory.LibraryID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if !ok {
		writeError(w, http.StatusBadRequest, fmt.Errorf("target library not found"))
		return
	}

	mediaType := firstNonEmpty(directory.MediaType, string(targetLibrary.MediaType))
	minStableAge, err := parseOptionalDurationSeconds(r.URL.Query().Get("min_stable_seconds"))
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	taskRecord := s.tasks.Enqueue(task.TypeLibraryScan, "scan download directory: "+directory.Name)
	taskRecord, _ = s.tasks.Start(taskRecord.ID)
	s.tasks.Log(taskRecord.ID, task.LogLevelInfo, "walking "+directory.Path)
	if minStableAge > 0 {
		s.tasks.Log(taskRecord.ID, task.LogLevelInfo, fmt.Sprintf("skipping files modified within %s", minStableAge))
	}

	files, err := scanner.NewScanner().Walk(scanner.ScanRequest{
		Root:           directory.Path,
		MinModifiedAge: minStableAge,
		Library: scanner.LibraryInfo{
			ID:        targetLibrary.ID,
			Name:      targetLibrary.Name,
			Path:      directory.Path,
			MediaType: mediaType,
		},
	})
	if err != nil {
		taskRecord, _ = s.tasks.Fail(taskRecord.ID, err)
		writeError(w, http.StatusBadRequest, err)
		return
	}

	imported, _, err := s.importScannedFiles(r.Context(), files, "")
	if err != nil {
		taskRecord, _ = s.tasks.Fail(taskRecord.ID, err)
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	s.tasks.Log(taskRecord.ID, task.LogLevelInfo, fmt.Sprintf("imported %d download files", len(imported)))
	taskRecord, _ = s.tasks.Succeed(taskRecord.ID, fmt.Sprintf("scan download directory: %s (%d files)", directory.Name, len(imported)))
	writeJSON(w, http.StatusAccepted, map[string]any{
		"task":               taskRecord,
		"download_directory": directory,
		"target_library":     targetLibrary,
		"files":              files,
		"imported":           imported,
		"count":              len(imported),
	})
}

func (s *Server) importScannedFiles(ctx context.Context, files []scanner.ParsedFile, markMissingLibraryID string) ([]media.File, int, error) {
	if s.scanDB == nil {
		return importScannedFilesWithStores(ctx, s.catalog, s.mediaFiles, files, markMissingLibraryID)
	}

	tx, err := s.scanDB.BeginTx(ctx, nil)
	if err != nil {
		return nil, 0, err
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	imported, missingCount, err := importScannedFilesWithStores(ctx, catalog.NewSQLStore(tx), media.NewSQLStore(tx), files, markMissingLibraryID)
	if err != nil {
		return nil, 0, err
	}
	if err := tx.Commit(); err != nil {
		return nil, 0, err
	}
	committed = true
	return imported, missingCount, nil
}

func importScannedFilesWithStores(ctx context.Context, catalogStore catalog.Store, mediaStore media.Store, files []scanner.ParsedFile, markMissingLibraryID string) ([]media.File, int, error) {
	imported := make([]media.File, 0, len(files))
	availablePaths := make([]string, 0, len(files))
	for _, file := range files {
		mediaID, versionID, err := ensureCatalogVersionForParsedFile(ctx, catalogStore, mediaStore, file)
		if err != nil {
			return nil, 0, err
		}
		stored, err := mediaStore.UpsertFile(ctx, media.FileInput{
			MediaID:           mediaID,
			VersionID:         versionID,
			LibraryID:         file.LibraryID,
			Path:              file.Path,
			FileName:          file.FileName,
			Extension:         file.Extension,
			Size:              file.Size,
			ModifiedAt:        file.ModifiedAt,
			DetectedMediaType: file.MediaType,
			ParsedTitle:       file.Title,
			ParsedYear:        file.Year,
			ParsedSeason:      file.Season,
			ParsedEpisode:     file.Episode,
			ParsedNumber:      file.Number,
		})
		if err != nil {
			return nil, 0, err
		}
		imported = append(imported, stored)
		availablePaths = append(availablePaths, file.Path)
	}

	if markMissingLibraryID == "" {
		return imported, 0, nil
	}
	missingCount, err := mediaStore.MarkMissingByLibrary(ctx, markMissingLibraryID, availablePaths)
	if err != nil {
		return nil, 0, err
	}
	return imported, missingCount, nil
}

func (s *Server) ensureCatalogVersionForParsedFile(ctx context.Context, file scanner.ParsedFile) (string, string, error) {
	return ensureCatalogVersionForParsedFile(ctx, s.catalog, s.mediaFiles, file)
}

func ensureCatalogVersionForParsedFile(ctx context.Context, catalogStore catalog.Store, mediaStore media.Store, file scanner.ParsedFile) (string, string, error) {
	existingFile, ok, err := mediaStore.GetFileByPath(ctx, file.Path)
	if err != nil {
		return "", "", err
	}
	if ok && existingFile.MediaID != "" && existingFile.VersionID != "" {
		return existingFile.MediaID, existingFile.VersionID, nil
	}

	mediaID := existingFile.MediaID
	if mediaID == "" {
		items, err := catalogStore.ListItems(ctx, catalog.ItemQuery{
			LibraryID: file.LibraryID,
			MediaType: file.MediaType,
			Title:     firstNonEmpty(file.Title, file.Number),
			Year:      file.Year,
		})
		if err != nil {
			return "", "", err
		}
		if len(items) > 0 {
			mediaID = items[0].ID
		} else {
			item, err := catalogStore.CreateItem(ctx, catalog.ItemInput{
				LibraryID:    file.LibraryID,
				MediaType:    file.MediaType,
				Title:        firstNonEmpty(file.Title, file.Number),
				DisplayTitle: firstNonEmpty(file.Title, file.Number),
				Year:         file.Year,
				DisplayLang:  "zh-CN",
				Status:       "pending",
				MatchStatus:  catalog.MatchStatusUnmatched,
			})
			if err != nil {
				return "", "", err
			}
			mediaID = item.ID
		}
	}

	versionID := existingFile.VersionID
	if versionID == "" {
		versions, err := catalogStore.ListVersions(ctx, mediaID)
		if err != nil {
			return "", "", err
		}
		version, err := catalogStore.CreateVersion(ctx, mediaID, catalog.VersionInput{
			Name:          versionName(file),
			Resolution:    file.Resolution,
			Source:        file.Source,
			VideoCodec:    file.VideoCodec,
			AudioCodec:    file.AudioCodec,
			HDRFormat:     file.HDRFormat,
			ReleaseGroup:  file.ReleaseGroup,
			SubtitleFlags: strings.Join(file.Subtitles, ","),
			QualityScore:  versionQualityScore(file),
			IsDefault:     len(versions) == 0,
		})
		if err != nil {
			return "", "", err
		}
		versionID = version.ID
	}

	return mediaID, versionID, nil
}

func versionName(file scanner.ParsedFile) string {
	parts := []string{file.Resolution, file.Source, file.VideoCodec, file.HDRFormat}
	var cleaned []string
	for _, part := range parts {
		if part != "" {
			cleaned = append(cleaned, part)
		}
	}
	if len(cleaned) == 0 {
		return "Default"
	}
	return strings.Join(cleaned, " ")
}

func versionQualityScore(file scanner.ParsedFile) int {
	score := 0
	switch file.Resolution {
	case "4320p":
		score += 50
	case "2160p":
		score += 40
	case "1080p":
		score += 30
	case "720p":
		score += 20
	default:
		score += 10
	}
	switch file.Source {
	case "remux":
		score += 30
	case "bluray":
		score += 25
	case "web-dl":
		score += 15
	}
	if file.HDRFormat != "" && file.HDRFormat != "sdr" {
		score += 5
	}
	return score
}

func (s *Server) handleListMediaFiles(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")
	if path != "" {
		file, ok, err := s.mediaFiles.GetFileByPath(r.Context(), path)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		if !ok {
			writeError(w, http.StatusNotFound, fmt.Errorf("media file not found"))
			return
		}
		writeJSON(w, http.StatusOK, file)
		return
	}

	libraryID := r.URL.Query().Get("library_id")
	if libraryID == "" {
		writeError(w, http.StatusBadRequest, fmt.Errorf("library_id or path is required"))
		return
	}
	files, err := s.mediaFiles.ListFiles(r.Context(), media.FileQuery{
		LibraryID: libraryID,
		Status:    media.FileStatus(r.URL.Query().Get("file_status")),
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, files)
}

func (s *Server) handleListMedia(w http.ResponseWriter, r *http.Request) {
	year, err := parseOptionalInt(r.URL.Query().Get("year"))
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	items, err := s.catalog.ListItems(r.Context(), catalog.ItemQuery{
		LibraryID:    r.URL.Query().Get("library_id"),
		MediaType:    r.URL.Query().Get("type"),
		Title:        r.URL.Query().Get("title"),
		Year:         year,
		MatchStatus:  catalog.MatchStatus(r.URL.Query().Get("match_status")),
		PersonID:     r.URL.Query().Get("person_id"),
		TagID:        r.URL.Query().Get("tag_id"),
		CollectionID: r.URL.Query().Get("collection_id"),
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (s *Server) handleCreateMedia(w http.ResponseWriter, r *http.Request) {
	var input catalog.ItemInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	item, err := s.catalog.CreateItem(r.Context(), input)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusCreated, item)
}

func (s *Server) handleGetMedia(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimRight(r.URL.Path, "/")
	if strings.HasSuffix(path, "/versions") {
		s.handleListMediaVersions(w, r)
		return
	}
	if strings.HasSuffix(path, "/translations") {
		s.handleListMediaTranslationsForItem(w, r)
		return
	}
	id, err := pathID(r.URL.Path, "/api/media/")
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	item, ok, err := s.catalog.GetItem(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if !ok {
		writeError(w, http.StatusNotFound, fmt.Errorf("media item not found"))
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (s *Server) handleUpdateMedia(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r.URL.Path, "/api/media/")
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	var input catalog.ItemUpdate
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	item, ok, err := s.catalog.UpdateItem(r.Context(), id, input)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if !ok {
		writeError(w, http.StatusNotFound, fmt.Errorf("media item not found"))
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (s *Server) handleMediaAction(w http.ResponseWriter, r *http.Request) {
	parts, err := pathParts(r.URL.Path, "/api/media/")
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	switch {
	case len(parts) == 2 && parts[1] == "versions":
		s.handleCreateMediaVersion(w, r, parts[0])
	case len(parts) == 4 && parts[1] == "versions" && parts[3] == "default":
		s.handleSetDefaultMediaVersion(w, r, parts[0], parts[2])
	case len(parts) == 2 && parts[1] == "people":
		s.handleAddMediaPerson(w, r, parts[0])
	case len(parts) == 2 && parts[1] == "tags":
		s.handleAddMediaTag(w, r, parts[0])
	case len(parts) == 2 && parts[1] == "translate":
		s.handleTranslateMedia(w, r, parts[0])
	case len(parts) == 2 && parts[1] == "lock":
		s.handleSetMediaLock(w, r, parts[0], true)
	case len(parts) == 2 && parts[1] == "unlock":
		s.handleSetMediaLock(w, r, parts[0], false)
	case len(parts) == 2 && parts[1] == "rescrape":
		s.handleRescrapeMedia(w, r, parts[0])
	default:
		writeError(w, http.StatusBadRequest, fmt.Errorf("unsupported media action"))
	}
}

func (s *Server) handleListMediaVersions(w http.ResponseWriter, r *http.Request) {
	parts, err := pathParts(r.URL.Path, "/api/media/")
	if err != nil || len(parts) != 2 || parts[1] != "versions" {
		writeError(w, http.StatusBadRequest, fmt.Errorf("invalid media versions path"))
		return
	}
	if _, ok, err := s.catalog.GetItem(r.Context(), parts[0]); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	} else if !ok {
		writeError(w, http.StatusNotFound, fmt.Errorf("media item not found"))
		return
	}
	versions, err := s.catalog.ListVersions(r.Context(), parts[0])
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, versions)
}

func (s *Server) handleCreateMediaVersion(w http.ResponseWriter, r *http.Request, mediaID string) {
	if _, ok, err := s.catalog.GetItem(r.Context(), mediaID); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	} else if !ok {
		writeError(w, http.StatusNotFound, fmt.Errorf("media item not found"))
		return
	}
	var input catalog.VersionInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	version, err := s.catalog.CreateVersion(r.Context(), mediaID, input)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusCreated, version)
}

func (s *Server) handleSetDefaultMediaVersion(w http.ResponseWriter, r *http.Request, mediaID, versionID string) {
	version, ok, err := s.catalog.SetDefaultVersion(r.Context(), mediaID, versionID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if !ok {
		writeError(w, http.StatusNotFound, fmt.Errorf("media version not found"))
		return
	}
	writeJSON(w, http.StatusOK, version)
}

func (s *Server) handleSetMediaLock(w http.ResponseWriter, r *http.Request, mediaID string, locked bool) {
	matchStatus := catalog.MatchStatusMatched
	if locked {
		matchStatus = catalog.MatchStatusLocked
	}
	item, ok, err := s.catalog.UpdateItem(r.Context(), mediaID, catalog.ItemUpdate{
		Locked:      &locked,
		MatchStatus: &matchStatus,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if !ok {
		writeError(w, http.StatusNotFound, fmt.Errorf("media item not found"))
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (s *Server) handleRescrapeMedia(w http.ResponseWriter, r *http.Request, mediaID string) {
	item, ok, err := s.catalog.GetItem(r.Context(), mediaID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if !ok {
		writeError(w, http.StatusNotFound, fmt.Errorf("media item not found"))
		return
	}
	taskRecord := s.tasks.Enqueue(task.TypeScrapeMedia, "rescrape media: "+firstNonEmpty(item.DisplayTitle, item.Title, item.ID))
	s.tasks.Log(taskRecord.ID, task.LogLevelInfo, "media queued for rescrape: "+item.ID)
	writeJSON(w, http.StatusAccepted, map[string]any{
		"task":     taskRecord,
		"media_id": item.ID,
	})
}

func (s *Server) handleListMediaTranslations(w http.ResponseWriter, r *http.Request) {
	mediaID := r.URL.Query().Get("media_id")
	if mediaID == "" {
		writeError(w, http.StatusBadRequest, fmt.Errorf("media_id is required"))
		return
	}
	items, err := s.localization.ListMetadata(r.Context(), localization.MetadataQuery{
		MediaID:  mediaID,
		Language: r.URL.Query().Get("language"),
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (s *Server) handleListMediaTranslationsForItem(w http.ResponseWriter, r *http.Request) {
	parts, err := pathParts(r.URL.Path, "/api/media/")
	if err != nil || len(parts) != 2 || parts[1] != "translations" {
		writeError(w, http.StatusBadRequest, fmt.Errorf("invalid media translations path"))
		return
	}
	items, err := s.localization.ListMetadata(r.Context(), localization.MetadataQuery{
		MediaID:  parts[0],
		Language: r.URL.Query().Get("language"),
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (s *Server) handleUpsertMediaTranslation(w http.ResponseWriter, r *http.Request) {
	var input localization.MetadataInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	item, err := s.localization.UpsertMetadata(r.Context(), input)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusCreated, item)
}

func (s *Server) handleTranslateMedia(w http.ResponseWriter, r *http.Request, mediaID string) {
	if _, ok, err := s.catalog.GetItem(r.Context(), mediaID); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	} else if !ok {
		writeError(w, http.StatusNotFound, fmt.Errorf("media item not found"))
		return
	}
	var input localization.TranslateInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	cache, metadata, err := s.localization.Translate(r.Context(), mediaID, input)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{
		"translation": cache,
		"metadata":    metadata,
	})
}

func (s *Server) handleAddMediaPerson(w http.ResponseWriter, r *http.Request, mediaID string) {
	var input catalog.MediaPersonInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if err := s.catalog.AddMediaPerson(r.Context(), mediaID, input); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]string{"status": "created"})
}

func (s *Server) handleAddMediaTag(w http.ResponseWriter, r *http.Request, mediaID string) {
	var input catalog.MediaTagInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if err := s.catalog.AddMediaTag(r.Context(), mediaID, input); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]string{"status": "created"})
}

func (s *Server) handleListPeople(w http.ResponseWriter, r *http.Request) {
	people, err := s.catalog.ListPeople(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, people)
}

func (s *Server) handleCreatePerson(w http.ResponseWriter, r *http.Request) {
	var input catalog.PersonInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	person, err := s.catalog.CreatePerson(r.Context(), input)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusCreated, person)
}

func (s *Server) handleGetPerson(w http.ResponseWriter, r *http.Request) {
	id, action, err := pathIDAction(strings.TrimRight(r.URL.Path, "/"), "/api/people/")
	if err == nil && action == "media" {
		s.handleListMediaByReference(w, r, catalog.ItemQuery{PersonID: id})
		return
	}
	id, err = pathID(r.URL.Path, "/api/people/")
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	person, ok, err := s.catalog.GetPerson(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if !ok {
		writeError(w, http.StatusNotFound, fmt.Errorf("person not found"))
		return
	}
	writeJSON(w, http.StatusOK, person)
}

func (s *Server) handleListTags(w http.ResponseWriter, r *http.Request) {
	tags, err := s.catalog.ListTags(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, tags)
}

func (s *Server) handleCreateTag(w http.ResponseWriter, r *http.Request) {
	var input catalog.TagInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	tag, err := s.catalog.CreateTag(r.Context(), input)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusCreated, tag)
}

func (s *Server) handleGetTag(w http.ResponseWriter, r *http.Request) {
	id, action, err := pathIDAction(strings.TrimRight(r.URL.Path, "/"), "/api/tags/")
	if err == nil && action == "media" {
		s.handleListMediaByReference(w, r, catalog.ItemQuery{TagID: id})
		return
	}
	id, err = pathID(r.URL.Path, "/api/tags/")
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	tag, ok, err := s.catalog.GetTag(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if !ok {
		writeError(w, http.StatusNotFound, fmt.Errorf("tag not found"))
		return
	}
	writeJSON(w, http.StatusOK, tag)
}

func (s *Server) handleListCollections(w http.ResponseWriter, r *http.Request) {
	collections, err := s.catalog.ListCollections(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, collections)
}

func (s *Server) handleCreateCollection(w http.ResponseWriter, r *http.Request) {
	var input catalog.CollectionInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	collection, err := s.catalog.CreateCollection(r.Context(), input)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusCreated, collection)
}

func (s *Server) handleGetCollection(w http.ResponseWriter, r *http.Request) {
	id, action, err := pathIDAction(strings.TrimRight(r.URL.Path, "/"), "/api/collections/")
	if err == nil && action == "media" {
		s.handleListMediaByReference(w, r, catalog.ItemQuery{CollectionID: id})
		return
	}
	id, err = pathID(r.URL.Path, "/api/collections/")
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	collection, ok, err := s.catalog.GetCollection(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if !ok {
		writeError(w, http.StatusNotFound, fmt.Errorf("collection not found"))
		return
	}
	writeJSON(w, http.StatusOK, collection)
}

func (s *Server) handleCollectionAction(w http.ResponseWriter, r *http.Request) {
	id, action, err := pathIDAction(r.URL.Path, "/api/collections/")
	if err != nil || action != "items" {
		writeError(w, http.StatusBadRequest, fmt.Errorf("unsupported collection action"))
		return
	}
	var input catalog.CollectionItemInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if err := s.catalog.AddCollectionItem(r.Context(), id, input); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]string{"status": "created"})
}

func (s *Server) handleListMediaByReference(w http.ResponseWriter, r *http.Request, query catalog.ItemQuery) {
	items, err := s.catalog.ListItems(r.Context(), query)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (s *Server) handleListScrapeCandidates(w http.ResponseWriter, r *http.Request) {
	query := scraper.CandidateQuery{
		MediaFileID: r.URL.Query().Get("media_file_id"),
		MediaID:     r.URL.Query().Get("media_id"),
	}
	if query.MediaFileID == "" && query.MediaID == "" {
		writeError(w, http.StatusBadRequest, fmt.Errorf("media_file_id or media_id is required"))
		return
	}
	candidates, err := s.scraper.ListCandidates(r.Context(), query)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, candidates)
}

func (s *Server) handleCreateScrapeCandidate(w http.ResponseWriter, r *http.Request) {
	var input scraper.CandidateInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if err := s.scoreScrapeCandidate(r.Context(), &input); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	created, err := s.scraper.CreateCandidate(r.Context(), input)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if err := s.refreshMatchStatus(r.Context(), created); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

func (s *Server) scoreScrapeCandidate(ctx context.Context, input *scraper.CandidateInput) error {
	if input.Score > 0 && len(input.ScoreReasons) > 0 {
		return nil
	}
	if input.MediaFileID == "" {
		return nil
	}
	file, ok, err := s.mediaFiles.GetFile(ctx, input.MediaFileID)
	if err != nil {
		return err
	}
	if !ok {
		return nil
	}
	if input.MediaID == "" {
		input.MediaID = file.MediaID
	}
	if input.Score > 0 && len(input.ScoreReasons) > 0 {
		return nil
	}
	scored := scraper.ScoreCandidate(scraper.ParsedMedia{
		Title:  file.ParsedTitle,
		Year:   file.ParsedYear,
		Number: file.ParsedNumber,
	}, *input)
	input.Score = scored.Score
	input.ScoreReasons = scored.ScoreReasons
	return nil
}

func (s *Server) refreshMatchStatus(ctx context.Context, candidate scraper.StoredCandidate) error {
	if candidate.MediaID == "" {
		return nil
	}
	item, ok, err := s.catalog.GetItem(ctx, candidate.MediaID)
	if err != nil || !ok {
		return err
	}
	if item.Locked {
		return nil
	}
	candidates, err := s.scraper.ListCandidates(ctx, scraper.CandidateQuery{MediaID: candidate.MediaID})
	if err != nil {
		return err
	}
	bestScore := 0
	for _, candidate := range candidates {
		if candidate.Score > bestScore {
			bestScore = candidate.Score
		}
	}
	status := catalog.MatchStatus(metadata.DecideMatch(bestScore, len(candidates)))
	_, _, err = s.catalog.UpdateItem(ctx, candidate.MediaID, catalog.ItemUpdate{MatchStatus: &status})
	return err
}

func (s *Server) handleListScrapeDecisions(w http.ResponseWriter, r *http.Request) {
	mediaID := r.URL.Query().Get("media_id")
	if mediaID == "" {
		writeError(w, http.StatusBadRequest, fmt.Errorf("media_id is required"))
		return
	}
	decisions, err := s.scraper.ListDecisions(r.Context(), scraper.DecisionQuery{MediaID: mediaID})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, decisions)
}

func (s *Server) handleCreateScrapeDecision(w http.ResponseWriter, r *http.Request) {
	var input scraper.DecisionInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	created, err := s.scraper.CreateDecision(r.Context(), input)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	applied, err := s.applyScrapeDecision(r.Context(), input)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{
		"decision": created,
		"applied":  applied,
	})
}

func (s *Server) applyScrapeDecision(ctx context.Context, input scraper.DecisionInput) (map[string]any, error) {
	if input.Decision != "" && input.Decision != scraper.DecisionSelect {
		return map[string]any{"status": "skipped", "reason": "decision is not select"}, nil
	}
	if input.CandidateID == "" {
		return map[string]any{"status": "skipped", "reason": "candidate_id is empty"}, nil
	}
	candidate, ok, err := s.scraper.GetCandidate(ctx, input.CandidateID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return map[string]any{"status": "skipped", "reason": "candidate not found"}, nil
	}

	matchStatus := catalog.MatchStatusMatched
	locked := input.Locked
	update := catalog.ItemUpdate{
		MatchStatus: &matchStatus,
		Locked:      &locked,
	}
	if candidate.Title != "" {
		title := candidate.Title
		update.Title = &title
		update.DisplayTitle = &title
	}
	if candidate.OriginalTitle != "" {
		originalTitle := candidate.OriginalTitle
		update.OriginalTitle = &originalTitle
	}
	if candidate.Year > 0 {
		year := candidate.Year
		update.Year = &year
	}
	if candidate.Overview != "" {
		overview := candidate.Overview
		update.Overview = &overview
	}
	item, ok, err := s.catalog.UpdateItem(ctx, input.MediaID, update)
	if err != nil {
		return nil, err
	}
	if !ok {
		return map[string]any{"status": "skipped", "reason": "media item not found"}, nil
	}
	if candidate.Provider != "" && candidate.ExternalID != "" {
		if _, err := s.catalog.UpsertExternalID(ctx, catalog.ExternalIDInput{
			EntityType: "media",
			EntityID:   input.MediaID,
			Provider:   candidate.Provider,
			ExternalID: candidate.ExternalID,
		}); err != nil {
			return nil, err
		}
	}
	if candidate.Title != "" {
		if _, err := s.localization.UpsertMetadata(ctx, localization.MetadataInput{
			MediaID:   input.MediaID,
			Language:  item.DisplayLang,
			FieldName: "title",
			Value:     candidate.Title,
			Source:    "scraper",
			Provider:  candidate.Provider,
			Locked:    input.Locked,
		}); err != nil {
			return nil, err
		}
	}
	if candidate.Overview != "" {
		if _, err := s.localization.UpsertMetadata(ctx, localization.MetadataInput{
			MediaID:   input.MediaID,
			Language:  item.DisplayLang,
			FieldName: "overview",
			Value:     candidate.Overview,
			Source:    "scraper",
			Provider:  candidate.Provider,
			Locked:    input.Locked,
		}); err != nil {
			return nil, err
		}
	}
	return map[string]any{
		"status":       "applied",
		"media_id":     input.MediaID,
		"candidate_id": input.CandidateID,
		"provider":     candidate.Provider,
		"external_id":  candidate.ExternalID,
	}, nil
}

func (s *Server) handleListSTRMRules(w http.ResponseWriter, r *http.Request) {
	rules, err := s.strm.ListRules(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, rules)
}

func (s *Server) handleCreateSTRMRule(w http.ResponseWriter, r *http.Request) {
	var input strm.RuleInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	rule, err := s.strm.CreateRule(r.Context(), input)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusCreated, rule)
}

func (s *Server) handleGetSTRMRule(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r.URL.Path, "/api/strm/rules/")
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	rule, ok, err := s.strm.GetRule(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if !ok {
		writeError(w, http.StatusNotFound, fmt.Errorf("strm rule not found"))
		return
	}
	writeJSON(w, http.StatusOK, rule)
}

func (s *Server) handleUpdateSTRMRule(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r.URL.Path, "/api/strm/rules/")
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	var input strm.RuleUpdate
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	rule, ok, err := s.strm.UpdateRule(r.Context(), id, input)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if !ok {
		writeError(w, http.StatusNotFound, fmt.Errorf("strm rule not found"))
		return
	}
	writeJSON(w, http.StatusOK, rule)
}

func (s *Server) handleDeleteSTRMRule(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r.URL.Path, "/api/strm/rules/")
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	ok, err := s.strm.DeleteRule(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if !ok {
		writeError(w, http.StatusNotFound, fmt.Errorf("strm rule not found"))
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleGenerateNFO(w http.ResponseWriter, r *http.Request) {
	var input struct {
		MediaID   string              `json:"media_id"`
		Media     nfo.MediaInfo       `json:"media"`
		Metadata  []nfo.MetadataField `json:"metadata"`
		Language  string              `json:"language"`
		FileName  string              `json:"file_name"`
		OutputDir string              `json:"output_dir"`
		DryRun    bool                `json:"dry_run"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if input.MediaID != "" {
		item, ok, err := s.catalog.GetItem(r.Context(), input.MediaID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		if !ok {
			writeError(w, http.StatusNotFound, fmt.Errorf("media item not found"))
			return
		}
		input.Media = nfo.MediaInfo{
			ID:            item.ID,
			MediaType:     item.MediaType,
			Title:         item.Title,
			OriginalTitle: item.OriginalTitle,
			DisplayTitle:  item.DisplayTitle,
			Year:          item.Year,
			Overview:      item.Overview,
			Runtime:       item.Runtime,
			ReleaseDate:   item.ReleaseDate,
		}
		if input.Language == "" {
			input.Language = item.DisplayLang
		}
		fields, err := s.localization.ListMetadata(r.Context(), localization.MetadataQuery{MediaID: input.MediaID, Language: input.Language})
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		input.Metadata = make([]nfo.MetadataField, 0, len(fields))
		for _, field := range fields {
			input.Metadata = append(input.Metadata, nfo.MetadataField{
				Language:  field.Language,
				FieldName: field.FieldName,
				Value:     field.Value,
			})
		}
	}
	plan, err := nfo.NewGenerator().Generate(nfo.GenerateRequest{
		Media:     input.Media,
		Metadata:  input.Metadata,
		Language:  input.Language,
		FileName:  input.FileName,
		OutputDir: input.OutputDir,
		DryRun:    input.DryRun,
	})
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	taskRecord := s.tasks.Enqueue(task.TypeGenerateNFO, "generate nfo: "+firstNonEmpty(input.Media.DisplayTitle, input.Media.Title, input.Media.ID))
	writeJSON(w, http.StatusAccepted, map[string]any{
		"task": taskRecord,
		"plan": plan,
	})
}

func (s *Server) handleGenerateSTRM(w http.ResponseWriter, r *http.Request) {
	var input strm.GenerateRequest
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	rule, ok, err := s.strm.GetRule(r.Context(), input.RuleID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if !ok {
		writeError(w, http.StatusNotFound, fmt.Errorf("strm rule not found"))
		return
	}
	if input.LibraryID != "" && len(input.Files) == 0 {
		files, err := s.mediaFiles.ListFiles(r.Context(), media.FileQuery{LibraryID: input.LibraryID, Status: media.FileStatusAvailable})
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		input.Files = make([]strm.FileInfo, 0, len(files))
		for _, file := range files {
			input.Files = append(input.Files, strm.FileInfo{
				ID:        file.ID,
				LibraryID: file.LibraryID,
				Path:      file.Path,
				FileName:  file.FileName,
				Status:    string(file.Status),
			})
		}
	}
	plan, err := strm.NewPlanner().Build(rule, input)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	taskRecord := s.tasks.Enqueue(task.TypeGenerateSTRM, "generate strm: "+rule.Name)
	writeJSON(w, http.StatusAccepted, map[string]any{
		"task": taskRecord,
		"plan": plan,
	})
}

func (s *Server) handleValidateSTRM(w http.ResponseWriter, r *http.Request) {
	var input strm.ValidateRequest
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, strm.NewPlanner().Validate(input.Rule, input.Path))
}

func (s *Server) handleListTasks(w http.ResponseWriter, r *http.Request) {
	query := task.Query{
		Status: task.Status(r.URL.Query().Get("status")),
		Type:   task.Type(r.URL.Query().Get("type")),
	}
	if query.Status == "" && query.Type == "" {
		writeJSON(w, http.StatusOK, s.tasks.List())
		return
	}
	writeJSON(w, http.StatusOK, s.tasks.ListByQuery(query))
}

func (s *Server) handleGetTask(w http.ResponseWriter, r *http.Request) {
	if strings.HasSuffix(strings.TrimRight(r.URL.Path, "/"), "/logs") {
		s.handleListTaskLogs(w, r)
		return
	}
	id, err := pathID(r.URL.Path, "/api/tasks/")
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	found, ok := s.tasks.Get(id)
	if !ok {
		writeError(w, http.StatusNotFound, fmt.Errorf("task not found"))
		return
	}
	writeJSON(w, http.StatusOK, found)
}

func (s *Server) handleListTaskLogs(w http.ResponseWriter, r *http.Request) {
	id, action, err := pathIDAction(strings.TrimRight(r.URL.Path, "/"), "/api/tasks/")
	if err != nil || action != "logs" {
		writeError(w, http.StatusBadRequest, fmt.Errorf("invalid task logs path"))
		return
	}
	if _, ok := s.tasks.Get(id); !ok {
		writeError(w, http.StatusNotFound, fmt.Errorf("task not found"))
		return
	}
	writeJSON(w, http.StatusOK, s.tasks.Logs(id))
}

func (s *Server) handleTaskAction(w http.ResponseWriter, r *http.Request) {
	id, action, err := pathIDAction(r.URL.Path, "/api/tasks/")
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	switch action {
	case "retry":
		found, ok := s.tasks.Get(id)
		if !ok {
			writeError(w, http.StatusNotFound, fmt.Errorf("task not found"))
			return
		}
		retry := s.tasks.Enqueue(found.Type, "retry: "+found.Message)
		writeJSON(w, http.StatusCreated, retry)
	case "cancel":
		canceled, ok := s.tasks.Cancel(id)
		if !ok {
			writeError(w, http.StatusNotFound, fmt.Errorf("task not found"))
			return
		}
		writeJSON(w, http.StatusOK, canceled)
	default:
		writeError(w, http.StatusBadRequest, fmt.Errorf("unsupported task action %q", action))
	}
}

func (s *Server) handleListOrganizerRules(w http.ResponseWriter, r *http.Request) {
	rules, err := s.organizer.ListRules(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, rules)
}

func (s *Server) handleCreateOrganizerRule(w http.ResponseWriter, r *http.Request) {
	var input organizer.RuleInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	rule, err := s.organizer.CreateRule(r.Context(), input)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusCreated, rule)
}

func (s *Server) handleGetOrganizerRule(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r.URL.Path, "/api/organizer/rules/")
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	rule, ok, err := s.organizer.GetRule(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if !ok {
		writeError(w, http.StatusNotFound, fmt.Errorf("organizer rule not found"))
		return
	}
	writeJSON(w, http.StatusOK, rule)
}

func (s *Server) handleUpdateOrganizerRule(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r.URL.Path, "/api/organizer/rules/")
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	var input organizer.RuleUpdate
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	rule, ok, err := s.organizer.UpdateRule(r.Context(), id, input)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if !ok {
		writeError(w, http.StatusNotFound, fmt.Errorf("organizer rule not found"))
		return
	}
	writeJSON(w, http.StatusOK, rule)
}

func (s *Server) handleDeleteOrganizerRule(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r.URL.Path, "/api/organizer/rules/")
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	ok, err := s.organizer.DeleteRule(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if !ok {
		writeError(w, http.StatusNotFound, fmt.Errorf("organizer rule not found"))
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleCreateOrganizerPlan(w http.ResponseWriter, r *http.Request) {
	var input organizer.PlanRequest
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if input.RuleID != "" {
		rule, ok, err := s.organizer.GetRule(r.Context(), input.RuleID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		if !ok {
			writeError(w, http.StatusNotFound, fmt.Errorf("organizer rule not found"))
			return
		}
		input.Rule = rule
	}
	if input.MediaID != "" && input.Media.ID == "" {
		built, err := s.organizerPlanRequestFromMedia(r.Context(), input)
		if err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		input = built
	}
	if input.LibraryID != "" && input.MediaID == "" && input.Media.ID == "" && len(input.Files) == 0 {
		built, err := s.organizerPlanRequestFromLibrary(r.Context(), input)
		if err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		input = built
	}
	plan, err := organizer.NewPlanner().Build(input)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	plan = organizer.FilterPlanActions(plan, organizer.ActionStatus(input.ActionStatus))
	saved, err := s.organizer.SavePlan(r.Context(), plan)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusCreated, saved)
}

func (s *Server) organizerPlanRequestFromMedia(ctx context.Context, input organizer.PlanRequest) (organizer.PlanRequest, error) {
	item, ok, err := s.catalog.GetItem(ctx, input.MediaID)
	if err != nil {
		return organizer.PlanRequest{}, err
	}
	if !ok {
		return organizer.PlanRequest{}, fmt.Errorf("media item not found")
	}
	versions, err := s.catalog.ListVersions(ctx, item.ID)
	if err != nil {
		return organizer.PlanRequest{}, err
	}
	files, err := s.mediaFiles.ListFiles(ctx, media.FileQuery{LibraryID: item.LibraryID, MediaID: item.ID, Status: media.FileStatusAvailable})
	if err != nil {
		return organizer.PlanRequest{}, err
	}
	if len(files) == 0 {
		return organizer.PlanRequest{}, fmt.Errorf("media item has no available files")
	}

	input.Media = organizer.MediaInfo{
		ID:            item.ID,
		LibraryID:     item.LibraryID,
		MediaType:     item.MediaType,
		Title:         item.Title,
		OriginalTitle: item.OriginalTitle,
		DisplayTitle:  item.DisplayTitle,
		Year:          item.Year,
	}
	input.Versions = make([]organizer.VersionInfo, 0, len(versions))
	for _, version := range versions {
		input.Versions = append(input.Versions, organizer.VersionInfo{
			ID:           version.ID,
			Name:         version.Name,
			Resolution:   version.Resolution,
			Source:       version.Source,
			VideoCodec:   version.VideoCodec,
			AudioCodec:   version.AudioCodec,
			HDRFormat:    version.HDRFormat,
			Edition:      version.Edition,
			ReleaseGroup: version.ReleaseGroup,
			IsDefault:    version.IsDefault,
		})
	}
	input.Files = make([]organizer.FileInfo, 0, len(files))
	for _, file := range files {
		input.Files = append(input.Files, organizer.FileInfo{
			ID:        file.ID,
			MediaID:   file.MediaID,
			VersionID: file.VersionID,
			Path:      file.Path,
			FileName:  file.FileName,
			Extension: file.Extension,
			Season:    file.ParsedSeason,
			Episode:   file.ParsedEpisode,
			Number:    file.ParsedNumber,
		})
	}
	if input.LibraryID == "" {
		input.LibraryID = item.LibraryID
	}
	return input, nil
}

func (s *Server) organizerPlanRequestFromLibrary(ctx context.Context, input organizer.PlanRequest) (organizer.PlanRequest, error) {
	mediaType := firstNonEmpty(input.MediaType, input.Rule.MediaType)
	sourcePathPrefix := strings.TrimSpace(input.SourcePathPrefix)
	if input.DownloadDirectoryID != "" {
		directory, ok, err := s.downloads.Get(ctx, input.DownloadDirectoryID)
		if err != nil {
			return organizer.PlanRequest{}, err
		}
		if !ok {
			return organizer.PlanRequest{}, fmt.Errorf("download directory not found")
		}
		if directory.LibraryID != input.LibraryID {
			return organizer.PlanRequest{}, fmt.Errorf("download directory does not belong to library")
		}
		sourcePathPrefix = firstNonEmpty(sourcePathPrefix, directory.Path)
	}
	itemQuery := catalog.ItemQuery{
		LibraryID:   input.LibraryID,
		MediaType:   mediaType,
		MatchStatus: catalog.MatchStatus(input.MatchStatus),
	}
	items, err := s.catalog.ListItems(ctx, itemQuery)
	if err != nil {
		return organizer.PlanRequest{}, err
	}
	fileStatus := media.FileStatus(input.FileStatus)
	if fileStatus == "" {
		fileStatus = media.FileStatusAvailable
	}
	files, err := s.mediaFiles.ListFiles(ctx, media.FileQuery{LibraryID: input.LibraryID, Status: fileStatus})
	if err != nil {
		return organizer.PlanRequest{}, err
	}
	if len(items) == 0 || len(files) == 0 {
		return organizer.PlanRequest{}, fmt.Errorf("library has no files matching organizer filters")
	}

	itemByID := make(map[string]catalog.Item, len(items))
	for _, item := range items {
		itemByID[item.ID] = item
	}
	versionByID := make(map[string]catalog.Version)
	for _, item := range items {
		versions, err := s.catalog.ListVersions(ctx, item.ID)
		if err != nil {
			return organizer.PlanRequest{}, err
		}
		for _, version := range versions {
			versionByID[version.ID] = version
		}
	}

	input.Media = organizer.MediaInfo{
		ID:        "library-" + input.LibraryID,
		LibraryID: input.LibraryID,
		MediaType: mediaType,
		Title:     "Library " + input.LibraryID,
	}
	input.Versions = make([]organizer.VersionInfo, 0, len(versionByID))
	for _, version := range versionByID {
		input.Versions = append(input.Versions, organizer.VersionInfo{
			ID:           version.ID,
			Name:         version.Name,
			Resolution:   version.Resolution,
			Source:       version.Source,
			VideoCodec:   version.VideoCodec,
			AudioCodec:   version.AudioCodec,
			HDRFormat:    version.HDRFormat,
			Edition:      version.Edition,
			ReleaseGroup: version.ReleaseGroup,
			IsDefault:    version.IsDefault,
		})
	}
	input.Files = make([]organizer.FileInfo, 0, len(files))
	for _, file := range files {
		if sourcePathPrefix != "" && !pathWithinPrefix(file.Path, sourcePathPrefix) {
			continue
		}
		item, ok := itemByID[file.MediaID]
		if !ok {
			continue
		}
		input.Files = append(input.Files, organizer.FileInfo{
			ID:            file.ID,
			MediaID:       file.MediaID,
			VersionID:     file.VersionID,
			Path:          file.Path,
			FileName:      file.FileName,
			Extension:     file.Extension,
			Season:        file.ParsedSeason,
			Episode:       file.ParsedEpisode,
			Number:        file.ParsedNumber,
			MediaTitle:    item.Title,
			DisplayTitle:  item.DisplayTitle,
			OriginalTitle: item.OriginalTitle,
			Year:          item.Year,
			MediaType:     item.MediaType,
		})
	}
	if len(input.Files) == 0 {
		return organizer.PlanRequest{}, fmt.Errorf("library has no files matching organizer filters")
	}
	return input, nil
}

func (s *Server) handleGetOrganizerPlan(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r.URL.Path, "/api/organizer/plans/")
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	plan, ok, err := s.organizer.GetPlan(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if !ok {
		writeError(w, http.StatusNotFound, fmt.Errorf("organizer plan not found"))
		return
	}
	writeJSON(w, http.StatusOK, plan)
}

func (s *Server) handleOrganizerPlanAction(w http.ResponseWriter, r *http.Request) {
	id, action, err := pathIDAction(r.URL.Path, "/api/organizer/plans/")
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	switch action {
	case "execute":
		s.handleExecuteOrganizerPlan(w, r, id)
	case "retry":
		s.handleRetryOrganizerPlan(w, r, id)
	case "skip-conflicts":
		s.handleSkipOrganizerPlanConflicts(w, r, id)
	case "cancel":
		s.handleCancelOrganizerPlan(w, r, id)
	default:
		writeError(w, http.StatusBadRequest, fmt.Errorf("unsupported organizer plan action %q", action))
	}
}

func (s *Server) handleExecuteOrganizerPlan(w http.ResponseWriter, r *http.Request, id string) {
	plan, ok, err := s.organizer.GetPlan(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if !ok {
		writeError(w, http.StatusNotFound, fmt.Errorf("organizer plan not found"))
		return
	}
	if plan.Status == organizer.PlanSucceeded || plan.Status == organizer.PlanCanceled {
		writeError(w, http.StatusBadRequest, fmt.Errorf("organizer plan cannot be executed from status %q", plan.Status))
		return
	}
	taskRecord, saved, err := s.executeOrganizerPlan(r.Context(), plan, "execute organizer plan: "+plan.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]any{
		"task": taskRecord,
		"plan": saved,
	})
}

func (s *Server) handleRetryOrganizerPlan(w http.ResponseWriter, r *http.Request, id string) {
	plan, ok, err := s.organizer.GetPlan(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if !ok {
		writeError(w, http.StatusNotFound, fmt.Errorf("organizer plan not found"))
		return
	}
	if plan.Status != organizer.PlanFailed {
		writeError(w, http.StatusBadRequest, fmt.Errorf("organizer plan cannot be retried from status %q", plan.Status))
		return
	}
	prepared, retryable := organizer.PrepareRetry(plan, time.Now().UTC())
	if retryable == 0 {
		writeError(w, http.StatusBadRequest, fmt.Errorf("organizer plan has no failed actions to retry"))
		return
	}
	taskRecord, saved, err := s.executeOrganizerPlan(r.Context(), prepared, "retry organizer plan: "+plan.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]any{
		"task":      taskRecord,
		"plan":      saved,
		"retryable": retryable,
	})
}

func (s *Server) handleSkipOrganizerPlanConflicts(w http.ResponseWriter, r *http.Request, id string) {
	plan, ok, err := s.organizer.GetPlan(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if !ok {
		writeError(w, http.StatusNotFound, fmt.Errorf("organizer plan not found"))
		return
	}
	if plan.Status == organizer.PlanSucceeded || plan.Status == organizer.PlanCanceled {
		writeError(w, http.StatusBadRequest, fmt.Errorf("organizer plan conflicts cannot be skipped from status %q", plan.Status))
		return
	}
	updated, changed := organizer.SkipConflicts(plan, time.Now().UTC())
	if changed == 0 {
		writeError(w, http.StatusBadRequest, fmt.Errorf("organizer plan has no conflicts to skip"))
		return
	}
	saved, err := s.organizer.UpdatePlan(r.Context(), updated)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"plan":    saved,
		"changed": changed,
	})
}

func (s *Server) executeOrganizerPlan(ctx context.Context, plan organizer.Plan, message string) (task.Task, organizer.Plan, error) {
	taskRecord := s.tasks.Enqueue(task.TypeOrganizeFiles, message)
	taskRecord, _ = s.tasks.Start(taskRecord.ID)
	executed := organizer.NewExecutor().Execute(ctx, plan)
	saved, err := s.organizer.UpdatePlan(ctx, executed)
	if err != nil {
		taskRecord, _ = s.tasks.Fail(taskRecord.ID, err)
		return taskRecord, organizer.Plan{}, err
	}
	for index, action := range saved.Actions {
		if action.Status == organizer.ActionSucceeded && action.MediaFileID != "" {
			if _, ok, err := s.mediaFiles.UpdateFilePath(ctx, action.MediaFileID, action.TargetPath); err != nil {
				saved.Actions[index].Status = organizer.ActionFailed
				saved.Actions[index].Error = "update media file path: " + err.Error()
				saved.Status = organizer.PlanFailed
			} else if !ok {
				s.tasks.Log(taskRecord.ID, task.LogLevelWarn, "media file not found for path update: "+action.MediaFileID)
			}
		}
	}
	if saved.Status == organizer.PlanFailed {
		saved.Summary = organizer.SummarizeActions(saved.Actions)
		var updateErr error
		saved, updateErr = s.organizer.UpdatePlan(ctx, saved)
		if updateErr != nil {
			taskRecord, _ = s.tasks.Fail(taskRecord.ID, updateErr)
			return taskRecord, organizer.Plan{}, updateErr
		}
	}
	for _, action := range saved.Actions {
		s.tasks.Log(taskRecord.ID, task.LogLevelInfo, fmt.Sprintf("%s %s -> %s: %s", action.ActionType, action.SourcePath, action.TargetPath, action.Status))
	}
	if saved.Status == organizer.PlanFailed {
		taskRecord, _ = s.tasks.Fail(taskRecord.ID, fmt.Errorf("organizer plan failed"))
	} else {
		taskRecord, _ = s.tasks.Succeed(taskRecord.ID, message)
	}
	return taskRecord, saved, nil
}

func (s *Server) handleCancelOrganizerPlan(w http.ResponseWriter, r *http.Request, id string) {
	plan, ok, err := s.organizer.GetPlan(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if !ok {
		writeError(w, http.StatusNotFound, fmt.Errorf("organizer plan not found"))
		return
	}
	if plan.Status == organizer.PlanSucceeded {
		writeError(w, http.StatusBadRequest, fmt.Errorf("succeeded organizer plan cannot be canceled"))
		return
	}
	plan.Status = organizer.PlanCanceled
	plan.UpdatedAt = time.Now().UTC()
	saved, err := s.organizer.UpdatePlan(r.Context(), plan)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, saved)
}

func (s *Server) handleListOrganizerActions(w http.ResponseWriter, r *http.Request) {
	planID := r.URL.Query().Get("plan_id")
	if planID == "" {
		writeError(w, http.StatusBadRequest, fmt.Errorf("plan_id is required"))
		return
	}
	actions, err := s.organizer.ListActions(r.Context(), planID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, actions)
}

func (s *Server) handleListAutomations(w http.ResponseWriter, r *http.Request) {
	query := automation.Query{
		Type: automation.Type(r.URL.Query().Get("automation_type")),
	}
	if enabledValue := r.URL.Query().Get("enabled"); enabledValue != "" {
		enabled, err := parseBoolQuery(enabledValue)
		if err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		query.Enabled = &enabled
	}

	var automations []automation.Automation
	var err error
	if query.Type == "" && query.Enabled == nil {
		automations, err = s.automations.List(r.Context())
	} else {
		automations, err = s.automations.ListByQuery(r.Context(), query)
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, automations)
}

func (s *Server) handleCreateAutomation(w http.ResponseWriter, r *http.Request) {
	var input automation.CreateInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	created, err := s.automations.Create(r.Context(), input)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

func (s *Server) handleRunDueAutomations(w http.ResponseWriter, r *http.Request) {
	now, err := parseOptionalTime(r.URL.Query().Get("now"))
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	results, err := s.RunDueAutomations(r.Context(), now)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	writeJSON(w, http.StatusAccepted, map[string]any{
		"now":     now,
		"count":   len(results),
		"results": results,
	})
}

func (s *Server) RunDueAutomations(ctx context.Context, now time.Time) ([]map[string]any, error) {
	enabled := true
	automations, err := s.automations.ListByQuery(ctx, automation.Query{Enabled: &enabled})
	if err != nil {
		return nil, err
	}

	results := make([]map[string]any, 0)
	for _, item := range automations {
		if item.NextRunAt == nil || item.NextRunAt.After(now) {
			continue
		}
		taskRecord, run, err := s.queueAutomationRun(ctx, item)
		result := map[string]any{"automation": item}
		if err != nil {
			result["error"] = err.Error()
		} else {
			result["task"] = taskRecord
			result["run"] = run
		}
		results = append(results, result)
	}
	return results, nil
}

func (s *Server) handleGetAutomation(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r.URL.Path, "/api/automations/")
	if err != nil {
		actionID, action, actionErr := pathIDAction(r.URL.Path, "/api/automations/")
		if actionErr == nil && action == "runs" {
			runs, err := s.automations.ListRuns(r.Context(), actionID)
			if err != nil {
				writeError(w, http.StatusInternalServerError, err)
				return
			}
			writeJSON(w, http.StatusOK, runs)
			return
		}
		writeError(w, http.StatusBadRequest, err)
		return
	}
	found, ok, err := s.automations.Get(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if !ok {
		writeError(w, http.StatusNotFound, fmt.Errorf("automation not found"))
		return
	}
	writeJSON(w, http.StatusOK, found)
}

func (s *Server) handleUpdateAutomation(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r.URL.Path, "/api/automations/")
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	var input automation.UpdateInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	updated, ok, err := s.automations.Update(r.Context(), id, input)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if !ok {
		writeError(w, http.StatusNotFound, fmt.Errorf("automation not found"))
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

func (s *Server) handleDeleteAutomation(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r.URL.Path, "/api/automations/")
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	deleted, err := s.automations.Delete(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if !deleted {
		writeError(w, http.StatusNotFound, fmt.Errorf("automation not found"))
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleAutomationAction(w http.ResponseWriter, r *http.Request) {
	id, action, err := pathIDAction(r.URL.Path, "/api/automations/")
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	switch action {
	case "pause":
		enabled := false
		s.updateAutomationEnabled(w, r, id, enabled)
	case "resume":
		enabled := true
		s.updateAutomationEnabled(w, r, id, enabled)
	case "run":
		found, ok, err := s.automations.Get(r.Context(), id)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		if !ok {
			writeError(w, http.StatusNotFound, fmt.Errorf("automation not found"))
			return
		}
		queued, run, err := s.queueAutomationRun(r.Context(), found)
		if err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		writeJSON(w, http.StatusAccepted, map[string]any{"task": queued, "run": run})
	case "runs":
		runs, err := s.automations.ListRuns(r.Context(), id)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		writeJSON(w, http.StatusOK, runs)
	default:
		writeError(w, http.StatusBadRequest, fmt.Errorf("unsupported automation action %q", action))
	}
}

func (s *Server) queueAutomationRun(ctx context.Context, item automation.Automation) (task.Task, automation.Run, error) {
	taskType, err := automationTaskType(item.Type)
	if err != nil {
		return task.Task{}, automation.Run{}, err
	}
	queued := s.tasks.Enqueue(taskType, "automation: "+item.Name)
	s.tasks.Log(queued.ID, task.LogLevelInfo, fmt.Sprintf("automation %s (%s) queued task %s", item.ID, item.Type, queued.ID))
	run, err := s.automations.RecordRun(ctx, automation.RecordRunInput{
		AutomationID: item.ID,
		TaskID:       queued.ID,
		Status:       automation.RunPending,
	})
	if err != nil {
		return task.Task{}, automation.Run{}, err
	}
	s.tasks.Log(queued.ID, task.LogLevelInfo, "automation run recorded: "+run.ID)
	return queued, run, nil
}

func (s *Server) updateAutomationEnabled(w http.ResponseWriter, r *http.Request, id string, enabled bool) {
	updated, ok, err := s.automations.Update(r.Context(), id, automation.UpdateInput{Enabled: &enabled})
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if !ok {
		writeError(w, http.StatusNotFound, fmt.Errorf("automation not found"))
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

func automationTaskType(automationType automation.Type) (task.Type, error) {
	switch automationType {
	case automation.TypeScanLibrary:
		return task.TypeLibraryScan, nil
	case automation.TypeScrapePending:
		return task.TypeScrapeMedia, nil
	case automation.TypeTranslateMissing:
		return task.TypeTranslateMetadata, nil
	case automation.TypeOrganizeFiles:
		return task.TypeOrganizeFiles, nil
	case automation.TypeGenerateNFO:
		return task.TypeGenerateNFO, nil
	case automation.TypeGenerateSTRM:
		return task.TypeGenerateSTRM, nil
	case automation.TypeRefreshServer:
		return task.TypeRefreshServer, nil
	case automation.TypeCleanupMissing:
		return task.TypeCleanupMissing, nil
	default:
		return "", fmt.Errorf("unsupported automation type %q", automationType)
	}
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func writeError(w http.ResponseWriter, status int, err error) {
	writeJSON(w, status, map[string]map[string]string{
		"error": {
			"message": err.Error(),
		},
	})
}

func pathID(path, prefix string) (string, error) {
	if !strings.HasPrefix(path, prefix) {
		return "", fmt.Errorf("invalid path")
	}
	id := strings.Trim(strings.TrimPrefix(path, prefix), "/")
	if id == "" || strings.Contains(id, "/") {
		return "", fmt.Errorf("invalid id")
	}
	return id, nil
}

func pathIDAction(path, prefix string) (string, string, error) {
	if !strings.HasPrefix(path, prefix) {
		return "", "", fmt.Errorf("invalid path")
	}
	rest := strings.Trim(strings.TrimPrefix(path, prefix), "/")
	parts := strings.Split(rest, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("invalid action path")
	}
	return parts[0], parts[1], nil
}

func pathParts(path, prefix string) ([]string, error) {
	if !strings.HasPrefix(path, prefix) {
		return nil, fmt.Errorf("invalid path")
	}
	rest := strings.Trim(strings.TrimPrefix(path, prefix), "/")
	if rest == "" {
		return nil, fmt.Errorf("invalid path")
	}
	parts := strings.Split(rest, "/")
	for _, part := range parts {
		if part == "" {
			return nil, fmt.Errorf("invalid path")
		}
	}
	return parts, nil
}

func parseOptionalInt(value string) (int, error) {
	if value == "" {
		return 0, nil
	}
	var parsed int
	if _, err := fmt.Sscanf(value, "%d", &parsed); err != nil {
		return 0, fmt.Errorf("invalid integer query value %q", value)
	}
	return parsed, nil
}

func parseOptionalDurationSeconds(value string) (time.Duration, error) {
	seconds, err := parseOptionalInt(value)
	if err != nil {
		return 0, err
	}
	if seconds < 0 {
		return 0, fmt.Errorf("duration seconds cannot be negative")
	}
	return time.Duration(seconds) * time.Second, nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func pathWithinPrefix(path string, prefix string) bool {
	cleanPath := filepath.Clean(path)
	cleanPrefix := filepath.Clean(prefix)
	if cleanPath == cleanPrefix {
		return true
	}
	if cleanPrefix == "." || cleanPrefix == string(filepath.Separator) {
		return strings.HasPrefix(cleanPath, cleanPrefix)
	}
	return strings.HasPrefix(cleanPath, cleanPrefix+string(filepath.Separator))
}

func parseOptionalTime(value string) (time.Time, error) {
	if value == "" {
		return time.Now().UTC(), nil
	}
	parsed, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid time query value %q", value)
	}
	return parsed.UTC(), nil
}

func parseBoolQuery(value string) (bool, error) {
	switch strings.ToLower(value) {
	case "true", "1", "yes":
		return true, nil
	case "false", "0", "no":
		return false, nil
	default:
		return false, fmt.Errorf("invalid boolean query value %q", value)
	}
}
