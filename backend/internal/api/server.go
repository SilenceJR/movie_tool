package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"movie-tool/backend/internal/ai"
	"movie-tool/backend/internal/automation"
	"movie-tool/backend/internal/catalog"
	"movie-tool/backend/internal/config"
	"movie-tool/backend/internal/library"
	"movie-tool/backend/internal/localization"
	"movie-tool/backend/internal/media"
	"movie-tool/backend/internal/organizer"
	"movie-tool/backend/internal/scanner"
	"movie-tool/backend/internal/scraper"
	"movie-tool/backend/internal/task"
)

type Server struct {
	cfg          config.Config
	mux          *http.ServeMux
	ai           ai.Store
	automations  automation.Store
	catalog      catalog.Store
	libraries    library.Store
	localization localization.Store
	mediaFiles   media.Store
	organizer    organizer.Store
	scraper      scraper.Store
	tasks        *task.Queue
}

type Dependencies struct {
	AI           ai.Store
	Automations  automation.Store
	Catalog      catalog.Store
	Libraries    library.Store
	Localization localization.Store
	MediaFiles   media.Store
	Organizer    organizer.Store
	Scraper      scraper.Store
	Tasks        *task.Queue
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
	if deps.Tasks == nil {
		deps.Tasks = task.NewQueue()
	}

	server := &Server{
		cfg:          cfg,
		mux:          http.NewServeMux(),
		ai:           deps.AI,
		automations:  deps.Automations,
		catalog:      deps.Catalog,
		libraries:    deps.Libraries,
		localization: deps.Localization,
		mediaFiles:   deps.MediaFiles,
		organizer:    deps.Organizer,
		scraper:      deps.Scraper,
		tasks:        deps.Tasks,
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

func (s *Server) routes() {
	s.mux.HandleFunc("GET /api/health", s.handleHealth)
	s.mux.HandleFunc("GET /api/config", s.handleConfig)
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
	s.mux.HandleFunc("GET /api/tasks", s.handleListTasks)
	s.mux.HandleFunc("GET /api/tasks/", s.handleGetTask)
	s.mux.HandleFunc("POST /api/tasks/", s.handleTaskAction)
	s.mux.HandleFunc("POST /api/organizer/plan", s.handleCreateOrganizerPlan)
	s.mux.HandleFunc("GET /api/organizer/plans/", s.handleGetOrganizerPlan)
	s.mux.HandleFunc("GET /api/organizer/actions", s.handleListOrganizerActions)
	s.mux.HandleFunc("GET /api/automations", s.handleListAutomations)
	s.mux.HandleFunc("POST /api/automations", s.handleCreateAutomation)
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

	files, err := scanner.NewScanner().Walk(scanner.ScanRequest{
		Library: scanner.LibraryInfo{
			ID:        found.ID,
			Name:      found.Name,
			Path:      found.Path,
			MediaType: string(found.MediaType),
		},
	})
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	imported := make([]media.File, 0, len(files))
	availablePaths := make([]string, 0, len(files))
	for _, file := range files {
		stored, err := s.mediaFiles.UpsertFile(r.Context(), media.FileInput{
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
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		imported = append(imported, stored)
		availablePaths = append(availablePaths, file.Path)
	}

	missingCount, err := s.mediaFiles.MarkMissingByLibrary(r.Context(), found.ID, availablePaths)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	taskRecord := s.tasks.Enqueue(task.TypeLibraryScan, "scan library: "+found.Name)
	writeJSON(w, http.StatusAccepted, map[string]any{
		"task":          taskRecord,
		"files":         files,
		"imported":      imported,
		"count":         len(files),
		"missing_count": missingCount,
	})
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
	created, err := s.scraper.CreateCandidate(r.Context(), input)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusCreated, created)
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
	writeJSON(w, http.StatusCreated, created)
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

func (s *Server) handleCreateOrganizerPlan(w http.ResponseWriter, r *http.Request) {
	var input organizer.PlanRequest
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	plan, err := organizer.NewPlanner().Build(input)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	saved, err := s.organizer.SavePlan(r.Context(), plan)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusCreated, saved)
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
		taskType, err := automationTaskType(found.Type)
		if err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		queued := s.tasks.Enqueue(taskType, "automation: "+found.Name)
		run, err := s.automations.RecordRun(r.Context(), automation.RecordRunInput{
			AutomationID: found.ID,
			TaskID:       queued.ID,
			Status:       automation.RunPending,
		})
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
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
