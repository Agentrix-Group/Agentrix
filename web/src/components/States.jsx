import React from 'react';
import { AlertCircle, Inbox, Loader2, RefreshCw } from 'lucide-react';
import { useTranslation } from 'react-i18next';

export function LoadingState({ message }) {
  const { t } = useTranslation('common');
  return (
    <div className="state state-loading" role="status" aria-live="polite">
      <Loader2 size={28} className="spin" aria-hidden="true" />
      <span>{message || t('status.loading')}</span>
    </div>
  );
}

export function SkeletonRows({ rows = 3 }) {
  return (
    <div className="skeleton-list" aria-hidden="true">
      {Array.from({ length: rows }, (_, i) => <div key={i} className="skeleton-row" />)}
    </div>
  );
}

/** Maps an ApiError to a translated, user-facing message. */
export function useErrorMessage() {
  const { t } = useTranslation('errors');
  return (error) => {
    if (!error) return '';
    const byCode = error.code && t(`codes.${error.code}`, { defaultValue: '' });
    if (byCode) return byCode;
    if (error.isNetwork) return t('network');
    if (error.status >= 500) return t('server');
    return error.message || t('unknown');
  };
}

export function ErrorState({ error, onRetry, title }) {
  const { t } = useTranslation('common');
  const describe = useErrorMessage();
  return (
    <div className="state state-error card" role="alert">
      <AlertCircle size={24} aria-hidden="true" />
      <div>
        <h3>{title || t('status.error')}</h3>
        <p>{describe(error)}</p>
        {error?.requestId && <p className="muted small">{t('labels.requestId')}: <code>{error.requestId}</code></p>}
        {onRetry && (
          <button type="button" className="btn btn-secondary" onClick={onRetry}>
            <RefreshCw size={16} aria-hidden="true" /> {t('buttons.retry')}
          </button>
        )}
      </div>
    </div>
  );
}

export function EmptyState({ title, description, action, icon: Icon = Inbox }) {
  return (
    <div className="state state-empty card">
      <Icon size={28} aria-hidden="true" />
      {title && <h3>{title}</h3>}
      {description && <p>{description}</p>}
      {action}
    </div>
  );
}
