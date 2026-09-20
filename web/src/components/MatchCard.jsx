import React from 'react';
import { useTranslation } from 'react-i18next';
import { Play, ScanEye, Loader2, Radio } from 'lucide-react';
import { formatNumber } from '../i18n/formatters.js';

export function MatchCard({ match, onWatchReplay, onTriggerRun, canRun = false, isExecuting = false }) {
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

  const isSimulating = match.status === 'running';

  return (
    <div className="card">
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        <strong>{t('matches:card.matchNumber', { id: match.id.substring(0, 8) })}</strong>
        <span className={`badge ${getBadgeClass(match.status)}`} style={{ display: 'inline-flex', alignItems: 'center', gap: '6px' }}>
          {isSimulating && (
            <span
              className="pulse-dot"
              aria-hidden="true"
              style={{
                width: 8,
                height: 8,
                borderRadius: '50%',
                backgroundColor: 'currentColor',
                display: 'inline-block',
              }}
            />
          )}
          {getStatusText(match.status)}
        </span>
      </div>
      <p style={{ color: 'var(--text-secondary)', fontSize: '0.9rem', margin: '8px 0' }}>
        {t('matches:card.game', { game: match.game_id })} | {t('matches:card.seed', { seed: match.seed })}
      </p>

      {isSimulating && (
        <div
          style={{
            margin: '10px 0',
            padding: '8px 12px',
            background: 'rgba(59, 130, 246, 0.08)',
            borderLeft: '3px solid var(--accent, #3b82f6)',
            borderRadius: '4px',
            display: 'flex',
            alignItems: 'center',
            gap: '8px',
            fontSize: '0.84rem',
          }}
        >
          <Radio size={16} color="var(--accent, #3b82f6)" aria-hidden="true" />
          <span>{t('matches:runningState.liveDescription')}</span>
        </div>
      )}

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
        {canRun && match.status === 'pending' && typeof onTriggerRun === 'function' && (
          <button
            className="btn"
            onClick={() => onTriggerRun(match.id)}
            disabled={isExecuting}
            style={{ display: 'inline-flex', alignItems: 'center', gap: '6px' }}
          >
            {isExecuting ? (
              <>
                <Loader2 size={16} className="spin" style={{ animation: 'spin 0.8s linear infinite' }} aria-hidden="true" />
                <span>{t('matches:card.runningMatch')}</span>
              </>
            ) : (
              <>
                <Play size={17} aria-hidden="true" />
                <span>{t('matches:card.runMatch')}</span>
              </>
            )}
          </button>
        )}
      </div>
    </div>
  );
}
