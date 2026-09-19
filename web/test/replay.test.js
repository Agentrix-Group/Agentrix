import { describe, expect, it } from 'vitest';
import { parseReplayNDJSON } from '../src/viewer/ReplayViewer.jsx';

describe('Starfighter NDJSON replay', () => {
  const validMetadata = {
    type: 'metadata',
    replay_id: 'rep-1',
    match_id: 'm1',
    game_id: 'starfighter',
    seed: 42,
    participants: ['p1', 'p2'],
    fixed_timestep_ms: 17,
    created_at: '2026-09-19T00:00:00Z',
  };

  it('parses metadata, exact sequential snapshots, and final result', () => {
    const raw = [
      JSON.stringify(validMetadata),
      JSON.stringify({
        type: 'snapshot',
        tick: 0,
        state_hash: 'h0',
        public_snapshot: { tick: 0, fighters: [], bullets: [], stateHash: 'h0' },
      }),
      JSON.stringify({
        type: 'snapshot',
        tick: 1,
        state_hash: 'h1',
        public_snapshot: { tick: 1, fighters: [], bullets: [], stateHash: 'h1' },
      }),
      JSON.stringify({
        type: 'result',
        final_tick: 1,
        winner: 'p1',
        reason: 'eliminated',
        scores: { p1: 100, p2: 0 },
        final_state_hash: 'h1',
        finished_at: '2026-09-19T00:00:01Z',
      }),
      '',
    ].join('\n');

    const replay = parseReplayNDJSON(raw);
    expect(replay.metadata.match_id).toBe('m1');
    expect(replay.snapshots).toHaveLength(2);
    expect(replay.result.winner).toBe('p1');
    expect(replay.result.reason).toBe('eliminated');
  });

  it('rejects a desynchronized snapshot sequence', () => {
    const raw = [
      JSON.stringify(validMetadata),
      JSON.stringify({
        type: 'snapshot',
        tick: 1,
        state_hash: 'h1',
        public_snapshot: { tick: 1, stateHash: 'h1' },
      }),
      JSON.stringify({
        type: 'result',
        final_tick: 1,
        winner: 'p1',
        reason: 'eliminated',
        scores: {},
        final_state_hash: 'h1',
        finished_at: '2026-09-19T00:00:01Z',
      }),
    ].join('\n');

    expect(() => parseReplayNDJSON(raw)).toThrow(/tick mismatch/i);
  });

  it('rejects a non-competitive result reason', () => {
    const raw = [
      JSON.stringify(validMetadata),
      JSON.stringify({
        type: 'snapshot',
        tick: 0,
        state_hash: 'h0',
        public_snapshot: { tick: 0, stateHash: 'h0' },
      }),
      JSON.stringify({
        type: 'result',
        final_tick: 0,
        winner: '',
        reason: 'execution_error',
        scores: {},
        final_state_hash: 'h0',
        finished_at: '2026-09-19T00:00:01Z',
      }),
    ].join('\n');

    expect(() => parseReplayNDJSON(raw)).toThrow(/reason is invalid/i);
  });
});
