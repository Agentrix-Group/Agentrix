import i18n from 'i18next';
import { initReactI18next } from 'react-i18next';

// Spanish locales
import esCommon from './locales/es/common.json';
import esNavigation from './locales/es/navigation.json';
import esAuth from './locales/es/auth.json';
import esHome from './locales/es/home.json';
import esAgents from './locales/es/agents.json';
import esMatches from './locales/es/matches.json';
import esRankings from './locales/es/rankings.json';
import esViewer from './locales/es/viewer.json';
import esErrors from './locales/es/errors.json';

// English locales
import enCommon from './locales/en/common.json';
import enNavigation from './locales/en/navigation.json';
import enAuth from './locales/en/auth.json';
import enHome from './locales/en/home.json';
import enAgents from './locales/en/agents.json';
import enMatches from './locales/en/matches.json';
import enRankings from './locales/en/rankings.json';
import enViewer from './locales/en/viewer.json';
import enErrors from './locales/en/errors.json';

export const defaultNamespace = 'common';
export const namespaces = [
  'common',
  'navigation',
  'auth',
  'home',
  'agents',
  'matches',
  'rankings',
  'viewer',
  'errors',
];

export const resources = {
  es: {
    common: esCommon,
    navigation: esNavigation,
    auth: esAuth,
    home: esHome,
    agents: esAgents,
    matches: esMatches,
    rankings: esRankings,
    viewer: esViewer,
    errors: esErrors,
  },
  en: {
    common: enCommon,
    navigation: enNavigation,
    auth: enAuth,
    home: enHome,
    agents: enAgents,
    matches: enMatches,
    rankings: enRankings,
    viewer: enViewer,
    errors: enErrors,
  },
};

export function resolveInitialLanguage() {
  // 1. Stored preference
  if (typeof window !== 'undefined' && window.localStorage) {
    const stored = window.localStorage.getItem('agentrix_lang');
    if (stored === 'es' || stored === 'en') {
      return stored;
    }
  }

  // 2. Browser language
  if (typeof navigator !== 'undefined') {
    const browserLang = (navigator.language || (navigator.languages && navigator.languages[0]) || '').toLowerCase();
    if (browserLang.startsWith('en')) {
      return 'en';
    }
    if (browserLang.startsWith('es')) {
      return 'es';
    }
  }

  // 3. Default fallback
  return 'es';
}

const initialLang = resolveInitialLanguage();

i18n
  .use(initReactI18next)
  .init({
    resources,
    lng: initialLang,
    fallbackLng: 'es',
    supportedLngs: ['es', 'en'],
    defaultNS: defaultNamespace,
    ns: namespaces,
    interpolation: {
      escapeValue: false, // React already escapes values
    },
    react: {
      useSuspense: false,
    },
  });

// Update document language tag and persist on change
if (typeof document !== 'undefined') {
  document.documentElement.lang = initialLang;
}

i18n.on('languageChanged', (lng) => {
  if (typeof document !== 'undefined') {
    document.documentElement.lang = lng;
  }
  if (typeof window !== 'undefined' && window.localStorage) {
    window.localStorage.setItem('agentrix_lang', lng);
  }
});

export default i18n;
