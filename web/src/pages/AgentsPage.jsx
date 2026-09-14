import React, { useEffect, useState } from 'react';
import { ApiService } from '../service/apiService.js';

export function AgentsPage({ currentUser }) {
  const [agents, setAgents] = useState([]);
  const [contests, setContests] = useState([]);
  const [selectedAgent, setSelectedAgent] = useState(null);
  const [code, setCode] = useState('');
  const [language, setLanguage] = useState('python');
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
        game_id: 'arena-basica',
        description: newAgentDesc.trim(),
      });
      setNewAgentName('');
      setNewAgentDesc('');
      setStatusMsg('Bot created successfully!');
      loadAgents();
    } catch (err) {
      setErrorMsg(err.message || 'Failed to create bot');
    }
  };

  const handleSubmitCode = async (e) => {
    e.preventDefault();
    if (!selectedAgent || !code.trim()) return;
    setStatusMsg('');
    setErrorMsg('');

    try {
      const res = await ApiService.submitCode(selectedAgent.id, code, language);
      setStatusMsg(`Code submission v${res.version || 'new'} created successfully!`);
      setCode('');
      ApiService.listSubmissions(selectedAgent.id).then((subs) => setSubmissions(subs || []));
    } catch (err) {
      setErrorMsg(err.message || 'Failed to submit code');
    }
  };

  const handleEnroll = async () => {
    if (!selectedAgent || !selectedContestId) return;
    setStatusMsg('');
    setErrorMsg('');

    try {
      await ApiService.enrollAgent(selectedContestId, selectedAgent.id);
      setStatusMsg(`Enrolled '${selectedAgent.name}' in contest '${selectedContestId}'!`);
    } catch (err) {
      setErrorMsg(err.message || 'Failed to enroll agent');
    }
  };

  if (!currentUser) {
    return (
      <div className="card" style={{ textAlign: 'center', margin: '40px auto', maxWidth: '400px' }}>
        <h3>Authentication Required</h3>
        <p style={{ color: 'var(--text-secondary)' }}>Please log in to manage your agents and code submissions.</p>
      </div>
    );
  }

  return (
    <div>
      <h1>My Autonomous Agents</h1>
      <p style={{ color: 'var(--text-secondary)' }}>
        Develop, upload, and enroll competitive bots in arena tournaments.
      </p>

      {statusMsg && (
        <div style={{ padding: '12px', background: '#22c55e20', color: '#4ade80', borderRadius: '6px', margin: '16px 0' }}>
          {statusMsg}
        </div>
      )}
      {errorMsg && (
        <div style={{ padding: '12px', background: '#ef444420', color: '#f87171', borderRadius: '6px', margin: '16px 0' }}>
          {errorMsg}
        </div>
      )}

      <div style={{ display: 'grid', gridTemplateColumns: '1fr 2fr', gap: '24px', marginTop: '24px' }}>
        {/* Left Column: Create Bot & Bot List */}
        <div>
          <div className="card" style={{ marginBottom: '24px' }}>
            <h3>Register New Bot</h3>
            <form onSubmit={handleCreateAgent} style={{ display: 'flex', flexDirection: 'column', gap: '12px' }}>
              <input
                type="text"
                placeholder="Bot Name (e.g. HunterAlpha)"
                value={newAgentName}
                onChange={(e) => setNewAgentName(e.target.value)}
                style={{ padding: '8px 12px', background: 'var(--bg-primary)', border: '1px solid var(--border)', color: '#fff', borderRadius: '4px' }}
                required
              />
              <textarea
                placeholder="Description / Strategy notes"
                value={newAgentDesc}
                onChange={(e) => setNewAgentDesc(e.target.value)}
                rows={2}
                style={{ padding: '8px 12px', background: 'var(--bg-primary)', border: '1px solid var(--border)', color: '#fff', borderRadius: '4px', resize: 'vertical' }}
              />
              <button type="submit" className="btn">Create Bot</button>
            </form>
          </div>

          <h3>Your Bots ({agents.length})</h3>
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
                  ID: {ag.id.substring(0, 8)}...
                </div>
              </div>
            ))}
            {agents.length === 0 && (
              <p style={{ color: 'var(--text-secondary)' }}>No bots registered yet. Create one above!</p>
            )}
          </div>
        </div>

        {/* Right Column: Code Submissions & Contest Enrollment */}
        <div>
          {selectedAgent ? (
            <div>
              <div className="card" style={{ marginBottom: '24px' }}>
                <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                  <h2>{selectedAgent.name}</h2>
                  <div style={{ display: 'flex', gap: '8px', alignItems: 'center' }}>
                    <select
                      value={selectedContestId}
                      onChange={(e) => setSelectedContestId(e.target.value)}
                      style={{ padding: '6px 10px', background: 'var(--bg-primary)', color: '#fff', border: '1px solid var(--border)', borderRadius: '4px' }}
                    >
                      {contests.map((c) => (
                        <option key={c.id} value={c.id}>{c.name}</option>
                      ))}
                    </select>
                    <button className="btn" onClick={handleEnroll}>Enroll in Tournament</button>
                  </div>
                </div>
                <p style={{ color: 'var(--text-secondary)' }}>{selectedAgent.description || 'No description'}</p>
              </div>

              <div className="card" style={{ marginBottom: '24px' }}>
                <h3>Upload Code Submission</h3>
                <form onSubmit={handleSubmitCode} style={{ display: 'flex', flexDirection: 'column', gap: '12px' }}>
                  <div style={{ display: 'flex', gap: '12px', alignItems: 'center' }}>
                    <label style={{ fontSize: '0.9rem', color: 'var(--text-secondary)' }}>Language:</label>
                    <select
                      value={language}
                      onChange={(e) => setLanguage(e.target.value)}
                      style={{ padding: '6px 10px', background: 'var(--bg-primary)', color: '#fff', border: '1px solid var(--border)', borderRadius: '4px' }}
                    >
                      <option value="python">Python 3</option>
                      <option value="javascript">JavaScript / Node</option>
                      <option value="go">Go</option>
                    </select>
                  </div>
                  <textarea
                    rows={10}
                    placeholder={`#!/usr/bin/env python3\n# Your agent logic here...\nimport sys, json\nprint(json.dumps({'type': 'ATTACK'}))`}
                    value={code}
                    onChange={(e) => setCode(e.target.value)}
                    style={{
                      fontFamily: 'monospace',
                      fontSize: '0.85rem',
                      padding: '12px',
                      background: 'var(--bg-primary)',
                      border: '1px solid var(--border)',
                      color: '#fff',
                      borderRadius: '4px',
                    }}
                    required
                  />
                  <button type="submit" className="btn">Publish Code Submission</button>
                </form>
              </div>

              <div className="card">
                <h3>Version History</h3>
                {submissions.length > 0 ? (
                  <table className="table">
                    <thead>
                      <tr>
                        <th>Version</th>
                        <th>Language</th>
                        <th>Status</th>
                        <th>Path</th>
                        <th>Created</th>
                      </tr>
                    </thead>
                    <tbody>
                      {submissions.map((s) => (
                        <tr key={s.id}>
                          <td><strong>v{s.version}</strong></td>
                          <td>{s.language}</td>
                          <td><span className="badge badge-finished">{s.status}</span></td>
                          <td style={{ fontSize: '0.8rem', color: 'var(--text-secondary)' }}>{s.code_path}</td>
                          <td style={{ fontSize: '0.8rem' }}>{new Date(s.created_at).toLocaleString()}</td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                ) : (
                  <p style={{ color: 'var(--text-secondary)' }}>No code versions uploaded yet.</p>
                )}
              </div>
            </div>
          ) : (
            <div className="card">Select or create a bot to view details.</div>
          )}
        </div>
      </div>
    </div>
  );
}
