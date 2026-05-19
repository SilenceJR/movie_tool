CREATE TABLE IF NOT EXISTS download_directories (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  path TEXT NOT NULL,
  library_id TEXT NOT NULL,
  media_type TEXT,
  action_mode TEXT NOT NULL DEFAULT 'hardlink',
  enabled INTEGER NOT NULL DEFAULT 1,
  watch_enabled INTEGER NOT NULL DEFAULT 1,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  FOREIGN KEY (library_id) REFERENCES libraries(id)
);

CREATE INDEX IF NOT EXISTS idx_download_directories_library ON download_directories(library_id);
CREATE INDEX IF NOT EXISTS idx_download_directories_enabled ON download_directories(enabled, watch_enabled);
