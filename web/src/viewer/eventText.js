// Traduce los eventos públicos de Starfighter a frases legibles para el
// visor de repeticiones. El replay guarda cada evento como texto JSON
// (`{"type":"hit","player_id":1,...}`); nunca se muestra ese texto crudo.

/** Devuelve el evento como objeto, o null si no se puede interpretar. */
export function parseEvent(raw) {
  if (raw && typeof raw === 'object') return raw;
  if (typeof raw !== 'string') return null;
  try {
    const parsed = JSON.parse(raw);
    return parsed && typeof parsed === 'object' ? parsed : null;
  } catch {
    return null;
  }
}

/** Nombre visible de un slot: su identificador en el replay. */
export function playerName(playerId, participants = [], t) {
  if (playerId === null || playerId === undefined) return null;
  return participants[playerId] || t('viewer:events.unknownPlayer', { number: Number(playerId) + 1 });
}

// El motor describe el origen del daño en inglés ("an asteroid",
// "P1's bullet", "disqualified"); se traduce con el nombre del tirador.
function describeSource(source, participants, t) {
  if (source === 'an asteroid') return t('viewer:events.sources.asteroid');
  if (source === 'disqualified') return t('viewer:events.sources.disqualified');
  const bullet = /^P(\d+)'s bullet$/.exec(source || '');
  if (bullet) {
    return t('viewer:events.sources.bulletOf', { player: playerName(Number(bullet[1]), participants, t) });
  }
  if (source === 'a bullet') return t('viewer:events.sources.bullet');
  return source || t('viewer:events.sources.unknown');
}

function formatDamage(damage) {
  const value = Number(damage);
  if (!Number.isFinite(value)) return '?';
  return Number.isInteger(value) ? String(value) : value.toFixed(1);
}

/**
 * Describe un evento. `kind` sirve para el estilo (fired, hit, destroyed,
 * ended, other) y `text` es la frase traducida.
 */
export function describeEvent(raw, participants, t) {
  const event = parseEvent(raw);
  if (!event) {
    return { kind: 'other', text: t('viewer:events.unknown') };
  }
  const player = playerName(event.player_id, participants, t);
  switch (event.type) {
    case 'fired':
      return { kind: 'fired', text: t('viewer:events.fired', { player }) };
    case 'hit': {
      const text = t('viewer:events.hit', {
        player,
        damage: formatDamage(event.damage),
        source: describeSource(event.source, participants, t),
      });
      return { kind: 'hit', text: event.shielded ? `${text} ${t('viewer:events.shielded')}` : text };
    }
    case 'destroyed':
      if (event.source === 'disqualified') {
        return { kind: 'destroyed', text: t('viewer:events.disqualified', { player }) };
      }
      if (event.killer !== null && event.killer !== undefined) {
        return {
          kind: 'destroyed',
          text: t('viewer:events.destroyedBy', { killer: playerName(event.killer, participants, t), player }),
        };
      }
      return {
        kind: 'destroyed',
        text: t('viewer:events.destroyed', { player, source: describeSource(event.source, participants, t) }),
      };
    case 'match_ended':
      return event.winner === null || event.winner === undefined
        ? { kind: 'ended', text: t('viewer:events.endedNoWinner') }
        : { kind: 'ended', text: t('viewer:events.endedWinner', { player: playerName(event.winner, participants, t) }) };
    default:
      return { kind: 'other', text: t('viewer:events.unknown') };
  }
}

/** Motivo de fin legible ("eliminated" -> "Por eliminación"). */
export function describeReason(reason, t) {
  const known = ['eliminated', 'timeout', 'score_limit'];
  return known.includes(reason) ? t(`viewer:reasons.${reason}`) : reason || '';
}
