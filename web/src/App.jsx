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
import { Router, useRouter } from './router/Router.jsx';
import { ErrorBoundary } from './components/ErrorBoundary.jsx';
import { X } from 'lucide-react';

export function AppContent() {
  const { t } = useTranslation(['viewer', 'common']);
  const { route, params, navigate } = useRouter();
  const [currentUser, setCurrentUser] = useState(null);

  const selectedReplayId = params?.id || null;

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
    navigate('/');
  };

  const handleLoginSuccess = (user) => {
    setCurrentUser(user);
    navigate('/agents');
  };

  const handleWatchReplay = (replayId) => {
    if (replayId) {
      navigate(`/replays/${encodeURIComponent(replayId)}`);
    } else {
      navigate('/viewer');
    }
  };

  const handleSelectTab = (tab) => {
    switch (tab) {
      case 'home':
        navigate('/');
        break;
      case 'matches':
        navigate('/matches');
        break;
      case 'rankings':
        navigate('/rankings');
        break;
      case 'agents':
        navigate('/agents');
        break;
      case 'auth':
        navigate('/auth');
        break;
      case 'viewer':
        navigate(selectedReplayId ? `/replays/${encodeURIComponent(selectedReplayId)}` : '/viewer');
        break;
      default:
        navigate('/');
    }
  };

  return (
    <div style={{ minHeight: '100vh', background: 'var(--bg-primary)' }}>
      <Navbar
        activeTab={route}
        onSelectTab={handleSelectTab}
        currentUser={currentUser}
        onLogout={handleLogout}
        hasReplay={Boolean(selectedReplayId || route === 'viewer')}
      />

      <main className="main-content">
        <ErrorBoundary>
          {route === 'home' && (
            <HomePage onWatchReplay={handleWatchReplay} />
          )}
          {route === 'agents' && (
            <AgentsPage currentUser={currentUser} />
          )}
          {route === 'matches' && (
            <MatchesPage onWatchReplay={handleWatchReplay} currentUser={currentUser} />
          )}
          {route === 'rankings' && (
            <RankingsPage />
          )}
          {route === 'viewer' && (
            <div>
              <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '20px' }}>
                <h1>{t('viewer:title')}</h1>
                {selectedReplayId && (
                  <button className="btn" onClick={() => navigate('/viewer')}>
                    <X size={17} aria-hidden="true" /> {t('viewer:clearSelection')}
                  </button>
                )}
              </div>
              <ReplayViewer replayId={selectedReplayId} onBrowseMatches={() => navigate('/matches')} />
            </div>
          )}
          {route === 'auth' && (
            <AuthPage onLoginSuccess={handleLoginSuccess} />
          )}
        </ErrorBoundary>
      </main>
    </div>
  );
}

export function App() {
  return (
    <ErrorBoundary>
      <Router>
        <AppContent />
      </Router>
    </ErrorBoundary>
  );
}

export default App;
