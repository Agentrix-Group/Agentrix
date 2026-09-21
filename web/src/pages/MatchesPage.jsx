import React from 'react';
import { useTranslation } from 'react-i18next';
import { Swords } from 'lucide-react';
import { ApiService } from '../service/apiService.js';
import { useResource, usePolling } from '../hooks/hooks.js';
import { Link } from '../router/Router.jsx';
import { SkeletonRows, ErrorState, EmptyState } from '../components/States.jsx';
import { StatusBadge } from '../components/StatusBadge.jsx';
import { formatDateTime } from '../i18n/formatters.js';

export default function MatchesPage({ contestId }) {
  const { t, i18n } = useTranslation('matches');
  const matches = useResource((signal) => ApiService.listMatches(contestId, { signal }), [contestId]);
  const live = (matches.data?.items || []).some((m) => m.state === 'queued' || m.state === 'running');
  usePolling(matches.reload, 3000, live);
  return (
    <div className="page">
      <div className="page-header">
        <h1>{t('title')}</h1>
        {contestId && <Link to={`/contests/${contestId}`} className="btn btn-secondary">{t('backToContest')}</Link>}
      </div>
      {matches.loading && !matches.data && <SkeletonRows rows={5} />}
      {matches.error && <ErrorState error={matches.error} onRetry={matches.reload} />}
      {matches.data && matches.data.items.length === 0 && <EmptyState icon={Swords} title={t('empty')} />}
      {matches.data && matches.data.items.length > 0 && (
        <div className="table-wrap card">
          <table>
            <thead><tr><th>{t('roster')}</th><th>{t('mode')}</th><th>{t('state')}</th><th>{t('updated')}</th></tr></thead>
            <tbody>
              {matches.data.items.map((m) => (
                <tr key={m.id}>
                  <td><Link to={`/matches/${m.id}`}>{m.slots.map((s) => s.display_name).join(' vs ')}</Link></td>
                  <td>{t(`modes.${m.mode}`)}</td>
                  <td><StatusBadge state={m.state} /></td>
                  <td className="muted small">{formatDateTime(m.updated_at, i18n.language)}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}
