import React, { createContext, useCallback, useContext, useMemo, useState } from 'react';
import { CheckCircle2, AlertTriangle, Info, X } from 'lucide-react';
import { useTranslation } from 'react-i18next';

const ToastContext = createContext(null);
const ICONS = { success: CheckCircle2, error: AlertTriangle, info: Info };

/** Accessible, non-blocking feedback that replaces alert(). */
export function ToastProvider({ children }) {
  const { t } = useTranslation('common');
  const [toasts, setToasts] = useState([]);
  const dismiss = useCallback((id) => setToasts((list) => list.filter((toast) => toast.id !== id)), []);
  const push = useCallback((kind, message, { timeout = 6000 } = {}) => {
    const id = `${Date.now()}-${Math.random()}`;
    setToasts((list) => [...list.slice(-3), { id, kind, message }]);
    if (timeout) setTimeout(() => dismiss(id), timeout);
  }, [dismiss]);
  const api = useMemo(() => ({
    success: (m, o) => push('success', m, o),
    error: (m, o) => push('error', m, { timeout: 10000, ...o }),
    info: (m, o) => push('info', m, o),
  }), [push]);
  return (
    <ToastContext.Provider value={api}>
      {children}
      <div className="toast-region" role="region" aria-label={t('labels.notifications')}>
        {toasts.map((toast) => {
          const Icon = ICONS[toast.kind];
          return (
            <div key={toast.id} className={`toast toast-${toast.kind}`} role={toast.kind === 'error' ? 'alert' : 'status'}>
              <Icon size={18} aria-hidden="true" />
              <span>{toast.message}</span>
              <button type="button" className="icon-button" onClick={() => dismiss(toast.id)} aria-label={t('buttons.dismiss')}>
                <X size={16} aria-hidden="true" />
              </button>
            </div>
          );
        })}
      </div>
    </ToastContext.Provider>
  );
}

export function useToast() {
  const context = useContext(ToastContext);
  if (!context) throw new Error('useToast must be used within <ToastProvider>');
  return context;
}
