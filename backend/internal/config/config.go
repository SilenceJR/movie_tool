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
