# Movie Tool

Movie Tool 是一个面向 NAS / 家庭局域网的媒体资产管理与刮削系统，目标是提供 Go 后端与 Flutter 多端前端，统一管理电影、电视剧、动漫、AV、本地媒体、远程 STRM、元数据缓存、合集、人物、标签与 Emby/Jellyfin/Plex 输出。

## 核心目标

- 扫描并监听媒体目录，支持增量更新、文件丢失标记与清理策略。
- 对电影、电视剧、动漫、AV 进行元数据刮削、缓存、翻译与人工确认。
- 支持刮削正确性检测，低置信度或多候选时进入手动选择。
- 默认中文展示，AV 元数据缺中文时自动翻译并优先展示译文。
- 支持媒体多版本管理，例如 1080p、2160p、Remux、HDR、国配、中字、STRM 版本。
- 支持文件整理，相同媒体自动归入同一个作品文件夹，并保留多版本、字幕、NFO、图片关系。
- 支持自动化管理，可定时扫描、刮削、整理、生成 NFO/STRM、同步媒体服务器与清理丢失文件。
- 支持合集、人物、组织、标签关系查询。
- 兼容 Emby、Jellyfin、Plex、Kodi NFO 与 STRM 文件工作流。
- 支持 AI 辅助识别、候选判断、翻译、标签规范化与简介生成。

## 技术方向

- 后端：Go
- 前端：Flutter Web / Desktop / Mobile
- 数据库：SQLite 起步，PostgreSQL 可选
- 部署：Docker / Docker Compose / 原生二进制

## 当前内容

```text
docs/                 项目开发文档
backend/              Go 后端骨架
frontend/             Flutter 前端规划目录
deployments/          Docker 与部署配置
```

## 快速查看

启动后端后可直接打开内置 Web 控制台：

```text
http://127.0.0.1:8080/
```

控制台随 Go 服务一起启动，会展示当前已实现能力、数据概况、近期任务和下载目录监听状态。

## 文档入口

- [需求说明](docs/requirements.md)
- [系统架构](docs/architecture.md)
- [数据库设计](docs/database-schema.md)
- [API 设计](docs/api-design.md)
- [刮削规格](docs/scraper-spec.md)
- [AI 与翻译设计](docs/ai-translation.md)
- [前端设计](docs/frontend-design.md)
- [开发说明](docs/development.md)
- [开发路线图](docs/roadmap.md)
- [部署说明](docs/deployment.md)
