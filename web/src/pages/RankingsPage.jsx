import React, { useEffect, useState, useCallback, useMemo, useRef } from 'react';
import { useTranslation } from 'react-i18next';
import { AlertCircle, RefreshCw, Search, Radio } from 'lucide-react';
import { ApiService } from '../service/apiService.js';
import { formatNumber } from '../i18n/formatters.js';

export function RankingsPage({ initialContestId = '' }) {
  const { t, i18n } = useTranslation(['rankings', 'common']);
  const currentLang = i18n.language?.startsWith('en') ? 'en' : 'es';

  const [contests, setContests] = useState([]);
  const [selectedContestId, setSelectedContestId] = useState(initialContestId);
  const [rankings, setRankings] = useState([]);
  const [searchQuery, setSearchQuery] = useState('');
  const [sortBy, setSortBy] = useState('score'); // 'score' | 'wins' | 'matches' | 'winRate'
  const [status, setStatus] = useState('loading'); // 'loading' | 'error' | 'success'
  const [errorMessage, setErrorMessage] = useState('');
  const isMountedRef = useRef(true);

  useEffect(() => {
    isMountedRef.current = true;
    return () => {
      isMountedRef.current = false;
    };
  }, []);

  // Load contest options for dropdown
  useEffect(() => {
    ApiService.listContests()
      .then((data) => {
        if (!isMountedRef.current) return;
        setContests(data || []);
      })
      .catch(() => {
        if (!isMountedRef.current) return;
        setContests([]);
      });
  }, []);

  // Synchronize initialContestId if prop changes
  useEffect(() => {
    if (initialContestId !== undefined) {
      setSelectedContestId(initialContestId);
    }
  }, [initialContestId]);

  const loadRankings = useCallback((showSpinner = true) => {
    if (showSpinner) {
      setStatus('loading');
      setErrorMessage('');
    }
    return ApiService.listRankings(selectedContestId || undefined)
      .then((data) => {
        if (!isMountedRef.current) return;
        setRankings(data || []);
        setStatus('success');
      })
      .catch((err) => {
        if (!isMountedRef.current) return;
        if (showSpinner) {
          setErrorMessage(err?.message || t('common:messages.operationFailed'));
          setStatus('error');
        }
      });
  }, [selectedContestId, t]);

  useEffect(() => {
    loadRankings(true);
  }, [loadRankings]);

  // Determine if currently selected contest is running/live
  const selectedContest = contests.find((c) => c.id === selectedContestId);
  const isLiveActive = selectedContest && (selectedContest.state === 'in_progress' || selectedContest.state === 'live_final');

  // Background polling for live active contests
  useEffect(() => {
    if (!isLiveActive) return;

    const intervalId = setInterval(() => {
      loadRankings(false);
    }, 3500);

    return () => clearInterval(intervalId);
  }, [isLiveActive, loadRankings]);

  // Filter and sort rankings in-memory
  const processedRankings = useMemo(() => {
    const query = searchQuery.trim().toLowerCase();
    let filtered = rankings;

    if (query) {
      filtered = filtered.filter((r) =>
        (r.agent_id && r.agent_id.toLowerCase().includes(query)) ||
        (r.participant_id && r.participant_id.toLowerCase().includes(query))
      );
    }

    const withWinRate = filtered.map((r) => {
      const played = Number(r.matches_played) || 0;
      const wins = Number(r.wins) || 0;
      const winRate = played > 0 ? (wins / played) * 100 : 0;
      return { ...r, calculatedWinRate: winRate };
    });

    const sorted = [...withWinRate].sort((a, b) => {
      if (sortBy === 'wins') {
        return (b.wins || 0) - (a.wins || 0);
      }
      if (sortBy === 'matches') {
        return (b.matches_played || 0) - (a.matches_played || 0);
      }
      if (sortBy === 'winRate') {
        return b.calculatedWinRate - a.calculatedWinRate;
      }
      // Default: score
      return (b.score || 0) - (a.score || 0);
    });

    return sorted;
  }, [rankings, searchQuery, sortBy]);

  return (
    <div>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', flexWrap: 'wrap', gap: '12px', marginBottom: '16px' }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: '10px' }}>
          <h1>{t('rankings:title')}</h1>
          {isLiveActive && (
            <span
              className="badge badge-running"
              style={{ display: 'inline-flex', alignItems: 'center', gap: '5px', fontSize: '0.8rem' }}
            >
              <Radio size={14} color="currentColor" aria-hidden="true" />
              <span>{t('rankings:liveBadge')}</span>
            </span>
          )}
        </div>
        <button
          type="button"
          className="btn btn-secondary"
          onClick={() => loadRankings(true)}
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
            onClick={() => loadRankings(true)}
            style={{ display: 'inline-flex', alignItems: 'center', gap: '6px' }}
          >
            <RefreshCw size={15} aria-hidden="true" /> {t('common:buttons.retry')}
          </button>
        </div>
      )}

      {/* Filter and sorting controls */}
      <div
        className="card"
        style={{
          marginBottom: '20px',
          padding: '16px',
          display: 'flex',
          flexWrap: 'wrap',
          gap: '16px',
          alignItems: 'flex-end',
        }}
      >
        {/* Contest selector */}
        <div style={{ flex: '1 1 240px' }}>
          <label htmlFor="contest-filter-select" style={{ display: 'block', fontSize: '0.85rem', fontWeight: 600, marginBottom: '6px' }}>
            {t('rankings:contestFilter')}
          </label>
          <select
            id="contest-filter-select"
            value={selectedContestId}
            onChange={(e) => setSelectedContestId(e.target.value)}
            style={{ width: '100%', padding: '8px 12px' }}
          >
            <option value="">{t('rankings:allContests')}</option>
            {contests.map((c) => (
              <option key={c.id} value={c.id}>
                {c.name} ({t(`home:status.${c.state}`, { defaultValue: c.state })})
              </option>
            ))}
          </select>
        </div>

        {/* Search input */}
        <div style={{ flex: '1 1 240px' }}>
          <label htmlFor="ranking-search-input" style={{ display: 'block', fontSize: '0.85rem', fontWeight: 600, marginBottom: '6px' }}>
            {t('common:buttons.search', { defaultValue: 'Buscar' })}
          </label>
          <div style={{ position: 'relative' }}>
            <Search
              size={16}
              color="var(--text-secondary)"
              aria-hidden="true"
              style={{ position: 'absolute', left: '10px', top: '50%', transform: 'translateY(-50%)' }}
            />
            <input
              id="ranking-search-input"
              type="search"
              placeholder={t('rankings:searchPlaceholder')}
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              style={{ width: '100%', padding: '8px 12px 8px 34px' }}
            />
          </div>
        </div>

        {/* Sort by selector */}
        <div style={{ flex: '1 1 200px' }}>
          <label htmlFor="ranking-sort-select" style={{ display: 'block', fontSize: '0.85rem', fontWeight: 600, marginBottom: '6px' }}>
            {t('rankings:sortBy')}
          </label>
          <select
            id="ranking-sort-select"
            value={sortBy}
            onChange={(e) => setSortBy(e.target.value)}
            style={{ width: '100%', padding: '8px 12px' }}
          >
            <option value="score">{t('rankings:sortOptions.score')}</option>
            <option value="wins">{t('rankings:sortOptions.wins')}</option>
            <option value="matches">{t('rankings:sortOptions.matches')}</option>
            <option value="winRate">{t('rankings:sortOptions.winRate')}</option>
          </select>
        </div>
      </div>

      <table className="table">
        <thead>
          <tr>
            <th>{t('rankings:table.rank')}</th>
            <th>{t('rankings:table.agent')}</th>
            <th>{t('rankings:table.participant')}</th>
            <th>{t('rankings:table.score')}</th>
            <th>{t('rankings:table.matches')}</th>
            <th>{t('rankings:table.wdl')}</th>
            <th>{t('rankings:table.winRate')}</th>
          </tr>
        </thead>
        <tbody>
          {status === 'loading' ? (
            <tr>
              <td colSpan="7" style={{ textAlign: 'center', color: 'var(--text-secondary)', padding: '24px' }}>
                {t('common:buttons.loading')}
              </td>
            </tr>
          ) : processedRankings.length > 0 ? (
            processedRankings.map((r, index) => (
              <tr key={r.id || `${r.agent_id}-${index}`}>
                <td><strong>#{r.rank !== undefined ? r.rank : index + 1}</strong></td>
                <td>{r.agent_id}</td>
                <td>{r.participant_id}</td>
                <td><strong>{formatNumber(r.score, currentLang)}</strong></td>
                <td>{formatNumber(r.matches_played, currentLang)}</td>
                <td>
                  {formatNumber(r.wins, currentLang)} / {formatNumber(r.draws, currentLang)} / {formatNumber(r.losses, currentLang)}
                </td>
                <td>{formatNumber(Math.round(r.calculatedWinRate), currentLang)}%</td>
              </tr>
            ))
          ) : rankings.length > 0 ? (
            <tr>
              <td colSpan="7" style={{ textAlign: 'center', color: 'var(--text-secondary)', padding: '24px' }}>
                {t('rankings:emptySearch')}
              </td>
            </tr>
          ) : (
            <tr>
              <td colSpan="7" style={{ textAlign: 'center', color: 'var(--text-secondary)', padding: '24px' }}>
                {t('rankings:empty')}
              </td>
            </tr>
          )}
        </tbody>
      </table>
    </div>
  );
}
