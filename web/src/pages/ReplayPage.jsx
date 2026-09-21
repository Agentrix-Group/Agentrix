import React from 'react';
import { useTranslation } from 'react-i18next';
import { ApiService } from '../service/apiService.js';
import { useResource } from '../hooks/hooks.js';
import { Link } from '../router/Router.jsx';
import { LoadingState, ErrorState } from '../components/States.jsx';
import { parseReplay } from '../viewer/replayParser.js';
import { ReplayViewer } from '../viewer/ReplayViewer.jsx';

export default function ReplayPage({ replayId }) {
  const { t } = useTranslation('viewer');
  const data = useResource(async (signal) => {
    const meta = await ApiService.getReplay(replayId, { signal });
    const [raw, match] = await Promise.all([
      ApiService.streamReplay(replayId, { signal }),
      ApiService.getMatch(meta.match_id, { signal }),
    ]);
    const replay = parseReplay(raw);
    const names = Object.fromEntries(match.slots.map((s) => [s.id, s.display_name]));
    return { meta, replay, match, names };
  }, [replayId]);

  if (data.loading) return <LoadingState message={t('loading')} />;
  if (data.error) return <ErrorState error={data.error} onRetry={data.reload} title={t('unavailable')} />;
  const { meta, replay, match, names } = data.data;
  const winner = replay.result.winner ? names[replay.result.winner] || replay.result.winner : t('noWinner');
  return (
    <div className="page">
      <div className="page-header">
        <div>
          <p className="eyebrow"><Link to={`/matches/${match.id}`}>{t('backToMatch')}</Link></p>
          <h1>{match.slots.map((s) => s.display_name).join(' vs ')}</h1>
        </div>
      </div>
      <ReplayViewer replay={replay} names={names} />
      <section className="card">
        <dl className="detail-grid">
          <dt>{t('winner')}</dt><dd>{winner}</dd>
          <dt>{t('reason')}</dt><dd>{t(`reasons.${replay.result.reason}`, { defaultValue: replay.result.reason })}</dd>
          <dt>{t('frames')}</dt><dd>{meta.frame_count}</dd>
          <dt>{t('digest')}</dt><dd><code>{meta.sha256}</code> <span className="muted small">({t('verifiedServerSide')})</span></dd>
          <dt>{t('finalHash')}</dt><dd><code>{replay.result.final_state_hash}</code></dd>
          <dt>{t('engine')}</dt><dd><code>{replay.metadata.engine_version} · {replay.metadata.engine_sha256.slice(0, 16)}</code></dd>
          <dt>{t('spec')}</dt><dd><code>{replay.metadata.spec_hash.slice(0, 16)}</code></dd>
        </dl>
      </section>
    </div>
  );
}
