import React from 'react';
import { AlertCircle, RefreshCw } from 'lucide-react';
import { useTranslation } from 'react-i18next';

export function ErrorState({ message, onRetry, retryLabel, title, className = '' }) {
  const { t } = useTranslation('common');
  const displayTitle = title || t('messages.error', 'Error');
  const displayMessage = message || t('messages.operationFailed', 'Ha ocurrido un error');
  const displayRetry = retryLabel || t('buttons.retry', 'Reintentar');

  return (
    <div
      role="alert"
      className={`card error-state ${className}`}
      style={{
        borderLeft: '4px solid var(--danger, #ef4444)',
        padding: '24px',
        margin: '24px 0',
        background: 'var(--card-bg, #1e2430)',
      }}
    >
      <div style={{ display: 'flex', alignItems: 'flex-start', gap: '16px' }}>
        <div
          style={{
            display: 'inline-flex',
            padding: '8px',
            borderRadius: '50%',
            background: 'rgba(239, 68, 68, 0.15)',
            color: 'var(--danger, #ef4444)',
            flexShrink: 0,
          }}
        >
          <AlertCircle size={24} aria-hidden="true" />
        </div>
        <div style={{ flex: 1 }}>
          {title && (
            <h3 style={{ margin: '0 0 6px 0', fontSize: '1.1rem', color: 'var(--text-primary, #f8fafc)' }}>
              {displayTitle}
            </h3>
          )}
          <p style={{ margin: '0 0 16px 0', color: 'var(--text-secondary, #94a3b8)', fontSize: '0.95rem' }}>
            {displayMessage}
          </p>
          {onRetry && (
            <button
              type="button"
              className="btn btn-secondary"
              onClick={onRetry}
              style={{ display: 'inline-flex', alignItems: 'center', gap: '8px' }}
            >
              <RefreshCw size={16} aria-hidden="true" /> {displayRetry}
            </button>
          )}
        </div>
      </div>
    </div>
  );
}

export default ErrorState;
