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
- 已有 `/api/config`。
- 已有 `/api/libraries` CRUD；生产入口使用 SQL store，测试默认使用内存 store。
- 已有 `/api/tasks` 列表与任务 retry 入口。
- 已有 `/api/organizer/plan` dry-run 整理计划入口。
- 已有 `/api/libraries/{id}/scan` 扫描入口，当前返回解析结果并创建扫描任务记录，尚未写入数据库。
- 已有内存版 `/api/automations` CRUD、pause、resume、run、runs。
- 已有媒体文件解析器。
- 已有第一版数据库迁移 SQL。
- 已有 migration runner 与 schema_migrations 记录逻辑。
- 已有内存 task queue。
- 已有自动化 scheduler，可把到期自动化转换为普通 task。
- 已有自动化内存 store，支持运行历史和 next_run_at 计算。
- 已有 SQLite 打开、迁移执行和 libraries SQL store 接线代码。
- 当前环境无法下载 SQLite driver；补齐 `modernc.org/sqlite` 依赖及空白导入后，生产入口即可使用真实 SQLite。

### automation

- 已有 `Store` 接口和内存版 `MemoryStore`，支持自动化规则 List/Get/Create/Update/Delete。
- `MemoryStore` 默认创建启用的自动化，并基于 `NextRun` 自动维护 `next_run_at`；暂停时清空，恢复或修改计划时重新计算。
- 已支持 `RecordRun`/`ListRuns` 记录自动化运行历史；内存实现用于开发和单元测试，尚未接入 API server。

### organizer

- 已有 dry-run 文件整理计划器：输入媒体基础信息、版本信息、文件列表和 Rule，输出 Plan 与待执行 Actions。
- 默认支持 movie、tv、av 模板；Rule 未指定模板时按媒体类型补齐默认模板。
- planner 只生成计划，不执行真实移动、复制或链接操作；同一媒体的多版本文件会落入同一媒体目录。

### scanner

- 已有目录扫描服务，基于 `filepath.WalkDir` 递归发现媒体文件。
- 支持常见视频扩展名，例如 `.mkv`、`.mp4`、`.avi`、`.m2ts`、`.ts`。
- 忽略隐藏文件、隐藏目录和非媒体 sidecar 文件。
- 扫描输出会带上 library 元信息，并复用 `ParseFile` 的标题、年份、季集、番号、版本解析能力。

### scanner

- 已有文件名解析器 `ParseFile`，可识别电影年份、剧集季集、AV 番号、分辨率、来源、编码、HDR、字幕标记和发布组。
- 已有目录扫描器 `Scanner.Walk` / `Walk`：输入扫描根目录和库信息，递归发现常见视频扩展名并返回 `ParsedFile` 列表。
- 扫描阶段只做文件发现与解析，不连接数据库，不写入 `media_files`；隐藏文件/目录与字幕、NFO、图片等 sidecar 文件不会进入媒体列表。

## 4. 下一步建议

```text
1. 接入 SQLite。
2. 将应用启动流程接入 migration runner。
3. 实现 libraries CRUD。
4. 实现 scanner 将文件写入 media_files。
5. 实现 task queue。
```
