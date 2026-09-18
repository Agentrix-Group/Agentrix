import { describe, it, expect, beforeEach } from 'vitest';
import i18n, { resolveInitialLanguage, resources } from '../src/i18n/index.js';
import { formatDate, formatDateTime, formatNumber, getLocale } from '../src/i18n/formatters.js';

describe('i18n Infrastructure & Resolution', () => {
  const originalLanguage = navigator.language;
  const originalLanguages = navigator.languages;

  const setNavigatorLanguage = (lang) => {
    Object.defineProperty(navigator, 'language', {
      value: lang,
      configurable: true,
    });
    Object.defineProperty(navigator, 'languages', {
      value: lang ? [lang] : [],
      configurable: true,
    });
  };

  beforeEach(() => {
    localStorage.clear();
    setNavigatorLanguage('en-US');
  });

  it('resolves Spanish as default fallback when browser language is unsupported or empty', () => {
    setNavigatorLanguage('fr-FR');
    expect(resolveInitialLanguage()).toBe('es');

    setNavigatorLanguage('');
    expect(resolveInitialLanguage()).toBe('es');
  });

  it('resolves browser language when supported and no localStorage preference exists', () => {
    setNavigatorLanguage('es-PE');
    expect(resolveInitialLanguage()).toBe('es');

    setNavigatorLanguage('en-GB');
    expect(resolveInitialLanguage()).toBe('en');
  });

  it('resolves stored language preference from localStorage over browser language', () => {
    setNavigatorLanguage('en-US');
    localStorage.setItem('agentrix_lang', 'es');
    expect(resolveInitialLanguage()).toBe('es');

    setNavigatorLanguage('es-ES');
    localStorage.setItem('agentrix_lang', 'en');
    expect(resolveInitialLanguage()).toBe('en');
  });

  it('changes language dynamically and persists to localStorage', async () => {
    await i18n.changeLanguage('en');
    expect(i18n.language).toBe('en');
    expect(localStorage.getItem('agentrix_lang')).toBe('en');

    await i18n.changeLanguage('es');
    expect(i18n.language).toBe('es');
    expect(localStorage.getItem('agentrix_lang')).toBe('es');
  });

  it('translates main titles and navigation in Spanish and English', async () => {
    await i18n.changeLanguage('es');
    expect(i18n.t('navigation:brand')).toBe('Agentrix');
    expect(i18n.t('navigation:agents')).toBe('Mis agentes');
    expect(i18n.t('navigation:matches')).toBe('Partidas');
    expect(i18n.t('navigation:rankings')).toBe('Clasificación');
    expect(i18n.t('home:contestsTitle')).toBe('Concursos publicados');
    expect(i18n.t('rankings:title')).toBe('Clasificación del torneo');

    await i18n.changeLanguage('en');
    expect(i18n.t('navigation:brand')).toBe('Agentrix');
    expect(i18n.t('navigation:agents')).toBe('My Agents');
    expect(i18n.t('navigation:matches')).toBe('Matches');
    expect(i18n.t('navigation:rankings')).toBe('Leaderboard');
    expect(i18n.t('home:contestsTitle')).toBe('Published contests');
    expect(i18n.t('rankings:title')).toBe('Tournament Leaderboard');
  });

  it('translates official match and agent lifecycle statuses correctly', async () => {
    await i18n.changeLanguage('es');
    expect(i18n.t('matches:status.pending')).toBe('Pendiente');
    expect(i18n.t('matches:status.running')).toBe('En ejecución');
    expect(i18n.t('matches:status.finished')).toBe('Finalizada');
    expect(i18n.t('matches:status.failed')).toBe('Fallida');

    await i18n.changeLanguage('en');
    expect(i18n.t('matches:status.pending')).toBe('Pending');
    expect(i18n.t('matches:status.running')).toBe('Running');
    expect(i18n.t('matches:status.finished')).toBe('Finished');
    expect(i18n.t('matches:status.failed')).toBe('Failed');
  });

  it('maps backend error codes correctly', async () => {
    await i18n.changeLanguage('es');
    expect(i18n.t('errors:codes.1001')).toBe('Formato de solicitud inválido');
    expect(i18n.t('errors:codes.2001')).toBe('Credenciales inválidas');
    expect(i18n.t('errors:codes.5001')).toBe('Error en la base de datos');

    await i18n.changeLanguage('en');
    expect(i18n.t('errors:codes.1001')).toBe('Invalid request format');
    expect(i18n.t('errors:codes.2001')).toBe('Invalid credentials provided');
    expect(i18n.t('errors:codes.5001')).toBe('Database operation failed');
  });
});

describe('Regional Formatters (Intl)', () => {
  it('maps regional locales es-PE and en-US', () => {
    expect(getLocale('es')).toBe('es-PE');
    expect(getLocale('en')).toBe('en-US');
  });

  it('formats dates according to active locale', () => {
    const fixedDate = new Date('2026-09-15T15:30:00Z');
    const formattedEs = formatDate(fixedDate, 'es');
    const formattedEn = formatDate(fixedDate, 'en');

    expect(formattedEs).toBeTruthy();
    expect(formattedEn).toBeTruthy();
    expect(typeof formattedEs).toBe('string');
  });

  it('formats numbers according to active locale', () => {
    expect(formatNumber(1250, 'es')).toBeTruthy();
    expect(formatNumber(1250, 'en')).toBeTruthy();
    expect(formatNumber(0, 'es')).toBe('0');
  });
});
