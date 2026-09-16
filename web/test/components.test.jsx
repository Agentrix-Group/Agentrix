import React from 'react';
import { describe, it, expect, beforeEach } from 'vitest';
import { render, screen, fireEvent, act } from '@testing-library/react';
import i18n from '../src/i18n/index.js';
import { Navbar } from '../src/components/Navbar.jsx';
import { MatchCard } from '../src/components/MatchCard.jsx';
import { RankingsPage } from '../src/pages/RankingsPage.jsx';

describe('Localized React Components Rendering', () => {
  beforeEach(async () => {
    localStorage.clear();
    await act(async () => {
      await i18n.changeLanguage('es');
    });
  });

  it('renders Navbar in Spanish by default', () => {
    render(<Navbar activeTab="home" onSelectTab={() => {}} currentUser={null} onLogout={() => {}} />);

    expect(screen.getByText('Plataforma Agentrix')).toBeDefined();
    expect(screen.getByText('Inicio')).toBeDefined();
    expect(screen.getByText('Mis agentes')).toBeDefined();
    expect(screen.getByText('Partidas')).toBeDefined();
    expect(screen.getByText('Clasificación')).toBeDefined();
    expect(screen.getByText('Visor de repeticiones')).toBeDefined();
    expect(screen.getByText('Iniciar sesión / Registrarse')).toBeDefined();
  });

  it('switches Navbar language to English via the language selector', async () => {
    render(<Navbar activeTab="home" onSelectTab={() => {}} currentUser={null} onLogout={() => {}} />);

    const select = screen.getByLabelText('Seleccionar idioma');
    fireEvent.change(select, { target: { value: 'en' } });

    expect(screen.getByText('Agentrix Platform')).toBeDefined();
    expect(screen.getByText('Dashboard')).toBeDefined();
    expect(screen.getByText('My Agents')).toBeDefined();
    expect(screen.getByText('Matches')).toBeDefined();
    expect(screen.getByText('Leaderboard')).toBeDefined();
    expect(screen.getByText('Replay Viewer')).toBeDefined();
    expect(screen.getByText('Sign In / Register')).toBeDefined();
  });

  it('renders MatchCard with localized status and attributes in Spanish and English', async () => {
    const mockMatch = {
      id: 'm-12345678-abcd',
      game_id: 'arena-basica',
      status: 'finished',
      seed: 42,
      replay_id: 'rep-001',
      results: [
        { id: 'r1', rank: 1, submission_id: 'sub-alpha', score: 100 },
      ],
    };

    const { rerender } = render(
      <MatchCard match={mockMatch} onWatchReplay={() => {}} onTriggerRun={() => {}} />
    );

    // Spanish verification
    expect(screen.getByText('Partida #m-123456')).toBeDefined();
    expect(screen.getByText('Finalizada')).toBeDefined();
    expect(screen.getByText(/Juego: arena-basica/)).toBeDefined();
    expect(screen.getByText(/Puesto 1: sub-alpha/)).toBeDefined();
    expect(screen.getByText('Ver repetición')).toBeDefined();

    // Switch to English
    await act(async () => {
      await i18n.changeLanguage('en');
    });
    rerender(<MatchCard match={mockMatch} onWatchReplay={() => {}} onTriggerRun={() => {}} />);

    expect(screen.getByText('Match #m-123456')).toBeDefined();
    expect(screen.getByText('Finished')).toBeDefined();
    expect(screen.getByText(/Game: arena-basica/)).toBeDefined();
    expect(screen.getByText(/Rank 1: sub-alpha/)).toBeDefined();
    expect(screen.getByText('Watch Replay')).toBeDefined();
  });

  it('renders RankingsPage table headers localized in Spanish and English', async () => {
    const { rerender } = render(<RankingsPage />);

    expect(screen.getByText('Clasificación del torneo')).toBeDefined();
    expect(screen.getByText('Puesto')).toBeDefined();
    expect(screen.getByText('Agente')).toBeDefined();
    expect(screen.getByText('Participante')).toBeDefined();
    expect(screen.getByText('Puntuación')).toBeDefined();
    expect(screen.getByText('Partidas')).toBeDefined();
    expect(screen.getByText('V / E / D')).toBeDefined();

    await act(async () => {
      await i18n.changeLanguage('en');
    });
    rerender(<RankingsPage />);

    expect(screen.getByText('Tournament Leaderboard')).toBeDefined();
    expect(screen.getByText('Rank')).toBeDefined();
    expect(screen.getByText('Agent')).toBeDefined();
    expect(screen.getByText('Participant')).toBeDefined();
    expect(screen.getByText('Score')).toBeDefined();
    expect(screen.getByText('Matches')).toBeDefined();
    expect(screen.getByText('W / D / L')).toBeDefined();
  });
});
