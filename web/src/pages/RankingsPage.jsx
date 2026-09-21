import React, { useRef, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { RefreshCw, Camera } from 'lucide-react';
import { ApiService } from '../service/apiService.js';
import { useResource, usePolling, useFlip } from '../hooks/hooks.js';
import { useSession } from '../auth/SessionContext.jsx';
import { useRouter } from '../router/Router.jsx';
import { LoadingState, ErrorState, EmptyState, useErrorMessage } from '../components/States.jsx';
import { useToast } from '../components/Toast.jsx';
import { formatDateTime } from '../i18n/formatters.js';

function RankingTable({ rows }) {
  const { t } = useTranslation('rankings');
  const bodyRef = useRef(null);
  useFlip(bodyRef, rows.map((r) => `${r.entry_id}:${r.rank}`).join('|'));
  return (
    <div className="table-wrap">
      <table className="ranking-table">
        <thead>
          <tr><th>{t('rank')}</th><th>{t('agent')}</th><th>{t('owner')}</th><th>{t('points')}</th><th>{t('played')}</th>
            <th>{t('wins')}</th><th>{t('draws')}</th><th>{t('losses')}</th><th>{t('disqualifications')}</th><th>{t('scoreDiff')}</th></tr>
        </thead>
        <tbody ref={bodyRef}>
          {rows.map((r) => (
            <tr key={r.entry_id} data-flip-key={r.entry_id}>
              <td><span className={`rank-pill rank-${Math.min(r.rank, 4)}`}>{r.rank}</span></td>
              <td>{r.agent_name}</td>
              <td className="muted">{r.username}</td>
              <td><strong>{r.points}</strong></td>
              <td>{r.matches_played}</td>
              <td>{r.wins}</td>
              <td>{r.draws}</td>
              <td>{r.losses}</td>
              <td>{r.disqualifications}</td>
              <td>{r.score_for - r.score_against}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

export default function RankingsPage({ contestId }) {
  const { t, i18n } = useTranslation('rankings');
  const { can } = useSession();
  const { navigate } = useRouter();
  const toast = useToast();
  const describe = useErrorMessage();
  const contests = useResource((signal) => ApiService.listContests({ signal }), []);
  const selected = contestId || contests.data?.items.find((c) => c.state !== 'draft')?.id || '';
  const rankings = useResource((signal) => (selected ? ApiService.getRankings(selected, { signal }) : Promise.resolve(null)), [selected]);
  const snapshots = useResource((signal) => (selected ? ApiService.listSnapshots(selected, { signal }) : Promise.resolve(null)), [selected]);
  const [snapshotVersion, setSnapshotVersion] = useState('');
  usePolling(rankings.reload, 3000, Boolean(rankings.data?.stale));

  const publish = async () => {
    try {
      const snap = await ApiService.publishSnapshot(selected);
      toast.success(t('published', { version: snap.version }));
      snapshots.reload();
    } catch (err) {
      toast.error(describe(err));
    }
  };
  const recalculate = async () => {
    try {
      rankings.setData(await ApiService.recalculateRankings(selected));
    } catch (err) {
      toast.error(describe(err));
    }
  };
  const snapshot = snapshots.data?.items.find((s) => String(s.version) === snapshotVersion);
  const rows = snapshot ? snapshot.rankings : rankings.data?.rankings || [];

  return (
    <div className="page">
      <div className="page-header">
        <h1>{t('title')}</h1>
        {contests.data && (
          <label className="field inline">
            <span>{t('contest')}</span>
            <select value={selected} onChange={(e) => navigate(`/rankings?contest=${e.target.value}`)}>
              {contests.data.items.map((c) => <option key={c.id} value={c.id}>{c.name}</option>)}
            </select>
          </label>
        )}
      </div>
      {contests.loading && <LoadingState />}
      {contests.error && <ErrorState error={contests.error} onRetry={contests.reload} />}
      {contests.data && !selected && <EmptyState title={t('noContest')} />}
      {selected && (
        <section className="card" aria-labelledby="ranking-title">
          <div className="card-row">
            <h2 id="ranking-title">{snapshot ? t('snapshotTitle', { version: snapshot.version }) : t('current')}</h2>
            <div className="actions">
              {snapshots.data && snapshots.data.items.length > 0 && (
                <label className="field inline">
                  <span>{t('view')}</span>
                  <select value={snapshotVersion} onChange={(e) => setSnapshotVersion(e.target.value)}>
                    <option value="">{t('live')}</option>
                    {snapshots.data.items.map((s) => <option key={s.version} value={s.version}>v{s.version} · {formatDateTime(s.published_at, i18n.language)}</option>)}
                  </select>
                </label>
              )}
              {can('rankings:publish') && (
                <>
                  <button type="button" className="btn btn-secondary" onClick={recalculate}><RefreshCw size={16} aria-hidden="true" /> {t('recalculate')}</button>
                  <button type="button" className="btn" onClick={publish}><Camera size={16} aria-hidden="true" /> {t('publish')}</button>
                </>
              )}
            </div>
          </div>
          {rankings.loading && !rankings.data && <LoadingState />}
          {rankings.error && <ErrorState error={rankings.error} onRetry={rankings.reload} />}
          {rankings.data && !snapshot && rankings.data.stale && <p className="banner banner-info" role="status">{t('stale')}</p>}
          {rows.length === 0 && rankings.data ? <EmptyState title={t('empty')} /> : <RankingTable rows={rows} />}
          {rankings.data && !snapshot && (
            <p className="muted small">{t('provenance', { count: rankings.data.applied_runs_count })} <code>{rankings.data.applied_runs_digest.slice(0, 16)}</code></p>
          )}
          {snapshot && <p className="muted small">{t('snapshotProvenance')} <code>{snapshot.rankings_sha256.slice(0, 16)}</code></p>}
        </section>
      )}
    </div>
  );
}
