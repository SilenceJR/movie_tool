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

func TestAIProviderCRUD(t *testing.T) {
	server := NewServer(config.Config{Host: "127.0.0.1", Port: "0"})

	createResponse := httptest.NewRecorder()
	createRequest := httptest.NewRequest(
		http.MethodPost,
		"/api/ai/providers",
		bytes.NewBufferString(`{"name":"OpenAI","provider_type":"openai","api_key":"secret","default_model":"gpt-4.1-mini","enabled":true}`),
	)
	server.ServeHTTP(createResponse, createRequest)
	if createResponse.Code != http.StatusCreated {
		t.Fatalf("expected 201 provider, got %d body=%s", createResponse.Code, createResponse.Body.String())
	}
	if bytes.Contains(createResponse.Body.Bytes(), []byte("secret")) {
		t.Fatal("provider response leaked api key")
	}

	var created map[string]any
	if err := json.NewDecoder(createResponse.Body).Decode(&created); err != nil {
		t.Fatal(err)
	}
	if created["has_api_key"] != true {
		t.Fatalf("expected has_api_key true, got %#v", created["has_api_key"])
	}
	id := created["id"].(string)

	updateResponse := httptest.NewRecorder()
	updateRequest := httptest.NewRequest(
		http.MethodPatch,
		"/api/ai/providers/"+id,
		bytes.NewBufferString(`{"default_model":"gpt-4.1","enabled":false}`),
	)
	server.ServeHTTP(updateResponse, updateRequest)
	if updateResponse.Code != http.StatusOK {
		t.Fatalf("expected 200 update, got %d body=%s", updateResponse.Code, updateResponse.Body.String())
	}

	testResponse := httptest.NewRecorder()
	testRequest := httptest.NewRequest(http.MethodPost, "/api/ai/providers/"+id+"/test", nil)
	server.ServeHTTP(testResponse, testRequest)
	if testResponse.Code != http.StatusOK {
		t.Fatalf("expected 200 test, got %d body=%s", testResponse.Code, testResponse.Body.String())
	}

	listResponse := httptest.NewRecorder()
	listRequest := httptest.NewRequest(http.MethodGet, "/api/ai/providers", nil)
	server.ServeHTTP(listResponse, listRequest)
	if listResponse.Code != http.StatusOK {
		t.Fatalf("expected 200 list, got %d body=%s", listResponse.Code, listResponse.Body.String())
	}
	var providers []map[string]any
	if err := json.NewDecoder(listResponse.Body).Decode(&providers); err != nil {
		t.Fatal(err)
	}
	if len(providers) != 1 {
		t.Fatalf("expected one provider, got %d", len(providers))
	}

	deleteResponse := httptest.NewRecorder()
	deleteRequest := httptest.NewRequest(http.MethodDelete, "/api/ai/providers/"+id, nil)
	server.ServeHTTP(deleteResponse, deleteRequest)
	if deleteResponse.Code != http.StatusNoContent {
		t.Fatalf("expected 204 delete, got %d body=%s", deleteResponse.Code, deleteResponse.Body.String())
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

func TestMediaCatalogAndVersions(t *testing.T) {
	server := NewServer(config.Config{Host: "127.0.0.1", Port: "0"})

	createResponse := httptest.NewRecorder()
	createRequest := httptest.NewRequest(
		http.MethodPost,
		"/api/media",
		bytes.NewBufferString(`{"library_id":"library-1","media_type":"movie","title":"Inception","display_title":"盗梦空间","year":2010,"match_status":"matched"}`),
	)
	server.ServeHTTP(createResponse, createRequest)
	if createResponse.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d body=%s", createResponse.Code, createResponse.Body.String())
	}

	var created map[string]any
	if err := json.NewDecoder(createResponse.Body).Decode(&created); err != nil {
		t.Fatal(err)
	}
	mediaID := created["id"].(string)

	listResponse := httptest.NewRecorder()
	listRequest := httptest.NewRequest(http.MethodGet, "/api/media?title=盗梦&type=movie&year=2010", nil)
	server.ServeHTTP(listResponse, listRequest)
	if listResponse.Code != http.StatusOK {
		t.Fatalf("expected 200 list, got %d body=%s", listResponse.Code, listResponse.Body.String())
	}
	var items []map[string]any
	if err := json.NewDecoder(listResponse.Body).Decode(&items); err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 media item, got %d", len(items))
	}

	versionResponse := httptest.NewRecorder()
	versionRequest := httptest.NewRequest(
		http.MethodPost,
		"/api/media/"+mediaID+"/versions",
		bytes.NewBufferString(`{"name":"4K Remux","resolution":"2160p","source":"bluray","quality_score":95,"is_default":true}`),
	)
	server.ServeHTTP(versionResponse, versionRequest)
	if versionResponse.Code != http.StatusCreated {
		t.Fatalf("expected 201 version, got %d body=%s", versionResponse.Code, versionResponse.Body.String())
	}

	secondVersionResponse := httptest.NewRecorder()
	secondVersionRequest := httptest.NewRequest(
		http.MethodPost,
		"/api/media/"+mediaID+"/versions",
		bytes.NewBufferString(`{"name":"1080p Web","resolution":"1080p","source":"web","quality_score":80}`),
	)
	server.ServeHTTP(secondVersionResponse, secondVersionRequest)
	if secondVersionResponse.Code != http.StatusCreated {
		t.Fatalf("expected 201 second version, got %d body=%s", secondVersionResponse.Code, secondVersionResponse.Body.String())
	}
	var secondVersion map[string]any
	if err := json.NewDecoder(secondVersionResponse.Body).Decode(&secondVersion); err != nil {
		t.Fatal(err)
	}

	defaultResponse := httptest.NewRecorder()
	defaultRequest := httptest.NewRequest(http.MethodPost, "/api/media/"+mediaID+"/versions/"+secondVersion["id"].(string)+"/default", nil)
	server.ServeHTTP(defaultResponse, defaultRequest)
	if defaultResponse.Code != http.StatusOK {
		t.Fatalf("expected 200 default, got %d body=%s", defaultResponse.Code, defaultResponse.Body.String())
	}

	versionsResponse := httptest.NewRecorder()
	versionsRequest := httptest.NewRequest(http.MethodGet, "/api/media/"+mediaID+"/versions", nil)
	server.ServeHTTP(versionsResponse, versionsRequest)
	if versionsResponse.Code != http.StatusOK {
		t.Fatalf("expected 200 versions, got %d body=%s", versionsResponse.Code, versionsResponse.Body.String())
	}
	var versions []map[string]any
	if err := json.NewDecoder(versionsResponse.Body).Decode(&versions); err != nil {
		t.Fatal(err)
	}
	if len(versions) != 2 {
		t.Fatalf("expected 2 versions, got %d", len(versions))
	}
	if versions[0]["id"] != secondVersion["id"] || versions[0]["is_default"] != true {
		t.Fatalf("expected second version as default first entry, got %#v", versions[0])
	}
}

func TestPeopleTagsCollectionsMediaQueries(t *testing.T) {
	server := NewServer(config.Config{Host: "127.0.0.1", Port: "0"})

	mediaID := createJSON(t, server, "/api/media", `{"library_id":"library-1","media_type":"av","title":"Movie A"}`)["id"].(string)
	personID := createJSON(t, server, "/api/people", `{"name":"Actor A","localized_name":"演员A"}`)["id"].(string)
	tagID := createJSON(t, server, "/api/tags", `{"name":"Series A","category":"series"}`)["id"].(string)
	collectionID := createJSON(t, server, "/api/collections", `{"name":"Collection A","type":"series"}`)["id"].(string)

	postJSON(t, server, "/api/media/"+mediaID+"/people", `{"person_id":"`+personID+`","role":"actor"}`)
	postJSON(t, server, "/api/media/"+mediaID+"/tags", `{"tag_id":"`+tagID+`","source":"manual"}`)
	postJSON(t, server, "/api/collections/"+collectionID+"/items", `{"media_id":"`+mediaID+`","relation_type":"member"}`)

	assertMediaReference(t, server, "/api/people/"+personID+"/media", mediaID)
	assertMediaReference(t, server, "/api/tags/"+tagID+"/media", mediaID)
	assertMediaReference(t, server, "/api/collections/"+collectionID+"/media", mediaID)
	assertMediaReference(t, server, "/api/media?person_id="+personID+"&tag_id="+tagID+"&collection_id="+collectionID, mediaID)
}

func TestMediaTranslations(t *testing.T) {
	server := NewServer(config.Config{Host: "127.0.0.1", Port: "0"})

	mediaID := createJSON(t, server, "/api/media", `{"library_id":"library-1","media_type":"av","title":"Movie A"}`)["id"].(string)
	translateResponse := httptest.NewRecorder()
	translateRequest := httptest.NewRequest(
		http.MethodPost,
		"/api/media/"+mediaID+"/translate",
		bytes.NewBufferString(`{"source_language":"ja-JP","target_language":"zh-CN","field_name":"overview","source_text":"Japanese overview","translated_text":"中文简介","provider":"manual","apply_to_media":true}`),
	)
	server.ServeHTTP(translateResponse, translateRequest)
	if translateResponse.Code != http.StatusCreated {
		t.Fatalf("expected 201 translate, got %d body=%s", translateResponse.Code, translateResponse.Body.String())
	}

	var translated map[string]any
	if err := json.NewDecoder(translateResponse.Body).Decode(&translated); err != nil {
		t.Fatal(err)
	}
	if translated["metadata"] == nil {
		t.Fatalf("expected applied metadata, got %#v", translated)
	}

	assertTranslationList(t, server, "/api/media/"+mediaID+"/translations?language=zh-CN", "中文简介")
	assertTranslationList(t, server, "/api/media-translations?media_id="+mediaID+"&language=zh-CN", "中文简介")

	upsertResponse := httptest.NewRecorder()
	upsertRequest := httptest.NewRequest(
		http.MethodPost,
		"/api/media-translations",
		bytes.NewBufferString(`{"media_id":"`+mediaID+`","language":"zh-CN","field_name":"title","value":"中文标题","source":"manual"}`),
	)
	server.ServeHTTP(upsertResponse, upsertRequest)
	if upsertResponse.Code != http.StatusCreated {
		t.Fatalf("expected 201 upsert translation, got %d body=%s", upsertResponse.Code, upsertResponse.Body.String())
	}
}

func TestGetTask(t *testing.T) {
	server := NewServer(config.Config{Host: "127.0.0.1", Port: "0"})

	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "Inception.2010.mkv"), []byte("movie"), 0o644); err != nil {
		t.Fatal(err)
	}
	createResponse := httptest.NewRecorder()
	createRequest := httptest.NewRequest(
		http.MethodPost,
		"/api/libraries",
		bytes.NewBufferString(`{"name":"Movies","media_type":"movie","path":"`+root+`"}`),
	)
	server.ServeHTTP(createResponse, createRequest)

	var library map[string]any
	if err := json.NewDecoder(createResponse.Body).Decode(&library); err != nil {
		t.Fatal(err)
	}

	scanResponse := httptest.NewRecorder()
	scanRequest := httptest.NewRequest(http.MethodPost, "/api/libraries/"+library["id"].(string)+"/scan", nil)
	server.ServeHTTP(scanResponse, scanRequest)
	if scanResponse.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d", scanResponse.Code)
	}

	var scanBody map[string]any
	if err := json.NewDecoder(scanResponse.Body).Decode(&scanBody); err != nil {
		t.Fatal(err)
	}
	taskBody := scanBody["task"].(map[string]any)
	taskID := taskBody["id"].(string)

	getResponse := httptest.NewRecorder()
	getRequest := httptest.NewRequest(http.MethodGet, "/api/tasks/"+taskID, nil)
	server.ServeHTTP(getResponse, getRequest)
	if getResponse.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", getResponse.Code, getResponse.Body.String())
	}

	logsResponse := httptest.NewRecorder()
	logsRequest := httptest.NewRequest(http.MethodGet, "/api/tasks/"+taskID+"/logs", nil)
	server.ServeHTTP(logsResponse, logsRequest)
	if logsResponse.Code != http.StatusOK {
		t.Fatalf("expected 200 logs, got %d body=%s", logsResponse.Code, logsResponse.Body.String())
	}

	var logs []map[string]any
	if err := json.NewDecoder(logsResponse.Body).Decode(&logs); err != nil {
		t.Fatal(err)
	}
	if len(logs) == 0 {
		t.Fatal("expected scan task logs")
	}

	cancelResponse := httptest.NewRecorder()
	cancelRequest := httptest.NewRequest(http.MethodPost, "/api/tasks/"+taskID+"/cancel", nil)
	server.ServeHTTP(cancelResponse, cancelRequest)
	if cancelResponse.Code != http.StatusOK {
		t.Fatalf("expected 200 cancel, got %d body=%s", cancelResponse.Code, cancelResponse.Body.String())
	}

	var canceled map[string]any
	if err := json.NewDecoder(cancelResponse.Body).Decode(&canceled); err != nil {
		t.Fatal(err)
	}
	if canceled["status"] != "canceled" {
		t.Fatalf("expected canceled status, got %#v", canceled["status"])
	}

	canceledListResponse := httptest.NewRecorder()
	canceledListRequest := httptest.NewRequest(http.MethodGet, "/api/tasks?status=canceled", nil)
	server.ServeHTTP(canceledListResponse, canceledListRequest)
	if canceledListResponse.Code != http.StatusOK {
		t.Fatalf("expected 200 canceled list, got %d body=%s", canceledListResponse.Code, canceledListResponse.Body.String())
	}

	var canceledTasks []map[string]any
	if err := json.NewDecoder(canceledListResponse.Body).Decode(&canceledTasks); err != nil {
		t.Fatal(err)
	}
	if len(canceledTasks) != 1 {
		t.Fatalf("expected 1 canceled task, got %d", len(canceledTasks))
	}
}

func createJSON(t *testing.T, server http.Handler, path string, body string) map[string]any {
	t.Helper()
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, path, bytes.NewBufferString(body))
	server.ServeHTTP(response, request)
	if response.Code != http.StatusCreated {
		t.Fatalf("expected 201 from %s, got %d body=%s", path, response.Code, response.Body.String())
	}
	var decoded map[string]any
	if err := json.NewDecoder(response.Body).Decode(&decoded); err != nil {
		t.Fatal(err)
	}
	return decoded
}

func postJSON(t *testing.T, server http.Handler, path string, body string) {
	t.Helper()
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, path, bytes.NewBufferString(body))
	server.ServeHTTP(response, request)
	if response.Code != http.StatusCreated {
		t.Fatalf("expected 201 from %s, got %d body=%s", path, response.Code, response.Body.String())
	}
}

func assertMediaReference(t *testing.T, server http.Handler, path string, mediaID string) {
	t.Helper()
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, path, nil)
	server.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("expected 200 from %s, got %d body=%s", path, response.Code, response.Body.String())
	}
	var items []map[string]any
	if err := json.NewDecoder(response.Body).Decode(&items); err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0]["id"] != mediaID {
		t.Fatalf("expected media %s from %s, got %#v", mediaID, path, items)
	}
}

func assertTranslationList(t *testing.T, server http.Handler, path string, value string) {
	t.Helper()
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, path, nil)
	server.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("expected 200 from %s, got %d body=%s", path, response.Code, response.Body.String())
	}
	var items []map[string]any
	if err := json.NewDecoder(response.Body).Decode(&items); err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0]["value"] != value {
		t.Fatalf("expected translated value %q from %s, got %#v", value, path, items)
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

	planID := plan["id"].(string)
	getResponse := httptest.NewRecorder()
	getRequest := httptest.NewRequest(http.MethodGet, "/api/organizer/plans/"+planID, nil)
	server.ServeHTTP(getResponse, getRequest)
	if getResponse.Code != http.StatusOK {
		t.Fatalf("expected 200 getting plan, got %d body=%s", getResponse.Code, getResponse.Body.String())
	}

	actionsResponse := httptest.NewRecorder()
	actionsRequest := httptest.NewRequest(http.MethodGet, "/api/organizer/actions?plan_id="+planID, nil)
	server.ServeHTTP(actionsResponse, actionsRequest)
	if actionsResponse.Code != http.StatusOK {
		t.Fatalf("expected 200 listing actions, got %d body=%s", actionsResponse.Code, actionsResponse.Body.String())
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

	pathResponse := httptest.NewRecorder()
	pathRequest := httptest.NewRequest(http.MethodGet, "/api/media-files?path="+filepath.Join(root, "Inception.2010.mkv"), nil)
	server.ServeHTTP(pathResponse, pathRequest)
	if pathResponse.Code != http.StatusOK {
		t.Fatalf("expected 200 getting media file by path, got %d body=%s", pathResponse.Code, pathResponse.Body.String())
	}

	var fileByPath map[string]any
	if err := json.NewDecoder(pathResponse.Body).Decode(&fileByPath); err != nil {
		t.Fatal(err)
	}
	if fileByPath["parsed_title"] != "Inception" {
		t.Fatalf("expected parsed title by path, got %#v", fileByPath["parsed_title"])
	}
}

func TestScanLibraryMarksMissingFiles(t *testing.T) {
	root := t.TempDir()
	keepPath := filepath.Join(root, "Keep.2020.mkv")
	missingPath := filepath.Join(root, "Gone.2020.mkv")
	if err := os.WriteFile(keepPath, []byte("keep"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(missingPath, []byte("gone"), 0o644); err != nil {
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
	libraryID := created["id"].(string)

	firstScan := httptest.NewRecorder()
	server.ServeHTTP(firstScan, httptest.NewRequest(http.MethodPost, "/api/libraries/"+libraryID+"/scan", nil))
	if firstScan.Code != http.StatusAccepted {
		t.Fatalf("expected first scan 202, got %d", firstScan.Code)
	}

	if err := os.Remove(missingPath); err != nil {
		t.Fatal(err)
	}

	secondScan := httptest.NewRecorder()
	server.ServeHTTP(secondScan, httptest.NewRequest(http.MethodPost, "/api/libraries/"+libraryID+"/scan", nil))
	if secondScan.Code != http.StatusAccepted {
		t.Fatalf("expected second scan 202, got %d body=%s", secondScan.Code, secondScan.Body.String())
	}

	var body map[string]any
	if err := json.NewDecoder(secondScan.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body["missing_count"].(float64) != 1 {
		t.Fatalf("expected missing_count 1, got %#v", body["missing_count"])
	}

	missingResponse := httptest.NewRecorder()
	missingRequest := httptest.NewRequest(http.MethodGet, "/api/media-files?library_id="+libraryID+"&file_status=missing", nil)
	server.ServeHTTP(missingResponse, missingRequest)
	if missingResponse.Code != http.StatusOK {
		t.Fatalf("expected 200 missing list, got %d body=%s", missingResponse.Code, missingResponse.Body.String())
	}

	var missingFiles []map[string]any
	if err := json.NewDecoder(missingResponse.Body).Decode(&missingFiles); err != nil {
		t.Fatal(err)
	}
	if len(missingFiles) != 1 {
		t.Fatalf("expected 1 missing file, got %d", len(missingFiles))
	}

	cleanupResponse := httptest.NewRecorder()
	cleanupRequest := httptest.NewRequest(http.MethodPost, "/api/libraries/"+libraryID+"/cleanup-missing", nil)
	server.ServeHTTP(cleanupResponse, cleanupRequest)
	if cleanupResponse.Code != http.StatusOK {
		t.Fatalf("expected 200 cleanup, got %d body=%s", cleanupResponse.Code, cleanupResponse.Body.String())
	}

	var cleanupBody map[string]any
	if err := json.NewDecoder(cleanupResponse.Body).Decode(&cleanupBody); err != nil {
		t.Fatal(err)
	}
	if cleanupBody["deleted_count"].(float64) != 1 {
		t.Fatalf("expected deleted_count 1, got %#v", cleanupBody["deleted_count"])
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

	disabledListResponse := httptest.NewRecorder()
	disabledListRequest := httptest.NewRequest(http.MethodGet, "/api/automations?enabled=false", nil)
	server.ServeHTTP(disabledListResponse, disabledListRequest)
	if disabledListResponse.Code != http.StatusOK {
		t.Fatalf("expected 200 disabled list, got %d body=%s", disabledListResponse.Code, disabledListResponse.Body.String())
	}

	var disabledAutomations []map[string]any
	if err := json.NewDecoder(disabledListResponse.Body).Decode(&disabledAutomations); err != nil {
		t.Fatal(err)
	}
	if len(disabledAutomations) != 1 {
		t.Fatalf("expected 1 disabled automation, got %d", len(disabledAutomations))
	}
}

func TestScrapeCandidates(t *testing.T) {
	server := NewServer(config.Config{Host: "127.0.0.1", Port: "0"})

	createResponse := httptest.NewRecorder()
	createRequest := httptest.NewRequest(
		http.MethodPost,
		"/api/scrape-candidates",
		bytes.NewBufferString(`{
			"media_file_id":"file-1",
			"provider":"tmdb",
			"external_id":"27205",
			"title":"Inception",
			"year":2010,
			"score":95,
			"score_reasons":["title_similarity","year_match"]
		}`),
	)
	server.ServeHTTP(createResponse, createRequest)
	if createResponse.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d body=%s", createResponse.Code, createResponse.Body.String())
	}

	listResponse := httptest.NewRecorder()
	listRequest := httptest.NewRequest(http.MethodGet, "/api/scrape-candidates?media_file_id=file-1", nil)
	server.ServeHTTP(listResponse, listRequest)
	if listResponse.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", listResponse.Code, listResponse.Body.String())
	}

	var candidates []map[string]any
	if err := json.NewDecoder(listResponse.Body).Decode(&candidates); err != nil {
		t.Fatal(err)
	}
	if len(candidates) != 1 {
		t.Fatalf("expected 1 candidate, got %d", len(candidates))
	}
	if candidates[0]["title"] != "Inception" {
		t.Fatalf("expected Inception candidate, got %#v", candidates[0]["title"])
	}
}

func TestScrapeDecisions(t *testing.T) {
	server := NewServer(config.Config{Host: "127.0.0.1", Port: "0"})

	createResponse := httptest.NewRecorder()
	createRequest := httptest.NewRequest(
		http.MethodPost,
		"/api/scrape-decisions",
		bytes.NewBufferString(`{
			"media_id":"media-1",
			"candidate_id":"candidate-1",
			"decision_source":"user",
			"decision":"select",
			"confidence":95,
			"reason":"用户确认",
			"locked":true
		}`),
	)
	server.ServeHTTP(createResponse, createRequest)
	if createResponse.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d body=%s", createResponse.Code, createResponse.Body.String())
	}

	listResponse := httptest.NewRecorder()
	listRequest := httptest.NewRequest(http.MethodGet, "/api/scrape-decisions?media_id=media-1", nil)
	server.ServeHTTP(listResponse, listRequest)
	if listResponse.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", listResponse.Code, listResponse.Body.String())
	}

	var decisions []map[string]any
	if err := json.NewDecoder(listResponse.Body).Decode(&decisions); err != nil {
		t.Fatal(err)
	}
	if len(decisions) != 1 {
		t.Fatalf("expected 1 decision, got %d", len(decisions))
	}
	if decisions[0]["locked"] != true {
		t.Fatalf("expected locked decision, got %#v", decisions[0]["locked"])
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
