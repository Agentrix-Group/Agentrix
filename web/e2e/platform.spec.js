import { test, expect } from '@playwright/test';

const PLAYER = { username: process.env.E2E_USERNAME || 'player', password: process.env.E2E_PASSWORD || 'agentrix-demo-player' };

async function login(page) {
  await page.goto('/auth');
  await page.getByLabel('Usuario').fill(PLAYER.username);
  await page.getByLabel('Contraseña').fill(PLAYER.password);
  await page.getByRole('button', { name: 'Ingresar' }).click();
  await expect(page.getByRole('button', { name: /salir/i })).toBeVisible();
}

test('public pages render without horizontal overflow', async ({ page }) => {
  for (const path of ['/', '/contests', '/matches', '/rankings']) {
    await page.goto(path);
    await expect(page.getByRole('navigation')).toBeVisible();
    const overflow = await page.evaluate(() => document.documentElement.scrollWidth - window.innerWidth);
    expect(overflow, `horizontal overflow on ${path}`).toBeLessThanOrEqual(0);
  }
});

test('unknown routes show the 404 page', async ({ page }) => {
  await page.goto('/viewer');
  await expect(page.getByRole('heading', { name: 'Página no encontrada' })).toBeVisible();
});

test('session survives a reload through the refresh cookie and never touches localStorage', async ({ page }) => {
  await login(page);
  await page.reload();
  await expect(page.getByRole('button', { name: /salir/i })).toBeVisible();
  const stored = await page.evaluate(() => Object.keys(window.localStorage).filter((k) => /token|session/i.test(k)));
  expect(stored).toEqual([]);
  await page.getByRole('button', { name: /salir/i }).click();
  await page.reload();
  await expect(page.getByRole('link', { name: 'Ingresar' })).toBeVisible();
});

test('a finished match opens its verified replay (reduced motion: paused)', async ({ page }) => {
  await page.emulateMedia({ reducedMotion: 'reduce' });
  await login(page);
  await page.goto('/matches');
  await page.locator('table a[href^="/matches/"]').first().click();
  await page.getByRole('link', { name: 'Ver replay' }).click();
  const canvas = page.getByRole('img', { name: /tick/i });
  await expect(canvas).toHaveAttribute('aria-label', /tick 0 /);
  await page.getByRole('button', { name: 'Siguiente tick' }).click();
  await expect(canvas).toHaveAttribute('aria-label', /tick 1 /);
});
