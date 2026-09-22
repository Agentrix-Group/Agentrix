import { describe, expect, it } from 'vitest';
import {
  canvasHeading,
  fighterSlot,
  PLAYER_COLORS,
  slotColor,
} from '../src/renderers/starfighter/canvasRenderer.js';

// Dirección en pantalla (y hacia abajo) de un ángulo de canvas.
const screenDirection = (angle) => ({ x: Math.cos(angle), y: Math.sin(angle) });

describe('Starfighter canvas renderer (ADR-0013)', () => {
  it('points the hull where the engine says the nose is', () => {
    // θ = 0: la nariz apunta a +Y del mundo, que en pantalla es hacia arriba.
    const up = screenDirection(canvasHeading(0));
    expect(up.x).toBeCloseTo(0);
    expect(up.y).toBeCloseTo(-1);
    // θ = π/2: nariz en (−sin θ, cos θ) = (−1, 0), hacia la izquierda.
    const left = screenDirection(canvasHeading(Math.PI / 2));
    expect(left.x).toBeCloseTo(-1);
    expect(left.y).toBeCloseTo(0);
  });

  it('gives each of five slots its own color', () => {
    const participants = ['a', 'b', 'c', 'd', 'e'];
    const colors = participants.map((slot) => slotColor(slot, participants));
    expect(new Set(colors).size).toBe(5);
    expect(colors).toEqual(PLAYER_COLORS);
  });

  it('keeps a slot color when another ship is destroyed', () => {
    const participants = ['a', 'b', 'c'];
    const before = [{ slot: 'a' }, { slot: 'b' }, { slot: 'c' }];
    const after = [{ slot: 'a' }, { slot: 'c' }];
    const colorOfC = (fighters) => {
      const index = fighters.findIndex((f) => fighterSlot(f) === 'c');
      return slotColor(fighterSlot(fighters[index]), participants, index);
    };
    expect(colorOfC(after)).toBe(colorOfC(before));
  });

  it('labels fighters by the slot of the public snapshot', () => {
    expect(fighterSlot({ slot: 'alpha', entityId: 1 })).toBe('alpha');
  });
});
