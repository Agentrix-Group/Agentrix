# Instrucciones de trabajo para agentes — Agentrix

## Alcance

Este archivo rige todo el repositorio `Agentrix`. El repositorio hermano `agentrix_engine` posee instrucciones propias. Agentrix es el nombre único del producto, repositorio y sistema.

## Precedencia

Cuando dos fuentes difieran, use este orden:

1. La instrucción más reciente y explícita de José Daniel.
2. Código ejecutable, contratos, migraciones, configuración y pruebas que efectivamente pasan, únicamente para describir el estado actual.
3. Decisiones `accepted` en `docs/decisions/` y la arquitectura objetivo aprobada.
4. Documentación activa enlazada desde `docs/index.md`.
5. Material bajo `docs/archive/`, que nunca es normativo.

La existencia de un archivo, stub, prueba o nombre no demuestra que una capacidad esté terminada. Diferencie siempre estado actual y objetivo.

## Mapa del repositorio

El código usa una organización horizontal y poco profunda:

```text
open-api/<feature>.yaml
  -> src/server/<feature>.go
  -> src/service/<feature>.go
  -> src/repository/<feature>.go
  -> src/model/<feature>.go
```

- `server`: HTTP, validación de entrada y presentación.
- `service`: casos de uso, autorización y transiciones.
- `repository`: SQL, persistencia y mapeo; nunca reglas del negocio.
- `model`: entidades, valores y estados; nunca HTTP o SQL.
- `connection`: conexiones, artefactos y cola técnica.
- `auth`: credenciales, sesiones y primitivas de autorización.
- `executor`: reserva de trabajos, bots, motor y coordinación de ticks.
- `engine`: cliente del protocolo Go↔Rust.
- `game`: manifest y registro de juegos de la plataforma.
- `replay`: escritura, validación y compresión del replay.
- `tracer`: eventos operativos y correlación.
- `common`: solo conceptos realmente transversales.

Conserve `repository`: hoy separa SQL de casos de uso. No amplíe su interfaz monolítica; introduzca interfaces pequeñas solo cuando un consumidor real las necesite.

## Invariantes vigentes

- Starfighter es el único juego del MVP actual.
- Go es plano de control y orquestador; Rust es autoridad de simulación y resultado.
- El motor nunca ejecuta bots ni accede a PostgreSQL.
- Los bots son procesos persistentes durante una partida.
- El ciclo comienza en tick 0: `state[0] -> action[0] -> state[1]`.
- Go no interpreta acciones, percepciones ni reglas específicas de Starfighter.
- Percepciones privadas y snapshot público son datos distintos.
- Producción debe fallar cerrado si no existe aislamiento real.
- Un hash detecta divergencia; no prueba por sí solo determinismo multiplataforma.
- Multi-juego, Gym y Rapier no anteceden al cierre de seguridad, ejecución única y despliegue del MVP.

## Comandos de verificación

Sin instalar ni actualizar dependencias:

```bash
GOCACHE=/tmp/agentrix-go-cache go build -mod=readonly ./...
GOCACHE=/tmp/agentrix-go-cache go test -mod=readonly ./...
GOCACHE=/tmp/agentrix-go-cache go vet -mod=readonly ./...
cd web && npm test -- --run
cd web && ./node_modules/.bin/vite build --outDir /tmp/agentrix-web-build --emptyOutDir
cd ../agentrix_engine && CARGO_TARGET_DIR=/tmp/agentrix-engine-target cargo test --locked --offline
git diff --check
```

Declare los fallos reales. La suite Go tiene fallos conocidos en `src/executor` dentro del entorno restringido observado el 2026-09-19; no los silencie ni los describa como éxito.

## Seguridad y operaciones

- No instale paquetes, cambie versiones, levante servicios externos ni ejecute scripts de base de datos sin autorización.
- `db-reset`, `setup_postgres.sh` y scripts SQL son mutaciones, no verificaciones.
- No ejecute código no confiable fuera de un sandbox real.
- No trate el fallback directo a `python3` ni la cola en memoria como comportamiento de producción aceptable.
- No registre secretos, código de bots, percepciones privadas, SQL ni rutas internas completas.
- Conserve cambios ajenos y el historial Git. No haga commit o push sin autorización.

## Archivos generados

`bin/`, `artifacts/`, `web/dist/`, `target/`, coberturas y `__pycache__/` son salidas generadas. No los use como fuente de requisitos ni los añada al control de versiones.

## Definición de terminado

Un cambio está terminado cuando:

- cita la decisión, requisito o caso de uso que implementa;
- preserva las fronteras anteriores;
- actualiza contratos y documentación afectados;
- prueba caminos positivos, negativos y de autorización relevantes;
- ejecuta las verificaciones proporcionales al cambio;
- distingue lo comprobado de lo que sigue incierto;
- no introduce nombres alternativos, abstracciones sin consumidor ni valores variables hardcodeados.

Implemente un solo corte aprobado a la vez. Antes de cambiar comportamiento, explique entidades, estados, autorización, entrada, salida y errores.
