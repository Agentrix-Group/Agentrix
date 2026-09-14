import React from 'react';

export function Navbar({ activeTab, onSelectTab, currentUser }) {
  return (
    <nav className="navbar">
      <div className="navbar-brand">Agentrix Platform</div>
      <div className="navbar-links">
        <span 
          className={`navbar-link ${activeTab === 'home' ? 'active' : ''}`}
          onClick={() => onSelectTab('home')}
        >
          Dashboard
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
      <div>
        {currentUser ? (
          <span>{currentUser.username}</span>
        ) : (
          <span className="badge badge-pending">Guest</span>
        )}
      </div>
    </nav>
  );
}
