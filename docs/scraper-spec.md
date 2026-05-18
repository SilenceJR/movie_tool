# 刮削规格

## 1. 刮削源

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

