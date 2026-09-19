> [!WARNING]
> Documento histórico. No es una fuente vigente de requisitos ni arquitectura.
> Consúltese `docs/index.md` y `docs/roadmap/current.md` para el estado actual.

# Limpieza documental — 2026-09-19

## 1. Resumen ejecutivo

Se contrastó la documentación de Agentrix con los dos repositorios ejecutables
y se sustituyó la colección de planes superpuestos por una jerarquía pequeña:
producto, arquitectura, operaciones, decisiones y un único roadmap. El
blueprint, los prompts y los planes anteriores se conservaron como historia con
una advertencia uniforme. La única copia eliminada fue
`docs/blueprint/BLUEPRINT_COMPLETO.md`, duplicado concatenado que permanece en
Git.

El estado vigente declara Starfighter como único juego actual, Bevy + Avian2D
como motor del MVP, el sandbox y la recuperación de jobs como incompletos, y
**60 Hz exactos** como objetivo aún no satisfecho por `17` ms. No se modificó
comportamiento: fuera de Markdown solo cambió un comentario Rust obsoleto.

## 2. Inventario antes y después

El inventario inicial contenía 22 Markdown en Agentrix y uno en
`agentrix_engine`. La clasificación indica el destino decidido durante la
auditoría.

| Clasificación | Antes | Después |
| --- | ---: | ---: |
| Canónico | 3 | 16 |
| Apoyo operativo | 2 | 6 |
| Histórico con advertencia | 2 | 18 |
| Por archivar | 4 | 0 |
| Por consolidar | 11 | 0 |
| Duplicado por eliminar | 1 | 0 |
| **Total en ambos repositorios** | **23** | **40** |

Después de la limpieza hay 38 documentos en Agentrix y dos en
`agentrix_engine`. El aumento corresponde a separar fuentes canónicas breves
por responsabilidad; no duplica roadmaps ni contratos.

## 3. Mapa de fuentes vigentes

| Pregunta | Fuente |
| --- | --- |
| ¿Qué producto se construye? | [`product/vision.md`](../../product/vision.md) |
| ¿Qué entra al MVP y qué existe hoy? | [`product/mvp-scope.md`](../../product/mvp-scope.md) |
| ¿Cómo está dividido el sistema? | [`architecture/overview.md`](../../architecture/overview.md) |
| ¿Cómo se ejecutan jobs, bots y ticks? | [`architecture/execution-model.md`](../../architecture/execution-model.md) |
| ¿Cuál es la frontera Go↔Rust? | [`architecture/engine-protocol.md`](../../architecture/engine-protocol.md) |
| ¿Cómo se agregará otro juego? | [`architecture/game-extension.md`](../../architecture/game-extension.md) |
| ¿Qué prueba un replay? | [`architecture/replay-and-results.md`](../../architecture/replay-and-results.md) |
| ¿Cómo se llegará a sim-core y Gym? | [`architecture/training-and-determinism.md`](../../architecture/training-and-determinism.md) |
| ¿Qué se hace después? | [`roadmap/current.md`](../../roadmap/current.md) |
| ¿Qué decisiones están aceptadas? | [`decisions/index.md`](../../decisions/index.md) |

[`docs/index.md`](../../index.md) es la puerta de entrada y declara la
autoridad relativa de estas fuentes.

## 4. Movimientos

| Origen | Destino |
| --- | --- |
| `docs/adr/ADR-001_motor_externo_rust_y_protocolo_integracion.md` | `docs/decisions/0001-external-rust-engine.md` |
| `docs/blueprint/README.md`, `01`–`10` y `source/` | `docs/archive/blueprint/` |
| `docs/blueprint/00_ADAPTACION_AGENTRIX.md` | `docs/archive/audits/00_ADAPTACION_AGENTRIX.md` |
| `AGENTRIX_MASTER_ARCHITECTURE.md` | `docs/archive/plans/AGENTRIX_MASTER_ARCHITECTURE.md` |
| `COMO_EMPEZAR.md`, `PROMPT_INICIAL.md` | `docs/archive/prompts/` |

Los movimientos se hicieron en el árbol de trabajo sin alterar el contenido,
salvo la advertencia histórica y enlaces que dejaron de ser válidos. Git puede
detectar las renombradas por similitud al preparar el commit.

## 5. Consolidaciones y eliminación

| Fuente anterior | Destino de su contenido vigente |
| --- | --- |
| Blueprint `02`, `07` y `09` | `product/`, `operations/` y `roadmap/current.md` |
| Blueprint `03`, `04`, `05` y `08` | `architecture/` y `product/mvp-scope.md` |
| Blueprint `06` | `architecture/replay-and-results.md`; el detalle visual queda histórico |
| Blueprint `01` y `10` | `decisions/`, `docs/index.md` y reglas de avance del roadmap |
| Plan maestro y ADR-0001 | ADR-0002, arquitectura vigente y roadmap |
| README dispersos de protocolo y engine | Contrato local corregido y límites explícitos |
| `BLUEPRINT_COMPLETO.md` | Eliminado: copia exacta/concatenada; originales archivados e historial Git disponible |

## 6. Contradicciones resueltas

- **Producto:** Agentrix es el único nombre vigente.
- **Juego:** Starfighter es el único juego implementado; multi-juego es una
  etapa posterior, no una capacidad actual.
- **Física:** el MVP usa Avian2D. Rapier queda como decisión futura condicionada
  por benchmarks y conformidad multiplataforma.
- **Frecuencia:** la decisión es 60 Hz exactos. `17` ms describe la divergencia
  actual y no se redondea documentalmente a 60 Hz.
- **Entrenamiento:** no existe sim-core ni interfaz Gym actual; ambos dependen de
  la certificación del MVP y del segundo juego.
- **Sandbox:** Bubblewrap opcional con fallback directo no equivale a rootless
  OCI ni a aislamiento fail-closed.
- **Procesamiento único:** una reserva PostgreSQL de dos minutos no equivale a
  lease renovable, fencing ni commit final idempotente.
- **Despliegue:** el Dockerfile actual no empaqueta una unidad completa de
  producción ni separa API y worker.
- **Replay:** el NDJSON progresivo y los hashes existen, pero el sellado
  inmutable y la publicación atómica siguen parciales.

## 7. Decisiones pendientes

No quedan decisiones documentales abiertas para esta reorganización. Las
decisiones técnicas todavía no ejecutadas —representación exacta de 60 Hz,
sim-core, segundo juego, Gym y posible Rapier— tienen criterios y orden en el
roadmap; no deben resolverse por inferencia durante otro corte.

## 8. Validaciones

| Comprobación | Resultado observado |
| --- | --- |
| `go build -mod=readonly` | Correcto |
| `go vet -mod=readonly ./...` | Correcto |
| `go test -mod=readonly ./...` | Falla en `src/executor`: procesos de bot quedan `crashed` en este runner y el caso de timeout concluye `score_limit` |
| `npm test` en `web/` | Correcto: 6 archivos, 23 pruebas |
| build Vite con salida temporal | Correcto |
| `cargo test --locked --offline` con target temporal | Correcto: 11 unitarias y 1 integración; 1 estadística ignorada |
| build Rust release sobre un target vacío | No concluyó: test y release simultáneos agotaron la cuota temporal durante el enlace; la suite Rust posterior sí compiló y pasó |
| Comprobación de enlaces Markdown locales | Correcto después de la reorganización |
| `git diff --check` en ambos repositorios | Correcto |

No se instalaron dependencias, no se levantaron servicios y no se modificó una
base de datos.

## 9. Riesgos restantes

- La suite Go no está completamente verde; aún debe separarse una restricción
  del runner de un defecto real del executor.
- Los bots no cuentan con aislamiento de producción ni límites completos.
- Los jobs carecen de renovación de lease, fencing y finalización idempotente.
- El protocolo no valida bilateralmente todos los campos del sobre.
- Los 60 Hz exactos requieren cambiar contrato y configuración, no solo texto.
- La imagen de despliegue no incluye todavía todos los artefactos de ejecución.
- No existe certificación determinista cruzada ni flujo E2E desde entorno limpio.

## 10. Resumen del diff por repositorio

### Agentrix

- Reescritos `README.md` y `AGENTS.md`.
- Creada la jerarquía `docs/product`, `docs/architecture`, `docs/operations`,
  `docs/decisions`, `docs/roadmap` y `docs/archive`.
- Corregidos los README de protocolo y bots de referencia.
- Archivadas 16 fuentes históricas y eliminado el duplicado completo.
- Sin cambios en código, esquemas, dependencias ni configuración ejecutable.

### agentrix_engine

- Reescrito `README.md` y añadido `AGENTS.md`.
- Actualizado un comentario en `src/lib.rs` para apuntar a
  `docs/architecture/replay-and-results.md` del repositorio principal.
- Sin cambios de comportamiento, dependencias o configuración.
