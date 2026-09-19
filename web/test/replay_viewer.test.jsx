import React from 'react';
import { describe, it, expect, beforeEach, vi } from 'vitest';
import { render, screen, fireEvent, act, waitFor } from '@testing-library/react';
import i18n from '../src/i18n/index.js';
import { ReplayViewer } from '../src/components/replay/ReplayViewer.jsx';
import { ApiService } from '../src/service/apiService.js';

const SAMPLE_NDJSON = [
  JSON.stringify({
    type: 'metadata',
    replay_id: 'rep-star-123',
    match_id: 'match-star-123',
    game_id: 'starfighter',
    participants: ['hunter', 'evasive'],
    seed: 42,
    fixed_timestep_ms: 17,
    created_at: '2026-09-19T10:00:00Z',
  }),
  JSON.stringify({
    type: 'snapshot',
    tick: 0,
    state_hash: 'hash-0',
    public_snapshot: {
      tick: 0,
      stateHash: 'hash-0',
      fighters: [
        { playerId: 'hunter', position: { x: -600, y: 0 }, rotation: 0, health: 100, shieldActive: false },
        { playerId: 'evasive', position: { x: 600, y: 0 }, rotation: 3.14, health: 100, shieldActive: true },
      ],
      bullets: [],
    },
  }),
  JSON.stringify({
    type: 'snapshot',
    tick: 1,
    state_hash: 'hash-1',
    public_snapshot: {
      tick: 1,
      stateHash: 'hash-1',
      fighters: [
        { playerId: 'hunter', position: { x: -590, y: 0 }, rotation: 0, health: 100, shieldActive: false },
        { playerId: 'evasive', position: { x: 595, y: 0 }, rotation: 3.14, health: 100, shieldActive: true },
      ],
      bullets: [{ position: { x: -550, y: 0 } }],
    },
  }),
  JSON.stringify({
    type: 'result',
    final_tick: 1,
    final_state_hash: 'hash-1',
    winner: 'hunter',
    reason: 'eliminated',
    scores: { hunter: 100, evasive: 20 },
    finished_at: '2026-09-19T10:00:01Z',
  }),
].join('\n');

describe('Starfighter Canvas 2D ReplayViewer Component', () => {
  beforeEach(async () => {
    await act(async () => {
      await i18n.changeLanguage('es');
    });
    vi.spyOn(ApiService, 'streamReplay').mockResolvedValue(SAMPLE_NDJSON);
  });

  it('renders canvas, timeline scrubber, and speed controls (0.5x, 1x, 2x, 4x)', async () => {
    render(<ReplayViewer replayId="rep-star-123" />);

    // Wait for the replay to parse and load
    await waitFor(() => {
      expect(screen.getByText('STARFIGHTER · REPLAY')).toBeDefined();
    });

    // Check speed control buttons
    expect(screen.getByText('0.5×')).toBeDefined();
    expect(screen.getByText('1×')).toBeDefined();
    expect(screen.getByText('2×')).toBeDefined();
    expect(screen.getByText('4×')).toBeDefined();

    // Check play / pause button exists
    const playButton = screen.getByRole('button', { name: /reproducir/i });
    expect(playButton).toBeDefined();

    // Check timeline scrubber slider
    const slider = screen.getByRole('slider', { name: /línea de tiempo/i });
    expect(slider).toBeDefined();
    expect(slider.getAttribute('max')).toBe('1');
    expect(slider.getAttribute('value')).toBe('0');

    // Click 2x speed
    const speed2xButton = screen.getByText('2×');
    fireEvent.click(speed2xButton);
    expect(speed2xButton.className).toContain('active');

    // Toggle playback
    fireEvent.click(playButton);
    expect(screen.getByRole('button', { name: /pausar/i })).toBeDefined();

    // Scrubber interaction updates frame index
    fireEvent.change(slider, { target: { value: '1' } });
    expect(slider.getAttribute('value')).toBe('1');
  });
});
