#!/usr/bin/env bash
# Ensures ../agentrix_engine is checked out at the commit pinned in
# engine.lock. It never discards local changes: a dirty or divergent
# checkout is reported and the script fails.
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
LOCK="$ROOT/engine.lock"
DEST="${ENGINE_DIR:-$ROOT/../agentrix_engine}"
REPO="$(python3 -c 'import json,sys;print(json.load(open(sys.argv[1]))["repository"])' "$LOCK")"
COMMIT="$(python3 -c 'import json,sys;print(json.load(open(sys.argv[1]))["commit"])' "$LOCK")"

if [ ! -d "$DEST/.git" ]; then
  git clone --quiet "$REPO" "$DEST"
  git -C "$DEST" checkout --quiet "$COMMIT"
fi
HEAD="$(git -C "$DEST" rev-parse HEAD)"
if [ "$HEAD" != "$COMMIT" ]; then
  echo "engine checkout at $HEAD, engine.lock pins $COMMIT" >&2
  echo "check out the pinned commit (or update engine.lock deliberately)" >&2
  exit 1
fi
if [ -n "$(git -C "$DEST" status --porcelain)" ] && [ "${ALLOW_DIRTY_ENGINE:-0}" != "1" ]; then
  echo "engine checkout has uncommitted changes; set ALLOW_DIRTY_ENGINE=1 for local development only" >&2
  exit 1
fi
echo "engine $DEST at $COMMIT"
