package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"movie-tool/backend/internal/config"
)

func TestHealth(t *testing.T) {
	server := NewServer(config.Config{Host: "127.0.0.1", Port: "0"})
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/health", nil)

	server.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", response.Code)
	}

	var body map[string]string
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body["status"] != "ok" {
		t.Fatalf("expected ok status, got %q", body["status"])
	}
}

func TestCreateAndListLibraries(t *testing.T) {
	server := NewServer(config.Config{Host: "127.0.0.1", Port: "0"})

	createResponse := httptest.NewRecorder()
	createRequest := httptest.NewRequest(
		http.MethodPost,
		"/api/libraries",
		bytes.NewBufferString(`{"name":"Movies","media_type":"movie","path":"/media/movies"}`),
	)
	server.ServeHTTP(createResponse, createRequest)

	if createResponse.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d body=%s", createResponse.Code, createResponse.Body.String())
	}

	listResponse := httptest.NewRecorder()
	listRequest := httptest.NewRequest(http.MethodGet, "/api/libraries", nil)
	server.ServeHTTP(listResponse, listRequest)

	if listResponse.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", listResponse.Code)
	}

	var libraries []map[string]any
	if err := json.NewDecoder(listResponse.Body).Decode(&libraries); err != nil {
		t.Fatal(err)
	}
	if len(libraries) != 1 {
		t.Fatalf("expected 1 library, got %d", len(libraries))
	}
}

func TestLibraryDetailUpdateDelete(t *testing.T) {
	server := NewServer(config.Config{Host: "127.0.0.1", Port: "0"})

	createResponse := httptest.NewRecorder()
	createRequest := httptest.NewRequest(
		http.MethodPost,
		"/api/libraries",
		bytes.NewBufferString(`{"name":"Movies","media_type":"movie","path":"/media/movies"}`),
	)
	server.ServeHTTP(createResponse, createRequest)
	if createResponse.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d body=%s", createResponse.Code, createResponse.Body.String())
	}

	var created map[string]any
	if err := json.NewDecoder(createResponse.Body).Decode(&created); err != nil {
		t.Fatal(err)
	}
	id := created["id"].(string)

	updateResponse := httptest.NewRecorder()
	updateRequest := httptest.NewRequest(
		http.MethodPatch,
		"/api/libraries/"+id,
		bytes.NewBufferString(`{"name":"Films","watch_enabled":true}`),
	)
	server.ServeHTTP(updateResponse, updateRequest)
	if updateResponse.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", updateResponse.Code, updateResponse.Body.String())
	}

	getResponse := httptest.NewRecorder()
	getRequest := httptest.NewRequest(http.MethodGet, "/api/libraries/"+id, nil)
	server.ServeHTTP(getResponse, getRequest)
	if getResponse.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", getResponse.Code)
	}

	var found map[string]any
	if err := json.NewDecoder(getResponse.Body).Decode(&found); err != nil {
		t.Fatal(err)
	}
	if found["name"] != "Films" {
		t.Fatalf("expected updated name, got %q", found["name"])
	}

	deleteResponse := httptest.NewRecorder()
	deleteRequest := httptest.NewRequest(http.MethodDelete, "/api/libraries/"+id, nil)
	server.ServeHTTP(deleteResponse, deleteRequest)
	if deleteResponse.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", deleteResponse.Code)
	}
}

func TestListTasks(t *testing.T) {
	server := NewServer(config.Config{Host: "127.0.0.1", Port: "0"})
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/tasks", nil)

	server.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", response.Code)
	}

	var tasks []map[string]any
	if err := json.NewDecoder(response.Body).Decode(&tasks); err != nil {
		t.Fatal(err)
	}
	if len(tasks) != 0 {
		t.Fatalf("expected no tasks, got %d", len(tasks))
	}
}

func TestCreateOrganizerPlan(t *testing.T) {
	server := NewServer(config.Config{Host: "127.0.0.1", Port: "0"})
	response := httptest.NewRecorder()
	request := httptest.NewRequest(
		http.MethodPost,
		"/api/organizer/plan",
		bytes.NewBufferString(`{
			"media":{"id":"m1","library_id":"l1","media_type":"movie","title":"Inception","year":2010},
			"versions":[{"id":"v1","resolution":"2160p","source":"bluray"}],
			"files":[{"id":"f1","media_id":"m1","version_id":"v1","path":"/downloads/Inception.mkv"}],
			"rule":{"library_id":"l1","target_root":"/library/movies","action_mode":"move","enabled":true}
		}`),
	)

	server.ServeHTTP(response, request)

	if response.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d body=%s", response.Code, response.Body.String())
	}

	var plan map[string]any
	if err := json.NewDecoder(response.Body).Decode(&plan); err != nil {
		t.Fatal(err)
	}
	if plan["dry_run"] != true {
		t.Fatalf("expected dry_run plan, got %#v", plan["dry_run"])
	}
}

func TestScanLibrary(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "Inception.2010.mkv"), []byte("movie"), 0o644); err != nil {
		t.Fatal(err)
	}

	server := NewServer(config.Config{Host: "127.0.0.1", Port: "0"})
	createResponse := httptest.NewRecorder()
	createRequest := httptest.NewRequest(
		http.MethodPost,
		"/api/libraries",
		bytes.NewBufferString(`{"name":"Movies","media_type":"movie","path":"`+root+`"}`),
	)
	server.ServeHTTP(createResponse, createRequest)
	if createResponse.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", createResponse.Code)
	}

	var created map[string]any
	if err := json.NewDecoder(createResponse.Body).Decode(&created); err != nil {
		t.Fatal(err)
	}

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/libraries/"+created["id"].(string)+"/scan", nil)
	server.ServeHTTP(response, request)

	if response.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d body=%s", response.Code, response.Body.String())
	}

	var body map[string]any
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body["count"].(float64) != 1 {
		t.Fatalf("expected count 1, got %#v", body["count"])
	}

	filesResponse := httptest.NewRecorder()
	filesRequest := httptest.NewRequest(http.MethodGet, "/api/media-files?library_id="+created["id"].(string), nil)
	server.ServeHTTP(filesResponse, filesRequest)
	if filesResponse.Code != http.StatusOK {
		t.Fatalf("expected 200 listing media files, got %d body=%s", filesResponse.Code, filesResponse.Body.String())
	}

	var files []map[string]any
	if err := json.NewDecoder(filesResponse.Body).Decode(&files); err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 {
		t.Fatalf("expected 1 stored media file, got %d", len(files))
	}
	if files[0]["parsed_title"] != "Inception" {
		t.Fatalf("expected parsed title Inception, got %#v", files[0]["parsed_title"])
	}
}

func TestAutomationCRUDAndRun(t *testing.T) {
	server := NewServer(config.Config{Host: "127.0.0.1", Port: "0"})

	createResponse := httptest.NewRecorder()
	createRequest := httptest.NewRequest(
		http.MethodPost,
		"/api/automations",
		bytes.NewBufferString(`{
			"name":"Scan Movies",
			"automation_type":"scan_library",
			"schedule_type":"interval",
			"schedule":"1h",
			"scope":{"library_id":"library-1"}
		}`),
	)
	server.ServeHTTP(createResponse, createRequest)
	if createResponse.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d body=%s", createResponse.Code, createResponse.Body.String())
	}

	var created map[string]any
	if err := json.NewDecoder(createResponse.Body).Decode(&created); err != nil {
		t.Fatal(err)
	}
	id := created["id"].(string)

	pauseResponse := httptest.NewRecorder()
	pauseRequest := httptest.NewRequest(http.MethodPost, "/api/automations/"+id+"/pause", nil)
	server.ServeHTTP(pauseResponse, pauseRequest)
	if pauseResponse.Code != http.StatusOK {
		t.Fatalf("expected 200 pause, got %d", pauseResponse.Code)
	}

	runResponse := httptest.NewRecorder()
	runRequest := httptest.NewRequest(http.MethodPost, "/api/automations/"+id+"/run", nil)
	server.ServeHTTP(runResponse, runRequest)
	if runResponse.Code != http.StatusAccepted {
		t.Fatalf("expected 202 run, got %d body=%s", runResponse.Code, runResponse.Body.String())
	}

	runsResponse := httptest.NewRecorder()
	runsRequest := httptest.NewRequest(http.MethodGet, "/api/automations/"+id+"/runs", nil)
	server.ServeHTTP(runsResponse, runsRequest)
	if runsResponse.Code != http.StatusOK {
		t.Fatalf("expected 200 runs, got %d", runsResponse.Code)
	}

	var runs []map[string]any
	if err := json.NewDecoder(runsResponse.Body).Decode(&runs); err != nil {
		t.Fatal(err)
	}
	if len(runs) != 1 {
		t.Fatalf("expected 1 run, got %d", len(runs))
	}
}

func TestConfig(t *testing.T) {
	server := NewServer(config.Config{
		Host:     "127.0.0.1",
		Port:     "8080",
		DataDir:  "./data",
		CacheDir: "./cache",
		Database: "./data/movie-tool.db",
	})
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/config", nil)

	server.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", response.Code)
	}

	var body map[string]string
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body["database"] != "./data/movie-tool.db" {
		t.Fatalf("expected database config, got %q", body["database"])
	}
}
