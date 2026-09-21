import React, { useState } from 'react';
import { useTranslation } from 'react-i18next';
import { RefreshCw, UserPlus } from 'lucide-react';
import { ApiService } from '../service/apiService.js';
import { useResource, usePolling } from '../hooks/hooks.js';
import { useSession } from '../auth/SessionContext.jsx';
import { LoadingState, ErrorState, useErrorMessage } from '../components/States.jsx';
import { StatusBadge } from '../components/StatusBadge.jsx';
import { Modal } from '../components/Modal.jsx';
import { useToast } from '../components/Toast.jsx';
import { formatDateTime } from '../i18n/formatters.js';

const ROLES = ['admin', 'organizer', 'player', 'referee', 'spectator'];
const STATUSES = ['active', 'suspended', 'disabled'];

function DetailValue({ value }) {
  if (Array.isArray(value)) {
    return value.length === 0 ? <span className="muted">—</span> : (
      <ul className="detail-list">{value.map((v, i) => <li key={i}><DetailValue value={v} /></li>)}</ul>
    );
  }
  if (value && typeof value === 'object') {
    return <dl className="detail-grid">{Object.entries(value).map(([k, v]) => <React.Fragment key={k}><dt>{k}</dt><dd><DetailValue value={v} /></dd></React.Fragment>)}</dl>;
  }
  return <code>{String(value)}</code>;
}

/** Renders the readiness DTO exactly as reported by the backend. A failed
 * request is shown as "unknown", never as healthy. */
function Readiness() {
  const { t, i18n } = useTranslation('admin');
  const readiness = useResource((signal) => ApiService.readiness({ signal }), []);
  usePolling(readiness.reload, 10000, true);
  const status = readiness.error ? 'unknown' : readiness.data?.status;
  return (
    <section className="card" aria-labelledby="readiness-title">
      <div className="card-row">
        <h2 id="readiness-title">{t('readiness')} {status && <StatusBadge state={status} />}</h2>
        <button type="button" className="btn btn-secondary" onClick={readiness.reload}><RefreshCw size={16} aria-hidden="true" /> {t('refresh')}</button>
      </div>
      {readiness.loading && !readiness.data && <LoadingState />}
      {readiness.error && <ErrorState error={readiness.error} onRetry={readiness.reload} title={t('readinessUnavailable')} />}
      {readiness.data && !readiness.error && (
        <>
          <p className="muted small">{t('checkedAt', { at: formatDateTime(readiness.data.checked_at, i18n.language) })}</p>
          <div className="component-grid">
            {readiness.data.components.map((c) => (
              <article key={c.name} className={`component component-${c.status}`}>
                <header className="card-row"><h3>{t(`components.${c.name}`, { defaultValue: c.name })}</h3><StatusBadge state={c.status} /></header>
                <p>{c.message}</p>
                {c.details && <details><summary>{t('details')}</summary><DetailValue value={c.details} /></details>}
              </article>
            ))}
          </div>
        </>
      )}
    </section>
  );
}

function CreateUserModal({ onClose, onCreated }) {
  const { t } = useTranslation('admin');
  const describe = useErrorMessage();
  const [form, setForm] = useState({ username: '', email: '', password: '', roles: ['player'] });
  const [error, setError] = useState(null);
  const submit = async (e) => {
    e.preventDefault();
    try {
      onCreated(await ApiService.createUser(form));
    } catch (err) {
      setError(describe(err));
    }
  };
  return (
    <Modal title={t('createUser')} onClose={onClose}>
      <form className="form" onSubmit={submit}>
        {['username', 'email', 'password'].map((k) => (
          <label key={k} className="field"><span>{t(k)}</span>
            <input required type={k === 'password' ? 'password' : k === 'email' ? 'email' : 'text'} value={form[k]} onChange={(e) => setForm({ ...form, [k]: e.target.value })} />
          </label>
        ))}
        <fieldset className="field"><legend>{t('roles')}</legend>
          {ROLES.map((r) => (
            <label key={r} className="checkbox-row"><input type="checkbox" checked={form.roles.includes(r)}
              onChange={() => setForm({ ...form, roles: form.roles.includes(r) ? form.roles.filter((x) => x !== r) : [...form.roles, r] })} /> {r}</label>
          ))}
        </fieldset>
        {error && <p className="form-error" role="alert">{error}</p>}
        <div className="form-actions">
          <button type="button" className="btn btn-secondary" onClick={onClose}>{t('common:buttons.cancel')}</button>
          <button type="submit" className="btn" disabled={form.roles.length === 0}>{t('createUser')}</button>
        </div>
      </form>
    </Modal>
  );
}

function RolesModal({ user, onClose, onSaved }) {
  const { t } = useTranslation('admin');
  const describe = useErrorMessage();
  const [roles, setRoles] = useState(user.roles);
  const [error, setError] = useState(null);
  const save = async (e) => {
    e.preventDefault();
    try {
      onSaved(await ApiService.replaceRoles(user.id, roles));
    } catch (err) {
      setError(describe(err));
    }
  };
  return (
    <Modal title={t('editRoles', { name: user.username })} onClose={onClose}>
      <form className="form" onSubmit={save}>
        <fieldset className="field"><legend>{t('roles')}</legend>
          {ROLES.map((r) => (
            <label key={r} className="checkbox-row"><input type="checkbox" checked={roles.includes(r)}
              onChange={() => setRoles(roles.includes(r) ? roles.filter((x) => x !== r) : [...roles, r])} /> {r}</label>
          ))}
        </fieldset>
        <p className="muted small">{t('rolesImmediate')}</p>
        {error && <p className="form-error" role="alert">{error}</p>}
        <div className="form-actions">
          <button type="button" className="btn btn-secondary" onClick={onClose}>{t('common:buttons.cancel')}</button>
          <button type="submit" className="btn" disabled={roles.length === 0}>{t('save')}</button>
        </div>
      </form>
    </Modal>
  );
}

function Users() {
  const { t } = useTranslation('admin');
  const { can, currentUser } = useSession();
  const toast = useToast();
  const describe = useErrorMessage();
  const [filter, setFilter] = useState('');
  const users = useResource((signal) => ApiService.listUsers(filter || undefined, { signal }), [filter]);
  const [dialog, setDialog] = useState(null);

  const setStatus = async (user, status) => {
    try {
      await ApiService.setUserStatus(user.id, status);
      toast.success(t('statusChanged', { name: user.username, status: t(`common:states.${status}`) }));
      users.reload();
    } catch (err) {
      toast.error(describe(err));
    }
  };

  return (
    <section className="card" aria-labelledby="users-title">
      <div className="card-row">
        <h2 id="users-title">{t('users')}</h2>
        <div className="actions">
          <label className="field inline"><span>{t('filter')}</span>
            <select value={filter} onChange={(e) => setFilter(e.target.value)}>
              <option value="">{t('all')}</option>
              {STATUSES.map((s) => <option key={s} value={s}>{t(`common:states.${s}`)}</option>)}
            </select>
          </label>
          {can('users:roles:manage') && <button type="button" className="btn" onClick={() => setDialog({ kind: 'create' })}><UserPlus size={16} aria-hidden="true" /> {t('createUser')}</button>}
        </div>
      </div>
      {users.loading && !users.data && <LoadingState />}
      {users.error && <ErrorState error={users.error} onRetry={users.reload} />}
      {users.data && (
        <div className="table-wrap">
          <table>
            <thead><tr><th>{t('username')}</th><th>{t('email')}</th><th>{t('roles')}</th><th>{t('status')}</th><th><span className="visually-hidden">{t('actions')}</span></th></tr></thead>
            <tbody>
              {users.data.items.map((u) => (
                <tr key={u.id}>
                  <td>{u.username}</td>
                  <td className="muted">{u.email}</td>
                  <td>{u.roles.join(', ')}</td>
                  <td>
                    {u.id === currentUser?.id || !can('users:update:any') ? <StatusBadge state={u.status} /> : (
                      <label>
                        <span className="visually-hidden">{t('statusOf', { name: u.username })}</span>
                        <select value={u.status} onChange={(e) => setStatus(u, e.target.value)}>
                          {STATUSES.map((s) => <option key={s} value={s}>{t(`common:states.${s}`)}</option>)}
                        </select>
                      </label>
                    )}
                  </td>
                  <td>{can('users:roles:manage') && <button type="button" className="btn btn-ghost" onClick={() => setDialog({ kind: 'roles', user: u })}>{t('editRolesShort')}</button>}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
      {dialog?.kind === 'create' && <CreateUserModal onClose={() => setDialog(null)} onCreated={() => { setDialog(null); users.reload(); }} />}
      {dialog?.kind === 'roles' && <RolesModal user={dialog.user} onClose={() => setDialog(null)} onSaved={() => { setDialog(null); users.reload(); }} />}
    </section>
  );
}

export default function AdminPage() {
  const { t } = useTranslation('admin');
  const { can } = useSession();
  return (
    <div className="page">
      <h1>{t('title')}</h1>
      <Readiness />
      {can('users:read:any') && <Users />}
    </div>
  );
}
