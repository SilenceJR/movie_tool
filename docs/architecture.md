# 系统架构

## 1. 总体架构

```text
Flutter Web/Desktop/Mobile
        |
        | REST API / WebSocket
        v
Go Backend
        |
        +-- API Layer
        +-- Task Queue
        +-- Library Scanner
        +-- File Watcher
        +-- Metadata Resolver
        +-- File Organizer
        +-- Automation Scheduler
        +-- Scraper Providers
        +-- AI / Translation
        +-- n8n Workflow Adapter
        +-- NFO / STRM Generator
        +-- Emby/Jellyfin/Plex Adapter
        |
        v
SQLite / PostgreSQL
        |
        v
Cache Directory / Media Directory

本地 AI/RAG 目标：

```text
Movie Tool Backend
        |
        | OpenAI-compatible API
        v
oMLX(macOS) / Ollama(Windows NVIDIA)
        |
        +-- Embedding: 文本 -> 向量
        +-- Chat: 搜索结果 -> 总结/问答
        |
        v
Qdrant 单机向量库
        |
        v
路径、chunk、媒体元数据 payload
```

后续可选 AI 工作流栈：

```text
Movie Tool Backend
        |
        | REST / webhook
        v
n8n + PostgreSQL
        |
        +-- 调用 oMLX / Ollama
        +-- 调用 Qdrant
        +-- 编排候选判断、翻译、标签规范化
```

详细边界与实施顺序见 [AI 工作流目标方案](ai-workflow-target.md)。
```

## 2. 后端模块

### 2.1 api

提供 REST API 与 WebSocket/SSE。

职责：

- 媒体库配置
- 媒体查询
- 刮削候选选择
- 任务管理
- AI 配置
- 服务器集成配置

### 2.2 library

管理媒体库配置、路径、语言、刮削策略、缓存策略。

### 2.3 scanner

负责全量扫描与目录遍历。

输出：

- 文件路径
- 文件大小
- 修改时间
- 媒体类型初判
- 文件名解析结果

### 2.4 watcher

负责增量监听。

事件：

- created
- modified
- deleted
- moved

### 2.5 organizer

负责文件整理计划与执行。

职责：

- 根据 `media_item`、`media_version`、`media_file` 生成目标目录。
- 保证相同媒体进入同一个作品文件夹。
- 处理多版本文件命名。
- 识别并连带整理字幕、NFO、图片等 sidecar 文件。
- 在执行前生成 dry-run 计划。
- 执行移动、复制、硬链接、软链接。
- 记录每个文件动作，便于审计与回滚。

整理原则：

```text
先刮削/确认作品
-> 生成整理计划
-> 用户确认
-> 执行文件动作
-> 更新 media_files 路径
-> 触发 NFO/STRM/媒体服务器刷新
```

### 2.6 scraper

统一刮削接口。

```go
type Scraper interface {
    Name() string
    Supports(mediaType string) bool
    Search(ctx context.Context, query SearchQuery) ([]Candidate, error)
    Fetch(ctx context.Context, candidate Candidate) (*Metadata, error)
}
```

### 2.7 metadata

负责候选评分、元数据合并、语言 fallback、锁定字段处理。

### 2.8 ai

负责 AI Provider 配置、模型调用、结构化输出校验。

AI 模块继续承担 Provider 与模型调用抽象。当前优先接入 oMLX/Ollama + Qdrant 本地 RAG；复杂多步骤 AI 流程后续交给 n8n 编排，并通过 workflow adapter 回写候选判断、翻译、标签和简介结果。

### 2.9 translation

负责翻译任务、翻译缓存、本地词典、字段锁定。

### 2.10 collection

负责合集识别、合集规则、手动合集、智能合集。

### 2.11 search

负责统一查询。

查询维度：

- 标题
- 人物
- 标签
- 组织
- 合集
- 版本属性
- 文件状态

### 2.12 nfo

负责 NFO 输出。

目标：

- Kodi NFO
- Emby/Jellyfin NFO
- Plex 可读结构

### 2.13 strm

负责 STRM 生成、路径映射、失效检测。

### 2.14 integration

负责外部媒体服务器集成。

适配器：

- Emby
- Jellyfin
- Plex

### 2.15 task

负责异步任务。

任务类型：

- library_scan
- scrape_media
- download_images
- translate_metadata
- organize_files
- generate_nfo
- generate_strm
- refresh_server
- cleanup_missing

### 2.16 automation

负责自动化规则和调度。

职责：

- 保存自动化规则。
- 计算下次运行时间。
- 到期后创建对应 task。
- 记录运行历史。
- 支持暂停、恢复、手动立即运行。

自动化类型：

```text
scan_library
scrape_pending
translate_missing
organize_files
generate_nfo
generate_strm
refresh_server
cleanup_missing
```

## 3. 数据流

### 3.1 新文件处理

```text
文件新增
-> watcher 收到事件
-> scanner 解析文件
-> version parser 识别版本
-> media matcher 查找已有作品
-> scraper 搜索候选
-> scoring 评分
-> 高置信度自动匹配 / 低置信度进入人工确认
-> metadata fetch
-> translation fallback
-> 写入数据库
-> 可选生成文件整理计划
-> 用户确认后整理文件
-> 下载图片
-> 生成 NFO / STRM
-> 刷新媒体服务器
```

### 3.2 文件删除处理

```text
文件消失
-> 标记 media_file 为 missing
-> 如果作品无可用文件，media_item 标记为 unavailable
-> 保留元数据与用户编辑
-> 达到清理策略后才删除或归档
```

### 3.3 AV 中文 fallback

```text
AV 刮削
-> 获取原始日文/英文元数据
-> 检查 zh-CN 字段
-> 缺失则先过本地词典
-> 调用翻译 Provider
-> 保存 original 与 zh-CN
-> display 字段优先使用 zh-CN
```

## 4. 部署模式

### 4.1 单机 Docker

推荐 NAS 使用。

```text
movie-tool-server
sqlite database
/config
/cache
/media mounted readonly or readwrite
```

### 4.2 原生二进制

适合 Windows/macOS/Linux 桌面。

### 4.3 PostgreSQL 模式

后续用于大媒体库或多实例。
