package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"movie-tool/backend/internal/automation"
	"movie-tool/backend/internal/config"
	"movie-tool/backend/internal/library"
	"movie-tool/backend/internal/organizer"
	"movie-tool/backend/internal/scanner"
	"movie-tool/backend/internal/task"
)

type Server struct {
	cfg         config.Config
	mux         *http.ServeMux
	automations automation.Store
	libraries   library.Store
	tasks       *task.Queue
}

type Dependencies struct {
	Automations automation.Store
	Libraries   library.Store
	Tasks       *task.Queue
}

func NewServer(cfg config.Config) *Server {
	return NewServerWithDependencies(cfg, Dependencies{})
}

func NewServerWithDependencies(cfg config.Config, deps Dependencies) *Server {
	if deps.Automations == nil {
		deps.Automations = automation.NewMemoryStore()
	}
	if deps.Libraries == nil {
		deps.Libraries = library.NewMemoryStore()
	}
	if deps.Tasks == nil {
		deps.Tasks = task.NewQueue()
	}

	server := &Server{
		cfg:         cfg,
		mux:         http.NewServeMux(),
		automations: deps.Automations,
		libraries:   deps.Libraries,
		tasks:       deps.Tasks,
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
	s.mux.HandleFunc("GET /api/libraries", s.handleListLibraries)
	s.mux.HandleFunc("POST /api/libraries", s.handleCreateLibrary)
	s.mux.HandleFunc("GET /api/libraries/", s.handleGetLibrary)
	s.mux.HandleFunc("PATCH /api/libraries/", s.handleUpdateLibrary)
	s.mux.HandleFunc("POST /api/libraries/", s.handleLibraryAction)
	s.mux.HandleFunc("DELETE /api/libraries/", s.handleDeleteLibrary)
	s.mux.HandleFunc("GET /api/tasks", s.handleListTasks)
	s.mux.HandleFunc("POST /api/tasks/", s.handleTaskAction)
	s.mux.HandleFunc("POST /api/organizer/plan", s.handleCreateOrganizerPlan)
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
	default:
		writeError(w, http.StatusBadRequest, fmt.Errorf("unsupported library action %q", action))
	}
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

	taskRecord := s.tasks.Enqueue(task.TypeLibraryScan, "scan library: "+found.Name)
	writeJSON(w, http.StatusAccepted, map[string]any{
		"task":  taskRecord,
		"files": files,
		"count": len(files),
	})
}

func (s *Server) handleListTasks(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, s.tasks.List())
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
	writeJSON(w, http.StatusCreated, plan)
}

func (s *Server) handleListAutomations(w http.ResponseWriter, r *http.Request) {
	automations, err := s.automations.List(r.Context())
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
