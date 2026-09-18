# Trazabilidad y aprobación

## 1. Cobertura del cuestionario

| Respuestas | Decisión representada | Documento principal |
| --- | --- | --- |
| AR-001–AR-006 | propósito, prioridades y límites | 02_vision_y_alcance.md |
| AR-007–AR-011 | actores y autoridad | 04_actores_permisos_y_casos_uso.md |
| AR-012–AR-017 | experiencia completa y frontera | 02_vision_y_alcance.md, 05_flujos_estados_y_secuencias.md |
| AR-018–AR-024 | concurso, categorías, calendario y reglas | 03_dominio_y_modelo_clases.md, 05_flujos_estados_y_secuencias.md |
| AR-025–AR-030 | juego, versiones, percepción y azar | 03_dominio_y_modelo_clases.md, 08_arquitectura_tecnica.md |
| AR-031–AR-037 | agentes, envíos y selección | 03_dominio_y_modelo_clases.md, 04_actores_permisos_y_casos_uso.md |
| AR-038–AR-045 | partida, ticks, límites y aislamiento | 05_flujos_estados_y_secuencias.md, 08_arquitectura_tecnica.md |
| AR-046–AR-051 | resultados, justicia y reclamos | 04_actores_permisos_y_casos_uso.md, 07_requisitos.md |
| AR-052–AR-057 | directo, replay y experiencia pública | 06_ux_y_lenguaje_visual.md |
| AR-058–AR-062 | permisos, administración y organización | 04_actores_permisos_y_casos_uso.md |
| AR-063–AR-070 | lenguaje, identidad, relaciones e historia | 03_dominio_y_modelo_clases.md |
| AR-071–AR-077 | navegación y lenguaje visual | 06_ux_y_lenguaje_visual.md |
| AR-078–AR-083 | datos, acceso, secretos y respaldo | 07_requisitos.md, 08_arquitectura_tecnica.md |
| AR-084–AR-088 | escala, prioridad operativa y diagnóstico | 07_requisitos.md, 09_mvp_y_hoja_de_ruta.md |
| AR-089–AR-094 | restricciones y legibilidad técnica | 08_arquitectura_tecnica.md |
| AR-095–AR-100 | MVP, riesgo y crecimiento | 09_mvp_y_hoja_de_ruta.md |
| AR-101–AR-105 | control, pruebas y transferencia | README.md, 09_mvp_y_hoja_de_ruta.md |
| AR-106–AR-111 | síntesis y consistencia | 01_revision_y_resoluciones.md, este documento |

Todas las respuestas tienen representación. Las frases que no fijaban un valor concreto fueron convertidas en parámetro configurable o especificación pendiente, no en código arbitrario.

## 2. Trazabilidad de entidades

| Entidad | Necesidad | Fuente | Casos de uso |
| --- | --- | --- | --- |
| Cuenta | identidad privada y acceso | AR-020, AR-072, AR-078, AR-080 | UC-001, UC-002, UC-031 |
| Rol, Permiso, AsignaciónRol | permisos dinámicos | AR-058, AR-059, AR-071, AR-082 | UC-031 |
| Participante | unidad individual o equipo | AR-008, AR-021 | UC-003, UC-004 |
| Inscripción | elegibilidad, categoría y pago | AR-020 | UC-009, UC-010 |
| Concurso | totalidad organizativa | AR-018, AR-019 | UC-005, UC-008, UC-029, UC-030 |
| Reglamento | reglas fijas | AR-024 | UC-007 |
| Categoría | separar dificultad o percepción | AR-018, AR-030 | UC-006 |
| Fase y Ronda | estructura competitiva | AR-018, AR-022 | UC-020 |
| Juego y VersiónJuego | reglas y evolución | AR-025–AR-030 | UC-012, UC-013 |
| Agente y VersiónAgente | estrategia y mejoras | AR-001, AR-031, AR-037 | UC-014, UC-015 |
| Envío y Artefacto | presentación verificable | AR-035, AR-036 | UC-015 |
| Validación | declarar elegibilidad | AR-032, AR-034, AR-036 | UC-016 |
| SelecciónFinal | un agente aprobado para cierre | AR-037 | UC-018, UC-019 |
| Partida | unidad competitiva | AR-038–AR-040 | UC-020, UC-021 |
| EjecuciónPartida | distinguir intentos y fallos | AR-042, AR-045, AR-051 | UC-021, UC-025 |
| EventoTick | replay y auditoría | AR-029, AR-044 | UC-021, UC-028 |
| ResultadoPartida | consecuencia competitiva | AR-046, AR-051 | UC-023, UC-026 |
| Clasificación | evolución y top | AR-016, AR-047 | UC-027 |
| Replay | exploración posterior | AR-029, AR-055 | UC-028 |
| Incidente y Resolución | corregir sin borrar historia | AR-024, AR-050, AR-051 | UC-024, UC-025 |

## 3. Trazabilidad de la arquitectura

| Decisión técnica | Necesidad | Fuente |
| --- | --- | --- |
| Backend Go | backend no JavaScript, legible y concurrente | AR-089, AR-094 |
| PostgreSQL | preferencia relacional y consistencia | AR-064–AR-069, AR-089 |
| Almacenamiento de objetos | modelos, logs y replays grandes | AR-035, AR-044, AR-083 |
| API modular | funcionalidades separadas y entendibles | AR-005, AR-094 |
| Worker separado | priorizar y aislar partidas | AR-042, AR-045, AR-086, AR-091 |
| Sandbox sin red | secreto e integridad | AR-043, AR-049, AR-082 |
| Protocolo por ticks | equidad temporal | AR-033, AR-039, AR-040 |
| Event log | replay sin reejecutar agentes | AR-029, AR-044, AR-055 |
| WebSocket público | una partida en vivo | AR-052–AR-054 |
| RBAC con alcance | permisos dinámicos seguros | AR-058, AR-059, AR-071 |
| Digests y snapshots | historia defendible | AR-027–AR-029, AR-044, AR-051 |
| Repositorio horizontal por responsabilidad técnica | navegación simple y trazabilidad por nombre | DP-001, AR-094, AR-105 |
| Consola operativa y eventos estructurados | diagnóstico atribuible sin saturación ni exposición de secretos | AR-036, AR-042, AR-074, AR-087, RF-034, RF-065 |

## 4. Decisiones derivadas y decisión posterior

Marca cada una:

- [ ] RD-001: Concurso como término canónico.
- [ ] RD-002: Árbitro y Productor separados de Espectador.
- [ ] RD-003: reglas congeladas y calendario reprogramable.
- [ ] RD-004: una versión activa con historia por commit y digest.
- [ ] RD-005: replay por eventos y semilla complementaria.
- [ ] RD-006: estados específicos en vez de active universal.
- [ ] RD-007: permisos de negocio en vez de CRUD de tablas.
- [ ] RD-008: unit tests espejados más suites de integración separadas.
- [ ] RD-009: extensibilidad en el modelo y una variante por punto en MVP.
- [ ] RD-010: reintentar fallos técnicos, no fallos atribuibles al agente.
- [x] DP-001: repositorio horizontal y poco profundo al estilo de la referencia Capibara.
- [x] DP-002: Agentrix como nombre único y definitivo.
- [ ] ATD-001: Go para API y workers.
- [ ] ATD-002: TypeScript, React, Vite y PixiJS para web.
- [x] ATD-003: PostgreSQL para datos estructurados.
- [ ] ATD-004: almacenamiento de objetos para artefactos.
- [ ] ATD-005: cola persistente inicial en PostgreSQL.
- [ ] ATD-006: WebSocket para directo.
- [ ] ATD-007: protocolo JSON Lines por stdin/stdout.
- [ ] ATD-008: paquete comprimido con manifest.
- [ ] ATD-009: workers Linux y sandbox OCI endurecido.
- [ ] ATD-010: motor de juego fuera del proceso de la API.
- [ ] ATD-011: replay autoritativo por eventos.
- [ ] ATD-012: RBAC con capacidades y alcance.
- [ ] ATD-013: identidad técnica por commit, digest y versión de contrato.
- [ ] ATD-014: borrado controlado según tipo de dato.
- [ ] ATD-015: elegir entre las alternativas documentadas; se recomienda `repository` con interfaces pequeñas.
- [x] ATD-016: consola humana compacta, JSON estructurado y fallos registrados una sola vez.

## 5. Parámetros por completar

Estas casillas no bloquean los experimentos de Etapa 0, pero sí las etapas indicadas:

- [ ] PE-001: escala simultánea y latencias, antes de infraestructura.
- [ ] PE-002: presupuesto y equipos disponibles, antes de despliegue.
- [ ] PE-003: política institucional de datos, pagos y menores, antes de inscripciones.
- [ ] PE-004: reclamos, antes de publicar reglamento.
- [ ] PE-005: puntuación, antes de partidas oficiales.
- [ ] PE-006: runners y lenguajes, antes del kit.
- [ ] PE-007: límites de recursos, antes de validar envíos.
- [ ] PE-008: recuperación y segundo factor, antes de usuarios reales.

## 6. Puerta de entrada a implementación

Esta puerta se refiere a nueva implementación alineada con el blueprint. El código existente continúa clasificado como prototipo y no demuestra que estas casillas estén satisfechas.

La etapa de diseño está lista para convertirse en código cuando:

- [ ] José Daniel aprueba o corrige RD-001 a RD-010.
- [ ] José Daniel aprueba o corrige ATD-001 a ATD-015.
- [x] José Daniel decidió DP-001: estructura horizontal y poco profunda.
- [x] José Daniel decidió DP-002: Agentrix es el nombre definitivo.
- [ ] El mapa global de clases se entiende sin explicación externa.
- [ ] Los términos Concurso, Categoría, Participante, Agente, Envío, Partida, Ejecución y Resultado tienen un único significado.
- [ ] Cada entidad del MVP participa en al menos un caso de uso del MVP.
- [ ] Cada caso de uso del MVP tiene actor, autorización, estado inicial, resultado y fallos.
- [ ] El juego de demostración tiene reglas y protocolo propios.
- [ ] Los experimentos de aislamiento y replay tienen criterios de aceptación.
- [ ] La estructura propuesta del repositorio resulta navegable.
- [ ] No se crea ninguna carpeta o abstracción solo por una necesidad futura.
- [x] Primer corte de interfaz pública para consultar concursos definido y aprobado por José Daniel el 2026-09-18. Etapa 0 y los cambios de backend de producción permanecen aplazados.

## 7. Cambios durante la implementación

Si una prueba descubre que una decisión no funciona:

1. se registra la evidencia;
2. se identifica la decisión RD, ATD, RF o AR afectada;
3. se propone una sustitución;
4. José Daniel la aprueba;
5. la decisión anterior queda marcada como reemplazada;
6. se actualizan modelo, requisitos, pruebas y código en la misma PR.

## 8. Conclusión

El blueprint es coherente para comenzar experimentos técnicos. No está autorizado todavía a convertirse en implementación de producción: primero debe completarse la lista de aprobación y, antes de cada capacidad real, fijarse los parámetros que esa capacidad consume.
