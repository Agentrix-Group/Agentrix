import React, { useEffect, useState } from 'react';
import { ApiService } from '../service/apiService.js';

export function RankingsPage() {
  const [rankings, setRankings] = useState([]);

  useEffect(() => {
    ApiService.listRankings().then((data) => setRankings(data || [])).catch(() => {});
  }, []);

  return (
    <div>
      <h1>Tournament Leaderboard</h1>
      <table className="table">
        <thead>
          <tr>
            <th>Rank</th>
            <th>Agent</th>
            <th>Participant</th>
            <th>Score</th>
            <th>Matches</th>
            <th>W / D / L</th>
          </tr>
        </thead>
        <tbody>
          {rankings.length > 0 ? (
            rankings.map((r) => (
              <tr key={r.id}>
                <td><strong>#{r.rank}</strong></td>
                <td>{r.agent_id}</td>
                <td>{r.participant_id}</td>
                <td><strong>{r.score}</strong></td>
                <td>{r.matches_played}</td>
                <td>{r.wins} / {r.draws} / {r.losses}</td>
              </tr>
            ))
          ) : (
            <tr>
              <td colSpan="6" style={{ textAlign: 'center', color: 'var(--text-secondary)' }}>
                No rankings data available yet.
              </td>
            </tr>
          )}
        </tbody>
      </table>
    </div>
  );
}
