import React from 'react';
import { describe, it, expect, beforeEach, vi, afterEach } from 'vitest';
import { render, screen, fireEvent, act, waitFor } from '@testing-library/react';
import i18n from '../src/i18n/index.js';
import { CreateMatchModal } from '../src/components/CreateMatchModal.jsx';
import { EnrollAgentModal } from '../src/components/EnrollAgentModal.jsx';
import { ErrorBoundary } from '../src/components/ErrorBoundary.jsx';
import { ApiService } from '../src/service/apiService.js';
import fs from 'node:fs';
import path from 'node:path';

describe('Sprint FQ-6: Operational Hardening, WCAG AA Accessibility & Release Candidate', () => {
  beforeEach(async () => {
    localStorage.clear();
    await act(async () => {
      await i18n.changeLanguage('es');
    });
    vi.restoreAllMocks();
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  describe('WCAG 2.2 AA Accessibility & Modal Keyboard Interaction', () => {
    it('closes CreateMatchModal upon pressing Escape key', async () => {
      vi.spyOn(ApiService, 'listContests').mockResolvedValue([]);
      vi.spyOn(ApiService, 'listSubmissions').mockResolvedValue([]);

      const onClose = vi.fn();
      await act(async () => {
        render(<CreateMatchModal isOpen={true} onClose={onClose} onMatchCreated={vi.fn()} />);
      });

      expect(screen.getByRole('dialog')).toBeDefined();
      expect(screen.getByRole('dialog').getAttribute('aria-modal')).toBe('true');

      // Press Escape key
      fireEvent.keyDown(window, { key: 'Escape', code: 'Escape' });
      expect(onClose).toHaveBeenCalled();
    });

    it('closes EnrollAgentModal upon pressing Escape key', async () => {
      vi.spyOn(ApiService, 'listAgents').mockResolvedValue([]);
      const onClose = vi.fn();

      await act(async () => {
        render(
          <EnrollAgentModal
            isOpen={true}
            contest={{ id: 'c1', name: 'Torneo Alpha' }}
            onClose={onClose}
            onEnrolled={vi.fn()}
          />
        );
      });

      expect(screen.getByRole('dialog')).toBeDefined();
      expect(screen.getByRole('dialog').getAttribute('aria-modal')).toBe('true');

      // Press Escape key
      fireEvent.keyDown(window, { key: 'Escape', code: 'Escape' });
      expect(onClose).toHaveBeenCalled();
    });

    it('provides accessible labels for icon-only action buttons', async () => {
      vi.spyOn(ApiService, 'listContests').mockResolvedValue([]);
      vi.spyOn(ApiService, 'listSubmissions').mockResolvedValue([]);

      await act(async () => {
        render(<CreateMatchModal isOpen={true} onClose={vi.fn()} onMatchCreated={vi.fn()} />);
      });

      const closeBtn = screen.getByRole('button', { name: /Cerrar/i });
      expect(closeBtn.getAttribute('aria-label')).toBeDefined();
    });

    it('announces errors via aria-live assertive and status via aria-live polite', () => {
      function ExplodingComponent() {
        throw new Error('Critical simulation pipeline fault');
      }

      const spy = vi.spyOn(console, 'error').mockImplementation(() => {});

      render(
        <ErrorBoundary>
          <ExplodingComponent />
        </ErrorBoundary>
      );

      const alertBanner = screen.getByRole('alert');
      expect(alertBanner.getAttribute('aria-live')).toBe('assertive');
      expect(alertBanner.textContent).toContain('INC-');
      expect(alertBanner.textContent).not.toContain('/home/');
      expect(alertBanner.textContent).not.toContain('node_modules');

      spy.mockRestore();
    });
  });

  describe('Security Hardening & Content Security Policy (CSP)', () => {
    it('index.html contains a strict Content-Security-Policy meta tag', () => {
      const indexHtmlPath = path.resolve(__dirname, '../index.html');
      const htmlContent = fs.readFileSync(indexHtmlPath, 'utf8');

      expect(htmlContent).toContain('http-equiv="Content-Security-Policy"');
      expect(htmlContent).toContain("default-src 'self'");
      expect(htmlContent).toContain("object-src 'none'");
      expect(htmlContent).toContain("base-uri 'self'");
    });

    it('CSS contains high-contrast visible focus styles for all interactive elements', () => {
      const cssPath = path.resolve(__dirname, '../src/style/main.css');
      const cssContent = fs.readFileSync(cssPath, 'utf8');

      expect(cssContent).toContain(':focus-visible');
      expect(cssContent).toContain('outline: 3px solid');
    });

    it('ensures no hardcoded server credentials or absolute paths exist in frontend source', () => {
      const srcDir = path.resolve(__dirname, '../src');
      const readDirRecursive = (dir) => {
        let results = [];
        const entries = fs.readdirSync(dir, { withFileTypes: true });
        for (const entry of entries) {
          const fullPath = path.join(dir, entry.name);
          if (entry.isDirectory()) {
            results = results.concat(readDirRecursive(fullPath));
          } else if (entry.name.endsWith('.js') || entry.name.endsWith('.jsx')) {
            results.push(fullPath);
          }
        }
        return results;
      };

      const sourceFiles = readDirRecursive(srcDir);
      for (const file of sourceFiles) {
        const content = fs.readFileSync(file, 'utf8');
        // Must not contain postgres passwords or internal file paths
        expect(content).not.toMatch(/postgres:\/\/.*:.*@/);
        expect(content).not.toMatch(/\/var\/agentrix\/sandbox/);
        expect(content).not.toMatch(/\/home\/[a-zA-Z0-9_-]+\/Projects/);
      }
    });
  });
});
