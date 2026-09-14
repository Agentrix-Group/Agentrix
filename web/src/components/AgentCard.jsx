import React from 'react';

export function AgentCard({ agent }) {
  return (
    <div className="card">
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        <strong>{agent.name}</strong>
        <span className="badge badge-finished">Active</span>
      </div>
      <p style={{ color: 'var(--text-secondary)', fontSize: '0.9rem', margin: '8px 0' }}>
        {agent.description || 'No description provided'}
      </p>
      <div style={{ fontSize: '0.8rem', color: 'var(--text-secondary)' }}>
        Game: {agent.game_id} | Author: {agent.participant_id}
      </div>
    </div>
  );
}
