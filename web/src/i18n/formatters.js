/**
 * Regional localized formatters for dates, times, numbers, and currencies
 * Supports 'es-PE' for Spanish and 'en-US' for English.
 */

export function getLocale(lang) {
  return lang === 'en' ? 'en-US' : 'es-PE';
}

export function formatDate(value, lang = 'es', options = {}) {
  if (!value) return '';
  const date = value instanceof Date ? value : new Date(value);
  if (isNaN(date.getTime())) return '';
  const locale = getLocale(lang);
  return new Intl.DateTimeFormat(locale, {
    year: 'numeric',
    month: 'short',
    day: 'numeric',
    ...options,
  }).format(date);
}

export function formatDateTime(value, lang = 'es', options = {}) {
  if (!value) return '';
  const date = value instanceof Date ? value : new Date(value);
  if (isNaN(date.getTime())) return '';
  const locale = getLocale(lang);
  return new Intl.DateTimeFormat(locale, {
    year: 'numeric',
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
    ...options,
  }).format(date);
}

export function formatNumber(value, lang = 'es', options = {}) {
  if (value === null || value === undefined || isNaN(Number(value))) return '0';
  const locale = getLocale(lang);
  return new Intl.NumberFormat(locale, options).format(Number(value));
}
