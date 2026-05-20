# 刮削规格

## 1. 刮削源

当前优先级调整为：先做可验证的数据获取，再把稳定候选写入数据库。AV 资源为主线；普通电影/电视剧使用 TMDB 与豆瓣兜底。

已落地的验证接口：

```http
GET /api/scrapers
GET /api/scrapers/av/parse?number=FC2-PPV-1234567
GET /api/scrapers/av/search?number=FC2-PPV-1234567&source=auto
GET /api/scrapers/av/search?number=SSNI-00123&source=javdb
GET /api/scrapers/av/search?number=SSNI-00123&source=javbus
GET /api/scrapers/av/fetch?external_id=javdb:/v/example&source=javdb
GET /api/scrapers/av/fetch?external_id=javbus:/SSNI-00123&source=javbus
POST /api/scrapers/av/candidates
GET /api/scrapers/tmdb/search?media_type=movie&title=Inception&year=2010&language=zh-CN
GET /api/scrapers/tmdb/fetch?media_type=movie&external_id=27205&language=zh-CN
```

`tmdb` 使用 `TMDB_API_KEY` 鉴权，`TMDB_BASE_URL` 默认 `https://api.themoviedb.org`。搜索/详情接口默认只验证远端可获取和字段映射，不直接写入 `scrape_candidates`，后续由显式“保存候选/选择候选”流程入库。
`av/parse` 已支持番号归一化和源路由验证，返回 `normalized`、`kind`、`prefix`、`digits`、`preferred_providers`，为后续逐平台 live search/fetch 提供稳定入口。
`av/search` 支持 `source=auto`，会按番号推荐源选择当前已实现的 live 源；例如 FC2 会先记录跳过未实现的 `fc2`，再兜底到 JavDB/JavBus 等已接入源做可获取性验证，并在响应的 `source_selection` 中返回实际选择和跳过原因。
`av/search` 与 `av/fetch` 已接入第一版 JavDB HTML 源解析，`JAVDB_BASE_URL` 默认 `https://javdb.com`；该实现用于验证搜索页/详情页可获取与字段映射，可返回标准番号、封面、发行日期、时长、演员、片商、系列、标签等结构化字段，默认不写入数据库。
`source=javbus` 已接入第一版 JavBus HTML 源解析，`JAVBUS_BASE_URL` 默认 `https://www.javbus.com`；当前用于验证搜索页/详情页可获取和字段映射，可返回标准番号、封面、发行日期、时长、片商、系列、标签等字段，默认不写入数据库。
内置控制台的 AV 刮削验证面板已支持在 JavDB/JavBus 间切换数据源；搜索候选后可先拉取详情验证字段映射，再由用户显式保存为 `scrape_candidates`。
`POST /api/scrapers/{provider}/candidates` 用于显式保存已经验证过的 live candidate，必须绑定 `media_id` 或 `media_file_id`，并复用现有候选评分和匹配状态刷新流程。

### 普通媒体

- TMDB
- TVDB
- IMDb
- Douban，可选

### 动漫

- Bangumi
- AniList
- TMDB Anime

### AV

- JavDB
- JavBus
- FC2
- MGStage
- R18
- AVSOX
- Jav321
- 本地番号规则

AV provider 接入顺序：

```text
1. 番号解析与源路由：ABC-123、FC2-PPV-1234567、HEYZO-1234、CARIB-123456-789 已有基础实现。
2. live search/fetch 验证：JavDB/JavBus 已有第一版 search/fetch，先返回候选、封面、发行日期、时长、片商、系列、标签等字段，不写库。
3. 字段归一化：番号、原始标题、中文标题、发行日期、时长、演员、片商、系列、标签、封面。
4. 候选评分：番号精确匹配优先，再参考标题、年份/发行日期、演员、片商。
5. 显式入库：验证通过后通过 `POST /api/scrapers/{provider}/candidates` 写入 scrape_candidates，再进入人工选择或自动决策。
```

## 2. 文件名解析

应解析：

```text
标题
年份
季
集
番号
清晰度
片源
视频编码
音频编码
HDR 类型
字幕标记
发行组
版本名
```

AV 番号示例：

```text
ABC-123
FC2-PPV-1234567
HEYZO-1234
CARIB-123456-789
```

## 3. 候选评分

评分范围：0-100。

推荐权重：

```text
外部 ID 精确匹配：+60
番号精确匹配：+60
标题相似度：0-30
原始标题相似度：0-20
年份匹配：+15
季集匹配：+20
演员/导演匹配：+10
片商/制作公司匹配：+10
媒体类型匹配：+10
严重冲突：-30
```

自动决策：

```text
>= 90 自动匹配
70-89 ambiguous，需要人工确认
< 70 low_confidence，需要人工确认
无候选 unmatched
用户确认 locked
```

## 4. 候选展示

前端应展示：

```text
候选来源
标题
原始标题
年份/发行日期
海报
简介
人物
片商/制作公司
匹配分数
匹配原因
冲突原因
AI 推荐原因，可选
```

## 5. 元数据合并策略

优先级：

```text
用户锁定字段
> 用户手动编辑
> 选定刮削源
> 其他刮削源补充字段
> AI/翻译生成
```

## 6. AV 中文翻译策略

默认显示优先级：

```text
人工中文
> 数据源中文
> AI/机器翻译中文
> 原始日文/英文
```

需要翻译：

```text
标题
简介
标签
系列名
片商别名
```

谨慎翻译：

```text
演员名
导演名
片商正式名称
```

不翻译：

```text
番号
外部 ID
发行日期
时长
文件版本字段
```

## 7. 多版本识别

版本字段：

```text
resolution
source
video_codec
audio_codec
hdr_format
edition
release_group
subtitle_flags
audio_languages
quality_score
```

同一作品新文件进入时：

```text
先匹配 media_item
再匹配 media_version
如果版本属性不同，创建新 media_version
如果路径不同但版本属性相同，作为同版本文件或重复文件处理
```
