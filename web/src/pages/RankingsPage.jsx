import React, { useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { ApiService } from '../service/apiService.js';
import { formatNumber } from '../i18n/formatters.js';

export function RankingsPage() {
  const { t, i18n } = useTranslation(['rankings', 'common']);
  const currentLang = i18n.language?.startsWith('en') ? 'en' : 'es';
  const [rankings, setRankings] = useState([]);

  useEffect(() => {
    ApiService.listRankings().then((data) => setRankings(data || [])).catch(() => {});
  }, []);

  return (
    <div>
      <h1>{t('rankings:title')}</h1>
      <table className="table">
        <thead>
          <tr>
            <th>{t('rankings:table.rank')}</th>
            <th>{t('rankings:table.agent')}</th>
            <th>{t('rankings:table.participant')}</th>
            <th>{t('rankings:table.score')}</th>
            <th>{t('rankings:table.matches')}</th>
            <th>{t('rankings:table.wdl')}</th>
          </tr>
        </thead>
        <tbody>
          {rankings.length > 0 ? (
            rankings.map((r) => (
              <tr key={r.id}>
                <td><strong>#{r.rank}</strong></td>
                <td>{r.agent_id}</td>
                <td>{r.participant_id}</td>
                <td><strong>{formatNumber(r.score, currentLang)}</strong></td>
                <td>{formatNumber(r.matches_played, currentLang)}</td>
                <td>
                  {formatNumber(r.wins, currentLang)} / {formatNumber(r.draws, currentLang)} / {formatNumber(r.losses, currentLang)}
                </td>
              </tr>
            ))
          ) : (
            <tr>
              <td colSpan="6" style={{ textAlign: 'center', color: 'var(--text-secondary)' }}>
                {t('rankings:empty')}
              </td>
            </tr>
          )}
        </tbody>
      </table>
    </div>
  );
}
