import React, { useState, useRef, useEffect } from 'react';
import { useTranslation } from 'react-i18next';
import { FileArchive, XCircle, CheckCircle2, AlertCircle, Cpu, FileCode, Brain, FileJson } from 'lucide-react';
import { MAX_BUNDLE_BYTES, RUNTIME_ML_CPU, UnreadableZipError, inspectBundle } from './bundleInspector.js';

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

  if (file.size > MAX_BUNDLE_BYTES) {
    return {
      valid: false,
      error: t('agents:messages.fileTooLarge', { defaultValue: 'File exceeds the maximum allowed limit of 50 MB.' }),
    };
  }

  return { valid: true, error: null };
}

/** Traduce un problema del inspector a un mensaje para el usuario. */
export function describeBundleProblem(problem, t) {
  return t(`agents:bundle.problems.${problem.code}`, problem.params);
}

const KIND_ICONS = { manifest: FileJson, module: FileCode, model: Brain };

/** Lista el contenido de un paquete ya inspeccionado: runtime y archivos. */
export function BundleContents({ inspection }) {
  const { t } = useTranslation(['agents']);
  if (!inspection) return null;
  const runtime = inspection.runtime;
  return (
    <div className="bundle-contents" aria-label={t('agents:bundle.contentsTitle')}>
      <div className="bundle-contents-runtime">
        <Cpu size={16} aria-hidden="true" />
        <span>{t('agents:bundle.runtimeLabel')}</span>
        <strong>
          {runtime
            ? t(`agents:bundle.runtimes.${runtime === RUNTIME_ML_CPU ? 'mlCpu' : 'stdlib'}`)
            : t('agents:bundle.runtimes.unknown')}
        </strong>
      </div>
      {runtime && (
        <div className="bundle-contents-runtime-help">
          {t(`agents:bundle.runtimeHelp.${runtime === RUNTIME_ML_CPU ? 'mlCpu' : 'stdlib'}`)}
        </div>
      )}
      <ul className="bundle-contents-files">
        {inspection.files.map((file) => {
          const Icon = KIND_ICONS[file.kind] || FileCode;
          return (
            <li key={file.name}>
              <Icon size={14} aria-hidden="true" />
              <code>{file.name}</code>
              <span className="bundle-contents-kind">{t(`agents:bundle.kinds.${file.kind}`)}</span>
              <span className="bundle-contents-size">{formatBytes(file.size)}</span>
            </li>
          );
        })}
      </ul>
    </div>
  );
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
  const [inspection, setInspection] = useState(null);
  const fileInputRef = useRef(null);

  // Muestra runtime y archivos del ZIP elegido. Si el paquete rompe una
  // regla del servidor se avisa antes de subirlo; si el navegador no puede
  // leerlo, decide el servidor.
  useEffect(() => {
    let cancelled = false;
    setInspection(null);
    if (!bundle || typeof bundle.arrayBuffer !== 'function') return undefined;
    (async () => {
      try {
        const result = await inspectBundle(await bundle.arrayBuffer());
        if (cancelled) return;
        setInspection(result);
        if (result.problems.length > 0) {
          setValidationError(describeBundleProblem(result.problems[0], t));
        }
      } catch (err) {
        if (!(err instanceof UnreadableZipError) && !(err instanceof RangeError)) {
          console.warn('bundle inspection failed', err);
        }
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [bundle]);

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
          {t('agents:submission.zipHelp')}
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

      {bundle && inspection && <BundleContents inspection={inspection} />}

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
