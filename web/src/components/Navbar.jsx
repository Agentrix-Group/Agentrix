import React from 'react';

export function Navbar({ activeTab, onSelectTab, currentUser, onLogout }) {
  return (
    <nav className="navbar">
      <div className="navbar-brand" onClick={() => onSelectTab('home')} style={{ cursor: 'pointer' }}>
        Agentrix Platform
      </div>
      <div className="navbar-links">
        <span
          className={`navbar-link ${activeTab === 'home' ? 'active' : ''}`}
          onClick={() => onSelectTab('home')}
        >
          Dashboard
        </span>
        <span
          className={`navbar-link ${activeTab === 'agents' ? 'active' : ''}`}
          onClick={() => onSelectTab('agents')}
        >
          My Bots
        </span>
        <span
          className={`navbar-link ${activeTab === 'matches' ? 'active' : ''}`}
          onClick={() => onSelectTab('matches')}
        >
          Matches
        </span>
        <span
          className={`navbar-link ${activeTab === 'rankings' ? 'active' : ''}`}
          onClick={() => onSelectTab('rankings')}
        >
          Leaderboard
        </span>
        <span
          className={`navbar-link ${activeTab === 'viewer' ? 'active' : ''}`}
          onClick={() => onSelectTab('viewer')}
        >
          Replay Viewer
        </span>
      </div>
      <div style={{ display: 'flex', alignItems: 'center', gap: '12px' }}>
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
              Sign Out
            </button>
          </>
        ) : (
          <button
            className="btn"
            onClick={() => onSelectTab('auth')}
            style={{ padding: '6px 14px', fontSize: '0.85rem' }}
          >
            Sign In / Register
          </button>
        )}
      </div>
    </nav>
  );
}
