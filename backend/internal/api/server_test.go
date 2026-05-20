package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"movie-tool/backend/internal/catalog"
	"movie-tool/backend/internal/config"
	"movie-tool/backend/internal/media"
	"movie-tool/backend/internal/scanner"
	"movie-tool/backend/internal/task"
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

func TestIntegrationCRUDAndRefresh(t *testing.T) {
	server := NewServer(config.Config{Host: "127.0.0.1", Port: "0"})

	createResponse := httptest.NewRecorder()
	createRequest := httptest.NewRequest(
		http.MethodPost,
		"/api/integrations",
		bytes.NewBufferString(`{"name":"Emby","server_type":"emby","base_url":"http://emby.local/","api_key":"secret","enabled":true}`),
	)
	server.ServeHTTP(createResponse, createRequest)
	if createResponse.Code != http.StatusCreated {
		t.Fatalf("expected 201 integration, got %d body=%s", createResponse.Code, createResponse.Body.String())
	}
	if bytes.Contains(createResponse.Body.Bytes(), []byte("secret")) {
		t.Fatal("integration response leaked api key")
	}
	var created map[string]any
	if err := json.NewDecoder(createResponse.Body).Decode(&created); err != nil {
		t.Fatal(err)
	}
	if created["has_api_key"] != true {
		t.Fatalf("expected has_api_key true, got %#v", created["has_api_key"])
	}
	id := created["id"].(string)

	testResponse := httptest.NewRecorder()
	testRequest := httptest.NewRequest(http.MethodPost, "/api/integrations/"+id+"/test", nil)
	server.ServeHTTP(testResponse, testRequest)
	if testResponse.Code != http.StatusOK {
		t.Fatalf("expected 200 test, got %d body=%s", testResponse.Code, testResponse.Body.String())
	}

	refreshResponse := httptest.NewRecorder()
	refreshRequest := httptest.NewRequest(
		http.MethodPost,
		"/api/integrations/"+id+"/refresh",
		bytes.NewBufferString(`{"library_id":"library-1"}`),
	)
	server.ServeHTTP(refreshResponse, refreshRequest)
	if refreshResponse.Code != http.StatusAccepted {
		t.Fatalf("expected 202 refresh, got %d body=%s", refreshResponse.Code, refreshResponse.Body.String())
	}
	var refreshed map[string]any
	if err := json.NewDecoder(refreshResponse.Body).Decode(&refreshed); err != nil {
		t.Fatal(err)
	}
	plan := refreshed["plan"].(map[string]any)
	if plan["endpoint"] != "http://emby.local/Library/Refresh" {
		t.Fatalf("unexpected refresh endpoint %#v", plan["endpoint"])
	}
	taskBody := refreshed["task"].(map[string]any)
	if taskBody["type"] != "refresh_server" {
		t.Fatalf("expected refresh_server task, got %#v", taskBody["type"])
	}

	deleteResponse := httptest.NewRecorder()
	deleteRequest := httptest.NewRequest(http.MethodDelete, "/api/integrations/"+id, nil)
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

	lockResponse := httptest.NewRecorder()
	lockRequest := httptest.NewRequest(http.MethodPost, "/api/media/"+mediaID+"/lock", nil)
	server.ServeHTTP(lockResponse, lockRequest)
	if lockResponse.Code != http.StatusOK {
		t.Fatalf("expected 200 lock, got %d body=%s", lockResponse.Code, lockResponse.Body.String())
	}
	var locked map[string]any
	if err := json.NewDecoder(lockResponse.Body).Decode(&locked); err != nil {
		t.Fatal(err)
	}
	if locked["locked"] != true || locked["match_status"] != "locked" {
		t.Fatalf("expected locked media, got %#v", locked)
	}

	unlockResponse := httptest.NewRecorder()
	unlockRequest := httptest.NewRequest(http.MethodPost, "/api/media/"+mediaID+"/unlock", nil)
	server.ServeHTTP(unlockResponse, unlockRequest)
	if unlockResponse.Code != http.StatusOK {
		t.Fatalf("expected 200 unlock, got %d body=%s", unlockResponse.Code, unlockResponse.Body.String())
	}
	var unlocked map[string]any
	if err := json.NewDecoder(unlockResponse.Body).Decode(&unlocked); err != nil {
		t.Fatal(err)
	}
	if unlocked["locked"] != false || unlocked["match_status"] != "matched" {
		t.Fatalf("expected unlocked media, got %#v", unlocked)
	}

	rescrapeResponse := httptest.NewRecorder()
	rescrapeRequest := httptest.NewRequest(http.MethodPost, "/api/media/"+mediaID+"/rescrape", nil)
	server.ServeHTTP(rescrapeResponse, rescrapeRequest)
	if rescrapeResponse.Code != http.StatusAccepted {
		t.Fatalf("expected 202 rescrape, got %d body=%s", rescrapeResponse.Code, rescrapeResponse.Body.String())
	}
	var rescrape map[string]any
	if err := json.NewDecoder(rescrapeResponse.Body).Decode(&rescrape); err != nil {
		t.Fatal(err)
	}
	taskBody := rescrape["task"].(map[string]any)
	if taskBody["type"] != "scrape_media" {
		t.Fatalf("expected scrape_media task, got %#v", taskBody["type"])
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

func TestSTRMRuleAndGeneratePlan(t *testing.T) {
	server := NewServer(config.Config{Host: "127.0.0.1", Port: "0"})

	root := t.TempDir()
	output := filepath.Join(t.TempDir(), "strm")
	if err := os.MkdirAll(filepath.Join(root, "Movies"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "Movies", "Inception.2010.mkv"), []byte("movie"), 0o644); err != nil {
		t.Fatal(err)
	}

	library := createJSON(t, server, "/api/libraries", `{"name":"Movies","media_type":"movie","path":"`+root+`"}`)
	scanResponse := httptest.NewRecorder()
	scanRequest := httptest.NewRequest(http.MethodPost, "/api/libraries/"+library["id"].(string)+"/scan", nil)
	server.ServeHTTP(scanResponse, scanRequest)
	if scanResponse.Code != http.StatusAccepted {
		t.Fatalf("expected 202 scan, got %d body=%s", scanResponse.Code, scanResponse.Body.String())
	}

	rule := createJSON(t, server, "/api/strm/rules", `{"name":"LAN","source_prefix":"`+root+`","target_prefix":"http://nas.local/media","output_path":"`+output+`","enabled":true}`)
	validateResponse := httptest.NewRecorder()
	validateRequest := httptest.NewRequest(
		http.MethodPost,
		"/api/strm/validate",
		bytes.NewBufferString(`{"rule":{"name":"LAN","source_prefix":"`+root+`","target_prefix":"http://nas.local/media","output_path":"`+output+`"},"path":"`+filepath.Join(root, "Movies", "Inception.2010.mkv")+`"}`),
	)
	server.ServeHTTP(validateResponse, validateRequest)
	if validateResponse.Code != http.StatusOK {
		t.Fatalf("expected 200 validate, got %d body=%s", validateResponse.Code, validateResponse.Body.String())
	}

	generateResponse := httptest.NewRecorder()
	generateRequest := httptest.NewRequest(
		http.MethodPost,
		"/api/strm/generate",
		bytes.NewBufferString(`{"rule_id":"`+rule["id"].(string)+`","library_id":"`+library["id"].(string)+`","dry_run":true}`),
	)
	server.ServeHTTP(generateResponse, generateRequest)
	if generateResponse.Code != http.StatusAccepted {
		t.Fatalf("expected 202 generate, got %d body=%s", generateResponse.Code, generateResponse.Body.String())
	}

	var body map[string]any
	if err := json.NewDecoder(generateResponse.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	plan := body["plan"].(map[string]any)
	entries := plan["entries"].([]any)
	if len(entries) != 1 {
		t.Fatalf("expected 1 strm entry, got %d", len(entries))
	}
	entry := entries[0].(map[string]any)
	if entry["target_url"] != "http://nas.local/media/Movies/Inception.2010.mkv" {
		t.Fatalf("unexpected target url %#v", entry["target_url"])
	}
	if entry["content"] != "http://nas.local/media/Movies/Inception.2010.mkv\n" {
		t.Fatalf("unexpected strm content %#v", entry["content"])
	}
}

func TestGenerateNFOFromMediaAndTranslations(t *testing.T) {
	server := NewServer(config.Config{Host: "127.0.0.1", Port: "0"})

	mediaID := createJSON(t, server, "/api/media", `{"library_id":"library-1","media_type":"movie","title":"Inception","original_title":"Inception","year":2010,"overview":"English plot"}`)["id"].(string)
	postJSON(t, server, "/api/media-translations", `{"media_id":"`+mediaID+`","language":"zh-CN","field_name":"title","value":"盗梦空间","source":"manual"}`)
	postJSON(t, server, "/api/media-translations", `{"media_id":"`+mediaID+`","language":"zh-CN","field_name":"overview","value":"中文简介","source":"manual"}`)

	response := httptest.NewRecorder()
	request := httptest.NewRequest(
		http.MethodPost,
		"/api/nfo/generate",
		bytes.NewBufferString(`{"media_id":"`+mediaID+`","language":"zh-CN","output_dir":"/library/Inception","dry_run":true}`),
	)
	server.ServeHTTP(response, request)
	if response.Code != http.StatusAccepted {
		t.Fatalf("expected 202 nfo generate, got %d body=%s", response.Code, response.Body.String())
	}

	var body map[string]any
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	taskBody := body["task"].(map[string]any)
	if taskBody["type"] != "generate_nfo" {
		t.Fatalf("expected generate_nfo task, got %#v", taskBody["type"])
	}
	plan := body["plan"].(map[string]any)
	entries := plan["entries"].([]any)
	if len(entries) != 1 {
		t.Fatalf("expected one nfo entry, got %d", len(entries))
	}
	entry := entries[0].(map[string]any)
	content := entry["content"].(string)
	if !strings.Contains(content, "<title>盗梦空间</title>") {
		t.Fatalf("expected localized title in nfo:\n%s", content)
	}
	if !strings.Contains(content, "<plot>中文简介</plot>") {
		t.Fatalf("expected localized plot in nfo:\n%s", content)
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
	if taskBody["status"] != "succeeded" {
		t.Fatalf("expected completed scan task, got %#v", taskBody["status"])
	}

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
	if logs[len(logs)-1]["message"] != "task succeeded" {
		t.Fatalf("expected final success log, got %#v", logs[len(logs)-1])
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
	for _, item := range items {
		if item["value"] == value {
			return
		}
	}
	if len(items) == 0 {
		t.Fatalf("expected translated value %q from %s, got %#v", value, path, items)
	}
	t.Fatalf("expected translated value %q from %s, got %#v", value, path, items)
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

func TestCreateOrganizerPlanFiltersActionsByStatus(t *testing.T) {
	server := NewServer(config.Config{Host: "127.0.0.1", Port: "0"})
	root := t.TempDir()
	targetRoot := filepath.Join(root, "library")
	existingTarget := filepath.Join(targetRoot, "Inception (2010)", "Inception - 1080p.mkv")
	if err := os.MkdirAll(filepath.Dir(existingTarget), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(existingTarget, []byte("existing"), 0o644); err != nil {
		t.Fatal(err)
	}

	plan := createJSON(t, server, "/api/organizer/plan", `{
		"media":{"id":"m1","library_id":"l1","media_type":"movie","title":"Inception","year":2010},
		"versions":[{"id":"v1","resolution":"1080p"},{"id":"v2","resolution":"2160p"}],
		"files":[
			{"id":"f1","media_id":"m1","version_id":"v1","path":"/downloads/Inception.mkv"},
			{"id":"f2","media_id":"m1","version_id":"v2","path":"/downloads/Inception.2160p.mkv","file_name":"Inception.2160p.mkv"}
		],
		"rule":{"library_id":"l1","target_root":"`+targetRoot+`","folder_template":"{{title}} ({{year}})","file_template":"{{title}} - {{resolution}}","action_mode":"copy","conflict_policy":"overwrite_with_confirmation","enabled":true},
		"action_status":"conflict"
	}`)

	actions := plan["actions"].([]any)
	if len(actions) != 1 {
		t.Fatalf("expected one conflict action, got %d", len(actions))
	}
	action := actions[0].(map[string]any)
	if action["status"] != "conflict" || action["target_path"] != existingTarget {
		t.Fatalf("expected filtered conflict action, got %#v", action)
	}
	summary := plan["summary"].(map[string]any)
	if summary["total_actions"].(float64) != 1 || summary["conflict_count"].(float64) != 1 {
		t.Fatalf("expected filtered conflict summary, got %#v", summary)
	}
}

func TestSkipOrganizerPlanConflicts(t *testing.T) {
	server := NewServer(config.Config{Host: "127.0.0.1", Port: "0"})
	root := t.TempDir()
	targetRoot := filepath.Join(root, "library")
	existingTarget := filepath.Join(targetRoot, "Inception (2010)", "Inception - 1080p.mkv")
	if err := os.MkdirAll(filepath.Dir(existingTarget), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(existingTarget, []byte("existing"), 0o644); err != nil {
		t.Fatal(err)
	}

	plan := createJSON(t, server, "/api/organizer/plan", `{
		"media":{"id":"m1","library_id":"l1","media_type":"movie","title":"Inception","year":2010},
		"versions":[{"id":"v1","resolution":"1080p"}],
		"files":[{"id":"f1","media_id":"m1","version_id":"v1","path":"/downloads/Inception.mkv"}],
		"rule":{"library_id":"l1","target_root":"`+targetRoot+`","folder_template":"{{title}} ({{year}})","file_template":"{{title}} - {{resolution}}","action_mode":"copy","conflict_policy":"overwrite_with_confirmation","enabled":true}
	}`)

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/organizer/plans/"+plan["id"].(string)+"/skip-conflicts", nil)
	server.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("expected 200 skip conflicts, got %d body=%s", response.Code, response.Body.String())
	}
	var body map[string]any
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body["changed"].(float64) != 1 {
		t.Fatalf("expected one changed conflict, got %#v", body)
	}
	updated := body["plan"].(map[string]any)
	actions := updated["actions"].([]any)
	action := actions[0].(map[string]any)
	if action["status"] != "skipped" {
		t.Fatalf("expected skipped conflict action, got %#v", action)
	}
}

func TestRenameOrganizerPlanConflicts(t *testing.T) {
	server := NewServer(config.Config{Host: "127.0.0.1", Port: "0"})
	root := t.TempDir()
	targetRoot := filepath.Join(root, "library")
	existingTarget := filepath.Join(targetRoot, "Inception (2010)", "Inception - 1080p.mkv")
	existingRenamedTarget := filepath.Join(targetRoot, "Inception (2010)", "Inception - 1080p (1).mkv")
	if err := os.MkdirAll(filepath.Dir(existingTarget), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(existingTarget, []byte("existing"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(existingRenamedTarget, []byte("existing renamed"), 0o644); err != nil {
		t.Fatal(err)
	}

	plan := createJSON(t, server, "/api/organizer/plan", `{
		"media":{"id":"m1","library_id":"l1","media_type":"movie","title":"Inception","year":2010},
		"versions":[{"id":"v1","resolution":"1080p"}],
		"files":[{"id":"f1","media_id":"m1","version_id":"v1","path":"/downloads/Inception.mkv"}],
		"rule":{"library_id":"l1","target_root":"`+targetRoot+`","folder_template":"{{title}} ({{year}})","file_template":"{{title}} - {{resolution}}","action_mode":"copy","conflict_policy":"overwrite_with_confirmation","enabled":true}
	}`)

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/organizer/plans/"+plan["id"].(string)+"/rename-conflicts", nil)
	server.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("expected 200 rename conflicts, got %d body=%s", response.Code, response.Body.String())
	}
	var body map[string]any
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	updated := body["plan"].(map[string]any)
	actions := updated["actions"].([]any)
	action := actions[0].(map[string]any)
	want := filepath.Join(targetRoot, "Inception (2010)", "Inception - 1080p (2).mkv")
	if action["status"] != "pending" || action["target_path"] != want {
		t.Fatalf("expected pending renamed conflict target %q, got %#v", want, action)
	}
}

func TestExecuteOrganizerPlan(t *testing.T) {
	server := NewServer(config.Config{Host: "127.0.0.1", Port: "0"})
	root := t.TempDir()
	source := filepath.Join(root, "downloads", "Inception.mkv")
	targetRoot := filepath.Join(root, "library")
	if err := os.MkdirAll(filepath.Dir(source), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(source, []byte("movie"), 0o644); err != nil {
		t.Fatal(err)
	}

	plan := createJSON(t, server, "/api/organizer/plan", `{
		"media":{"id":"m1","library_id":"l1","media_type":"movie","title":"Inception","year":2010},
		"versions":[{"id":"v1","resolution":"1080p","source":"web-dl"}],
		"files":[{"id":"f1","media_id":"m1","version_id":"v1","path":"`+source+`"}],
		"rule":{"library_id":"l1","target_root":"`+targetRoot+`","folder_template":"{{title}} ({{year}})","file_template":"{{title}} - {{resolution}}","action_mode":"copy","enabled":true}
	}`)
	planID := plan["id"].(string)

	executeResponse := httptest.NewRecorder()
	executeRequest := httptest.NewRequest(http.MethodPost, "/api/organizer/plans/"+planID+"/execute", nil)
	server.ServeHTTP(executeResponse, executeRequest)
	if executeResponse.Code != http.StatusAccepted {
		t.Fatalf("expected 202 execute, got %d body=%s", executeResponse.Code, executeResponse.Body.String())
	}

	var body map[string]any
	if err := json.NewDecoder(executeResponse.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	executed := body["plan"].(map[string]any)
	if executed["status"] != "succeeded" || executed["dry_run"] != false {
		t.Fatalf("expected executed plan, got %#v", executed)
	}
	actions := executed["actions"].([]any)
	action := actions[0].(map[string]any)
	if action["status"] != "succeeded" {
		t.Fatalf("expected succeeded action, got %#v", action)
	}
	target := filepath.Join(targetRoot, "Inception (2010)", "Inception - 1080p.mkv")
	content, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("expected copied target: %v", err)
	}
	if string(content) != "movie" {
		t.Fatalf("unexpected target content %q", string(content))
	}

	logsResponse := httptest.NewRecorder()
	taskBody := body["task"].(map[string]any)
	logsRequest := httptest.NewRequest(http.MethodGet, "/api/tasks/"+taskBody["id"].(string)+"/logs", nil)
	server.ServeHTTP(logsResponse, logsRequest)
	if logsResponse.Code != http.StatusOK {
		t.Fatalf("expected task logs, got %d body=%s", logsResponse.Code, logsResponse.Body.String())
	}
}

func TestOrganizerRulesAndPlanByRuleID(t *testing.T) {
	server := NewServer(config.Config{Host: "127.0.0.1", Port: "0"})

	rule := createJSON(t, server, "/api/organizer/rules", `{"name":"Movies","library_id":"l1","media_type":"movie","target_root":"/library/movies","folder_template":"{{title}} ({{year}})","file_template":"{{title}} - {{resolution}}","action_mode":"move","enabled":true}`)
	ruleID := rule["id"].(string)

	updateResponse := httptest.NewRecorder()
	updateRequest := httptest.NewRequest(
		http.MethodPatch,
		"/api/organizer/rules/"+ruleID,
		bytes.NewBufferString(`{"conflict_policy":"rename"}`),
	)
	server.ServeHTTP(updateResponse, updateRequest)
	if updateResponse.Code != http.StatusOK {
		t.Fatalf("expected 200 rule update, got %d body=%s", updateResponse.Code, updateResponse.Body.String())
	}

	listResponse := httptest.NewRecorder()
	listRequest := httptest.NewRequest(http.MethodGet, "/api/organizer/rules", nil)
	server.ServeHTTP(listResponse, listRequest)
	if listResponse.Code != http.StatusOK {
		t.Fatalf("expected 200 rule list, got %d body=%s", listResponse.Code, listResponse.Body.String())
	}

	planResponse := httptest.NewRecorder()
	planRequest := httptest.NewRequest(
		http.MethodPost,
		"/api/organizer/plan",
		bytes.NewBufferString(`{
			"media":{"id":"m1","library_id":"l1","media_type":"movie","title":"Inception","year":2010},
			"versions":[{"id":"v1","resolution":"2160p","source":"bluray"}],
			"files":[{"id":"f1","media_id":"m1","version_id":"v1","path":"/downloads/Inception.mkv"}],
			"rule_id":"`+ruleID+`"
		}`),
	)
	server.ServeHTTP(planResponse, planRequest)
	if planResponse.Code != http.StatusCreated {
		t.Fatalf("expected 201 plan, got %d body=%s", planResponse.Code, planResponse.Body.String())
	}
	var plan map[string]any
	if err := json.NewDecoder(planResponse.Body).Decode(&plan); err != nil {
		t.Fatal(err)
	}
	actions := plan["actions"].([]any)
	action := actions[0].(map[string]any)
	if action["target_path"] != "/library/movies/Inception (2010)/Inception - 2160p.mkv" {
		t.Fatalf("unexpected target path %#v", action["target_path"])
	}

	deleteResponse := httptest.NewRecorder()
	deleteRequest := httptest.NewRequest(http.MethodDelete, "/api/organizer/rules/"+ruleID, nil)
	server.ServeHTTP(deleteResponse, deleteRequest)
	if deleteResponse.Code != http.StatusNoContent {
		t.Fatalf("expected 204 rule delete, got %d body=%s", deleteResponse.Code, deleteResponse.Body.String())
	}
}

func TestCreateOrganizerPlanByLibraryID(t *testing.T) {
	root := t.TempDir()
	firstPath := filepath.Join(root, "downloads", "Inception.2010.1080p.mkv")
	secondPath := filepath.Join(root, "downloads", "Interstellar.2014.2160p.mkv")
	if err := os.MkdirAll(filepath.Dir(firstPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(firstPath, []byte("first"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(secondPath, []byte("second"), 0o644); err != nil {
		t.Fatal(err)
	}

	server := NewServer(config.Config{Host: "127.0.0.1", Port: "0"})
	library := createJSON(t, server, "/api/libraries", `{"name":"Movies","media_type":"movie","path":"`+filepath.Dir(firstPath)+`"}`)
	libraryID := library["id"].(string)
	scanResponse := httptest.NewRecorder()
	scanRequest := httptest.NewRequest(http.MethodPost, "/api/libraries/"+libraryID+"/scan", nil)
	server.ServeHTTP(scanResponse, scanRequest)
	if scanResponse.Code != http.StatusAccepted {
		t.Fatalf("expected 202 scan, got %d body=%s", scanResponse.Code, scanResponse.Body.String())
	}

	rule := createJSON(t, server, "/api/organizer/rules", `{
		"name":"Movies",
		"library_id":"`+libraryID+`",
		"media_type":"movie",
		"target_root":"`+filepath.Join(root, "library")+`",
		"folder_template":"{{title}} ({{year}})",
		"file_template":"{{title}} - {{resolution}}",
		"action_mode":"copy",
		"enabled":true
	}`)

	planResponse := httptest.NewRecorder()
	planRequest := httptest.NewRequest(
		http.MethodPost,
		"/api/organizer/plan",
		bytes.NewBufferString(`{"rule_id":"`+rule["id"].(string)+`","library_id":"`+libraryID+`"}`),
	)
	server.ServeHTTP(planResponse, planRequest)
	if planResponse.Code != http.StatusCreated {
		t.Fatalf("expected 201 library organizer plan, got %d body=%s", planResponse.Code, planResponse.Body.String())
	}

	var plan map[string]any
	if err := json.NewDecoder(planResponse.Body).Decode(&plan); err != nil {
		t.Fatal(err)
	}
	actions := plan["actions"].([]any)
	if len(actions) != 2 {
		t.Fatalf("expected two library actions, got %d", len(actions))
	}
	targets := []string{
		actions[0].(map[string]any)["target_path"].(string),
		actions[1].(map[string]any)["target_path"].(string),
	}
	joined := strings.Join(targets, "\n")
	if !strings.Contains(joined, "Inception (2010)") || !strings.Contains(joined, "Interstellar (2014)") {
		t.Fatalf("expected per-media target folders, got:\n%s", joined)
	}

	items, err := server.catalog.ListItems(context.Background(), catalog.ItemQuery{LibraryID: libraryID, Title: "Inception"})
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Fatalf("expected scanned Inception item, got %+v", items)
	}
	matched := catalog.MatchStatusMatched
	if _, ok, err := server.catalog.UpdateItem(context.Background(), items[0].ID, catalog.ItemUpdate{MatchStatus: &matched}); err != nil || !ok {
		t.Fatalf("mark item matched: ok=%v err=%v", ok, err)
	}

	filteredResponse := httptest.NewRecorder()
	filteredRequest := httptest.NewRequest(
		http.MethodPost,
		"/api/organizer/plan",
		bytes.NewBufferString(`{"rule_id":"`+rule["id"].(string)+`","library_id":"`+libraryID+`","match_status":"matched","media_type":"movie","file_status":"available"}`),
	)
	server.ServeHTTP(filteredResponse, filteredRequest)
	if filteredResponse.Code != http.StatusCreated {
		t.Fatalf("expected 201 filtered library organizer plan, got %d body=%s", filteredResponse.Code, filteredResponse.Body.String())
	}
	var filteredPlan map[string]any
	if err := json.NewDecoder(filteredResponse.Body).Decode(&filteredPlan); err != nil {
		t.Fatal(err)
	}
	filteredActions := filteredPlan["actions"].([]any)
	if len(filteredActions) != 1 {
		t.Fatalf("expected one filtered library action, got %d", len(filteredActions))
	}
	target := filteredActions[0].(map[string]any)["target_path"].(string)
	if !strings.Contains(target, "Inception (2010)") || strings.Contains(target, "Interstellar") {
		t.Fatalf("expected filtered target for matched Inception only, got %s", target)
	}
}

func TestScanLibrary(t *testing.T) {
	root := t.TempDir()
	filePath := filepath.Join(root, "Inception.2010.2160p.BluRay.REMUX.HEVC.HDR10-GROUP.mkv")
	if err := os.WriteFile(filePath, []byte("movie"), 0o644); err != nil {
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
	if body["batch_count"].(float64) != 1 {
		t.Fatalf("expected batch_count 1, got %#v", body["batch_count"])
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
	if files[0]["media_id"] == "" || files[0]["version_id"] == "" {
		t.Fatalf("expected scan to link media/version, got %#v", files[0])
	}

	pathResponse := httptest.NewRecorder()
	pathRequest := httptest.NewRequest(http.MethodGet, "/api/media-files?path="+filePath, nil)
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

	mediaResponse := httptest.NewRecorder()
	mediaRequest := httptest.NewRequest(http.MethodGet, "/api/media?library_id="+created["id"].(string)+"&title=Inception", nil)
	server.ServeHTTP(mediaResponse, mediaRequest)
	if mediaResponse.Code != http.StatusOK {
		t.Fatalf("expected 200 listing catalog media, got %d body=%s", mediaResponse.Code, mediaResponse.Body.String())
	}
	var items []map[string]any
	if err := json.NewDecoder(mediaResponse.Body).Decode(&items); err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Fatalf("expected one catalog media item, got %d", len(items))
	}
	versionsResponse := httptest.NewRecorder()
	versionsRequest := httptest.NewRequest(http.MethodGet, "/api/media/"+items[0]["id"].(string)+"/versions", nil)
	server.ServeHTTP(versionsResponse, versionsRequest)
	if versionsResponse.Code != http.StatusOK {
		t.Fatalf("expected 200 listing versions, got %d body=%s", versionsResponse.Code, versionsResponse.Body.String())
	}
	var versions []map[string]any
	if err := json.NewDecoder(versionsResponse.Body).Decode(&versions); err != nil {
		t.Fatal(err)
	}
	if len(versions) != 1 {
		t.Fatalf("expected one scanned version, got %d", len(versions))
	}
	if versions[0]["resolution"] != "2160p" || versions[0]["source"] != "remux" || versions[0]["video_codec"] != "hevc" || versions[0]["hdr_format"] != "hdr10" {
		t.Fatalf("unexpected parsed version metadata: %#v", versions[0])
	}
}

func TestScanLibraryBatchSize(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{"Inception.2010.mkv", "Interstellar.2014.mkv", "Tenet.2020.mkv"} {
		if err := os.WriteFile(filepath.Join(root, name), []byte("movie"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	server := NewServer(config.Config{Host: "127.0.0.1", Port: "0"})
	library := createJSON(t, server, "/api/libraries", `{"name":"Movies","media_type":"movie","path":"`+root+`"}`)

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/libraries/"+library["id"].(string)+"/scan?batch_size=2", nil)
	server.ServeHTTP(response, request)
	if response.Code != http.StatusAccepted {
		t.Fatalf("expected 202 batched scan, got %d body=%s", response.Code, response.Body.String())
	}
	var body map[string]any
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body["batch_count"].(float64) != 2 {
		t.Fatalf("expected two scan batches, got %#v", body["batch_count"])
	}
	if len(body["imported"].([]any)) != 3 {
		t.Fatalf("expected three imported files, got %#v", body["imported"])
	}
}

func TestImportScannedFilesContinueOnError(t *testing.T) {
	server := NewServer(config.Config{Host: "127.0.0.1", Port: "0"})
	files := []scanner.ParsedFile{
		{
			LibraryID: "library-1",
			MediaType: "movie",
			Path:      "/downloads/Inception.2010.mkv",
			FileName:  "Inception.2010.mkv",
			Extension: ".mkv",
			Title:     "Inception",
			Year:      2010,
		},
		{
			MediaType: "movie",
			Path:      "/downloads/Broken.2020.mkv",
			FileName:  "Broken.2020.mkv",
			Extension: ".mkv",
			Title:     "Broken",
			Year:      2020,
		},
		{
			LibraryID: "library-1",
			Path:      "/downloads/Unknown.2021.mkv",
			FileName:  "Unknown.2021.mkv",
			Extension: ".mkv",
			Title:     "Unknown",
			Year:      2021,
		},
	}

	imported, missingCount, batches, failures, err := server.importScannedFiles(context.Background(), files, "", 2, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(imported) != 1 || missingCount != 0 || batches != 2 {
		t.Fatalf("unexpected partial import result imported=%d missing=%d batches=%d", len(imported), missingCount, batches)
	}
	if len(failures) != 2 || failures[0].Path != "/downloads/Broken.2020.mkv" || failures[1].Path != "/downloads/Unknown.2021.mkv" {
		t.Fatalf("expected two failed files, got %+v", failures)
	}
	failedFiles, err := server.mediaFiles.ListFiles(context.Background(), media.FileQuery{LibraryID: "library-1", Status: media.FileStatusFailed})
	if err != nil {
		t.Fatal(err)
	}
	if len(failedFiles) != 1 || failedFiles[0].Path != "/downloads/Unknown.2021.mkv" || failedFiles[0].FailureError == "" {
		t.Fatalf("expected failed file to be persisted, got %#v", failedFiles)
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

func TestDownloadDirectoryScanAndOrganizerPlan(t *testing.T) {
	mediaRoot := t.TempDir()
	downloadRoot := t.TempDir()
	otherDownloadRoot := t.TempDir()
	downloadPath := filepath.Join(downloadRoot, "Inception.2010.1080p.WEB-DL.H264-GROUP.mkv")
	if err := os.WriteFile(downloadPath, []byte("movie"), 0o644); err != nil {
		t.Fatal(err)
	}
	otherDownloadPath := filepath.Join(otherDownloadRoot, "Interstellar.2014.1080p.WEB-DL.H264-GROUP.mkv")
	if err := os.WriteFile(otherDownloadPath, []byte("other"), 0o644); err != nil {
		t.Fatal(err)
	}

	server := NewServer(config.Config{Host: "127.0.0.1", Port: "0"})
	library := createJSON(t, server, "/api/libraries", `{"name":"Movies","media_type":"movie","path":"`+mediaRoot+`"}`)
	libraryID := library["id"].(string)
	downloadDir := createJSON(t, server, "/api/download-directories", `{
		"name":"Completed",
		"path":"`+downloadRoot+`",
		"library_id":"`+libraryID+`",
		"action_mode":"hardlink",
		"enabled":true,
		"watch_enabled":true
	}`)
	otherDownloadDir := createJSON(t, server, "/api/download-directories", `{
		"name":"Other Completed",
		"path":"`+otherDownloadRoot+`",
		"library_id":"`+libraryID+`",
		"action_mode":"hardlink",
		"enabled":true,
		"watch_enabled":true
	}`)

	scanResponse := httptest.NewRecorder()
	scanRequest := httptest.NewRequest(http.MethodPost, "/api/download-directories/"+downloadDir["id"].(string)+"/scan", nil)
	server.ServeHTTP(scanResponse, scanRequest)
	if scanResponse.Code != http.StatusAccepted {
		t.Fatalf("expected 202 download scan, got %d body=%s", scanResponse.Code, scanResponse.Body.String())
	}
	var scanBody map[string]any
	if err := json.NewDecoder(scanResponse.Body).Decode(&scanBody); err != nil {
		t.Fatal(err)
	}
	imported := scanBody["imported"].([]any)
	if len(imported) != 1 {
		t.Fatalf("expected 1 imported download file, got %d", len(imported))
	}
	file := imported[0].(map[string]any)
	mediaID := file["media_id"].(string)
	if mediaID == "" {
		t.Fatalf("expected download scan to link media, got %#v", file)
	}
	otherScanResponse := httptest.NewRecorder()
	otherScanRequest := httptest.NewRequest(http.MethodPost, "/api/download-directories/"+otherDownloadDir["id"].(string)+"/scan", nil)
	server.ServeHTTP(otherScanResponse, otherScanRequest)
	if otherScanResponse.Code != http.StatusAccepted {
		t.Fatalf("expected 202 other download scan, got %d body=%s", otherScanResponse.Code, otherScanResponse.Body.String())
	}

	rule := createJSON(t, server, "/api/organizer/rules", `{
		"name":"Movies copy",
		"library_id":"`+libraryID+`",
		"media_type":"movie",
		"target_root":"`+mediaRoot+`",
		"action_mode":"copy",
		"conflict_policy":"skip",
		"enabled":true
	}`)

	scanWithPlanResponse := httptest.NewRecorder()
	scanWithPlanRequest := httptest.NewRequest(http.MethodPost, "/api/download-directories/"+downloadDir["id"].(string)+"/scan?organizer_rule_id="+rule["id"].(string), nil)
	server.ServeHTTP(scanWithPlanResponse, scanWithPlanRequest)
	if scanWithPlanResponse.Code != http.StatusAccepted {
		t.Fatalf("expected 202 download scan with organizer plan, got %d body=%s", scanWithPlanResponse.Code, scanWithPlanResponse.Body.String())
	}
	var scanWithPlanBody map[string]any
	if err := json.NewDecoder(scanWithPlanResponse.Body).Decode(&scanWithPlanBody); err != nil {
		t.Fatal(err)
	}
	planFromScan := scanWithPlanBody["organizer_plan"].(map[string]any)
	planFromScanActions := planFromScan["actions"].([]any)
	if len(planFromScanActions) != 1 {
		t.Fatalf("expected scan-created organizer plan for one download directory file, got %d", len(planFromScanActions))
	}
	if planFromScanActions[0].(map[string]any)["source_path"] != downloadPath {
		t.Fatalf("expected scan-created plan source %q, got %#v", downloadPath, planFromScanActions[0])
	}

	filteredPlanResponse := httptest.NewRecorder()
	filteredPlanRequest := httptest.NewRequest(
		http.MethodPost,
		"/api/organizer/plan",
		bytes.NewBufferString(`{"rule_id":"`+rule["id"].(string)+`","library_id":"`+libraryID+`","download_directory_id":"`+downloadDir["id"].(string)+`"}`),
	)
	server.ServeHTTP(filteredPlanResponse, filteredPlanRequest)
	if filteredPlanResponse.Code != http.StatusCreated {
		t.Fatalf("expected 201 filtered organizer plan, got %d body=%s", filteredPlanResponse.Code, filteredPlanResponse.Body.String())
	}
	var filteredPlan map[string]any
	if err := json.NewDecoder(filteredPlanResponse.Body).Decode(&filteredPlan); err != nil {
		t.Fatal(err)
	}
	filteredActions := filteredPlan["actions"].([]any)
	if len(filteredActions) != 1 {
		t.Fatalf("expected one filtered action, got %d", len(filteredActions))
	}
	filteredSource := filteredActions[0].(map[string]any)["source_path"].(string)
	if filteredSource != downloadPath {
		t.Fatalf("expected filtered source %q, got %q", downloadPath, filteredSource)
	}

	planResponse := httptest.NewRecorder()
	planRequest := httptest.NewRequest(
		http.MethodPost,
		"/api/organizer/plan",
		bytes.NewBufferString(`{"rule_id":"`+rule["id"].(string)+`","media_id":"`+mediaID+`"}`),
	)
	server.ServeHTTP(planResponse, planRequest)
	if planResponse.Code != http.StatusCreated {
		t.Fatalf("expected 201 organizer plan, got %d body=%s", planResponse.Code, planResponse.Body.String())
	}
	var plan map[string]any
	if err := json.NewDecoder(planResponse.Body).Decode(&plan); err != nil {
		t.Fatal(err)
	}
	actions := plan["actions"].([]any)
	if len(actions) != 1 {
		t.Fatalf("expected one organize action, got %d", len(actions))
	}
	action := actions[0].(map[string]any)
	if action["action_type"] != "copy" {
		t.Fatalf("expected copy action, got %#v", action["action_type"])
	}
	if action["source_path"] != downloadPath {
		t.Fatalf("expected source download path, got %#v", action["source_path"])
	}
	if !strings.HasPrefix(action["target_path"].(string), mediaRoot) {
		t.Fatalf("expected target under media root, got %#v", action["target_path"])
	}

	targetPath := action["target_path"].(string)
	executeResponse := httptest.NewRecorder()
	executeRequest := httptest.NewRequest(http.MethodPost, "/api/organizer/plans/"+plan["id"].(string)+"/execute", nil)
	server.ServeHTTP(executeResponse, executeRequest)
	if executeResponse.Code != http.StatusAccepted {
		t.Fatalf("expected 202 execute, got %d body=%s", executeResponse.Code, executeResponse.Body.String())
	}

	if content, err := os.ReadFile(targetPath); err != nil || string(content) != "movie" {
		t.Fatalf("expected organized target content, content=%q err=%v", string(content), err)
	}
	fileResponse := httptest.NewRecorder()
	fileRequest := httptest.NewRequest(http.MethodGet, "/api/media-files?path="+url.QueryEscape(targetPath), nil)
	server.ServeHTTP(fileResponse, fileRequest)
	if fileResponse.Code != http.StatusOK {
		t.Fatalf("expected media file path update, got %d body=%s", fileResponse.Code, fileResponse.Body.String())
	}
	var updatedFile map[string]any
	if err := json.NewDecoder(fileResponse.Body).Decode(&updatedFile); err != nil {
		t.Fatal(err)
	}
	if updatedFile["id"] != file["id"] || updatedFile["path"] != targetPath {
		t.Fatalf("expected media file to point at target path, got %#v", updatedFile)
	}
}

func TestRunDownloadDirectoryWatchScansOnlyEnabledWatchDirectories(t *testing.T) {
	mediaRoot := t.TempDir()
	watchedRoot := t.TempDir()
	disabledRoot := t.TempDir()
	notWatchedRoot := t.TempDir()

	watchedPath := filepath.Join(watchedRoot, "Arrival.2016.1080p.WEB-DL.mkv")
	if err := os.WriteFile(watchedPath, []byte("watched"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(disabledRoot, "Disabled.2011.mkv"), []byte("disabled"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(notWatchedRoot, "Not.Watched.2012.mkv"), []byte("not watched"), 0o644); err != nil {
		t.Fatal(err)
	}

	server := NewServer(config.Config{Host: "127.0.0.1", Port: "0"})
	library := createJSON(t, server, "/api/libraries", `{"name":"Movies","media_type":"movie","path":"`+mediaRoot+`"}`)
	libraryID := library["id"].(string)
	rule := createJSON(t, server, "/api/organizer/rules", `{
		"name":"Watch copy",
		"library_id":"`+libraryID+`",
		"media_type":"movie",
		"target_root":"`+mediaRoot+`",
		"action_mode":"copy",
		"conflict_policy":"skip",
		"enabled":true
	}`)
	watchedDir := createJSON(t, server, "/api/download-directories", `{
		"name":"Watched",
		"path":"`+watchedRoot+`",
		"library_id":"`+libraryID+`",
		"action_mode":"hardlink",
		"organizer_rule_id":"`+rule["id"].(string)+`",
		"enabled":true,
		"watch_enabled":true
	}`)
	_ = createJSON(t, server, "/api/download-directories", `{
		"name":"Disabled",
		"path":"`+disabledRoot+`",
		"library_id":"`+libraryID+`",
		"action_mode":"hardlink",
		"enabled":false,
		"watch_enabled":true
	}`)
	_ = createJSON(t, server, "/api/download-directories", `{
		"name":"Not watched",
		"path":"`+notWatchedRoot+`",
		"library_id":"`+libraryID+`",
		"action_mode":"hardlink",
		"enabled":true,
		"watch_enabled":false
	}`)

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/download-directories/watch/run", nil)
	server.ServeHTTP(response, request)
	if response.Code != http.StatusAccepted {
		t.Fatalf("expected 202 watch run, got %d body=%s", response.Code, response.Body.String())
	}

	var body map[string]any
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body["count"] != float64(1) || body["failure_count"] != float64(0) {
		t.Fatalf("expected one successful watch scan and no failures, got %#v", body)
	}
	taskBody := body["task"].(map[string]any)
	if taskBody["type"] != "download_watch" || taskBody["status"] != "succeeded" {
		t.Fatalf("expected succeeded download_watch parent task, got %#v", taskBody)
	}
	directories := body["download_directories"].([]any)
	if len(directories) != 1 || directories[0].(map[string]any)["id"] != watchedDir["id"] {
		t.Fatalf("expected only watched directory to be scanned, got %#v", directories)
	}
	results := body["results"].([]any)
	imported := results[0].(map[string]any)["imported"].([]any)
	if len(imported) != 1 {
		t.Fatalf("expected one imported watched file, got %#v", imported)
	}
	if imported[0].(map[string]any)["path"] != watchedPath {
		t.Fatalf("expected imported watched path %q, got %#v", watchedPath, imported[0])
	}
	plan := results[0].(map[string]any)["organizer_plan"].(map[string]any)
	actions := plan["actions"].([]any)
	if len(actions) != 1 || actions[0].(map[string]any)["source_path"] != watchedPath {
		t.Fatalf("expected default organizer rule to create plan for watched file, got %#v", plan)
	}
}

func TestRunDownloadDirectoryWatchSkipsWhenAlreadyRunning(t *testing.T) {
	server := NewServer(config.Config{Host: "127.0.0.1", Port: "0"})
	server.downloadWatchMu.Lock()
	defer server.downloadWatchMu.Unlock()

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/download-directories/watch/run", nil)
	server.ServeHTTP(response, request)
	if response.Code != http.StatusAccepted {
		t.Fatalf("expected 202 skipped watch run, got %d body=%s", response.Code, response.Body.String())
	}

	var body map[string]any
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body["skipped"] != true || body["skip_reason"] == "" {
		t.Fatalf("expected skipped watch run response, got %#v", body)
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
	var runBody map[string]any
	if err := json.NewDecoder(runResponse.Body).Decode(&runBody); err != nil {
		t.Fatal(err)
	}
	taskBody := runBody["task"].(map[string]any)
	taskID := taskBody["id"].(string)
	logsResponse := httptest.NewRecorder()
	logsRequest := httptest.NewRequest(http.MethodGet, "/api/tasks/"+taskID+"/logs", nil)
	server.ServeHTTP(logsResponse, logsRequest)
	if logsResponse.Code != http.StatusOK {
		t.Fatalf("expected 200 task logs, got %d body=%s", logsResponse.Code, logsResponse.Body.String())
	}
	var logs []map[string]any
	if err := json.NewDecoder(logsResponse.Body).Decode(&logs); err != nil {
		t.Fatal(err)
	}
	if len(logs) < 3 {
		t.Fatalf("expected automation context logs, got %#v", logs)
	}
	if logs[1]["message"] != "automation "+id+" (scan_library) queued task "+taskID {
		t.Fatalf("expected automation context log, got %#v", logs[1])
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

func TestRunDueAutomations(t *testing.T) {
	server := NewServer(config.Config{Host: "127.0.0.1", Port: "0"})

	created := createJSON(t, server, "/api/automations", `{
		"name":"Due Scan",
		"automation_type":"scan_library",
		"schedule_type":"interval",
		"schedule":"1h",
		"scope":{"library_id":"library-1"}
	}`)
	nextRunAt, err := time.Parse(time.RFC3339Nano, created["next_run_at"].(string))
	if err != nil {
		t.Fatal(err)
	}

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/automations/run-due?now="+url.QueryEscape(nextRunAt.Add(time.Second).Format(time.RFC3339Nano)), nil)
	server.ServeHTTP(response, request)
	if response.Code != http.StatusAccepted {
		t.Fatalf("expected 202 run due, got %d body=%s", response.Code, response.Body.String())
	}
	var body map[string]any
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body["count"].(float64) != 1 {
		t.Fatalf("expected one due automation, got %#v", body)
	}
	results := body["results"].([]any)
	result := results[0].(map[string]any)
	taskBody := result["task"].(map[string]any)
	if taskBody["type"] != "library_scan" {
		t.Fatalf("expected library_scan task, got %#v", taskBody)
	}

	runsResponse := httptest.NewRecorder()
	runsRequest := httptest.NewRequest(http.MethodGet, "/api/automations/"+created["id"].(string)+"/runs", nil)
	server.ServeHTTP(runsResponse, runsRequest)
	if runsResponse.Code != http.StatusOK {
		t.Fatalf("expected 200 runs, got %d body=%s", runsResponse.Code, runsResponse.Body.String())
	}
	var runs []map[string]any
	if err := json.NewDecoder(runsResponse.Body).Decode(&runs); err != nil {
		t.Fatal(err)
	}
	if len(runs) != 1 {
		t.Fatalf("expected one run, got %d", len(runs))
	}
}

func TestRunDueAutomationsServiceMethod(t *testing.T) {
	server := NewServer(config.Config{Host: "127.0.0.1", Port: "0"})

	created := createJSON(t, server, "/api/automations", `{
		"name":"Due Cleanup",
		"automation_type":"cleanup_missing",
		"schedule_type":"interval",
		"schedule":"1h"
	}`)
	nextRunAt, err := time.Parse(time.RFC3339Nano, created["next_run_at"].(string))
	if err != nil {
		t.Fatal(err)
	}

	results, err := server.RunDueAutomations(context.Background(), nextRunAt.Add(time.Second))
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 {
		t.Fatalf("expected one due result, got %#v", results)
	}
	taskBody := results[0]["task"].(task.Task)
	if taskBody.Type != task.TypeCleanupMissing {
		t.Fatalf("expected cleanup_missing task, got %#v", taskBody)
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

func TestScrapeCandidateScoresFromScannedFile(t *testing.T) {
	root := t.TempDir()
	filePath := filepath.Join(root, "Inception.2010.mkv")
	if err := os.WriteFile(filePath, []byte("movie"), 0o644); err != nil {
		t.Fatal(err)
	}

	server := NewServer(config.Config{Host: "127.0.0.1", Port: "0"})
	library := createJSON(t, server, "/api/libraries", `{"name":"Movies","media_type":"movie","path":"`+root+`"}`)

	scanResponse := httptest.NewRecorder()
	scanRequest := httptest.NewRequest(http.MethodPost, "/api/libraries/"+library["id"].(string)+"/scan", nil)
	server.ServeHTTP(scanResponse, scanRequest)
	if scanResponse.Code != http.StatusAccepted {
		t.Fatalf("expected 202 scan, got %d body=%s", scanResponse.Code, scanResponse.Body.String())
	}

	filesResponse := httptest.NewRecorder()
	filesRequest := httptest.NewRequest(http.MethodGet, "/api/media-files?path="+filePath, nil)
	server.ServeHTTP(filesResponse, filesRequest)
	if filesResponse.Code != http.StatusOK {
		t.Fatalf("expected 200 file, got %d body=%s", filesResponse.Code, filesResponse.Body.String())
	}
	var file map[string]any
	if err := json.NewDecoder(filesResponse.Body).Decode(&file); err != nil {
		t.Fatal(err)
	}

	candidateResponse := httptest.NewRecorder()
	candidateRequest := httptest.NewRequest(
		http.MethodPost,
		"/api/scrape-candidates",
		bytes.NewBufferString(`{
			"media_file_id":"`+file["id"].(string)+`",
			"provider":"tmdb",
			"external_id":"27205",
			"title":"Inception",
			"year":2010
		}`),
	)
	server.ServeHTTP(candidateResponse, candidateRequest)
	if candidateResponse.Code != http.StatusCreated {
		t.Fatalf("expected 201 candidate, got %d body=%s", candidateResponse.Code, candidateResponse.Body.String())
	}

	var candidate map[string]any
	if err := json.NewDecoder(candidateResponse.Body).Decode(&candidate); err != nil {
		t.Fatal(err)
	}
	if candidate["media_id"] != file["media_id"] {
		t.Fatalf("expected candidate media_id from scanned file, got %#v", candidate)
	}
	if candidate["score"].(float64) != 45 {
		t.Fatalf("expected score 45, got %#v", candidate["score"])
	}

	mediaResponse := httptest.NewRecorder()
	mediaRequest := httptest.NewRequest(http.MethodGet, "/api/media/"+file["media_id"].(string), nil)
	server.ServeHTTP(mediaResponse, mediaRequest)
	if mediaResponse.Code != http.StatusOK {
		t.Fatalf("expected 200 media, got %d body=%s", mediaResponse.Code, mediaResponse.Body.String())
	}
	var mediaItem map[string]any
	if err := json.NewDecoder(mediaResponse.Body).Decode(&mediaItem); err != nil {
		t.Fatal(err)
	}
	if mediaItem["match_status"] != "low_confidence" {
		t.Fatalf("expected low_confidence match status, got %#v", mediaItem["match_status"])
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

func TestScrapeDecisionAppliesCandidateMetadata(t *testing.T) {
	server := NewServer(config.Config{Host: "127.0.0.1", Port: "0"})

	mediaID := createJSON(t, server, "/api/media", `{"library_id":"library-1","media_type":"movie","title":"Unknown","display_language":"zh-CN"}`)["id"].(string)
	candidateResponse := httptest.NewRecorder()
	candidateRequest := httptest.NewRequest(
		http.MethodPost,
		"/api/scrape-candidates",
		bytes.NewBufferString(`{
			"media_id":"`+mediaID+`",
			"provider":"tmdb",
			"external_id":"27205",
			"title":"盗梦空间",
			"original_title":"Inception",
			"year":2010,
			"overview":"中文简介",
			"score":96
		}`),
	)
	server.ServeHTTP(candidateResponse, candidateRequest)
	if candidateResponse.Code != http.StatusCreated {
		t.Fatalf("expected 201 candidate, got %d body=%s", candidateResponse.Code, candidateResponse.Body.String())
	}
	var candidate map[string]any
	if err := json.NewDecoder(candidateResponse.Body).Decode(&candidate); err != nil {
		t.Fatal(err)
	}

	decisionResponse := httptest.NewRecorder()
	decisionRequest := httptest.NewRequest(
		http.MethodPost,
		"/api/scrape-decisions",
		bytes.NewBufferString(`{"media_id":"`+mediaID+`","candidate_id":"`+candidate["id"].(string)+`","decision_source":"user","decision":"select","confidence":96,"locked":true}`),
	)
	server.ServeHTTP(decisionResponse, decisionRequest)
	if decisionResponse.Code != http.StatusCreated {
		t.Fatalf("expected 201 decision, got %d body=%s", decisionResponse.Code, decisionResponse.Body.String())
	}
	var decisionBody map[string]any
	if err := json.NewDecoder(decisionResponse.Body).Decode(&decisionBody); err != nil {
		t.Fatal(err)
	}
	applied := decisionBody["applied"].(map[string]any)
	if applied["status"] != "applied" {
		t.Fatalf("expected applied metadata, got %#v", applied)
	}

	getMediaResponse := httptest.NewRecorder()
	getMediaRequest := httptest.NewRequest(http.MethodGet, "/api/media/"+mediaID, nil)
	server.ServeHTTP(getMediaResponse, getMediaRequest)
	if getMediaResponse.Code != http.StatusOK {
		t.Fatalf("expected 200 media, got %d body=%s", getMediaResponse.Code, getMediaResponse.Body.String())
	}
	var mediaItem map[string]any
	if err := json.NewDecoder(getMediaResponse.Body).Decode(&mediaItem); err != nil {
		t.Fatal(err)
	}
	if mediaItem["title"] != "盗梦空间" || mediaItem["match_status"] != "matched" || mediaItem["locked"] != true {
		t.Fatalf("expected applied catalog metadata, got %#v", mediaItem)
	}
	externalIDs, err := server.catalog.ListExternalIDs(context.Background(), "media", mediaID)
	if err != nil {
		t.Fatal(err)
	}
	if len(externalIDs) != 1 || externalIDs[0].Provider != "tmdb" || externalIDs[0].ExternalID != "27205" {
		t.Fatalf("expected selected candidate external id, got %#v", externalIDs)
	}

	assertTranslationList(t, server, "/api/media/"+mediaID+"/translations?language=zh-CN", "盗梦空间")
}

func TestScrapeDecisionDoesNotOverwriteWithEmptyCandidateFields(t *testing.T) {
	server := NewServer(config.Config{Host: "127.0.0.1", Port: "0"})

	mediaID := createJSON(t, server, "/api/media", `{
		"library_id":"library-1",
		"media_type":"movie",
		"title":"Existing Title",
		"original_title":"Existing Original",
		"display_title":"Existing Display",
		"year":1999,
		"overview":"Existing overview",
		"display_language":"zh-CN"
	}`)["id"].(string)
	candidate := createJSON(t, server, "/api/scrape-candidates", `{
		"media_id":"`+mediaID+`",
		"provider":"manual",
		"external_id":"manual-empty",
		"score":80
	}`)

	decisionResponse := httptest.NewRecorder()
	decisionRequest := httptest.NewRequest(
		http.MethodPost,
		"/api/scrape-decisions",
		bytes.NewBufferString(`{"media_id":"`+mediaID+`","candidate_id":"`+candidate["id"].(string)+`","decision":"select"}`),
	)
	server.ServeHTTP(decisionResponse, decisionRequest)
	if decisionResponse.Code != http.StatusCreated {
		t.Fatalf("expected 201 decision, got %d body=%s", decisionResponse.Code, decisionResponse.Body.String())
	}

	getMediaResponse := httptest.NewRecorder()
	getMediaRequest := httptest.NewRequest(http.MethodGet, "/api/media/"+mediaID, nil)
	server.ServeHTTP(getMediaResponse, getMediaRequest)
	if getMediaResponse.Code != http.StatusOK {
		t.Fatalf("expected 200 media, got %d body=%s", getMediaResponse.Code, getMediaResponse.Body.String())
	}
	var mediaItem map[string]any
	if err := json.NewDecoder(getMediaResponse.Body).Decode(&mediaItem); err != nil {
		t.Fatal(err)
	}
	if mediaItem["title"] != "Existing Title" || mediaItem["original_title"] != "Existing Original" || mediaItem["display_title"] != "Existing Display" || mediaItem["overview"] != "Existing overview" {
		t.Fatalf("expected existing metadata to be preserved, got %#v", mediaItem)
	}
	if mediaItem["year"].(float64) != 1999 {
		t.Fatalf("expected existing year, got %#v", mediaItem["year"])
	}
	if mediaItem["match_status"] != "matched" {
		t.Fatalf("expected matched status, got %#v", mediaItem["match_status"])
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
