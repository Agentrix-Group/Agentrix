import React from 'react';
import { useTranslation } from 'react-i18next';
import { Navbar } from './components/Navbar.jsx';
import { HomePage } from './pages/HomePage.jsx';
import { MatchesPage } from './pages/MatchesPage.jsx';
import { RankingsPage } from './pages/RankingsPage.jsx';
import { AgentsPage } from './pages/AgentsPage.jsx';
import { AuthPage } from './pages/AuthPage.jsx';
import { ReplayViewer } from './viewer/ReplayViewer.jsx';
import { Router, useRouter } from './router/Router.jsx';
import { ErrorBoundary } from './components/ErrorBoundary.jsx';
import { SessionProvider, useSession } from './auth/SessionContext.jsx';
import { SessionExpiredBanner } from './components/SessionExpiredBanner.jsx';
import { ProtectedRoute } from './components/ProtectedRoute.jsx';
import { X } from 'lucide-react';

export function AppContent() {
  const { t } = useTranslation(['viewer', 'common']);
  const { route, params, navigate } = useRouter();
  const { currentUser, logout } = useSession();

  const selectedReplayId = params?.id || null;

  const handleLogout = () => {
    logout();
    navigate('/');
  };

  const handleLoginSuccess = (user, redirectTarget) => {
    if (redirectTarget) {
      navigate(redirectTarget);
    } else {
      navigate('/agents');
    }
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

      <SessionExpiredBanner />

      <main className="main-content">
        <ErrorBoundary>
          {route === 'home' && (
            <HomePage onWatchReplay={handleWatchReplay} />
          )}
          {route === 'agents' && (
            <ProtectedRoute>
              <AgentsPage currentUser={currentUser} />
            </ProtectedRoute>
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
      <SessionProvider>
        <Router>
          <AppContent />
        </Router>
      </SessionProvider>
    </ErrorBoundary>
  );
}

export default App;
