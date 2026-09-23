// Lee un paquete ZIP de bot en el navegador para mostrar su contenido y
// runtime antes de subirlo (ADR-0014, N5). Replica las reglas de
// src/service/bundle.go; el servidor sigue siendo quien decide.

export const MAX_BUNDLE_BYTES = 50 * 1024 * 1024;
export const MAX_MODULE_BYTES = 1024 * 1024;
export const MAX_MODEL_BYTES = 20 * 1024 * 1024;
export const MAX_BUNDLE_FILES = 64;

export const RUNTIME_STDLIB = 'python-stdlib';
export const RUNTIME_ML_CPU = 'python-ml-cpu';

const MODULE_NAME = /^[A-Za-z_][A-Za-z0-9_]*\.py$/;
const MODEL_SEGMENT = /^[A-Za-z0-9_][A-Za-z0-9_.-]*$/;
const MODEL_EXTENSIONS = ['.onnx', '.safetensors', '.npz', '.json'];
const MANIFEST_FIELDS = ['name', 'entrypoint', 'protocol_version', 'runtime'];

const EOCD_SIGNATURE = 0x06054b50;
const CENTRAL_SIGNATURE = 0x02014b50;
const LOCAL_SIGNATURE = 0x04034b50;

/** Error de lectura: el ZIP no se pudo interpretar en el navegador. */
export class UnreadableZipError extends Error {}

/**
 * Devuelve las entradas del directorio central del ZIP:
 * { name, method, compressedSize, size, offset, isDir }.
 */
export function readZipEntries(buffer) {
  const view = new DataView(buffer);
  const minEnd = Math.max(0, buffer.byteLength - 22 - 0xffff);
  let eocd = -1;
  for (let i = buffer.byteLength - 22; i >= minEnd; i -= 1) {
    if (view.getUint32(i, true) === EOCD_SIGNATURE) {
      eocd = i;
      break;
    }
  }
  if (eocd < 0) throw new UnreadableZipError('end of central directory not found');
  const count = view.getUint16(eocd + 10, true);
  let offset = view.getUint32(eocd + 16, true);
  if (count === 0xffff || offset === 0xffffffff) throw new UnreadableZipError('ZIP64 is not supported here');

  const decoder = new TextDecoder();
  const entries = [];
  for (let i = 0; i < count; i += 1) {
    if (offset + 46 > buffer.byteLength || view.getUint32(offset, true) !== CENTRAL_SIGNATURE) {
      throw new UnreadableZipError('malformed central directory');
    }
    const nameLength = view.getUint16(offset + 28, true);
    const extraLength = view.getUint16(offset + 30, true);
    const commentLength = view.getUint16(offset + 32, true);
    const name = decoder.decode(new Uint8Array(buffer, offset + 46, nameLength));
    entries.push({
      name,
      method: view.getUint16(offset + 10, true),
      compressedSize: view.getUint32(offset + 20, true),
      size: view.getUint32(offset + 24, true),
      offset: view.getUint32(offset + 42, true),
      isDir: name.endsWith('/'),
    });
    offset += 46 + nameLength + extraLength + commentLength;
  }
  return entries;
}

/** Descomprime una entrada (stored o deflate) y la devuelve como texto. */
export async function readZipEntryText(buffer, entry) {
  const view = new DataView(buffer);
  if (view.getUint32(entry.offset, true) !== LOCAL_SIGNATURE) {
    throw new UnreadableZipError('malformed local header');
  }
  const start = entry.offset + 30 + view.getUint16(entry.offset + 26, true) + view.getUint16(entry.offset + 28, true);
  const data = new Uint8Array(buffer, start, entry.compressedSize);
  if (entry.method === 0) return new TextDecoder().decode(data);
  if (entry.method !== 8 || typeof DecompressionStream === 'undefined') {
    throw new UnreadableZipError('unsupported compression');
  }
  const stream = new Response(data).body.pipeThrough(new DecompressionStream('deflate-raw'));
  return new Response(stream).text();
}

function extension(name) {
  const dot = name.lastIndexOf('.');
  return dot < 0 ? '' : name.slice(dot).toLowerCase();
}

/**
 * Clasifica una ruta como lo hace classifyBundlePath en el servidor.
 * Devuelve { kind: 'manifest'|'module'|'model', limit } o { problem }.
 */
export function classifyBundlePath(name) {
  if (name === 'agentrix.json') return { kind: 'manifest', limit: MAX_MODULE_BYTES };
  const segments = name.split('/');
  if (segments.some((s) => s === '' || s === '.' || s === '..') || name.startsWith('/') || name.includes('\\')) {
    return { problem: { code: 'unsafePath', params: { name } } };
  }
  if (segments.length === 1) {
    return MODULE_NAME.test(name)
      ? { kind: 'module', limit: MAX_MODULE_BYTES }
      : { problem: { code: 'unexpectedRoot', params: { name } } };
  }
  if (segments[0] !== 'model') return { problem: { code: 'unexpectedPath', params: { name } } };
  if (!segments.slice(1).every((s) => MODEL_SEGMENT.test(s))) {
    return { problem: { code: 'unsafePath', params: { name } } };
  }
  if (!MODEL_EXTENSIONS.includes(extension(name))) {
    return { problem: { code: 'modelExtension', params: { name } } };
  }
  return { kind: 'model', limit: MAX_MODEL_BYTES };
}

function checkManifest(text) {
  let manifest;
  try {
    manifest = JSON.parse(text);
  } catch {
    return { problem: { code: 'invalidManifest', params: {} } };
  }
  if (!manifest || typeof manifest !== 'object' || Array.isArray(manifest)) {
    return { problem: { code: 'invalidManifest', params: {} } };
  }
  // encoding/json compara los nombres de campo sin distinguir mayúsculas.
  manifest = Object.fromEntries(Object.entries(manifest).map(([key, value]) => [key.toLowerCase(), value]));
  const unknown = Object.keys(manifest).find((key) => !MANIFEST_FIELDS.includes(key));
  if (unknown) return { problem: { code: 'unknownManifestField', params: { field: unknown } } };
  const name = typeof manifest.name === 'string' ? manifest.name : '';
  if (name.length === 0 || new TextEncoder().encode(name).length > 80) {
    return { problem: { code: 'invalidName', params: {} } };
  }
  const runtime = manifest.runtime || RUNTIME_STDLIB;
  if (runtime !== RUNTIME_STDLIB && runtime !== RUNTIME_ML_CPU) {
    return { problem: { code: 'invalidRuntime', params: { runtime: String(manifest.runtime) } } };
  }
  return { name, runtime };
}

/**
 * Inspecciona un paquete. Devuelve
 * { files: [{ name, size, kind }], runtime, botName, problems: [{ code, params }] }
 * o lanza UnreadableZipError si el navegador no puede leerlo.
 */
export async function inspectBundle(buffer) {
  const entries = readZipEntries(buffer).filter((entry) => !entry.isDir);
  const problems = [];
  const files = [];
  let total = 0;

  if (entries.length > MAX_BUNDLE_FILES) {
    problems.push({ code: 'tooManyFiles', params: { count: entries.length, max: MAX_BUNDLE_FILES } });
  }
  for (const entry of entries) {
    const result = classifyBundlePath(entry.name);
    if (result.problem) {
      problems.push(result.problem);
      continue;
    }
    if (entry.size > result.limit) {
      problems.push({ code: 'fileTooLarge', params: { name: entry.name, limitMb: result.limit / (1024 * 1024) } });
    }
    total += entry.size;
    files.push({ name: entry.name, size: entry.size, kind: result.kind });
  }
  if (total > MAX_BUNDLE_BYTES) problems.push({ code: 'totalTooLarge', params: {} });

  const manifestEntry = entries.find((entry) => entry.name === 'agentrix.json');
  const botEntry = entries.find((entry) => entry.name === 'bot.py');
  if (!manifestEntry || !botEntry || botEntry.size === 0) {
    problems.push({ code: 'missingRequired', params: {} });
  }

  let runtime = null;
  let botName = null;
  if (manifestEntry && manifestEntry.size <= MAX_MODULE_BYTES) {
    const checked = checkManifest(await readZipEntryText(buffer, manifestEntry));
    if (checked.problem) {
      problems.push(checked.problem);
    } else {
      runtime = checked.runtime;
      botName = checked.name;
    }
  }

  files.sort((a, b) => a.name.localeCompare(b.name));
  return { files, runtime, botName, problems };
}
