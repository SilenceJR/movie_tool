# 开发路线图

## Phase 0: 项目骨架

- 创建 Go 后端目录。
- 创建 Flutter 前端目录。
- 整理开发文档。
- 确定数据库 schema。
- 确定 API 草案。

## Phase 1: 后端 MVP

- 配置加载。
- SQLite 初始化与迁移。
- 媒体库 CRUD。
- 全量扫描。
- 文件解析。
- 基础任务系统。
- 自动化规则模型。
- 媒体文件入库。
- REST API。

## Phase 2: 刮削 MVP

- Scraper 接口。
- TMDB 电影/电视剧 live search/fetch 验证接口。
- live scraper 结果显式保存为候选。
- AV 番号解析与源路由验证。
- AV 基础刮削源逐个平台验证可获取性。
- 候选评分。
- 人工选择 API。
- 元数据缓存。

## Phase 3: 翻译与 AI

- AI Provider 配置。
- OpenAI-compatible 调用。
- 本地 RAG MVP：入库脚本、查询脚本、Qdrant collection 自动创建。
- macOS oMLX 与 Windows/NVIDIA Ollama 的 OpenAI-compatible 接入约定。
- Qdrant 单机向量库接入约定。
- n8n self-hosted AI 工作流栈部署目标。
- n8n workflow adapter、健康检查和回调验签。
- AV 中文 fallback 翻译。
- 翻译缓存。
- AI 候选辅助判断。

## Phase 4: 多版本、合集、人物标签

- media_versions。
- 版本解析。
- 人物/标签/组织入库。
- 合集识别。
- 高级查询 API。
- 同作品文件夹整理计划。
- 文件整理 dry-run 与执行记录。
- 自动化调度器。

## Phase 5: 输出兼容

- NFO 生成。
- 图片缓存。
- STRM 生成。
- Emby/Jellyfin 刷新。
- Plex 刷新。

## Phase 6: Flutter 前端

- 仪表盘。
- 媒体库配置。
- 媒体浏览。
- 待确认。
- 媒体详情。
- 任务中心。
- AI 设置。
- 文件整理界面。

## Phase 7: 稳定化

- Docker Compose。
- Movie Tool + Qdrant + n8n + Ollama/oMLX 的可选 Compose 部署档。
- 备份与恢复。
- 日志。
- 权限。
- 大库性能优化。
- 插件式刮削源。
