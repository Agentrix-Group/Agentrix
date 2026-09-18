import React from 'react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { fireEvent, render, screen } from '@testing-library/react';
import i18n from '../src/i18n/index.js';
import { HomePage } from '../src/pages/HomePage.jsx';
import { ApiService } from '../src/service/apiService.js';

describe('Public contest discovery', () => {
  beforeEach(async () => {
    await i18n.changeLanguage('es');
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it('shows public contest summaries with dates and a meaningful state', async () => {
    vi.spyOn(ApiService, 'listContests').mockResolvedValue([{
      id: 'contest-1',
      name: 'Concurso de prueba',
      description: 'Agentes en una arena.',
      state: 'registration_open',
      starts_at: '2026-10-01T12:00:00Z',
      ends_at: '2026-11-01T12:00:00Z',
    }]);

    render(<HomePage />);

    expect(await screen.findByText('Concurso de prueba')).toBeDefined();
    expect(screen.getByText('Inscripción abierta')).toBeDefined();
    expect(screen.getByText(/Inicio:/)).toBeDefined();
    expect(screen.getByText(/Cierre:/)).toBeDefined();
  });

  it('distinguishes an API failure from an empty public list and can retry', async () => {
    const listContests = vi.spyOn(ApiService, 'listContests')
      .mockRejectedValueOnce(new Error('unavailable'))
      .mockResolvedValueOnce([]);

    render(<HomePage />);

    expect(await screen.findByRole('alert')).toBeDefined();
    expect(screen.queryByText('Aún no hay concursos publicados')).toBeNull();

    fireEvent.click(screen.getByRole('button', { name: 'Reintentar' }));

    expect(await screen.findByText('Aún no hay concursos publicados')).toBeDefined();
    expect(listContests).toHaveBeenCalledTimes(2);
  });
});
