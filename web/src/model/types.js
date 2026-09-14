/**
 * Model definitions and enumerations for Agentrix Web
 */

export const MatchStatus = {
  PENDING: 'pending',
  RUNNING: 'running',
  FINISHED: 'finished',
  FAILED: 'failed'
};

export const Roles = {
  ADMIN: 'admin',
  PARTICIPANT: 'participant',
  REFEREE: 'referee',
  SPECTATOR: 'spectator'
};

export const ActionTypes = {
  UP: 'UP',
  DOWN: 'DOWN',
  LEFT: 'LEFT',
  RIGHT: 'RIGHT',
  ATTACK: 'ATTACK',
  SHIELD: 'SHIELD',
  REST: 'REST'
};
