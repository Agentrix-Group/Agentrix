/**
 * DTO Contracts and Type Definitions for Agentrix Web
 * Defines the shapes expected from the backend API according to open-api contracts.
 * Resolves F1.3 and establishes Sprint FQ-0 typing foundation.
 */

/**
 * @typedef {'pending' | 'running' | 'finished' | 'failed' | 'cancelled'} MatchStatus
 */

/**
 * @typedef {'published' | 'registration_open' | 'preparation' | 'in_progress' | 'final_selection' | 'live_final' | 'finished' | 'archived'} ContestState
 */

/**
 * @typedef {'pending_validation' | 'validating' | 'ready' | 'rejected'} SubmissionStatus
 */

/**
 * @typedef {Object} User
 * @property {string} id
 * @property {string} username
 * @property {string} [email]
 * @property {string} [role_id]
 * @property {string} [role]
 */

/**
 * @typedef {Object} Contest
 * @property {string} id
 * @property {string} name
 * @property {string} [description]
 * @property {ContestState} state
 * @property {string} [game_id]
 * @property {string} [starts_at]
 * @property {string} [ends_at]
 * @property {string} [created_at]
 */

/**
 * @typedef {Object} Agent
 * @property {string} id
 * @property {string} name
 * @property {string} [description]
 * @property {string} [game_id]
 * @property {string} [owner_user_id]
 * @property {string} [created_at]
 */

/**
 * @typedef {Object} Submission
 * @property {string} id
 * @property {string} agent_id
 * @property {number} version
 * @property {string} language
 * @property {SubmissionStatus} status
 * @property {boolean} active
 * @property {string} created_at
 * @property {Agent} [agent]
 */

/**
 * @typedef {Object} MatchResult
 * @property {string} id
 * @property {number} rank
 * @property {string} submission_id
 * @property {number} score
 */

/**
 * @typedef {Object} Match
 * @property {string} id
 * @property {string} game_id
 * @property {MatchStatus} status
 * @property {number} [seed]
 * @property {string} [contest_id]
 * @property {string} [replay_id]
 * @property {MatchResult[]} [results]
 * @property {string} [created_at]
 * @property {string} [finished_at]
 */

/**
 * @typedef {Object} RankingEntry
 * @property {string} id
 * @property {number} rank
 * @property {string} agent_id
 * @property {string} user_id
 * @property {number} score
 * @property {number} matches_played
 * @property {number} wins
 * @property {number} draws
 * @property {number} losses
 */

/**
 * Runtime DTO validator helpers
 */
export function isValidMatch(match) {
  return Boolean(
    match &&
    typeof match.id === 'string' &&
    typeof match.game_id === 'string' &&
    typeof match.status === 'string'
  );
}

export function isValidContest(contest) {
  return Boolean(
    contest &&
    typeof contest.id === 'string' &&
    typeof contest.name === 'string' &&
    typeof contest.state === 'string'
  );
}

export function isValidSubmission(sub) {
  return Boolean(
    sub &&
    typeof sub.id === 'string' &&
    typeof sub.agent_id === 'string' &&
    typeof sub.version === 'number' &&
    typeof sub.language === 'string' &&
    typeof sub.status === 'string'
  );
}
