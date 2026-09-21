import React from 'react';
import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import i18n from '../src/i18n/index.js';
import { ContestsPage } from '../src/pages/ContestsPage.jsx';
import { ApiService } from '../src/service/apiService.js';

const mockContests = [
  {
    id: 'contest-star-2026',
    name: 'Copa Starfighter 2026',
    description: 'Torneo clasificatorio de combate espacial con Bevy Rapier.',
    game_id: 'starfighter',
    state: 'in_progress',
    start_date: '2026-10-01T00:00:00Z',
  },
  {
    id: 'contest-academy-2026',
    name: 'Torneo de Cadetes',
    description: 'Competencia para pilotos principiantes con naves ligeras.',
    game_id: 'starfighter',
    state: 'scheduled',
    start_date: '2026-11-15T00:00:00Z',
  },
];

const mockRankings = [
  {
    id: 'rank-1',
    agent_id: 'agent-hunter',
    user_id: 'pilot_alpha',
    score: 15,
    matches_played: 5,
    wins: 5,
    draws: 0,
    losses: 0,
  },
];

const mockEntries = [
  {
    id: 'entry-1',
    agent_id: 'agent-hunter',
    agent_name: 'StarHunter',
    username: 'pilot_alpha',
    status: 'active',
  },
];

describe('Contests Page Feature (Section 5.7)', () => {
  beforeEach(async () => {
    await i18n.changeLanguage('es');
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it('renders list of contests with search filter', async () => {
    vi.spyOn(ApiService, 'listContests').mockResolvedValueOnce(mockContests);

    render(<ContestsPage currentUser={null} />);

    expect(await screen.findByText('Copa Starfighter 2026')).toBeDefined();
    expect(screen.getByText('Torneo de Cadetes')).toBeDefined();

    // Filter by search query
    const searchInput = screen.getByLabelText('Buscar torneos');
    fireEvent.change(searchInput, { target: { value: 'Cadetes' } });

    expect(screen.queryByText('Copa Starfighter 2026')).toBeNull();
    expect(screen.getByText('Torneo de Cadetes')).toBeDefined();
  });

  it('navigates to contest detail view and switches tabs', async () => {
    vi.spyOn(ApiService, 'listContests').mockResolvedValue(mockContests);
    vi.spyOn(ApiService, 'getContest').mockResolvedValue(mockContests[0]);
    vi.spyOn(ApiService, 'listRankings').mockResolvedValue(mockRankings);
    vi.spyOn(ApiService, 'listContestEntries').mockResolvedValue(mockEntries);

    render(<ContestsPage contestId="contest-star-2026" currentUser={null} />);

    // Contest title and overview
    expect(await screen.findByText('Copa Starfighter 2026')).toBeDefined();
    expect(screen.getByText('Formato de Puntuación')).toBeDefined();

    // Switch to Leaderboard tab
    const leaderboardTab = screen.getByRole('button', { name: /Clasificación/i });
    fireEvent.click(leaderboardTab);

    expect(await screen.findByText('agent-hunter')).toBeDefined();
    expect(screen.getByText('pilot_alpha')).toBeDefined();

    // Switch to Rules tab
    const rulesTab = screen.getByRole('button', { name: 'Reglas del juego' });
    fireEvent.click(rulesTab);

    expect(screen.getByText(/Reglas Oficiales de Simulación: Starfighter/i)).toBeDefined();
    expect(screen.getByText(/Ciclo de Simulación \(60 Hz\)/i)).toBeDefined();

    // Switch to Entries tab
    const entriesTab = screen.getByRole('button', { name: /Agentes inscritos/i });
    fireEvent.click(entriesTab);

    expect(await screen.findByText('StarHunter')).toBeDefined();
  });

  it('displays ErrorState and allows retrying when API fails', async () => {
    const listSpy = vi.spyOn(ApiService, 'listContests')
      .mockRejectedValueOnce(new Error('Servicio de torneos temporalmente caído'))
      .mockResolvedValueOnce(mockContests);

    render(<ContestsPage currentUser={null} />);

    expect(await screen.findByRole('alert')).toBeDefined();
    expect(screen.getByText('Servicio de torneos temporalmente caído')).toBeDefined();

    // Retry
    const retryBtn = screen.getByRole('button', { name: 'Reintentar' });
    fireEvent.click(retryBtn);

    expect(await screen.findByText('Copa Starfighter 2026')).toBeDefined();
    expect(listSpy).toHaveBeenCalledTimes(2);
  });
});
