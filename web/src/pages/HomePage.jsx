import React from 'react';
import { useTranslation } from 'react-i18next';
import { Trophy, Swords, Bot } from 'lucide-react';
import { ApiService } from '../service/apiService.js';
import { useResource } from '../hooks/hooks.js';
import { Link } from '../router/Router.jsx';
import { SkeletonRows, ErrorState } from '../components/States.jsx';
import { StatusBadge } from '../components/StatusBadge.jsx';
import { formatDateTime } from '../i18n/formatters.js';

export default function HomePage() {
  const { t, i18n } = useTranslation('home');
  const contests = useResource((signal) => ApiService.listContests({ signal }), []);
  const matches = useResource((signal) => ApiService.listMatches(undefined, { signal }), []);

  return (
    <div className="page">
      <section className="hero">
        <h1>{t('title')}</h1>
        <p className="lead">{t('subtitle')}</p>
        <div className="hero-actions">
          <Link to="/contests" className="btn"><Trophy size={16} aria-hidden="true" /> {t('browseContests')}</Link>
          <Link to="/agents" className="btn btn-secondary"><Bot size={16} aria-hidden="true" /> {t('myAgents')}</Link>
        </div>
      </section>
      <div className="grid-2">
        <section className="card" aria-labelledby="home-contests">
          <h2 id="home-contests">{t('activeContests')}</h2>
          {contests.loading && <SkeletonRows />}
          {contests.error && <ErrorState error={contests.error} onRetry={contests.reload} />}
          {contests.data && (contests.data.items.length === 0 ? <p className="muted">{t('noContests')}</p> : (
            <ul className="list">
              {contests.data.items.slice(0, 5).map((c) => (
                <li key={c.id} className="list-row">
                  <Link to={`/contests/${c.id}`}>{c.name}</Link>
                  <StatusBadge state={c.state} />
                </li>
              ))}
            </ul>
          ))}
        </section>
        <section className="card" aria-labelledby="home-matches">
          <h2 id="home-matches"><Swords size={18} aria-hidden="true" /> {t('recentMatches')}</h2>
          {matches.loading && <SkeletonRows />}
          {matches.error && <ErrorState error={matches.error} onRetry={matches.reload} />}
          {matches.data && (matches.data.items.length === 0 ? <p className="muted">{t('noMatches')}</p> : (
            <ul className="list">
              {matches.data.items.slice(0, 6).map((m) => (
                <li key={m.id} className="list-row">
                  <Link to={`/matches/${m.id}`}>{m.slots.map((s) => s.display_name).join(' vs ')}</Link>
                  <span className="muted small">{formatDateTime(m.finished_at || m.created_at, i18n.language)}</span>
                  <StatusBadge state={m.state} />
                </li>
              ))}
            </ul>
          ))}
        </section>
      </div>
    </div>
  );
}
