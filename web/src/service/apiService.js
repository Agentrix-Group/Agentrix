/**
 * High-level API service functions
 */

import { api } from '../api/client.js';

export const ApiService = {
  // Auth
  login: async (username, password) => {
    const data = await api.post('/auth/login', { username, password });
    if (data.token?.access_token) {
      localStorage.setItem('agentrix_token', data.token.access_token);
    }
    return data;
  },
  getCurrentUser: () => api.get('/me'),

  // Contests
  listContests: () => api.get('/contests'),
  getContest: (id) => api.get(`/contests/${id}`),

  // Games
  listGames: () => api.get('/games'),
  getGame: (id) => api.get(`/games/${id}`),

  // Agents
  listAgents: (participantId) => {
    const query = participantId ? `?participant_id=${encodeURIComponent(participantId)}` : '';
    return api.get(`/agents${query}`);
  },

  // Matches
  listMatches: (contestId) => {
    const query = contestId ? `?contest_id=${encodeURIComponent(contestId)}` : '';
    return api.get(`/matches${query}`);
  },
  getMatch: (id) => api.get(`/matches/${id}`),
  scheduleMatch: (matchData) => api.post('/matches', matchData),
  runMatch: (id) => api.post(`/matches/${id}/run`),

  // Rankings
  listRankings: (contestId) => {
    const query = contestId ? `?contest_id=${encodeURIComponent(contestId)}` : '';
    return api.get(`/rankings${query}`);
  },

  // Replays
  getReplay: (id) => api.get(`/replays/${id}`),
  streamReplay: (id) => api.get(`/replays/${id}/stream`),
};
