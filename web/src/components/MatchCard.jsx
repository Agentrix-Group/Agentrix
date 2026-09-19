import React from 'react';
import { useTranslation } from 'react-i18next';
import { Play, ScanEye } from 'lucide-react';
import { formatNumber } from '../i18n/formatters.js';

export function MatchCard({ match, onWatchReplay, onTriggerRun }) {
  const { t, i18n } = useTranslation(['matches', 'common']);
  const currentLang = i18n.language?.startsWith('en') ? 'en' : 'es';

  const getBadgeClass = (status) => {
    switch (status) {
      case 'running': return 'badge-running';
      case 'finished': return 'badge-finished';
      case 'failed': return 'badge-failed';
      default: return 'badge-pending';
    }
  };

  const getStatusText = (status) => {
    return t(`matches:status.${status}`, { defaultValue: status });
  };

  return (
    <div className="card">
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        <strong>{t('matches:card.matchNumber', { id: match.id.substring(0, 8) })}</strong>
        <span className={`badge ${getBadgeClass(match.status)}`}>{getStatusText(match.status)}</span>
      </div>
      <p style={{ color: 'var(--text-secondary)', fontSize: '0.9rem', margin: '8px 0' }}>
        {t('matches:card.game', { game: match.game_id })} | {t('matches:card.seed', { seed: match.seed })}
      </p>

      {match.results && match.results.length > 0 && (
        <div style={{ margin: '12px 0', fontSize: '0.85rem' }}>
          <strong>{t('matches:card.results')}</strong>
          {match.results.map((r) => (
            <div key={r.id}>
              {t('matches:card.rankResult', {
                rank: r.rank,
                submission: r.submission_id,
                score: formatNumber(r.score, currentLang),
              })}
            </div>
          ))}
        </div>
      )}

      <div style={{ display: 'flex', gap: '8px', marginTop: '16px' }}>
        {match.status === 'finished' && match.replay_id && (
          <button className="btn" onClick={() => onWatchReplay(match.replay_id)}>
            <ScanEye size={17} aria-hidden="true" /> {t('matches:card.watchReplay')}
          </button>
        )}
        {match.status === 'pending' && typeof onTriggerRun === 'function' && (
          <button className="btn" onClick={() => onTriggerRun(match.id)}>
            <Play size={17} aria-hidden="true" /> {t('matches:card.runMatch')}
          </button>
        )}
      </div>
    </div>
  );
}
