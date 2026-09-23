# Documentación de Agentrix

Este índice es la entrada canónica a la documentación. Un documento que no aparezca aquí como normativo no debe usarse para dirigir desarrollo nuevo.

## Fuentes normativas

| Documento | Propósito | Autoridad |
| --- | --- | --- |
| [Visión](product/vision.md) | Producto, principios y dirección de largo plazo | Objetivo aprobado |
| [Alcance del MVP](product/mvp-scope.md) | Qué existe, qué falta y qué se excluye | Estado actual y alcance |
| [Arquitectura](architecture/overview.md) | Componentes, fronteras y estado | Arquitectura vigente |
| [Modelo de ejecución](architecture/execution-model.md) | Ticks, bots, sandbox y trabajos | Arquitectura vigente |
| [Protocolo del motor](architecture/engine-protocol.md) | Frontera Go↔Rust | Contrato y estado |
| [Extensión de juegos](architecture/game-extension.md) | Camino desde Starfighter a más juegos | Objetivo posterior al MVP |
| [Replay y resultados](architecture/replay-and-results.md) | Evidencia autoritativa y publicación | Arquitectura vigente |
| [Entrenamiento y determinismo](architecture/training-and-determinism.md) | sim-core, Gym y decisión de físicas | Objetivo posterior al MVP |
| [Roadmap vigente](roadmap/current.md) | Único orden de trabajo aprobado | Prioridad de ejecución |
| [Decisiones](decisions/index.md) | Registro de ADR vigentes y sustituidas | Decisiones arquitectónicas |

## Guías operativas

| Documento | Propósito |
| --- | --- |
| [Desarrollo y pruebas](operations/development-and-testing.md) | Requisitos, comandos y pruebas disponibles |
| [Despliegue](operations/deployment.md) | Estado real y objetivo de empaquetado |
| [Seguridad](operations/security.md) | Amenazas, brechas actuales y condición de producción |
| [Protocolo local](../protocol/engine/v1/README.md) | Detalle del contrato Go↔Rust |
| [Bots de referencia](../games/starfighter/examples/README.md) | Ejemplos del MVP |
| [Bots con red neuronal](operations/neural-bots.md) | Cómo empaquetar y subir un bot con modelo (ADR-0014) |

Estas guías están subordinadas a las fuentes normativas y al código cuando describen el estado actual.

## Estado documental

Las etiquetas usadas en arquitectura son:

- **Implementado:** existe evidencia ejecutable y la comprobación relevante pasa.
- **Parcial:** existe una parte útil, pero faltan garantías o integración.
- **Objetivo aprobado:** dirección normativa aún no implementada.
- **No implementado:** no existe evidencia ejecutable actual.

## Historia

[docs/archive/](archive/index.md) contiene cuestionarios, blueprints, planes, prompts y auditorías anteriores. Se conserva para explicar decisiones, pero no es fuente vigente de requisitos o arquitectura.
