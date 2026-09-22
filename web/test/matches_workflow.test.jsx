import React from 'react';
import { describe, it, expect, beforeEach, vi, afterEach } from 'vitest';
import { render, screen, fireEvent, act, waitFor } from '@testing-library/react';
import i18n from '../src/i18n/index.js';
import { MatchesPage } from '../src/pages/MatchesPage.jsx';
import { CreateMatchModal, validParticipants } from '../src/components/CreateMatchModal.jsx';
import { MatchCard } from '../src/components/MatchCard.jsx';
import { ApiService } from '../src/service/apiService.js';

describe('Sprint FQ-3: Match Creation, Configuration & Live Run Workflow', () => {
  beforeEach(async () => {
    localStorage.clear();
    await act(async () => {
      await i18n.changeLanguage('es');
    });
    vi.restoreAllMocks();
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  describe('CreateMatchModal Wizard', () => {
    const mockContests = [
      { id: 'c-1', name: 'Torneo Apertura 2026' },
      { id: 'c-2', name: 'Liga Estelar' },
    ];

    const mockSubmissions = [
      { id: 'sub-alpha', version: 1, language: 'python' },
      { id: 'sub-beta', version: 2, language: 'python' },
    ];

    it('renders modal with Starfighter game locked and loads contests and submissions', async () => {
      vi.spyOn(ApiService, 'listContests').mockResolvedValue(mockContests);
      vi.spyOn(ApiService, 'listSubmissions').mockResolvedValue(mockSubmissions);

      await act(async () => {
        render(<CreateMatchModal isOpen={true} onClose={vi.fn()} onMatchCreated={vi.fn()} />);
      });

      expect(screen.getByText('Crear nueva partida de Starfighter')).toBeDefined();
      expect(screen.getByText('Starfighter (Oficial)')).toBeDefined();
      expect(screen.getByText('Torneo Apertura 2026')).toBeDefined();

      await waitFor(() => {
        expect(ApiService.listContests).toHaveBeenCalled();
        expect(ApiService.listSubmissions).toHaveBeenCalled();
      });
    });

    it('allows randomizing seed with random button', async () => {
      vi.spyOn(ApiService, 'listContests').mockResolvedValue([]);
      vi.spyOn(ApiService, 'listSubmissions').mockResolvedValue([]);

      await act(async () => {
        render(<CreateMatchModal isOpen={true} onClose={vi.fn()} onMatchCreated={vi.fn()} />);
      });

      const seedInput = screen.getByLabelText(/Semilla determinista/i);
      const initialVal = seedInput.value;

      const randomBtn = screen.getByRole('button', { name: /Aleatoria/i });
      fireEvent.click(randomBtn);

      // Value should still be a valid number string
      expect(Number(seedInput.value)).toBeGreaterThan(0);
    });

    it('submits scheduled match payload and closes modal on success', async () => {
      vi.spyOn(ApiService, 'listContests').mockResolvedValue(mockContests);
      vi.spyOn(ApiService, 'listSubmissions').mockResolvedValue(mockSubmissions);
      vi.spyOn(ApiService, 'scheduleMatch').mockResolvedValue({
        http_status_code: 201,
        message: 'Match m-test-123 scheduled successfully',
      });
      vi.spyOn(ApiService, 'runMatch').mockResolvedValue({
        http_status_code: 202,
        message: 'Match queued',
      });

      const onClose = vi.fn();
      const onCreated = vi.fn();

      await act(async () => {
        render(
          <CreateMatchModal
            isOpen={true}
            onClose={onClose}
            onMatchCreated={onCreated}
          />
        );
      });

      await waitFor(() => {
        expect(screen.getByRole('button', { name: /Crear partida/i })).toBeDefined();
      });

      // Submit form
      await act(async () => {
        fireEvent.click(screen.getByRole('button', { name: /Crear partida/i }));
      });

      await waitFor(() => {
        expect(ApiService.scheduleMatch).toHaveBeenCalledWith({
          contest_id: undefined,
          game_id: 'starfighter',
          submission_ids: ['sub-alpha', 'sub-beta'],
          seed: expect.any(Number),
        });
        expect(ApiService.runMatch).toHaveBeenCalledWith('m-test-123');
        expect(onCreated).toHaveBeenCalled();
        expect(onClose).toHaveBeenCalled();
      });
    });

    it('displays error alert when scheduleMatch fails', async () => {
      vi.spyOn(ApiService, 'listContests').mockResolvedValue([]);
      vi.spyOn(ApiService, 'listSubmissions').mockResolvedValue(mockSubmissions);
      vi.spyOn(ApiService, 'scheduleMatch').mockRejectedValue(new Error('Contest is closed'));

      await act(async () => {
        render(<CreateMatchModal isOpen={true} onClose={vi.fn()} onMatchCreated={vi.fn()} />);
      });

      await act(async () => {
        fireEvent.click(screen.getByRole('button', { name: /Crear partida/i }));
      });

      await waitFor(() => {
        expect(screen.getByRole('alert').textContent).toContain('Contest is closed');
      });
    });

    it('allows manual submission IDs entry when toggled', async () => {
      vi.spyOn(ApiService, 'listContests').mockResolvedValue([]);
      vi.spyOn(ApiService, 'listSubmissions').mockResolvedValue(mockSubmissions);
      vi.spyOn(ApiService, 'scheduleMatch').mockResolvedValue({ message: 'OK' });

      await act(async () => {
        render(<CreateMatchModal isOpen={true} onClose={vi.fn()} onMatchCreated={vi.fn()} />);
      });

      const toggleBtn = screen.getByRole('button', { name: /Ingresar IDs manuales/i });
      fireEvent.click(toggleBtn);

      const p1Input = screen.getByPlaceholderText(/ID de submission o bot 1/i);
      const p2Input = screen.getByPlaceholderText(/ID de submission o bot 2/i);

      fireEvent.change(p1Input, { target: { value: 'custom-sub-1' } });
      fireEvent.change(p2Input, { target: { value: 'custom-sub-2' } });

      await act(async () => {
        fireEvent.click(screen.getByRole('button', { name: /Crear partida/i }));
      });

      await waitFor(() => {
        expect(ApiService.scheduleMatch).toHaveBeenCalledWith(
          expect.objectContaining({
            submission_ids: ['custom-sub-1', 'custom-sub-2'],
          })
        );
      });
    });
  });

  describe('CreateMatchModal free-for-all (ADR-0013)', () => {
    const fiveSubmissions = ['s1', 's2', 's3', 's4', 's5'].map((id, i) => ({
      id, version: i + 1, language: 'python',
    }));

    const renderModal = async () => {
      vi.spyOn(ApiService, 'listContests').mockResolvedValue([]);
      vi.spyOn(ApiService, 'listSubmissions').mockResolvedValue(fiveSubmissions);
      vi.spyOn(ApiService, 'scheduleMatch').mockResolvedValue({ message: 'OK' });
      await act(async () => {
        render(<CreateMatchModal isOpen={true} onClose={vi.fn()} onMatchCreated={vi.fn()} />);
      });
      await waitFor(() => expect(screen.getByLabelText(/Bot Jugador 2/i).value).toBe('s2'));
    };

    it('adds players up to five and submits all of them', async () => {
      await renderModal();
      const addButton = screen.getByRole('button', { name: /Añadir jugador/i });
      for (let n = 3; n <= 5; n += 1) {
        fireEvent.click(addButton);
        fireEvent.change(screen.getByLabelText(new RegExp(`Bot Jugador ${n}`, 'i')), {
          target: { value: `s${n}` },
        });
      }
      expect(addButton.disabled).toBe(true);
      expect(screen.getByText(/Participantes \(5;/i)).toBeDefined();

      await act(async () => {
        fireEvent.click(screen.getByRole('button', { name: /Crear partida/i }));
      });
      await waitFor(() => {
        expect(ApiService.scheduleMatch).toHaveBeenCalledWith(
          expect.objectContaining({ submission_ids: ['s1', 's2', 's3', 's4', 's5'] }),
        );
      });
    });

    it('rejects the same submission in two slots without calling the API', async () => {
      await renderModal();
      fireEvent.change(screen.getByLabelText(/Bot Jugador 2/i), { target: { value: 's1' } });
      await act(async () => {
        fireEvent.click(screen.getByRole('button', { name: /Crear partida/i }));
      });
      expect(screen.getByRole('alert').textContent).toContain('submissions distintas');
      expect(ApiService.scheduleMatch).not.toHaveBeenCalled();
    });

    it('removes a player but never goes below two', async () => {
      await renderModal();
      expect(screen.queryByRole('button', { name: /Quitar jugador/i })).toBeNull();
      fireEvent.click(screen.getByRole('button', { name: /Añadir jugador/i }));
      fireEvent.click(screen.getByRole('button', { name: /Quitar jugador 3/i }));
      expect(screen.queryByLabelText(/Bot Jugador 3/i)).toBeNull();
      expect(screen.queryByRole('button', { name: /Quitar jugador/i })).toBeNull();
    });

    it('validates participant lists', () => {
      expect(validParticipants([' a ', 'b'])).toEqual(['a', 'b']);
      expect(validParticipants(['a', 'b', 'c', 'd', 'e'])).toHaveLength(5);
      expect(validParticipants(['a'])).toBeNull();
      expect(validParticipants(['a', 'b', 'c', 'd', 'e', 'f'])).toBeNull();
      expect(validParticipants(['a', 'a'])).toBeNull();
      expect(validParticipants(['a', ' '])).toBeNull();
    });
  });

  describe('MatchCard Interaction & Live Indicators', () => {
    it('renders pending match with Run button when user has permission', () => {
      const match = {
        id: 'match-10101010-pending',
        game_id: 'starfighter',
        seed: 42,
        status: 'pending',
      };
      const onRun = vi.fn();

      render(<MatchCard match={match} canRun={true} onTriggerRun={onRun} />);

      expect(screen.getByText('Partida #match-10')).toBeDefined();
      expect(screen.getByText('Pendiente')).toBeDefined();
      const runBtn = screen.getByRole('button', { name: /Ejecutar partida/i });
      expect(runBtn).toBeDefined();

      fireEvent.click(runBtn);
      expect(onRun).toHaveBeenCalledWith(match.id);
    });

    it('renders running match with live simulation indicator', () => {
      const match = {
        id: 'match-20202020-running',
        game_id: 'starfighter',
        seed: 777,
        status: 'running',
      };

      render(<MatchCard match={match} canRun={true} />);

      expect(screen.getByText('En ejecución')).toBeDefined();
      expect(screen.getByText(/Simulación en curso por el motor Bevy/i)).toBeDefined();
    });

    it('renders finished match with results and replay link', () => {
      const match = {
        id: 'match-30303030-finished',
        game_id: 'starfighter',
        seed: 12345,
        status: 'finished',
        replay_id: 'rep-999',
        results: [
          { id: 'r1', rank: 1, submission_id: 'bot-winner', score: 100 },
          { id: 'r2', rank: 2, submission_id: 'bot-loser', score: 25 },
        ],
      };
      const onReplay = vi.fn();

      render(<MatchCard match={match} onWatchReplay={onReplay} />);

      expect(screen.getByText('Finalizada')).toBeDefined();
      expect(screen.getByText(/Puesto 1: bot-winner/i)).toBeDefined();
      expect(screen.getByText(/Puesto 2: bot-loser/i)).toBeDefined();

      const replayBtn = screen.getByRole('button', { name: /Ver repetición/i });
      fireEvent.click(replayBtn);
      expect(onReplay).toHaveBeenCalledWith('rep-999');
    });

    it('disables button and shows spinner while isExecuting is true', () => {
      const match = {
        id: 'match-40404040-pending',
        game_id: 'starfighter',
        seed: 42,
        status: 'pending',
      };

      render(<MatchCard match={match} canRun={true} isExecuting={true} onTriggerRun={vi.fn()} />);

      const runBtn = screen.getByRole('button', { name: /Ejecutando.../i });
      expect(runBtn.disabled).toBe(true);
    });
  });

  describe('MatchesPage End-to-End Workflow', () => {
    it('shows "Nueva partida" button only for users with execution capabilities', async () => {
      vi.spyOn(ApiService, 'listMatches').mockResolvedValue([]);

      // Unauthorized user
      const { unmount } = render(
        <MatchesPage currentUser={{ username: 'user1', role: 'contestant', capabilities: [] }} />
      );
      await waitFor(() => {
        expect(screen.queryByRole('button', { name: /Nueva partida/i })).toBeNull();
      });
      unmount();

      // Authorized user (admin)
      render(
        <MatchesPage currentUser={{ username: 'admin1', role: 'admin', capabilities: ['admin'] }} />
      );
      await waitFor(() => {
        expect(screen.getByRole('button', { name: /Nueva partida/i })).toBeDefined();
      });
    });

    it('opens wizard modal when clicking "Nueva partida" and refreshes list on creation', async () => {
      vi.spyOn(ApiService, 'listMatches').mockResolvedValue([]);
      vi.spyOn(ApiService, 'listContests').mockResolvedValue([]);
      vi.spyOn(ApiService, 'listSubmissions').mockResolvedValue([]);
      vi.spyOn(ApiService, 'scheduleMatch').mockResolvedValue({
        message: 'Match m-new-1 scheduled successfully',
      });

      render(<MatchesPage canRun={true} />);

      await waitFor(() => {
        expect(screen.getByRole('button', { name: /Nueva partida/i })).toBeDefined();
      });

      fireEvent.click(screen.getByRole('button', { name: /Nueva partida/i }));

      expect(screen.getByText('Crear nueva partida de Starfighter')).toBeDefined();
    });

    it('triggers match execution and handles error alert if run fails', async () => {
      const mockMatches = [
        {
          id: 'match-pending-fail',
          game_id: 'starfighter',
          seed: 100,
          status: 'pending',
        },
      ];
      vi.spyOn(ApiService, 'listMatches').mockResolvedValue(mockMatches);
      vi.spyOn(ApiService, 'runMatch').mockRejectedValue(new Error('Execution worker unavailable'));

      render(<MatchesPage canRun={true} />);

      await waitFor(() => {
        expect(screen.getByRole('button', { name: /Ejecutar partida/i })).toBeDefined();
      });

      await act(async () => {
        fireEvent.click(screen.getByRole('button', { name: /Ejecutar partida/i }));
      });

      await waitFor(() => {
        expect(screen.getByRole('alert').textContent).toContain('Execution worker unavailable');
      });

      // Dismiss alert
      const closeAlertBtn = screen.getByRole('button', { name: /Cerrar/i });
      fireEvent.click(closeAlertBtn);
      expect(screen.queryByRole('alert')).toBeNull();
    });

    it('filters matches according to filter buttons', async () => {
      const mockMatches = [
        { id: 'm-pending', game_id: 'starfighter', seed: 1, status: 'pending' },
        { id: 'm-running', game_id: 'starfighter', seed: 2, status: 'running' },
        { id: 'm-finished', game_id: 'starfighter', seed: 3, status: 'finished' },
      ];
      vi.spyOn(ApiService, 'listMatches').mockResolvedValue(mockMatches);

      render(<MatchesPage canRun={true} />);

      await waitFor(() => {
        expect(screen.getByText('Partida #m-pendin')).toBeDefined();
        expect(screen.getByText('Partida #m-runnin')).toBeDefined();
        expect(screen.getByText('Partida #m-finish')).toBeDefined();
      });

      // Click "En curso" filter
      fireEvent.click(screen.getByRole('button', { name: /En curso/i }));

      expect(screen.queryByText('Partida #m-pendin')).toBeNull();
      expect(screen.getByText('Partida #m-runnin')).toBeDefined();
      expect(screen.queryByText('Partida #m-finish')).toBeNull();

      // Click "Finalizadas" filter
      fireEvent.click(screen.getByRole('button', { name: /Finalizadas/i }));

      expect(screen.queryByText('Partida #m-runnin')).toBeNull();
      expect(screen.getByText('Partida #m-finish')).toBeDefined();
    });
  });
});
