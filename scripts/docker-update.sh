#!/usr/bin/env sh
set -eu

ROOT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
COMPOSE_FILE="${COMPOSE_FILE:-$ROOT_DIR/deployments/compose/docker-compose.yml}"
LOCK_DIR="${TMPDIR:-/tmp}/movie-tool-docker-update.lock"

log() {
  printf '[docker-update] %s\n' "$*"
}

cleanup() {
  rmdir "$LOCK_DIR" 2>/dev/null || true
}

if ! command -v docker >/dev/null 2>&1; then
  log "docker CLI not found; start OrbStack or install Docker CLI."
  exit 1
fi

if ! docker info >/dev/null 2>&1; then
  log "docker is not reachable; start OrbStack and try again."
  exit 1
fi

if ! mkdir "$LOCK_DIR" 2>/dev/null; then
  log "another update is already running; skipping this trigger."
  exit 0
fi
trap cleanup EXIT INT TERM

log "building and refreshing local containers with $COMPOSE_FILE"
cd "$ROOT_DIR"
docker compose -f "$COMPOSE_FILE" up --build -d --remove-orphans
log "local docker deployment is up to date."
