import React, { useState, useEffect } from 'react';
import { useTranslation } from 'react-i18next';
import { X, Play, Shuffle, AlertCircle, Loader2, Swords, Plus, Trash2 } from 'lucide-react';
import { ApiService } from '../service/apiService.js';

// Starfighter 0.4.0 admite de 2 a 5 jugadores todos contra todos (ADR-0013).
export const MIN_PLAYERS = 2;
export const MAX_PLAYERS = 5;

/**
 * Devuelve los ids limpios si forman una partida válida (entre MIN_PLAYERS y
 * MAX_PLAYERS submissions distintas y no vacías), o null si no.
 */
export function validParticipants(ids) {
  const cleaned = ids.map((id) => id.trim());
  if (cleaned.length < MIN_PLAYERS || cleaned.length > MAX_PLAYERS) return null;
  if (cleaned.some((id) => !id)) return null;
  if (new Set(cleaned).size !== cleaned.length) return null;
  return cleaned;
}

export function CreateMatchModal({ isOpen, onClose, onMatchCreated, currentUser }) {
  const { t } = useTranslation(['matches', 'common']);

  const [contests, setContests] = useState([]);
  const [submissions, setSubmissions] = useState([]);
  const [selectedContestId, setSelectedContestId] = useState('');
  // Un valor por slot; ambas listas tienen siempre la misma longitud.
  const [selected, setSelected] = useState(['', '']);
  const [manualIds, setManualIds] = useState(['', '']);
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
        // Una misma submission no puede ocupar dos slots: con una sola
        // disponible, el segundo slot queda vacío para completarlo a mano.
        if (list.length >= 1) {
          setSelected([list[0].id, list[1]?.id || '']);
          setManualIds(['', '']);
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

  const slotCount = selected.length;
  const slotValues = useManual ? manualIds : selected;

  const updateSlot = (index, value) => {
    const setter = useManual ? setManualIds : setSelected;
    setter((prev) => prev.map((current, i) => (i === index ? value : current)));
  };

  const addSlot = () => {
    if (slotCount >= MAX_PLAYERS) return;
    setSelected((prev) => [...prev, '']);
    setManualIds((prev) => [...prev, '']);
  };

  const removeSlot = (index) => {
    if (slotCount <= MIN_PLAYERS) return;
    setSelected((prev) => prev.filter((_, i) => i !== index));
    setManualIds((prev) => prev.filter((_, i) => i !== index));
  };

  const handleSubmit = async (e) => {
    e.preventDefault();
    setErrorMsg('');

    const participants = validParticipants(slotValues);
    if (!participants) {
      setErrorMsg(t('matches:wizard.validationError', { min: MIN_PLAYERS, max: MAX_PLAYERS }));
      return;
    }

    setIsSubmitting(true);

    try {
      const payload = {
        contest_id: selectedContestId || undefined,
        game_id: 'starfighter',
        submission_ids: participants,
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
            <span style={{ fontSize: '0.85rem', fontWeight: 600 }}>
              {t('matches:wizard.participantsLabel', { count: slotCount, min: MIN_PLAYERS, max: MAX_PLAYERS })}
            </span>
            <button
              type="button"
              className="btn btn-secondary btn-compact"
              style={{ fontSize: '0.78rem', padding: '2px 8px' }}
              onClick={() => setUseManual(!useManual)}
            >
              {useManual ? 'Seleccionar de lista' : 'Ingresar IDs manuales'}
            </button>
          </div>

          {slotValues.map((value, index) => {
            const number = index + 1;
            const fieldId = useManual || submissions.length === 0
              ? `player-${number}-input`
              : `player-${number}-select`;
            return (
              <div className="modal-form-group" key={index}>
                <label htmlFor={fieldId}>{t('matches:wizard.playerLabel', { number })}</label>
                <div style={{ display: 'flex', gap: '8px' }}>
                  {!useManual && submissions.length > 0 ? (
                    <select
                      id={fieldId}
                      value={value}
                      onChange={(e) => updateSlot(index, e.target.value)}
                      required
                      style={{ flex: 1 }}
                    >
                      <option value="">{t('matches:wizard.selectSubmissionPlaceholder')}</option>
                      {submissions.map((s) => (
                        <option key={s.id} value={s.id}>
                          v{s.version} — {s.language} (#{s.id ? s.id.slice(0, 8) : '—'})
                        </option>
                      ))}
                    </select>
                  ) : (
                    <input
                      id={fieldId}
                      type="text"
                      placeholder={`ID de submission o bot ${number} (ej. sub-00${number})`}
                      value={value}
                      onChange={(e) => updateSlot(index, e.target.value)}
                      required
                      style={{ flex: 1 }}
                    />
                  )}
                  {slotCount > MIN_PLAYERS && (
                    <button
                      type="button"
                      className="btn btn-secondary btn-compact"
                      onClick={() => removeSlot(index)}
                      aria-label={t('matches:wizard.removePlayer', { number })}
                      title={t('matches:wizard.removePlayer', { number })}
                    >
                      <Trash2 size={14} aria-hidden="true" />
                    </button>
                  )}
                </div>
              </div>
            );
          })}

          <button
            type="button"
            className="btn btn-secondary btn-compact"
            onClick={addSlot}
            disabled={slotCount >= MAX_PLAYERS}
            style={{ display: 'inline-flex', alignItems: 'center', gap: '4px', alignSelf: 'flex-start' }}
          >
            <Plus size={14} aria-hidden="true" />
            <span>{t('matches:wizard.addPlayer', { max: MAX_PLAYERS })}</span>
          </button>

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
