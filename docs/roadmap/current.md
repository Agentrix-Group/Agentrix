# Roadmap vigente

Este es el único roadmap normativo de Agentrix.

```text
baseline -> sandbox -> lease/fencing -> protocolo/configuración
         -> commit/replay -> despliegue -> certificación MVP
         -> sim-core -> segundo juego -> Gym -> decisión Rapier
```

## Estado

| Etapa | Estado | Salida necesaria |
| --- | --- | --- |
| Baseline | Implementado | Suites Go/Rust/Web 100% pasando con -race; tags de baseline; CI workflow; benchmark 100 partidas medido |
| Sandbox | Implementado | Interfaz BotRuntime con Bubblewrap mínimo/tmpfs/no-net, Podman rootless, entorno limpio sin secretos y fail-closed |
| Lease/fencing | Implementado | Fencing token monotónico, lease_until, heartbeat de renovación y cancelación inmediata en pérdida de lease |
| Protocolo/configuración | Implementado | Validación bilateral de sobres, máquina de estados formal, 60 Hz exactos y StarfighterConfig autoritativo |
| Commit/replay | Implementado | Sello inmutable, commit idempotente cercado con run_id, tabla match_runs y publicación atómica |
| Despliegue | Implementado | API/worker separados, Dockerfile.api/worker, docker-compose y fail-closed en producción |
| Certificación MVP | Implementado | Suite E2E canónica, rechazo de zombis por fencing token, replay atómico verificado bit a bit y ADR-0008 |
| sim-core | No iniciado | Mismas reglas fuera de IPC/renderer |
| Segundo juego | No iniciado | Juego discreto sin modificar el loop central |
| Gym | No iniciado | API vectorizada sobre el mismo sim-core |
| Decisión Rapier | No iniciada | Benchmark y conformidad x86_64/ARM64 |

## Reglas de avance

- No se inicia una etapa que dependa de una garantía anterior abierta.
- Un archivo o test existente no cierra una etapa: la salida debe ejecutarse y conservar evidencia.
- Un fallo de seguridad, doble ejecución o replay inconsistente reabre la etapa correspondiente.
- Multi-juego, Gym y Rapier nunca se adelantan para embellecer la arquitectura del MVP.

## Próximo corte

Post-MVP (sim-core): extracción de la lógica de simulación fuera de Bevy/IPC para inferencia rápida y soporte Gym sin alterar las garantías transaccionales del MVP cerrado.

## Roadmap paralelo de Frontend y QA (ADR-0009)

Para garantizar la entrega integral del MVP de usuario sin desincronización entre backend y UI, rige el siguiente roadmap paralelo de vertical slices:

```text
fundación QA (FQ-0) -> sesión (FQ-1) -> admisión (FQ-2) -> match/run (FQ-3)
                     -> replay (FQ-4) -> concurso/ranking (FQ-5) -> release (FQ-6)
```

| Sprint | Enfoque Frontend | Enfoque QA | Gate | Estado |
| --- | --- | --- | --- | --- |
| FQ-0 | Routing nativo, Error Boundary, cliente HTTP tipado, saneamiento de errores y `code_path` | Cobertura base, mock de red, tests de router/estados | Login, home y navegación estables en suite unitaria y de integración | Completado |
| FQ-1 | Session provider, renovación de tokens, rutas protegidas y capabilities | 401/403, sesión caducada, refresh concurrente | Flujo de autenticación y autorización robusto | Completado |
| FQ-2 | Upload con progreso, estados de admisión (`pending_validation`, `validating`, `ready`, `rejected`) | Validación de bundles ZIP, timeouts, causas de rechazo sanitizadas | Admisión asíncrona visible en UI | Completado |
| FQ-3 | Asistente de creación de partida, selección de submissions y seguimiento de runs | Estados terminales de match, idempotencia, fencing | Ejecución de partida de extremo a extremo | Completado |
| FQ-4 | Streaming progresivo, Web Worker, verificación SHA-256 y renderizado adaptativo | Screenshots dorados, playback 60 Hz, integridad bit a bit | Replay Canvas escalable y verificable | Completado |
| FQ-5 | Detalle de concurso, inscripción guiada, rankings y filtros | Paginación, zonas horarias, paridad ES/EN | Experiencia de torneo completa | Completado |
| FQ-6 | Panel de operaciones, observabilidad sin secretos, accesibilidad WCAG AA | E2E en exploradores reales, Lighthouse, auditoría de seguridad | Release candidate del MVP certificado | Pendiente |
