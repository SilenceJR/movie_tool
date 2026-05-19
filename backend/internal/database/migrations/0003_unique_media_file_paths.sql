DROP INDEX IF EXISTS idx_media_files_path;

CREATE UNIQUE INDEX IF NOT EXISTS idx_media_files_normalized_path_unique
ON media_files(normalized_path);
