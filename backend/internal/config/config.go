package config

import "os"

type Config struct {
	Host      string
	Port      string
	DataDir   string
	CacheDir  string
	Database  string
	PublicURL string
}

func Load() Config {
	return Config{
		Host:      env("MOVIE_TOOL_HOST", "0.0.0.0"),
		Port:      env("MOVIE_TOOL_PORT", "8080"),
		DataDir:   env("MOVIE_TOOL_DATA_DIR", "./data"),
		CacheDir:  env("MOVIE_TOOL_CACHE_DIR", "./cache"),
		Database:  env("MOVIE_TOOL_DATABASE", "./data/movie-tool.db"),
		PublicURL: env("MOVIE_TOOL_PUBLIC_URL", ""),
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
