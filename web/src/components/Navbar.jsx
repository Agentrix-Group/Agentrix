import React from 'react';
import { useTranslation } from 'react-i18next';
import {
  Bot,
  Home,
  Languages,
  LogIn,
  LogOut,
  Orbit,
  PlaySquare,
  Swords,
  Trophy,
  UserRound,
} from 'lucide-react';

const NAV_ITEMS = [
  { id: 'home', label: 'home', Icon: Home },
  { id: 'agents', label: 'agents', Icon: Bot },
  { id: 'matches', label: 'matches', Icon: Swords },
  { id: 'rankings', label: 'rankings', Icon: Trophy },
  { id: 'viewer', label: 'viewer', Icon: PlaySquare },
];

export function Navbar({ activeTab, onSelectTab, currentUser, onLogout }) {
  const { t, i18n } = useTranslation(['navigation', 'common']);
  const currentLanguage = i18n.language?.startsWith('en') ? 'en' : 'es';

  return (
    <nav className="navbar" aria-label={t('navigation:brand')}>
      <button className="navbar-brand" type="button" onClick={() => onSelectTab('home')}>
        <Orbit size={23} aria-hidden="true" />
        {t('navigation:brand')}
      </button>

      <div className="navbar-links">
        {NAV_ITEMS.map(({ id, label, Icon }) => (
          <button
            key={id}
            type="button"
            className={`navbar-link ${activeTab === id ? 'active' : ''}`}
            onClick={() => onSelectTab(id)}
            aria-current={activeTab === id ? 'page' : undefined}
          >
            <Icon size={16} aria-hidden="true" />
            {t(`navigation:${label}`)}
          </button>
        ))}
      </div>

      <div className="navbar-actions">
        <label className="language-control">
          <Languages size={16} aria-hidden="true" />
          <select
            aria-label={t('navigation:selectLanguage')}
            value={currentLanguage}
            onChange={(event) => i18n.changeLanguage(event.target.value)}
          >
            <option value="es">{t('navigation:languages.es')}</option>
            <option value="en">{t('navigation:languages.en')}</option>
          </select>
        </label>

        {currentUser ? (
          <>
            <span className="current-user"><UserRound size={16} aria-hidden="true" /><strong>{currentUser.username}</strong></span>
            <button className="btn btn-secondary btn-compact" type="button" onClick={onLogout}>
              <LogOut size={16} aria-hidden="true" /> {t('navigation:signOut')}
            </button>
          </>
        ) : (
          <button className="btn btn-compact" type="button" onClick={() => onSelectTab('auth')}>
            <LogIn size={16} aria-hidden="true" /> {t('navigation:signIn')}
          </button>
        )}
      </div>
    </nav>
  );
}
