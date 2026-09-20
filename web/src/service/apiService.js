/**
 * High-level API service functions for Agentrix
 */

import { api, buildUrl } from '../api/client.js';

export const ApiService = {
  // Auth
  login: async (username, password) => {
    const data = await api.post('/auth/login', { username, password });
    if (data.token?.access_token) {
      localStorage.setItem('agentrix_token', data.token.access_token);
    }
    return data;
  },
  register: (username, email, password) =>
    api.post('/auth/register', { username, email, password }),
  logout: () => {
    localStorage.removeItem('agentrix_token');
  },
  getCurrentUser: () => api.get('/me'),

  // Contests
  listContests: () => api.get('/contests'),
  getContest: (id) => api.get(`/contests/${encodeURIComponent(id)}`),
  listContestAgents: (contestId) => api.get(`/contests/${encodeURIComponent(contestId)}/agents`),
  enrollAgent: (contestId, agentId) =>
    api.post(`/contests/${encodeURIComponent(contestId)}/agents`, { agent_id: agentId }),

  // Games
  listGames: () => api.get('/games'),
  getGame: (id) => api.get(`/games/${encodeURIComponent(id)}`),

  // Agents
  listAgents: (participantId) =>
    api.get(buildUrl('/agents', participantId ? { participant_id: participantId } : {})),
  createAgent: (agentData) => api.post('/agents', agentData),

  // Submissions
  listSubmissions: (agentId) =>
    api.get(buildUrl('/submissions', agentId ? { agent_id: agentId } : {})),
  uploadBotBundle: (agentId, file) => {
    const form = new FormData();
    form.append('agent_id', agentId);
    form.append('bundle', file);
    return api.form('/submissions/upload', form);
  },

  // Matches
  listMatches: (contestId) =>
    api.get(buildUrl('/matches', contestId ? { contest_id: contestId } : {})),
  getMatch: (id) => api.get(`/matches/${encodeURIComponent(id)}`),
  scheduleMatch: (matchData) => api.post('/matches', matchData),
  runMatch: (id) => api.post(`/matches/${encodeURIComponent(id)}/run`),

  // Rankings
  listRankings: (contestId) =>
    api.get(buildUrl('/rankings', contestId ? { contest_id: contestId } : {})),

  // Replays
  getReplay: (id) => api.get(`/replays/${encodeURIComponent(id)}`),
  streamReplay: (id) => api.text(`/replays/${encodeURIComponent(id)}/stream`),
};
