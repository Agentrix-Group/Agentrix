# Bots de referencia de Starfighter

Esta carpeta contiene tres estrategias de referencia escritas como scripts de
Python 3. Hablan el protocolo JSON Lines `init` / `perception` / `action` /
`end` por `stdin/stdout` y solo usan la biblioteca estándar.

- `bot_random.py`: selecciona una acción aleatoria por tick y sirve como piso
  de comparación.
- `bot_hunter.py`: se aproxima al rival más cercano, gira hacia él y dispara
  cuando lo detecta en el radar.
- `bot_evasive.py`: se aleja y activa el escudo ante peligro; en otro caso
  conserva inercia y recupera energía.

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
