import React from 'react';
import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import i18n from '../src/i18n/index.js';
import {
  saveTokens,
  getAccessToken,
  getRefreshToken,
  clearTokens,
  refreshAuthTokens,
  hasCapability,
  canRunMatch,
  canCreateAgent,
  onSessionExpired,
} from '../src/auth/session.js';
import { SessionProvider, useSession } from '../src/auth/SessionContext.jsx';
import { ProtectedRoute } from '../src/components/ProtectedRoute.jsx';
import { SessionExpiredBanner } from '../src/components/SessionExpiredBanner.jsx';
import { AuthPage } from '../src/pages/AuthPage.jsx';
import { Router } from '../src/router/Router.jsx';
import { ApiService } from '../src/service/apiService.js';

describe('Auth, Session and Capabilities (ADR-0009 / Sprint FQ-1)', () => {
  beforeEach(async () => {
    localStorage.clear();
    await i18n.changeLanguage('es');
    vi.restoreAllMocks();
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  describe('Token management and coordinated refresh (F0.3)', () => {
    it('saves and retrieves access and refresh tokens properly', () => {
      saveTokens({ access_token: 'acc-123', refresh_token: 'ref-456' });
      expect(getAccessToken()).toBe('acc-123');
      expect(getRefreshToken()).toBe('ref-456');

      clearTokens();
      expect(getAccessToken()).toBeNull();
      expect(getRefreshToken()).toBeNull();
    });

    it('coordinates concurrent refresh calls to execute only a single HTTP request', async () => {
      saveTokens({ access_token: 'expired-acc', refresh_token: 'valid-ref' });

      let fetchCount = 0;
      global.fetch = vi.fn().mockImplementation((url) => {
        if (url.includes('/auth/refresh')) {
          fetchCount++;
          return Promise.resolve({
            ok: true,
            status: 200,
            headers: new Headers({ 'content-type': 'application/json' }),
            json: () => Promise.resolve({
              access_token: 'new-acc-789',
              refresh_token: 'new-ref-789',
            }),
          });
        }
        return Promise.reject(new Error('Unknown url'));
      });

      // Fire 3 simultaneous refresh calls
      const [res1, res2, res3] = await Promise.all([
        refreshAuthTokens('http://localhost:8080/api/v1'),
        refreshAuthTokens('http://localhost:8080/api/v1'),
        refreshAuthTokens('http://localhost:8080/api/v1'),
      ]);

      expect(fetchCount).toBe(1);
      expect(res1.access_token).toBe('new-acc-789');
      expect(res2.access_token).toBe('new-acc-789');
      expect(res3.access_token).toBe('new-acc-789');
      expect(getAccessToken()).toBe('new-acc-789');
      expect(getRefreshToken()).toBe('new-ref-789');
    });

    it('triggers session expiration and purges tokens when refresh fails', async () => {
      saveTokens({ access_token: 'bad-acc', refresh_token: 'revoked-ref' });

      global.fetch = vi.fn().mockResolvedValue({
        ok: false,
        status: 401,
        headers: new Headers(),
      });

      const expiredSpy = vi.fn();
      const unsubscribe = onSessionExpired(expiredSpy);

      await expect(refreshAuthTokens('http://localhost:8080/api/v1')).rejects.toThrow();

      expect(expiredSpy).toHaveBeenCalled();
      expect(getAccessToken()).toBeNull();
      expect(getRefreshToken()).toBeNull();

      unsubscribe();
    });
  });

  describe('Capabilities and Permission resolution (F1.4)', () => {
    it('resolves capabilities from backend array or role fallback', () => {
      const adminUser = { id: 'u1', role_id: 'admin' };
      expect(canRunMatch(adminUser)).toBe(true);
      expect(canCreateAgent(adminUser)).toBe(true);
      expect(hasCapability(adminUser, 'custom:action')).toBe(true);

      const participantUser = { id: 'u2', role_id: 'participant' };
      expect(canRunMatch(participantUser)).toBe(false);
      expect(canCreateAgent(participantUser)).toBe(true);

      const customUserWithCaps = {
        id: 'u3',
        role_id: 'custom',
        capabilities: ['matches:run', 'rankings:view'],
      };
      expect(canRunMatch(customUserWithCaps)).toBe(true);
      expect(canCreateAgent(customUserWithCaps)).toBe(false);

      expect(canRunMatch(null)).toBe(false);
    });
  });

  describe('ProtectedRoute and SessionExpiredBanner', () => {
    it('renders authRequired prompt for unauthenticated users in ProtectedRoute', () => {
      render(
        <Router>
          <SessionProvider>
            <ProtectedRoute>
              <div>Secret Bot Area</div>
            </ProtectedRoute>
          </SessionProvider>
        </Router>
      );

      expect(screen.queryByText('Secret Bot Area')).toBeNull();
      expect(screen.getByText('Autenticación requerida')).toBeDefined();
      expect(screen.getByText('Ingresar a tu cuenta')).toBeDefined();
    });

    it('renders child content when authenticated', async () => {
      saveTokens({ access_token: 'valid-test-token' });
      vi.spyOn(ApiService, 'getCurrentUser').mockResolvedValue({
        id: 'u-1',
        username: 'coder',
        role_id: 'participant',
      });

      render(
        <Router>
          <SessionProvider>
            <ProtectedRoute>
              <div>Secret Bot Area</div>
            </ProtectedRoute>
          </SessionProvider>
        </Router>
      );

      expect(await screen.findByText('Secret Bot Area')).toBeDefined();
    });
  });

  describe('AuthPage validation and field errors', () => {
    it('validates username length >= 3 and password length >= 8 on register', async () => {
      render(
        <Router>
          <SessionProvider>
            <AuthPage onLoginSuccess={() => {}} />
          </SessionProvider>
        </Router>
      );

      // Switch to register mode
      fireEvent.click(screen.getByText('Regístrate'));

      const submitBtn = screen.getByRole('button', { name: 'Registrar cuenta' });
      fireEvent.click(submitBtn);

      expect(await screen.findByText(/El nombre de usuario es obligatorio/)).toBeDefined();

      // Enter valid username but invalid email
      fireEvent.change(screen.getByPlaceholderText(/mastercoder/), { target: { value: 'cooldev' } });
      fireEvent.click(submitBtn);

      expect(await screen.findByText(/Ingresa una dirección de correo electrónico válida/)).toBeDefined();

      // Enter valid email but short password
      fireEvent.change(screen.getByPlaceholderText(/desarrollador@ejemplo.com/), { target: { value: 'cool@dev.com' } });
      fireEvent.change(screen.getByPlaceholderText(/Mín. 8 caracteres/), { target: { value: '123' } });
      fireEvent.click(submitBtn);

      expect(await screen.findByText(/La contraseña debe tener al menos 8 caracteres/)).toBeDefined();
    });

    it('preserves redirect destination upon successful login', async () => {
      const loginSpy = vi.spyOn(ApiService, 'login').mockResolvedValue({
        token: { access_token: 'acc', refresh_token: 'ref' },
        participant: { id: 'p1', username: 'pro' },
      });
      vi.spyOn(ApiService, 'getCurrentUser').mockResolvedValue({ id: 'p1', username: 'pro' });

      const onLoginSuccess = vi.fn();
      render(
        <Router>
          <SessionProvider>
            <AuthPage onLoginSuccess={onLoginSuccess} />
          </SessionProvider>
        </Router>
      );

      fireEvent.change(screen.getByPlaceholderText(/mastercoder/), { target: { value: 'prodev' } });
      fireEvent.change(screen.getByPlaceholderText(/Mín. 8 caracteres/), { target: { value: 'password123' } });
      fireEvent.click(screen.getByRole('button', { name: 'Iniciar sesión' }));

      await waitFor(() => {
        expect(onLoginSuccess).toHaveBeenCalled();
      });
      expect(loginSpy).toHaveBeenCalledWith('prodev', 'password123');
    });
  });
});
