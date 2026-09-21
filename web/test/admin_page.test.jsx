import React from 'react';
import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import i18n from '../src/i18n/index.js';
import { AdminPage } from '../src/pages/AdminPage.jsx';
import { ProtectedRoute } from '../src/components/ProtectedRoute.jsx';
import { ApiService } from '../src/service/apiService.js';
import * as sessionModule from '../src/auth/session.js';

const mockUsers = [
  {
    id: 'usr-admin-1',
    username: 'admin',
    email: 'admin@agentrix.dev',
    role_id: 'admin',
    active: true,
    created_at: '2026-01-01T00:00:00Z',
  },
  {
    id: 'usr-pilot-1',
    username: 'pilot_alpha',
    email: 'alpha@agentrix.dev',
    role_id: 'pilot',
    active: true,
    created_at: '2026-02-01T00:00:00Z',
  },
];

const mockContests = [
  {
    id: 'starfighter-cup-2026',
    name: 'Copa Starfighter 2026',
    game_id: 'starfighter',
    state: 'in_progress',
  },
];

describe('Admin Page Feature (Section 5.7)', () => {
  beforeEach(async () => {
    await i18n.changeLanguage('es');
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it('renders AdminPage tabs and users list for administrator', async () => {
    vi.spyOn(ApiService, 'listUsers').mockResolvedValue(mockUsers);

    render(
      <AdminPage
        currentUser={{
          id: 'usr-admin-1',
          username: 'admin',
          role_id: 'admin',
          capabilities: ['admin:access'],
        }}
      />
    );

    expect(screen.getByText('Panel de Administración')).toBeDefined();
    expect(await screen.findByText('admin@agentrix.dev')).toBeDefined();
    expect(screen.getByText('pilot_alpha')).toBeDefined();
  });

  it('allows recalculating rankings and publishing snapshot from Contests tab', async () => {
    vi.spyOn(ApiService, 'listContests').mockResolvedValue(mockContests);
    const recalcSpy = vi.spyOn(ApiService, 'recalculateRankings').mockResolvedValue([
      { id: 'rank-1', agent_id: 'agent-hunter', score: 10 },
    ]);
    const publishSpy = vi.spyOn(ApiService, 'publishRankingSnapshot').mockResolvedValue({
      id: 'snap-1',
      contest_id: 'starfighter-cup-2026',
      version: 2,
    });

    render(
      <AdminPage
        currentUser={{
          id: 'usr-admin-1',
          username: 'admin',
          role_id: 'admin',
          capabilities: ['admin:access'],
        }}
      />
    );

    // Switch to Contests tab
    const contestsTab = screen.getByRole('button', { name: /Gestión de Torneos/i });
    fireEvent.click(contestsTab);

    expect(await screen.findByText('Copa Starfighter 2026')).toBeDefined();

    // Click recalculate rankings
    const recalcBtn = screen.getByRole('button', { name: /Recalcular Puntuaciones/i });
    fireEvent.click(recalcBtn);

    await waitFor(() => {
      expect(recalcSpy).toHaveBeenCalledWith('starfighter-cup-2026');
    });
    expect(await screen.findByText(/Puntuaciones recalculadas exitosamente/i)).toBeDefined();

    // Click publish snapshot
    const publishBtn = screen.getByRole('button', { name: /Publicar Snapshot/i });
    fireEvent.click(publishBtn);

    await waitFor(() => {
      expect(publishSpy).toHaveBeenCalledWith('starfighter-cup-2026');
    });
    expect(await screen.findByText(/Snapshot publicado exitosamente: Versión v2/i)).toBeDefined();
  });

  it('displays system audit and engine health in Audit tab', async () => {
    vi.spyOn(ApiService, 'getHealth').mockResolvedValue({
      status: 'healthy',
      database: 'connected',
    });

    render(
      <AdminPage
        currentUser={{
          id: 'usr-admin-1',
          username: 'admin',
          role_id: 'admin',
          capabilities: ['admin:access'],
        }}
      />
    );

    const auditTab = screen.getByRole('button', { name: /Auditoría y Salud/i });
    fireEvent.click(auditTab);

    expect(screen.getByText('Estado del Servidor')).toBeDefined();
    expect(screen.getByText('Base de Datos y Migraciones')).toBeDefined();
    expect(screen.getByText('Motor Starfighter Rust')).toBeDefined();
    expect(screen.getByText(/60.0 Hz Fixed Timestep/i)).toBeDefined();
  });
});
