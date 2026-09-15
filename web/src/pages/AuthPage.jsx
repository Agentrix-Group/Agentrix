import React, { useState } from 'react';
import { ApiService } from '../service/apiService.js';

export function AuthPage({ onLoginSuccess }) {
  const [isRegister, setIsRegister] = useState(false);
  const [username, setUsername] = useState('');
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState('');
  const [success, setSuccess] = useState('');
  const [loading, setLoading] = useState(false);

  const handleSubmit = async (e) => {
    e.preventDefault();
    setError('');
    setSuccess('');
    setLoading(true);

    try {
      if (isRegister) {
        await ApiService.register(username, email, password);
        setSuccess('Registration successful! You can now log in.');
        setIsRegister(false);
      } else {
        const res = await ApiService.login(username, password);
        if (onLoginSuccess) {
          onLoginSuccess(res.participant);
        }
      }
    } catch (err) {
      setError(err.message || 'Operation failed');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div style={{ maxWidth: '420px', margin: '48px auto' }}>
      <div className="card">
        <h2 style={{ textAlign: 'center', marginBottom: '8px' }}>
          {isRegister ? 'Create Account' : 'Welcome Back'}
        </h2>
        <p style={{ textAlign: 'center', color: 'var(--text-secondary)', marginBottom: '24px', fontSize: '0.9rem' }}>
          {isRegister
            ? 'Join Agentrix to compete in automated bot arenas'
            : 'Log in with your competitive bot developer account'}
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
              Username
            </label>
            <input
              type="text"
              value={username}
              onChange={(e) => setUsername(e.target.value)}
              required
              placeholder="e.g. mastercoder"
              style={{ width: '100%', padding: '10px 12px' }}
            />
          </div>

          {isRegister && (
            <div>
              <label style={{ display: 'block', marginBottom: '6px', fontSize: '0.85rem', color: 'var(--text-secondary)' }}>
                Email Address
              </label>
              <input
                type="email"
                value={email}
                onChange={(e) => setEmail(e.target.value)}
                required
                placeholder="developer@example.com"
                style={{ width: '100%', padding: '10px 12px' }}
              />
            </div>
          )}

          <div>
            <label style={{ display: 'block', marginBottom: '6px', fontSize: '0.85rem', color: 'var(--text-secondary)' }}>
              Password
            </label>
            <input
              type="password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              required
              placeholder="Min. 6 characters"
              style={{ width: '100%', padding: '10px 12px' }}
            />
          </div>

          <button type="submit" className="btn" disabled={loading} style={{ width: '100%', padding: '12px', marginTop: '8px' }}>
            {loading ? 'Processing...' : isRegister ? 'Register Account' : 'Sign In'}
          </button>
        </form>

        <div style={{ textAlign: 'center', marginTop: '20px', fontSize: '0.85rem' }}>
          {isRegister ? (
            <span style={{ color: 'var(--text-secondary)' }}>
              Already have an account?{' '}
              <a
                href="#login"
                onClick={(e) => { e.preventDefault(); setIsRegister(false); setError(''); }}
                style={{ color: 'var(--accent)', textDecoration: 'none', fontWeight: '600' }}
              >
                Log In
              </a>
            </span>
          ) : (
            <span style={{ color: 'var(--text-secondary)' }}>
              Don't have an account yet?{' '}
              <a
                href="#register"
                onClick={(e) => { e.preventDefault(); setIsRegister(true); setError(''); }}
                style={{ color: 'var(--accent)', textDecoration: 'none', fontWeight: '600' }}
              >
                Register
              </a>
            </span>
          )}
        </div>
      </div>
    </div>
  );
}
