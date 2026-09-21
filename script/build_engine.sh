#!/usr/bin/env bash
# Builds bin/starfighter-engine from the pinned engine checkout and prints
# its SHA-256 (the digest workers announce and specs pin).
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
ENGINE="${ENGINE_DIR:-$ROOT/../agentrix_engine}"
"$ROOT/script/checkout_engine.sh"
TARGET="${CARGO_TARGET_DIR:-$HOME/.cache/agentrix-engine-target}"
CARGO_TARGET_DIR="$TARGET" cargo build --locked --release --bin starfighter-engine --manifest-path "$ENGINE/Cargo.toml"
mkdir -p "$ROOT/bin"
install -m 0755 "$TARGET/release/starfighter-engine" "$ROOT/bin/starfighter-engine"
sha256sum "$ROOT/bin/starfighter-engine"
