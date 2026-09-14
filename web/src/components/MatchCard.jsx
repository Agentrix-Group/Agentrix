import React from 'react';

export function MatchCard({ match, onWatchReplay, onTriggerRun }) {
  const getBadgeClass = (status) => {
    switch (status) {
      case 'running': return 'badge-running';
      case 'finished': return 'badge-finished';
      case 'failed': return 'badge-failed';
      default: return 'badge-pending';
    }
  };

  return (
    <div className="card">
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        <strong>Match #{match.id.substring(0, 8)}</strong>
        <span className={`badge ${getBadgeClass(match.status)}`}>{match.status}</span>
      </div>
      <p style={{ color: 'var(--text-secondary)', fontSize: '0.9rem', margin: '8px 0' }}>
        Game: {match.game_id} | Seed: {match.seed}
      </p>

      {match.results && match.results.length > 0 && (
        <div style={{ margin: '12px 0', fontSize: '0.85rem' }}>
          <strong>Results:</strong>
          {match.results.map((r) => (
            <div key={r.id}>
              Rank {r.rank}: {r.submission_id} ({r.score} pts)
            </div>
          ))}
        </div>
      )}

      <div style={{ display: 'flex', gap: '8px', marginTop: '16px' }}>
        {match.status === 'finished' && match.replay_id && (
          <button className="btn" onClick={() => onWatchReplay(match.replay_id)}>
            Watch Replay
          </button>
        )}
        {match.status === 'pending' && (
          <button className="btn" onClick={() => onTriggerRun(match.id)}>
            Run Match
          </button>
        )}
      </div>
    </div>
  );
}
