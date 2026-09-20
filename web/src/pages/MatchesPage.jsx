import React, { useEffect, useState, useCallback } from 'react';
import { useTranslation } from 'react-i18next';
import { AlertCircle, RefreshCw } from 'lucide-react';
import { ApiService } from '../service/apiService.js';
import { MatchCard } from '../components/MatchCard.jsx';
import { canRunMatch } from '../auth/session.js';

export function MatchesPage({ onWatchReplay, currentUser, canRun }) {
  const { t } = useTranslation(['matches', 'common']);
  const [matches, setMatches] = useState([]);
  const [status, setStatus] = useState('loading'); // 'loading' | 'error' | 'success'
  const [errorMessage, setErrorMessage] = useState('');
  const [actionError, setActionError] = useState('');
  const [filter, setFilter] = useState('all');

  const loadMatches = useCallback(() => {
    setStatus('loading');
    setErrorMessage('');
    ApiService.listMatches()
      .then((data) => {
        setMatches(data || []);
        setStatus('success');
      })
      .catch((err) => {
        setErrorMessage(err?.message || t('common:messages.operationFailed'));
        setStatus('error');
      });
  }, [t]);

  useEffect(() => {
    loadMatches();
  }, [loadMatches]);

  const handleTriggerRun = async (matchId) => {
    setActionError('');
    try {
      await ApiService.runMatch(matchId);
      loadMatches();
    } catch (e) {
      setActionError(t('matches:messages.runError', { error: e.message || e }));
    }
  };

  const filteredMatches = matches.filter((m) => {
    if (filter === 'all') return true;
    return m.status === filter;
  });

  return (
    <div>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        <h1>{t('matches:title')}</h1>
        <button
          type="button"
          className="btn btn-secondary"
          onClick={loadMatches}
          style={{ display: 'inline-flex', alignItems: 'center', gap: '6px' }}
        >
          <RefreshCw size={17} aria-hidden="true" /> {t('matches:refresh')}
        </button>
      </div>

      {actionError && (
        <div
          role="alert"
          aria-live="assertive"
          className="card"
          style={{
            margin: '16px 0',
            borderLeft: '4px solid var(--danger, #ef4444)',
            background: 'var(--danger-bg, rgba(239, 68, 68, 0.1))',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'space-between',
            gap: '12px',
          }}
        >
          <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
            <AlertCircle size={20} color="var(--danger, #ef4444)" aria-hidden="true" />
            <span>{actionError}</span>
          </div>
          <button type="button" className="btn btn-secondary" onClick={() => setActionError('')}>
            {t('common:buttons.close')}
          </button>
        </div>
      )}

      {status === 'error' && (
        <div
          role="alert"
          aria-live="assertive"
          className="card"
          style={{
            margin: '16px 0',
            borderLeft: '4px solid var(--danger, #ef4444)',
            background: 'var(--danger-bg, rgba(239, 68, 68, 0.1))',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'space-between',
            gap: '12px',
          }}
        >
          <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
            <AlertCircle size={20} color="var(--danger, #ef4444)" aria-hidden="true" />
            <span>{errorMessage || t('common:messages.operationFailed')}</span>
          </div>
          <button
            type="button"
            className="btn"
            onClick={loadMatches}
            style={{ display: 'inline-flex', alignItems: 'center', gap: '6px' }}
          >
            <RefreshCw size={15} aria-hidden="true" /> {t('common:buttons.retry')}
          </button>
        </div>
      )}

      <div style={{ display: 'flex', gap: '8px', flexWrap: 'wrap', margin: '16px 0 24px' }}>
        {['all', 'pending', 'running', 'finished'].map((statusKey) => {
          const count = statusKey === 'all' ? matches.length : matches.filter((m) => m.status === statusKey).length;
          return (
            <button
              key={statusKey}
              type="button"
              className={`btn ${filter === statusKey ? '' : 'btn-secondary'}`}
              onClick={() => setFilter(statusKey)}
            >
              {t(`matches:filter.${statusKey}`)} ({count})
            </button>
          );
        })}
      </div>

      {status === 'loading' ? (
        <p>{t('matches:loading')}</p>
      ) : status === 'error' ? null : (
        <div className="grid-cards">
          {filteredMatches.length > 0 ? (
            filteredMatches.map((m) => (
              <MatchCard
                key={m.id}
                match={m}
                onWatchReplay={onWatchReplay}
                onTriggerRun={handleTriggerRun}
                canRun={canRun !== undefined ? canRun : canRunMatch(currentUser)}
              />
            ))
          ) : (
            <div className="card">{t('matches:empty')}</div>
          )}
        </div>
      )}
    </div>
  );
}
