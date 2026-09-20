import React from 'react';
import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import i18n from '../src/i18n/index.js';
import { RankingsPage } from '../src/pages/RankingsPage.jsx';
import { MatchesPage } from '../src/pages/MatchesPage.jsx';
import { ApiService } from '../src/service/apiService.js';

describe('Error States and Recovery (ADR-0009 / F0.6)', () => {
  beforeEach(async () => {
    await i18n.changeLanguage('es');
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  describe('RankingsPage error handling', () => {
    it('displays error alert and does NOT mask network failure as empty list', async () => {
      vi.spyOn(ApiService, 'listRankings').mockRejectedValueOnce(new Error('Conexión fallida'));

      render(<RankingsPage />);

      const alert = await screen.findByRole('alert');
      expect(alert).toBeDefined();
      expect(screen.getByText('Conexión fallida')).toBeDefined();
      // Retry button is available
      const retryBtn = screen.getByRole('button', { name: 'Reintentar' });
      expect(retryBtn).toBeDefined();
    });

    it('allows retrying after failure and loads ranking table on success', async () => {
      const listSpy = vi.spyOn(ApiService, 'listRankings')
        .mockRejectedValueOnce(new Error('Error de servidor'))
        .mockResolvedValueOnce([
          {
            id: 'rank-1',
            rank: 1,
            agent_id: 'bot-omega',
            user_id: 'player-1',
            score: 1500,
            matches_played: 10,
            wins: 8,
            draws: 1,
            losses: 1,
          },
        ]);

      render(<RankingsPage />);

      expect(await screen.findByRole('alert')).toBeDefined();
      expect(screen.getByText('Error de servidor')).toBeDefined();

      fireEvent.click(screen.getByRole('button', { name: 'Reintentar' }));

      await waitFor(() => {
        expect(screen.queryByRole('alert')).toBeNull();
      });

      expect(screen.getByText('bot-omega')).toBeDefined();
      expect(listSpy).toHaveBeenCalledTimes(2);
    });
  });

  describe('MatchesPage error handling', () => {
    it('displays error alert and retry button when listMatches fails', async () => {
      vi.spyOn(ApiService, 'listMatches').mockRejectedValueOnce(new Error('Servicio de partidas no disponible'));

      render(<MatchesPage currentUser={null} />);

      const alert = await screen.findByRole('alert');
      expect(alert).toBeDefined();
      expect(screen.getByText('Servicio de partidas no disponible')).toBeDefined();

      const retryBtn = screen.getByRole('button', { name: 'Reintentar' });
      expect(retryBtn).toBeDefined();
    });

    it('allows retrying match listing after failure', async () => {
      const listSpy = vi.spyOn(ApiService, 'listMatches')
        .mockRejectedValueOnce(new Error('Fallo de red'))
        .mockResolvedValueOnce([
          {
            id: 'm-99',
            game_id: 'starfighter',
            status: 'running',
            seed: 1234,
          },
        ]);

      render(<MatchesPage currentUser={null} />);

      expect(await screen.findByRole('alert')).toBeDefined();

      fireEvent.click(screen.getByRole('button', { name: 'Reintentar' }));

      await waitFor(() => {
        expect(screen.queryByRole('alert')).toBeNull();
      });

      expect(screen.getByText('Partida #m-99')).toBeDefined();
      expect(listSpy).toHaveBeenCalledTimes(2);
    });

    it('displays inline accessible error alert instead of native alert on execution failure', async () => {
      vi.spyOn(ApiService, 'listMatches').mockResolvedValue([
        { id: 'm-pending-run', game_id: 'starfighter', status: 'pending', seed: 5 },
      ]);
      vi.spyOn(ApiService, 'runMatch').mockRejectedValue(new Error('Sin permisos para ejecutar'));

      render(<MatchesPage currentUser={{ role_id: 'admin' }} canRun={true} />);

      const runBtn = await screen.findByRole('button', { name: 'Ejecutar partida' });
      fireEvent.click(runBtn);

      const alert = await screen.findByRole('alert');
      expect(alert).toBeDefined();
      expect(screen.getByText(/Sin permisos para ejecutar/)).toBeDefined();
    });
  });
});
