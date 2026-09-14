import React, { useEffect, useState } from 'react';
import { ApiService } from '../service/apiService.js';
import { MatchCard } from '../components/MatchCard.jsx';

export function MatchesPage({ onWatchReplay }) {
  const [matches, setMatches] = useState([]);
  const [loading, setLoading] = useState(true);

  const loadMatches = () => {
    setLoading(true);
    ApiService.listMatches()
      .then((data) => {
        setMatches(data || []);
        setLoading(false);
      })
      .catch(() => setLoading(false));
  };

  useEffect(() => {
    loadMatches();
  }, []);

  const handleTriggerRun = async (matchId) => {
    try {
      await ApiService.runMatch(matchId);
      loadMatches();
    } catch (e) {
      alert(`Failed to run match: ${e.message}`);
    }
  };

  return (
    <div>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        <h1>Matches & Evaluations</h1>
        <button className="btn" onClick={loadMatches}>Refresh</button>
      </div>

      {loading ? (
        <p>Loading matches...</p>
      ) : (
        <div className="grid-cards">
          {matches.length > 0 ? (
            matches.map((m) => (
              <MatchCard
                key={m.id}
                match={m}
                onWatchReplay={onWatchReplay}
                onTriggerRun={handleTriggerRun}
              />
            ))
          ) : (
            <div className="card">No matches found.</div>
          )}
        </div>
      )}
    </div>
  );
}
