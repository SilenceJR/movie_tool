package config

import (
	"os"
	"time"
)

type Config struct {
	Host                      string
	Port                      string
	DataDir                   string
	CacheDir                  string
	Database                  string
	PublicURL                 string
	DownloadWatchInterval     time.Duration
	DownloadWatchMinStableAge time.Duration
	RAGOpenAIBaseURL          string
	RAGOpenAIAPIKey           string
	RAGEmbeddingModel         string
	RAGChatModel              string
	RAGQdrantURL              string
	RAGCollection             string
	TMDBBaseURL               string
	TMDBAPIKey                string
	JavDBBaseURL              string
	JavBusBaseURL             string
}

func Load() Config {
	return Config{
		Host:                      env("MOVIE_TOOL_HOST", "0.0.0.0"),
		Port:                      env("MOVIE_TOOL_PORT", "8080"),
		DataDir:                   env("MOVIE_TOOL_DATA_DIR", "./data"),
		CacheDir:                  env("MOVIE_TOOL_CACHE_DIR", "./cache"),
		Database:                  env("MOVIE_TOOL_DATABASE", "./data/movie-tool.db"),
		PublicURL:                 env("MOVIE_TOOL_PUBLIC_URL", ""),
		DownloadWatchInterval:     envDuration("MOVIE_TOOL_DOWNLOAD_WATCH_INTERVAL", 5*time.Minute),
		DownloadWatchMinStableAge: envDuration("MOVIE_TOOL_DOWNLOAD_WATCH_MIN_STABLE_AGE", 2*time.Minute),
		RAGOpenAIBaseURL:          env("RAG_OPENAI_BASE_URL", env("OMLX_URL", "http://localhost:8000/v1")),
		RAGOpenAIAPIKey:           env("RAG_OPENAI_API_KEY", env("OMLX_API_KEY", "")),
		RAGEmbeddingModel:         env("RAG_EMBEDDING_MODEL", "Qwen3-Embedding-4B-4bit-DWQ"),
		RAGChatModel:              env("RAG_CHAT_MODEL", "Qwen3.5-4B-MLX-4bit"),
		RAGQdrantURL:              env("QDRANT_URL", "http://localhost:6333"),
		RAGCollection:             env("RAG_COLLECTION", "local_files"),
		TMDBBaseURL:               env("TMDB_BASE_URL", "https://api.themoviedb.org"),
		TMDBAPIKey:                env("TMDB_API_KEY", ""),
		JavDBBaseURL:              env("JAVDB_BASE_URL", "https://javdb.com"),
		JavBusBaseURL:             env("JAVBUS_BASE_URL", "https://www.javbus.com"),
	}
}

func (c Config) Addr() string {
	return c.Host + ":" + c.Port
}

func env(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}

func envDuration(key string, fallback time.Duration) time.Duration {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := time.ParseDuration(value)
	if err != nil {
		return fallback
	}
	return parsed
}
