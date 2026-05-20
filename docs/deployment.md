# 部署说明

## 1. Docker

推荐目录：

```text
/config      配置与数据库
/cache       图片、翻译、刮削缓存
/media       媒体目录
/strm        STRM 输出目录
```

## 2. 环境变量

```text
MOVIE_TOOL_HOST=0.0.0.0
MOVIE_TOOL_PORT=8080
MOVIE_TOOL_DATA_DIR=/config
MOVIE_TOOL_CACHE_DIR=/cache
MOVIE_TOOL_DATABASE=/config/movie-tool.db
MOVIE_TOOL_DOWNLOAD_WATCH_INTERVAL=5m
MOVIE_TOOL_DOWNLOAD_WATCH_MIN_STABLE_AGE=2m
```

- `MOVIE_TOOL_DOWNLOAD_WATCH_INTERVAL` 控制下载目录后台监听轮询间隔，使用 Go duration 格式，例如 `30s`、`5m`、`1h`。
- `MOVIE_TOOL_DOWNLOAD_WATCH_MIN_STABLE_AGE` 控制文件最近修改时间稳定阈值，未达到阈值的文件会留到后续轮询再扫描，避免下载中的文件被提前入库。

## 3. Docker Compose 示例

见：

```text
deployments/compose/docker-compose.yml
```

## 3.1 可选 AI 工作流栈

项目 AI 能力按两层部署：先跑通本地 RAG，再用 n8n 做复杂流程编排。macOS 推荐 oMLX，Windows + NVIDIA 推荐 Ollama CUDA；Qdrant 作为统一向量库。

本地 RAG 组合：

```text
oMLX 或 Ollama
Qdrant
scripts/local_rag_ingest.py
scripts/local_rag_search.py
```

n8n 工作流组合：

```text
n8n
PostgreSQL for n8n
Qdrant
Ollama 或宿主机 oMLX
```

配置示例：

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

访问地址：

```text
Movie Tool: http://127.0.0.1:8080/
n8n:        http://127.0.0.1:5678/
Qdrant:     http://127.0.0.1:6333/
Ollama:     http://127.0.0.1:11434/
```

如果宿主机已经运行 Ollama，可不启用 `ai-cpu` profile，并在 `.env.ai` 中把 `OLLAMA_BASE_URL` 改成 `http://host.docker.internal:11434`。

macOS + oMLX 推荐：

```text
RAG_OPENAI_BASE_URL=http://host.docker.internal:8000/v1
RAG_EMBEDDING_MODEL=Qwen3-Embedding-4B-4bit-DWQ
RAG_CHAT_MODEL=Qwen3.5-4B-MLX-4bit
```

Windows + NVIDIA + Ollama 推荐：

```text
RAG_OPENAI_BASE_URL=http://host.docker.internal:11434/v1
RAG_EMBEDDING_MODEL=<ollama embedding model>
RAG_CHAT_MODEL=<ollama chat model>
```

媒体目录可通过环境变量指定：

```bash
MOVIE_TOOL_MEDIA_DIR=/path/to/media
```

## 4. 本地自动更新

本地使用 OrbStack 时，Docker CLI 与 Docker Compose 命令保持不变。可以手动刷新容器：

```bash
make docker-update
```

如需在每次本地提交或推送代码后自动重建并更新容器，启用仓库内 Git hooks：

```bash
make install-git-hooks
```

启用后，`.githooks/post-commit` 和 `.githooks/post-push` 会调用：

```text
scripts/docker-update.sh
```

## 5. 权限建议

- 如需写入媒体同目录，容器必须具备媒体目录写权限。
- 如只使用全局缓存与 NFO/STRM 输出目录，可将媒体目录只读挂载。
- 删除清理功能应默认只清理数据库记录，不删除真实媒体文件。
