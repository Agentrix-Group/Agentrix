import React from 'react';
import { describe, it, expect, beforeEach, vi, afterEach } from 'vitest';
import { render, screen, fireEvent, act, waitFor } from '@testing-library/react';
import i18n from '../src/i18n/index.js';
import { BundleDropzone, validateBundleFile, formatBytes } from '../src/components/BundleDropzone.jsx';
import { StarterKitCard } from '../src/components/StarterKitCard.jsx';
import { AdmissionStatusBanner, sanitizeAdmissionError } from '../src/components/AdmissionStatusBanner.jsx';
import { AgentsPage } from '../src/pages/AgentsPage.jsx';
import { ApiService } from '../src/service/apiService.js';

describe('Sprint FQ-2: Asynchronous Bot Admission & Validation Feedback', () => {
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

  describe('Client-Side Bundle Validation (BundleDropzone)', () => {
    it('formats bytes correctly into human readable units', () => {
      expect(formatBytes(0)).toBe('0 B');
      expect(formatBytes(1024)).toBe('1 KB');
      expect(formatBytes(1024 * 1024 * 1.5)).toBe('1.5 MB');
    });

    it('rejects files larger than 2 MiB before uploading', () => {
      const mockT = (key) => key;
      const oversizedFile = new File(['a'.repeat(100)], 'huge.zip', { type: 'application/zip' });
      Object.defineProperty(oversizedFile, 'size', { value: 3 * 1024 * 1024 }); // 3 MiB

      const result = validateBundleFile(oversizedFile, mockT);
      expect(result.valid).toBe(false);
      expect(result.error).toBe('agents:messages.fileTooLarge');
    });

    it('rejects files that do not have a .zip extension or zip MIME type', () => {
      const mockT = (key) => key;
      const pyFile = new File(['print("hello")'], 'bot.py', { type: 'text/x-python' });

      const result = validateBundleFile(pyFile, mockT);
      expect(result.valid).toBe(false);
      expect(result.error).toBe('agents:messages.invalidZipType');
    });

    it('accepts valid zip files within 2 MiB', () => {
      const mockT = (key) => key;
      const validZip = new File(['fake zip content'], 'my_bot.zip', { type: 'application/zip' });
      Object.defineProperty(validZip, 'size', { value: 50 * 1024 }); // 50 KB

      const result = validateBundleFile(validZip, mockT);
      expect(result.valid).toBe(true);
      expect(result.error).toBeNull();
    });

    it('renders error alert when oversized file is dropped or selected', async () => {
      let selected = null;
      let validationError = null;

      const { rerender } = render(
        <BundleDropzone
          bundle={selected}
          onFileSelect={(f) => { selected = f; }}
          validationError={validationError}
          setValidationError={(err) => { validationError = err; }}
        />
      );

      const fileInput = document.querySelector('input[type="file"]');
      const oversizedFile = new File(['content'], 'big.zip', { type: 'application/zip' });
      Object.defineProperty(oversizedFile, 'size', { value: 4 * 1024 * 1024 });

      await act(async () => {
        fireEvent.change(fileInput, { target: { files: [oversizedFile] } });
      });

      rerender(
        <BundleDropzone
          bundle={selected}
          onFileSelect={(f) => { selected = f; }}
          validationError={validationError}
          setValidationError={(err) => { validationError = err; }}
        />
      );

      expect(validationError).toContain('2 MiB');
      expect(screen.getByRole('alert')).toBeDefined();
    });

    it('displays file name, size, and clear button when a valid file is chosen', async () => {
      let selected = null;
      let validationError = null;

      const validFile = new File(['fake zip bytes'], 'hunter.zip', { type: 'application/zip' });
      Object.defineProperty(validFile, 'size', { value: 64 * 1024 });

      render(
        <BundleDropzone
          bundle={validFile}
          onFileSelect={(f) => { selected = f; }}
          validationError={validationError}
          setValidationError={(err) => { validationError = err; }}
        />
      );

      expect(screen.getAllByText(/hunter\.zip/).length).toBeGreaterThanOrEqual(1);
      expect(screen.getByText(/64 KB/)).toBeDefined();

      const clearBtn = screen.getByRole('button', { name: /Quitar/i });
      expect(clearBtn).toBeDefined();

      await act(async () => {
        fireEvent.click(clearBtn);
      });
      expect(selected).toBeNull();
    });
  });

  describe('Starter Kit Card & Reference Template', () => {
    it('provides a direct download link for starfighter-starter.zip', () => {
      render(<StarterKitCard />);

      const downloadLink = screen.getByRole('link', { name: /Descargar.*Starter Kit/i });
      expect(downloadLink).toBeDefined();
      expect(downloadLink.getAttribute('href')).toBe('/starfighter-starter.zip');
      expect(downloadLink.getAttribute('download')).toBe('starfighter-starter.zip');
    });

    it('toggles package specification preview with agentrix.json and bot.py overview', async () => {
      render(<StarterKitCard />);

      expect(screen.queryByText(/bot\.py \(resumen\):/)).toBeNull();

      const toggleBtn = screen.getByRole('button', { name: /Ver especificación/i });
      await act(async () => {
        fireEvent.click(toggleBtn);
      });

      expect(screen.getByText(/"protocol_version": "1.0"/)).toBeDefined();
      expect(screen.getByText(/bot\.py \(resumen\):/)).toBeDefined();

      // Collapse back
      await act(async () => {
        fireEvent.click(screen.getByRole('button', { name: /Ocultar/i }));
      });
      expect(screen.queryByText(/bot\.py \(resumen\):/)).toBeNull();
    });
  });

  describe('Admission Status & Sanitized Feedback', () => {
    it('sanitizes internal server paths and preserves essential file names', () => {
      const rawError = 'bot admission failed: /tmp/agentrix-admission-847291/bot.py: line 42 SyntaxError: unexpected EOF';
      const sanitized = sanitizeAdmissionError(rawError);

      expect(sanitized).not.toContain('/tmp/agentrix-admission-847291');
      expect(sanitized).toContain('bot.py: line 42 SyntaxError: unexpected EOF');
    });

    it('renders progress pulse during validating stage', () => {
      render(<AdmissionStatusBanner stage="validating" />);

      const statusCard = screen.getByRole('status');
      expect(statusCard).toBeDefined();
      expect(screen.getByText(/Validando en sandbox/i)).toBeDefined();
    });

    it('renders structured rejection alert with checklist and reference download', () => {
      render(
        <AdmissionStatusBanner
          stage="rejected"
          errorDetail="bot failed admission tick: invalid Python syntax: line 5 invalid syntax"
        />
      );

      const alert = screen.getByRole('alert');
      expect(alert).toBeDefined();
      expect(screen.getByText(/Fallo en la validación del bot/i)).toBeDefined();
      expect(screen.getByText(/invalid Python syntax/i)).toBeDefined();
      expect(screen.getByText(/Comprobaciones necesarias/i)).toBeDefined();
      expect(screen.getByText(/Descarga el Starter Kit/i)).toBeDefined();
    });
  });

  describe('AgentsPage Asynchronous Admission Integration', () => {
    const mockUser = { id: 'usr-1', username: 'alice', role_id: 'participant' };
    const mockAgent = { id: 'agent-42', name: 'AlphaShip', game_id: 'starfighter', description: 'Test ship' };

    it('renders agent details, template card, dropzone, and version history', async () => {
      vi.spyOn(ApiService, 'listAgents').mockResolvedValue([mockAgent]);
      vi.spyOn(ApiService, 'listContests').mockResolvedValue([]);
      vi.spyOn(ApiService, 'listSubmissions').mockResolvedValue([
        {
          id: 'sub-001',
          agent_id: 'agent-42',
          version: 1,
          language: 'python',
          status: 'ready',
          active: true,
          created_at: '2026-09-20T10:00:00Z',
        },
      ]);

      await act(async () => {
        render(<AgentsPage currentUser={mockUser} />);
      });

      expect(screen.getByRole('heading', { level: 2, name: 'AlphaShip' })).toBeDefined();
      expect(screen.getByText(/Plantilla de inicio para Starfighter/i)).toBeDefined();
      expect(screen.getByText(/Admitir bot de Starfighter/i)).toBeDefined();

      await waitFor(() => {
        expect(screen.getByText('v1')).toBeDefined();
        expect(screen.getByText('Admitido')).toBeDefined();
        expect(screen.getAllByText('Activo').length).toBeGreaterThan(0);
      });
    });

    it('handles synchronous upload success and updates submission history', async () => {
      vi.spyOn(ApiService, 'listAgents').mockResolvedValue([mockAgent]);
      vi.spyOn(ApiService, 'listContests').mockResolvedValue([]);
      vi.spyOn(ApiService, 'listSubmissions').mockResolvedValue([]);
      const uploadSpy = vi.spyOn(ApiService, 'uploadBotBundle').mockResolvedValue({
        id: 'sub-002',
        agent_id: 'agent-42',
        version: 2,
        status: 'ready',
        active: true,
      });

      await act(async () => {
        render(<AgentsPage currentUser={mockUser} />);
      });

      // Select valid file
      const fileInput = document.querySelector('input[type="file"]');
      const validZip = new File(['valid content'], 'bot.zip', { type: 'application/zip' });
      Object.defineProperty(validZip, 'size', { value: 10 * 1024 });

      await act(async () => {
        fireEvent.change(fileInput, { target: { files: [validZip] } });
      });

      const submitBtn = screen.getByRole('button', { name: /Validar y publicar ZIP/i });
      expect(submitBtn.hasAttribute('disabled')).toBe(false);

      await act(async () => {
        fireEvent.click(submitBtn);
      });

      expect(uploadSpy).toHaveBeenCalledWith('agent-42', validZip);
      await waitFor(() => {
        expect(screen.getByText(/Bot Python v2 admitido correctamente/i)).toBeDefined();
      });
    });

    it('handles asynchronous admission via polling until ready state', async () => {
      vi.spyOn(ApiService, 'listAgents').mockResolvedValue([mockAgent]);
      vi.spyOn(ApiService, 'listContests').mockResolvedValue([]);
      vi.spyOn(ApiService, 'listSubmissions').mockResolvedValue([]);

      // 1. Initial upload returns 'validating'
      vi.spyOn(ApiService, 'uploadBotBundle').mockResolvedValue({
        id: 'sub-async-1',
        agent_id: 'agent-42',
        version: 3,
        status: 'validating',
      });

      // 2. Poll returns 'ready'
      const getSubSpy = vi.spyOn(ApiService, 'getSubmission')
        .mockResolvedValueOnce({
          id: 'sub-async-1',
          agent_id: 'agent-42',
          version: 3,
          status: 'ready',
          active: true,
        });

      await act(async () => {
        render(<AgentsPage currentUser={mockUser} />);
      });

      const fileInput = document.querySelector('input[type="file"]');
      const validZip = new File(['content'], 'bot.zip', { type: 'application/zip' });
      Object.defineProperty(validZip, 'size', { value: 12 * 1024 });

      await act(async () => {
        fireEvent.change(fileInput, { target: { files: [validZip] } });
      });

      const submitBtn = screen.getByRole('button', { name: /Validar y publicar ZIP/i });
      await act(async () => {
        fireEvent.click(submitBtn);
      });

      // Polling is called and transitions to ready
      await waitFor(() => {
        expect(getSubSpy).toHaveBeenCalledWith('sub-async-1');
      });

      await waitFor(() => {
        expect(screen.getByText(/Bot Python v3 admitido correctamente/i)).toBeDefined();
      });
    });

    it('renders rejection banner and allows expanding error reason in history when rejected', async () => {
      vi.spyOn(ApiService, 'listAgents').mockResolvedValue([mockAgent]);
      vi.spyOn(ApiService, 'listContests').mockResolvedValue([]);
      vi.spyOn(ApiService, 'listSubmissions').mockResolvedValue([
        {
          id: 'sub-rejected-1',
          agent_id: 'agent-42',
          version: 1,
          language: 'python',
          status: 'rejected',
          active: true,
          error_detail: 'bot failed admission tick: timeout after 100ms',
          created_at: '2026-09-20T11:00:00Z',
        },
      ]);

      await act(async () => {
        render(<AgentsPage currentUser={mockUser} />);
      });

      // Table displays status 'Rechazado'
      await waitFor(() => {
        expect(screen.getByText('Rechazado')).toBeDefined();
      });

      // Click "Ver causa" to expand error details
      const viewReasonBtn = screen.getByRole('button', { name: /Ver causa/i });
      await act(async () => {
        fireEvent.click(viewReasonBtn);
      });

      await waitFor(() => {
        expect(screen.getByText(/timeout after 100ms/i)).toBeDefined();
      });
    });
  });
});
