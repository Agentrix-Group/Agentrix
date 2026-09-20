/**
 * Starfighter Replay Parser & SHA-256 Integrity Verifier
 * Compliant with ADR-0008, ADR-0009 and ATD-007 specifications.
 */

// Fallback pure JS SHA-256 implementation if window.crypto.subtle is unavailable
function sha256Fallback(bytes) {
  const K = [
    0x428a2f98, 0x71374491, 0xb5c0fbcf, 0xe9b5dba5, 0x3956c25b, 0x59f111f1, 0x923f82a4, 0xab1c5ed5,
    0xd807aa98, 0x12835b01, 0x243185be, 0x550c7dc3, 0x72be5d74, 0x80deb1fe, 0x9bdc06a7, 0xc19bf174,
    0xe49b69c1, 0xefbe4786, 0x0fc19dc6, 0x240ca1cc, 0x2de92c6f, 0x4a7484aa, 0x5cb0a9dc, 0x76f988da,
    0x983e5152, 0xa831c66d, 0xb00327c8, 0xbf597fc7, 0xc6e00bf3, 0xd5a79147, 0x06ca6351, 0x14292967,
    0x27b70a85, 0x2e1b2138, 0x4d2c6dfc, 0x53380d13, 0x650a7354, 0x766a0abb, 0x81c2c92e, 0x92722c85,
    0xa2bfe8a1, 0xa81a664b, 0xc24b8b70, 0xc76c51a3, 0xd192e819, 0xd6990624, 0xf40e3585, 0x106aa070,
    0x19a4c116, 0x1e376c08, 0x2748774c, 0x34b0bcb5, 0x391c0cb3, 0x4ed8aa4a, 0x5b9cca4f, 0x682e6ff3,
    0x748f82ee, 0x78a5636f, 0x84c87814, 0x8cc70208, 0x90befffa, 0xa4506ceb, 0xbef9a3f7, 0xc67178f2,
  ];

  let H0 = 0x6a09e667, H1 = 0xbb67ae85, H2 = 0x3c6ef372, H3 = 0xa54ff53a;
  let H4 = 0x510e527f, H5 = 0x9b05688c, H6 = 0x1f83d9ab, H7 = 0x5be0cd19;

  const len = bytes.length;
  const bitLen = len * 8;
  const withPadLen = ((len + 8) >> 6) + 1 << 6;
  const padded = new Uint8Array(withPadLen);
  padded.set(bytes);
  padded[len] = 0x80;

  const view = new DataView(padded.buffer);
  view.setUint32(withPadLen - 4, bitLen, false);

  const W = new Uint32Array(64);
  for (let i = 0; i < withPadLen; i += 64) {
    for (let t = 0; t < 16; t++) {
      W[t] = view.getUint32(i + (t << 2), false);
    }
    for (let t = 16; t < 64; t++) {
      const s0 = ((W[t - 15] >>> 7) | (W[t - 15] << 25)) ^ ((W[t - 15] >>> 18) | (W[t - 15] << 14)) ^ (W[t - 15] >>> 3);
      const s1 = ((W[t - 2] >>> 17) | (W[t - 2] << 15)) ^ ((W[t - 2] >>> 19) | (W[t - 2] << 13)) ^ (W[t - 2] >>> 10);
      W[t] = (W[t - 16] + s0 + W[t - 7] + s1) | 0;
    }

    let a = H0, b = H1, c = H2, d = H3, e = H4, f = H5, g = H6, h = H7;
    for (let t = 0; t < 64; t++) {
      const S1 = ((e >>> 6) | (e << 26)) ^ ((e >>> 11) | (e << 21)) ^ ((e >>> 25) | (e << 7));
      const ch = (e & f) ^ (~e & g);
      const temp1 = (h + S1 + ch + K[t] + W[t]) | 0;
      const S0 = ((a >>> 2) | (a << 30)) ^ ((a >>> 13) | (a << 19)) ^ ((a >>> 22) | (a << 10));
      const maj = (a & b) ^ (a & c) ^ (b & c);
      const temp2 = (S0 + maj) | 0;

      h = g;
      g = f;
      f = e;
      e = (d + temp1) | 0;
      d = c;
      c = b;
      b = a;
      a = (temp1 + temp2) | 0;
    }

    H0 = (H0 + a) | 0;
    H1 = (H1 + b) | 0;
    H2 = (H2 + c) | 0;
    H3 = (H3 + d) | 0;
    H4 = (H4 + e) | 0;
    H5 = (H5 + f) | 0;
    H6 = (H6 + g) | 0;
    H7 = (H7 + h) | 0;
  }

  const result = [H0, H1, H2, H3, H4, H5, H6, H7]
    .map((h) => (h >>> 0).toString(16).padStart(8, '0'))
    .join('');
  return result;
}

export async function computeSha256(raw) {
  const bytes = typeof raw === 'string' ? new TextEncoder().encode(raw) : new Uint8Array(raw);
  try {
    if (typeof crypto !== 'undefined' && crypto.subtle && typeof crypto.subtle.digest === 'function') {
      const hashBuffer = await crypto.subtle.digest('SHA-256', bytes);
      const hashArray = Array.from(new Uint8Array(hashBuffer));
      return hashArray.map((b) => b.toString(16).padStart(2, '0')).join('');
    }
  } catch {
    // Fall back to pure JS if subtle crypto fails
  }
  return sha256Fallback(bytes);
}

export function parseReplayNDJSON(raw) {
  if (!raw || typeof raw !== 'string') {
    throw new Error('Replay content is empty or invalid');
  }

  const records = raw
    .split(/\r?\n/)
    .map((line) => line.trim())
    .filter(Boolean)
    .map((line, idx) => {
      try {
        return JSON.parse(line);
      } catch (err) {
        throw new Error(`Invalid JSON at line ${idx + 1}: ${err.message}`);
      }
    });

  const metadata = records[0];
  if (records.length < 3 || metadata?.type !== 'metadata') {
    throw new Error('Replay metadata is missing');
  }

  if (
    !metadata.replay_id
    || !metadata.match_id
    || metadata.game_id !== 'starfighter'
    || metadata.participants?.length !== 2
    || metadata.participants.some((id) => !id)
    || metadata.participants[0] === metadata.participants[1]
    || !Number.isInteger(metadata.seed)
    || metadata.fixed_timestep_ms <= 0
    || Number.isNaN(Date.parse(metadata.created_at))
  ) {
    throw new Error('Replay metadata is invalid');
  }

  // Extract adaptive arena dimensions from metadata or default to 2000x1000
  metadata.arena_width = Number(metadata.arena_width || metadata.arenaWidth || 2000);
  metadata.arena_height = Number(metadata.arena_height || metadata.arenaHeight || 1000);

  const result = records.at(-1);
  if (result.type !== 'result') {
    throw new Error('Replay result is missing');
  }

  const snapshots = records.slice(1, -1);
  if (snapshots.some((record) => record.type !== 'snapshot')) {
    throw new Error('Replay contains an unknown record');
  }

  snapshots.forEach((frame, index) => {
    if (
      frame.tick !== index
      || frame.public_snapshot?.tick !== index
      || !frame.state_hash
      || frame.public_snapshot?.stateHash !== frame.state_hash
    ) {
      throw new Error(`Replay tick mismatch at frame ${index}`);
    }
  });

  const lastSnapshot = snapshots.at(-1);
  if (!lastSnapshot || result.final_tick !== lastSnapshot.tick || result.final_state_hash !== lastSnapshot.state_hash) {
    throw new Error('Replay result does not seal the final snapshot');
  }

  if (
    !['eliminated', 'timeout', 'score_limit'].includes(result.reason)
    || !result.scores
    || Number.isNaN(Date.parse(result.finished_at))
  ) {
    throw new Error('Replay result reason is invalid');
  }

  return { metadata, snapshots, result };
}

/**
 * Asynchronously parse replay and compute cryptographic hash.
 * Supports running off-main-thread with automatic verification of SHA-256.
 */
export async function parseReplayAsync(raw, expectedSha256 = null) {
  // Yield to avoid freezing the caller frame before heavy parsing
  await new Promise((resolve) => setTimeout(resolve, 0));

  const [replay, computedSha256] = await Promise.all([
    Promise.resolve().then(() => parseReplayNDJSON(raw)),
    computeSha256(raw),
  ]);

  let integrityStatus = 'unverified';
  const targetHash = expectedSha256 || replay.metadata.sha256;

  if (targetHash) {
    integrityStatus = computedSha256.toLowerCase() === targetHash.toLowerCase()
      ? 'verified'
      : 'mismatch';
  }

  return {
    replay,
    computedSha256,
    integrityStatus,
  };
}
