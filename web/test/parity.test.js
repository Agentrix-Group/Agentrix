import { describe, it, expect } from 'vitest';
import { checkParity } from '../src/i18n/verifyParity.js';

describe('Catalog Key Parity', () => {
  it('guarantees 100% parity between Spanish and English catalogs with no missing keys', () => {
    const isParity = checkParity();
    expect(isParity).toBe(true);
  });
});
