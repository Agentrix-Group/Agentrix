import React, { useEffect, useState, useCallback, useRef } from 'react';
import { useTranslation } from 'react-i18next';
import { AlertCircle, RefreshCw, Plus, CheckCircle } from 'lucide-react';
import { ApiService } from '../service/apiService.js';
import { MatchCard } from '../components/MatchCard.jsx';
import { CreateMatchModal } from '../components/CreateMatchModal.jsx';
import { canRunMatch } from '../auth/session.js';

export function MatchesPage({ onWatchReplay, currentUser, canRun }) {
  const { t } = useTranslation(['matches', 'common']);
  const [matches, setMatches] = useState([]);
  const [status, setStatus] = useState('loading'); // 'loading' | 'error' | 'success'
  const [errorMessage, setErrorMessage] = useState('');
  const [actionError, setActionError] = useState('');
  const [successMessage, setSuccessMessage] = useState('');
  const [filter, setFilter] = useState('all');
  const [isModalOpen, setIsModalOpen] = useState(false);
  const [executingMatchIds, setExecutingMatchIds] = useState(() => new Set());
  const isMountedRef = useRef(true);

  const canExecute = canRun !== undefined ? canRun : canRunMatch(currentUser);

  useEffect(() => {
    isMountedRef.current = true;
    return () => {
      isMountedRef.current = false;
    };
  }, []);

  const loadMatches = useCallback((showSpinner = true) => {
    if (showSpinner) {
      setStatus('loading');
      setErrorMessage('');
    }
    return ApiService.listMatches()
      .then((data) => {
        if (!isMountedRef.current) return;
        setMatches(data || []);
        setStatus('success');
      })
      .catch((err) => {
        if (!isMountedRef.current) return;
        if (showSpinner) {
          setErrorMessage(err?.message || t('common:messages.operationFailed'));
          setStatus('error');
        }
      });
  }, [t]);

  useEffect(() => {
    loadMatches(true);
  }, [loadMatches]);

  // Live background polling when any match is in 'running' status
  useEffect(() => {
    const hasRunning = matches.some((m) => m.status === 'running');
    if (!hasRunning) return;

    const intervalId = setInterval(() => {
      loadMatches(false);
    }, 2500);

    return () => clearInterval(intervalId);
  }, [matches, loadMatches]);

  const handleTriggerRun = async (matchId) => {
    setActionError('');
    setSuccessMessage('');
    setExecutingMatchIds((prev) => new Set(prev).add(matchId));
    try {
      await ApiService.runMatch(matchId);
      await loadMatches(false);
    } catch (e) {
      if (isMountedRef.current) {
        setActionError(t('matches:messages.runError', { error: e.message || e }));
      }
    } finally {
      if (isMountedRef.current) {
        setExecutingMatchIds((prev) => {
          const next = new Set(prev);
          next.delete(matchId);
          return next;
        });
      }
    }
  };

  const handleMatchCreated = () => {
    setSuccessMessage(t('matches:messages.createSuccess'));
    loadMatches(false);
  };

  const filteredMatches = matches.filter((m) => {
    if (filter === 'all') return true;
    return m.status === filter;
  });

  return (
    <div>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', flexWrap: 'wrap', gap: '12px' }}>
        <h1>{t('matches:title')}</h1>
        <div style={{ display: 'flex', gap: '8px', alignItems: 'center' }}>
          {canExecute && (
            <button
              type="button"
              className="btn"
              onClick={() => setIsModalOpen(true)}
              style={{ display: 'inline-flex', alignItems: 'center', gap: '6px' }}
            >
              <Plus size={17} aria-hidden="true" /> {t('matches:createButton')}
            </button>
          )}
          <button
            type="button"
            className="btn btn-secondary"
            onClick={() => loadMatches(true)}
            style={{ display: 'inline-flex', alignItems: 'center', gap: '6px' }}
          >
            <RefreshCw size={17} aria-hidden="true" /> {t('matches:refresh')}
          </button>
        </div>
      </div>

      {successMessage && (
        <div
          role="status"
          aria-live="polite"
          className="card"
          style={{
            margin: '16px 0',
            borderLeft: '4px solid var(--success, #22c55e)',
            background: 'var(--success-bg, rgba(34, 197, 94, 0.1))',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'space-between',
            gap: '12px',
          }}
        >
          <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
            <CheckCircle size={20} color="var(--success, #22c55e)" aria-hidden="true" />
            <span>{successMessage}</span>
          </div>
          <button type="button" className="btn btn-secondary" onClick={() => setSuccessMessage('')}>
            {t('common:buttons.close')}
          </button>
        </div>
      )}

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
            onClick={() => loadMatches(true)}
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
                canRun={canExecute}
                isExecuting={executingMatchIds.has(m.id)}
              />
            ))
          ) : (
            <div className="card">{t('matches:empty')}</div>
          )}
        </div>
      )}

      <CreateMatchModal
        isOpen={isModalOpen}
        onClose={() => setIsModalOpen(false)}
        onMatchCreated={handleMatchCreated}
        currentUser={currentUser}
      />
    </div>
  );
}
