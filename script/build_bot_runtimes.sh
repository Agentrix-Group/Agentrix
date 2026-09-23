#!/usr/bin/env bash
# Construye los runtimes de bots de ADR-0014 en una carpeta autocontenida:
#
#   <dir>/python/                      Python 3.12 independiente (python-build-standalone)
#   <dir>/python-ml-cpu/               entorno con numpy, onnxruntime y safetensors
#
# Solo descarga paquetes al construir; nunca durante una partida. Las
# dependencias se instalan exclusivamente desde el lock con hashes.
#
# Uso: script/build_bot_runtimes.sh [dir]   (por defecto bin/runtimes)
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DEST="$(realpath -m "${1:-$ROOT/bin/runtimes}")"
PYTHON_VERSION="3.12.14"
LOCK="$ROOT/runtimes/python-ml-cpu/requirements.lock"

command -v uv >/dev/null || { echo "uv is required (https://docs.astral.sh/uv/)" >&2; exit 1; }
mkdir -p "$DEST"

echo "==> Python $PYTHON_VERSION standalone in $DEST/python"
uv python install --install-dir "$DEST/python" --no-bin "$PYTHON_VERSION"
PYTHON="$(find "$DEST/python" -maxdepth 3 -path "*cpython-$PYTHON_VERSION-*/bin/python3.12" | head -1)"
[ -x "$PYTHON" ] || { echo "standalone python $PYTHON_VERSION not found" >&2; exit 1; }

echo "==> python-ml-cpu environment"
rm -rf "$DEST/python-ml-cpu"
uv venv --python "$PYTHON" "$DEST/python-ml-cpu"
uv pip install --python "$DEST/python-ml-cpu/bin/python" --require-hashes --no-deps -r "$LOCK"

"$DEST/python-ml-cpu/bin/python" - <<'PY'
import numpy, onnxruntime, safetensors, sys
print("   python", sys.version.split()[0], "numpy", numpy.__version__,
      "onnxruntime", onnxruntime.__version__, "safetensors", safetensors.__version__)
PY
echo "==> Bot runtimes ready in $DEST"
