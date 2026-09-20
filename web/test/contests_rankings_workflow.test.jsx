import React from 'react';
import { describe, it, expect, beforeEach, vi, afterEach } from 'vitest';
import { render, screen, fireEvent, act, waitFor } from '@testing-library/react';
import i18n from '../src/i18n/index.js';
import { EnrollAgentModal } from '../src/components/EnrollAgentModal.jsx';
import { HomePage } from '../src/pages/HomePage.jsx';
import { RankingsPage } from '../src/pages/RankingsPage.jsx';
import { ApiService } from '../src/service/apiService.js';

describe('Sprint FQ-5: Tournament Experience & Real-Time Rankings', () => {
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

  describe('EnrollAgentModal Guided Enrollment', () => {
    const mockContest = {
      id: 'contest-tournament-2026',
      name: 'Gran Torneo Apertura 2026',
      description: 'Competencia estelar para agentes universitarios.',
      state: 'registration_open',
    };

    const mockAgents = [
      { id: 'agent-falcon', name: 'Millennium Falcon', game_id: 'starfighter' },
      { id: 'agent-viper', name: 'Star Viper', game_id: 'starfighter' },
    ];

    it('loads agents and blocks submission if the agent has no ready version', async () => {
      vi.spyOn(ApiService, 'listAgents').mockResolvedValue(mockAgents);
      // Falcon only has validating/rejected versions
      vi.spyOn(ApiService, 'listSubmissions').mockResolvedValue([
        { id: 'sub-1', version: 1, status: 'rejected' },
        { id: 'sub-2', version: 2, status: 'validating' },
      ]);

      render(
        <EnrollAgentModal
          isOpen={true}
          contest={mockContest}
          onClose={vi.fn()}
          onEnrolled={vi.fn()}
        />
      );

      await waitFor(() => {
        expect(screen.getByText('Gran Torneo Apertura 2026')).toBeDefined();
      });

      // Shows warning that no version is ready
      await waitFor(() => {
        expect(screen.getByText(/no tiene una versión lista para competir/i)).toBeDefined();
      });

      const submitBtn = screen.getByRole('button', { name: /Confirmar inscripción/i });
      expect(submitBtn.disabled).toBe(true);
    });

    it('enables submission when selected agent has a ready version and submits successfully', async () => {
      vi.spyOn(ApiService, 'listAgents').mockResolvedValue(mockAgents);
      vi.spyOn(ApiService, 'listSubmissions').mockResolvedValue([
        { id: 'sub-ready-1', version: 1, status: 'ready', language: 'python' },
      ]);
      vi.spyOn(ApiService, 'enrollAgent').mockResolvedValue({
        ranking: { id: 'rank-1', contest_id: mockContest.id, agent_id: 'agent-falcon' },
      });

      const onEnrolled = vi.fn();
      const onClose = vi.fn();

      render(
        <EnrollAgentModal
          isOpen={true}
          contest={mockContest}
          onClose={onClose}
          onEnrolled={onEnrolled}
        />
      );

      await waitFor(() => {
        expect(screen.getByText(/Versión validada disponible: v1/i)).toBeDefined();
      });

      const submitBtn = screen.getByRole('button', { name: /Confirmar inscripción/i });
      expect(submitBtn.disabled).toBe(false);

      await act(async () => {
        fireEvent.click(submitBtn);
      });

      await waitFor(() => {
        expect(ApiService.enrollAgent).toHaveBeenCalledWith(mockContest.id, 'agent-falcon');
        expect(onEnrolled).toHaveBeenCalled();
        expect(onClose).toHaveBeenCalled();
      });
    });

    it('handles 409 conflict when agent is already enrolled', async () => {
      vi.spyOn(ApiService, 'listAgents').mockResolvedValue(mockAgents);
      vi.spyOn(ApiService, 'listSubmissions').mockResolvedValue([
        { id: 'sub-ready-1', version: 1, status: 'ready', language: 'python' },
      ]);
      const conflictError = new Error('agent already enrolled in contest');
      conflictError.status = 409;
      vi.spyOn(ApiService, 'enrollAgent').mockRejectedValue(conflictError);

      render(
        <EnrollAgentModal
          isOpen={true}
          contest={mockContest}
          onClose={vi.fn()}
          onEnrolled={vi.fn()}
        />
      );

      await waitFor(() => {
        expect(screen.getByText(/Versión validada disponible/i)).toBeDefined();
      });

      const submitBtn = screen.getByRole('button', { name: /Confirmar inscripción/i });
      await act(async () => {
        fireEvent.click(submitBtn);
      });

      await waitFor(() => {
        expect(screen.getByRole('alert').textContent).toContain('ya se encuentra inscrito');
      });
    });
  });

  describe('HomePage Tournament Experience', () => {
    it('shows "Inscribir agente" button on contests for authenticated users', async () => {
      const contests = [
        {
          id: 'c-open',
          name: 'Liga de Otoño',
          description: 'Torneo regular',
          state: 'registration_open',
          starts_at: '2026-10-01T00:00:00Z',
          ends_at: '2026-10-15T00:00:00Z',
        },
      ];

      vi.spyOn(ApiService, 'listContests').mockResolvedValue(contests);
      vi.spyOn(ApiService, 'listMatches').mockResolvedValue([]);
      vi.spyOn(ApiService, 'listAgents').mockResolvedValue([]);

      const user = { username: 'testuser', role: 'contestant' };
      render(<HomePage currentUser={user} onSelectContestRankings={vi.fn()} />);

      await waitFor(() => {
        expect(screen.getByText('Liga de Otoño')).toBeDefined();
        expect(screen.getByRole('button', { name: /Inscribir agente/i })).toBeDefined();
        expect(screen.getByRole('button', { name: /Ver clasificación/i })).toBeDefined();
      });

      // Clicking opens modal
      fireEvent.click(screen.getByRole('button', { name: /Inscribir agente/i }));
      expect(screen.getByText('Inscribir agente en el concurso')).toBeDefined();
    });
  });

  describe('RankingsPage Real-Time Leaderboard & Filters', () => {
    const mockContests = [
      { id: 'c-1', name: 'Torneo Apertura', state: 'finished' },
      { id: 'c-live', name: 'Copa Universitaria', state: 'in_progress' },
    ];

    const mockRankings = [
      {
        id: 'r-1',
        rank: 1,
        agent_id: 'bot-champion',
        participant_id: 'user-alice',
        score: 1500,
        matches_played: 10,
        wins: 9,
        draws: 1,
        losses: 0,
      },
      {
        id: 'r-2',
        rank: 2,
        agent_id: 'bot-challenger',
        participant_id: 'user-bob',
        score: 1200,
        matches_played: 10,
        wins: 7,
        draws: 0,
        losses: 3,
      },
      {
        id: 'r-3',
        rank: 3,
        agent_id: 'bot-veteran',
        participant_id: 'user-charlie',
        score: 900,
        matches_played: 8,
        wins: 4,
        draws: 2,
        losses: 2,
      },
    ];

    it('renders leaderboard table with rank, score, matches, W/D/L and win rate %', async () => {
      vi.spyOn(ApiService, 'listContests').mockResolvedValue(mockContests);
      vi.spyOn(ApiService, 'listRankings').mockResolvedValue(mockRankings);

      render(<RankingsPage />);

      await waitFor(() => {
        expect(screen.getByText('bot-champion')).toBeDefined();
        expect(screen.getByText('user-alice')).toBeDefined();
        expect(screen.getByText(/90%/)).toBeDefined(); // Win rate (9/10)
      });
    });

    it('filters rankings by search input', async () => {
      vi.spyOn(ApiService, 'listContests').mockResolvedValue(mockContests);
      vi.spyOn(ApiService, 'listRankings').mockResolvedValue(mockRankings);

      render(<RankingsPage />);

      await waitFor(() => {
        expect(screen.getByText('bot-champion')).toBeDefined();
        expect(screen.getByText('bot-challenger')).toBeDefined();
      });

      const searchInput = screen.getByPlaceholderText(/Buscar por agente o participante/i);
      fireEvent.change(searchInput, { target: { value: 'bob' } });

      expect(screen.queryByText('bot-champion')).toBeNull();
      expect(screen.getByText('bot-challenger')).toBeDefined();
    });

    it('filters rankings by selected tournament and shows live badge for in_progress contests', async () => {
      vi.spyOn(ApiService, 'listContests').mockResolvedValue(mockContests);
      vi.spyOn(ApiService, 'listRankings').mockResolvedValue(mockRankings);

      render(<RankingsPage />);

      await waitFor(() => {
        expect(screen.getByText('Todos los concursos (Global)')).toBeDefined();
      });

      // Select in_progress contest
      const contestSelect = screen.getByLabelText(/Filtrar por concurso/i);
      await act(async () => {
        fireEvent.change(contestSelect, { target: { value: 'c-live' } });
      });

      await waitFor(() => {
        expect(ApiService.listRankings).toHaveBeenCalledWith('c-live');
        expect(screen.getByText(/Actualización en vivo/i)).toBeDefined();
      });
    });

    it('sorts by matches played and win rate', async () => {
      vi.spyOn(ApiService, 'listContests').mockResolvedValue(mockContests);
      vi.spyOn(ApiService, 'listRankings').mockResolvedValue(mockRankings);

      render(<RankingsPage />);

      await waitFor(() => {
        expect(screen.getByText('bot-champion')).toBeDefined();
      });

      const sortSelect = screen.getByLabelText(/Ordenar por/i);

      // Sort by win rate
      fireEvent.change(sortSelect, { target: { value: 'winRate' } });
      expect(screen.getByText('bot-champion')).toBeDefined();

      // Sort by matches
      fireEvent.change(sortSelect, { target: { value: 'matches' } });
      expect(screen.getByText('bot-champion')).toBeDefined();
    });
  });
});
