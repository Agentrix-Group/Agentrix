import React, { useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { UserPlus, Swords, ListOrdered } from 'lucide-react';
import { ApiService } from '../service/apiService.js';
import { useResource } from '../hooks/hooks.js';
import { useSession } from '../auth/SessionContext.jsx';
import { Link, useRouter } from '../router/Router.jsx';
import { LoadingState, ErrorState, EmptyState, useErrorMessage } from '../components/States.jsx';
import { StatusBadge } from '../components/StatusBadge.jsx';
import { Modal } from '../components/Modal.jsx';
import { ReasonDialog } from '../components/ReasonDialog.jsx';
import { useToast } from '../components/Toast.jsx';

/** Enrollment always locks an explicit, ready submission of the agent. */
function EnrollModal({ contest, onClose, onEnrolled }) {
  const { t } = useTranslation('contests');
  const describe = useErrorMessage();
  const agents = useResource((signal) => ApiService.listAgents({ signal }), []);
  const eligibleAgents = (agents.data?.items || []).filter((a) => a.game_id === contest.game_id && a.status === 'active');
  const [agentId, setAgentId] = useState('');
  const submissions = useResource((signal) => (agentId ? ApiService.listSubmissions(agentId, { signal }) : Promise.resolve(null)), [agentId]);
  const ready = (submissions.data?.items || []).filter((s) => s.status === 'ready').sort((a, b) => b.version - a.version);
  const [submissionId, setSubmissionId] = useState('');
  const [error, setError] = useState(null);
  const [busy, setBusy] = useState(false);

  const submit = async (e) => {
    e.preventDefault();
    setBusy(true);
    setError(null);
    try {
      onEnrolled(await ApiService.enroll(contest.id, agentId, submissionId));
    } catch (err) {
      setError(describe(err));
    } finally {
      setBusy(false);
    }
  };
  return (
    <Modal title={t('enrollTitle', { name: contest.name })} onClose={onClose}>
      {agents.loading ? <LoadingState /> : (
        <form className="form" onSubmit={submit}>
          {eligibleAgents.length === 0 && <p className="muted">{t('noEligibleAgents')}</p>}
          <label className="field"><span>{t('agent')}</span>
            <select required value={agentId} onChange={(e) => { setAgentId(e.target.value); setSubmissionId(''); }}>
              <option value="">{t('chooseAgent')}</option>
              {eligibleAgents.map((a) => <option key={a.id} value={a.id}>{a.name}</option>)}
            </select>
          </label>
          <label className="field"><span>{t('submission')}</span>
            <select required value={submissionId} onChange={(e) => setSubmissionId(e.target.value)} disabled={!agentId}>
              <option value="">{t('chooseSubmission')}</option>
              {ready.map((s) => <option key={s.id} value={s.id}>v{s.version} · {s.artifact_sha256.slice(0, 12)}</option>)}
            </select>
          </label>
          {agentId && submissions.data && ready.length === 0 && <p className="muted small">{t('noReadySubmissions')}</p>}
          <p className="muted small">{t('lockExplanation')}</p>
          {error && <p className="form-error" role="alert">{error}</p>}
          <div className="form-actions">
            <button type="button" className="btn btn-secondary" onClick={onClose}>{t('common:buttons.cancel')}</button>
            <button type="submit" className="btn" disabled={busy || !agentId || !submissionId}>{t('enroll')}</button>
          </div>
        </form>
      )}
    </Modal>
  );
}

/** Competitive matches are built only from eligible (enrolled) entries,
 * without duplicates and with the game's player count. */
function CreateMatchModal({ contest, entries, onClose, onCreated }) {
  const { t } = useTranslation('contests');
  const describe = useErrorMessage();
  const eligible = entries.filter((e) => e.status === 'enrolled');
  const [selected, setSelected] = useState([]);
  const [seed, setSeed] = useState('');
  const [error, setError] = useState(null);
  const [busy, setBusy] = useState(false);
  const toggle = (id) => setSelected((list) => (list.includes(id) ? list.filter((x) => x !== id) : [...list, id]));
  const submit = async (e) => {
    e.preventDefault();
    setBusy(true);
    setError(null);
    try {
      const body = { contest_id: contest.id, mode: 'competitive', entry_ids: selected };
      if (seed !== '') body.seed = Number(seed);
      onCreated(await ApiService.createMatch(body));
    } catch (err) {
      setError(describe(err));
    } finally {
      setBusy(false);
    }
  };
  return (
    <Modal title={t('createMatch')} onClose={onClose}>
      <form className="form" onSubmit={submit}>
        <fieldset className="field">
          <legend>{t('rosterOrder')}</legend>
          {eligible.map((entry) => {
            const position = selected.indexOf(entry.id);
            return (
              <label key={entry.id} className="checkbox-row">
                <input type="checkbox" checked={position >= 0} onChange={() => toggle(entry.id)} />
                <span>{entry.agent_name} ({entry.username}) · v{entry.submission_version}</span>
                {position >= 0 && <span className="badge badge-info">{t('slot', { n: position + 1 })}</span>}
              </label>
            );
          })}
        </fieldset>
        <label className="field"><span>{t('seedOptional')}</span>
          <input type="number" min="0" step="1" value={seed} onChange={(e) => setSeed(e.target.value)} />
        </label>
        {error && <p className="form-error" role="alert">{error}</p>}
        <div className="form-actions">
          <button type="button" className="btn btn-secondary" onClick={onClose}>{t('common:buttons.cancel')}</button>
          <button type="submit" className="btn" disabled={busy || selected.length < 2}>{t('createMatch')}</button>
        </div>
      </form>
    </Modal>
  );
}

export default function ContestPage({ contestId }) {
  const { t } = useTranslation('contests');
  const { can, currentUser } = useSession();
  const { navigate } = useRouter();
  const toast = useToast();
  const describe = useErrorMessage();
  const contest = useResource((signal) => ApiService.getContest(contestId, { signal }), [contestId]);
  const entries = useResource((signal) => ApiService.listEntries(contestId, { signal }), [contestId]);
  const [dialog, setDialog] = useState(null);

  const myEntries = useMemo(() => (entries.data?.items || []).filter((e) => e.user_id === currentUser?.id), [entries.data, currentUser]);
  if (contest.loading) return <LoadingState />;
  if (contest.error) return <ErrorState error={contest.error} onRetry={contest.reload} />;
  const c = contest.data;

  const transition = async (state) => {
    try {
      contest.setData(await ApiService.transitionContest(c.id, state));
      toast.success(t('transitioned', { state: t(`common:states.${state}`) }));
    } catch (err) {
      toast.error(describe(err));
    }
  };
  const entryAction = (entry, kind) => setDialog({ kind, entry });

  return (
    <div className="page">
      <div className="page-header">
        <div>
          <p className="eyebrow">{c.game_id}</p>
          <h1>{c.name} <StatusBadge state={c.state} /></h1>
        </div>
        <div className="actions">
          <Link to={`/rankings?contest=${c.id}`} className="btn btn-secondary"><ListOrdered size={16} aria-hidden="true" /> {t('rankings')}</Link>
          <Link to={`/matches?contest=${c.id}`} className="btn btn-secondary"><Swords size={16} aria-hidden="true" /> {t('matches')}</Link>
        </div>
      </div>
      {c.description && <p className="lead">{c.description}</p>}

      {can('contests:manage') && c.allowed_transitions.length > 0 && (
        <section className="card" aria-labelledby="transitions">
          <h2 id="transitions">{t('stateMachine')}</h2>
          <div className="actions">
            {c.allowed_transitions.map((s) => (
              <button key={s} type="button" className={`btn ${s === 'cancelled' ? 'btn-danger' : 'btn-secondary'}`} onClick={() => transition(s)}>
                {t('moveTo', { state: t(`common:states.${s}`) })}
              </button>
            ))}
          </div>
        </section>
      )}

      <section className="card" aria-labelledby="roster">
        <div className="card-row">
          <h2 id="roster">{t('roster')}</h2>
          <div className="actions">
            {c.state === 'registration_open' && can('entries:create:own') && (
              <button type="button" className="btn" onClick={() => setDialog({ kind: 'enroll' })}><UserPlus size={16} aria-hidden="true" /> {t('enroll')}</button>
            )}
            {can('matches:create') && (c.state === 'registration_closed' || c.state === 'running') && entries.data && (
              <button type="button" className="btn" onClick={() => setDialog({ kind: 'match' })}><Swords size={16} aria-hidden="true" /> {t('createMatch')}</button>
            )}
          </div>
        </div>
        {entries.loading && <LoadingState />}
        {entries.error && <ErrorState error={entries.error} onRetry={entries.reload} />}
        {entries.data && entries.data.items.length === 0 && <EmptyState title={t('noEntries')} />}
        {entries.data && entries.data.items.length > 0 && (
          <div className="table-wrap">
            <table>
              <thead><tr><th>{t('agent')}</th><th>{t('owner')}</th><th>{t('lockedVersion')}</th><th>{t('status')}</th><th><span className="visually-hidden">{t('actions')}</span></th></tr></thead>
              <tbody>
                {entries.data.items.map((e) => (
                  <tr key={e.id}>
                    <td>{e.agent_name}</td>
                    <td>{e.username}</td>
                    <td>v{e.submission_version}</td>
                    <td><StatusBadge state={e.status} />{e.status_reason && <span className="muted small"> · {e.status_reason}</span>}</td>
                    <td className="row-actions">
                      {e.status === 'enrolled' && (e.user_id === currentUser?.id || can('entries:manage:any')) && !['finished', 'cancelled', 'archived'].includes(c.state) && (
                        <button type="button" className="btn btn-ghost" onClick={() => entryAction(e, 'withdraw')}>{t('withdraw')}</button>
                      )}
                      {e.status === 'enrolled' && can('entries:manage:any') && !['finished', 'cancelled', 'archived'].includes(c.state) && (
                        <button type="button" className="btn btn-ghost danger" onClick={() => entryAction(e, 'disqualify')}>{t('disqualify')}</button>
                      )}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
        {myEntries.length > 0 && <p className="muted small">{t('yourEntries', { count: myEntries.length })}</p>}
      </section>

      {dialog?.kind === 'enroll' && (
        <EnrollModal contest={c} onClose={() => setDialog(null)}
          onEnrolled={() => { setDialog(null); entries.reload(); toast.success(t('enrolled')); }} />
      )}
      {dialog?.kind === 'match' && (
        <CreateMatchModal contest={c} entries={entries.data.items} onClose={() => setDialog(null)}
          onCreated={(m) => { setDialog(null); navigate(`/matches/${m.id}`); }} />
      )}
      {(dialog?.kind === 'withdraw' || dialog?.kind === 'disqualify') && (
        <ReasonDialog title={t(dialog.kind)} danger={dialog.kind === 'disqualify'} confirmLabel={t(dialog.kind)}
          description={t(`${dialog.kind}Description`, { name: dialog.entry.agent_name })} onClose={() => setDialog(null)}
          onConfirm={async (reason) => {
            try {
              if (dialog.kind === 'withdraw') await ApiService.withdrawEntry(c.id, dialog.entry.id, reason);
              else await ApiService.disqualifyEntry(c.id, dialog.entry.id, reason);
              setDialog(null);
              entries.reload();
            } catch (err) {
              toast.error(describe(err));
            }
          }} />
      )}
    </div>
  );
}
