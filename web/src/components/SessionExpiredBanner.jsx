import React from 'react';
import { useTranslation } from 'react-i18next';
import { useSession } from '../auth/SessionContext.jsx';
import { Link } from '../router/Router.jsx';

export function SessionExpiredBanner() {
  const { t } = useTranslation('common');
  const { sessionExpired, dismissExpired } = useSession();
  if (!sessionExpired) return null;
  return (
    <div className="banner banner-warning" role="alert">
      <span>{t('auth.expired')}</span>
      <Link to="/auth" className="btn btn-secondary" onClick={dismissExpired}>{t('auth.signIn')}</Link>
    </div>
  );
}
