import React, { useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Plus, Bot, Power } from 'lucide-react';
import { ApiService } from '../service/apiService.js';
import { useResource, usePolling } from '../hooks/hooks.js';
import { useSession } from '../auth/SessionContext.jsx';
import { Link, useRouter } from '../router/Router.jsx';
import { LoadingState, SkeletonRows, ErrorState, EmptyState, useErrorMessage } from '../components/States.jsx';
import { StatusBadge } from '../components/StatusBadge.jsx';
import { Modal } from '../components/Modal.jsx';
import { useToast } from '../components/Toast.jsx';
import { BundleDropzone } from '../components/BundleDropzone.jsx';
import { formatDateTime } from '../i18n/formatters.js';

function CreateAgentModal({ games, onClose, onCreated }) {
  const { t } = useTranslation('agents');
  const describe = useErrorMessage();
  const [form, setForm] = useState({ game_id: games[0]?.id || '', name: '', description: '' });
  const [error, setError] = useState(null);
  const submit = async (e) => {
    e.preventDefault();
    setError(null);
    try {
      onCreated(await ApiService.createAgent(form));
    } catch (err) {
      setError(describe(err));
    }
  };
  return (
    <Modal title={t('create')} onClose={onClose}>
      <form className="form" onSubmit={submit}>
        <label className="field"><span>{t('game')}</span>
          <select value={form.game_id} onChange={(e) => setForm({ ...form, game_id: e.target.value })}>
            {games.map((g) => <option key={g.id} value={g.id}>{g.name}</option>)}
          </select>
        </label>
        <label className="field"><span>{t('name')}</span>
          <input required maxLength={64} value={form.name} onChange={(e) => setForm({ ...form, name: e.target.value })} />
        </label>
        <label className="field"><span>{t('description')}</span>
          <textarea maxLength={2000} value={form.description} onChange={(e) => setForm({ ...form, description: e.target.value })} />
        </label>
        {error && <p className="form-error" role="alert">{error}</p>}
        <div className="form-actions">
          <button type="button" className="btn btn-secondary" onClick={onClose}>{t('common:buttons.cancel')}</button>
          <button type="submit" className="btn">{t('create')}</button>
        </div>
      </form>
    </Modal>
  );
}

function AgentDetail({ agentId, onChanged }) {
  const { t, i18n } = useTranslation('agents');
  const toast = useToast();
  const describe = useErrorMessage();
  const agent = useResource((signal) => ApiService.getAgent(agentId, { signal }), [agentId]);
  const subs = useResource((signal) => ApiService.listSubmissions(agentId, { signal }), [agentId]);
  const [uploading, setUploading] = useState(false);
  const admitting = (subs.data?.items || []).some((s) => s.status === 'validating');
  usePolling(subs.reload, 1500, admitting);

  if (agent.loading) return <LoadingState />;
  if (agent.error) return <ErrorState error={agent.error} onRetry={agent.reload} />;
  const a = agent.data;

  const upload = async (file) => {
    setUploading(true);
    try {
      const sub = await ApiService.uploadSubmission(a.id, file);
      toast.info(t('uploaded', { version: sub.version }));
      subs.reload();
    } catch (err) {
      toast.error(describe(err));
    } finally {
      setUploading(false);
    }
  };
  const setStatus = async (status) => {
    try {
      agent.setData(await ApiService.setAgentStatus(a.id, status));
      onChanged();
    } catch (err) {
      toast.error(describe(err));
    }
  };
  const disable = async (sub) => {
    try {
      await ApiService.disableSubmission(sub.id);
      subs.reload();
    } catch (err) {
      toast.error(describe(err));
    }
  };

  return (
    <section className="card" aria-labelledby="agent-title">
      <div className="card-row">
        <h2 id="agent-title">{a.name} <StatusBadge state={a.status} /></h2>
        <button type="button" className="btn btn-secondary" onClick={() => setStatus(a.status === 'active' ? 'disabled' : 'active')}>
          <Power size={16} aria-hidden="true" /> {a.status === 'active' ? t('disable') : t('activate')}
        </button>
      </div>
      {a.description && <p className="muted">{a.description}</p>}
      {a.status === 'active' && <BundleDropzone onUpload={upload} busy={uploading} />}
      <h3>{t('versions')}</h3>
      {subs.loading && !subs.data && <SkeletonRows />}
      {subs.error && <ErrorState error={subs.error} onRetry={subs.reload} />}
      {subs.data && subs.data.items.length === 0 && <p className="muted">{t('noVersions')}</p>}
      {subs.data && subs.data.items.length > 0 && (
        <div className="table-wrap">
          <table>
            <thead><tr><th>{t('version')}</th><th>{t('status')}</th><th>{t('digest')}</th><th>{t('uploadedAt')}</th><th><span className="visually-hidden">{t('actions')}</span></th></tr></thead>
            <tbody>
              {[...subs.data.items].reverse().map((s) => (
                <tr key={s.id}>
                  <td>v{s.version}</td>
                  <td>
                    <StatusBadge state={s.status} />
                    {s.status === 'validating' && <span className="muted small"> {t('admissionRunning')}</span>}
                    {s.admission_error && <details className="small"><summary>{t('whyRejected')}</summary><pre className="admission-error">{s.admission_error}</pre></details>}
                  </td>
                  <td className="small"><code title={s.artifact_sha256}>{s.artifact_sha256.slice(0, 12)}</code></td>
                  <td className="small muted">{formatDateTime(s.created_at, i18n.language)}</td>
                  <td>{s.status === 'ready' && <button type="button" className="btn btn-ghost" onClick={() => disable(s)}>{t('retire')}</button>}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </section>
  );
}

export default function AgentsPage({ agentId }) {
  const { t } = useTranslation('agents');
  const { can } = useSession();
  const { navigate } = useRouter();
  const agents = useResource((signal) => ApiService.listAgents({ signal }), []);
  const games = useResource((signal) => ApiService.listGames({ signal }), []);
  const [creating, setCreating] = useState(false);
  const selected = agentId || agents.data?.items[0]?.id;

  return (
    <div className="page">
      <div className="page-header">
        <h1>{t('title')}</h1>
        {can('agents:create') && games.data && (
          <button type="button" className="btn" onClick={() => setCreating(true)}><Plus size={16} aria-hidden="true" /> {t('create')}</button>
        )}
      </div>
      {agents.loading && <SkeletonRows />}
      {agents.error && <ErrorState error={agents.error} onRetry={agents.reload} />}
      {agents.data && agents.data.items.length === 0 && <EmptyState icon={Bot} title={t('empty')} description={t('emptyDescription')} />}
      {agents.data && agents.data.items.length > 0 && (
        <div className="split">
          <nav className="card side-list" aria-label={t('title')}>
            <ul className="list">
              {agents.data.items.map((a) => (
                <li key={a.id}>
                  <Link to={`/agents/${a.id}`} className={`list-link ${a.id === selected ? 'active' : ''}`}>
                    {a.name} <StatusBadge state={a.status} />
                  </Link>
                </li>
              ))}
            </ul>
          </nav>
          {selected && <AgentDetail key={selected} agentId={selected} onChanged={agents.reload} />}
        </div>
      )}
      {creating && (
        <CreateAgentModal games={games.data.items} onClose={() => setCreating(false)}
          onCreated={(a) => { setCreating(false); agents.reload(); navigate(`/agents/${a.id}`); }} />
      )}
    </div>
  );
}
