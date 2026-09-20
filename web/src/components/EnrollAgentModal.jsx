import React, { useState, useEffect } from 'react';
import { useTranslation } from 'react-i18next';
import { X, CheckCircle2, AlertCircle, Loader2, Trophy, ShieldAlert } from 'lucide-react';
import { ApiService } from '../service/apiService.js';

export function EnrollAgentModal({ isOpen, contest, onClose, onEnrolled }) {
  const { t } = useTranslation(['home', 'common']);
  const [agents, setAgents] = useState([]);
  const [selectedAgentId, setSelectedAgentId] = useState('');
  const [agentSubmissions, setAgentSubmissions] = useState([]);
  const [isLoadingAgents, setIsLoadingAgents] = useState(false);
  const [isLoadingSubmissions, setIsLoadingSubmissions] = useState(false);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [errorMsg, setErrorMsg] = useState('');

  useEffect(() => {
    if (!isOpen || !contest) return;

    setErrorMsg('');
    setIsSubmitting(false);
    setIsLoadingAgents(true);

    ApiService.listAgents()
      .then((data) => {
        const list = data || [];
        setAgents(list);
        if (list.length > 0) {
          setSelectedAgentId(list[0].id);
        }
      })
      .catch((err) => {
        setErrorMsg(err.message || t('common:messages.operationFailed'));
      })
      .finally(() => {
        setIsLoadingAgents(false);
      });

    const handleKeyDown = (e) => {
      if (e.key === 'Escape') {
        onClose();
      }
    };
    window.addEventListener('keydown', handleKeyDown);
    return () => window.removeEventListener('keydown', handleKeyDown);
  }, [isOpen, contest, onClose, t]);

  // Load submissions whenever selectedAgentId changes
  useEffect(() => {
    if (!selectedAgentId) {
      setAgentSubmissions([]);
      return;
    }

    setIsLoadingSubmissions(true);
    ApiService.listSubmissions(selectedAgentId)
      .then((data) => {
        setAgentSubmissions(data || []);
      })
      .catch(() => {
        setAgentSubmissions([]);
      })
      .finally(() => {
        setIsLoadingSubmissions(false);
      });
  }, [selectedAgentId]);

  if (!isOpen || !contest) return null;

  const hasReadySubmission = agentSubmissions.some((s) => s.status === 'ready');
  const readySubmission = agentSubmissions.find((s) => s.status === 'ready');

  const handleSubmit = async (e) => {
    e.preventDefault();
    if (!selectedAgentId) return;

    if (!hasReadySubmission) {
      setErrorMsg(t('home:enrollModal.noReadySubmissions'));
      return;
    }

    setIsSubmitting(true);
    setErrorMsg('');

    try {
      const res = await ApiService.enrollAgent(contest.id, selectedAgentId);
      if (typeof onEnrolled === 'function') {
        onEnrolled(res);
      }
      onClose();
    } catch (err) {
      const msg = err.message || '';
      if (err.status === 409 || msg.includes('already enrolled') || msg.includes('already_exists')) {
        setErrorMsg(t('home:enrollModal.alreadyEnrolled'));
      } else if (err.status === 400 || msg.includes('registration is closed') || msg.includes('closed')) {
        setErrorMsg(t('home:enrollModal.closed'));
      } else {
        setErrorMsg(msg || t('common:messages.operationFailed'));
      }
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
        aria-labelledby="enroll-modal-title"
      >
        <div className="modal-header">
          <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
            <Trophy size={20} color="var(--accent)" aria-hidden="true" />
            <h3 id="enroll-modal-title">{t('home:enrollModal.title')}</h3>
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

        <div style={{ margin: '8px 0 16px' }}>
          <strong style={{ fontSize: '1.05rem' }}>{contest.name}</strong>
          <p style={{ color: 'var(--text-secondary)', fontSize: '0.85rem', margin: '4px 0 0' }}>
            {contest.description || t('home:noDescription')}
          </p>
        </div>

        {errorMsg && (
          <div
            role="alert"
            style={{
              padding: '10px 14px',
              background: 'var(--danger-bg)',
              color: 'var(--danger-text)',
              borderRadius: '6px',
              marginBottom: '16px',
              display: 'flex',
              alignItems: 'center',
              gap: '8px',
              fontSize: '0.88rem',
            }}
          >
            <AlertCircle size={18} aria-hidden="true" />
            <span>{errorMsg}</span>
          </div>
        )}

        <form onSubmit={handleSubmit} className="modal-form">
          <div className="modal-form-group">
            <label htmlFor="enroll-agent-select">{t('home:enrollModal.selectAgent')}</label>
            {isLoadingAgents ? (
              <p style={{ fontSize: '0.88rem', color: 'var(--text-secondary)' }}>{t('common:buttons.loading')}</p>
            ) : agents.length > 0 ? (
              <select
                id="enroll-agent-select"
                value={selectedAgentId}
                onChange={(e) => setSelectedAgentId(e.target.value)}
                required
              >
                {agents.map((ag) => (
                  <option key={ag.id} value={ag.id}>
                    {ag.name} ({ag.game_id || 'starfighter'})
                  </option>
                ))}
              </select>
            ) : (
              <div
                style={{
                  padding: '12px',
                  background: 'var(--bg-surface-elevated)',
                  borderRadius: '6px',
                  fontSize: '0.88rem',
                  color: 'var(--text-secondary)',
                }}
              >
                {t('home:enrollModal.noAgents')}
              </div>
            )}
          </div>

          {selectedAgentId && (
            <div style={{ marginTop: '8px' }}>
              {isLoadingSubmissions ? (
                <p style={{ fontSize: '0.85rem', color: 'var(--text-secondary)' }}>{t('common:buttons.loading')}</p>
              ) : hasReadySubmission ? (
                <div
                  style={{
                    padding: '8px 12px',
                    background: 'var(--success-bg)',
                    color: 'var(--success-text)',
                    borderRadius: '6px',
                    fontSize: '0.84rem',
                    display: 'flex',
                    alignItems: 'center',
                    gap: '6px',
                  }}
                >
                  <CheckCircle2 size={16} color="var(--success)" aria-hidden="true" />
                  <span>
                    Versión validada disponible: v{readySubmission.version} ({readySubmission.language})
                  </span>
                </div>
              ) : (
                <div
                  style={{
                    padding: '10px 12px',
                    background: 'rgba(234, 179, 8, 0.1)',
                    borderLeft: '3px solid #eab308',
                    borderRadius: '4px',
                    fontSize: '0.84rem',
                    display: 'flex',
                    alignItems: 'center',
                    gap: '8px',
                  }}
                >
                  <ShieldAlert size={18} color="#eab308" aria-hidden="true" />
                  <span>{t('home:enrollModal.noReadySubmissions')}</span>
                </div>
              )}
            </div>
          )}

          <p style={{ color: 'var(--text-secondary)', fontSize: '0.8rem', marginTop: '12px' }}>
            {t('home:enrollModal.readyNotice')}
          </p>

          <div className="modal-footer">
            <button
              type="button"
              className="btn btn-secondary"
              onClick={onClose}
              disabled={isSubmitting}
            >
              {t('home:enrollModal.cancel')}
            </button>
            <button
              type="submit"
              className="btn"
              disabled={isSubmitting || !hasReadySubmission || agents.length === 0}
              style={{ display: 'inline-flex', alignItems: 'center', gap: '6px' }}
            >
              {isSubmitting ? (
                <>
                  <Loader2 size={16} className="spin" style={{ animation: 'spin 0.8s linear infinite' }} aria-hidden="true" />
                  <span>{t('home:enrollModal.submitting')}</span>
                </>
              ) : (
                <span>{t('home:enrollModal.submit')}</span>
              )}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}
