import React from 'react';
import { AlertTriangle, Copy, Check, RefreshCw } from 'lucide-react';

export class ErrorBoundary extends React.Component {
  constructor(props) {
    super(props);
    this.state = {
      hasError: false,
      error: null,
      incidentId: null,
      copied: false,
    };
  }

  static getDerivedStateFromError(error) {
    const incidentId = `INC-${Date.now().toString(36).toUpperCase()}-${Math.random().toString(36).substring(2, 6).toUpperCase()}`;
    return {
      hasError: true,
      error,
      incidentId,
      copied: false,
    };
  }

  componentDidCatch(error, errorInfo) {
    // Sanitized logging for operational telemetry
    console.error(`[Agentrix ErrorBoundary] Incident ID: ${this.state.incidentId}`, {
      message: error?.message,
      componentStack: errorInfo?.componentStack,
    });
  }

  handleCopyIncidentId = () => {
    if (this.state.incidentId && navigator.clipboard) {
      navigator.clipboard.writeText(this.state.incidentId).then(() => {
        this.setState({ copied: true });
        setTimeout(() => this.setState({ copied: false }), 2500);
      }).catch(() => {});
    }
  };

  handleReset = () => {
    this.setState({ hasError: false, error: null, incidentId: null });
    if (this.props.onReset) {
      this.props.onReset();
    }
  };

  render() {
    if (this.state.hasError) {
      if (this.props.fallback) {
        return typeof this.props.fallback === 'function'
          ? this.props.fallback({
              error: this.state.error,
              incidentId: this.state.incidentId,
              reset: this.handleReset,
            })
          : this.props.fallback;
      }

      return (
        <div
          role="alert"
          aria-live="assertive"
          style={{
            maxWidth: '560px',
            margin: '48px auto',
            padding: '28px',
            background: 'var(--card-bg, #1e2430)',
            border: '1px solid var(--danger-border, rgba(239, 68, 68, 0.4))',
            borderRadius: '12px',
            boxShadow: '0 8px 24px rgba(0,0,0,0.3)',
            color: 'var(--text-primary, #f8fafc)',
            textAlign: 'center',
          }}
        >
          <div style={{ display: 'inline-flex', padding: '12px', background: 'rgba(239, 68, 68, 0.15)', borderRadius: '50%', marginBottom: '16px' }}>
            <AlertTriangle size={36} color="var(--danger, #ef4444)" />
          </div>

          <h2 style={{ fontSize: '1.4rem', marginBottom: '8px' }}>
            Ha ocurrido un error inesperado
          </h2>
          <p style={{ color: 'var(--text-secondary, #94a3b8)', fontSize: '0.95rem', marginBottom: '20px' }}>
            La aplicación encontró un problema de visualización. Los datos de la simulación y su cuenta permanecen seguros.
          </p>

          <div
            style={{
              display: 'inline-flex',
              alignItems: 'center',
              gap: '10px',
              padding: '8px 16px',
              background: 'rgba(0, 0, 0, 0.3)',
              borderRadius: '6px',
              border: '1px solid var(--border-color, #334155)',
              fontSize: '0.85rem',
              fontFamily: 'monospace',
              marginBottom: '24px',
            }}
          >
            <span>ID de incidente: <strong>{this.state.incidentId}</strong></span>
            <button
              type="button"
              onClick={this.handleCopyIncidentId}
              title="Copiar ID de incidente"
              style={{
                background: 'transparent',
                border: 'none',
                cursor: 'pointer',
                color: this.state.copied ? 'var(--success, #22c55e)' : 'var(--text-secondary, #94a3b8)',
                display: 'inline-flex',
                alignItems: 'center',
              }}
            >
              {this.state.copied ? <Check size={16} /> : <Copy size={16} />}
            </button>
          </div>

          <div style={{ display: 'flex', gap: '12px', justifyContent: 'center' }}>
            <button
              type="button"
              className="btn"
              onClick={this.handleReset}
              style={{ display: 'inline-flex', alignItems: 'center', gap: '6px' }}
            >
              <RefreshCw size={16} /> Reintentar
            </button>
            <button
              type="button"
              className="btn btn-secondary"
              onClick={() => { window.location.href = '/'; }}
            >
              Ir al inicio
            </button>
          </div>
        </div>
      );
    }

    return this.props.children;
  }
}
