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
