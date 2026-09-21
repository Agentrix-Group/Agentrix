import { describe, it, expect } from 'vitest';
import { parseReplay, tickIntervalMs, ReplayFormatError } from '../src/viewer/replayParser.js';

const metadata = {
  type: 'metadata', format: 'agentrix-replay/2', match_id: 'm1', run_id: 'r1', spec_hash: 'a'.repeat(64),
  tick_rate: { numerator: 60, denominator: 1 }, participants: [{ slot_id: 's1', player_id: 'p1' }, { slot_id: 's2', player_id: 'p2' }],
};
const snap = (tick, hash) => ({ type: 'snapshot', tick, state_hash: hash, public_snapshot: { tick } });
const build = (records) => records.map((r) => JSON.stringify(r)).join('\n');

describe('replay parser v2', () => {
  const frames = [snap(0, 'h0'), snap(1, 'h1'), snap(2, 'h2')];
  const result = { type: 'result', final_tick: 2, final_state_hash: 'h2', termination_reason: 'elimination' };

  it('parses a sealed stream', () => {
    const replay = parseReplay(build([metadata, ...frames, result]));
    expect(replay.frames).toHaveLength(3);
    expect(replay.metadata.tick_rate).toEqual({ numerator: 60, denominator: 1 });
  });

  it('rejects legacy formats, gaps, broken seals and truncated streams', () => {
    expect(() => parseReplay(build([{ ...metadata, format: undefined, fixed_timestep_ms: 17 }, ...frames, result]))).toThrow(ReplayFormatError);
    expect(() => parseReplay(build([metadata, frames[0], frames[2], result]))).toThrow(/position 1/);
    expect(() => parseReplay(build([metadata, ...frames, { ...result, final_state_hash: 'hx' }]))).toThrow(/seal/);
    expect(() => parseReplay(build([metadata, ...frames]))).toThrow(/not sealed/);
    expect(() => parseReplay(`${build([metadata])}\n{broken`)).toThrow(/line 2/);
    expect(() => parseReplay('')).toThrow(/empty/);
  });

  it('derives the exact tick interval from the rational rate', () => {
    expect(tickIntervalMs({ numerator: 60, denominator: 1 })).toBeCloseTo(16.6667, 3);
    expect(tickIntervalMs({ numerator: 30, denominator: 1 })).toBe(1000 / 30);
    expect(tickIntervalMs({ numerator: 1, denominator: 2 })).toBe(2000);
  });
});
