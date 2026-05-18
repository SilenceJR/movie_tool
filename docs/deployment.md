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
```

## 3. Docker Compose 示例

见：

```text
deployments/compose/docker-compose.yml
```

## 4. 权限建议

- 如需写入媒体同目录，容器必须具备媒体目录写权限。
- 如只使用全局缓存与 NFO/STRM 输出目录，可将媒体目录只读挂载。
- 删除清理功能应默认只清理数据库记录，不删除真实媒体文件。

