import React, { useEffect, useState } from 'react';
import { ApiService } from '../service/apiService.js';
import { MatchCard } from '../components/MatchCard.jsx';

export function HomePage({ onWatchReplay }) {
  const [contests, setContests] = useState([]);
  const [recentMatches, setRecentMatches] = useState([]);

  useEffect(() => {
    ApiService.listContests().then(setContests).catch(() => {});
    ApiService.listMatches().then((matches) => setRecentMatches(matches.slice(0, 6))).catch(() => {});
  }, []);

  return (
    <div>
      <h1>Agentrix Overview</h1>
      <p style={{ color: 'var(--text-secondary)' }}>
        Autonomous agent competition and evaluation platform.
      </p>

      <section style={{ marginTop: '32px' }}>
        <h2>Active Contests</h2>
        <div className="grid-cards">
          {contests.length > 0 ? (
            contests.map((c) => (
              <div key={c.id} className="card">
                <h3>{c.name}</h3>
                <p>{c.description}</p>
                <div style={{ fontSize: '0.85rem', color: 'var(--text-secondary)' }}>
                  Game: {c.game_id} | Status: <span className="badge badge-finished">{c.status}</span>
                </div>
              </div>
            ))
          ) : (
            <div className="card">No active contests found.</div>
          )}
        </div>
      </section>

      <section style={{ marginTop: '32px' }}>
        <h2>Recent Matches</h2>
        <div className="grid-cards">
          {recentMatches.length > 0 ? (
            recentMatches.map((m) => (
              <MatchCard key={m.id} match={m} onWatchReplay={onWatchReplay} onTriggerRun={() => {}} />
            ))
          ) : (
            <div className="card">No matches played yet.</div>
          )}
        </div>
      </section>
    </div>
  );
}
