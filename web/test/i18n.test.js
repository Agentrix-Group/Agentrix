import { describe, it, expect, afterEach } from 'vitest';
import i18n, { resolveInitialLanguage, setLanguage } from '../src/i18n/index.js';

describe('i18n', () => {
  afterEach(() => setLanguage('es'));

  it('resolves the stored preference before the browser language', () => {
    window.localStorage.setItem('agentrix_lang', 'en');
    expect(resolveInitialLanguage()).toBe('en');
    window.localStorage.setItem('agentrix_lang', 'xx');
    expect(['es', 'en']).toContain(resolveInitialLanguage());
  });

  it('switches language, persists it and updates <html lang>', () => {
    setLanguage('en');
    expect(i18n.language).toBe('en');
    expect(window.localStorage.getItem('agentrix_lang')).toBe('en');
    expect(document.documentElement.lang).toBe('en');
    expect(i18n.t('navigation:contests')).toBe('Contests');
    setLanguage('es');
    expect(i18n.t('navigation:contests')).toBe('Concursos');
  });

  it('translates canonical states and backend error codes', () => {
    expect(i18n.t('common:states.registration_open')).toBe('Inscripción abierta');
    expect(i18n.t('errors:codes.roster_frozen')).toBe('La lista de participantes está congelada.');
    setLanguage('en');
    expect(i18n.t('errors:codes.roster_frozen')).toBe('The roster is frozen.');
  });
});
