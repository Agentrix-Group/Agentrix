import React, { useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { RefreshCw } from 'lucide-react';
import { ApiService } from '../service/apiService.js';
import { MatchCard } from '../components/MatchCard.jsx';

export function MatchesPage({ onWatchReplay, currentUser, canRun }) {
  const { t } = useTranslation(['matches', 'common']);
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
      alert(t('matches:messages.runError', { error: e.message }));
    }
  };

  return (
    <div>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        <h1>{t('matches:title')}</h1>
        <button className="btn" onClick={loadMatches}><RefreshCw size={17} aria-hidden="true" /> {t('matches:refresh')}</button>
      </div>

      {loading ? (
        <p>{t('matches:loading')}</p>
      ) : (
        <div className="grid-cards">
          {matches.length > 0 ? (
            matches.map((m) => (
              <MatchCard
                key={m.id}
                match={m}
                onWatchReplay={onWatchReplay}
                onTriggerRun={handleTriggerRun}
                canRun={canRun !== undefined ? canRun : (currentUser?.role_id === 'admin')}
              />
            ))
          ) : (
            <div className="card">{t('matches:empty')}</div>
          )}
        </div>
      )}
    </div>
  );
}
