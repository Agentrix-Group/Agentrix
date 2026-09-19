# ADR-0007: Separación de roles de API y Worker y contenedores reproducibles

## Estado

accepted

## Fecha

2026-09-19

## Contexto

El punto de entrada anterior (`main.go`) ejecutaba de manera monolítica el servidor HTTP de la API, el validador de bots, la cola en memoria/PostgreSQL y el worker pool de simulación en un mismo proceso. Esto presentaba los siguientes riesgos:

1. **Superficie de ataque compartida:** El proceso que atendía peticiones HTTP públicas de internet era el mismo que ejecutaba subprocesos de Python y el motor de simulación.
2. **Escalabilidad desacoplada imposible:** No era posible escalar horizontalmente los workers de ejecución de partidas sin duplicar instancias del servidor HTTP.
3. **Falta de contención en producción (fail-closed):** El worker no verificaba de forma taxativa en arranque que estuvieran disponibles PostgreSQL (la cola autoritativa) y un sandbox real de aislamiento (Bubblewrap o Podman), arriesgando la ejecución no confiable de código fuera de un sandbox en entornos productivos.
4. **Imágenes de contenedor no segregadas:** El `Dockerfile` histórico compilaba un único binario sobre distroless sin Python ni Bubblewrap, impidiendo que el worker funcionara en contenedor.

## Decisión

1. **Separación de binarios y roles de ejecución:**
   - `cmd/api`: Binario exclusivo de la API HTTP. Maneja autenticación, gestión de participantes y concursos, recepción y validación sintáctica de bundles, y encolado de trabajos en PostgreSQL. **No ejecuta el motor de simulación ni procesos de bots de participantes.**
   - `cmd/worker`: Binario exclusivo del worker de partidas. Realiza dequeue transaccional con lease, invoca el motor oficial Starfighter y los bots dentro de `BotRuntime` (Bubblewrap o Podman), realiza el commit transaccional cercado (`CommitMatchResult`) y publica los replays. **No expone ningún puerto HTTP.**
   - `main.go`: Punto de entrada universal que permite seleccionar el rol mediante la variable de entorno `AGENTRIX_ROLE=api|worker|all` o argumento cli (`agentrix api`, `agentrix worker`), manteniendo compatibilidad completa en desarrollo local (`all`).

2. **Políticas fail-closed en arranque de producción:**
   - Si `MODE != "dev"`, el worker verifica estrictamente:
     a. Conexión activa a PostgreSQL (`ErrProductionDatabaseRequired`).
     b. Disponibilidad de Bubblewrap o Podman (`ErrProductionSandboxUnavailable`).
     Ante la ausencia de cualquiera de ellos, el worker aborta inmediatamente en el arranque con error fatal.

3. **Empaquetado y contenedores reproducibles:**
   - `Dockerfile.api`: Contenedor mínimo para el servidor HTTP.
   - `Dockerfile.worker`: Contenedor basado en Debian que incluye `bubblewrap`, `python3`, `zstd`, el binario oficial del motor `starfighter-engine` y el binario `agentrix-worker`.
   - `docker-compose.yml`: Orquestación local reproducible con servicios `postgres`, `api` y `worker`.

## Consecuencias

- Aislamiento completo de la superficie de red de la ejecución de código no confiable.
- Escalado independiente de la capacidad de procesamiento de partidas y del tráfico web.
- Cumplimiento de las invariantes de `AGENTS.md` (Go orquestador, motor headless, aislamiento estricto en producción).

## Evidencia relacionada

- `cmd/api/main.go` y `cmd/worker/main.go`.
- `Dockerfile.api`, `Dockerfile.worker` y `docker-compose.yml`.
- `src/executor/worker.go` y `src/config/config.go`.
