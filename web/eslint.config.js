import js from '@eslint/js';
import globals from 'globals';
import reactHooks from 'eslint-plugin-react-hooks';

export default [
  { ignores: ['dist', 'node_modules', 'playwright-report', 'test-results'] },
  {
    files: ['**/*.{js,jsx}'],
    languageOptions: {
      ecmaVersion: 2023,
      sourceType: 'module',
      parserOptions: { ecmaFeatures: { jsx: true } },
      globals: { ...globals.browser, ...globals.node },
    },
    plugins: { 'react-hooks': reactHooks },
    rules: {
      ...js.configs.recommended.rules,
      ...reactHooks.configs.recommended.rules,
      'no-unused-vars': ['error', { varsIgnorePattern: '^[A-Z_]|^React$', argsIgnorePattern: '^_|^[A-Z]' }],
      'no-console': ['error', { allow: ['error', 'warn'] }],
    },
  },
  {
    files: ['scripts/**', 'src/i18n/verifyParity.js', 'e2e/**', 'playwright.config.js'],
    rules: { 'no-console': 'off' },
  },
  {
    files: ['test/**', 'e2e/**', '**/*.test.{js,jsx}'],
    languageOptions: { globals: { ...globals.browser, ...globals.node, vi: 'readonly', describe: 'readonly', it: 'readonly', expect: 'readonly', beforeEach: 'readonly', afterEach: 'readonly' } },
  },
];
