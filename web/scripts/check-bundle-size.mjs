// Bundle budget: the entry chunk (loaded on every page) and the total
// JavaScript, gzip-compressed. Fails the build when exceeded.
import { readdirSync, readFileSync, statSync } from 'node:fs';
import { join } from 'node:path';
import { gzipSync } from 'node:zlib';

const BUDGET = { entryKB: 110, totalKB: 260 };
const dir = new URL('../dist/assets/', import.meta.url).pathname;
const html = readFileSync(new URL('../dist/index.html', import.meta.url), 'utf8');
const entryName = (html.match(/assets\/(index-[^"]+\.js)/) || [])[1];
let total = 0;
let entry = 0;
for (const file of readdirSync(dir).filter((f) => f.endsWith('.js'))) {
  const size = gzipSync(readFileSync(join(dir, file))).length;
  total += size;
  if (file === entryName) entry = size;
  console.log(`${file.padEnd(40)} ${(statSync(join(dir, file)).size / 1024).toFixed(1)} KB  gzip ${(size / 1024).toFixed(1)} KB`);
}
console.log(`entry ${(entry / 1024).toFixed(1)} KB gzip (budget ${BUDGET.entryKB}); total ${(total / 1024).toFixed(1)} KB gzip (budget ${BUDGET.totalKB})`);
if (!entryName) throw new Error('entry chunk not found in dist/index.html');
if (entry / 1024 > BUDGET.entryKB || total / 1024 > BUDGET.totalKB) {
  console.error('bundle budget exceeded');
  process.exit(1);
}
