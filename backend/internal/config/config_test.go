package config

import (
	"testing"
	"time"
)

func TestLoadDownloadWatchDurations(t *testing.T) {
	t.Setenv("MOVIE_TOOL_DOWNLOAD_WATCH_INTERVAL", "30s")
	t.Setenv("MOVIE_TOOL_DOWNLOAD_WATCH_MIN_STABLE_AGE", "90s")

	cfg := Load()

	if cfg.DownloadWatchInterval != 30*time.Second {
		t.Fatalf("expected download watch interval 30s, got %s", cfg.DownloadWatchInterval)
	}
	if cfg.DownloadWatchMinStableAge != 90*time.Second {
		t.Fatalf("expected download watch min stable age 90s, got %s", cfg.DownloadWatchMinStableAge)
	}
}

func TestLoadDownloadWatchDurationsFallbackOnInvalidValues(t *testing.T) {
	t.Setenv("MOVIE_TOOL_DOWNLOAD_WATCH_INTERVAL", "soon")
	t.Setenv("MOVIE_TOOL_DOWNLOAD_WATCH_MIN_STABLE_AGE", "later")

	cfg := Load()

	if cfg.DownloadWatchInterval != 5*time.Minute {
		t.Fatalf("expected default download watch interval, got %s", cfg.DownloadWatchInterval)
	}
	if cfg.DownloadWatchMinStableAge != 2*time.Minute {
		t.Fatalf("expected default download watch min stable age, got %s", cfg.DownloadWatchMinStableAge)
	}
}
