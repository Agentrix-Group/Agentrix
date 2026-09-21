import React from 'react';
import { useTranslation } from 'react-i18next';
import { ShieldAlert } from 'lucide-react';
import { useSession } from '../auth/SessionContext.jsx';
import { Link, useRouter } from '../router/Router.jsx';
import { LoadingState, EmptyState } from './States.jsx';

/** Guards a page on the client. The API enforces the same policy. */
export function ProtectedRoute({ children, capability }) {
  const { t } = useTranslation('common');
  const { isAuthenticated, isLoading, can } = useSession();
  const { path } = useRouter();
  if (isLoading) return <LoadingState />;
  if (!isAuthenticated) {
    return (
      <EmptyState icon={ShieldAlert} title={t('auth.required')} description={t('auth.requiredDescription')}
        action={<Link className="btn" to={`/auth?next=${encodeURIComponent(path)}`}>{t('auth.signIn')}</Link>} />
    );
  }
  if (capability && !can(capability)) {
    return <EmptyState icon={ShieldAlert} title={t('auth.forbidden')} description={t('auth.forbiddenDescription')} />;
  }
  return children;
}
