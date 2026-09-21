import React, { useRef, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Modal } from './Modal.jsx';

/** Asks for the mandatory reason of an audited command. */
export function ReasonDialog({ title, description, confirmLabel, onConfirm, onClose, danger = false }) {
  const { t } = useTranslation('common');
  const [reason, setReason] = useState('');
  const [busy, setBusy] = useState(false);
  const inputRef = useRef(null);
  const submit = async (event) => {
    event.preventDefault();
    if (!reason.trim()) return;
    setBusy(true);
    try {
      await onConfirm(reason.trim());
    } finally {
      setBusy(false);
    }
  };
  return (
    <Modal title={title} onClose={onClose} initialFocusRef={inputRef}>
      <form onSubmit={submit} className="form">
        {description && <p>{description}</p>}
        <label className="field">
          <span>{t('labels.reason')}</span>
          <textarea ref={inputRef} required maxLength={500} value={reason} onChange={(e) => setReason(e.target.value)} />
        </label>
        <div className="form-actions">
          <button type="button" className="btn btn-secondary" onClick={onClose}>{t('buttons.cancel')}</button>
          <button type="submit" className={`btn ${danger ? 'btn-danger' : ''}`} disabled={busy || !reason.trim()}>{confirmLabel}</button>
        </div>
      </form>
    </Modal>
  );
}
