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
