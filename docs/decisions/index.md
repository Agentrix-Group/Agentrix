# Decisiones arquitectónicas

Las decisiones se conservan sin reescribir retrospectivamente su contexto. Una decisión reemplazada permanece visible con estado `superseded` y enlace a su sucesora.

| ID | Estado | Decisión |
| --- | --- | --- |
| [0001](0001-external-rust-engine.md) | superseded | Motor externo Rust; la parte que fijaba Rapier fue sustituida |
| [0002](0002-starfighter-avian-mvp.md) | accepted | Starfighter con Avian para el MVP y ruta condicionada hacia sim-core/Gym/Rapier |
| [0003](0003-isolation-runtime-and-storage.md) | accepted | Runtime de aislamiento Podman rootless (fail-closed) y almacenamiento S3/MinIO |
| [0004](0004-execution-identity-and-disqualification-policy.md) | accepted | Identidad de ejecución (run_id, digests) y resolución simétrica de descalificación |
| [0005](0005-protocol-state-machine-and-exact-timestep.md) | accepted | Máquina de estados bilateral en el protocolo IPC y simulación exacta a 60 Hz |
| [0006](0006-transactional-match-commit-and-atomic-replay.md) | accepted | Commit transaccional de partidas, tabla match_runs y publicación atómica de replay |
| [0007](0007-separated-api-and-worker-roles-and-reproducible-containers.md) | accepted | Separación de roles de API y Worker y contenedores reproducibles |
| [0008](0008-mvp-canonical-certification-and-baseline-closure.md) | accepted | Certificación canónica del MVP y cierre formal del baseline |
| [0009](0009-frontend-qa-architecture-and-parallel-delivery.md) | accepted | Arquitectura de Frontend, Estrategia de QA y Entrega Paralela por Vertical Slices |

## Decisiones vigentes resumidas

- Agentrix usa Go para plano de control, API y worker.
- El motor oficial se ejecuta como proceso Rust headless separado.
- Starfighter y Python son las únicas opciones del MVP actual.
- PostgreSQL es la cola autoritativa fuera de desarrollo.
- El timestep objetivo es 60 Hz exactos.
- La entrega de producto se realiza de forma paralela por vertical slices frontend + backend + QA.
- Multi-juego, Gym y la decisión Avian/Rapier siguen el orden de [roadmap/current.md](../roadmap/current.md).

Las brechas de implementación no cambian una decisión: se documentan como estado parcial hasta que una prueba las cierre.
