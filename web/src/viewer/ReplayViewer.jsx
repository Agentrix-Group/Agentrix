import React, { useEffect, useMemo, useRef, useState } from 'react';
import { useTranslation } from 'react-i18next';
import {
  CirclePause,
  CirclePlay,
  Gauge,
  RotateCcw,
  Shield,
  SkipBack,
  SkipForward,
  Swords,
} from 'lucide-react';
import { ApiService } from '../service/apiService.js';
import { formatNumber } from '../i18n/formatters.js';
import { drawStarfighterArena, PLAYER_COLORS } from '../renderers/starfighter/canvasRenderer.js';

export function parseReplayNDJSON(raw) {
  const records = raw
    .split(/\r?\n/)
    .map((line) => line.trim())
    .filter(Boolean)
    .map((line) => JSON.parse(line));
  const metadata = records[0];
  if (records.length < 3 || metadata?.type !== 'metadata') {
    throw new Error('Replay metadata is missing');
  }
  if (
    !metadata.replay_id
    || !metadata.match_id
    || metadata.game_id !== 'starfighter'
    || metadata.participants?.length !== 2
    || metadata.participants.some((id) => !id)
    || metadata.participants[0] === metadata.participants[1]
    || !Number.isInteger(metadata.seed)
    || metadata.fixed_timestep_ms <= 0
    || Number.isNaN(Date.parse(metadata.created_at))
  ) {
    throw new Error('Replay metadata is invalid');
  }
  const result = records.at(-1);
  if (result.type !== 'result') {
    throw new Error('Replay result is missing');
  }
  const snapshots = records.slice(1, -1);
  if (snapshots.some((record) => record.type !== 'snapshot')) {
    throw new Error('Replay contains an unknown record');
  }
  snapshots.forEach((frame, index) => {
    if (
      frame.tick !== index
      || frame.public_snapshot?.tick !== index
      || !frame.state_hash
      || frame.public_snapshot?.stateHash !== frame.state_hash
    ) {
      throw new Error(`Replay tick mismatch at frame ${index}`);
    }
  });
  const lastSnapshot = snapshots.at(-1);
  if (!lastSnapshot || result.final_tick !== lastSnapshot.tick || result.final_state_hash !== lastSnapshot.state_hash) {
    throw new Error('Replay result does not seal the final snapshot');
  }
  if (
    !['eliminated', 'timeout', 'score_limit'].includes(result.reason)
    || !result.scores
    || Number.isNaN(Date.parse(result.finished_at))
  ) {
    throw new Error('Replay result reason is invalid');
  }
  return { metadata, snapshots, result };
}

export function ReplayViewer({ replayId, onBrowseMatches }) {
  const { t, i18n } = useTranslation(['viewer', 'common']);
  const currentLang = i18n.language?.startsWith('en') ? 'en' : 'es';
  const canvasRef = useRef(null);
  const [replay, setReplay] = useState(null);
  const [frameIndex, setFrameIndex] = useState(0);
  const [isPlaying, setIsPlaying] = useState(false);
  const [speed, setSpeed] = useState(1);
  const [error, setError] = useState('');

  useEffect(() => {
    let active = true;
    setReplay(null);
    setFrameIndex(0);
    setIsPlaying(false);
    setError('');
    if (!replayId) return () => {};
    ApiService.streamReplay(replayId)
      .then((raw) => {
        if (active) setReplay(parseReplayNDJSON(raw));
      })
      .catch(() => {
        if (active) setError(t('viewer:loadError'));
      });
    return () => {
      active = false;
    };
  }, [replayId, t]);

  useEffect(() => {
    if (!isPlaying || !replay?.snapshots?.length) return undefined;
    const timer = window.setInterval(() => {
      setFrameIndex((current) => {
        if (current >= replay.snapshots.length - 1) {
          setIsPlaying(false);
          return current;
        }
        return current + 1;
      });
    }, replay.metadata.fixed_timestep_ms / speed);
    return () => window.clearInterval(timer);
  }, [isPlaying, replay, speed]);

  const currentFrame = replay?.snapshots?.[frameIndex];
  useEffect(() => {
    if (canvasRef.current && currentFrame) drawStarfighterArena(canvasRef.current, currentFrame);
  }, [currentFrame]);

  const fighters = currentFrame?.public_snapshot?.fighters || currentFrame?.public_snapshot?.entities || [];
  const events = currentFrame?.events || currentFrame?.public_snapshot?.events || [];
  const totalFrames = Math.max(0, (replay?.snapshots?.length || 1) - 1);
  const matchLabel = useMemo(
    () => replay?.metadata?.match_id?.slice(0, 12) || replayId?.slice(0, 12),
    [replay, replayId],
  );

  if (!replayId) {
    return (
      <div className="viewer-empty">
        <Swords size={32} aria-hidden="true" />
        <p>{t('viewer:selectPrompt')}</p>
        {typeof onBrowseMatches === 'function' && (
          <button className="btn btn-secondary" style={{ marginTop: '12px' }} onClick={onBrowseMatches}>
            <Swords size={16} aria-hidden="true" /> {t('viewer:browseMatches')}
          </button>
        )}
      </div>
    );
  }
  if (error) return <div className="viewer-error">{error}</div>;
  if (!replay) return <div className="viewer-empty">{t('viewer:loading')}</div>;

  const step = (offset) => setFrameIndex((value) => Math.max(0, Math.min(totalFrames, value + offset)));

  return (
    <section className="replay-shell" aria-label={t('viewer:title')}>
      <header className="replay-header">
        <div>
          <span className="eyebrow">STARFIGHTER · REPLAY</span>
          <h2>{t('viewer:matchTitle', { matchId: matchLabel })}</h2>
          <p>{t('viewer:authoritative')}</p>
        </div>
        <div className="replay-result">
          <span>{t('viewer:result')}</span>
          <strong>{replay.result.winner || t('viewer:draw')}</strong>
          <small>{replay.result.reason}</small>
        </div>
      </header>

      <div className="replay-layout">
        <div className="arena-panel">
          <canvas ref={canvasRef} width={960} height={540} className="starfighter-canvas" />
          <div className="playback-bar">
            <div className="playback-buttons">
              <button className="icon-button" onClick={() => setFrameIndex(0)} aria-label={t('viewer:controls.reset')} title={t('viewer:controls.reset')}>
                <RotateCcw size={18} />
              </button>
              <button className="icon-button" onClick={() => step(-1)} aria-label={t('viewer:controls.prev')} title={t('viewer:controls.prev')}>
                <SkipBack size={18} />
              </button>
              <button className="play-button" onClick={() => setIsPlaying((value) => !value)}>
                {isPlaying ? <CirclePause size={21} /> : <CirclePlay size={21} />}
                {isPlaying ? t('viewer:controls.pause') : t('viewer:controls.play')}
              </button>
              <button className="icon-button" onClick={() => step(1)} aria-label={t('viewer:controls.next')} title={t('viewer:controls.next')}>
                <SkipForward size={18} />
              </button>
            </div>
            <div className="speed-control" aria-label={t('viewer:controls.speed')}>
              <Gauge size={17} aria-hidden="true" />
              {[0.5, 1, 2, 4].map((value) => (
                <button key={value} className={speed === value ? 'active' : ''} onClick={() => setSpeed(value)}>
                  {value}×
                </button>
              ))}
            </div>
          </div>
          <input
            className="timeline"
            type="range"
            min="0"
            max={totalFrames}
            value={frameIndex}
            onChange={(event) => {
              setIsPlaying(false);
              setFrameIndex(Number(event.target.value));
            }}
            aria-label={t('viewer:controls.timeline')}
          />
          <div className="timeline-labels">
            <span>{t('viewer:tickInfo', { current: formatNumber(currentFrame?.tick || 0, currentLang), total: formatNumber(totalFrames, currentLang) })}</span>
            <span>{currentFrame?.state_hash?.slice(0, 14)}…</span>
          </div>
        </div>

        <aside className="replay-sidebar">
          <div className="viewer-section">
            <h3><Swords size={18} /> {t('viewer:agentsTitle')}</h3>
            <div className="fighter-list">
              {fighters.map((fighter, index) => (
                <article className="fighter-card" key={fighter.playerId}>
                  <span className="fighter-dot" style={{ background: PLAYER_COLORS[index % PLAYER_COLORS.length] }} />
                  <div>
                    <strong>{fighter.playerId}</strong>
                    <span>{t('viewer:health', { value: Math.round(fighter.health) })}</span>
                  </div>
                  {fighter.shieldActive && <Shield size={18} aria-label={t('viewer:shieldActive')} />}
                </article>
              ))}
            </div>
          </div>
          <div className="viewer-section event-section">
            <h3>{t('viewer:eventsTitle')}</h3>
            <div className="event-log" aria-live="polite">
              {events.length ? events.map((event, index) => (
                <div key={`${currentFrame.tick}-${index}`}><span>{currentFrame.tick}</span>{event}</div>
              )) : <p>{t('viewer:noEvents')}</p>}
            </div>
          </div>
        </aside>
      </div>
    </section>
  );
}
