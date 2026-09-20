import React from 'react';
import { useTranslation } from 'react-i18next';
import { LogIn, ShieldAlert } from 'lucide-react';
import { useSession } from '../auth/SessionContext.jsx';
import { useRouter } from '../router/Router.jsx';

export function ProtectedRoute({ children, requiredCapability = null }) {
  const { t } = useTranslation(['agents', 'common']);
  const { isAuthenticated, isLoading, hasCapability } = useSession();
  const { currentPath, navigate } = useRouter();

  if (isLoading) {
    return (
      <div style={{ textAlign: 'center', padding: '60px 20px', color: 'var(--text-secondary)' }}>
        {t('common:buttons.loading')}
      </div>
    );
  }

  if (!isAuthenticated) {
    const handleGoLogin = () => {
      navigate(`/auth?redirect=${encodeURIComponent(currentPath)}`);
    };

    return (
      <div
        className="card"
        style={{
          textAlign: 'center',
          margin: '48px auto',
          maxWidth: '440px',
          padding: '36px 24px',
        }}
      >
        <div
          style={{
            display: 'inline-flex',
            padding: '12px',
            background: 'rgba(239, 68, 68, 0.12)',
            borderRadius: '50%',
            marginBottom: '16px',
          }}
        >
          <ShieldAlert size={32} color="var(--danger, #ef4444)" aria-hidden="true" />
        </div>
        <h3 style={{ marginBottom: '8px' }}>{t('agents:authRequired.title')}</h3>
        <p style={{ color: 'var(--text-secondary)', marginBottom: '24px', fontSize: '0.92rem' }}>
          {t('agents:authRequired.description')}
        </p>
        <button
          type="button"
          className="btn"
          onClick={handleGoLogin}
          style={{ display: 'inline-flex', alignItems: 'center', gap: '8px' }}
        >
          <LogIn size={16} aria-hidden="true" /> Ingresar a tu cuenta
        </button>
      </div>
    );
  }

  if (requiredCapability && !hasCapability(requiredCapability)) {
    return (
      <div
        role="alert"
        className="card"
        style={{
          textAlign: 'center',
          margin: '48px auto',
          maxWidth: '440px',
          borderLeft: '4px solid var(--danger, #ef4444)',
        }}
      >
        <h3>Acceso restringido</h3>
        <p style={{ color: 'var(--text-secondary)' }}>
          Tu cuenta no cuenta con los permisos requeridos para esta acción ({requiredCapability}).
        </p>
      </div>
    );
  }

  return children;
}
