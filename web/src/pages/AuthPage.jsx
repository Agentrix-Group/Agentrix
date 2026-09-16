import React, { useState } from 'react';
import { useTranslation } from 'react-i18next';
import { ApiService } from '../service/apiService.js';

export function AuthPage({ onLoginSuccess }) {
  const { t } = useTranslation(['auth', 'errors', 'common']);
  const [isRegister, setIsRegister] = useState(false);
  const [username, setUsername] = useState('');
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState('');
  const [success, setSuccess] = useState('');
  const [loading, setLoading] = useState(false);

  const resolveErrorMessage = (err) => {
    if (err?.data?.errorCode && t(`errors:codes.${err.data.errorCode}`) !== `errors:codes.${err.data.errorCode}`) {
      return t(`errors:codes.${err.data.errorCode}`);
    }
    return err.message || t('auth:messages.operationFailed');
  };

  const handleSubmit = async (e) => {
    e.preventDefault();
    setError('');
    setSuccess('');
    setLoading(true);

    try {
      if (isRegister) {
        await ApiService.register(username, email, password);
        setSuccess(t('auth:messages.registerSuccess'));
        setIsRegister(false);
      } else {
        const res = await ApiService.login(username, password);
        if (onLoginSuccess) {
          onLoginSuccess(res.participant);
        }
      }
    } catch (err) {
      setError(resolveErrorMessage(err));
    } finally {
      setLoading(false);
    }
  };

  return (
    <div style={{ maxWidth: '420px', margin: '48px auto' }}>
      <div className="card">
        <h2 style={{ textAlign: 'center', marginBottom: '8px' }}>
          {isRegister ? t('auth:createAccount') : t('auth:welcomeBack')}
        </h2>
        <p style={{ textAlign: 'center', color: 'var(--text-secondary)', marginBottom: '24px', fontSize: '0.9rem' }}>
          {isRegister ? t('auth:joinSubtitle') : t('auth:loginSubtitle')}
        </p>

        {error && (
          <div style={{ padding: '10px 14px', background: 'var(--danger-bg)', color: 'var(--danger-text)', borderRadius: '6px', marginBottom: '16px', fontSize: '0.85rem' }}>
            {error}
          </div>
        )}
        {success && (
          <div style={{ padding: '10px 14px', background: 'var(--success-bg)', color: 'var(--success-text)', borderRadius: '6px', marginBottom: '16px', fontSize: '0.85rem' }}>
            {success}
          </div>
        )}

        <form onSubmit={handleSubmit} style={{ display: 'flex', flexDirection: 'column', gap: '16px' }}>
          <div>
            <label style={{ display: 'block', marginBottom: '6px', fontSize: '0.85rem', color: 'var(--text-secondary)' }}>
              {t('auth:labels.username')}
            </label>
            <input
              type="text"
              value={username}
              onChange={(e) => setUsername(e.target.value)}
              required
              placeholder={t('auth:placeholders.username')}
              style={{ width: '100%', padding: '10px 12px' }}
            />
          </div>

          {isRegister && (
            <div>
              <label style={{ display: 'block', marginBottom: '6px', fontSize: '0.85rem', color: 'var(--text-secondary)' }}>
                {t('auth:labels.email')}
              </label>
              <input
                type="email"
                value={email}
                onChange={(e) => setEmail(e.target.value)}
                required
                placeholder={t('auth:placeholders.email')}
                style={{ width: '100%', padding: '10px 12px' }}
              />
            </div>
          )}

          <div>
            <label style={{ display: 'block', marginBottom: '6px', fontSize: '0.85rem', color: 'var(--text-secondary)' }}>
              {t('auth:labels.password')}
            </label>
            <input
              type="password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              required
              placeholder={t('auth:placeholders.password')}
              style={{ width: '100%', padding: '10px 12px' }}
            />
          </div>

          <button type="submit" className="btn" disabled={loading} style={{ width: '100%', padding: '12px', marginTop: '8px' }}>
            {loading ? t('auth:buttons.processing') : isRegister ? t('auth:buttons.register') : t('auth:buttons.login')}
          </button>
        </form>

        <div style={{ textAlign: 'center', marginTop: '20px', fontSize: '0.85rem' }}>
          {isRegister ? (
            <span style={{ color: 'var(--text-secondary)' }}>
              {t('auth:links.alreadyHaveAccount')}{' '}
              <a
                href="#login"
                onClick={(e) => { e.preventDefault(); setIsRegister(false); setError(''); }}
                style={{ color: 'var(--accent)', textDecoration: 'none', fontWeight: '600' }}
              >
                {t('auth:links.logIn')}
              </a>
            </span>
          ) : (
            <span style={{ color: 'var(--text-secondary)' }}>
              {t('auth:links.dontHaveAccount')}{' '}
              <a
                href="#register"
                onClick={(e) => { e.preventDefault(); setIsRegister(true); setError(''); }}
                style={{ color: 'var(--accent)', textDecoration: 'none', fontWeight: '600' }}
              >
                {t('auth:links.register')}
              </a>
            </span>
          )}
        </div>
      </div>
    </div>
  );
}
