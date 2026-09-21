import React from 'react';
import { describe, it, expect } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { resolveRoute, Router, Link, useRouter } from '../src/router/Router.jsx';

describe('router', () => {
  it('resolves canonical routes and 404s everything else', () => {
    expect(resolveRoute('/')).toEqual({ route: 'home', params: {} });
    expect(resolveRoute('/matches/abc')).toEqual({ route: 'match', params: { id: 'abc' } });
    expect(resolveRoute('/replays/r1')).toEqual({ route: 'replay', params: { id: 'r1' } });
    expect(resolveRoute('/viewer').route).toBe('notFound');
    expect(resolveRoute('/does/not/exist').route).toBe('notFound');
    expect(resolveRoute('/matches/a/b').route).toBe('notFound');
  });

  it('navigates with Link and exposes query parameters', () => {
    window.history.replaceState({}, '', '/');
    function Where() {
      const { route, query } = useRouter();
      return <p>{route}:{query.contest || ''}</p>;
    }
    render(<Router><Link to="/rankings?contest=c1">go</Link><Where /></Router>);
    fireEvent.click(screen.getByText('go'));
    expect(screen.getByText('rankings:c1')).toBeTruthy();
  });
});
