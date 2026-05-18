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

## 4. 媒体

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

## 5. 刮削候选

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

## 6. AI

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

## 7. 翻译

```http
POST /api/media/{id}/translate
GET /api/media/{id}/translations
PATCH /api/media/{id}/translations/{translationId}
POST /api/media/{id}/translations/{translationId}/lock
POST /api/media/{id}/translations/{translationId}/unlock
```

## 8. 合集

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

## 9. 人物、组织、标签

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

## 10. STRM

```http
GET /api/strm/rules
POST /api/strm/rules
PATCH /api/strm/rules/{id}
DELETE /api/strm/rules/{id}
POST /api/strm/generate
POST /api/strm/validate
```

## 11. 文件整理

```http
GET /api/organizer/rules
POST /api/organizer/rules
PATCH /api/organizer/rules/{id}
DELETE /api/organizer/rules/{id}
POST /api/organizer/plan
GET /api/organizer/plans/{id}
POST /api/organizer/plans/{id}/execute
POST /api/organizer/plans/{id}/cancel
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

## 12. 外部服务器集成

```http
GET /api/integrations
POST /api/integrations
PATCH /api/integrations/{id}
DELETE /api/integrations/{id}
POST /api/integrations/{id}/test
POST /api/integrations/{id}/refresh
```

## 13. 任务

```http
GET /api/tasks
GET /api/tasks/{id}
POST /api/tasks/{id}/cancel
POST /api/tasks/{id}/retry
GET /api/tasks/{id}/logs
GET /api/events
```

## 14. 自动化

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
