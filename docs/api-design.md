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
POST /api/download-directories/watch/run
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

`POST /api/download-directories/watch/run` 会返回批次观测字段：`summary`、`total_directories`、`total_discovered`、`total_imported`、`total_failed_files`、`organizer_plan_count`、`started_at`、`completed_at`、`duration_ms`。`summary` 中每个目录包含目录 ID/名称/path、`status`、子任务 ID、发现/导入/失败文件数、批次数、整理计划 ID；目录扫描失败时包含 HTTP 状态码与错误信息。触发时可传 `debounce_seconds`，如果距离上次成功进入扫描流程的完成时间仍在窗口内，会返回 `skipped=true` 和 `skip_reason`。

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
POST /api/media/{id}/ai/suggest-match
POST /api/media/{id}/ai/translate
POST /api/media/{id}/ai/normalize-tags
```

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
