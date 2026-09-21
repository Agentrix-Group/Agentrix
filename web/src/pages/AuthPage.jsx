import React, { useState } from 'react';
import { useTranslation } from 'react-i18next';
import { useSession } from '../auth/SessionContext.jsx';
import { useRouter } from '../router/Router.jsx';
import { ApiService } from '../service/apiService.js';
import { useErrorMessage } from '../components/States.jsx';
import { useToast } from '../components/Toast.jsx';

export default function AuthPage({ next }) {
  const { t } = useTranslation('auth');
  const { login } = useSession();
  const { navigate } = useRouter();
  const toast = useToast();
  const describe = useErrorMessage();
  const [mode, setMode] = useState('login');
  const [form, setForm] = useState({ username: '', email: '', password: '' });
  const [error, setError] = useState(null);
  const [busy, setBusy] = useState(false);
  const update = (key) => (e) => setForm((f) => ({ ...f, [key]: e.target.value }));

  const submit = async (event) => {
    event.preventDefault();
    setBusy(true);
    setError(null);
    try {
      if (mode === 'register') {
        await ApiService.register(form.username, form.email, form.password);
        toast.success(t('registered'));
      }
      await login(form.username, form.password);
      navigate(next && next.startsWith('/') && !next.startsWith('//') ? next : '/agents');
    } catch (err) {
      setError(describe(err));
    } finally {
      setBusy(false);
    }
  };

  return (
    <section className="auth-page card" aria-labelledby="auth-title">
      <h1 id="auth-title">{mode === 'login' ? t('loginTitle') : t('registerTitle')}</h1>
      <form className="form" onSubmit={submit} noValidate={false}>
        <label className="field">
          <span>{t('username')}</span>
          <input autoComplete="username" required minLength={3} maxLength={32} value={form.username} onChange={update('username')} />
        </label>
        {mode === 'register' && (
          <label className="field">
            <span>{t('email')}</span>
            <input type="email" autoComplete="email" required value={form.email} onChange={update('email')} />
          </label>
        )}
        <label className="field">
          <span>{t('password')}</span>
          <input type="password" autoComplete={mode === 'login' ? 'current-password' : 'new-password'} required minLength={6}
            maxLength={128} value={form.password} onChange={update('password')} />
        </label>
        {error && <p className="form-error" role="alert">{error}</p>}
        <button type="submit" className="btn btn-block" disabled={busy}>
          {busy ? t('working') : mode === 'login' ? t('login') : t('register')}
        </button>
      </form>
      <button type="button" className="btn btn-link" onClick={() => { setMode(mode === 'login' ? 'register' : 'login'); setError(null); }}>
        {mode === 'login' ? t('toRegister') : t('toLogin')}
      </button>
    </section>
  );
}
