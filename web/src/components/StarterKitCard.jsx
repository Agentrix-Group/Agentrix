import React, { useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Download, ChevronDown, ChevronUp, FileCode, Package } from 'lucide-react';

export function StarterKitCard() {
  const { t } = useTranslation(['agents', 'common']);
  const [showSpec, setShowSpec] = useState(false);

  const sampleManifest = JSON.stringify(
    {
      name: 'Starfighter Starter Bot',
      entrypoint: 'bot.py',
      protocol_version: '1.0',
    },
    null,
    2
  );

  const sampleBotPreview = `#!/usr/bin/env python3
"""Starfighter Starter Bot -- Agentrix ATD-007 reference."""
import json, sys

def send(msg):
    sys.stdout.write(json.dumps(msg) + "\\n")
    sys.stdout.flush()

def read():
    line = sys.stdin.readline()
    return json.loads(line) if line else sys.exit(0)

def main():
    init = read()
    assert init["type"] == "init"
    while True:
        msg = read()
        if msg["type"] == "end": break
        if msg["type"] == "perception":
            action = {"thrust": "FORWARD", "turn": "NONE", "shoot": False, "shield": False}
            send({"type": "action", "tick": msg["tick"], "action": action})

if __name__ == "__main__":
    main()`;

  return (
    <div className="starter-template-box card" style={{ marginBottom: '20px' }}>
      <div className="starter-actions">
        <div style={{ display: 'flex', alignItems: 'center', gap: '10px' }}>
          <Package size={22} color="var(--accent)" aria-hidden="true" />
          <div>
            <strong>{t('agents:submission.starterTitle', { defaultValue: 'Plantilla de inicio para Starfighter' })}</strong>
            <div style={{ fontSize: '0.82rem', color: 'var(--text-secondary)' }}>
              {t('agents:submission.starterHelp', { defaultValue: 'Paquete ZIP listo para admitir con agentrix.json y bot.py de referencia.' })}
            </div>
          </div>
        </div>

        <div style={{ display: 'flex', gap: '8px', alignItems: 'center' }}>
          <a
            href="/starfighter-starter.zip"
            download="starfighter-starter.zip"
            className="btn btn-compact"
            style={{ textDecoration: 'none', display: 'inline-flex', alignItems: 'center', gap: '6px' }}
          >
            <Download size={15} aria-hidden="true" />
            <span>{t('agents:submission.downloadStarter', { defaultValue: 'Descargar Starter Kit (.zip)' })}</span>
          </a>

          <button
            type="button"
            className="btn btn-compact"
            onClick={() => setShowSpec(!showSpec)}
            style={{ background: 'transparent', border: '1px solid var(--border)', color: 'var(--text-primary)' }}
            aria-expanded={showSpec}
          >
            <FileCode size={15} aria-hidden="true" />
            <span>{showSpec ? t('agents:submission.hideSpec', { defaultValue: 'Ocultar' }) : t('agents:submission.viewSpec', { defaultValue: 'Ver especificación' })}</span>
            {showSpec ? <ChevronUp size={14} aria-hidden="true" /> : <ChevronDown size={14} aria-hidden="true" />}
          </button>
        </div>
      </div>

      {showSpec && (
        <div className="starter-spec-details">
          <p style={{ marginTop: 0 }}>{t('agents:submission.manifestDesc', { defaultValue: "agentrix.json debe incluir name, entrypoint: 'bot.py' y protocol_version: '1.0'." })}</p>
          <strong>agentrix.json:</strong>
          <pre>{sampleManifest}</pre>
          <p style={{ marginTop: '12px' }}>{t('agents:submission.botDesc', { defaultValue: "bot.py debe manejar el mensaje 'init' y responder con 'action' en el tick 0." })}</p>
          <strong>bot.py (resumen):</strong>
          <pre>{sampleBotPreview}</pre>
        </div>
      )}
    </div>
  );
}
