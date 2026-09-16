import React, { useEffect, useRef, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { ApiService } from '../service/apiService.js';
import { formatNumber } from '../i18n/formatters.js';

export function ReplayViewer({ replayId }) {
  const { t, i18n } = useTranslation(['viewer', 'common']);
  const currentLang = i18n.language?.startsWith('en') ? 'en' : 'es';

  const canvasRef = useRef(null);
  const [replay, setReplay] = useState(null);
  const [currentTick, setCurrentTick] = useState(0);
  const [isPlaying, setIsPlaying] = useState(false);
  // Official Blueprint agent colors: Indigo, Mint, Ocre, Coral
  const colors = ['#6574D9', '#64BFA5', '#E9B760', '#D96C7A'];

  useEffect(() => {
    if (replayId) {
      ApiService.getReplay(replayId)
        .then((res) => {
          setReplay(res.data || res);
          setCurrentTick(0);
        })
        .catch((e) => console.error('Failed to load replay', e));
    }
  }, [replayId]);

  useEffect(() => {
    if (!isPlaying || !replay?.frames) return;
    const interval = setInterval(() => {
      setCurrentTick((prev) => {
        if (prev >= replay.frames.length - 1) {
          setIsPlaying(false);
          return prev;
        }
        return prev + 1;
      });
    }, 250);
    return () => clearInterval(interval);
  }, [isPlaying, replay]);

  useEffect(() => {
    const canvas = canvasRef.current;
    if (!canvas || !replay?.frames || replay.frames.length === 0) return;

    const ctx = canvas.getContext('2d');
    const gridSize = 10;
    const cellSize = canvas.width / gridSize;

    ctx.clearRect(0, 0, canvas.width, canvas.height);

    // Draw clean grid
    ctx.strokeStyle = '#e2e8f0';
    ctx.lineWidth = 1;
    for (let i = 0; i <= gridSize; i++) {
      ctx.beginPath();
      ctx.moveTo(i * cellSize, 0);
      ctx.lineTo(i * cellSize, canvas.height);
      ctx.stroke();

      ctx.beginPath();
      ctx.moveTo(0, i * cellSize);
      ctx.lineTo(canvas.width, i * cellSize);
      ctx.stroke();
    }

    const frame = replay.frames[currentTick];
    if (!frame || !frame.state?.players) return;

    let idx = 0;
    for (const [pId, p] of Object.entries(frame.state.players)) {
      const color = colors[idx % colors.length];
      idx++;

      if (p.alive) {
        const cx = p.x * cellSize + cellSize / 2;
        const cy = p.y * cellSize + cellSize / 2;
        const radius = cellSize * 0.35;

        // Shield aura
        if (p.shielded) {
          ctx.beginPath();
          ctx.arc(cx, cy, radius + 5, 0, Math.PI * 2);
          ctx.strokeStyle = '#60a5fa';
          ctx.lineWidth = 3;
          ctx.stroke();
        }

        ctx.beginPath();
        ctx.arc(cx, cy, radius, 0, Math.PI * 2);
        ctx.fillStyle = color;
        ctx.fill();

        ctx.fillStyle = '#ffffff';
        ctx.font = '10px sans-serif';
        ctx.textAlign = 'center';
        ctx.fillText(pId.substring(0, 5), cx, cy + 3);
      }
    }
  }, [currentTick, replay]);

  if (!replayId) {
    return <div className="card">{t('viewer:selectPrompt')}</div>;
  }

  const currentFrame = replay?.frames?.[currentTick];
  const totalTicks = (replay?.frames?.length || 1) - 1;

  return (
    <div className="card" style={{ maxWidth: '900px', margin: '0 auto' }}>
      <h2>{t('viewer:matchTitle', { matchId: replay?.match_id })}</h2>
      <div style={{ display: 'flex', gap: '20px', marginTop: '16px' }}>
        <div>
          <canvas
            ref={canvasRef}
            width={450}
            height={450}
            style={{ background: '#f8fafc', borderRadius: '8px', border: '1px solid var(--border)' }}
          />
          <div style={{ marginTop: '12px', display: 'flex', gap: '8px' }}>
            <button className="btn" onClick={() => setIsPlaying(!isPlaying)}>
              {isPlaying ? t('viewer:controls.pause') : t('viewer:controls.play')}
            </button>
            <button className="btn btn-secondary" onClick={() => setCurrentTick(Math.max(0, currentTick - 1))}>
              {t('viewer:controls.prev')}
            </button>
            <button className="btn btn-secondary" onClick={() => setCurrentTick(Math.min((replay?.frames?.length || 1) - 1, currentTick + 1))}>
              {t('viewer:controls.next')}
            </button>
            <button className="btn btn-secondary" onClick={() => setCurrentTick(0)}>
              {t('viewer:controls.reset')}
            </button>
          </div>
          <div style={{ marginTop: '8px', fontSize: '0.85rem', color: 'var(--text-secondary)' }}>
            {t('viewer:tickInfo', {
              current: formatNumber(currentTick, currentLang),
              total: formatNumber(totalTicks, currentLang),
            })}
          </div>
        </div>

        <div style={{ flex: 1 }}>
          <h3>{t('viewer:agentsTitle')}</h3>
          {currentFrame?.state?.players && (
            <div style={{ display: 'flex', flexDirection: 'column', gap: '8px', marginBottom: '16px' }}>
              {Object.entries(currentFrame.state.players).map(([id, p]) => (
                <div key={id} style={{ background: 'var(--bg-subtle)', border: '1px solid var(--border)', padding: '10px 14px', borderRadius: '6px' }}>
                  {t('viewer:agentStats', {
                    id,
                    hp: formatNumber(p.hp, currentLang),
                    energy: formatNumber(p.energy, currentLang),
                    score: formatNumber(p.score, currentLang),
                  })}
                </div>
              ))}
            </div>
          )}

          <h3>{t('viewer:eventsTitle')}</h3>
          <div style={{ background: 'var(--bg-subtle)', border: '1px solid var(--border)', height: '200px', overflowY: 'auto', padding: '10px 14px', fontSize: '0.85rem', fontFamily: 'monospace', borderRadius: '6px' }}>
            {currentFrame?.events?.map((ev, i) => (
              <div key={i} style={{ color: 'var(--text-primary)' }}>• {ev}</div>
            ))}
          </div>
        </div>
      </div>
    </div>
  );
}
