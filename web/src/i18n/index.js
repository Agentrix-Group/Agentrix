import i18n from 'i18next';
import { initReactI18next } from 'react-i18next';

import esCommon from './locales/es/common.json';
import esNavigation from './locales/es/navigation.json';
import esAuth from './locales/es/auth.json';
import esHome from './locales/es/home.json';
import esContests from './locales/es/contests.json';
import esMatches from './locales/es/matches.json';
import esRankings from './locales/es/rankings.json';
import esAgents from './locales/es/agents.json';
import esAdmin from './locales/es/admin.json';
import esViewer from './locales/es/viewer.json';
import esErrors from './locales/es/errors.json';
import enCommon from './locales/en/common.json';
import enNavigation from './locales/en/navigation.json';
import enAuth from './locales/en/auth.json';
import enHome from './locales/en/home.json';
import enContests from './locales/en/contests.json';
import enMatches from './locales/en/matches.json';
import enRankings from './locales/en/rankings.json';
import enAgents from './locales/en/agents.json';
import enAdmin from './locales/en/admin.json';
import enViewer from './locales/en/viewer.json';
import enErrors from './locales/en/errors.json';

export const namespaces = ['common', 'navigation', 'auth', 'home', 'contests', 'matches', 'rankings', 'agents', 'admin', 'viewer', 'errors'];
export const resources = { es: { common: esCommon, navigation: esNavigation, auth: esAuth, home: esHome, contests: esContests, matches: esMatches, rankings: esRankings, agents: esAgents, admin: esAdmin, viewer: esViewer, errors: esErrors }, en: { common: enCommon, navigation: enNavigation, auth: enAuth, home: enHome, contests: enContests, matches: enMatches, rankings: enRankings, agents: enAgents, admin: enAdmin, viewer: enViewer, errors: enErrors } };

const STORAGE_KEY = 'agentrix_lang';

export function resolveInitialLanguage() {
  try {
    const stored = window.localStorage.getItem(STORAGE_KEY);
    if (stored === 'es' || stored === 'en') return stored;
  } catch {
    // storage unavailable (private mode): fall back to the browser language
  }
  const browser = (typeof navigator !== 'undefined' && navigator.language) || '';
  return browser.toLowerCase().startsWith('en') ? 'en' : 'es';
}

i18n.use(initReactI18next).init({
  resources,
  lng: resolveInitialLanguage(),
  fallbackLng: 'es',
  ns: namespaces,
  defaultNS: 'common',
  interpolation: { escapeValue: false },
  react: { useSuspense: false },
});

if (typeof document !== 'undefined') document.documentElement.lang = i18n.language;

export function setLanguage(lang) {
  i18n.changeLanguage(lang);
  document.documentElement.lang = lang;
  try {
    window.localStorage.setItem(STORAGE_KEY, lang);
  } catch {
    // preference not persisted
  }
}

export default i18n;
