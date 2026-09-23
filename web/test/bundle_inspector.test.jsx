import React from 'react';
import fs from 'fs';
import path from 'path';
import { describe, it, expect, beforeEach } from 'vitest';
import { render, screen, fireEvent, act, waitFor } from '@testing-library/react';
import i18n from '../src/i18n/index.js';
import { inspectBundle, readZipEntries, UnreadableZipError } from '../src/components/bundleInspector.js';
import { BundleDropzone } from '../src/components/BundleDropzone.jsx';
import { NeuralTemplateCard, NEURAL_TEMPLATES } from '../src/components/NeuralTemplateCard.jsx';

// ZIP mínimo sin compresión. `sizes` permite declarar en el directorio
// central un tamaño distinto del real, para probar los límites sin crear
// archivos de decenas de MB.
function makeZip(files, sizes = {}) {
  const encoder = new TextEncoder();
  const locals = [];
  const centrals = [];
  let offset = 0;
  for (const [name, content] of Object.entries(files)) {
    const nameBytes = encoder.encode(name);
    const data = typeof content === 'string' ? encoder.encode(content) : content;
    const declared = sizes[name] ?? data.length;
    const local = new DataView(new ArrayBuffer(30));
    local.setUint32(0, 0x04034b50, true);
    local.setUint32(18, data.length, true);
    local.setUint32(22, declared, true);
    local.setUint16(26, nameBytes.length, true);
    locals.push(new Uint8Array(local.buffer), nameBytes, data);
    const central = new DataView(new ArrayBuffer(46));
    central.setUint32(0, 0x02014b50, true);
    central.setUint32(20, data.length, true);
    central.setUint32(24, declared, true);
    central.setUint16(28, nameBytes.length, true);
    central.setUint32(42, offset, true);
    centrals.push(new Uint8Array(central.buffer), nameBytes);
    offset += 30 + nameBytes.length + data.length;
  }
  const centralSize = centrals.reduce((sum, part) => sum + part.length, 0);
  const eocd = new DataView(new ArrayBuffer(22));
  eocd.setUint32(0, 0x06054b50, true);
  eocd.setUint16(8, Object.keys(files).length, true);
  eocd.setUint16(10, Object.keys(files).length, true);
  eocd.setUint32(12, centralSize, true);
  eocd.setUint32(16, offset, true);
  const parts = [...locals, ...centrals, new Uint8Array(eocd.buffer)];
  const out = new Uint8Array(parts.reduce((sum, part) => sum + part.length, 0));
  let at = 0;
  for (const part of parts) {
    out.set(part, at);
    at += part.length;
  }
  return out.buffer;
}

const manifest = (extra = {}) =>
  JSON.stringify({ name: 'Neural', entrypoint: 'bot.py', protocol_version: '1.0', ...extra });

const codes = (result) => result.problems.map((p) => p.code);

function readTemplate(file) {
  const bytes = fs.readFileSync(path.join(__dirname, '..', 'public', file));
  return bytes.buffer.slice(bytes.byteOffset, bytes.byteOffset + bytes.byteLength);
}

describe('ADR-0014 N5: bundle inspector', () => {
  it('reads the published neural templates (deflate) with runtime and files', async () => {
    for (const { file } of NEURAL_TEMPLATES) {
      const result = await inspectBundle(readTemplate(file));
      expect(result.problems).toEqual([]);
      expect(result.runtime).toBe('python-ml-cpu');
      const names = result.files.map((f) => f.name);
      expect(names).toContain('agentrix.json');
      expect(names).toContain('bot.py');
      expect(names).toContain('policy.py');
      expect(result.files.filter((f) => f.kind === 'model')).toHaveLength(1);
    }
  });

  it('defaults to python-stdlib for a v1 bundle', async () => {
    const result = await inspectBundle(makeZip({ 'agentrix.json': manifest(), 'bot.py': 'pass\n' }));
    expect(result.problems).toEqual([]);
    expect(result.runtime).toBe('python-stdlib');
  });

  it('mirrors the server path rules', async () => {
    const result = await inspectBundle(
      makeZip({
        'agentrix.json': manifest(),
        'bot.py': 'pass\n',
        'README.md': 'x',
        'weights/w.npz': 'x',
        'model/w.pkl': 'x',
        'model/../evil.json': '{}',
        'model/sub/ok.onnx': 'x',
      })
    );
    expect(codes(result)).toEqual(['unexpectedRoot', 'unexpectedPath', 'modelExtension', 'unsafePath']);
    expect(result.files.map((f) => f.name)).toContain('model/sub/ok.onnx');
  });

  it('enforces per-file, total and count limits', async () => {
    const big = await inspectBundle(
      makeZip(
        { 'agentrix.json': manifest(), 'bot.py': 'pass\n', 'helpers.py': 'x', 'model/a.onnx': 'x', 'model/b.onnx': 'x', 'model/c.onnx': 'x' },
        { 'helpers.py': 1024 * 1024 + 1, 'model/a.onnx': 20 * 1024 * 1024 + 1, 'model/b.onnx': 19 * 1024 * 1024, 'model/c.onnx': 19 * 1024 * 1024 }
      )
    );
    expect(codes(big)).toEqual(['fileTooLarge', 'fileTooLarge', 'totalTooLarge']);

    const files = { 'agentrix.json': manifest(), 'bot.py': 'pass\n' };
    for (let i = 0; i < 63; i += 1) files[`m${i}.py`] = 'x';
    expect(codes(await inspectBundle(makeZip(files)))).toEqual(['tooManyFiles']);
  });

  it('checks agentrix.json like the server', async () => {
    const check = async (text, bot = 'pass\n') => codes(await inspectBundle(makeZip({ 'agentrix.json': text, 'bot.py': bot })));
    expect(await check('{')).toEqual(['invalidManifest']);
    expect(await check(manifest({ extra: 1 }))).toEqual(['unknownManifestField']);
    expect(await check(manifest({ name: '' }))).toEqual(['invalidName']);
    expect(await check(manifest({ runtime: 'python-gpu' }))).toEqual(['invalidRuntime']);
    expect(await check(manifest(), '')).toEqual(['missingRequired']);
    expect(codes(await inspectBundle(makeZip({ 'bot.py': 'pass\n' })))).toEqual(['missingRequired']);
  });

  it('reports unreadable archives instead of guessing', () => {
    expect(() => readZipEntries(new TextEncoder().encode('not a zip at all, just text').buffer)).toThrow(UnreadableZipError);
  });
});

describe('ADR-0014 N5: upload form shows the package contents', () => {
  beforeEach(async () => {
    await act(async () => {
      await i18n.changeLanguage('es');
    });
  });

  const renderWithFile = async (buffer, name) => {
    const file = new File([buffer], name, { type: 'application/zip' });
    let validationError = null;
    const view = render(
      <BundleDropzone bundle={file} onFileSelect={() => {}} validationError={null} setValidationError={(e) => { validationError = e; }} />
    );
    return { view, getError: () => validationError };
  };

  it('lists runtime and files for the ONNX template', async () => {
    const { getError } = await renderWithFile(readTemplate('starfighter-neural-onnx.zip'), 'onnx.zip');
    await waitFor(() => expect(screen.getByText(/Python ML en CPU/)).toBeDefined());
    expect(screen.getByText('model/policy.onnx')).toBeDefined();
    expect(screen.getByText('policy.py')).toBeDefined();
    expect(getError()).toBeNull();
  });

  it('blocks an invalid package with the reason before uploading', async () => {
    const { getError } = await renderWithFile(
      makeZip({ 'agentrix.json': manifest({ runtime: 'python-gpu' }), 'bot.py': 'pass\n' }),
      'bad.zip'
    );
    await waitFor(() => expect(getError()).toContain('python-gpu'));
  });
});

describe('ADR-0014 N5: neural templates card', () => {
  beforeEach(async () => {
    await act(async () => {
      await i18n.changeLanguage('es');
    });
  });

  it('links both templates and they exist in public/', () => {
    render(<NeuralTemplateCard />);
    for (const { file } of NEURAL_TEMPLATES) {
      const link = screen.getAllByRole('link').find((a) => a.getAttribute('href') === `/${file}`);
      expect(link).toBeDefined();
      expect(link.getAttribute('download')).toBe(file);
      expect(fs.existsSync(path.join(__dirname, '..', 'public', file))).toBe(true);
    }
  });

  it('expands the step-by-step guide', async () => {
    render(<NeuralTemplateCard />);
    await act(async () => {
      fireEvent.click(screen.getByRole('button', { name: /Cómo subir un bot con red neuronal/i }));
    });
    expect(screen.getAllByText(/python-ml-cpu/).length).toBeGreaterThan(0);
    expect(screen.getByText(/10 s para cargar el modelo/)).toBeDefined();
  });
});
