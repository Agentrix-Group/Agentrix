/**
 * High-level API service functions for Agentrix
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
  register: (username, email, password) =>
    api.post('/auth/register', { username, email, password }),
  logout: () => {
    localStorage.removeItem('agentrix_token');
  },
  getCurrentUser: () => api.get('/me'),

  // Contests
  listContests: () => api.get('/contests'),
  getContest: (id) => api.get(`/contests/${id}`),
  listContestAgents: (contestId) => api.get(`/contests/${contestId}/agents`),
  enrollAgent: (contestId, agentId) =>
    api.post(`/contests/${contestId}/agents`, { agent_id: agentId }),

  // Games
  listGames: () => api.get('/games'),
  getGame: (id) => api.get(`/games/${id}`),

  // Agents
  listAgents: (participantId) => {
    const query = participantId ? `?participant_id=${encodeURIComponent(participantId)}` : '';
    return api.get(`/agents${query}`);
  },
  createAgent: (agentData) => api.post('/agents', agentData),

  // Submissions
  listSubmissions: (agentId) => {
    const query = agentId ? `?agent_id=${encodeURIComponent(agentId)}` : '';
    return api.get(`/submissions${query}`);
  },
  uploadBotBundle: (agentId, file) => {
    const form = new FormData();
    form.append('agent_id', agentId);
    form.append('bundle', file);
    return api.form('/submissions/upload', form);
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
  streamReplay: (id) => api.text(`/replays/${id}/stream`),
};
