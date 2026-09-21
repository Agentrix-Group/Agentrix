/**
 * Parser for agentrix-replay/2 NDJSON streams. The server verified the
 * artifact digest before publishing it; the browser validates the stream
 * structure (sequence, state hash chain seal) before rendering.
 */
export const REPLAY_FORMAT = 'agentrix-replay/2';

export class ReplayFormatError extends Error {
  constructor(message) {
    super(message);
    this.name = 'ReplayFormatError';
  }
}

export function parseReplay(raw) {
  if (typeof raw !== 'string' || raw.trim() === '') throw new ReplayFormatError('empty replay');
  const lines = raw.split('\n').filter((line) => line.trim() !== '');
  const records = lines.map((line, index) => {
    try {
      return JSON.parse(line);
    } catch {
      throw new ReplayFormatError(`line ${index + 1} is not valid JSON`);
    }
  });
  const metadata = records[0];
  if (metadata?.type !== 'metadata' || metadata.format !== REPLAY_FORMAT) {
    throw new ReplayFormatError(`unsupported replay format ${metadata?.format}`);
  }
  const rate = metadata.tick_rate;
  if (!rate || !(rate.numerator > 0) || !(rate.denominator > 0)) throw new ReplayFormatError('invalid tick_rate');
  if (!Array.isArray(metadata.participants) || metadata.participants.length === 0) throw new ReplayFormatError('no participants');
  const result = records.at(-1);
  if (result?.type !== 'result') throw new ReplayFormatError('replay is not sealed');
  const frames = records.slice(1, -1);
  frames.forEach((frame, index) => {
    if (frame.type !== 'snapshot' || frame.tick !== index || !frame.state_hash || typeof frame.public_snapshot !== 'object') {
      throw new ReplayFormatError(`invalid snapshot at position ${index}`);
    }
  });
  const last = frames.at(-1);
  if (!last || result.final_tick !== last.tick || result.final_state_hash !== last.state_hash) {
    throw new ReplayFormatError('result does not seal the final snapshot');
  }
  return { metadata, frames, result };
}

/** Milliseconds of simulated time per tick, from the exact ratio. */
export function tickIntervalMs(tickRate) {
  return (1000 * tickRate.denominator) / tickRate.numerator;
}
