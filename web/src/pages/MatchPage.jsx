import React, { useRef, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Play, Ban, Film } from 'lucide-react';
import { ApiService } from '../service/apiService.js';
import { newIdempotencyKey } from '../api/client.js';
import { useResource, usePolling } from '../hooks/hooks.js';
import { useSession } from '../auth/SessionContext.jsx';
import { Link } from '../router/Router.jsx';
import { LoadingState, ErrorState, useErrorMessage } from '../components/States.jsx';
import { StatusBadge } from '../components/StatusBadge.jsx';
import { RunStepper } from '../components/RunStepper.jsx';
import { ReasonDialog } from '../components/ReasonDialog.jsx';
import { useToast } from '../components/Toast.jsx';
import { formatDateTime } from '../i18n/formatters.js';

export default function MatchPage({ matchId }) {
  const { t, i18n } = useTranslation('matches');
  const { can } = useSession();
  const toast = useToast();
  const describe = useErrorMessage();
  const match = useResource((signal) => ApiService.getMatch(matchId, { signal }), [matchId]);
  const [busy, setBusy] = useState(false);
  const [cancelling, setCancelling] = useState(false);
  // One key per user intent: a retried click reuses it, a new intent gets a new key.
  const intentKey = useRef(null);

  const m = match.data;
  const inFlight = m && (m.state === 'queued' || m.state === 'running');
  const waitingReplay = m && m.state === 'finished' && (!m.replay || !m.replay.available);
  usePolling(match.reload, 2000, Boolean(inFlight || waitingReplay));

  if (match.loading && !m) return <LoadingState />;
  if (match.error && !m) return <ErrorState error={match.error} onRetry={match.reload} />;

  const run = async () => {
    setBusy(true);
    if (!intentKey.current) intentKey.current = newIdempotencyKey();
    try {
      const ticket = await ApiService.scheduleRun(m.id, intentKey.current);
      intentKey.current = null;
      toast.success(t('queued', { attempt: ticket.attempt }));
      match.reload();
    } catch (err) {
      if (err.code === 'run_in_progress') {
        intentKey.current = null;
        match.reload();
      }
      toast.error(describe(err));
    } finally {
      setBusy(false);
    }
  };

  return (
    <div className="page">
      <div className="page-header">
        <div>
          <p className="eyebrow">{t(`modes.${m.mode}`)} · {m.game_id}{m.contest_id && <> · <Link to={`/contests/${m.contest_id}`}>{t('contest')}</Link></>}</p>
          <h1>{m.slots.map((s) => s.display_name).join(' vs ')}</h1>
        </div>
        <div className="actions">
          {can('matches:run') && (m.state === 'scheduled' || m.state === 'failed') && (
            <button type="button" className="btn" onClick={run} disabled={busy}>
              <Play size={16} aria-hidden="true" /> {m.state === 'failed' ? t('retry') : t('run')}
            </button>
          )}
          {can('matches:cancel') && ['scheduled', 'queued', 'failed'].includes(m.state) && (
            <button type="button" className="btn btn-secondary" onClick={() => setCancelling(true)}><Ban size={16} aria-hidden="true" /> {t('cancel')}</button>
          )}
          {m.replay?.available && (
            <Link to={`/replays/${m.replay.id}`} className="btn btn-secondary"><Film size={16} aria-hidden="true" /> {t('watchReplay')}</Link>
          )}
        </div>
      </div>

      <section className="card" aria-labelledby="progress" aria-live="polite">
        <h2 id="progress">{t('progress')}</h2>
        <RunStepper state={m.state} />
        {m.state === 'finished' && m.replay && !m.replay.available && <p className="muted small">{t('replayPublishing')}</p>}
      </section>

      <div className="grid-2">
        <section className="card" aria-labelledby="slots">
          <h2 id="slots">{t('slots')}</h2>
          <ol className="list">
            {m.slots.map((s) => (
              <li key={s.id} className="list-row"><span className="badge badge-neutral">#{s.slot_index + 1}</span> {s.display_name}</li>
            ))}
          </ol>
          <p className="muted small">{t('seed')}: <code>{m.seed}</code></p>
        </section>
        <section className="card" aria-labelledby="results">
          <h2 id="results">{t('results')}</h2>
          {m.results.length === 0 ? <p className="muted">{t('noResults')}</p> : (
            <table>
              <thead><tr><th>{t('rank')}</th><th>{t('participant')}</th><th>{t('score')}</th><th>{t('outcome')}</th></tr></thead>
              <tbody>
                {[...m.results].sort((a, b) => a.rank - b.rank).map((r) => (
                  <tr key={r.slot_id}><td>{r.rank}</td><td>{r.display_name}</td><td>{r.score}</td><td><StatusBadge state={r.outcome} /></td></tr>
                ))}
              </tbody>
            </table>
          )}
        </section>
      </div>

      <section className="card" aria-labelledby="runs">
        <h2 id="runs">{t('runs')}</h2>
        {m.runs.length === 0 ? <p className="muted">{t('noRuns')}</p> : (
          <div className="table-wrap">
            <table>
              <thead><tr><th>{t('attempt')}</th><th>{t('state')}</th><th>{t('engine')}</th><th>{t('spec')}</th><th>{t('finished')}</th><th>{t('error')}</th></tr></thead>
              <tbody>
                {m.runs.map((r) => (
                  <tr key={r.id} className={r.id === m.committed_run_id ? 'row-highlight' : ''}>
                    <td>{r.attempt}{r.id === m.committed_run_id && <span className="badge badge-success"> {t('committed')}</span>}</td>
                    <td><StatusBadge state={r.state} /></td>
                    <td className="small"><code title={r.engine_sha256}>{r.engine_version} · {r.engine_sha256.slice(0, 10)}</code></td>
                    <td className="small"><code title={r.execution_spec_hash}>{r.execution_spec_hash.slice(0, 12)}</code> · {r.tick_rate.numerator}/{r.tick_rate.denominator} Hz</td>
                    <td className="small muted">{formatDateTime(r.finished_at, i18n.language)}</td>
                    <td className="small">{r.error_class && <><StatusBadge state={r.error_class} /> {r.error_message}</>}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </section>

      {cancelling && (
        <ReasonDialog title={t('cancel')} confirmLabel={t('cancel')} danger onClose={() => setCancelling(false)}
          onConfirm={async (reason) => {
            try {
              await ApiService.cancelMatch(m.id, reason);
              setCancelling(false);
              match.reload();
            } catch (err) {
              toast.error(describe(err));
            }
          }} />
      )}
    </div>
  );
}
