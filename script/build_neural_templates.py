#!/usr/bin/env python3
"""Arma las plantillas de bots con red neuronal para la web (ADR-0014, N5).

Genera, desde games/starfighter/examples/neural, dos paquetes v2 listos para
subir:

- web/public/starfighter-neural-onnx.zip  (model/policy.onnx, onnxruntime)
- web/public/starfighter-neural-npz.zip   (model/policy.npz, numpy)

Los ZIP son reproducibles: orden y fechas fijos, así que regenerarlos sin
cambios en los ejemplos no produce diferencias en git.

    python3 script/build_neural_templates.py
"""
import json
import pathlib
import zipfile

ROOT = pathlib.Path(__file__).resolve().parent.parent
NEURAL = ROOT / "games" / "starfighter" / "examples" / "neural"
OUT = ROOT / "web" / "public"
FIXED_DATE = (2026, 9, 22, 0, 0, 0)

TEMPLATES = {
    "starfighter-neural-onnx.zip": ("Starfighter Neural ONNX", "bot_onnx.py", "policy.onnx"),
    "starfighter-neural-npz.zip": ("Starfighter Neural NPZ", "bot_npz.py", "policy.npz"),
}


def build(zip_name, bot_name, bot_file, model_file):
    manifest = {
        "name": bot_name,
        "entrypoint": "bot.py",
        "protocol_version": "1.0",
        "runtime": "python-ml-cpu",
    }
    files = [
        ("agentrix.json", (json.dumps(manifest, indent=2) + "\n").encode()),
        ("bot.py", (NEURAL / bot_file).read_bytes()),
        ("policy.py", (NEURAL / "policy.py").read_bytes()),
        ("model/" + model_file, (NEURAL / "model" / model_file).read_bytes()),
    ]
    with zipfile.ZipFile(OUT / zip_name, "w", zipfile.ZIP_DEFLATED) as zf:
        for name, content in files:
            info = zipfile.ZipInfo(name, FIXED_DATE)
            info.compress_type = zipfile.ZIP_DEFLATED
            info.external_attr = 0o644 << 16
            zf.writestr(info, content)


def main():
    for zip_name, (bot_name, bot_file, model_file) in TEMPLATES.items():
        build(zip_name, bot_name, bot_file, model_file)
        print(OUT / zip_name)


if __name__ == "__main__":
    main()
