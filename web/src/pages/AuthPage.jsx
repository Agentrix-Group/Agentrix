import React, { useState } from 'react';
import { useTranslation } from 'react-i18next';
import { ApiService } from '../service/apiService.js';
import { useSession } from '../auth/SessionContext.jsx';
import { useRouter } from '../router/Router.jsx';

export function AuthPage({ onLoginSuccess }) {
  const { t } = useTranslation(['auth', 'errors', 'common']);
  const { login } = useSession();
  const { navigate } = useRouter();

  const [isRegister, setIsRegister] = useState(false);
  const [username, setUsername] = useState('');
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [fieldErrors, setFieldErrors] = useState({});
  const [error, setError] = useState('');
  const [success, setSuccess] = useState('');
  const [loading, setLoading] = useState(false);

  // Extract redirect query parameter if present
  const redirectTarget = typeof window !== 'undefined'
    ? new URLSearchParams(window.location.search).get('redirect')
    : null;

  const validateForm = () => {
    const errors = {};
    if (!username.trim() || username.trim().length < 3) {
      errors.username = t('auth:validation.usernameRequired');
    }
    if (isRegister) {
      const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
      if (!email.trim() || !emailRegex.test(email.trim())) {
        errors.email = t('auth:validation.emailInvalid');
      }
      if (!password || password.length < 8) {
        errors.password = t('auth:validation.passwordRequired');
      }
    } else {
      if (!password) {
        errors.password = t('auth:validation.passwordRequired');
      }
    }
    setFieldErrors(errors);
    return Object.keys(errors).length === 0;
  };

  const resolveErrorMessage = (err) => {
    if (err?.errorCode && t(`errors:codes.${err.errorCode}`) !== `errors:codes.${err.errorCode}`) {
      return t(`errors:codes.${err.errorCode}`);
    }
    if (err?.data?.errorCode && t(`errors:codes.${err.data.errorCode}`) !== `errors:codes.${err.data.errorCode}`) {
      return t(`errors:codes.${err.data.errorCode}`);
    }
    return err?.message || t('auth:messages.operationFailed');
  };

  const handleSubmit = async (e) => {
    e.preventDefault();
    setError('');
    setSuccess('');

    if (!validateForm()) {
      return;
    }

    setLoading(true);

    try {
      if (isRegister) {
        await ApiService.register(username.trim(), email.trim(), password);
        setSuccess(t('auth:messages.registerSuccess'));
        setIsRegister(false);
        setPassword('');
        setFieldErrors({});
      } else {
        const res = await login(username.trim(), password);
        const user = res?.participant || null;
        if (onLoginSuccess) {
          onLoginSuccess(user, redirectTarget);
        } else {
          navigate(redirectTarget || '/agents');
        }
      }
    } catch (err) {
      setError(resolveErrorMessage(err));
    } finally {
      setLoading(false);
    }
  };

  const toggleMode = (targetRegister) => {
    setIsRegister(targetRegister);
    setError('');
    setSuccess('');
    setFieldErrors({});
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
          <div
            role="alert"
            style={{
              padding: '10px 14px',
              background: 'var(--danger-bg, rgba(239, 68, 68, 0.12))',
              color: 'var(--danger, #ef4444)',
              borderRadius: '6px',
              marginBottom: '16px',
              fontSize: '0.85rem',
            }}
          >
            {error}
          </div>
        )}
        {success && (
          <div
            role="status"
            style={{
              padding: '10px 14px',
              background: 'var(--success-bg, rgba(34, 197, 94, 0.12))',
              color: 'var(--success, #22c55e)',
              borderRadius: '6px',
              marginBottom: '16px',
              fontSize: '0.85rem',
            }}
          >
            {success}
          </div>
        )}

        <form onSubmit={handleSubmit} style={{ display: 'flex', flexDirection: 'column', gap: '16px' }} noValidate>
          <div>
            <label
              htmlFor="auth-username"
              style={{ display: 'block', marginBottom: '6px', fontSize: '0.85rem', color: 'var(--text-secondary)' }}
            >
              {t('auth:labels.username')}
            </label>
            <input
              id="auth-username"
              type="text"
              value={username}
              onChange={(e) => {
                setUsername(e.target.value);
                if (fieldErrors.username) setFieldErrors({ ...fieldErrors, username: '' });
              }}
              required
              aria-invalid={Boolean(fieldErrors.username)}
              aria-describedby={fieldErrors.username ? 'username-error' : undefined}
              placeholder={t('auth:placeholders.username')}
              style={{
                width: '100%',
                padding: '10px 12px',
                borderColor: fieldErrors.username ? 'var(--danger, #ef4444)' : undefined,
              }}
            />
            {fieldErrors.username && (
              <span id="username-error" role="alert" style={{ color: 'var(--danger, #ef4444)', fontSize: '0.8rem', display: 'block', marginTop: '4px' }}>
                {fieldErrors.username}
              </span>
            )}
          </div>

          {isRegister && (
            <div>
              <label
                htmlFor="auth-email"
                style={{ display: 'block', marginBottom: '6px', fontSize: '0.85rem', color: 'var(--text-secondary)' }}
              >
                {t('auth:labels.email')}
              </label>
              <input
                id="auth-email"
                type="email"
                value={email}
                onChange={(e) => {
                  setEmail(e.target.value);
                  if (fieldErrors.email) setFieldErrors({ ...fieldErrors, email: '' });
                }}
                required
                aria-invalid={Boolean(fieldErrors.email)}
                aria-describedby={fieldErrors.email ? 'email-error' : undefined}
                placeholder={t('auth:placeholders.email')}
                style={{
                  width: '100%',
                  padding: '10px 12px',
                  borderColor: fieldErrors.email ? 'var(--danger, #ef4444)' : undefined,
                }}
              />
              {fieldErrors.email && (
                <span id="email-error" role="alert" style={{ color: 'var(--danger, #ef4444)', fontSize: '0.8rem', display: 'block', marginTop: '4px' }}>
                  {fieldErrors.email}
                </span>
              )}
            </div>
          )}

          <div>
            <label
              htmlFor="auth-password"
              style={{ display: 'block', marginBottom: '6px', fontSize: '0.85rem', color: 'var(--text-secondary)' }}
            >
              {t('auth:labels.password')}
            </label>
            <input
              id="auth-password"
              type="password"
              value={password}
              onChange={(e) => {
                setPassword(e.target.value);
                if (fieldErrors.password) setFieldErrors({ ...fieldErrors, password: '' });
              }}
              required
              aria-invalid={Boolean(fieldErrors.password)}
              aria-describedby={fieldErrors.password ? 'password-error' : undefined}
              placeholder={t('auth:placeholders.password')}
              style={{
                width: '100%',
                padding: '10px 12px',
                borderColor: fieldErrors.password ? 'var(--danger, #ef4444)' : undefined,
              }}
            />
            {fieldErrors.password && (
              <span id="password-error" role="alert" style={{ color: 'var(--danger, #ef4444)', fontSize: '0.8rem', display: 'block', marginTop: '4px' }}>
                {fieldErrors.password}
              </span>
            )}
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
                onClick={(e) => { e.preventDefault(); toggleMode(false); }}
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
                onClick={(e) => { e.preventDefault(); toggleMode(true); }}
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
