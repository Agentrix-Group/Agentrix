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
  CheckCircle2,
  AlertTriangle,
} from 'lucide-react';
import { ApiService } from '../service/apiService.js';
import { formatNumber } from '../i18n/formatters.js';
import { drawStarfighterArena, fighterSlot, slotColor } from '../renderers/starfighter/canvasRenderer.js';
import { describeEvent, describeReason } from './eventText.js';
import { parseReplayNDJSON, parseReplayAsync, computeSha256 } from './replayParser.js';

export { parseReplayNDJSON, parseReplayAsync, computeSha256 };

export function ReplayViewer({ replayId, onBrowseMatches, expectedSha256 = null }) {
  const { t, i18n } = useTranslation(['viewer', 'common']);
  const currentLang = i18n.language?.startsWith('en') ? 'en' : 'es';
  const canvasRef = useRef(null);
  const [replay, setReplay] = useState(null);
  const [frameIndex, setFrameIndex] = useState(0);
  const [isPlaying, setIsPlaying] = useState(false);
  const [speed, setSpeed] = useState(1);
  const [error, setError] = useState('');
  const [computedHash, setComputedHash] = useState('');
  const [integrityStatus, setIntegrityStatus] = useState('unverified'); // 'unverified' | 'verified' | 'mismatch' | 'calculating'

  useEffect(() => {
    let active = true;
    setReplay(null);
    setFrameIndex(0);
    setIsPlaying(false);
    setError('');
    setIntegrityStatus('calculating');
    setComputedHash('');

    if (!replayId) return () => {};

    // Fetch replay metadata and stream concurrently
    Promise.allSettled([
      ApiService.getReplay(replayId),
      ApiService.streamReplay(replayId),
    ])
      .then(async ([metaRes, streamRes]) => {
        if (!active) return;
        if (streamRes.status !== 'fulfilled' || !streamRes.value) {
          throw new Error(t('viewer:loadError'));
        }

        const raw = streamRes.value;
        const targetHash = expectedSha256
          || (metaRes.status === 'fulfilled' && metaRes.value?.sha256 ? metaRes.value.sha256 : null);

        const parsed = await parseReplayAsync(raw, targetHash);
        if (!active) return;

        setReplay(parsed.replay);
        setComputedHash(parsed.computedSha256);
        setIntegrityStatus(parsed.integrityStatus);
      })
      .catch((err) => {
        if (active) {
          setError(err?.message || t('viewer:loadError'));
        }
      });

    return () => {
      active = false;
    };
  }, [replayId, expectedSha256, t]);

  const totalFrames = Math.max(0, (replay?.snapshots?.length || 1) - 1);

  // 60 Hz frame playback with requestAnimationFrame and drift compensation
  useEffect(() => {
    if (!isPlaying || !replay?.snapshots?.length) return undefined;

    let animationFrameId;
    let intervalId;
    const frameDurationMs = (replay.metadata.fixed_timestep_ms || 16.67) / speed;

    if (typeof requestAnimationFrame !== 'undefined') {
      let lastTime = typeof performance !== 'undefined' ? performance.now() : Date.now();

      const tickLoop = (now) => {
        const currentTime = now || (typeof performance !== 'undefined' ? performance.now() : Date.now());
        const elapsed = currentTime - lastTime;

        if (elapsed >= frameDurationMs) {
          const framesToAdvance = Math.max(1, Math.floor(elapsed / frameDurationMs));
          lastTime = currentTime - (elapsed % frameDurationMs);

          setFrameIndex((current) => {
            const next = current + framesToAdvance;
            if (next >= replay.snapshots.length - 1) {
              setIsPlaying(false);
              return replay.snapshots.length - 1;
            }
            return next;
          });
        }
        animationFrameId = requestAnimationFrame(tickLoop);
      };

      animationFrameId = requestAnimationFrame(tickLoop);
      return () => cancelAnimationFrame(animationFrameId);
    }

    // Fallback if requestAnimationFrame is not supported in the environment
    intervalId = setInterval(() => {
      setFrameIndex((current) => {
        if (current >= replay.snapshots.length - 1) {
          setIsPlaying(false);
          return current;
        }
        return current + 1;
      });
    }, frameDurationMs);
    return () => clearInterval(intervalId);
  }, [isPlaying, replay, speed]);

  // Keyboard navigation shortcuts
  useEffect(() => {
    const handleKeyDown = (e) => {
      if (e.target && (e.target.tagName === 'INPUT' || e.target.tagName === 'TEXTAREA')) {
        return;
      }
      if (e.key === ' ' || e.code === 'Space') {
        e.preventDefault();
        setIsPlaying((val) => !val);
      } else if (e.key === 'ArrowLeft') {
        e.preventDefault();
        setIsPlaying(false);
        setFrameIndex((val) => Math.max(0, val - 1));
      } else if (e.key === 'ArrowRight') {
        e.preventDefault();
        setIsPlaying(false);
        setFrameIndex((val) => Math.min(totalFrames, val + 1));
      } else if (e.key === 'Home') {
        e.preventDefault();
        setIsPlaying(false);
        setFrameIndex(0);
      } else if (e.key === 'End') {
        e.preventDefault();
        setIsPlaying(false);
        setFrameIndex(totalFrames);
      }
    };

    window.addEventListener('keydown', handleKeyDown);
    return () => window.removeEventListener('keydown', handleKeyDown);
  }, [totalFrames]);

  const currentFrame = replay?.snapshots?.[frameIndex];

  // Canvas rendering on frame change with adaptive arena dimensions
  useEffect(() => {
    if (canvasRef.current && currentFrame) {
      drawStarfighterArena(canvasRef.current, currentFrame, replay?.metadata);
    }
  }, [currentFrame, replay?.metadata]);

  const fighters = currentFrame?.public_snapshot?.fighters || currentFrame?.public_snapshot?.entities || [];
  const events = currentFrame?.events || currentFrame?.public_snapshot?.events || [];
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
          <div style={{ display: 'flex', alignItems: 'center', gap: '8px', flexWrap: 'wrap' }}>
            <span className="eyebrow">STARFIGHTER · REPLAY</span>
            {integrityStatus === 'verified' && (
              <span
                className="badge badge-ready"
                title={t('viewer:integrity.hashTooltip', { hash: computedHash })}
                style={{ display: 'inline-flex', alignItems: 'center', gap: '4px', fontSize: '0.75rem', padding: '2px 8px' }}
              >
                <CheckCircle2 size={13} aria-hidden="true" />
                <span>{t('viewer:integrity.verified')}</span>
              </span>
            )}
            {integrityStatus === 'mismatch' && (
              <span
                className="badge badge-failed"
                title={t('viewer:integrity.hashTooltip', { hash: computedHash })}
                style={{ display: 'inline-flex', alignItems: 'center', gap: '4px', fontSize: '0.75rem', padding: '2px 8px' }}
              >
                <AlertTriangle size={13} aria-hidden="true" />
                <span>{t('viewer:integrity.mismatch')}</span>
              </span>
            )}
          </div>
          <h2>{t('viewer:matchTitle', { matchId: matchLabel })}</h2>
          <p>{t('viewer:authoritative')}</p>
        </div>
        <div className="replay-result">
          <span>{t('viewer:result')}</span>
          <strong>{replay.result.winner || t('viewer:draw')}</strong>
          <small>{describeReason(replay.result.reason, t)}</small>
        </div>
      </header>

      <div className="replay-layout">
        <div className="arena-panel">
          <canvas
            ref={canvasRef}
            width={960}
            height={540}
            className="starfighter-canvas"
            aria-label="Starfighter 2D Canvas Arena"
          />
          <div className="playback-bar">
            <div className="playback-buttons">
              <button
                type="button"
                className="icon-button"
                onClick={() => setFrameIndex(0)}
                aria-label={t('viewer:controls.reset')}
                title={t('viewer:controls.reset')}
              >
                <RotateCcw size={18} />
              </button>
              <button
                type="button"
                className="icon-button"
                onClick={() => step(-1)}
                aria-label={t('viewer:controls.prev')}
                title={t('viewer:controls.prev')}
              >
                <SkipBack size={18} />
              </button>
              <button
                type="button"
                className="play-button"
                onClick={() => setIsPlaying((value) => !value)}
                aria-label={isPlaying ? t('viewer:controls.pause') : t('viewer:controls.play')}
              >
                {isPlaying ? <CirclePause size={21} /> : <CirclePlay size={21} />}
                {isPlaying ? t('viewer:controls.pause') : t('viewer:controls.play')}
              </button>
              <button
                type="button"
                className="icon-button"
                onClick={() => step(1)}
                aria-label={t('viewer:controls.next')}
                title={t('viewer:controls.next')}
              >
                <SkipForward size={18} />
              </button>
            </div>
            <div className="speed-control" aria-label={t('viewer:controls.speed')}>
              <Gauge size={17} aria-hidden="true" />
              {[0.5, 1, 2, 4].map((value) => (
                <button
                  key={value}
                  type="button"
                  className={speed === value ? 'active' : ''}
                  onClick={() => setSpeed(value)}
                >
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
            <span>
              {t('viewer:tickInfo', {
                current: formatNumber(currentFrame?.tick || 0, currentLang),
                total: formatNumber(totalFrames, currentLang),
              })}
            </span>
            <span title={computedHash ? `SHA-256: ${computedHash}` : undefined}>
              {currentFrame?.state_hash ? `${currentFrame.state_hash.slice(0, 14)}…` : '—'}
            </span>
          </div>
        </div>

        <aside className="replay-sidebar">
          <div className="viewer-section">
            <h3><Swords size={18} /> {t('viewer:agentsTitle')}</h3>
            <div className="fighter-list">
              {fighters.map((fighter, index) => (
                <article className="fighter-card" key={fighterSlot(fighter) || index}>
                  <span
                    className="fighter-dot"
                    style={{ background: slotColor(fighterSlot(fighter), replay?.metadata?.participants, index) }}
                  />
                  <div>
                    <strong>{fighterSlot(fighter)}</strong>
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
              {events.length ? events.map((event, index) => {
                const { kind, text } = describeEvent(event, replay?.metadata?.participants, t);
                return (
                  <div key={`${currentFrame.tick}-${index}`} className={`event-${kind}`}>
                    <span>{currentFrame.tick}</span>
                    {text}
                  </div>
                );
              }) : <p>{t('viewer:noEvents')}</p>}
            </div>
          </div>
        </aside>
      </div>
    </section>
  );
}
