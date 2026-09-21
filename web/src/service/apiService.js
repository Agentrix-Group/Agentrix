/**
 * Endpoint layer of the Agentrix API (open-api/openapi.yaml). Every UI call
 * goes through these functions; they never invent defaults for fields the
 * API requires (status, submission_id, Idempotency-Key...).
 */
import { request, cookieRequest, buildQuery, newIdempotencyKey, BASE_URL } from '../api/client.js';
import { setSession, clearSession, getAccessToken } from '../auth/session.js';

const id = (value) => encodeURIComponent(value);

export const ApiService = {
  // --- Sessions -----------------------------------------------------------
  async login(username, password) {
    const session = await request('/auth/login', { method: 'POST', json: { username, password }, auth: false });
    setSession(session);
    return session;
  },
  register: (username, email, password) =>
    request('/auth/register', { method: 'POST', json: { username, email, password }, auth: false }),
  /** Restores a session from the refresh cookie (page reload). */
  async restore() {
    try {
      const session = await cookieRequest('/auth/refresh');
      setSession(session);
      return session;
    } catch {
      clearSession();
      return null;
    }
  },
  /** Revokes the session on the server, then forgets it locally. */
  async logout() {
    try {
      await cookieRequest('/auth/logout');
    } finally {
      clearSession();
    }
  },
  async logoutAll() {
    const result = await request('/auth/logout-all', { method: 'POST' });
    clearSession();
    return result;
  },
  me: (opts) => request('/me', opts),

  // --- Users (admin) --------------------------------------------------------
  listUsers: (status, opts) => request(`/users${buildQuery({ status })}`, opts),
  createUser: (data) => request('/users', { method: 'POST', json: data }),
  replaceRoles: (userId, roles) => request(`/users/${id(userId)}/roles`, { method: 'PUT', json: { roles } }),
  setUserStatus: (userId, status) => {
    if (!['active', 'suspended', 'disabled'].includes(status)) {
      throw new Error(`setUserStatus requires an explicit status, got ${status}`);
    }
    return request(`/users/${id(userId)}/status`, { method: 'PUT', json: { status } });
  },

  // --- Games ------------------------------------------------------------------
  listGames: (opts) => request('/games', opts),
  getGame: (gameId, opts) => request(`/games/${id(gameId)}`, opts),

  // --- Agents and submissions ---------------------------------------------
  listAgents: (opts) => request('/agents', opts),
  getAgent: (agentId, opts) => request(`/agents/${id(agentId)}`, opts),
  createAgent: (data) => request('/agents', { method: 'POST', json: data }),
  updateAgent: (agentId, patch) => request(`/agents/${id(agentId)}`, { method: 'PATCH', json: patch }),
  setAgentStatus: (agentId, status) => {
    if (status !== 'active' && status !== 'disabled') {
      throw new Error(`setAgentStatus requires an explicit status, got ${status}`);
    }
    return request(`/agents/${id(agentId)}/status`, { method: 'PUT', json: { status } });
  },
  listSubmissions: (agentId, opts) => request(`/agents/${id(agentId)}/submissions`, opts),
  getSubmission: (submissionId, opts) => request(`/submissions/${id(submissionId)}`, opts),
  uploadSubmission: (agentId, file) => {
    const form = new FormData();
    form.append('bundle', file);
    return request(`/agents/${id(agentId)}/submissions`, { method: 'POST', form, timeout: 60000 });
  },
  disableSubmission: (submissionId) => request(`/submissions/${id(submissionId)}/disable`, { method: 'POST' }),

  // --- Contests and entries -------------------------------------------------
  listContests: (opts) => request('/contests', opts),
  getContest: (contestId, opts) => request(`/contests/${id(contestId)}`, opts),
  createContest: (data) => request('/contests', { method: 'POST', json: data }),
  updateContest: (contestId, patch) => request(`/contests/${id(contestId)}`, { method: 'PATCH', json: patch }),
  transitionContest: (contestId, state, reason) =>
    request(`/contests/${id(contestId)}/transitions`, { method: 'POST', json: reason ? { state, reason } : { state } }),
  listEntries: (contestId, opts) => request(`/contests/${id(contestId)}/entries`, opts),
  enroll: (contestId, agentId, submissionId) => {
    if (!agentId || !submissionId) throw new Error('enroll requires agent_id and an explicit submission_id');
    return request(`/contests/${id(contestId)}/entries`, { method: 'POST', json: { agent_id: agentId, submission_id: submissionId } });
  },
  resubmitEntry: (contestId, entryId, submissionId) =>
    request(`/contests/${id(contestId)}/entries/${id(entryId)}/submission`, { method: 'PUT', json: { submission_id: submissionId } }),
  withdrawEntry: (contestId, entryId, reason) =>
    request(`/contests/${id(contestId)}/entries/${id(entryId)}/withdraw`, { method: 'POST', json: { reason } }),
  disqualifyEntry: (contestId, entryId, reason) =>
    request(`/contests/${id(contestId)}/entries/${id(entryId)}/disqualify`, { method: 'POST', json: { reason } }),

  // --- Rankings -----------------------------------------------------------------
  getRankings: (contestId, opts) => request(`/contests/${id(contestId)}/rankings`, opts),
  recalculateRankings: (contestId) => request(`/contests/${id(contestId)}/rankings/recalculate`, { method: 'POST' }),
  listSnapshots: (contestId, opts) => request(`/contests/${id(contestId)}/rankings/snapshots`, opts),
  publishSnapshot: (contestId) => request(`/contests/${id(contestId)}/rankings/snapshots`, { method: 'POST' }),
  getSnapshot: (contestId, version, opts) => request(`/contests/${id(contestId)}/rankings/snapshots/${id(version)}`, opts),

  // --- Matches --------------------------------------------------------------------
  listMatches: (contestId, opts) => request(`/matches${buildQuery({ contest_id: contestId })}`, opts),
  getMatch: (matchId, opts) => request(`/matches/${id(matchId)}`, opts),
  createMatch: (data) => request('/matches', { method: 'POST', json: data }),
  /**
   * Schedules a run. The key identifies one user intent: retrying the same
   * click with the same key returns the same run instead of a new one.
   */
  scheduleRun: (matchId, idempotencyKey = newIdempotencyKey()) =>
    request(`/matches/${id(matchId)}/runs`, { method: 'POST', headers: { 'Idempotency-Key': idempotencyKey } }),
  cancelMatch: (matchId, reason) => request(`/matches/${id(matchId)}/cancel`, { method: 'POST', json: { reason } }),

  // --- Replays ----------------------------------------------------------------------
  getReplay: (replayId, opts) => request(`/replays/${id(replayId)}`, opts),
  async streamReplay(replayId, { signal } = {}) {
    const headers = { Accept: 'application/x-ndjson' };
    const token = getAccessToken();
    if (token) headers.Authorization = `Bearer ${token}`;
    const response = await request(`/replays/${id(replayId)}/stream`, { raw: true, headers, signal, timeout: 60000 });
    return response.text();
  },

  // --- Operations -------------------------------------------------------------------
  readiness: (opts) => request('/admin/readiness', opts),
  baseUrl: BASE_URL,
};
