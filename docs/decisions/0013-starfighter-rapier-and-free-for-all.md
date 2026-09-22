# ADR-0013: Starfighter sobre Rapier y partidas todos contra todos

## Estado

accepted

## Fecha

2026-09-22

## Contexto

[ADR-0002](0002-starfighter-avian-mvp.md) mantuvo Avian2D durante el MVP y
condicionó Rapier a sim-core, segundo juego, Gym y una comparación en x86_64
y ARM64. Con el MVP cerrado ([ADR-0008](0008-mvp-canonical-certification-and-baseline-closure.md)),
se decide adelantar el cambio de backend físico y ampliar Starfighter de un
duelo 1 vs 1 a partidas de hasta cinco agentes, todos contra todos.

La auditoría del motor (`agentrix_engine@486e22b`) encontró:

- `bevy_rapier2d` 0.36, el único plugin compatible con Bevy 0.19, depende de
  `rapier2d =0.35.0-glamx0.2`, una pre-release; la versión estable es
  `rapier2d` 0.35.3.
- El hash autoritativo incluye estado interno de contactos de Avian, por lo
  que cualquier cambio de backend invalida los vectores dorados de
  `starfighter 0.3.0-core.1`.
- Bugs independientes del backend: el allocator de entidades nunca libera
  ids (tras `max_entities` las naves dejan de disparar), una colisión puede
  aplicarse dos veces en un tick, las balas propias empujan a su emisor y el
  spawn circular deja naves fuera de la arena con más de dos jugadores.
- La plataforma Go asume exactamente dos jugadores en validación de manifest,
  replay, spec de ejecución y política de puntaje.

## Decisión

- Starfighter usa `rapier2d` 0.35.3 directamente, con `enhanced-determinism`.
  Bevy ECS se conserva para reglas y estado del juego; no se usa
  `bevy_rapier2d`.
- El motor avanza la física exactamente un paso por `advance`, con `dt`
  derivado de `TickRate`, sin acumulador de `Time<Fixed>`.
- Las balas no son cuerpos rígidos: su impacto se resuelve con un barrido de
  forma sobre la trayectoria del tick.
- Las funciones trascendentes de las reglas usan `libm`, no `std`.
- Se mantienen Bevy 0.19.1 y rustc 1.98.1, las últimas versiones estables;
  Bevy 0.20 sigue en release candidate.
- Starfighter admite de 2 a 5 jugadores por partida, todos contra todos y sin
  equipos.
- El resultado de partida es una clasificación por orden de eliminación; las
  eliminaciones del mismo tick empatan y, al agotar el límite de ticks,
  desempata la salud restante. `Destroyed` registra al autor de la baja.
- Los puntos de spawn se reparten sobre una elipse al 60 % de la arena,
  orientados al centro, con asignación de slot permutada por la semilla.
- El cambio publica una versión nueva del juego, `starfighter 0.4.0`, con
  schemas y vectores dorados propios. `0.3.0-core.1` y sus replays no se
  reinterpretan.
- La política de puntaje de torneo pasa de victoria/empate/derrota a puntos
  por posición para partidas de más de dos jugadores. La tabla de puntos se
  fija en el corte de plataforma (F6) y enmienda la política estándar de
  [ADR-0012](0012-deterministic-rankings-and-recalculation-policy.md); sus
  criterios de desempate y snapshots se conservan.

## Criterios de aceptación

| Fase | Criterio |
| --- | --- |
| F0 | Este ADR y la línea base de throughput con Avian guardada en `agentrix_engine/benches/baseline/avian2d-f0.md` |
| F1 | Bugs de auditoría corregidos con Avian; suite completa en verde y partida de 10 000 ticks sin agotar entidades |
| F2 | Dependencias actualizadas; vectores dorados de `0.3.0-core.1` idénticos |
| F3 | Rapier con 2 jugadores: suite en verde, 0 atravesamientos de bala en 1500–3000 u/s y a 10 000 u/s, cadenas de commitments idénticas en procesos limpios y throughput comparado con F0 |
| F4 | 2 a 5 jugadores: ninguna nave fuera de la arena y victoria por slot dentro de 3σ de 1/N en partidas aleatorias |
| F5 | `starfighter 0.4.0` con vectores dorados reproducidos por el runner de conformidad |
| F6 | Plataforma Go/web: suite con `-race` en verde y partida E2E real de cinco bots con replay sellado |
| F7 | Opcional: hashes idénticos entre x86_64 y ARM64; si no se ejecuta, no se afirma determinismo entre plataformas |

## Consecuencias

- Hasta completar F6, la plataforma sigue ofreciendo solo `0.3.0-core.1`
  1 vs 1; el engine nuevo no se publica en `engine.lock` antes de ese corte.
- La capability anunciada cambia de `avian2d` a `rapier2d` y debe cambiar en
  el mismo corte en el engine, el cliente Go y `cmd/fake-engine`.
- El build de `agentrix_engine` necesita descargar dependencias nuevas una
  vez; después se vuelve a verificar con `--locked --offline`.
- `enhanced-determinism` no convierte en determinista el sistema completo:
  la afirmación entre arquitecturas sigue exigiendo la evidencia de F7.
- Gym, sim-core y segundo juego mantienen su orden; este ADR solo adelanta el
  backend físico y el formato de partida.

## Sustituye

Sustituye en [ADR-0002](0002-starfighter-avian-mvp.md) la permanencia de
Avian2D, la exclusión de Rapier como dependencia y el duelo de dos jugadores.
Conserva de esa decisión la separación Go/Rust, Starfighter como único juego
oficial y el objetivo de 60 Hz exactos.
