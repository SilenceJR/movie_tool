#!/usr/bin/env sh
set -eu

ROOT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)

git -C "$ROOT_DIR" config core.hooksPath .githooks

printf 'Git hooks enabled: core.hooksPath=.githooks\n'
printf 'post-commit and post-push will run: scripts/docker-update.sh\n'
