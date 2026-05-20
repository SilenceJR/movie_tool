# AI 工作流目标方案

## 1. 目标

Movie Tool 的 AI 目标分两层推进：

1. 第一优先跑通本地 RAG 闭环：OpenAI-compatible 模型服务负责 embedding/chat，Qdrant 存向量和路径，Movie Tool 保存媒体资产、文件路径和元数据。
2. 第二阶段采用 n8n self-hosted AI starter kit 思路，把候选判断、翻译、标签规范化等复杂流程交给 n8n 可视化编排。

平台策略：

- macOS：优先使用 oMLX/MLX，默认 API 为 `http://localhost:8000/v1`，管理后台为 `http://localhost:8000/admin`。
- Windows + NVIDIA：优先使用 Ollama CUDA，默认 API 为 `http://localhost:11434/v1`。
- Qdrant：两端统一使用 `http://localhost:6333`，先单机部署；40T 媒体场景只把路径、字幕、简介、图片向量和元数据写入向量库，不存视频本体。

参考目标：

- `https://github.com/n8n-io/self-hosted-ai-starter-kit`
- 用户提供的 ChatGPT 分享链接：`https://chatgpt.com/share/6a0daac0-d358-83ea-96a7-ba0a9f6804f4`。当前本地自动化无法直接读取该分享内容，后续如有额外方案细节，需要人工补充到本文。

## 2. 目标架构

```text
Movie Tool Web / API
        |
        v
Movie Tool Go Backend + SQLite
        |
        +-- 扫描、整理、刮削候选、任务和自动化
        +-- 对外提供 REST API 与后续 webhook
        |
        +------------------------------+
                                       |
                                       v
                           n8n Workflow Runtime
                                       |
                    +------------------+------------------+
                    |                  |                  |
                    v                  v                  v
          oMLX / Ollama 本地模型   Qdrant 向量库       外部刮削/媒体服务
                    |
                    v
        结构化返回候选判断、翻译、标签、简介、纠错建议
```

## 3. 职责边界

### Movie Tool 保持负责

- 媒体库、下载目录、扫描、入库、整理计划与执行。
- 任务系统、自动化规则、运行历史、错误审计。
- 刮削候选、人工确认、元数据落库。
- AI Provider 配置和基础 OpenAI-compatible/Ollama 接入。
- 内置 Web 控制台展示本项目状态。

### 本地 RAG 层负责

- 文档、代码、日志、字幕、简介等文本切块。
- 调用 oMLX/Ollama 的 OpenAI-compatible embedding API 生成向量。
- 把向量、文件路径、chunk、mtime、size 等 payload 写入 Qdrant。
- 查询时对用户问题生成向量，Qdrant 搜索相似内容，再由 chat model 总结回答。

### n8n AI Stack 后续负责

- AI 工作流可视化编排，例如候选二次判断、翻译、简介生成、标签规范化、番号纠错。
- 长链路人工审核流，例如低置信度候选推送、人工确认后回调 Movie Tool。
- 向量检索增强，例如把历史确认、标题别名、演员/片商词典写入 Qdrant 后供后续匹配使用。
- 多模型路由，例如优先本地 Ollama，必要时转 OpenAI-compatible Provider。

## 4. 当前项目需要调整

1. 部署层增加可选 AI/RAG 工作流栈。
   - 已新增 `deployments/compose/docker-compose.ai.yml`。
   - 默认组合 n8n、PostgreSQL、Qdrant、Ollama。
   - macOS 本地开发推荐不启用 Ollama profile，直接使用宿主机 oMLX。
   - 与现有 `deployments/compose/docker-compose.yml` 通过 Compose 多文件共同启动。

2. 配置层明确 AI 工作流服务地址。
   - 本地 RAG 脚本统一使用 `RAG_OPENAI_BASE_URL`、`RAG_OPENAI_API_KEY`、`RAG_EMBEDDING_MODEL`、`RAG_CHAT_MODEL`、`QDRANT_URL`。
   - n8n 使用 `MOVIE_TOOL_API_URL` 调用 Movie Tool。
   - Movie Tool 后续增加 n8n webhook/base URL 配置，用于触发工作流。
   - oMLX 可配置为 `provider_type=openai_compatible`，默认 `base_url=http://localhost:8000/v1`。
   - Windows/NVIDIA Ollama 可配置为 `provider_type=ollama` 或 OpenAI-compatible，默认 `base_url=http://localhost:11434/v1`。

3. 后端接口需要补齐面向工作流的稳定契约。
   - 先增加本地 RAG 入库/查询脚本，跑通文本到 Qdrant 的闭环。
   - 再增加 webhook 触发记录和回调验签。
   - 为候选判断、翻译、标签规范化提供明确的 job API。
   - 任务日志记录 n8n execution ID，便于跨系统排障。
   - 自动化规则增加 `run_n8n_workflow` 或类似 action，把 Movie Tool 自动化和 n8n workflow 串起来。

4. 数据层需要补齐跨系统引用。
   - AI provider 继续存本地模型/外部模型配置。
   - metadata/translation/decision 记录 n8n workflow ID、execution ID、模型、提示词版本和置信度。
   - 后续可为 Qdrant collection 命名、embedding model 和索引版本增加配置表。

5. 前端/控制台需要展示 AI 工作流状态。
   - 首页增加 AI stack 健康、n8n URL、Ollama Provider、Qdrant 连接状态。
   - AI 设置页提供 n8n webhook/base URL、workflow 映射和测试按钮。
   - 任务中心展示 n8n execution 链接。

## 5. 本地 RAG MVP

### 5.1 服务地址

macOS + oMLX：

```text
RAG_OPENAI_BASE_URL=http://localhost:8000/v1
QDRANT_URL=http://localhost:6333
RAG_EMBEDDING_MODEL=Qwen3-Embedding-4B-4bit-DWQ
RAG_CHAT_MODEL=Qwen3.5-4B-MLX-4bit
```

如果 oMLX 开启 API Key：

```text
RAG_OPENAI_API_KEY=omlx
```

Windows + NVIDIA + Ollama：

```text
RAG_OPENAI_BASE_URL=http://localhost:11434/v1
QDRANT_URL=http://localhost:6333
RAG_EMBEDDING_MODEL=<ollama embedding model>
RAG_CHAT_MODEL=<ollama chat model>
```

### 5.2 先确认模型服务

```bash
curl http://localhost:8000/v1/models \
  -H "Authorization: Bearer ${RAG_OPENAI_API_KEY:-}"
```

oMLX 的真实模型 ID 以 `/v1/models` 返回为准。

### 5.3 入库脚本

```bash
RAG_OPENAI_BASE_URL=http://localhost:8000/v1 \
RAG_OPENAI_API_KEY=omlx \
RAG_EMBEDDING_MODEL=Qwen3-Embedding-4B-4bit-DWQ \
QDRANT_URL=http://localhost:6333 \
scripts/local_rag_ingest.py /path/to/documents
```

默认支持 `.txt/.md/.json/.yaml/.yml/.log/.py/.go/.rs/.dart/.js/.ts`，会自动探测 embedding 维度并创建 Qdrant collection。

### 5.4 查询脚本

```bash
RAG_OPENAI_BASE_URL=http://localhost:8000/v1 \
RAG_OPENAI_API_KEY=omlx \
RAG_EMBEDDING_MODEL=Qwen3-Embedding-4B-4bit-DWQ \
RAG_CHAT_MODEL=Qwen3.5-4B-MLX-4bit \
QDRANT_URL=http://localhost:6333 \
scripts/local_rag_search.py "我记得有个文件讲 AirPods UUID 异步回调"
```

流程：

```text
文件/文本 -> embedding -> Qdrant
用户问题 -> embedding -> Qdrant 搜索 -> chat model 总结回答
```

## 6. n8n 推荐启动方式

复制并编辑环境变量：

```bash
cp deployments/compose/ai.env.example deployments/compose/.env.ai
```

启动 Movie Tool + AI 工作流栈：

```bash
docker compose \
  --env-file deployments/compose/.env.ai \
  -f deployments/compose/docker-compose.yml \
  -f deployments/compose/docker-compose.ai.yml \
  --profile ai-cpu \
  up -d
```

访问：

```text
Movie Tool: http://127.0.0.1:8080/
n8n:        http://127.0.0.1:5678/
Qdrant:     http://127.0.0.1:6333/
Ollama:     http://127.0.0.1:11434/
```

如果 macOS 已在宿主机运行 oMLX，可不启用 `ai-cpu` profile，并把 n8n 的模型服务地址配置为宿主机 OpenAI-compatible API，例如 `http://host.docker.internal:8000/v1`。如果 Windows + NVIDIA 使用 Ollama，则把地址配置为 `http://host.docker.internal:11434/v1` 或 Compose 内的 `http://ollama:11434/v1`。

## 7. Qdrant Collection 建议

先用 Qdrant 单机，不上 Milvus Distributed。40T 媒体本体不会进入向量库，预计向量量级主要来自字幕、简介、文档和图片/关键帧 embedding，几十万到几百万向量时 Qdrant 单机足够。

推荐 collection：

```text
documents   文档、代码、日志
media_text  标题、简介、字幕、标签、演员、片商
media_image 海报、截图、关键帧图片向量
```

SQLite/PostgreSQL 保持保存文件本体索引：

```text
path, filename, extension, size, hash, mtime, media_type, poster_path, subtitle_path
```

## 8. 下一步实施顺序

1. 先用 `scripts/local_rag_ingest.py` 和 `scripts/local_rag_search.py` 跑通本地文档 RAG。
2. 在 Movie Tool 增加 AI/RAG 配置模型：OpenAI-compatible base URL、API key、embedding model、chat model、Qdrant URL、collection。
3. 控制台展示 oMLX/Ollama、Qdrant、collection 健康状态。
4. 增加 media_text 入库任务，把媒体标题、简介、字幕、标签写入 Qdrant。
5. 增加自然语言媒体搜索 API：问题 -> embedding -> Qdrant -> Qwen/Ollama 总结。
6. 后续再增加 n8n workflow adapter，用于候选判断、翻译、标签规范化和人工审核流。
