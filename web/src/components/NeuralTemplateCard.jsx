import React, { useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Brain, Download, ChevronDown, ChevronUp, BookOpen } from 'lucide-react';

// Plantillas de bots con red neuronal (ADR-0014, N5). Los ZIP se generan
// con `make neural-templates` desde games/starfighter/examples/neural.
export const NEURAL_TEMPLATES = [
  { file: 'starfighter-neural-onnx.zip', key: 'onnx' },
  { file: 'starfighter-neural-npz.zip', key: 'npz' },
];

const GUIDE_STEPS = ['download', 'model', 'manifest', 'limits', 'upload'];

const SAMPLE_MANIFEST = JSON.stringify(
  {
    name: 'MiBotNeuronal',
    entrypoint: 'bot.py',
    protocol_version: '1.0',
    runtime: 'python-ml-cpu',
  },
  null,
  2
);

export function NeuralTemplateCard() {
  const { t } = useTranslation(['agents']);
  const [showGuide, setShowGuide] = useState(false);

  return (
    <div className="starter-template-box card" style={{ marginBottom: '20px' }}>
      <div className="starter-actions">
        <div style={{ display: 'flex', alignItems: 'center', gap: '10px' }}>
          <Brain size={22} color="var(--accent)" aria-hidden="true" />
          <div>
            <strong>{t('agents:neural.title')}</strong>
            <div style={{ fontSize: '0.82rem', color: 'var(--text-secondary)' }}>{t('agents:neural.help')}</div>
          </div>
        </div>

        <div style={{ display: 'flex', gap: '8px', alignItems: 'center', flexWrap: 'wrap' }}>
          {NEURAL_TEMPLATES.map(({ file, key }) => (
            <a
              key={file}
              href={`/${file}`}
              download={file}
              className="btn btn-compact"
              style={{ textDecoration: 'none', display: 'inline-flex', alignItems: 'center', gap: '6px' }}
            >
              <Download size={15} aria-hidden="true" />
              <span>{t(`agents:neural.download.${key}`)}</span>
            </a>
          ))}
          <button
            type="button"
            className="btn btn-compact"
            onClick={() => setShowGuide(!showGuide)}
            style={{ background: 'transparent', border: '1px solid var(--border)', color: 'var(--text-primary)' }}
            aria-expanded={showGuide}
          >
            <BookOpen size={15} aria-hidden="true" />
            <span>{showGuide ? t('agents:neural.hideGuide') : t('agents:neural.showGuide')}</span>
            {showGuide ? <ChevronUp size={14} aria-hidden="true" /> : <ChevronDown size={14} aria-hidden="true" />}
          </button>
        </div>
      </div>

      {showGuide && (
        <div className="starter-spec-details">
          <strong>{t('agents:neural.guideTitle')}</strong>
          <ol className="neural-guide-steps">
            {GUIDE_STEPS.map((step) => (
              <li key={step}>{t(`agents:neural.steps.${step}`)}</li>
            ))}
          </ol>
          <strong>agentrix.json:</strong>
          <pre>{SAMPLE_MANIFEST}</pre>
          <p style={{ marginBottom: 0 }}>{t('agents:neural.runtimeVersions')}</p>
        </div>
      )}
    </div>
  );
}
