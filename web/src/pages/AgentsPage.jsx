import React, { useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { ApiService } from '../service/apiService.js';
import { formatDateTime, formatNumber } from '../i18n/formatters.js';
import { Bot, FileArchive, ShieldCheck, UploadCloud, UserPlus } from 'lucide-react';

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

  const loadAgents = () => {
    if (!currentUser) return;
    ApiService.listAgents(currentUser.id)
      .then((data) => {
        setAgents(data || []);
        if (data && data.length > 0 && !selectedAgent) {
          selectAgent(data[0]);
        }
      })
      .catch(() => {});
  };

  const loadContests = () => {
    ApiService.listContests()
      .then((data) => {
        setContests(data || []);
        if (data && data.length > 0) {
          setSelectedContestId(data[0].id);
        }
      })
      .catch(() => {});
  };

  useEffect(() => {
    loadAgents();
    loadContests();
  }, [currentUser]);

  const selectAgent = (agent) => {
    setSelectedAgent(agent);
    setStatusMsg('');
    setErrorMsg('');
    ApiService.listSubmissions(agent.id)
      .then((subs) => setSubmissions(subs || []))
      .catch(() => setSubmissions([]));
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

  const handleSubmitCode = async (e) => {
    e.preventDefault();
    if (!selectedAgent || !bundle) return;
    setStatusMsg('');
    setErrorMsg('');

    try {
      const res = await ApiService.uploadBotBundle(selectedAgent.id, bundle);
      setStatusMsg(t('agents:messages.submitSuccess', { version: res.version || 'new' }));
      setBundle(null);
      ApiService.listSubmissions(selectedAgent.id).then((subs) => setSubmissions(subs || []));
    } catch (err) {
      setErrorMsg(err.message || t('agents:messages.submitError'));
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

  if (!currentUser) {
    return (
      <div className="card" style={{ textAlign: 'center', margin: '40px auto', maxWidth: '400px' }}>
        <h3>{t('agents:authRequired.title')}</h3>
        <p style={{ color: 'var(--text-secondary)' }}>{t('agents:authRequired.description')}</p>
      </div>
    );
  }

  const getStatusText = (status) => {
    return t(`common:status.${status}`, { defaultValue: status });
  };

  return (
    <div>
      <h1>{t('agents:title')}</h1>
      <p style={{ color: 'var(--text-secondary)' }}>
        {t('agents:subtitle')}
      </p>

      {statusMsg && (
        <div style={{ padding: '12px', background: 'var(--success-bg)', color: 'var(--success-text)', borderRadius: '6px', margin: '16px 0' }}>
          {statusMsg}
        </div>
      )}
      {errorMsg && (
        <div style={{ padding: '12px', background: 'var(--danger-bg)', color: 'var(--danger-text)', borderRadius: '6px', margin: '16px 0' }}>
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

              <div className="card" style={{ marginBottom: '24px' }}>
                <h3 style={{ display: 'flex', gap: '8px', alignItems: 'center' }}>
                  <UploadCloud size={20} /> {t('agents:submission.uploadTitle')}
                </h3>
                <form onSubmit={handleSubmitCode} style={{ display: 'flex', flexDirection: 'column', gap: '12px' }}>
                  <label className="bundle-dropzone">
                    <FileArchive size={30} aria-hidden="true" />
                    <strong>{bundle ? bundle.name : t('agents:submission.chooseZip')}</strong>
                    <span>{t('agents:submission.zipHelp')}</span>
                    <input
                      type="file"
                      accept=".zip,application/zip"
                      onChange={(event) => setBundle(event.target.files?.[0] || null)}
                      required
                    />
                  </label>
                  <div className="admission-note">
                    <ShieldCheck size={18} />
                    <span>{t('agents:submission.admissionCheck')}</span>
                  </div>
                  <button type="submit" className="btn" disabled={!bundle}>
                    <UploadCloud size={17} /> {t('agents:submission.publish')}
                  </button>
                </form>
              </div>

              <div className="card">
                <h3>{t('agents:submission.history')}</h3>
                {submissions.length > 0 ? (
                  <table className="table">
                    <thead>
                      <tr>
                        <th>{t('agents:submission.table.version')}</th>
                        <th>{t('agents:submission.table.language')}</th>
                        <th>{t('agents:submission.table.status')}</th>
                        <th>{t('agents:submission.table.path')}</th>
                        <th>{t('agents:submission.table.created')}</th>
                      </tr>
                    </thead>
                    <tbody>
                      {submissions.map((s) => (
                        <tr key={s.id}>
                          <td><strong>v{s.version}</strong></td>
                          <td>{s.language}</td>
                          <td><span className="badge badge-finished">{getStatusText(s.status)}</span></td>
                          <td style={{ fontSize: '0.8rem', color: 'var(--text-secondary)' }}>{s.code_path}</td>
                          <td style={{ fontSize: '0.8rem' }}>{formatDateTime(s.created_at, currentLang)}</td>
                        </tr>
                      ))}
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
