import React from 'react';
import { useTranslation } from 'react-i18next';
import { Clock, LogIn, X } from 'lucide-react';
import { useSession } from '../auth/SessionContext.jsx';
import { useRouter } from '../router/Router.jsx';

export function SessionExpiredBanner() {
  const { t } = useTranslation(['auth', 'common']);
  const { sessionExpired, dismissExpiredNotice } = useSession();
  const { navigate, currentPath } = useRouter();

  if (!sessionExpired) return null;

  const handleReconnect = () => {
    dismissExpiredNotice();
    const redirectQuery = currentPath && currentPath !== '/' && currentPath !== '/auth'
      ? `?redirect=${encodeURIComponent(currentPath)}`
      : '';
    navigate(`/auth${redirectQuery}`);
  };

  return (
    <div
      role="alert"
      aria-live="assertive"
      style={{
        background: 'linear-gradient(90deg, #b91c1c 0%, #dc2626 100%)',
        color: '#ffffff',
        padding: '12px 20px',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'space-between',
        flexWrap: 'wrap',
        gap: '12px',
        boxShadow: '0 4px 12px rgba(0,0,0,0.2)',
        position: 'sticky',
        top: 0,
        zIndex: 1000,
      }}
    >
      <div style={{ display: 'flex', alignItems: 'center', gap: '10px' }}>
        <Clock size={20} aria-hidden="true" />
        <span style={{ fontWeight: 600 }}>{t('auth:session.expiredTitle')}:</span>
        <span style={{ fontSize: '0.92rem' }}>{t('auth:session.expiredMessage')}</span>
      </div>

      <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
        <button
          type="button"
          onClick={handleReconnect}
          style={{
            background: '#ffffff',
            color: '#b91c1c',
            border: 'none',
            borderRadius: '6px',
            padding: '6px 14px',
            fontWeight: 600,
            cursor: 'pointer',
            display: 'inline-flex',
            alignItems: 'center',
            gap: '6px',
            fontSize: '0.88rem',
          }}
        >
          <LogIn size={15} aria-hidden="true" /> {t('auth:session.reconnect')}
        </button>
        <button
          type="button"
          onClick={dismissExpiredNotice}
          aria-label={t('common:buttons.close')}
          style={{
            background: 'transparent',
            border: 'none',
            color: '#ffffff',
            cursor: 'pointer',
            padding: '4px',
            display: 'inline-flex',
            alignItems: 'center',
          }}
        >
          <X size={18} aria-hidden="true" />
        </button>
      </div>
    </div>
  );
}
