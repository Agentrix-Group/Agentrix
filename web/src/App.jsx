import React, { useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Navbar } from './components/Navbar.jsx';
import { HomePage } from './pages/HomePage.jsx';
import { MatchesPage } from './pages/MatchesPage.jsx';
import { RankingsPage } from './pages/RankingsPage.jsx';
import { AgentsPage } from './pages/AgentsPage.jsx';
import { AuthPage } from './pages/AuthPage.jsx';
import { ReplayViewer } from './viewer/ReplayViewer.jsx';
import { ApiService } from './service/apiService.js';

export function App() {
  const { t } = useTranslation(['viewer', 'common']);
  const [activeTab, setActiveTab] = useState('home');
  const [currentUser, setCurrentUser] = useState(null);
  const [selectedReplayId, setSelectedReplayId] = useState(null);

  useEffect(() => {
    const token = localStorage.getItem('agentrix_token');
    if (token) {
      ApiService.getCurrentUser()
        .then((user) => setCurrentUser(user))
        .catch(() => {
          localStorage.removeItem('agentrix_token');
          setCurrentUser(null);
        });
    }
  }, []);

  const handleLogout = () => {
    ApiService.logout();
    setCurrentUser(null);
    setActiveTab('home');
  };

  const handleLoginSuccess = (user) => {
    setCurrentUser(user);
    setActiveTab('agents');
  };

  const handleWatchReplay = (replayId) => {
    setSelectedReplayId(replayId);
    setActiveTab('viewer');
  };

  return (
    <div style={{ minHeight: '100vh', background: 'var(--bg-primary)' }}>
      <Navbar
        activeTab={activeTab}
        onSelectTab={setActiveTab}
        currentUser={currentUser}
        onLogout={handleLogout}
        hasReplay={Boolean(selectedReplayId)}
      />

      <main className="main-content">
        {activeTab === 'home' && (
          <HomePage />
        )}
        {activeTab === 'agents' && (
          <AgentsPage currentUser={currentUser} />
        )}
        {activeTab === 'matches' && (
          <MatchesPage onWatchReplay={handleWatchReplay} currentUser={currentUser} />
        )}
        {activeTab === 'rankings' && (
          <RankingsPage />
        )}
        {activeTab === 'viewer' && (
          <div>
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '20px' }}>
              <h1>{t('viewer:title')}</h1>
              {selectedReplayId && (
                <button className="btn" onClick={() => setSelectedReplayId(null)}>
                  {t('viewer:clearSelection')}
                </button>
              )}
            </div>
            <ReplayViewer replayId={selectedReplayId} />
          </div>
        )}
        {activeTab === 'auth' && (
          <AuthPage onLoginSuccess={handleLoginSuccess} />
        )}
      </main>
    </div>
  );
}

export default App;
