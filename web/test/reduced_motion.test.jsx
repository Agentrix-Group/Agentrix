import React from 'react';
import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/react';
import { usePrefersReducedMotion } from '../src/hooks/hooks.js';
import { setReducedMotion } from './setup.js';

function Probe() {
  return <p>{usePrefersReducedMotion() ? 'reduced' : 'full'}</p>;
}

describe('reduced motion', () => {
  it('follows the user preference', () => {
    setReducedMotion(true);
    render(<Probe />);
    expect(screen.getByText('reduced')).toBeTruthy();
    setReducedMotion(false);
  });
  it('animates by default', () => {
    render(<Probe />);
    expect(screen.getByText('full')).toBeTruthy();
  });
});
