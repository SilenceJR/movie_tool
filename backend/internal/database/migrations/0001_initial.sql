CREATE TABLE IF NOT EXISTS libraries (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  media_type TEXT NOT NULL,
  path TEXT NOT NULL,
  language TEXT NOT NULL DEFAULT 'zh-CN',
  fallback_languages TEXT,
  cache_policy TEXT NOT NULL DEFAULT 'global',
  nfo_enabled INTEGER NOT NULL DEFAULT 1,
  strm_enabled INTEGER NOT NULL DEFAULT 0,
  watch_enabled INTEGER NOT NULL DEFAULT 1,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS media_items (
  id TEXT PRIMARY KEY,
  library_id TEXT NOT NULL,
  media_type TEXT NOT NULL,
  title TEXT,
  original_title TEXT,
  display_title TEXT,
  year INTEGER,
  overview TEXT,
  original_language TEXT,
  display_language TEXT NOT NULL DEFAULT 'zh-CN',
  release_date TEXT,
  runtime INTEGER,
  status TEXT NOT NULL DEFAULT 'pending',
  match_status TEXT NOT NULL DEFAULT 'unmatched',
  locked INTEGER NOT NULL DEFAULT 0,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  FOREIGN KEY (library_id) REFERENCES libraries(id)
);

CREATE TABLE IF NOT EXISTS media_versions (
  id TEXT PRIMARY KEY,
  media_id TEXT NOT NULL,
  name TEXT,
  resolution TEXT,
  source TEXT,
  video_codec TEXT,
  audio_codec TEXT,
  hdr_format TEXT,
  edition TEXT,
  release_group TEXT,
  audio_languages TEXT,
  subtitle_flags TEXT,
  quality_score INTEGER,
  is_default INTEGER NOT NULL DEFAULT 0,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  FOREIGN KEY (media_id) REFERENCES media_items(id)
);

CREATE TABLE IF NOT EXISTS media_files (
  id TEXT PRIMARY KEY,
  media_id TEXT,
  version_id TEXT,
  library_id TEXT NOT NULL,
  path TEXT NOT NULL,
  normalized_path TEXT NOT NULL,
  file_name TEXT NOT NULL,
  extension TEXT,
  size INTEGER,
  modified_at TEXT,
  file_status TEXT NOT NULL DEFAULT 'available',
  is_strm INTEGER NOT NULL DEFAULT 0,
  strm_target TEXT,
  detected_media_type TEXT,
  parsed_title TEXT,
  parsed_year INTEGER,
  parsed_season INTEGER,
  parsed_episode INTEGER,
  parsed_number TEXT,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  FOREIGN KEY (library_id) REFERENCES libraries(id),
  FOREIGN KEY (media_id) REFERENCES media_items(id),
  FOREIGN KEY (version_id) REFERENCES media_versions(id)
);

CREATE TABLE IF NOT EXISTS organizer_rules (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  library_id TEXT,
  media_type TEXT,
  target_root TEXT NOT NULL,
  folder_template TEXT NOT NULL,
  file_template TEXT NOT NULL,
  sidecar_policy TEXT NOT NULL DEFAULT 'include',
  action_mode TEXT NOT NULL DEFAULT 'move',
  conflict_policy TEXT NOT NULL DEFAULT 'skip',
  enabled INTEGER NOT NULL DEFAULT 1,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  FOREIGN KEY (library_id) REFERENCES libraries(id)
);

CREATE TABLE IF NOT EXISTS organizer_plans (
  id TEXT PRIMARY KEY,
  library_id TEXT,
  status TEXT NOT NULL,
  dry_run INTEGER NOT NULL DEFAULT 1,
  summary TEXT,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  FOREIGN KEY (library_id) REFERENCES libraries(id)
);

CREATE TABLE IF NOT EXISTS organizer_actions (
  id TEXT PRIMARY KEY,
  plan_id TEXT NOT NULL,
  media_id TEXT,
  media_file_id TEXT,
  action_type TEXT NOT NULL,
  source_path TEXT NOT NULL,
  target_path TEXT NOT NULL,
  status TEXT NOT NULL,
  conflict_reason TEXT,
  error TEXT,
  executed_at TEXT,
  created_at TEXT NOT NULL,
  FOREIGN KEY (plan_id) REFERENCES organizer_plans(id),
  FOREIGN KEY (media_id) REFERENCES media_items(id),
  FOREIGN KEY (media_file_id) REFERENCES media_files(id)
);

CREATE TABLE IF NOT EXISTS scrape_candidates (
  id TEXT PRIMARY KEY,
  media_file_id TEXT,
  media_id TEXT,
  provider TEXT NOT NULL,
  external_id TEXT,
  title TEXT,
  original_title TEXT,
  year INTEGER,
  poster_url TEXT,
  overview TEXT,
  score INTEGER NOT NULL,
  score_reasons TEXT,
  raw_payload TEXT,
  created_at TEXT NOT NULL,
  FOREIGN KEY (media_file_id) REFERENCES media_files(id),
  FOREIGN KEY (media_id) REFERENCES media_items(id)
);

CREATE TABLE IF NOT EXISTS scrape_decisions (
  id TEXT PRIMARY KEY,
  media_id TEXT NOT NULL,
  candidate_id TEXT,
  decision_source TEXT NOT NULL,
  decision TEXT NOT NULL,
  confidence INTEGER,
  reason TEXT,
  locked INTEGER NOT NULL DEFAULT 0,
  created_at TEXT NOT NULL,
  FOREIGN KEY (media_id) REFERENCES media_items(id),
  FOREIGN KEY (candidate_id) REFERENCES scrape_candidates(id)
);

CREATE TABLE IF NOT EXISTS external_ids (
  id TEXT PRIMARY KEY,
  entity_type TEXT NOT NULL,
  entity_id TEXT NOT NULL,
  provider TEXT NOT NULL,
  external_id TEXT NOT NULL,
  url TEXT,
  created_at TEXT NOT NULL,
  UNIQUE(entity_type, entity_id, provider)
);

CREATE TABLE IF NOT EXISTS localized_metadata (
  id TEXT PRIMARY KEY,
  media_id TEXT NOT NULL,
  language TEXT NOT NULL,
  field_name TEXT NOT NULL,
  value TEXT,
  source TEXT,
  provider TEXT,
  locked INTEGER NOT NULL DEFAULT 0,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  FOREIGN KEY (media_id) REFERENCES media_items(id),
  UNIQUE(media_id, language, field_name)
);

CREATE TABLE IF NOT EXISTS people (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  original_name TEXT,
  localized_name TEXT,
  gender TEXT,
  avatar TEXT,
  bio TEXT,
  birth_date TEXT,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS media_people (
  id TEXT PRIMARY KEY,
  media_id TEXT NOT NULL,
  person_id TEXT NOT NULL,
  role TEXT NOT NULL,
  character_name TEXT,
  sort_order INTEGER,
  FOREIGN KEY (media_id) REFERENCES media_items(id),
  FOREIGN KEY (person_id) REFERENCES people(id)
);

CREATE TABLE IF NOT EXISTS organizations (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  original_name TEXT,
  localized_name TEXT,
  type TEXT NOT NULL,
  logo TEXT,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS media_organizations (
  id TEXT PRIMARY KEY,
  media_id TEXT NOT NULL,
  organization_id TEXT NOT NULL,
  role TEXT NOT NULL,
  FOREIGN KEY (media_id) REFERENCES media_items(id),
  FOREIGN KEY (organization_id) REFERENCES organizations(id)
);

CREATE TABLE IF NOT EXISTS tags (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  normalized_name TEXT NOT NULL UNIQUE,
  category TEXT,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS media_tags (
  id TEXT PRIMARY KEY,
  media_id TEXT NOT NULL,
  tag_id TEXT NOT NULL,
  source TEXT,
  FOREIGN KEY (media_id) REFERENCES media_items(id),
  FOREIGN KEY (tag_id) REFERENCES tags(id)
);

CREATE TABLE IF NOT EXISTS collections (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  localized_name TEXT,
  type TEXT NOT NULL,
  description TEXT,
  poster TEXT,
  source TEXT,
  external_id TEXT,
  locked INTEGER NOT NULL DEFAULT 0,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS collection_items (
  id TEXT PRIMARY KEY,
  collection_id TEXT NOT NULL,
  media_id TEXT NOT NULL,
  sort_order INTEGER,
  relation_type TEXT,
  FOREIGN KEY (collection_id) REFERENCES collections(id),
  FOREIGN KEY (media_id) REFERENCES media_items(id)
);

CREATE TABLE IF NOT EXISTS ai_providers (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  provider_type TEXT NOT NULL,
  base_url TEXT,
  api_key_encrypted TEXT,
  default_model TEXT,
  enabled INTEGER NOT NULL DEFAULT 0,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS ai_decisions (
  id TEXT PRIMARY KEY,
  media_id TEXT,
  task_type TEXT NOT NULL,
  provider_id TEXT,
  model TEXT,
  input_hash TEXT,
  output_json TEXT,
  confidence INTEGER,
  reason TEXT,
  applied INTEGER NOT NULL DEFAULT 0,
  created_at TEXT NOT NULL,
  FOREIGN KEY (media_id) REFERENCES media_items(id),
  FOREIGN KEY (provider_id) REFERENCES ai_providers(id)
);

CREATE TABLE IF NOT EXISTS translation_cache (
  id TEXT PRIMARY KEY,
  source_language TEXT NOT NULL,
  target_language TEXT NOT NULL,
  source_text_hash TEXT NOT NULL,
  source_text TEXT NOT NULL,
  translated_text TEXT,
  provider TEXT,
  model TEXT,
  status TEXT NOT NULL,
  confidence INTEGER,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  UNIQUE(source_language, target_language, source_text_hash)
);

CREATE TABLE IF NOT EXISTS tasks (
  id TEXT PRIMARY KEY,
  task_type TEXT NOT NULL,
  status TEXT NOT NULL,
  progress INTEGER NOT NULL DEFAULT 0,
  message TEXT,
  error TEXT,
  payload TEXT,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS task_logs (
  id TEXT PRIMARY KEY,
  task_id TEXT NOT NULL,
  level TEXT NOT NULL,
  message TEXT NOT NULL,
  created_at TEXT NOT NULL,
  FOREIGN KEY (task_id) REFERENCES tasks(id)
);

CREATE TABLE IF NOT EXISTS automations (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  automation_type TEXT NOT NULL,
  schedule_type TEXT NOT NULL,
  schedule TEXT NOT NULL,
  scope TEXT,
  options TEXT,
  enabled INTEGER NOT NULL DEFAULT 1,
  last_run_at TEXT,
  next_run_at TEXT,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS automation_runs (
  id TEXT PRIMARY KEY,
  automation_id TEXT NOT NULL,
  task_id TEXT,
  status TEXT NOT NULL,
  started_at TEXT,
  finished_at TEXT,
  error TEXT,
  created_at TEXT NOT NULL,
  FOREIGN KEY (automation_id) REFERENCES automations(id),
  FOREIGN KEY (task_id) REFERENCES tasks(id)
);

CREATE TABLE IF NOT EXISTS strm_rules (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  source_prefix TEXT NOT NULL,
  target_prefix TEXT NOT NULL,
  output_path TEXT NOT NULL,
  enabled INTEGER NOT NULL DEFAULT 1,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS server_integrations (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  server_type TEXT NOT NULL,
  base_url TEXT NOT NULL,
  api_key_encrypted TEXT,
  enabled INTEGER NOT NULL DEFAULT 1,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_media_items_library ON media_items(library_id);
CREATE INDEX IF NOT EXISTS idx_media_items_type_year ON media_items(media_type, year);
CREATE UNIQUE INDEX IF NOT EXISTS idx_media_files_normalized_path_unique ON media_files(normalized_path);
CREATE INDEX IF NOT EXISTS idx_media_files_status ON media_files(file_status);
CREATE INDEX IF NOT EXISTS idx_organizer_actions_plan ON organizer_actions(plan_id);
CREATE INDEX IF NOT EXISTS idx_media_people_person ON media_people(person_id);
CREATE INDEX IF NOT EXISTS idx_media_tags_tag ON media_tags(tag_id);
CREATE INDEX IF NOT EXISTS idx_media_organizations_org ON media_organizations(organization_id);
CREATE INDEX IF NOT EXISTS idx_collection_items_collection ON collection_items(collection_id);
CREATE INDEX IF NOT EXISTS idx_external_ids_lookup ON external_ids(provider, external_id);
CREATE INDEX IF NOT EXISTS idx_tasks_status ON tasks(status);
CREATE INDEX IF NOT EXISTS idx_automations_next_run ON automations(enabled, next_run_at);
