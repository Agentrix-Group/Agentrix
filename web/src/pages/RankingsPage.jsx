import React, { useEffect, useState, useCallback } from 'react';
import { useTranslation } from 'react-i18next';
import { AlertCircle, RefreshCw } from 'lucide-react';
import { ApiService } from '../service/apiService.js';
import { formatNumber } from '../i18n/formatters.js';

export function RankingsPage() {
  const { t, i18n } = useTranslation(['rankings', 'common']);
  const currentLang = i18n.language?.startsWith('en') ? 'en' : 'es';
  const [rankings, setRankings] = useState([]);
  const [status, setStatus] = useState('loading'); // 'loading' | 'error' | 'success'
  const [errorMessage, setErrorMessage] = useState('');

  const loadRankings = useCallback(() => {
    setStatus('loading');
    setErrorMessage('');
    ApiService.listRankings()
      .then((data) => {
        setRankings(data || []);
        setStatus('success');
      })
      .catch((err) => {
        setErrorMessage(err?.message || t('common:messages.operationFailed'));
        setStatus('error');
      });
  }, [t]);

  useEffect(() => {
    loadRankings();
  }, [loadRankings]);

  return (
    <div>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '16px' }}>
        <h1>{t('rankings:title')}</h1>
        <button
          type="button"
          className="btn btn-secondary"
          onClick={loadRankings}
          aria-label={t('common:buttons.refresh')}
          style={{ display: 'inline-flex', alignItems: 'center', gap: '6px' }}
        >
          <RefreshCw size={16} aria-hidden="true" /> {t('common:buttons.refresh')}
        </button>
      </div>

      {status === 'error' && (
        <div
          role="alert"
          aria-live="assertive"
          className="card"
          style={{
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'space-between',
            gap: '12px',
            marginBottom: '16px',
            borderLeft: '4px solid var(--danger, #ef4444)',
            background: 'var(--danger-bg, rgba(239, 68, 68, 0.1))',
          }}
        >
          <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
            <AlertCircle size={20} color="var(--danger, #ef4444)" aria-hidden="true" />
            <span>{errorMessage || t('common:messages.operationFailed')}</span>
          </div>
          <button
            type="button"
            className="btn"
            onClick={loadRankings}
            style={{ display: 'inline-flex', alignItems: 'center', gap: '6px' }}
          >
            <RefreshCw size={15} aria-hidden="true" /> {t('common:buttons.retry')}
          </button>
        </div>
      )}

      <table className="table">
        <thead>
          <tr>
            <th>{t('rankings:table.rank')}</th>
            <th>{t('rankings:table.agent')}</th>
            <th>{t('rankings:table.participant')}</th>
            <th>{t('rankings:table.score')}</th>
            <th>{t('rankings:table.matches')}</th>
            <th>{t('rankings:table.wdl')}</th>
          </tr>
        </thead>
        <tbody>
          {status === 'loading' ? (
            <tr>
              <td colSpan="6" style={{ textAlign: 'center', color: 'var(--text-secondary)', padding: '24px' }}>
                {t('common:buttons.loading')}
              </td>
            </tr>
          ) : rankings.length > 0 ? (
            rankings.map((r) => (
              <tr key={r.id}>
                <td><strong>#{r.rank}</strong></td>
                <td>{r.agent_id}</td>
                <td>{r.participant_id}</td>
                <td><strong>{formatNumber(r.score, currentLang)}</strong></td>
                <td>{formatNumber(r.matches_played, currentLang)}</td>
                <td>
                  {formatNumber(r.wins, currentLang)} / {formatNumber(r.draws, currentLang)} / {formatNumber(r.losses, currentLang)}
                </td>
              </tr>
            ))
          ) : (
            <tr>
              <td colSpan="6" style={{ textAlign: 'center', color: 'var(--text-secondary)', padding: '24px' }}>
                {t('rankings:empty')}
              </td>
            </tr>
          )}
        </tbody>
      </table>
    </div>
  );
}
