import React, { useId, useRef, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { FileArchive, Loader2 } from 'lucide-react';

export const MAX_BUNDLE_BYTES = 2 * 1024 * 1024;

export function validateBundleFile(file) {
  if (!file) return 'noFile';
  const name = (file.name || '').toLowerCase();
  if (!name.endsWith('.zip')) return 'invalidZipType';
  if (file.size > MAX_BUNDLE_BYTES) return 'fileTooLarge';
  if (file.size === 0) return 'emptyFile';
  return null;
}

/** Drag & drop (or picker) for bot bundles. Uploading shows progress; the
 * admission verdict is reported by the submissions list, from the API. */
export function BundleDropzone({ onUpload, busy = false }) {
  const { t } = useTranslation('agents');
  const inputId = useId();
  const inputRef = useRef(null);
  const [dragging, setDragging] = useState(false);
  const [error, setError] = useState(null);

  const handle = (file) => {
    const problem = validateBundleFile(file);
    setError(problem ? t(`bundle.${problem}`) : null);
    if (inputRef.current) inputRef.current.value = '';
    if (!problem) onUpload(file);
  };

  return (
    <div className="dropzone-wrap">
      <label htmlFor={inputId} className={`dropzone ${dragging ? 'dragging' : ''} ${busy ? 'busy' : ''}`}
        onDragOver={(e) => { e.preventDefault(); if (!busy) setDragging(true); }}
        onDragLeave={() => setDragging(false)}
        onDrop={(e) => { e.preventDefault(); setDragging(false); if (!busy) handle(e.dataTransfer.files?.[0]); }}>
        {busy ? <Loader2 size={28} className="spin" aria-hidden="true" /> : <FileArchive size={28} aria-hidden="true" />}
        <strong>{busy ? t('bundle.uploading') : t('bundle.prompt')}</strong>
        <span className="muted small">{t('bundle.hint')}</span>
      </label>
      <input ref={inputRef} id={inputId} type="file" accept=".zip,application/zip" className="visually-hidden" disabled={busy}
        onChange={(e) => handle(e.target.files?.[0])} />
      {error && <p className="form-error" role="alert">{error}</p>}
    </div>
  );
}
