import React, { useState } from 'react';
import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { ErrorBoundary } from '../src/components/ErrorBoundary.jsx';

function ProblemChild({ shouldThrow }) {
  if (shouldThrow) {
    throw new Error('Explosion in child component');
  }
  return <div>Normal Content</div>;
}

function TestContainer() {
  const [hasError, setHasError] = useState(false);
  return (
    <div>
      <button onClick={() => setHasError(true)}>Trigger Error</button>
      <ErrorBoundary onReset={() => setHasError(false)}>
        <ProblemChild shouldThrow={hasError} />
      </ErrorBoundary>
    </div>
  );
}

describe('ErrorBoundary (ADR-0009 / F1.5)', () => {
  it('renders children when no error occurs', () => {
    render(
      <ErrorBoundary>
        <div>Hello World</div>
      </ErrorBoundary>
    );

    expect(screen.getByText('Hello World')).toBeDefined();
  });

  it('catches render errors and displays accessible alert with incident ID', () => {
    // Suppress console.error from React during intentional throw
    const spy = vi.spyOn(console, 'error').mockImplementation(() => {});

    render(
      <ErrorBoundary>
        <ProblemChild shouldThrow={true} />
      </ErrorBoundary>
    );

    expect(screen.getByRole('alert')).toBeDefined();
    expect(screen.getByText('Ha ocurrido un error inesperado')).toBeDefined();
    expect(screen.getByText(/ID de incidente:/)).toBeDefined();
    expect(screen.getByRole('button', { name: /Reintentar/i })).toBeDefined();

    spy.mockRestore();
  });

  it('allows user to recover via retry button after error is resolved', () => {
    const spy = vi.spyOn(console, 'error').mockImplementation(() => {});

    render(<TestContainer />);

    expect(screen.getByText('Normal Content')).toBeDefined();

    // Trigger error
    fireEvent.click(screen.getByText('Trigger Error'));
    expect(screen.getByRole('alert')).toBeDefined();

    // Click retry
    fireEvent.click(screen.getByRole('button', { name: /Reintentar/i }));
    expect(screen.getByText('Normal Content')).toBeDefined();

    spy.mockRestore();
  });
});
