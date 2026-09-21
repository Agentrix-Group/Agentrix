import React from 'react';
import { Loader2 } from 'lucide-react';
import { useTranslation } from 'react-i18next';

export function LoadingState({ message, size = 32, className = '' }) {
  const { t } = useTranslation('common');
  const displayMessage = message || t('buttons.loading', 'Cargando...');

  return (
    <div
      role="status"
      aria-live="polite"
      className={`loading-state ${className}`}
      style={{
        display: 'flex',
        flexDirection: 'column',
        alignItems: 'center',
        justifyContent: 'center',
        padding: '48px 24px',
        color: 'var(--text-secondary, #94a3b8)',
        gap: '14px',
      }}
    >
      <Loader2
        size={size}
        className="animate-spin"
        style={{
          animation: 'spin 1s linear infinite',
          color: 'var(--accent, #3b82f6)',
        }}
        aria-hidden="true"
      />
      <span style={{ fontSize: '0.95rem' }}>{displayMessage}</span>
    </div>
  );
}

export default LoadingState;
