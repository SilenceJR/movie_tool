#!/usr/bin/env python3
"""Index local text files into Qdrant using an OpenAI-compatible embedding API."""

from __future__ import annotations

import argparse
import hashlib
import json
import os
import sys
import time
import urllib.error
import urllib.request
from pathlib import Path


DEFAULT_EXTENSIONS = {
    ".txt",
    ".md",
    ".json",
    ".yaml",
    ".yml",
    ".log",
    ".py",
    ".go",
    ".rs",
    ".dart",
    ".js",
    ".ts",
}


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
            "input": text[: args.max_embedding_chars],
        },
        headers=auth_headers(args.omlx_api_key),
        timeout=args.timeout,
    )
    return response["data"][0]["embedding"]


def ensure_collection(args: argparse.Namespace, vector_size: int) -> None:
    url = f"{args.qdrant_url.rstrip('/')}/collections/{args.collection}"
    try:
        request_json("GET", url, timeout=args.timeout)
        print(f"collection exists: {args.collection}")
        return
    except RuntimeError as error:
        if " 404 " not in str(error):
            raise
    request_json(
        "PUT",
        url,
        {
            "vectors": {
                "size": vector_size,
                "distance": "Cosine",
                "on_disk": args.on_disk_vectors,
            }
        },
        timeout=args.timeout,
    )
    print(f"collection created: {args.collection} dim={vector_size}")


def chunk_text(text: str, size: int, overlap: int) -> list[str]:
    if size <= 0:
        raise ValueError("chunk size must be positive")
    if overlap < 0 or overlap >= size:
        raise ValueError("chunk overlap must be >= 0 and smaller than chunk size")
    chunks: list[str] = []
    start = 0
    while start < len(text):
        end = min(start + size, len(text))
        chunk = text[start:end].strip()
        if chunk:
            chunks.append(chunk)
        if end == len(text):
            break
        start = end - overlap
    return chunks


def stable_point_id(path: Path, index: int, chunk: str) -> str:
    digest = hashlib.sha256()
    digest.update(str(path).encode("utf-8"))
    digest.update(b"\0")
    digest.update(str(index).encode("utf-8"))
    digest.update(b"\0")
    digest.update(chunk[:512].encode("utf-8", errors="ignore"))
    hex_value = digest.hexdigest()
    return f"{hex_value[:8]}-{hex_value[8:12]}-{hex_value[12:16]}-{hex_value[16:20]}-{hex_value[20:32]}"


def iter_files(root: Path, extensions: set[str]):
    for path in root.rglob("*"):
        if not path.is_file():
            continue
        if path.suffix.lower() not in extensions:
            continue
        yield path


def flush_points(args: argparse.Namespace, points: list[dict]) -> None:
    if not points:
        return
    request_json(
        "PUT",
        f"{args.qdrant_url.rstrip('/')}/collections/{args.collection}/points",
        {"points": points},
        timeout=args.timeout,
    )


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="Index local text files into Qdrant via oMLX embeddings.")
    parser.add_argument("root", type=Path, help="Directory to scan.")
    parser.add_argument("--collection", default=os.getenv("RAG_COLLECTION", "local_files"))
    parser.add_argument("--openai-base-url", default=os.getenv("RAG_OPENAI_BASE_URL", os.getenv("OMLX_URL", "http://localhost:8000/v1")))
    parser.add_argument("--api-key", dest="omlx_api_key", default=os.getenv("RAG_OPENAI_API_KEY", os.getenv("OMLX_API_KEY", "")))
    parser.add_argument("--qdrant-url", default=os.getenv("QDRANT_URL", "http://localhost:6333"))
    parser.add_argument("--embedding-model", default=os.getenv("RAG_EMBEDDING_MODEL", "Qwen3-Embedding-4B-4bit-DWQ"))
    parser.add_argument("--chunk-size", type=int, default=int(os.getenv("RAG_CHUNK_SIZE", "1200")))
    parser.add_argument("--chunk-overlap", type=int, default=int(os.getenv("RAG_CHUNK_OVERLAP", "200")))
    parser.add_argument("--batch-size", type=int, default=int(os.getenv("RAG_BATCH_SIZE", "16")))
    parser.add_argument("--max-embedding-chars", type=int, default=int(os.getenv("RAG_MAX_EMBEDDING_CHARS", "8000")))
    parser.add_argument("--timeout", type=int, default=int(os.getenv("RAG_TIMEOUT", "120")))
    parser.add_argument("--on-disk-vectors", action="store_true", default=os.getenv("RAG_ON_DISK_VECTORS", "") == "1")
    parser.add_argument("--extensions", default=os.getenv("RAG_EXTENSIONS", ",".join(sorted(DEFAULT_EXTENSIONS))))
    return parser.parse_args()


def main() -> int:
    args = parse_args()
    root = args.root.expanduser().resolve()
    if not root.exists() or not root.is_dir():
        print(f"root directory not found: {root}", file=sys.stderr)
        return 2
    extensions = {item.strip().lower() for item in args.extensions.split(",") if item.strip()}
    probe = embed("collection dimension probe", args)
    ensure_collection(args, len(probe))

    points: list[dict] = []
    indexed_files = 0
    indexed_chunks = 0
    started = time.time()
    for path in iter_files(root, extensions):
        try:
            text = path.read_text(encoding="utf-8", errors="ignore")
        except OSError as error:
            print(f"skip unreadable file: {path} ({error})", file=sys.stderr)
            continue
        chunks = chunk_text(text, args.chunk_size, args.chunk_overlap)
        if not chunks:
            continue
        indexed_files += 1
        stat = path.stat()
        for index, chunk in enumerate(chunks):
            vector = embed(chunk, args)
            points.append(
                {
                    "id": stable_point_id(path, index, chunk),
                    "vector": vector,
                    "payload": {
                        "path": str(path),
                        "filename": path.name,
                        "extension": path.suffix.lower(),
                        "chunk_index": index,
                        "mtime": int(stat.st_mtime),
                        "size": stat.st_size,
                        "content": chunk[:2000],
                    },
                }
            )
            indexed_chunks += 1
            if len(points) >= args.batch_size:
                flush_points(args, points)
                points = []
        print(f"indexed: {path} chunks={len(chunks)}")
    flush_points(args, points)
    elapsed = time.time() - started
    print(f"done files={indexed_files} chunks={indexed_chunks} elapsed={elapsed:.1f}s collection={args.collection}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
