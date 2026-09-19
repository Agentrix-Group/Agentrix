# Alcance del MVP

## Objetivo

El MVP debe demostrar que una persona autorizada puede presentar un bot Python, validarlo, ejecutar una partida segura de Starfighter contra otro bot y consultar un resultado y replay verificables.

Starfighter es el único juego oficial del MVP. Esto es una restricción deliberada, no evidencia de una plataforma multi-juego terminada.

## Implementado

- API REST en Go y aplicación React.
- Motor Starfighter headless en Rust con Bevy y Avian2D.
- Protocolo `agentrix-engine/1` por JSON Lines sobre `stdin/stdout`.
- Proceso Python persistente por bot durante una partida.
- Paquete ZIP con exactamente `agentrix.json` y `bot.py`.
- Validación de estructura, sintaxis Python y una interacción de admisión.
- Tick 0 estricto: `state[0] -> action[0] -> state[1]`.
- Acciones y percepciones específicas del juego opacas para Go.
- Percepciones privadas y snapshots públicos separados.
- Replay NDJSON progresivo y visor Canvas 2D posterior a la partida.
- Reserva de trabajos PostgreSQL mediante `FOR UPDATE SKIP LOCKED`.

## Parcial

- **Aislamiento:** Bubblewrap deshabilita red y monta el host como solo lectura, pero hay fallback directo a Python y faltan filesystem mínimo, entorno limpio y límites completos.
- **Cola:** recupera una reserva vencida, pero carece de heartbeat, fencing e idempotencia de resultados.
- **Replay:** valida secuencia y hash final, pero no está sellado como artefacto inmutable ni registra todos los identificadores de reproducción.
- **Configuración:** el motor consume seed, jugadores, máximo de ticks, timestep y radar; otros valores del manifest continúan hardcodeados.
- **Autorización:** existe JWT y permisos, pero no el modelo completo de capacidades con alcance.
- **Resultados:** se persisten filas y se actualiza la partida, pero no existe commit transaccional cercado del resultado y replay.
- **Pruebas:** Go compila y pasa `vet`; la suite completa falla actualmente en pruebas de procesos de bots dentro del entorno restringido.

## No implementado

- Fallo cerrado en producción cuando falta el runtime de aislamiento.
- API y worker desplegables por separado.
- Leases renovables y fencing token.
- Sello inmutable y commit idempotente de replay/resultado.
- Imagen de producción con motor, Python, sandbox y frontend.
- Certificación de seguridad y recorrido end-to-end reproducible.
- Segundo juego, sim-core compartido, Gym vectorizado y benchmark Avian/Rapier.
- Equipos, inscripción real, arbitraje completo, final en vivo y WebSocket público.

## Exclusiones actuales

- Otros juegos o lenguajes de agentes.
- Reimplementación de reglas o físicas en Python.
- Rapier como dependencia o decisión cerrada.
- Visión computacional.
- Escalamiento horizontal automático.
- Soporte multi-organización.

## Criterio para certificar el MVP

El MVP no se considera cerrado hasta demostrar, al menos:

1. aislamiento rootless real sin fallback permisivo;
2. límites de CPU, memoria, PIDs, tiempo y output;
3. recuperación de un trabajo sin doble resultado mediante lease renovable y fencing;
4. replay y resultado sellados con commit idempotente;
5. imagen y despliegue reproducibles con API y worker separables;
6. recorrido automatizado completo con los mismos artefactos que producción;
7. documentación ejecutable por una persona nueva.
