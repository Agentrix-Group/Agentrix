import React, { useEffect, useState, useRef } from 'react';
import { useTranslation } from 'react-i18next';
import { ApiService } from '../service/apiService.js';
import { formatDateTime, formatNumber } from '../i18n/formatters.js';
import { Bot, ShieldCheck, UploadCloud, UserPlus, AlertCircle, ChevronDown, ChevronUp } from 'lucide-react';
import { BundleDropzone } from '../components/BundleDropzone.jsx';
import { StarterKitCard } from '../components/StarterKitCard.jsx';
import { AdmissionStatusBanner, sanitizeAdmissionError } from '../components/AdmissionStatusBanner.jsx';

export function AgentsPage({ currentUser }) {
  const { t, i18n } = useTranslation(['agents', 'common', 'errors']);
  const currentLang = i18n.language?.startsWith('en') ? 'en' : 'es';

  const [agents, setAgents] = useState([]);
  const [contests, setContests] = useState([]);
  const [selectedAgent, setSelectedAgent] = useState(null);
  const [bundle, setBundle] = useState(null);
  const [submissions, setSubmissions] = useState([]);
  const [newAgentName, setNewAgentName] = useState('');
  const [newAgentDesc, setNewAgentDesc] = useState('');
  const [selectedContestId, setSelectedContestId] = useState('');
  const [statusMsg, setStatusMsg] = useState('');
  const [errorMsg, setErrorMsg] = useState('');

  // FQ-2 Admission states
  const [validationError, setValidationError] = useState(null);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [admissionStage, setAdmissionStage] = useState('idle'); // 'idle' | 'uploading' | 'validating' | 'ready' | 'rejected'
  const [rejectionError, setRejectionError] = useState(null);
  const [expandedRejectionIds, setExpandedRejectionIds] = useState(new Set());
  const pollTimerRef = useRef(null);

  const loadAgents = () => {
    if (!currentUser) return;
    ApiService.listAgents(currentUser.id)
      .then((data) => {
        setAgents(data || []);
        if (data && data.length > 0 && !selectedAgent) {
          selectAgent(data[0]);
        }
      })
      .catch((err) => {
        setErrorMsg(err.message || 'No se pudieron cargar los agentes');
      });
  };

  const loadContests = () => {
    ApiService.listContests()
      .then((data) => {
        setContests(data || []);
        if (data && data.length > 0) {
          setSelectedContestId(data[0].id);
        }
      })
      .catch((err) => {
        setErrorMsg(err.message || 'No se pudieron cargar los concursos');
      });
  };

  const loadSubmissions = (agentId) => {
    if (!agentId) return;
    ApiService.listSubmissions(agentId)
      .then((subs) => setSubmissions(subs || []))
      .catch(() => setSubmissions([]));
  };

  useEffect(() => {
    loadAgents();
    loadContests();
    return () => {
      if (pollTimerRef.current) {
        clearTimeout(pollTimerRef.current);
      }
    };
  }, [currentUser]);

  const selectAgent = (agent) => {
    setSelectedAgent(agent);
    setStatusMsg('');
    setErrorMsg('');
    setValidationError(null);
    setAdmissionStage('idle');
    setRejectionError(null);
    setBundle(null);
    loadSubmissions(agent.id);
  };

  const handleCreateAgent = async (e) => {
    e.preventDefault();
    if (!newAgentName.trim()) return;
    setStatusMsg('');
    setErrorMsg('');

    try {
      await ApiService.createAgent({
        name: newAgentName.trim(),
        game_id: 'starfighter',
        description: newAgentDesc.trim(),
      });
      setNewAgentName('');
      setNewAgentDesc('');
      setStatusMsg(t('agents:messages.createdSuccess'));
      loadAgents();
    } catch (err) {
      setErrorMsg(err.message || t('agents:messages.createError'));
    }
  };

  const pollSubmissionAdmission = async (submissionId, agentId) => {
    setAdmissionStage('validating');
    const maxAttempts = 15;
    const intervalMs = 1500;

    for (let attempt = 0; attempt < maxAttempts; attempt++) {
      if (attempt > 0) {
        await new Promise((resolve) => {
          pollTimerRef.current = setTimeout(resolve, intervalMs);
        });
      }

      try {
        const sub = await ApiService.getSubmission(submissionId);
        if (!sub) continue;

        if (sub.status === 'ready') {
          setAdmissionStage('ready');
          setStatusMsg(t('agents:messages.submitSuccess', { version: sub.version || 'new' }));
          setBundle(null);
          setIsSubmitting(false);
          loadSubmissions(agentId);
          return;
        }

        if (sub.status === 'rejected' || sub.status === 'failed') {
          setAdmissionStage('rejected');
          setRejectionError(sub.error_detail || t('agents:messages.submitError'));
          setErrorMsg(t('agents:messages.submitError'));
          setIsSubmitting(false);
          loadSubmissions(agentId);
          return;
        }
      } catch {
        // Continue polling until maxAttempts
      }
    }

    // Timeout
    setAdmissionStage('idle');
    setErrorMsg(t('agents:messages.admissionTimeout'));
    setIsSubmitting(false);
    loadSubmissions(agentId);
  };

  const handleSubmitCode = async (e) => {
    e.preventDefault();
    if (!selectedAgent || !bundle || isSubmitting) return;

    setStatusMsg('');
    setErrorMsg('');
    setRejectionError(null);
    setIsSubmitting(true);
    setAdmissionStage('uploading');

    try {
      const res = await ApiService.uploadBotBundle(selectedAgent.id, bundle);
      const status = res.status || 'ready';

      if (status === 'validating' || status === 'pending' || status === 'pending_validation') {
        // Asynchronous admission pipeline: start polling
        await pollSubmissionAdmission(res.id, selectedAgent.id);
      } else if (status === 'ready') {
        setAdmissionStage('ready');
        setStatusMsg(t('agents:messages.submitSuccess', { version: res.version || 'new' }));
        setBundle(null);
        setIsSubmitting(false);
        loadSubmissions(selectedAgent.id);
      } else {
        // Rejected or failed
        setAdmissionStage('rejected');
        setRejectionError(res.error_detail || t('agents:messages.submitError'));
        setErrorMsg(t('agents:messages.submitError'));
        setIsSubmitting(false);
        loadSubmissions(selectedAgent.id);
      }
    } catch (err) {
      setAdmissionStage('rejected');
      setRejectionError(err.message || t('agents:messages.submitError'));
      setErrorMsg(t('agents:messages.submitError'));
      setIsSubmitting(false);
    }
  };

  const handleEnroll = async () => {
    if (!selectedAgent || !selectedContestId) return;
    setStatusMsg('');
    setErrorMsg('');

    try {
      await ApiService.enrollAgent(selectedContestId, selectedAgent.id);
      setStatusMsg(t('agents:messages.enrollSuccess', {
        name: selectedAgent.name,
        contest: selectedContestId,
      }));
    } catch (err) {
      setErrorMsg(err.message || t('agents:messages.enrollError'));
    }
  };

  const toggleRejectionRow = (subId) => {
    setExpandedRejectionIds((prev) => {
      const next = new Set(prev);
      if (next.has(subId)) {
        next.delete(subId);
      } else {
        next.add(subId);
      }
      return next;
    });
  };

  if (!currentUser) {
    return (
      <div className="card" style={{ textAlign: 'center', margin: '40px auto', maxWidth: '400px' }}>
        <h3>{t('agents:authRequired.title')}</h3>
        <p style={{ color: 'var(--text-secondary)' }}>{t('agents:authRequired.description')}</p>
      </div>
    );
  }

  const renderStatusBadge = (status) => {
    switch (status) {
      case 'ready':
        return <span className="badge badge-ready">{t('agents:submission.status.ready', { defaultValue: 'Admitido' })}</span>;
      case 'validating':
        return <span className="badge badge-validating">{t('agents:submission.status.validating', { defaultValue: 'Validando...' })}</span>;
      case 'pending':
        return <span className="badge badge-pending">{t('agents:submission.status.pending', { defaultValue: 'Pendiente' })}</span>;
      case 'pending_validation':
        return <span className="badge badge-pending">{t('agents:submission.status.pending_validation', { defaultValue: 'Pendiente de validación' })}</span>;
      case 'rejected':
      case 'failed':
        return <span className="badge badge-rejected">{t('agents:submission.status.rejected', { defaultValue: 'Rechazado' })}</span>;
      default:
        return <span className="badge badge-pending">{status}</span>;
    }
  };

  return (
    <div>
      <h1>{t('agents:title')}</h1>
      <p style={{ color: 'var(--text-secondary)' }}>
        {t('agents:subtitle')}
      </p>

      {statusMsg && (
        <div role="status" style={{ padding: '12px', background: 'var(--success-bg)', color: 'var(--success-text)', borderRadius: '6px', margin: '16px 0' }}>
          {statusMsg}
        </div>
      )}
      {errorMsg && !rejectionError && (
        <div role="alert" style={{ padding: '12px', background: 'var(--danger-bg)', color: 'var(--danger-text)', borderRadius: '6px', margin: '16px 0' }}>
          {errorMsg}
        </div>
      )}

      <div className="agents-layout">
        {/* Left Column: Create Bot & Bot List */}
        <div>
          <div className="card" style={{ marginBottom: '24px' }}>
            <h3>{t('agents:register.title')}</h3>
            <form onSubmit={handleCreateAgent} style={{ display: 'flex', flexDirection: 'column', gap: '12px' }}>
              <input
                type="text"
                placeholder={t('agents:register.namePlaceholder')}
                value={newAgentName}
                onChange={(e) => setNewAgentName(e.target.value)}
                style={{ padding: '8px 12px' }}
                required
              />
              <textarea
                placeholder={t('agents:register.descriptionPlaceholder')}
                value={newAgentDesc}
                onChange={(e) => setNewAgentDesc(e.target.value)}
                rows={2}
                style={{ padding: '8px 12px', resize: 'vertical' }}
              />
              <button type="submit" className="btn"><Bot size={17} /> {t('agents:register.submit')}</button>
            </form>
          </div>

          <h3>{t('agents:list.titleWithCount', { count: formatNumber(agents.length, currentLang) })}</h3>
          <div style={{ display: 'flex', flexDirection: 'column', gap: '10px' }}>
            {agents.map((ag) => (
              <div
                key={ag.id}
                className="card"
                onClick={() => selectAgent(ag)}
                style={{
                  cursor: 'pointer',
                  borderColor: selectedAgent?.id === ag.id ? 'var(--accent)' : 'var(--border)',
                  padding: '14px',
                }}
              >
                <div style={{ display: 'flex', justifyContent: 'space-between' }}>
                  <strong>{ag.name}</strong>
                  <span className="badge badge-finished">{ag.game_id}</span>
                </div>
                <div style={{ fontSize: '0.8rem', color: 'var(--text-secondary)', marginTop: '4px' }}>
                  {t('agents:list.idLabel', { id: ag.id.substring(0, 8) })}
                </div>
              </div>
            ))}
            {agents.length === 0 && (
              <p style={{ color: 'var(--text-secondary)' }}>{t('agents:list.empty')}</p>
            )}
          </div>
        </div>

        {/* Right Column: Code Submissions & Contest Enrollment */}
        <div>
          {selectedAgent ? (
            <div>
              <div className="card" style={{ marginBottom: '24px' }}>
                <div className="agent-detail-header">
                  <h2>{selectedAgent.name}</h2>
                  <div style={{ display: 'flex', gap: '8px', alignItems: 'center' }}>
                    <select
                      value={selectedContestId}
                      onChange={(e) => setSelectedContestId(e.target.value)}
                      style={{ padding: '6px 10px' }}
                    >
                      {contests.map((c) => (
                        <option key={c.id} value={c.id}>{c.name}</option>
                      ))}
                    </select>
                    <button className="btn" onClick={handleEnroll}><UserPlus size={17} /> {t('agents:details.enroll')}</button>
                  </div>
                </div>
                <p style={{ color: 'var(--text-secondary)' }}>{selectedAgent.description || t('agents:details.noDescription')}</p>
              </div>

              {/* Starter Template Box */}
              <StarterKitCard />

              {/* Upload Dropzone Card */}
              <div className="card" style={{ marginBottom: '24px' }}>
                <h3 style={{ display: 'flex', gap: '8px', alignItems: 'center' }}>
                  <UploadCloud size={20} /> {t('agents:submission.uploadTitle')}
                </h3>

                <form onSubmit={handleSubmitCode} style={{ display: 'flex', flexDirection: 'column', gap: '12px' }}>
                  <BundleDropzone
                    bundle={bundle}
                    onFileSelect={setBundle}
                    disabled={isSubmitting}
                    validationError={validationError}
                    setValidationError={setValidationError}
                  />

                  <div className="admission-note">
                    <ShieldCheck size={18} />
                    <span>{t('agents:submission.admissionCheck')}</span>
                  </div>

                  {/* Async admission stage & rejection feedback */}
                  <AdmissionStatusBanner
                    stage={admissionStage}
                    errorDetail={rejectionError}
                    onDismiss={() => {
                      setAdmissionStage('idle');
                      setRejectionError(null);
                    }}
                  />

                  <button
                    type="submit"
                    className="btn"
                    disabled={!bundle || isSubmitting || !!validationError}
                    style={{ display: 'inline-flex', alignItems: 'center', justifyContent: 'center', gap: '8px' }}
                  >
                    <UploadCloud size={17} />
                    {isSubmitting
                      ? (admissionStage === 'validating'
                          ? t('agents:submission.validationStages.validating')
                          : t('agents:submission.validationStages.uploading'))
                      : t('agents:submission.publish')}
                  </button>
                </form>
              </div>

              {/* History Card */}
              <div className="card">
                <h3>{t('agents:submission.history')}</h3>
                {submissions.length > 0 ? (
                  <table className="table">
                    <thead>
                      <tr>
                        <th>{t('agents:submission.table.version')}</th>
                        <th>{t('agents:submission.table.language')}</th>
                        <th>{t('agents:submission.table.status')}</th>
                        <th>{t('agents:submission.table.active')}</th>
                        <th>{t('agents:submission.table.id')}</th>
                        <th>{t('agents:submission.table.created')}</th>
                        <th>{t('agents:submission.table.actions')}</th>
                      </tr>
                    </thead>
                    <tbody>
                      {submissions.map((s, index) => {
                        const isLatestReady = s.status === 'ready' && index === submissions.findIndex((item) => item.status === 'ready');
                        const isExpanded = expandedRejectionIds.has(s.id);
                        const isRejected = s.status === 'rejected' || s.status === 'failed';

                        return (
                          <React.Fragment key={s.id}>
                            <tr>
                              <td><strong>v{s.version}</strong></td>
                              <td>{s.language}</td>
                              <td>{renderStatusBadge(s.status)}</td>
                              <td>
                                {isLatestReady && s.active ? (
                                  <span className="badge badge-active">{t('agents:submission.table.active')}</span>
                                ) : (
                                  <span style={{ color: 'var(--text-secondary)', fontSize: '0.8rem' }}>—</span>
                                )}
                              </td>
                              <td style={{ fontSize: '0.8rem', color: 'var(--text-secondary)' }}>#{s.id ? s.id.slice(0, 8) : '—'}</td>
                              <td style={{ fontSize: '0.8rem' }}>{formatDateTime(s.created_at, currentLang)}</td>
                              <td>
                                {isRejected && (
                                  <button
                                    type="button"
                                    className="btn btn-compact"
                                    style={{ padding: '2px 8px', fontSize: '0.78rem' }}
                                    onClick={() => toggleRejectionRow(s.id)}
                                  >
                                    {isExpanded
                                      ? t('agents:submission.table.hideReason')
                                      : t('agents:submission.table.viewReason')}
                                    {isExpanded ? <ChevronUp size={12} /> : <ChevronDown size={12} />}
                                  </button>
                                )}
                              </td>
                            </tr>
                            {isRejected && isExpanded && (
                              <tr>
                                <td colSpan="7" style={{ background: '#fff1f2', padding: '10px 14px' }}>
                                  <div style={{ color: '#9f1239', fontSize: '0.82rem' }}>
                                    <strong>{t('agents:submission.rejection.title')}: </strong>
                                    <span>{s.error_detail ? sanitizeAdmissionError(s.error_detail) : t('agents:submission.rejection.rule2')}</span>
                                  </div>
                                </td>
                              </tr>
                            )}
                          </React.Fragment>
                        );
                      })}
                    </tbody>
                  </table>
                ) : (
                  <p style={{ color: 'var(--text-secondary)' }}>{t('agents:submission.emptyHistory')}</p>
                )}
              </div>
            </div>
          ) : (
            <div className="card">{t('agents:details.selectPrompt')}</div>
          )}
        </div>
      </div>
    </div>
  );
}
