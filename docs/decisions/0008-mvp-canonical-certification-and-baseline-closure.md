# ADR-0008: Certificación canónica del MVP y cierre formal del baseline

## Estado

accepted

## Fecha

2026-09-19

## Contexto

El "Plan maestro de desarrollo de Agentrix" se inició sobre la base auditada `Agentrix@c088cf7` y `agentrix_engine@5f130fe` con el objetivo de convertir Agentrix en una plataforma determinista, segura, transaccional y completamente verificable antes de abordar abstracciones multi-juego, Gym o Rapier.

A lo largo de las Fases 1 a 6 se implementaron y verificaron todas las garantías críticas de la plataforma:

1. **Aislamiento y sandbox (Fase 1):** Abstracción `BotRuntime` con Bubblewrap (`--unshare-net`, `--tmpfs`, `--ro-bind`), Podman rootless y política fail-closed en producción. Validación de bots en admisión sin ejecutar código no confiable en el host.
2. **Lease y fencing token (Fase 2):** Fencing tokens monotónicos con `SELECT FOR UPDATE SKIP LOCKED` en PostgreSQL, heartbeat periódico de renovación de lease y cancelación inmediata del contexto de ejecución ante la pérdida de lease.
3. **Protocolo y simulación autoritativa (Fase 3):** Validación bilateral monotónica de secuencia (`ERR_INVALID_SEQUENCE`), versión y `matchId` en Go y Rust. Máquina de estados formal (`Ready` -> `Initialized` -> `Running` -> `Finished` -> `Shutdown`). Paso de tiempo exacto a 60 Hz (`1.0 / tick_hz` en Bevy) y configuración tipada `StarfighterConfig` sin constantes hardcodeadas.
4. **Persistencia transaccional y replay atómico (Fase 4):** Tabla `match_runs` con restricción única parcial sobre ejecuciones `committed`. Transacción atómica `CommitMatchResult` que valida el fencing token antes de persistir resultados. Replay escrito en ruta temporal, sellado con SHA-256 y tamaño en bytes, comprimido con Zstandard y publicado atómicamente solo tras confirmación del commit en base de datos. Descarte garantizado de escrituras y replays de workers zombi.
5. **Despliegue y roles segregados (Fase 5):** Binarios dedicados `cmd/api` (solo HTTP y cola) y `cmd/worker` (solo sandbox, motor y transacciones de commit). `Dockerfile.api`, `Dockerfile.worker` y `docker-compose.yml` reproducibles. Validación fail-closed en arranque de producción.

## Decisión

Declarar oficialmente certificada y cerrada la fase de MVP de Agentrix.

Las garantías operativas y arquitectónicas quedan formalmente selladas:
- Go es exclusivamente plano de control y orquestador; Rust es la única autoridad de simulación y resultado.
- Starfighter es el juego oficial canónico del MVP.
- Las partidas se ejecutan bajo aislamiento estricto y commit cercado.
- Los replays publicados son inmutables, completos y reproducibles bit a bit.

## Evidencia verificada

1. **Suites de pruebas automatizadas pasando al 100% con `-race`:**
   - Go: todas las pruebas unitarias y de integración de `src/...` pasan con detector de carreras activado.
   - Rust: todos los unit tests, integration tests y property-based tests en `agentrix_engine` pasan.
   - Web: 23 pruebas de interfaz y traducción en Vitest pasando; build de producción de Vite exitoso.
2. **Pruebas de concurrencia, fencing y rechazo de zombis:**
   - `TestZombieWorkerCommitRejected`: demuestra que un worker zombi con lease expirado no puede commitear resultados ni publicar replays.
   - `TestExecutor_CommitFailureDiscardsReplay`: demuestra la purga inmediata de artefactos temporales ante fallos transaccionales.
3. **Prueba E2E canónica de certificación:**
   - `TestMVPCanonicalCertification`: ejecución completa real de Starfighter engine + bots de Python + sandbox + commit transaccional + verificación bit a bit del digest SHA-256 del replay publicado.
4. **Métricas de rendimiento en 100 partidas consecutivas:**
   - `TestStarfighterIntegration_100MatchesDurationAndMemoryMetrics`: 100 partidas ejecutadas con latencia p50 de ~107ms y consumo de memoria estable.

## Próximos pasos (Post-MVP)

Con el MVP certificado y formalmente cerrado, quedan expeditas las etapas futuras del roadmap:
- Extracción del sim-core (motor desacoplado de dependencias Bevy de renderizado para máxima velocidad de inferencia).
- Segundo juego de la plataforma.
- Soporte para entornos vectorizados Gym / Gymnasium sobre el sim-core.
- Evaluación comparativa de Rapier vs Avian2D para determinismo multiplataforma certificado.
