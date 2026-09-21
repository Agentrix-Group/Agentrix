import React from 'react';
import { useTranslation } from 'react-i18next';
import { LogIn, LogOut, Languages } from 'lucide-react';
import { Link, useRouter } from '../router/Router.jsx';
import { useSession } from '../auth/SessionContext.jsx';
import { useToast } from './Toast.jsx';
import { setLanguage } from '../i18n/index.js';

export function Navbar() {
  const { t, i18n } = useTranslation('navigation');
  const { currentUser, isAuthenticated, logout, can } = useSession();
  const { route, navigate } = useRouter();
  const toast = useToast();

  const items = [
    { to: '/', key: 'home', routes: ['home'] },
    { to: '/contests', key: 'contests', routes: ['contests', 'contest'] },
    { to: '/matches', key: 'matches', routes: ['matches', 'match'] },
    { to: '/rankings', key: 'rankings', routes: ['rankings'] },
  ];
  if (can('agents:read:own')) items.push({ to: '/agents', key: 'agents', routes: ['agents', 'agent'] });
  if (can('admin:access')) items.push({ to: '/admin', key: 'admin', routes: ['admin'] });

  const onLogout = async () => {
    try {
      await logout();
      navigate('/');
    } catch {
      toast.error(t('logoutFailed'));
    }
  };

  return (
    <header className="site-header">
      <nav className="navbar" aria-label={t('main')}>
        <Link to="/" className="navbar-brand">Agentrix</Link>
        <ul className="nav-links">
          {items.map((item) => (
            <li key={item.key}>
              <Link to={item.to} className={`nav-link ${item.routes.includes(route) ? 'active' : ''}`}>{t(item.key)}</Link>
            </li>
          ))}
        </ul>
        <div className="nav-actions">
          <button type="button" className="btn btn-ghost" onClick={() => setLanguage(i18n.language === 'es' ? 'en' : 'es')}
            aria-label={t('switchLanguage')}>
            <Languages size={16} aria-hidden="true" /> {i18n.language === 'es' ? 'EN' : 'ES'}
          </button>
          {isAuthenticated ? (
            <>
              <span className="nav-user" title={currentUser.roles.join(', ')}>{currentUser.username}</span>
              <button type="button" className="btn btn-secondary" onClick={onLogout}>
                <LogOut size={16} aria-hidden="true" /> {t('logout')}
              </button>
            </>
          ) : (
            <Link to="/auth" className="btn"><LogIn size={16} aria-hidden="true" /> {t('login')}</Link>
          )}
        </div>
      </nav>
    </header>
  );
}
