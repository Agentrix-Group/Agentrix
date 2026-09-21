import React, { useState, useEffect } from 'react';
import { useTranslation } from 'react-i18next';
import { X, Play, Shuffle, AlertCircle, Loader2, Swords } from 'lucide-react';
import { ApiService } from '../service/apiService.js';

export function CreateMatchModal({ isOpen, onClose, onMatchCreated, currentUser }) {
  const { t } = useTranslation(['matches', 'common']);

  const [contests, setContests] = useState([]);
  const [submissions, setSubmissions] = useState([]);
  const [selectedContestId, setSelectedContestId] = useState('');
  const [sub1, setSub1] = useState('');
  const [sub2, setSub2] = useState('');
  const [manualSub1, setManualSub1] = useState('');
  const [manualSub2, setManualSub2] = useState('');
  const [useManual, setUseManual] = useState(false);
  const [seed, setSeed] = useState(() => Math.floor(Math.random() * 900000) + 100000);
  const [runImmediately, setRunImmediately] = useState(true);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [errorMsg, setErrorMsg] = useState('');

  useEffect(() => {
    if (!isOpen) return;

    setErrorMsg('');
    setIsSubmitting(false);

    // Load available contests
    ApiService.listContests()
      .then((data) => setContests(data || []))
      .catch(() => setContests([]));

    // Load available submissions
    ApiService.listSubmissions()
      .then((data) => {
        const list = data || [];
        setSubmissions(list);
        if (list.length >= 2) {
          setSub1(list[0].id);
          setSub2(list[1].id);
        } else if (list.length === 1) {
          setSub1(list[0].id);
          setSub2(list[0].id); // mirror match
        } else {
          setUseManual(true);
        }
      })
      .catch(() => {
        setSubmissions([]);
        setUseManual(true);
      });

    const handleKeyDown = (e) => {
      if (e.key === 'Escape') {
        onClose();
      }
    };
    window.addEventListener('keydown', handleKeyDown);
    return () => window.removeEventListener('keydown', handleKeyDown);
  }, [isOpen, onClose]);

  if (!isOpen) return null;

  const handleRandomizeSeed = () => {
    setSeed(Math.floor(Math.random() * 900000) + 100000);
  };

  const handleSubmit = async (e) => {
    e.preventDefault();
    setErrorMsg('');

    const player1 = useManual ? manualSub1.trim() : sub1;
    const player2 = useManual ? manualSub2.trim() : sub2;

    if (!player1 || !player2) {
      setErrorMsg(t('matches:wizard.validationError', {
        defaultValue: 'Debes especificar las dos submissions competidoras para Starfighter (1v1).',
      }));
      return;
    }

    setIsSubmitting(true);

    try {
      const payload = {
        contest_id: selectedContestId || undefined,
        game_id: 'starfighter',
        submission_ids: [player1, player2],
        seed: Number(seed) || 42,
      };

      const res = await ApiService.scheduleMatch(payload);

      // If scheduled match returned an ID and runImmediately is checked:
      if (runImmediately && res) {
        const matchId = res.match_id || res.match?.id || (res.message ? res.message.match(/Match\s+([a-zA-Z0-9_-]+)\s+scheduled/i)?.[1] : null);
        if (matchId) {
          try {
            await ApiService.runMatch(matchId);
          } catch {
            // Queue may already process it
          }
        }
      }

      if (onMatchCreated) {
        onMatchCreated(res);
      }
      onClose();
    } catch (err) {
      setErrorMsg(err.message || t('matches:messages.createError', { error: 'Unknown' }));
      setIsSubmitting(false);
    }
  };

  return (
    <div className="modal-backdrop" onClick={onClose} role="presentation">
      <div
        className="modal-content"
        onClick={(e) => e.stopPropagation()}
        role="dialog"
        aria-modal="true"
        aria-labelledby="create-match-title"
      >
        <div className="modal-header">
          <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
            <Swords size={20} color="var(--accent)" aria-hidden="true" />
            <h3 id="create-match-title">{t('matches:wizard.title')}</h3>
          </div>
          <button
            type="button"
            className="btn btn-secondary btn-compact"
            onClick={onClose}
            aria-label={t('common:buttons.close')}
            style={{ padding: '4px 8px' }}
          >
            <X size={16} aria-hidden="true" />
          </button>
        </div>

        <p style={{ color: 'var(--text-secondary)', fontSize: '0.88rem', marginTop: '-8px', marginBottom: '16px' }}>
          {t('matches:wizard.subtitle')}
        </p>

        {errorMsg && (
          <div
            role="alert"
            style={{
              padding: '10px 14px',
              background: 'var(--danger-bg)',
              color: 'var(--danger-text)',
              borderRadius: '6px',
              marginBottom: '14px',
              display: 'flex',
              alignItems: 'center',
              gap: '8px',
              fontSize: '0.86rem',
            }}
          >
            <AlertCircle size={18} aria-hidden="true" />
            <span>{errorMsg}</span>
          </div>
        )}

        <form onSubmit={handleSubmit} className="modal-form">
          {/* Game Selection (Locked for Starfighter MVP) */}
          <div className="modal-form-group">
            <label>{t('matches:wizard.gameLabel')}</label>
            <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
              <span className="badge badge-ready">{t('matches:wizard.gameOfficial')}</span>
              <span style={{ fontSize: '0.82rem', color: 'var(--text-secondary)' }}>60 Hz autoritativo</span>
            </div>
          </div>

          {/* Tournament selection */}
          <div className="modal-form-group">
            <label htmlFor="match-contest-select">{t('matches:wizard.contestLabel')}</label>
            <select
              id="match-contest-select"
              value={selectedContestId}
              onChange={(e) => setSelectedContestId(e.target.value)}
            >
              <option value="">{t('matches:wizard.noContest')}</option>
              {contests.map((c) => (
                <option key={c.id} value={c.id}>
                  {c.name}
                </option>
              ))}
            </select>
          </div>

          {/* Submissions selection mode toggle */}
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
            <span style={{ fontSize: '0.85rem', fontWeight: 600 }}>Participantes (1v1)</span>
            <button
              type="button"
              className="btn btn-secondary btn-compact"
              style={{ fontSize: '0.78rem', padding: '2px 8px' }}
              onClick={() => setUseManual(!useManual)}
            >
              {useManual ? 'Seleccionar de lista' : 'Ingresar IDs manuales'}
            </button>
          </div>

          {!useManual && submissions.length > 0 ? (
            <>
              {/* Slot 1: Bot 1 */}
              <div className="modal-form-group">
                <label htmlFor="player-1-select">{t('matches:wizard.player1Label')}</label>
                <select
                  id="player-1-select"
                  value={sub1}
                  onChange={(e) => setSub1(e.target.value)}
                  required
                >
                  <option value="">{t('matches:wizard.selectSubmissionPlaceholder')}</option>
                  {submissions.map((s) => (
                    <option key={s.id} value={s.id}>
                      v{s.version} — {s.language} (#{s.id ? s.id.slice(0, 8) : '—'})
                    </option>
                  ))}
                </select>
              </div>

              {/* Slot 2: Bot 2 */}
              <div className="modal-form-group">
                <label htmlFor="player-2-select">{t('matches:wizard.player2Label')}</label>
                <select
                  id="player-2-select"
                  value={sub2}
                  onChange={(e) => setSub2(e.target.value)}
                  required
                >
                  <option value="">{t('matches:wizard.selectSubmissionPlaceholder')}</option>
                  {submissions.map((s) => (
                    <option key={s.id} value={s.id}>
                      v{s.version} — {s.language} (#{s.id ? s.id.slice(0, 8) : '—'})
                    </option>
                  ))}
                </select>
              </div>
            </>
          ) : (
            <>
              {/* Manual Submission IDs */}
              <div className="modal-form-group">
                <label htmlFor="player-1-input">{t('matches:wizard.player1Label')}</label>
                <input
                  id="player-1-input"
                  type="text"
                  placeholder="ID de submission o bot 1 (ej. sub-001)"
                  value={manualSub1}
                  onChange={(e) => setManualSub1(e.target.value)}
                  required
                />
              </div>

              <div className="modal-form-group">
                <label htmlFor="player-2-input">{t('matches:wizard.player2Label')}</label>
                <input
                  id="player-2-input"
                  type="text"
                  placeholder="ID de submission o bot 2 (ej. sub-002)"
                  value={manualSub2}
                  onChange={(e) => setManualSub2(e.target.value)}
                  required
                />
              </div>
            </>
          )}

          {/* Seed Input */}
          <div className="modal-form-group">
            <label htmlFor="seed-input">{t('matches:wizard.seedLabel')}</label>
            <div style={{ display: 'flex', gap: '8px' }}>
              <input
                id="seed-input"
                type="number"
                value={seed}
                onChange={(e) => setSeed(e.target.value)}
                style={{ flex: 1 }}
                required
              />
              <button
                type="button"
                className="btn btn-secondary"
                onClick={handleRandomizeSeed}
                title={t('matches:wizard.randomSeed')}
                style={{ display: 'inline-flex', alignItems: 'center', gap: '4px' }}
              >
                <Shuffle size={16} aria-hidden="true" />
                <span>{t('matches:wizard.randomSeed')}</span>
              </button>
            </div>
          </div>

          {/* Run immediately checkbox */}
          <label style={{ display: 'flex', alignItems: 'center', gap: '8px', cursor: 'pointer', fontSize: '0.88rem', marginTop: '6px' }}>
            <input
              type="checkbox"
              checked={runImmediately}
              onChange={(e) => setRunImmediately(e.target.checked)}
            />
            <span>{t('matches:wizard.runImmediately')}</span>
          </label>

          {/* Footer Actions */}
          <div className="modal-footer">
            <button
              type="button"
              className="btn btn-secondary"
              onClick={onClose}
              disabled={isSubmitting}
            >
              {t('matches:wizard.cancel')}
            </button>
            <button
              type="submit"
              className="btn"
              disabled={isSubmitting}
              style={{ display: 'inline-flex', alignItems: 'center', gap: '6px' }}
            >
              {isSubmitting ? (
                <>
                  <Loader2 size={16} className="spin" style={{ animation: 'spin 0.8s linear infinite' }} aria-hidden="true" />
                  <span>{t('matches:wizard.submitting')}</span>
                </>
              ) : (
                <>
                  <Play size={16} aria-hidden="true" />
                  <span>{t('matches:wizard.submit')}</span>
                </>
              )}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}
