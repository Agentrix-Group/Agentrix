import React, { useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Plus, Trophy } from 'lucide-react';
import { ApiService } from '../service/apiService.js';
import { useResource } from '../hooks/hooks.js';
import { useSession } from '../auth/SessionContext.jsx';
import { Link, useRouter } from '../router/Router.jsx';
import { SkeletonRows, ErrorState, EmptyState, useErrorMessage } from '../components/States.jsx';
import { StatusBadge } from '../components/StatusBadge.jsx';
import { Modal } from '../components/Modal.jsx';
import { formatDate } from '../i18n/formatters.js';

function CreateContestModal({ games, onClose, onCreated }) {
  const { t } = useTranslation('contests');
  const describe = useErrorMessage();
  const [form, setForm] = useState({ game_id: games[0]?.id || '', name: '', description: '' });
  const [error, setError] = useState(null);
  const [busy, setBusy] = useState(false);
  const submit = async (e) => {
    e.preventDefault();
    setBusy(true);
    setError(null);
    try {
      onCreated(await ApiService.createContest(form));
    } catch (err) {
      setError(describe(err));
    } finally {
      setBusy(false);
    }
  };
  return (
    <Modal title={t('create')} onClose={onClose}>
      <form className="form" onSubmit={submit}>
        <label className="field"><span>{t('game')}</span>
          <select required value={form.game_id} onChange={(e) => setForm({ ...form, game_id: e.target.value })}>
            {games.map((g) => <option key={g.id} value={g.id}>{g.name} {g.version}</option>)}
          </select>
        </label>
        <label className="field"><span>{t('name')}</span>
          <input required maxLength={128} value={form.name} onChange={(e) => setForm({ ...form, name: e.target.value })} />
        </label>
        <label className="field"><span>{t('description')}</span>
          <textarea maxLength={4000} value={form.description} onChange={(e) => setForm({ ...form, description: e.target.value })} />
        </label>
        <p className="muted small">{t('defaultScoring')}</p>
        {error && <p className="form-error" role="alert">{error}</p>}
        <div className="form-actions">
          <button type="button" className="btn btn-secondary" onClick={onClose}>{t('common:buttons.cancel')}</button>
          <button type="submit" className="btn" disabled={busy}>{t('create')}</button>
        </div>
      </form>
    </Modal>
  );
}

export default function ContestsPage() {
  const { t, i18n } = useTranslation('contests');
  const { can } = useSession();
  const { navigate } = useRouter();
  const contests = useResource((signal) => ApiService.listContests({ signal }), []);
  const games = useResource((signal) => ApiService.listGames({ signal }), []);
  const [creating, setCreating] = useState(false);

  return (
    <div className="page">
      <div className="page-header">
        <h1>{t('title')}</h1>
        {can('contests:manage') && games.data && (
          <button type="button" className="btn" onClick={() => setCreating(true)}><Plus size={16} aria-hidden="true" /> {t('create')}</button>
        )}
      </div>
      {contests.loading && <SkeletonRows rows={4} />}
      {contests.error && <ErrorState error={contests.error} onRetry={contests.reload} />}
      {contests.data && contests.data.items.length === 0 && <EmptyState icon={Trophy} title={t('empty')} />}
      {contests.data && contests.data.items.length > 0 && (
        <div className="card-grid">
          {contests.data.items.map((c) => (
            <article key={c.id} className="card contest-card">
              <header className="card-row">
                <h2><Link to={`/contests/${c.id}`}>{c.name}</Link></h2>
                <StatusBadge state={c.state} />
              </header>
              <p className="muted">{c.description || t('noDescription')}</p>
              <p className="small muted">{c.game_id} · {t('created')} {formatDate(c.created_at, i18n.language)}</p>
            </article>
          ))}
        </div>
      )}
      {creating && (
        <CreateContestModal games={games.data.items} onClose={() => setCreating(false)}
          onCreated={(c) => { setCreating(false); navigate(`/contests/${c.id}`); }} />
      )}
    </div>
  );
}
