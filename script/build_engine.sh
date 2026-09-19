#!/usr/bin/env bash
set -euo pipefail

ENGINE_SRC="${1:-../agentrix_engine}"
TARGET_BIN="${2:-bin/starfighter-engine}"

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$REPO_ROOT"

if [ ! -d "$ENGINE_SRC" ]; then
    if [ -d "$REPO_ROOT/../agentrix_engine" ]; then
        ENGINE_SRC="$REPO_ROOT/../agentrix_engine"
    else
        echo "Error: Starfighter engine source directory '$ENGINE_SRC' not found."
        echo "Please set ENGINE_SRC environment variable or clone agentrix_engine in the parent directory."
        exit 1
    fi
fi

echo "==> Compiling Starfighter engine (release mode)..."
cargo build --release --bin starfighter-engine --manifest-path "$ENGINE_SRC/Cargo.toml"

mkdir -p "$(dirname "$TARGET_BIN")"
cp "$ENGINE_SRC/target/release/starfighter-engine" "$TARGET_BIN"
chmod +x "$TARGET_BIN"

echo "==> Starfighter engine successfully installed to $TARGET_BIN"
