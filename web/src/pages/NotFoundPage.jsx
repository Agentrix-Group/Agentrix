import React from 'react';
import { useTranslation } from 'react-i18next';
import { Compass } from 'lucide-react';
import { EmptyState } from '../components/States.jsx';
import { Link, useRouter } from '../router/Router.jsx';

export default function NotFoundPage() {
  const { t } = useTranslation('common');
  const { path } = useRouter();
  return (
    <section aria-labelledby="notfound-title">
      <h1 id="notfound-title" className="visually-hidden">404</h1>
      <EmptyState icon={Compass} title={t('notFound.title')} description={t('notFound.description', { path })}
        action={<Link to="/" className="btn">{t('notFound.home')}</Link>} />
    </section>
  );
}
