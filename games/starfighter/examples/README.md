# Bots de referencia de Starfighter

Esta carpeta contiene cuatro estrategias de referencia escritas como scripts de
Python 3. Hablan el protocolo JSON Lines `init` / `perception` / `action` /
`end` por `stdin/stdout` y solo usan la biblioteca estándar.

- `bot_random.py`: selecciona una acción aleatoria por tick y sirve como piso
  de comparación.
- `bot_hunter.py`: se aproxima al rival más cercano, gira hacia él y dispara
  cuando lo detecta en el radar.
- `bot_evasive.py`: se aleja y activa el escudo ante peligro; en otro caso
  conserva inercia y recupera energía.
- `bot_ace.py`: pensado para partidas de 2 a 5 naves todos contra todos
  (ADR-0013). Esquiva las balas enemigas que van a pasarle cerca, levanta el
  escudo solo ante un impacto inminente y, si no hay amenaza, ataca: elige el
  blanco más cercano (prefiriendo naves dañadas), apunta al punto de
  intercepción y mantiene una distancia preferida. Cada copia varía levemente
  su estilo según el id de su slot. No ve asteroides: la percepción no los
  incluye. `make db-bootstrap` ejecuta con cinco copias la partida de demo
  `match-star-ffa-demo-001`.

El manifiesto actual de Starfighter usa `bot_hunter.py` o `bot_evasive.py`
como participantes de respaldo cuando una partida programada no tiene una de
las entregas esperadas. Ese comportamiento pertenece al prototipo y no define
por sí mismo la política futura de sustituciones.

El worker mantiene cada proceso durante la partida. Si una respuesta excede
el tiempo por tick o no cumple el protocolo, registra el estado correspondiente
y lo entrega al motor; no inventa una acción válida silenciosamente.

## Límite de seguridad actual

Estos scripts son útiles para desarrollo local, pero su ejecución todavía no
constituye un sandbox de producción. Bubblewrap es opcional y existe una ruta
de ejecución directa con `python3`; tampoco están completos los límites de CPU,
memoria, procesos y salida. Consulte
[`docs/operations/security.md`](../../../docs/operations/security.md).
