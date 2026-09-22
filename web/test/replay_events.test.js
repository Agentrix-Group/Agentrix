import { beforeAll, describe, expect, it } from 'vitest';
import i18n from '../src/i18n/index.js';
import { describeEvent, describeReason } from '../src/viewer/eventText.js';

const participants = ['sub-star-ace-1', 'sub-star-ace-2', 'sub-star-ace-3'];

describe('Replay viewer events in plain language', () => {
  let t;
  beforeAll(async () => {
    await i18n.changeLanguage('es');
    t = i18n.getFixedT('es');
  });

  const text = (raw) => describeEvent(raw, participants, t).text;

  it('describes the events of the reported screenshot without raw JSON', () => {
    expect(text('{"damage":100,"player_id":1,"shielded":false,"source":"an asteroid","type":"hit"}'))
      .toBe('sub-star-ace-2 recibió 100 de daño de un asteroide');
    expect(text('{"killer":null,"player_id":1,"source":"an asteroid","type":"destroyed"}'))
      .toBe('sub-star-ace-2 fue destruido por un asteroide');
    expect(text('{"type":"match_ended","winner":2}')).toBe('Fin de la partida: gana sub-star-ace-3');
  });

  it('names the shooter of bullets and kills', () => {
    expect(text({ type: 'hit', player_id: 0, damage: 7.5, shielded: true, source: "P2's bullet" }))
      .toBe('sub-star-ace-1 recibió 7.5 de daño de una bala de sub-star-ace-3 (con escudo activo)');
    expect(text({ type: 'destroyed', player_id: 0, killer: 2, source: "P2's bullet" }))
      .toBe('sub-star-ace-3 derribó a sub-star-ace-1');
    expect(text({ type: 'fired', player_id: 1 })).toBe('sub-star-ace-2 disparó');
  });

  it('covers disqualification, draws and unknown input', () => {
    expect(text({ type: 'destroyed', player_id: 1, killer: null, source: 'disqualified' }))
      .toBe('sub-star-ace-2 fue descalificado y sale de la partida');
    expect(text({ type: 'match_ended', winner: null })).toBe('Fin de la partida sin ganador');
    expect(text({ type: 'fired', player_id: 7 })).toBe('Jugador 8 disparó');
    expect(text('not json')).toBe('Evento no reconocido');
    expect(text('{"type":"teleport"}')).toBe('Evento no reconocido');
  });

  it('gives each event a kind for styling', () => {
    expect(describeEvent({ type: 'destroyed', player_id: 0, killer: 1 }, participants, t).kind).toBe('destroyed');
    expect(describeEvent({ type: 'match_ended', winner: 1 }, participants, t).kind).toBe('ended');
  });

  it('translates the finish reason', () => {
    expect(describeReason('eliminated', t)).toBe('Por eliminación');
    expect(describeReason('score_limit', t)).toBe('Se agotó el tiempo de partida');
    expect(describeReason('timeout', i18n.getFixedT('en'))).toBe('A bot ran out of time');
  });
});
