import React from 'react';
import { useTranslation } from 'react-i18next';

export function Navbar({ activeTab, onSelectTab, currentUser, onLogout }) {
  const { t, i18n } = useTranslation(['navigation', 'common']);

  const currentLanguage = i18n.language?.startsWith('en') ? 'en' : 'es';

  const handleLanguageChange = (e) => {
    const newLang = e.target.value;
    i18n.changeLanguage(newLang);
  };

  return (
    <nav className="navbar">
      <div className="navbar-brand" onClick={() => onSelectTab('home')} style={{ cursor: 'pointer' }}>
        {t('navigation:brand')}
      </div>
      <div className="navbar-links">
        <span
          className={`navbar-link ${activeTab === 'home' ? 'active' : ''}`}
          onClick={() => onSelectTab('home')}
        >
          {t('navigation:home')}
        </span>
        <span
          className={`navbar-link ${activeTab === 'agents' ? 'active' : ''}`}
          onClick={() => onSelectTab('agents')}
        >
          {t('navigation:agents')}
        </span>
        <span
          className={`navbar-link ${activeTab === 'matches' ? 'active' : ''}`}
          onClick={() => onSelectTab('matches')}
        >
          {t('navigation:matches')}
        </span>
        <span
          className={`navbar-link ${activeTab === 'rankings' ? 'active' : ''}`}
          onClick={() => onSelectTab('rankings')}
        >
          {t('navigation:rankings')}
        </span>
        <span
          className={`navbar-link ${activeTab === 'viewer' ? 'active' : ''}`}
          onClick={() => onSelectTab('viewer')}
        >
          {t('navigation:viewer')}
        </span>
      </div>

      <div style={{ display: 'flex', alignItems: 'center', gap: '12px' }}>
        {/* Language selector */}
        <select
          aria-label={t('navigation:selectLanguage')}
          value={currentLanguage}
          onChange={handleLanguageChange}
          style={{
            padding: '4px 8px',
            fontSize: '0.85rem',
            borderRadius: '6px',
            border: '1px solid var(--border)',
            background: 'var(--bg-subtle)',
            color: 'var(--text-primary)',
            cursor: 'pointer',
          }}
        >
          <option value="es">{t('navigation:languages.es')}</option>
          <option value="en">{t('navigation:languages.en')}</option>
        </select>

        {currentUser ? (
          <>
            <span style={{ fontSize: '0.9rem', color: 'var(--text-secondary)' }}>
              👤 <strong>{currentUser.username}</strong>
            </span>
            <button
              className="btn btn-secondary"
              onClick={onLogout}
              style={{ padding: '5px 12px', fontSize: '0.8rem' }}
            >
              {t('navigation:signOut')}
            </button>
          </>
        ) : (
          <button
            className="btn"
            onClick={() => onSelectTab('auth')}
            style={{ padding: '6px 14px', fontSize: '0.85rem' }}
          >
            {t('navigation:signIn')}
          </button>
        )}
      </div>
    </nav>
  );
}
