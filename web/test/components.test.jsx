import React, { useState } from 'react';
import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent, act } from '@testing-library/react';
import { Modal } from '../src/components/Modal.jsx';
import { StatusBadge } from '../src/components/StatusBadge.jsx';
import { RunStepper } from '../src/components/RunStepper.jsx';
import { validateBundleFile } from '../src/components/BundleDropzone.jsx';

function Harness({ onClose = () => {} }) {
  const [open, setOpen] = useState(false);
  return (
    <>
      <button type="button" onClick={() => setOpen(true)}>open</button>
      {open && (
        <Modal title="Dialog" onClose={() => { onClose(); setOpen(false); }}>
          <input aria-label="first" />
          <button type="button">last</button>
        </Modal>
      )}
    </>
  );
}

describe('Modal accessibility', () => {
  it('focuses inside, traps Tab, closes with Escape and restores focus and scroll', () => {
    const onClose = vi.fn();
    render(<Harness onClose={onClose} />);
    const opener = screen.getByText('open');
    opener.focus();
    fireEvent.click(opener);
    const dialog = screen.getByRole('dialog');
    expect(dialog.getAttribute('aria-modal')).toBe('true');
    expect(document.body.style.overflow).toBe('hidden');
    const close = screen.getByLabelText(/cerrar|close/i);
    expect(document.activeElement).toBe(close);
    const last = screen.getByText('last');
    last.focus();
    fireEvent.keyDown(document, { key: 'Tab' });
    expect(document.activeElement).toBe(close);
    close.focus();
    fireEvent.keyDown(document, { key: 'Tab', shiftKey: true });
    expect(document.activeElement).toBe(last);
    act(() => { fireEvent.keyDown(document, { key: 'Escape' }); });
    expect(onClose).toHaveBeenCalled();
    expect(screen.queryByRole('dialog')).toBeNull();
    expect(document.activeElement).toBe(opener);
    expect(document.body.style.overflow).toBe('');
  });
});

describe('status components', () => {
  it('renders canonical states only from the value received', () => {
    render(<StatusBadge state="registration_open" />);
    expect(screen.getByText(/inscripción abierta|registration open/i)).toBeTruthy();
  });
  it('stepper marks failures without claiming success', () => {
    const { container } = render(<RunStepper state="failed" />);
    expect(container.querySelector('.step-error')).toBeTruthy();
    expect(container.querySelectorAll('.step-done').length).toBeLessThan(3);
  });
  it('validates bundle type and size on the client', () => {
    expect(validateBundleFile({ name: 'bot.py', size: 10 })).toBe('invalidZipType');
    expect(validateBundleFile({ name: 'bot.zip', size: 3 * 1024 * 1024 })).toBe('fileTooLarge');
    expect(validateBundleFile({ name: 'bot.zip', size: 100 })).toBeNull();
  });
});
