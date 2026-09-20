import React from 'react';
import { describe, it, expect, beforeEach, vi, afterEach } from 'vitest';
import { render, screen, fireEvent, act, waitFor } from '@testing-library/react';
import i18n from '../src/i18n/index.js';
import {
  computeSha256,
  parseReplayNDJSON,
  parseReplayAsync,
} from '../src/viewer/replayParser.js';
import { ReplayViewer } from '../src/viewer/ReplayViewer.jsx';
import { drawStarfighterArena } from '../src/renderers/starfighter/canvasRenderer.js';
import { ApiService } from '../src/service/apiService.js';

const VALID_METADATA = {
  type: 'metadata',
  replay_id: 'rep-perf-42',
  match_id: 'match-perf-42',
  game_id: 'starfighter',
  participants: ['bot-alpha', 'bot-omega'],
  seed: 42,
  fixed_timestep_ms: 16,
  created_at: '2026-09-19T12:00:00Z',
  arena_width: 2400,
  arena_height: 1200,
};

const SAMPLE_FRAMES = [
  JSON.stringify(VALID_METADATA),
  JSON.stringify({
    type: 'snapshot',
    tick: 0,
    state_hash: 'hash-0',
    public_snapshot: {
      tick: 0,
      stateHash: 'hash-0',
      fighters: [
        { playerId: 'bot-alpha', position: { x: -800, y: 0 }, rotation: 0, health: 100 },
        { playerId: 'bot-omega', position: { x: 800, y: 0 }, rotation: 3.14, health: 100 },
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
        { playerId: 'bot-alpha', position: { x: -750, y: 10 }, rotation: 0.1, health: 95 },
        { playerId: 'bot-omega', position: { x: 750, y: -10 }, rotation: 3.0, health: 100 },
      ],
      bullets: [{ position: { x: -700, y: 10 } }],
    },
  }),
  JSON.stringify({
    type: 'result',
    final_tick: 1,
    final_state_hash: 'hash-1',
    winner: 'bot-omega',
    reason: 'eliminated',
    scores: { 'bot-alpha': 10, 'bot-omega': 100 },
    finished_at: '2026-09-19T12:00:01Z',
  }),
].join('\n');

describe('Sprint FQ-4: High-Performance Replay Viewer & SHA-256 Integrity', () => {
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

  describe('Cryptographic SHA-256 Verification & Parser', () => {
    it('computes correct SHA-256 digest for standard test vectors', async () => {
      // Empty string
      const hashEmpty = await computeSha256('');
      expect(hashEmpty).toBe('e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855');

      // "abc"
      const hashAbc = await computeSha256('abc');
      expect(hashAbc).toBe('ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad');
    });

    it('parses NDJSON and extracts adaptive arena dimensions', () => {
      const parsed = parseReplayNDJSON(SAMPLE_FRAMES);
      expect(parsed.metadata.arena_width).toBe(2400);
      expect(parsed.metadata.arena_height).toBe(1200);
      expect(parsed.snapshots).toHaveLength(2);
      expect(parsed.result.winner).toBe('bot-omega');
    });

    it('verifies bit-for-bit integrity when expected hash matches', async () => {
      const computed = await computeSha256(SAMPLE_FRAMES);
      const res = await parseReplayAsync(SAMPLE_FRAMES, computed);

      expect(res.computedSha256).toBe(computed);
      expect(res.integrityStatus).toBe('verified');
    });

    it('detects integrity divergence when expected hash differs', async () => {
      const wrongHash = '0000000000000000000000000000000000000000000000000000000000000000';
      const res = await parseReplayAsync(SAMPLE_FRAMES, wrongHash);

      expect(res.integrityStatus).toBe('mismatch');
    });

    it('reports unverified when no expected hash is available', async () => {
      const res = await parseReplayAsync(SAMPLE_FRAMES, null);
      expect(res.integrityStatus).toBe('unverified');
    });
  });

  describe('Adaptive Canvas Arena Rendering', () => {
    it('renders without throwing when provided mock canvas context and custom arena', () => {
      const mockCtx = {
        clearRect: vi.fn(),
        createLinearGradient: vi.fn(() => ({ addColorStop: vi.fn() })),
        fillRect: vi.fn(),
        beginPath: vi.fn(),
        arc: vi.fn(),
        fill: vi.fn(),
        stroke: vi.fn(),
        strokeRect: vi.fn(),
        setLineDash: vi.fn(),
        save: vi.fn(),
        restore: vi.fn(),
        translate: vi.fn(),
        rotate: vi.fn(),
        moveTo: vi.fn(),
        lineTo: vi.fn(),
        closePath: vi.fn(),
        fillText: vi.fn(),
      };

      const mockCanvas = {
        width: 960,
        height: 540,
        getContext: vi.fn(() => mockCtx),
      };

      const frame = {
        public_snapshot: {
          fighters: [{ playerId: 'bot-1', position: { x: 0, y: 0 }, rotation: 0, health: 100 }],
          bullets: [{ position: { x: 50, y: 50 } }],
        },
      };

      drawStarfighterArena(mockCanvas, frame, { arena_width: 3000, arena_height: 1500 });

      expect(mockCanvas.getContext).toHaveBeenCalledWith('2d');
      expect(mockCtx.clearRect).toHaveBeenCalledWith(0, 0, 960, 540);
      expect(mockCtx.fillRect).toHaveBeenCalled();
    });
  });

  describe('ReplayViewer UI, Integrity Badge & Keyboard Navigation', () => {
    it('displays verified integrity badge when hash matches metadata', async () => {
      const expectedHash = await computeSha256(SAMPLE_FRAMES);
      vi.spyOn(ApiService, 'streamReplay').mockResolvedValue(SAMPLE_FRAMES);
      vi.spyOn(ApiService, 'getReplay').mockResolvedValue({
        id: 'rep-perf-42',
        sha256: expectedHash,
      });

      render(<ReplayViewer replayId="rep-perf-42" />);

      await waitFor(() => {
        expect(screen.getByText('Verificada bit a bit')).toBeDefined();
      });
    });

    it('displays mismatch badge when hash differs from metadata', async () => {
      vi.spyOn(ApiService, 'streamReplay').mockResolvedValue(SAMPLE_FRAMES);
      vi.spyOn(ApiService, 'getReplay').mockResolvedValue({
        id: 'rep-perf-42',
        sha256: 'tampered-or-divergent-hash',
      });

      render(<ReplayViewer replayId="rep-perf-42" />);

      await waitFor(() => {
        expect(screen.getByText('Divergencia detectada')).toBeDefined();
      });
    });

    it('supports keyboard navigation (Space to toggle play, Arrow keys to step ticks)', async () => {
      vi.spyOn(ApiService, 'streamReplay').mockResolvedValue(SAMPLE_FRAMES);
      vi.spyOn(ApiService, 'getReplay').mockResolvedValue({ id: 'rep-perf-42' });

      render(<ReplayViewer replayId="rep-perf-42" />);

      await waitFor(() => {
        expect(screen.getByRole('button', { name: /reproducir/i })).toBeDefined();
      });

      // Press Space to start playback
      fireEvent.keyDown(window, { key: ' ', code: 'Space' });
      await waitFor(() => {
        expect(screen.getByRole('button', { name: /pausar/i })).toBeDefined();
      });

      // Press Space to pause
      fireEvent.keyDown(window, { key: ' ', code: 'Space' });
      await waitFor(() => {
        expect(screen.getByRole('button', { name: /reproducir/i })).toBeDefined();
      });

      // Press ArrowRight to step forward to tick 1
      const slider = screen.getByRole('slider', { name: /línea de tiempo/i });
      fireEvent.keyDown(window, { key: 'ArrowRight' });
      expect(slider.getAttribute('value')).toBe('1');

      // Press ArrowLeft to step back to tick 0
      fireEvent.keyDown(window, { key: 'ArrowLeft' });
      expect(slider.getAttribute('value')).toBe('0');

      // Press End to jump to final tick
      fireEvent.keyDown(window, { key: 'End' });
      expect(slider.getAttribute('value')).toBe('1');

      // Press Home to jump to tick 0
      fireEvent.keyDown(window, { key: 'Home' });
      expect(slider.getAttribute('value')).toBe('0');
    });
  });
});
