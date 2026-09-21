import React from 'react';
import { Inbox } from 'lucide-react';

export function EmptyState({
  title,
  description,
  actionLabel,
  onAction,
  icon: Icon = Inbox,
  className = '',
}) {
  return (
    <div
      className={`card empty-state ${className}`}
      style={{
        textAlign: 'center',
        padding: '48px 24px',
        margin: '24px 0',
        background: 'var(--card-bg, #1e2430)',
        border: '1px dashed var(--border-color, #334155)',
        borderRadius: '12px',
      }}
    >
      <div
        style={{
          display: 'inline-flex',
          padding: '16px',
          borderRadius: '50%',
          background: 'rgba(59, 130, 246, 0.1)',
          color: 'var(--accent, #3b82f6)',
          marginBottom: '16px',
        }}
      >
        <Icon size={32} aria-hidden="true" />
      </div>
      {title && (
        <h3 style={{ margin: '0 0 8px 0', fontSize: '1.2rem', color: 'var(--text-primary, #f8fafc)' }}>
          {title}
        </h3>
      )}
      {description && (
        <p
          style={{
            margin: '0 auto 20px auto',
            maxWidth: '480px',
            color: 'var(--text-secondary, #94a3b8)',
            fontSize: '0.95rem',
            lineHeight: 1.5,
          }}
        >
          {description}
        </p>
      )}
      {actionLabel && onAction && (
        <button
          type="button"
          className="btn"
          onClick={onAction}
          style={{ display: 'inline-flex', alignItems: 'center', gap: '8px' }}
        >
          {actionLabel}
        </button>
      )}
    </div>
  );
}

export default EmptyState;
