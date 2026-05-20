# API 设计

## 1. API 风格

- REST API 为主。
- WebSocket 或 SSE 用于任务进度和扫描事件。
- 请求与响应使用 JSON。
- 错误响应统一格式。

## 2. 统一错误格式

```json
{
  "error": {
    "code": "LOW_CONFIDENCE_MATCH",
    "message": "刮削结果置信度过低，需要人工确认",
    "details": {}
  }
}
```

## 2.1 控制台

```http
GET /api/health
GET /api/config
GET /api/dashboard
GET /api/rag/config
GET /api/rag/health
```

`GET /api/dashboard` 为内置 Web 控制台提供汇总数据，包含媒体库、媒体条目、媒体文件、下载目录、自动化、任务风险计数、已实现能力清单、近期任务和最近下载目录监听批次摘要。
`GET /api/rag/config` 返回本地 RAG 配置，隐藏 API key，仅展示是否已配置；`GET /api/rag/health` 会探测 OpenAI-compatible 模型服务 `/models`、Qdrant `/collections` 与当前 collection。

## 3. 媒体库

```http
GET /api/libraries
POST /api/libraries
GET /api/libraries/{id}
PATCH /api/libraries/{id}
DELETE /api/libraries/{id}
POST /api/libraries/{id}/scan
POST /api/libraries/{id}/watch/start
POST /api/libraries/{id}/watch/stop
```

## 4. 下载目录

下载目录用于接入 BT/PT/网盘等完成目录。目录中的文件会作为待整理来源扫描入库，匹配到媒体后可通过 organizer 规则 dry-run 出硬链、软链、移动或复制到目标媒体库目录的动作。

```http
GET /api/download-directories
POST /api/download-directories
GET /api/download-directories/{id}
PATCH /api/download-directories/{id}
DELETE /api/download-directories/{id}
POST /api/download-directories/{id}/scan
GET /api/download-directories/watch/runs?status={status}&limit={n}&include_summary={bool}
POST /api/download-directories/watch/runs/{taskId}/retry-failed
POST /api/download-directories/watch/retry-failed?limit={n}
POST /api/download-directories/watch/run?directory_id={downloadDirectoryId}
POST /api/media-files/{id}/retry
POST /api/media-files/retry-failed?library_id={libraryId}&limit={n}&path_prefix={path}&media_type={type}&failure_contains={text}&failed_after={rfc3339}&failed_before={rfc3339}
POST /api/organizer/plans/{id}/rollback
```

创建下载目录示例：

```json
{
  "name": "PT 完成目录",
  "path": "/downloads/complete",
  "library_id": "library_movies",
  "media_type": "movie",
  "action_mode": "hardlink",
  "organizer_rule_id": "organizer_rule_movies",
  "enabled": true,
  "watch_enabled": true
}
```

`POST /api/download-directories/watch/run` 会返回批次观测字段：`summary`、`total_directories`、`total_discovered`、`total_imported`、`total_failed_files`、`organizer_plan_count`、`started_at`、`completed_at`、`duration_ms`。`summary` 中每个目录包含目录 ID/名称/path、`status`、子任务 ID、发现/导入/失败文件数、批次数、整理计划 ID；目录扫描失败时包含 HTTP 状态码与错误信息。触发时可传 `debounce_seconds`，如果距离上次成功进入扫描流程的完成时间仍在窗口内，会返回 `skipped=true` 和 `skip_reason`；也可传一个或多个 `directory_id` 只重跑指定监听目录。如果指定目录不存在或未启用监听，会返回 skipped，避免空批次被误判为成功重跑。
`GET /api/download-directories/watch/runs` 返回最近的 `download_watch` 任务历史，可按 `status` 过滤并用 `limit` 限制数量；传 `include_summary=true` 时会直接返回从结构化任务日志解析出的目录级摘要。
`POST /api/download-directories/watch/runs/{taskId}/retry-failed` 会读取指定 `download_watch` 任务中的结构化目录摘要，只重跑其中失败的下载目录，并返回来源任务、重试目录 ID 列表和新的监听批次结果；修复目录路径或权限后可直接用该入口做目录级失败重试。
`POST /api/download-directories/watch/retry-failed` 会按最新到最旧读取最近 `limit` 个 `download_watch` 任务，合并仍未被后续成功批次覆盖的失败目录并去重后统一重跑；未传 `limit` 时默认检查最近 20 个批次，适合前端提供“一键重试近期失败目录”。

## 5. 媒体

```http
GET /api/media
GET /api/media/{id}
PATCH /api/media/{id}
POST /api/media/{id}/rescrape
POST /api/media/{id}/lock
POST /api/media/{id}/unlock
GET /api/media/{id}/versions
POST /api/media/{id}/versions/{versionId}/default
```

查询参数：

```text
type
title
year
person_id
tag_id
collection_id
organization_id
file_status
resolution
source
has_subtitle
match_status
page
page_size
sort
```

## 6. 刮削候选

先提供“验证源可获取数据”的 live scraper API，再把验证通过的候选显式写入 `scrape_candidates`。这样 AV 源、TMDB、豆瓣等 provider 接入时，可以先确认远端可搜索、可详情获取、字段映射稳定，再进入数据库和自动决策。

```http
GET /api/scrapers
GET /api/scrapers/av/parse?number={number}
GET /api/scrapers/av/search?number={number}&source=javdb
GET /api/scrapers/av/search?number={number}&source=javbus
GET /api/scrapers/av/fetch?external_id={providerScopedId}&source=javdb
GET /api/scrapers/av/fetch?external_id={providerScopedId}&source=javbus
POST /api/scrapers/{provider}/candidates
GET /api/scrapers/{provider}/search?media_type={movie|tv|av}&title={title}&year={year}&number={number}&language={language}
GET /api/scrapers/{provider}/fetch?media_type={movie|tv|av}&external_id={externalId}&language={language}
```

当前已实现：

- `tmdb`：电影/电视剧兜底源，配置 `TMDB_API_KEY` 与可选 `TMDB_BASE_URL` 后可用；`search` 与 `fetch` 默认只返回验证结果，不写入候选表。
- `av/parse`：AV 番号解析和源路由验证，支持标准番号、FC2、HEYZO、CARIB/1PONDO/10MUSUME 等基础格式；只返回归一化番号和推荐抓取源顺序，不写入候选表。
- `av/search` 与 `av/fetch`：当前默认 `source=javdb`，配置 `JAVDB_BASE_URL` 后可验证 JavDB 搜索页和详情页字段映射；详情会返回发行日期、时长、演员、片商、系列、标签等结构化字段，默认不写入候选表。
- `source=javbus`：配置 `JAVBUS_BASE_URL` 后可验证 JavBus 搜索页和详情页字段映射；当前返回标题、年份、封面、发行日期、时长、片商、系列、标签等字段，默认不写入候选表。
- `POST /api/scrapers/{provider}/candidates`：显式把 live search/fetch 中确认可用的候选保存到 `scrape_candidates`，请求必须带 `media_id` 或 `media_file_id`，会复用现有候选评分和 `match_status` 刷新逻辑。

保存 live candidate 示例：

```json
{
  "media_id": "media-1",
  "source": "javdb",
  "candidate": {
    "provider": "javdb",
    "external_id": "javdb:/v/example",
    "title": "SSNI-00123 Example Title",
    "original_title": "SSNI-00123 Example Title",
    "year": 2020,
    "poster_url": "https://javdb.com/covers/ssni.jpg",
    "overview": "Example overview",
    "score": 90,
    "score_reasons": ["番号精确匹配"]
  }
}
```

规划中：

- `av/search` 与 `av/fetch` 多源扩展：按 FC2/MGStage/R18/Jav321 等源逐个验证可用性。
- `douban`：中文电影/电视剧补充兜底。

```http
GET /api/media/{id}/scrape-candidates
POST /api/media/{id}/scrape-candidates/search
POST /api/media/{id}/scrape-candidates/{candidateId}/select
POST /api/media/{id}/scrape-candidates/ignore
POST /api/media/{id}/scrape-candidates/manual
```

选择候选请求：

```json
{
  "lock": true,
  "apply_images": true,
  "apply_people": true,
  "apply_collections": true
}
```

## 7. AI

```http
GET /api/ai/providers
POST /api/ai/providers
PATCH /api/ai/providers/{id}
DELETE /api/ai/providers/{id}
POST /api/ai/providers/{id}/test
GET /api/ai/workflows
POST /api/ai/workflows
POST /api/ai/workflows/{id}/test
POST /api/ai/workflows/{id}/run
POST /api/ai/workflows/{id}/callback
POST /api/media/{id}/ai/suggest-match
POST /api/media/{id}/ai/translate
POST /api/media/{id}/ai/normalize-tags
```

AI 工作流目标采用 n8n 编排。后续 `/api/ai/workflows` 用于保存 n8n workflow 映射、base URL、webhook secret、默认输入模板和健康检查结果；`run` 创建 Movie Tool task 并记录 n8n execution ID，`callback` 负责验签后回写候选判断、翻译、标签规范化等结构化结果。

## 8. 翻译

```http
POST /api/media/{id}/translate
GET /api/media/{id}/translations
PATCH /api/media/{id}/translations/{translationId}
POST /api/media/{id}/translations/{translationId}/lock
POST /api/media/{id}/translations/{translationId}/unlock
```

## 9. 合集

```http
GET /api/collections
POST /api/collections
GET /api/collections/{id}
PATCH /api/collections/{id}
DELETE /api/collections/{id}
POST /api/collections/{id}/items
DELETE /api/collections/{id}/items/{mediaId}
POST /api/collections/rules/run
```

## 10. 人物、组织、标签

```http
GET /api/people
GET /api/people/{id}
GET /api/people/{id}/media

GET /api/organizations
GET /api/organizations/{id}
GET /api/organizations/{id}/media

GET /api/tags
PATCH /api/tags/{id}
POST /api/tags/merge
GET /api/tags/{id}/media
```

## 11. STRM

```http
GET /api/strm/rules
POST /api/strm/rules
PATCH /api/strm/rules/{id}
DELETE /api/strm/rules/{id}
POST /api/strm/generate
POST /api/strm/validate
```

## 12. 文件整理

```http
GET /api/organizer/rules
POST /api/organizer/rules
PATCH /api/organizer/rules/{id}
DELETE /api/organizer/rules/{id}
POST /api/organizer/plan
GET /api/organizer/plans/{id}
POST /api/organizer/plans/{id}/execute
POST /api/organizer/plans/{id}/cancel
GET /api/organizer/failures/preview?plan_id={planId}&action_id={actionId}&action_type={mode}&error_contains={text}&source_path_prefix={path}&target_path_prefix={path}
POST /api/organizer/plans/{id}/skip-failed?action_id={actionId}&action_type={mode}&error_contains={text}&source_path_prefix={path}&target_path_prefix={path}
GET /api/organizer/actions
```

生成整理计划请求：

```json
{
  "library_id": "library_1",
  "media_ids": ["media_1"],
  "mode": "move",
  "dry_run": true,
  "include_sidecars": true
}
```

整理动作类型：

```text
move
copy
hardlink
symlink
```

冲突策略：

```text
skip
rename
overwrite_with_confirmation
```

冲突处理：

```http
GET /api/organizer/conflicts/preview?plan_id={planId}&operation={skip|rename|confirm-overwrite}&action_id={actionId}&action_type={mode}&conflict_reason={reason}&source_path_prefix={path}&target_path_prefix={path}
POST /api/organizer/plans/{id}/skip-conflicts?action_id={actionId}&action_type={mode}&conflict_reason={reason}&source_path_prefix={path}&target_path_prefix={path}
POST /api/organizer/plans/{id}/rename-conflicts?action_id={actionId}&action_type={mode}&conflict_reason={reason}&source_path_prefix={path}&target_path_prefix={path}
POST /api/organizer/plans/{id}/confirm-overwrite-conflicts?action_id={actionId}&action_type={mode}&conflict_reason={reason}&source_path_prefix={path}&target_path_prefix={path}
```

## 13. 外部服务器集成

```http
GET /api/integrations
POST /api/integrations
PATCH /api/integrations/{id}
DELETE /api/integrations/{id}
POST /api/integrations/{id}/test
POST /api/integrations/{id}/refresh
```

## 14. 任务

```http
GET /api/tasks
GET /api/tasks/{id}
POST /api/tasks/{id}/cancel
POST /api/tasks/{id}/retry
GET /api/tasks/{id}/logs
GET /api/events
```

## 15. 自动化

```http
GET /api/automations
POST /api/automations
GET /api/automations/{id}
PATCH /api/automations/{id}
DELETE /api/automations/{id}
POST /api/automations/{id}/pause
POST /api/automations/{id}/resume
POST /api/automations/{id}/run
GET /api/automations/{id}/runs
```

创建自动化示例：

```json
{
  "name": "每日凌晨扫描电影库",
  "automation_type": "scan_library",
  "schedule_type": "cron",
  "schedule": "0 3 * * *",
  "scope": {
    "library_id": "library_movies"
  },
  "options": {
    "dry_run": false,
    "retry_limit": 3
  },
  "enabled": true
}
```
