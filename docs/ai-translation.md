# AI 与翻译设计

## 1. AI Provider

支持：

```text
OpenAI
Gemini
Claude
Ollama
OpenAI-compatible
自定义 HTTP Provider
```

配置项：

```text
provider_type
base_url
api_key
default_model
temperature
max_tokens
enabled
```

## 2. AI 使用场景

- 混乱文件名解析。
- 多候选刮削结果判断。
- AV 番号纠错。
- 标题、简介、标签翻译。
- 标签规范化。
- 合集建议。
- 简介生成或润色。

## 3. 隐私开关

用户应可配置：

```text
是否允许发送文件名
是否允许发送路径
是否允许发送刮削简介
是否允许发送演员/女优信息
是否允许发送标签
```

默认建议：

```text
发送文件名：开启
发送完整路径：关闭
发送简介：开启
发送人物信息：开启
```

## 4. AI 候选判断输出

AI 必须返回结构化 JSON：

```json
{
  "candidate_id": "candidate_1",
  "confidence": 82,
  "reason": "文件名年份、标题关键词和演员信息与候选 1 更接近",
  "risks": ["标题存在别名，仍建议用户确认"]
}
```

## 5. 翻译策略

翻译优先级：

```text
本地人工词典
> 翻译缓存
> AI 翻译
> 传统翻译 API
> 原文 fallback
```

AV 专用词典：

```text
女優 -> 女优
メーカー -> 片商
シリーズ -> 系列
企画 -> 企划
独占配信 -> 独家发布
```

## 6. 字段锁定

以下情况不可自动覆盖：

```text
用户手动编辑
用户锁定翻译
用户锁定匹配结果
```

重新翻译必须由用户明确触发。

