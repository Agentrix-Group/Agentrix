import React, { Suspense, lazy } from 'react';
import { Router, useRouter } from './router/Router.jsx';
import { SessionProvider } from './auth/SessionContext.jsx';
import { ToastProvider } from './components/Toast.jsx';
import { Navbar } from './components/Navbar.jsx';
import { SessionExpiredBanner } from './components/SessionExpiredBanner.jsx';
import { ErrorBoundary } from './components/ErrorBoundary.jsx';
import { ProtectedRoute } from './components/ProtectedRoute.jsx';
import { LoadingState } from './components/States.jsx';

// Route-level code splitting: each page is its own chunk.
const HomePage = lazy(() => import('./pages/HomePage.jsx'));
const ContestsPage = lazy(() => import('./pages/ContestsPage.jsx'));
const ContestPage = lazy(() => import('./pages/ContestPage.jsx'));
const MatchesPage = lazy(() => import('./pages/MatchesPage.jsx'));
const MatchPage = lazy(() => import('./pages/MatchPage.jsx'));
const RankingsPage = lazy(() => import('./pages/RankingsPage.jsx'));
const AgentsPage = lazy(() => import('./pages/AgentsPage.jsx'));
const AdminPage = lazy(() => import('./pages/AdminPage.jsx'));
const AuthPage = lazy(() => import('./pages/AuthPage.jsx'));
const ReplayPage = lazy(() => import('./pages/ReplayPage.jsx'));
const NotFoundPage = lazy(() => import('./pages/NotFoundPage.jsx'));

function Page() {
  const { route, params, query } = useRouter();
  switch (route) {
    case 'home': return <HomePage />;
    case 'contests': return <ContestsPage />;
    case 'contest': return <ContestPage contestId={params.id} />;
    case 'matches': return <MatchesPage contestId={query.contest} />;
    case 'match': return <MatchPage matchId={params.id} />;
    case 'rankings': return <RankingsPage contestId={query.contest} />;
    case 'agents': return <ProtectedRoute capability="agents:read:own"><AgentsPage /></ProtectedRoute>;
    case 'agent': return <ProtectedRoute capability="agents:read:own"><AgentsPage agentId={params.id} /></ProtectedRoute>;
    case 'admin': return <ProtectedRoute capability="admin:access"><AdminPage /></ProtectedRoute>;
    case 'auth': return <AuthPage next={query.next} />;
    case 'replay': return <ReplayPage replayId={params.id} />;
    default: return <NotFoundPage />;
  }
}

export function AppShell() {
  const { path } = useRouter();
  return (
    <div className="app">
      <a className="skip-link" href="#main">Skip to content</a>
      <Navbar />
      <SessionExpiredBanner />
      <main id="main" className="main-content" tabIndex={-1}>
        <ErrorBoundary key={path}>
          <Suspense fallback={<LoadingState />}>
            <div className="route-view" key={path}><Page /></div>
          </Suspense>
        </ErrorBoundary>
      </main>
    </div>
  );
}

export function App() {
  return (
    <ErrorBoundary>
      <ToastProvider>
        <SessionProvider>
          <Router>
            <AppShell />
          </Router>
        </SessionProvider>
      </ToastProvider>
    </ErrorBoundary>
  );
}

export default App;
