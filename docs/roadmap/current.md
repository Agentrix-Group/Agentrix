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
| Baseline | En curso | Build, suites y recorrido real reproducibles; la suite Go aún falla en executor |
| Sandbox | Pendiente | Rootless, mínimo, sin red, límites completos y fail-closed |
| Lease/fencing | Pendiente | Renovación, token monotónico y prueba de recuperación sin duplicado |
| Protocolo/configuración | Parcial | Validación bilateral y 60 Hz exactos; retirar hardcodes configurables |
| Commit/replay | Parcial | Sello inmutable y commit idempotente cercado |
| Despliegue | Pendiente | API/worker separados e imágenes completas reproducibles |
| Certificación MVP | Pendiente | Seguridad, recuperación y E2E desde entorno limpio |
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

Cerrar la baseline de procesos de bots: explicar y corregir los fallos actuales de `src/executor`, ejecutar la suite completa en un entorno compatible y separar claramente fallo del código de limitación del runner. Ese corte no autoriza aún cambios al sandbox de producción.
