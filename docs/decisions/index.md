# Decisiones arquitectónicas

Las decisiones se conservan sin reescribir retrospectivamente su contexto. Una decisión reemplazada permanece visible con estado `superseded` y enlace a su sucesora.

| ID | Estado | Decisión |
| --- | --- | --- |
| [0001](0001-external-rust-engine.md) | superseded | Motor externo Rust; la parte que fijaba Rapier fue sustituida |
| [0002](0002-starfighter-avian-mvp.md) | accepted | Starfighter con Avian para el MVP y ruta condicionada hacia sim-core/Gym/Rapier |

## Decisiones vigentes resumidas

- Agentrix usa Go para plano de control, API y worker.
- El motor oficial se ejecuta como proceso Rust headless separado.
- Starfighter y Python son las únicas opciones del MVP actual.
- PostgreSQL es la cola autoritativa fuera de desarrollo.
- El timestep objetivo es 60 Hz exactos.
- Multi-juego, Gym y la decisión Avian/Rapier siguen el orden de [roadmap/current.md](../roadmap/current.md).

Las brechas de implementación no cambian una decisión: se documentan como estado parcial hasta que una prueba las cierre.
