# 数据库设计

## 1. 设计原则

- `media_items` 表示作品。
- `media_versions` 表示同一作品的不同版本。
- `media_files` 表示具体文件或 STRM。
- 元数据原文、译文、人工编辑应分层保存。
- 人物、组织、标签、合集应为通用模型，AV 通过角色和分类扩展。

## 2. 核心枚举

### media_type

```text
movie
tv
anime
av
documentary
other
```

### file_status

```text
available
missing
deleted
ignored
pending
failed
```

### match_status

```text
matched
ambiguous
low_confidence
unmatched
locked
```

## 3. 表结构草案

### libraries

```sql
CREATE TABLE libraries (
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
```

### media_items

```sql
CREATE TABLE media_items (
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
```

### media_versions

```sql
CREATE TABLE media_versions (
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
```

### media_files

```sql
CREATE TABLE media_files (
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
```

### download_directories

```sql
CREATE TABLE download_directories (
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
```

说明：

- `path` 是下载完成目录或中转目录，不等同于媒体库目标目录。
- `library_id` 指向整理后的目标媒体库，用于确定分类和默认媒体类型。
- `action_mode` 表示匹配后默认整理动作，可为 `move`、`copy`、`hardlink`、`symlink`。
- `watch_enabled` 表示该目录后续可被 watcher 监听；当前实现先支持配置与扫描入口。

### organizer_rules

```sql
CREATE TABLE organizer_rules (
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
```

### organizer_plans

```sql
CREATE TABLE organizer_plans (
  id TEXT PRIMARY KEY,
  library_id TEXT,
  status TEXT NOT NULL,
  dry_run INTEGER NOT NULL DEFAULT 1,
  summary TEXT,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  FOREIGN KEY (library_id) REFERENCES libraries(id)
);
```

### organizer_actions

```sql
CREATE TABLE organizer_actions (
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
```

### external_ids

```sql
CREATE TABLE external_ids (
  id TEXT PRIMARY KEY,
  entity_type TEXT NOT NULL,
  entity_id TEXT NOT NULL,
  provider TEXT NOT NULL,
  external_id TEXT NOT NULL,
  url TEXT,
  created_at TEXT NOT NULL,
  UNIQUE(entity_type, entity_id, provider)
);
```

### localized_metadata

```sql
CREATE TABLE localized_metadata (
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
```

### scrape_candidates

```sql
CREATE TABLE scrape_candidates (
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
```

### scrape_decisions

```sql
CREATE TABLE scrape_decisions (
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
```

### people

```sql
CREATE TABLE people (
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
```

### media_people

```sql
CREATE TABLE media_people (
  id TEXT PRIMARY KEY,
  media_id TEXT NOT NULL,
  person_id TEXT NOT NULL,
  role TEXT NOT NULL,
  character_name TEXT,
  sort_order INTEGER,
  FOREIGN KEY (media_id) REFERENCES media_items(id),
  FOREIGN KEY (person_id) REFERENCES people(id)
);
```

### organizations

```sql
CREATE TABLE organizations (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  original_name TEXT,
  localized_name TEXT,
  type TEXT NOT NULL,
  logo TEXT,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);
```

### tags

```sql
CREATE TABLE tags (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  normalized_name TEXT NOT NULL,
  category TEXT,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  UNIQUE(normalized_name)
);
```

### media_tags

```sql
CREATE TABLE media_tags (
  id TEXT PRIMARY KEY,
  media_id TEXT NOT NULL,
  tag_id TEXT NOT NULL,
  source TEXT,
  FOREIGN KEY (media_id) REFERENCES media_items(id),
  FOREIGN KEY (tag_id) REFERENCES tags(id)
);
```

### collections

```sql
CREATE TABLE collections (
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
```

### collection_items

```sql
CREATE TABLE collection_items (
  id TEXT PRIMARY KEY,
  collection_id TEXT NOT NULL,
  media_id TEXT NOT NULL,
  sort_order INTEGER,
  relation_type TEXT,
  FOREIGN KEY (collection_id) REFERENCES collections(id),
  FOREIGN KEY (media_id) REFERENCES media_items(id)
);
```

### ai_providers

```sql
CREATE TABLE ai_providers (
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
```

### ai_decisions

```sql
CREATE TABLE ai_decisions (
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
  FOREIGN KEY (media_id) REFERENCES media_items(id)
);
```

### translation_cache

```sql
CREATE TABLE translation_cache (
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
```

### automations

```sql
CREATE TABLE automations (
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
```

### automation_runs

```sql
CREATE TABLE automation_runs (
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
```

## 4. 索引建议

```sql
CREATE INDEX idx_media_items_library ON media_items(library_id);
CREATE INDEX idx_media_items_type_year ON media_items(media_type, year);
CREATE UNIQUE INDEX idx_media_files_normalized_path_unique ON media_files(normalized_path);
CREATE INDEX idx_media_files_status ON media_files(file_status);
CREATE INDEX idx_organizer_actions_plan ON organizer_actions(plan_id);
CREATE INDEX idx_media_people_person ON media_people(person_id);
CREATE INDEX idx_media_tags_tag ON media_tags(tag_id);
CREATE INDEX idx_collection_items_collection ON collection_items(collection_id);
CREATE INDEX idx_external_ids_lookup ON external_ids(provider, external_id);
CREATE INDEX idx_automations_next_run ON automations(enabled, next_run_at);
```
