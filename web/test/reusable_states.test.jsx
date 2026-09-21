import React from 'react';
import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { LoadingState } from '../src/components/LoadingState.jsx';
import { ErrorState } from '../src/components/ErrorState.jsx';
import { EmptyState } from '../src/components/EmptyState.jsx';

describe('Reusable UI State Components (Section 5.7)', () => {
  describe('LoadingState', () => {
    it('renders accessible role=status and custom message', () => {
      render(<LoadingState message="Cargando datos del motor..." />);

      const statusEl = screen.getByRole('status');
      expect(statusEl).toBeDefined();
      expect(screen.getByText('Cargando datos del motor...')).toBeDefined();
    });

    it('renders default fallback message when none provided', () => {
      render(<LoadingState />);

      expect(screen.getByRole('status')).toBeDefined();
    });
  });

  describe('ErrorState', () => {
    it('renders accessible role=alert with title and message', () => {
      render(
        <ErrorState
          title="Fallo de conexión"
          message="No se pudo contactar al motor de simulación."
        />
      );

      expect(screen.getByRole('alert')).toBeDefined();
      expect(screen.getByText('Fallo de conexión')).toBeDefined();
      expect(screen.getByText('No se pudo contactar al motor de simulación.')).toBeDefined();
    });

    it('triggers onRetry callback when retry button is clicked', () => {
      const handleRetry = vi.fn();
      render(
        <ErrorState
          message="Error de red"
          onRetry={handleRetry}
          retryLabel="Reintentar operación"
        />
      );

      const retryBtn = screen.getByRole('button', { name: 'Reintentar operación' });
      fireEvent.click(retryBtn);
      expect(handleRetry).toHaveBeenCalledTimes(1);
    });
  });

  describe('EmptyState', () => {
    it('renders empty title and description with action button', () => {
      const handleAction = vi.fn();
      render(
        <EmptyState
          title="Sin elementos registrados"
          description="Aún no hay registros en esta sección."
          actionLabel="Crear nuevo registro"
          onAction={handleAction}
        />
      );

      expect(screen.getByText('Sin elementos registrados')).toBeDefined();
      expect(screen.getByText('Aún no hay registros en esta sección.')).toBeDefined();

      const btn = screen.getByRole('button', { name: 'Crear nuevo registro' });
      fireEvent.click(btn);
      expect(handleAction).toHaveBeenCalledTimes(1);
    });
  });
});
