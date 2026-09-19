import React, { useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { ApiService } from '../service/apiService.js';
import { MatchCard } from '../components/MatchCard.jsx';

export function HomePage({ onWatchReplay }) {
  const { t } = useTranslation(['home', 'common']);
  const [contests, setContests] = useState([]);
  const [recentMatches, setRecentMatches] = useState([]);

  useEffect(() => {
    ApiService.listContests().then(setContests).catch(() => {});
    ApiService.listMatches().then((matches) => setRecentMatches(matches.slice(0, 6))).catch(() => {});
  }, []);

  const getStatusText = (status) => {
    return t(`common:status.${status}`, { defaultValue: status });
  };

  return (
    <div>
      <h1>{t('home:title')}</h1>
      <p style={{ color: 'var(--text-secondary)' }}>
        {t('home:subtitle')}
      </p>

      <section style={{ marginTop: '32px' }}>
        <h2>{t('home:activeContests')}</h2>
        <div className="grid-cards">
          {contests.length > 0 ? (
            contests.map((c) => (
              <div key={c.id} className="card">
                <h3>{c.name}</h3>
                <p>{c.description}</p>
                <div style={{ fontSize: '0.85rem', color: 'var(--text-secondary)' }}>
                  {t('home:gameLabel', { game: c.game_id || c.category_id || '-' })} | {t('home:statusLabel')}{' '}
                  <span className="badge badge-finished">{getStatusText(c.status || c.state)}</span>
                </div>
              </div>
            ))
          ) : (
            <div className="card">{t('home:noContests')}</div>
          )}
        </div>
      </section>

      <section style={{ marginTop: '32px' }}>
        <h2>{t('home:recentMatches')}</h2>
        <div className="grid-cards">
          {recentMatches.length > 0 ? (
            recentMatches.map((m) => (
              <MatchCard key={m.id} match={m} onWatchReplay={onWatchReplay} />
            ))
          ) : (
            <div className="card">{t('home:noMatches')}</div>
          )}
        </div>
      </section>
    </div>
  );
}
