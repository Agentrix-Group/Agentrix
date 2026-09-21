import React from 'react';
import { describe, it, expect } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import RankingsPage from '../src/pages/RankingsPage.jsx';
import { Router } from '../src/router/Router.jsx';
import { SessionProvider } from '../src/auth/SessionContext.jsx';
import { ToastProvider } from '../src/components/Toast.jsx';
import { mockFetch, jsonResponse, errorResponse } from './helpers.js';

const contests = { items: [{ id: 'c1', name: 'Copa', state: 'running' }] };
const rankings = {
  contest_id: 'c1', stale: false, applied_runs_count: 1, applied_runs_digest: 'f'.repeat(64),
  rankings: [{ entry_id: 'e1', rank: 1, agent_name: 'Alpha', username: 'alice', points: 3, matches_played: 1, wins: 1, draws: 0, losses: 0, disqualifications: 0, score_for: 5, score_against: 1 }],
};
const organizer = { access_token: 't', user: { id: 'o', username: 'org', roles: ['organizer'], capabilities: ['rankings:publish', 'rankings:view'] } };

function renderPage() {
  return render(<Router><SessionProvider><ToastProvider><RankingsPage /></ToastProvider></SessionProvider></Router>);
}

describe('RankingsPage', () => {
  it('shows a retryable error instead of an empty table when the API fails', async () => {
    let attempts = 0;
    mockFetch({
      'POST /auth/refresh': errorResponse(401, 'invalid_refresh_token'),
      'GET /contests': jsonResponse(200, contests),
      'GET /contests/c1/rankings': () => { attempts += 1; return attempts === 1 ? errorResponse(503, 'unavailable') : jsonResponse(200, rankings); },
      'GET /contests/c1/rankings/snapshots': jsonResponse(200, { items: [] }),
    });
    renderPage();
    
    const retry = await screen.findByRole('button', { name: /reintentar/i });
    expect(screen.queryByText('Sin participantes clasificados.')).toBeNull();
    fireEvent.click(retry);
    expect(await screen.findByText('Alpha')).toBeTruthy();
  });

  it('only offers publishing to principals with rankings:publish', async () => {
    mockFetch({
      'POST /auth/refresh': errorResponse(401, 'invalid_refresh_token'),
      'GET /contests': jsonResponse(200, contests),
      'GET /contests/c1/rankings': jsonResponse(200, rankings),
      'GET /contests/c1/rankings/snapshots': jsonResponse(200, { items: [] }),
    });
    const { unmount } = renderPage();
    await screen.findByText('Alpha');
    expect(screen.queryByText(/publicar snapshot/i)).toBeNull();
    unmount();

    mockFetch({
      'POST /auth/refresh': jsonResponse(200, organizer),
      'GET /contests': jsonResponse(200, contests),
      'GET /contests/c1/rankings': jsonResponse(200, rankings),
      'GET /contests/c1/rankings/snapshots': jsonResponse(200, { items: [] }),
    });
    renderPage();
    await waitFor(() => expect(screen.getByText(/publicar snapshot/i)).toBeTruthy());
  });
});
