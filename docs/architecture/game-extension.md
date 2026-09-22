# Extensión de juegos

## Estado actual

**No implementado como plataforma extensible.** Starfighter es el único juego registrable y ejecutable. El registry, el validador, el servicio y el executor contienen guardas explícitas para `starfighter` y de 2 a 5 jugadores (ADR-0013).

Esto es correcto para el MVP, pero no debe confundirse con la arquitectura objetivo.

## Fronteras objetivo

Cada juego aportará:

- engine autoritativo;
- configuración versionada;
- schemas de acción, percepción y snapshot público;
- bots de referencia;
- renderer;
- reglas y política de resultado.

La plataforma aportará:

- protocolo común y ciclo de proceso;
- reserva de trabajos y sandbox de bots;
- transporte opaco de acciones y percepciones;
- almacenamiento y sellado de evidencia;
- publicación de resultados y replay.

Agregar un juego no debe exigir modificar el bucle central del executor. Las diferencias del juego se resuelven dentro de su engine, schemas y renderer.

## Orden de extracción

La generalización ocurre después de certificar el MVP de Starfighter:

1. Extraer una API de simulación estable.
2. Separar `sim-core` de Bevy, IPC y renderer sin cambiar reglas.
3. Hacer que el engine Starfighter consuma ese mismo core.
4. Añadir un segundo juego discreto y sin física.
5. Confirmar que el executor no requiere una rama por juego.

El segundo juego es la prueba de la abstracción. Antes de él, no se crean plugins, carpetas ni traits sin consumidor real.

## Compatibilidad

Una partida debe fijar, como mínimo, `game_id`, `game_version`, digest del engine, versión del protocolo, hash de configuración y seed. Un cambio de componentes crea una nueva versión publicable; no altera partidas históricas.
