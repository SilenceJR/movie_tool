# 开发说明

## 1. 后端

运行测试：

```bash
make backend-test
```

启动后端：

```bash
make backend-run
```

健康检查：

```bash
curl http://127.0.0.1:8080/api/health
```

内置 Web 控制台：

```text
http://127.0.0.1:8080/
```

数据库迁移：

- 迁移 SQL 放在 `backend/internal/database/migrations/*.sql`，文件名按字典序执行。
- `database.Runner` 接收最小 `ExecContext`/`QueryContext` 数据库接口，单元测试可使用 fake DB，不依赖真实 SQLite driver。
- Runner 会先确保 `schema_migrations(version TEXT PRIMARY KEY, applied_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP)` 存在，再读取已应用版本，跳过已记录迁移。
- 每个新迁移执行成功后写入 `schema_migrations`；如果某个迁移失败，Runner 会停止并且不会记录该失败版本。

## 2. 目录约定

```text
backend/internal/api          API 路由
backend/internal/automation   自动化规则与调度
backend/internal/config       配置
backend/internal/database     数据库与迁移
backend/internal/scanner      文件扫描与解析
backend/internal/organizer    文件整理计划与执行
backend/internal/scraper      刮削接口
backend/internal/metadata     评分与元数据合并
backend/internal/ai           AI Provider
backend/internal/translation  翻译
backend/internal/task         任务系统
```

## 3. 当前后端状态

- 已有 HTTP server。
- 已有 `/api/health`。
- 已有随服务启动的内置 Web 控制台，根路径 `/` 和 `/ui` 可直接查看当前能力清单、数据概况、近期任务和下载目录监听批次；控制台数据来自 `/api/dashboard`，并已接入媒体库/下载目录快速创建、配置列表展示、AV 番号解析、JavDB/JavBus live 数据源切换搜索、候选详情拉取验证、候选显式保存、已保存候选加载和选择锁定。
- 已有 `/api/config`。
- 已有 SQLite 驱动注册、数据库打开、连接 PRAGMA、embedded migration runner 启动集成。
- 已有 `/api/libraries` CRUD；生产入口使用 SQL store，测试默认使用内存 store。
- 已有 `/api/tasks` 列表、日志、取消、retry 入口；扫描/清理这类同步执行的 API 会记录 running/succeeded/failed 状态和任务日志。
- 已有 `/api/organizer/plan` dry-run 整理计划入口。
- 已有 `/api/libraries/{id}/scan` 扫描入口，会递归发现媒体文件，创建或复用 media item/version，写入 `media_files`，并标记缺失文件。
- 已有 `/api/download-directories` CRUD、`/api/download-directories/{id}/scan` 和 `/api/download-directories/watch/run`；下载目录可绑定目标媒体库和默认整理规则，扫描完成目录文件并作为待整理来源入库，且可用 `min_stable_seconds` 跳过仍在写入的近期文件；扫描时传入 `organizer_rule_id` 可覆盖目录默认规则，并同步生成限定该下载目录来源的整理 dry-run。
- 下载目录监听运行入口会只处理同时 `enabled` 和 `watch_enabled` 的目录，并复用单目录扫描、批量入库、失败隔离与可选整理计划生成逻辑；生产入口已启动后台轮询器，轮询间隔与稳定时间可通过环境变量配置，默认每 5 分钟触发一次，且跳过 2 分钟内仍在变化的文件。
- 每次下载目录监听批次会生成 `download_watch` 父任务，记录扫描目录数量、目录级成功/失败摘要与失败原因；目录级摘要会以 `watch summary: {...}` 结构化 JSON 日志写入父任务，具体目录扫描仍保留各自的 `library_scan` 子任务记录。
- `GET /api/download-directories/watch/runs` 可查询 `download_watch` 历史任务，支持 `status`、`limit` 与 `include_summary=true`；开启摘要时接口会直接返回从结构化任务日志解析出的目录级摘要，前端无需再自行解析日志。
- `POST /api/download-directories/watch/runs/{taskId}/retry-failed` 可基于历史 `download_watch` 任务里的失败目录摘要，只重跑失败下载目录；修复目录路径、权限或临时 IO 问题后，不需要重新扫所有监听目录。
- `POST /api/download-directories/watch/retry-failed` 可读取最近若干个 `download_watch` 批次，按最新状态合并仍未解决的失败目录并去重后统一重跑；默认检查最近 20 个批次，避免同一目录在多次失败日志里被重复重试。
- 下载目录监听响应已包含批次级 `summary`、总目录数、发现/入库/失败文件数、整理计划数、开始/完成时间与耗时；每个目录会给出 succeeded/failed、子任务、导入数量、失败数量和整理计划 ID，便于前端任务中心与自动化观测。
- 下载目录监听批次已增加进程内去重保护；如果上一轮仍在运行，新的手动或后台触发会返回 skipped 状态，避免重复扫描与重复生成整理计划。手动触发可传 `debounce_seconds`，在上次完成时间仍位于去抖窗口内时直接返回 skipped，便于接入文件系统事件后抑制短时间重复扫描；也可传一个或多个 `directory_id` 只重跑指定监听目录，用于目录级失败重试；如果指定目录不存在或未启用监听，会返回 skipped，避免空批次被误判为成功重跑。
- 已有 `/api/automations` CRUD、pause、resume、run、runs 和 run-due；生产入口使用 SQL store，手动 run 或 due tick 会创建 task 与 automation_run。
- 已有 `/api/scrape-candidates` 与 `/api/scrape-decisions`；候选可基于已扫描 `media_file` 的解析字段自动评分，并刷新作品 `match_status`。
- 已新增 live scraper 验证入口 `/api/scrapers`、`/api/scrapers/tmdb/search`、`/api/scrapers/tmdb/fetch`、`/api/scrapers/av/parse`、`/api/scrapers/av/search`、`/api/scrapers/av/fetch` 与 `POST /api/scrapers/{provider}/candidates`；TMDB 通过 `TMDB_API_KEY` 和可选 `TMDB_BASE_URL` 配置，电影/电视剧搜索与详情获取默认只验证远端数据和字段映射，不写入候选表。AV 解析已支持标准番号、FC2、HEYZO、CARIB/1PONDO/10MUSUME 的归一化和推荐源路由，`source=auto` 会跳过未实现推荐源并选择当前已接入的 JavDB/JavBus 做 live 验证；JavDB/JavBus search/fetch 已统一返回标准番号和封面 URL，便于后续评分、海报下载和媒体图片索引；验证通过的 live candidate 可显式保存到 `scrape_candidates`，复用现有评分和匹配状态刷新逻辑。
- 项目 AI 目标已调整为先跑通本地 RAG 闭环，再接 n8n 工作流编排；macOS 使用 oMLX，Windows + NVIDIA 使用 Ollama，Qdrant 统一保存向量和路径。已新增可选 Compose 扩展 `deployments/compose/docker-compose.ai.yml`、RAG 入库脚本 `scripts/local_rag_ingest.py`、查询脚本 `scripts/local_rag_search.py`、`GET /api/rag/config` 与 `GET /api/rag/health`；控制台已展示 RAG 模型服务、模型名、Qdrant 和 collection 配置。详细方案见 `docs/ai-workflow-target.md`。
- 已有媒体文件解析器。
- 已有第一版数据库迁移 SQL。
- 已有 migration runner 与 schema_migrations 记录逻辑。
- 已有内存 task queue。
- 已有自动化 scheduler，可把到期自动化转换为普通 task。
- 已有自动化内存 store，支持运行历史和 next_run_at 计算。
- 已有 SQLite 打开、迁移执行和各核心 SQL store 接线代码。

### automation

- 已有 `Store` 接口和内存版 `MemoryStore`，支持自动化规则 List/Get/Create/Update/Delete。
- `MemoryStore` 默认创建启用的自动化，并基于 `NextRun` 自动维护 `next_run_at`；暂停时清空，恢复或修改计划时重新计算。
- 已支持 `RecordRun`/`ListRuns` 记录自动化运行历史；API server 已接入内存与 SQL store。
- 已有 `POST /api/automations/run-due` 可触发到期规则并记录运行历史；生产入口会启动后台 ticker，定期触发到期自动化。

### organizer

- 已有 dry-run 文件整理计划器：输入媒体基础信息、版本信息、文件列表和 Rule，输出 Plan 与待执行 Actions。
- 默认支持 movie、tv、av 模板；Rule 未指定模板时按媒体类型补齐默认模板。
- planner 生成 dry-run 计划；执行入口会执行 pending 动作并记录 action 状态，支持 move/copy/hardlink/symlink，同一媒体的多版本文件会落入同一媒体目录。
- 执行成功后会把对应 `media_files` 路径回写到目标媒体目录；显式计划里不存在于库内的文件会跳过路径回写但保留执行结果。
- `POST /api/organizer/plan` 可显式传入 media/versions/files，也已支持 `rule_id + media_id` 或 `rule_id + library_id` 自动从 catalog/media_files 组装 dry-run，可按 `match_status`、`file_status`、`media_type`、`source_path_prefix`、`download_directory_id`、`action_status` 过滤批量计划，把下载目录来源文件按规则预览为 hardlink/symlink/move/copy 到目标媒体库目录。
- dry-run 会检测计划内重复目标和磁盘上已有目标；`skip` 会标记 skipped，`rename` 会预演重命名目标，`overwrite_with_confirmation` 会标记 conflict 等待确认。
- 冲突计划可通过 `POST /api/organizer/plans/{id}/skip-conflicts` 批量把 conflict 动作转为 skipped，也可通过 `POST /api/organizer/plans/{id}/rename-conflicts` 批量重命名目标并转回 pending；磁盘目标已存在的 overwrite conflict 可通过 `POST /api/organizer/plans/{id}/confirm-overwrite-conflicts` 批量确认，执行时会先删除既有目标再执行 move/copy/hardlink/symlink。三个冲突处理 API 均支持 `action_id`、`action_type`、`conflict_reason`、`source_path_prefix`、`target_path_prefix` 筛选，便于局部处理批量计划。
- `GET /api/organizer/conflicts/preview` 可使用同一组筛选参数预览 skip/rename/confirm-overwrite 会命中的冲突动作和数量，不修改计划状态，便于前端确认影响范围。
- 失败计划可通过 `POST /api/organizer/plans/{id}/retry` 重试失败动作；如果失败发生在媒体文件路径回写阶段，会只重试数据库路径回写，避免重复移动/复制已完成的文件操作。
- 失败计划也可通过 `GET /api/organizer/failures/preview` 预览人工修复影响范围，再通过 `POST /api/organizer/plans/{id}/skip-failed` 按 `action_id`、`action_type`、`error_contains`、`source_path_prefix`、`target_path_prefix` 将已人工处理或无需继续重试的 failed 动作标记为 skipped，并重新计算计划状态。
- 已成功执行的整理计划可通过 `POST /api/organizer/plans/{id}/rollback` 回滚；move 会把目标移回源路径，copy/hardlink/symlink 会删除目标，并同步恢复关联 `media_files` 路径；如果回滚中途失败，修复文件系统问题后可再次调用 rollback 继续恢复失败动作。
- 仍需补齐更细的批量计划过滤条件。

### scanner

- 已有目录扫描服务，基于 `filepath.WalkDir` 递归发现媒体文件。
- 支持常见视频扩展名，例如 `.mkv`、`.mp4`、`.avi`、`.m2ts`、`.ts`。
- 忽略隐藏文件、隐藏目录和非媒体 sidecar 文件。
- 扫描输出会带上 library 元信息，并复用 `ParseFile` 的标题、年份、季集、番号、版本解析能力。
- 扫描 API 会把解析结果落入 catalog 与 media_files；隐藏文件/目录与字幕、NFO、图片等 sidecar 文件不会进入媒体列表；下载目录扫描可复用文件修改时间稳定性过滤，为 watcher 接入预留安全边界。
- `media_files.normalized_path` 已通过迁移提升为唯一索引，避免同一路径被重复入库。
- 生产 SQLite 路径下，扫描导入会在同一事务内完成 media item、version、media_file 与 missing 标记更新；内存 store 保持原有测试路径。
- 扫描入口支持 `batch_size`，可把超大目录的导入拆成多批事务，并在响应中返回 `batch_count`。
- 扫描入口支持 `continue_on_error=true`，批次失败时会退回单文件导入，并在响应、任务日志和 `media_files` 的 `failed` 状态中记录失败文件与失败原因，避免个别坏文件阻断整批入库。
- 失败媒体文件可通过 `POST /api/media-files/{id}/retry` 单个重试，也可通过 `POST /api/media-files/retry-failed?library_id=...&limit=...` 按媒体库限量批量重试；批量重试支持 `path_prefix`、`media_type`、`failure_contains`、`failed_after`、`failed_before` 过滤，重试成功会清除 `failed` 状态、失败原因和失败时间。

### scraper

- 已有 TMDB live provider，用于普通电影/电视剧兜底验证；`search` 会映射候选标题、原始标题、年份、简介和海报，`fetch` 会返回可应用的基础 metadata。
- 已有 AV 番号解析器，用于 live scraper 前置路由；可归一化标准番号、FC2、HEYZO、CARIB/1PONDO/10MUSUME，并返回推荐 provider 顺序。
- 已有 JavDB live provider，用纯 Go HTTP 与 HTML 解析实现搜索页候选和详情页 metadata 提取；详情会映射发行日期、时长、演员、片商、系列、标签，接口默认只验证数据可获取，不写入数据库。
- 已有 JavBus live provider 第一版，用纯 Go HTTP 与 HTML 解析实现搜索页候选和详情页 metadata 提取；详情会映射发行日期、时长、片商、系列、标签，接口默认只验证数据可获取，不写入数据库。
- 已有 live candidate 显式保存入口，会把用户确认过的 live 候选写入 `scrape_candidates`，并复用自动评分与 `match_status` 刷新。
- 已有轻量候选评分规则：番号精确匹配、标题相似度、年份匹配/冲突，并输出 0-100 分与原因。
- 创建候选时，如果提供 `media_file_id` 且未提供完整分数，会读取已入库文件的 parsed title/year/number 自动评分。
- 候选创建后会基于当前候选列表刷新未锁定作品的 `match_status`，人工选择候选后会应用标题、年份、简介、本地化元数据和 `external_ids`；空候选字段不会覆盖作品已有元数据。

## 4. 下一步建议

```text
1. 继续扩展 AV live scraper：接入 FC2/MGStage/R18/Jav321 逐源 search/fetch 验证，并对各源字段做归一化对齐。
2. 继续完善控制台待确认体验：增加候选对比视图、候选详情预览和批量处理。
3. 将 RAG 配置从环境变量提升为可持久化、可在控制台编辑的配置模型。
4. 将 RAG 入库/查询能力接入后端任务系统，支持 media_text collection。
5. 增加自然语言媒体搜索 API：问题 -> embedding -> Qdrant -> Qwen/Ollama 总结。
6. 后续增加 run_n8n_workflow 自动化类型，并记录 n8n execution ID。
```
