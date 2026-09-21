import React, { useEffect, useMemo, useRef, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Play, Pause, SkipBack, SkipForward } from 'lucide-react';
import { drawStarfighterArena } from '../renderers/starfighter/canvasRenderer.js';
import { tickIntervalMs } from './replayParser.js';
import { usePrefersReducedMotion } from '../hooks/hooks.js';

const SPEEDS = [0.5, 1, 2, 4];

/** Plays a parsed replay. Only the current frame is drawn; the animation
 * loop advances by simulated time derived from the replay's tick rate. */
export function ReplayViewer({ replay, names = {} }) {
  const { t } = useTranslation('viewer');
  const canvasRef = useRef(null);
  const reducedMotion = usePrefersReducedMotion();
  const [frameIndex, setFrameIndex] = useState(0);
  const [playing, setPlaying] = useState(!reducedMotion);
  const [speed, setSpeed] = useState(1);
  const last = replay.frames.length - 1;
  const interval = useMemo(() => tickIntervalMs(replay.metadata.tick_rate), [replay]);

  useEffect(() => {
    if (!playing) return undefined;
    let raf;
    let previous = performance.now();
    let carry = 0;
    const loop = (now) => {
      carry += (now - previous) * speed;
      previous = now;
      const steps = Math.floor(carry / interval);
      if (steps > 0) {
        carry -= steps * interval;
        setFrameIndex((i) => {
          const next = Math.min(last, i + steps);
          if (next === last) setPlaying(false);
          return next;
        });
      }
      raf = requestAnimationFrame(loop);
    };
    raf = requestAnimationFrame(loop);
    return () => cancelAnimationFrame(raf);
  }, [playing, speed, interval, last]);

  useEffect(() => {
    const frame = replay.frames[frameIndex];
    const labeled = {
      ...frame,
      public_snapshot: {
        ...frame.public_snapshot,
        fighters: (frame.public_snapshot.fighters || []).map((f) => ({ ...f, playerId: names[f.playerId] || f.playerId })),
      },
    };
    drawStarfighterArena(canvasRef.current, labeled);
  }, [frameIndex, replay, names]);

  const frame = replay.frames[frameIndex];
  const seconds = ((frameIndex * interval) / 1000).toFixed(2);
  return (
    <div className="viewer">
      <canvas ref={canvasRef} width={1000} height={520} className="viewer-canvas" role="img"
        aria-label={t('canvasLabel', { tick: frame.tick, total: last })} />
      <div className="viewer-controls">
        <button type="button" className="icon-button" onClick={() => setFrameIndex(0)} aria-label={t('restart')}><SkipBack size={18} aria-hidden="true" /></button>
        <button type="button" className="btn" onClick={() => { if (frameIndex === last) setFrameIndex(0); setPlaying(!playing); }}>
          {playing ? <Pause size={16} aria-hidden="true" /> : <Play size={16} aria-hidden="true" />} {playing ? t('pause') : t('play')}
        </button>
        <button type="button" className="icon-button" onClick={() => setFrameIndex(Math.min(last, frameIndex + 1))} aria-label={t('step')}><SkipForward size={18} aria-hidden="true" /></button>
        <label className="viewer-scrub">
          <span className="visually-hidden">{t('position')}</span>
          <input type="range" min={0} max={last} value={frameIndex} onChange={(e) => { setPlaying(false); setFrameIndex(Number(e.target.value)); }} />
        </label>
        <label className="field inline"><span>{t('speed')}</span>
          <select value={speed} onChange={(e) => setSpeed(Number(e.target.value))}>
            {SPEEDS.map((s) => <option key={s} value={s}>{s}×</option>)}
          </select>
        </label>
      </div>
      <p className="muted small" aria-live="off">
        {t('tick', { tick: frame.tick, total: last })} · {seconds}s · {replay.metadata.tick_rate.numerator}/{replay.metadata.tick_rate.denominator} Hz · <code>{frame.state_hash.slice(0, 12)}</code>
      </p>
      {frame.events?.length > 0 && <ul className="event-list">{frame.events.slice(0, 5).map((ev, i) => <li key={i}>{ev}</li>)}</ul>}
    </div>
  );
}
