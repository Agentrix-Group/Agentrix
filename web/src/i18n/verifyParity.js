import fs from 'fs';
import path from 'path';
import { fileURLToPath } from 'url';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

const esDir = path.join(__dirname, 'locales', 'es');
const enDir = path.join(__dirname, 'locales', 'en');

function getKeys(obj, prefix = '') {
  let keys = [];
  for (const key of Object.keys(obj)) {
    const fullPath = prefix ? `${prefix}.${key}` : key;
    if (typeof obj[key] === 'object' && obj[key] !== null && !Array.isArray(obj[key])) {
      keys = keys.concat(getKeys(obj[key], fullPath));
    } else {
      keys.push(fullPath);
    }
  }
  return keys.sort();
}

export function checkParity() {
  const esFiles = fs.readdirSync(esDir).filter((f) => f.endsWith('.json')).sort();
  const enFiles = fs.readdirSync(enDir).filter((f) => f.endsWith('.json')).sort();

  let hasError = false;

  // Compare files
  for (const file of esFiles) {
    if (!enFiles.includes(file)) {
      console.error(`[Parity Error] File '${file}' found in 'es' but missing in 'en'`);
      hasError = true;
    }
  }

  for (const file of enFiles) {
    if (!esFiles.includes(file)) {
      console.error(`[Parity Error] File '${file}' found in 'en' but missing in 'es'`);
      hasError = true;
    }
  }

  // Compare keys within common files
  for (const file of esFiles) {
    if (!enFiles.includes(file)) continue;

    const esContent = JSON.parse(fs.readFileSync(path.join(esDir, file), 'utf-8'));
    const enContent = JSON.parse(fs.readFileSync(path.join(enDir, file), 'utf-8'));

    const esKeys = getKeys(esContent);
    const enKeys = getKeys(enContent);

    for (const key of esKeys) {
      if (!enKeys.includes(key)) {
        console.error(`[Parity Error] Key '${key}' in ${file} missing in 'en'`);
        hasError = true;
      }
    }

    for (const key of enKeys) {
      if (!esKeys.includes(key)) {
        console.error(`[Parity Error] Key '${key}' in ${file} missing in 'es'`);
        hasError = true;
      }
    }
  }

  if (hasError) {
    console.error('\n❌ Parity verification failed: translation catalogs have mismatched keys.');
    return false;
  }

  console.log(`✅ Parity verification passed: all ${esFiles.length} catalogs and keys are 100% synchronized!`);
  return true;
}

// Execute if run directly
if (process.argv[1] === fileURLToPath(import.meta.url)) {
  const passed = checkParity();
  process.exit(passed ? 0 : 1);
}
