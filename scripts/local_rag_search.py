#!/usr/bin/env python3
"""Search Qdrant with an OpenAI-compatible embedding API and summarize with chat completions."""

from __future__ import annotations

import argparse
import json
import os
import sys
import urllib.error
import urllib.request


def request_json(method: str, url: str, payload: dict | None = None, headers: dict | None = None, timeout: int = 60) -> dict:
    data = None if payload is None else json.dumps(payload).encode("utf-8")
    merged_headers = {"Content-Type": "application/json"}
    if headers:
        merged_headers.update(headers)
    request = urllib.request.Request(url, data=data, method=method, headers=merged_headers)
    try:
        with urllib.request.urlopen(request, timeout=timeout) as response:
            body = response.read().decode("utf-8")
    except urllib.error.HTTPError as error:
        body = error.read().decode("utf-8", errors="ignore")
        raise RuntimeError(f"{method} {url} failed: {error.code} {body}") from error
    if not body:
        return {}
    return json.loads(body)


def auth_headers(api_key: str) -> dict:
    if not api_key:
        return {}
    return {"Authorization": f"Bearer {api_key}"}


def api_base_url(args: argparse.Namespace) -> str:
    return args.openai_base_url.rstrip("/")


def embed(text: str, args: argparse.Namespace) -> list[float]:
    response = request_json(
        "POST",
        f"{api_base_url(args)}/embeddings",
        {
            "model": args.embedding_model,
            "input": text,
        },
        headers=auth_headers(args.omlx_api_key),
        timeout=args.timeout,
    )
    return response["data"][0]["embedding"]


def search(query: str, args: argparse.Namespace) -> list[dict]:
    vector = embed(query, args)
    response = request_json(
        "POST",
        f"{args.qdrant_url.rstrip('/')}/collections/{args.collection}/points/search",
        {
            "vector": vector,
            "limit": args.limit,
            "with_payload": True,
        },
        timeout=args.timeout,
    )
    return response.get("result", [])


def summarize(query: str, results: list[dict], args: argparse.Namespace) -> str:
    context = "\n\n".join(
        [
            f"文件路径：{item.get('payload', {}).get('path', '')}\n"
            f"分数：{item.get('score', 0):.4f}\n"
            f"内容片段：{item.get('payload', {}).get('content', '')}"
            for item in results
        ]
    )
    prompt = f"""你是本地文件搜索助手。
根据下面的搜索结果回答用户问题。
请优先返回最可能的文件路径，并说明原因；如果结果不足，请明确说不确定。

用户问题：
{query}

搜索结果：
{context}
"""
    response = request_json(
        "POST",
        f"{api_base_url(args)}/chat/completions",
        {
            "model": args.chat_model,
            "messages": [{"role": "user", "content": prompt}],
            "temperature": args.temperature,
        },
        headers=auth_headers(args.omlx_api_key),
        timeout=args.chat_timeout,
    )
    return response["choices"][0]["message"]["content"]


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="Search local RAG index and summarize results.")
    parser.add_argument("query", nargs="+", help="Natural-language query.")
    parser.add_argument("--collection", default=os.getenv("RAG_COLLECTION", "local_files"))
    parser.add_argument("--openai-base-url", default=os.getenv("RAG_OPENAI_BASE_URL", os.getenv("OMLX_URL", "http://localhost:8000/v1")))
    parser.add_argument("--api-key", dest="omlx_api_key", default=os.getenv("RAG_OPENAI_API_KEY", os.getenv("OMLX_API_KEY", "")))
    parser.add_argument("--qdrant-url", default=os.getenv("QDRANT_URL", "http://localhost:6333"))
    parser.add_argument("--embedding-model", default=os.getenv("RAG_EMBEDDING_MODEL", "Qwen3-Embedding-4B-4bit-DWQ"))
    parser.add_argument("--chat-model", default=os.getenv("RAG_CHAT_MODEL", "Qwen3.5-4B-MLX-4bit"))
    parser.add_argument("--limit", type=int, default=int(os.getenv("RAG_SEARCH_LIMIT", "5")))
    parser.add_argument("--temperature", type=float, default=float(os.getenv("RAG_TEMPERATURE", "0.2")))
    parser.add_argument("--timeout", type=int, default=int(os.getenv("RAG_TIMEOUT", "60")))
    parser.add_argument("--chat-timeout", type=int, default=int(os.getenv("RAG_CHAT_TIMEOUT", "120")))
    parser.add_argument("--no-summary", action="store_true", help="Only print raw Qdrant matches.")
    return parser.parse_args()


def print_results(results: list[dict]) -> None:
    for item in results:
        payload = item.get("payload", {})
        print(f"score: {item.get('score', 0):.4f}")
        print(f"file: {payload.get('path', '')}")
        print(f"chunk: {payload.get('chunk_index', '')}")
        print(f"content: {payload.get('content', '')[:500]}")
        print("-" * 80)


def main() -> int:
    args = parse_args()
    query = " ".join(args.query).strip()
    if not query:
        print("query is required", file=sys.stderr)
        return 2
    results = search(query, args)
    print_results(results)
    if not args.no_summary and results:
        print("\n总结：")
        print(summarize(query, results, args))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
