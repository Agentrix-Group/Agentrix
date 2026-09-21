import React, { useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Bot, House, Languages, LogIn, LogOut, Menu, Orbit, Trophy, Swords, UserRound, X, Play, CalendarDays, Shield } from 'lucide-react';

export function Navbar({ activeTab, onSelectTab, currentUser, onLogout, hasReplay = false }) {
  const { t, i18n } = useTranslation('navigation');
  const [menuOpen, setMenuOpen] = useState(false);
  const currentLanguage = i18n.language?.startsWith('en') ? 'en' : 'es';

  const selectTab = (tab) => {
    onSelectTab(tab);
    setMenuOpen(false);
  };

  const isAdmin = currentUser && (
    currentUser.role_id === 'admin' ||
    currentUser.role === 'admin' ||
    (Array.isArray(currentUser.capabilities) && currentUser.capabilities.includes('admin:access'))
  );

  const links = [
    { tab: 'home', label: t('home'), Icon: House },
    { tab: 'contests', label: t('contests'), Icon: CalendarDays },
    { tab: 'matches', label: t('matches'), Icon: Swords },
    { tab: 'rankings', label: t('rankings'), Icon: Trophy },
    ...(currentUser ? [{ tab: 'agents', label: t('agents'), Icon: Bot }] : []),
    ...(isAdmin ? [{ tab: 'admin', label: t('admin'), Icon: Shield }] : []),
    ...(hasReplay ? [{ tab: 'viewer', label: t('viewer'), Icon: Play }] : []),
  ];

  return (
    <header className="site-header">
      <nav className="navbar" aria-label={t('mainNavigation')}>
        <button className="navbar-brand" type="button" onClick={() => selectTab('home')}>
          <span className="brand-mark"><Orbit size={23} strokeWidth={1.8} aria-hidden="true" /></span>
          <span>{t('brand')}</span>
        </button>

        <button
          className="navbar-menu-button"
          type="button"
          aria-label={menuOpen ? t('closeMenu') : t('openMenu')}
          aria-controls="navbar-panel"
          aria-expanded={menuOpen}
          onClick={() => setMenuOpen(!menuOpen)}
        >
          {menuOpen ? <X size={22} aria-hidden="true" /> : <Menu size={22} aria-hidden="true" />}
        </button>

        <div className={`navbar-panel ${menuOpen ? 'is-open' : ''}`} id="navbar-panel">
          <div className="navbar-links">
            {links.map(({ tab, label, Icon }) => (
              <button
                key={tab}
                className={`navbar-link ${activeTab === tab ? 'active' : ''}`}
                type="button"
                aria-current={activeTab === tab ? 'page' : undefined}
                onClick={() => selectTab(tab)}
              >
                <Icon size={18} strokeWidth={1.8} aria-hidden="true" /> {label}
              </button>
            ))}
          </div>

          <div className="navbar-actions">
            <label className="language-control">
              <Languages size={17} aria-hidden="true" />
              <span className="sr-only">{t('selectLanguage')}</span>
              <select
                aria-label={t('selectLanguage')}
                value={currentLanguage}
                onChange={(event) => i18n.changeLanguage(event.target.value)}
              >
                <option value="es">{t('languages.es')}</option>
                <option value="en">{t('languages.en')}</option>
              </select>
            </label>

            {currentUser ? (
              <>
                <span className="navbar-user"><UserRound size={17} aria-hidden="true" /> {currentUser.username}</span>
                <button className="btn btn-secondary navbar-auth" type="button" onClick={() => { onLogout(); setMenuOpen(false); }}>
                  <LogOut size={17} aria-hidden="true" /> {t('signOut')}
                </button>
              </>
            ) : (
              <button className="btn navbar-auth" type="button" onClick={() => selectTab('auth')}>
                <LogIn size={17} aria-hidden="true" /> {t('signIn')}
              </button>
            )}
          </div>
        </div>
      </nav>
    </header>
  );
}
