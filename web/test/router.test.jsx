import React from 'react';
import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest';
import { render, screen, fireEvent, act } from '@testing-library/react';
import { Router, useRouter, Link, matchPattern } from '../src/router/Router.jsx';

function RouteDisplay() {
  const { currentPath, route, params, navigate } = useRouter();
  return (
    <div>
      <span data-testid="current-path">{currentPath}</span>
      <span data-testid="current-route">{route}</span>
      <span data-testid="param-id">{params?.id || 'none'}</span>
      <button onClick={() => navigate('/matches')}>Go Matches</button>
      <button onClick={() => navigate('/replays/rep-999')}>Go Replay 999</button>
      <Link to="/rankings">Rankings Link</Link>
    </div>
  );
}

describe('Client Router (ADR-0009 / F0.8)', () => {
  beforeEach(() => {
    window.history.pushState({}, '', '/');
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it('correctly matches static and parameterized patterns', () => {
    expect(matchPattern('/', '/').match).toBe(true);
    expect(matchPattern('/matches', '/matches').match).toBe(true);
    expect(matchPattern('/matches/123', '/matches').match).toBe(false);

    const matchReplay = matchPattern('/replays/rep-abc', '/replays/:id');
    expect(matchReplay.match).toBe(true);
    expect(matchReplay.params.id).toBe('rep-abc');

    const matchMismatch = matchPattern('/replays/rep-abc/extra', '/replays/:id');
    expect(matchMismatch.match).toBe(false);
  });

  it('provides route information based on current window location', () => {
    window.history.pushState({}, '', '/');
    render(
      <Router>
        <RouteDisplay />
      </Router>
    );

    expect(screen.getByTestId('current-path').textContent).toBe('/');
    expect(screen.getByTestId('current-route').textContent).toBe('home');
    expect(screen.getByTestId('param-id').textContent).toBe('none');
  });

  it('navigates to different paths using navigate function and updates URL', () => {
    render(
      <Router>
        <RouteDisplay />
      </Router>
    );

    fireEvent.click(screen.getByText('Go Matches'));
    expect(screen.getByTestId('current-path').textContent).toBe('/matches');
    expect(screen.getByTestId('current-route').textContent).toBe('matches');
    expect(window.location.pathname).toBe('/matches');

    fireEvent.click(screen.getByText('Go Replay 999'));
    expect(screen.getByTestId('current-path').textContent).toBe('/replays/rep-999');
    expect(screen.getByTestId('current-route').textContent).toBe('viewer');
    expect(screen.getByTestId('param-id').textContent).toBe('rep-999');
    expect(window.location.pathname).toBe('/replays/rep-999');
  });

  it('handles Link component click and updates active state with aria-current', () => {
    render(
      <Router>
        <RouteDisplay />
      </Router>
    );

    const link = screen.getByText('Rankings Link');
    expect(link.getAttribute('aria-current')).toBeNull();

    fireEvent.click(link);
    expect(screen.getByTestId('current-path').textContent).toBe('/rankings');
    expect(screen.getByTestId('current-route').textContent).toBe('rankings');
    expect(link.getAttribute('aria-current')).toBe('page');
  });

  it('synchronizes with popstate browser back/forward events', async () => {
    render(
      <Router>
        <RouteDisplay />
      </Router>
    );

    fireEvent.click(screen.getByText('Go Matches'));
    expect(screen.getByTestId('current-path').textContent).toBe('/matches');

    // Simulate browser back button
    await act(async () => {
      window.history.pushState({}, '', '/');
      window.dispatchEvent(new PopStateEvent('popstate'));
    });

    expect(screen.getByTestId('current-path').textContent).toBe('/');
    expect(screen.getByTestId('current-route').textContent).toBe('home');
  });
});
