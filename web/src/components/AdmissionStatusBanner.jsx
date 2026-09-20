import React from 'react';
import { useTranslation } from 'react-i18next';
import { AlertCircle, CheckCircle2, Download, Loader2, ShieldAlert } from 'lucide-react';

export function sanitizeAdmissionError(rawMessage) {
  if (!rawMessage || typeof rawMessage !== 'string') return '';
  // Strip server absolute paths such as /tmp/agentrix-admission-... or /artifacts/...
  return rawMessage
    .replace(/(?:\/[a-zA-Z0-9_.-]+)+\/bot\.py/g, 'bot.py')
    .replace(/(?:\/[a-zA-Z0-9_.-]+)+\/agentrix\.json/g, 'agentrix.json')
    .replace(/(?:\/[a-zA-Z0-9_.-]+)+/g, '[internal path]');
}

export function AdmissionStatusBanner({ stage, errorDetail, onDismiss }) {
  const { t } = useTranslation(['agents', 'common']);

  if (stage === 'validating') {
    return (
      <div className="admission-progress-card validating" role="status" aria-live="polite">
        <Loader2 size={20} className="spin" style={{ animation: 'spin 1s linear infinite' }} aria-hidden="true" />
        <div>
          <strong>{t('agents:submission.validationStages.validating', { defaultValue: 'Validando en sandbox (sintaxis, inicialización y tick 0)...' })}</strong>
          <div style={{ fontSize: '0.82rem', marginTop: '2px', opacity: 0.9 }}>
            {t('agents:submission.admissionCheck', { defaultValue: 'Agentrix comprobará sintaxis, inicialización y respuesta al tick 0.' })}
          </div>
        </div>
      </div>
    );
  }

  if (stage === 'uploading') {
    return (
      <div className="admission-progress-card validating" role="status" aria-live="polite">
        <Loader2 size={20} className="spin" style={{ animation: 'spin 1s linear infinite' }} aria-hidden="true" />
        <div>
          <strong>{t('agents:submission.validationStages.uploading', { defaultValue: 'Subiendo paquete ZIP...' })}</strong>
        </div>
      </div>
    );
  }

  if (stage === 'rejected' || errorDetail) {
    const sanitized = sanitizeAdmissionError(errorDetail);
    return (
      <div className="admission-rejection-card" role="alert">
        <h4>
          <ShieldAlert size={20} aria-hidden="true" />
          {t('agents:submission.rejection.title', { defaultValue: 'Fallo en la validación del bot' })}
        </h4>

        {sanitized && (
          <p style={{ margin: '6px 0 10px', fontWeight: 500, fontSize: '0.9rem' }}>
            {sanitized}
          </p>
        )}

        <div style={{ fontSize: '0.84rem', marginTop: '8px' }}>
          <strong>{t('agents:submission.rejection.guidanceTitle', { defaultValue: 'Comprobaciones necesarias para resolver el fallo:' })}</strong>
          <ul>
            <li>{t('agents:submission.rejection.rule1', { defaultValue: "El archivo agentrix.json debe existir en la raíz del ZIP con entrypoint='bot.py' y protocol_version='1.0'." })}</li>
            <li>{t('agents:submission.rejection.rule2', { defaultValue: 'bot.py debe tener sintaxis Python válida y responder al tick 0 en el tiempo estipulado.' })}</li>
            <li>{t('agents:submission.rejection.rule3', { defaultValue: 'No debe utilizar librerías fuera de la estándar de Python ni realizar llamadas bloqueantes o de red.' })}</li>
          </ul>
        </div>

        <div style={{ marginTop: '12px', display: 'flex', gap: '10px', alignItems: 'center' }}>
          <a
            href="/starfighter-starter.zip"
            download="starfighter-starter.zip"
            className="btn btn-compact"
            style={{
              background: '#fff',
              color: '#9f1239',
              border: '1px solid #fecdd3',
              textDecoration: 'none',
              display: 'inline-flex',
              alignItems: 'center',
              gap: '6px',
            }}
          >
            <Download size={14} aria-hidden="true" />
            <span>{t('agents:submission.rejection.downloadReference', { defaultValue: 'Descarga el Starter Kit para verificar la estructura requerida.' })}</span>
          </a>
        </div>
      </div>
    );
  }

  return null;
}
