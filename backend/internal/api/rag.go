package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type ragConfigResponse struct {
	OpenAIBaseURL  string `json:"openai_base_url"`
	HasAPIKey      bool   `json:"has_api_key"`
	EmbeddingModel string `json:"embedding_model"`
	ChatModel      string `json:"chat_model"`
	QdrantURL      string `json:"qdrant_url"`
	Collection     string `json:"collection"`
	PlatformHint   string `json:"platform_hint"`
}

type ragHealthResponse struct {
	Config     ragConfigResponse `json:"config"`
	ModelAPI   ragEndpointHealth `json:"model_api"`
	Qdrant     ragEndpointHealth `json:"qdrant"`
	Collection ragEndpointHealth `json:"collection"`
}

type ragEndpointHealth struct {
	Status     string `json:"status"`
	URL        string `json:"url"`
	Error      string `json:"error,omitempty"`
	StatusCode int    `json:"status_code,omitempty"`
}

func (s *Server) handleRAGConfig(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, s.ragConfigResponse())
}

func (s *Server) handleRAGHealth(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	modelURL := strings.TrimRight(s.cfg.RAGOpenAIBaseURL, "/") + "/models"
	qdrantURL := strings.TrimRight(s.cfg.RAGQdrantURL, "/") + "/collections"
	collectionURL := strings.TrimRight(s.cfg.RAGQdrantURL, "/") + "/collections/" + s.cfg.RAGCollection
	writeJSON(w, http.StatusOK, ragHealthResponse{
		Config:     s.ragConfigResponse(),
		ModelAPI:   probeGET(ctx, modelURL, s.cfg.RAGOpenAIAPIKey),
		Qdrant:     probeGET(ctx, qdrantURL, ""),
		Collection: probeGET(ctx, collectionURL, ""),
	})
}

func (s *Server) ragConfigResponse() ragConfigResponse {
	return ragConfigResponse{
		OpenAIBaseURL:  s.cfg.RAGOpenAIBaseURL,
		HasAPIKey:      s.cfg.RAGOpenAIAPIKey != "",
		EmbeddingModel: s.cfg.RAGEmbeddingModel,
		ChatModel:      s.cfg.RAGChatModel,
		QdrantURL:      s.cfg.RAGQdrantURL,
		Collection:     s.cfg.RAGCollection,
		PlatformHint:   inferRAGPlatform(s.cfg.RAGOpenAIBaseURL),
	}
}

func probeGET(ctx context.Context, url string, apiKey string) ragEndpointHealth {
	if strings.TrimSpace(url) == "" {
		return ragEndpointHealth{Status: "not_configured", URL: url, Error: "url is empty"}
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return ragEndpointHealth{Status: "invalid", URL: url, Error: err.Error()}
	}
	if apiKey != "" {
		request.Header.Set("Authorization", "Bearer "+apiKey)
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return ragEndpointHealth{Status: "unreachable", URL: url, Error: err.Error()}
	}
	defer response.Body.Close()
	status := "ok"
	if response.StatusCode >= 400 {
		status = "error"
		var body map[string]any
		if err := json.NewDecoder(response.Body).Decode(&body); err == nil && len(body) > 0 {
			return ragEndpointHealth{Status: status, URL: url, StatusCode: response.StatusCode, Error: fmt.Sprint(body)}
		}
	}
	return ragEndpointHealth{Status: status, URL: url, StatusCode: response.StatusCode}
}

func inferRAGPlatform(baseURL string) string {
	normalized := strings.ToLower(baseURL)
	switch {
	case strings.Contains(normalized, ":8000"):
		return "macos_omlx"
	case strings.Contains(normalized, ":11434"):
		return "windows_nvidia_ollama"
	default:
		return "openai_compatible"
	}
}
