import React, { useState, useRef } from 'react';
import { useTranslation } from 'react-i18next';
import { FileArchive, UploadCloud, XCircle, CheckCircle2, AlertCircle } from 'lucide-react';

const MAX_FILE_SIZE_BYTES = 2 * 1024 * 1024; // 2 MiB

export function formatBytes(bytes) {
  if (bytes === 0) return '0 B';
  const k = 1024;
  const sizes = ['B', 'KB', 'MB'];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + ' ' + sizes[i];
}

export function validateBundleFile(file, t) {
  if (!file) {
    return { valid: false, error: null };
  }
  const name = file.name || '';
  const isZip = name.toLowerCase().endsWith('.zip') ||
    file.type === 'application/zip' ||
    file.type === 'application/x-zip-compressed';

  if (!isZip) {
    return {
      valid: false,
      error: t('agents:messages.invalidZipType', { defaultValue: 'Only .zip package files are accepted.' }),
    };
  }

  if (file.size > MAX_FILE_SIZE_BYTES) {
    return {
      valid: false,
      error: t('agents:messages.fileTooLarge', { defaultValue: 'File exceeds the maximum allowed limit of 2 MiB.' }),
    };
  }

  return { valid: true, error: null };
}

export function BundleDropzone({
  bundle,
  onFileSelect,
  disabled = false,
  validationError,
  setValidationError,
}) {
  const { t } = useTranslation(['agents', 'common']);
  const [isDragging, setIsDragging] = useState(false);
  const fileInputRef = useRef(null);

  const processFile = (file) => {
    if (!file) {
      onFileSelect(null);
      setValidationError(null);
      return;
    }
    const result = validateBundleFile(file, t);
    if (!result.valid) {
      onFileSelect(null);
      setValidationError(result.error);
      if (fileInputRef.current) {
        fileInputRef.current.value = '';
      }
    } else {
      setValidationError(null);
      onFileSelect(file);
    }
  };

  const handleDragOver = (e) => {
    e.preventDefault();
    if (disabled) return;
    setIsDragging(true);
  };

  const handleDragLeave = (e) => {
    e.preventDefault();
    setIsDragging(false);
  };

  const handleDrop = (e) => {
    e.preventDefault();
    setIsDragging(false);
    if (disabled) return;
    const droppedFile = e.dataTransfer.files?.[0];
    if (droppedFile) {
      processFile(droppedFile);
    }
  };

  const handleInputChange = (e) => {
    const selectedFile = e.target.files?.[0];
    processFile(selectedFile || null);
  };

  const handleClear = (e) => {
    e.preventDefault();
    e.stopPropagation();
    onFileSelect(null);
    setValidationError(null);
    if (fileInputRef.current) {
      fileInputRef.current.value = '';
    }
  };

  return (
    <div className="bundle-dropzone-wrapper">
      <label
        className={`bundle-dropzone ${isDragging ? 'drag-active' : ''} ${disabled ? 'disabled' : ''}`}
        onDragOver={handleDragOver}
        onDragLeave={handleDragLeave}
        onDrop={handleDrop}
        htmlFor="bundle-file-input"
        style={{ opacity: disabled ? 0.6 : 1, cursor: disabled ? 'not-allowed' : 'pointer' }}
      >
        <FileArchive size={32} aria-hidden="true" />
        <strong>
          {bundle
            ? bundle.name
            : t('agents:submission.chooseZip', { defaultValue: 'Selecciona el paquete ZIP' })}
        </strong>
        <span>
          {t('agents:submission.zipHelp', { defaultValue: 'Debe contener agentrix.json y bot.py en la raíz. Máximo 2 MiB.' })}
        </span>
        <input
          id="bundle-file-input"
          ref={fileInputRef}
          type="file"
          accept=".zip,application/zip,application/x-zip-compressed"
          onChange={handleInputChange}
          disabled={disabled}
        />
      </label>

      {validationError && (
        <div
          role="alert"
          style={{
            display: 'flex',
            alignItems: 'center',
            gap: '8px',
            padding: '10px 14px',
            background: 'var(--danger-bg)',
            color: 'var(--danger-text)',
            borderRadius: '6px',
            marginTop: '8px',
            fontSize: '0.86rem',
          }}
        >
          <AlertCircle size={18} aria-hidden="true" />
          <span>{validationError}</span>
        </div>
      )}

      {bundle && !validationError && (
        <div className="bundle-selected-info">
          <div className="bundle-selected-meta">
            <CheckCircle2 size={18} color="var(--success-text)" aria-hidden="true" />
            <span>
              <strong>{bundle.name}</strong> ({formatBytes(bundle.size)})
            </span>
          </div>
          {!disabled && (
            <button
              type="button"
              onClick={handleClear}
              className="btn btn-compact"
              style={{ padding: '4px 8px', fontSize: '0.8rem' }}
              title={t('agents:submission.clearFile', { defaultValue: 'Quitar archivo' })}
            >
              <XCircle size={14} aria-hidden="true" /> {t('agents:submission.clearFile', { defaultValue: 'Quitar' })}
            </button>
          )}
        </div>
      )}
    </div>
  );
}
