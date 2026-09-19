> [!WARNING]
> Documento histórico. No es una fuente vigente de requisitos ni arquitectura.
> Consúltese `docs/index.md` y `docs/roadmap/current.md` para el estado actual.

# Revisión de decisiones y resoluciones

## 1. Decisiones firmes extraídas

| Área | Decisión normalizada | Fuente |
| --- | --- | --- |
| Propósito | Concurso universitario de agentes programados por participantes | AR-001–AR-003 |
| Prioridad | Rigor técnico y lógico por encima del espectáculo y la administración | AR-004, AR-108 |
| Escala | Una organización, aproximadamente 100 participantes, un concurso cada cuatro meses | AR-006, AR-062, AR-084 |
| Participación | Individual o por equipos; representación institucional opcional | AR-008, AR-020–AR-021 |
| Variantes | Categorías configurables por dificultad, forma de percepción u otras reglas | AR-018, AR-030 |
| Agentes | Se admiten algoritmos, búsqueda o aprendizaje automático dentro de límites explícitos | AR-031, AR-034 |
| Versiones | Un participante puede probar múltiples versiones, pero designa una versión aprobada para la final | AR-001, AR-037 |
| Simulación | Avance por ticks con tiempo real controlado y límites equivalentes | AR-033, AR-039–AR-041 |
| Percepción | Cada agente recibe solo la perspectiva permitida por el juego | AR-025, AR-029, AR-033 |
| Aleatoriedad | Mapas y apariciones pueden variar sin crear ventajas súbitas injustas | AR-001, AR-029, AR-048 |
| Replay | Se reconstruye desde eventos o estados registrados por tick | AR-029, AR-044, AR-055 |
| Ejecución | Sin Internet ni acceso a secretos, archivos internos o información prohibida | AR-043, AR-049 |
| Final | Una partida final en vivo, una a la vez, frente al auditorio | AR-012, AR-054, AR-057 |
| Datos | Estadísticas de todos los participantes; código y modelos de terceros permanecen privados | AR-030, AR-044, AR-046, AR-078, AR-082 |
| UI | Español, responsive, clara, temática luminosa y pastel, centrada en el concurso | AR-071–AR-077 |
| Persistencia | PostgreSQL es la preferencia; las partidas son el dato histórico prioritario | AR-083, AR-089, AR-092 |
| Backend | No usar JavaScript en backend | AR-089 |
| Calidad | Código legible, modular, sin hardcodeo, documentación paralela y pruebas ordenadas | AR-094, AR-101–AR-105 |
| Organización del código | Árbol horizontal, poco profundo y por responsabilidad técnica; una funcionalidad conserva nombres equivalentes entre OpenAPI, servidor, servicio, repositorio y modelo | DP-001 |
| Nombre del producto | Agentrix es el nombre único y definitivo del producto, repositorio y documentación | DP-002 |

## 2. Resoluciones derivadas

Estas resoluciones preservan la intención de las respuestas y eliminan contradicciones operativas.

### RD-001 — Terminología organizativa

**Conflicto:** AR-018 usa competencia como totalidad o ranking y considera concurso, torneo, evento, edición y temporada casi equivalentes. AR-063 pide no complicar la terminología.

**Resolución:** el concepto canónico será **Concurso**. Contiene categorías, fases, rondas y partidas. “Evento”, “torneo”, “edición” y “temporada” podrán aparecer como texto de presentación, pero no serán entidades distintas en la versión inicial. **Clasificación** será el ranking.

### RD-002 — Espectador y arbitraje

**Conflicto:** AR-007 y AR-011 hacen al espectador de solo lectura, mientras AR-010 le atribuye arbitraje o producción.

**Resolución:** **Espectador** conserva acceso exclusivamente público. Una misma persona podrá recibir además un rol independiente de **Árbitro** o **Productor**, pero esas capacidades nunca provienen del rol Espectador.

### RD-003 — Reglas fijas y calendario modificable

**Conflicto:** AR-023 permite cambiar fechas y rondas; AR-024 fija las reglas.

**Resolución:** el calendario operativo puede reprogramarse con notificación. El reglamento publicado queda congelado. Los vacíos se resuelven mediante una resolución del comité, registrada y pública; no se edita silenciosamente el reglamento usado por partidas ya disputadas.

### RD-004 — Una versión activa sin perder la historia

**Conflicto:** AR-027 desea conservar solo la versión actual del juego en la plataforma, pero AR-028, AR-029, AR-044 y AR-051 exigen evolución separada, replay y auditoría.

**Resolución:** habrá una sola versión activa por componente de juego, pero cada publicación y partida guardará el commit de Git y el digest inmutable del artefacto usado. El repositorio puede presentar solo lo actual sin perder la referencia histórica.

### RD-005 — Azar y replay

**Tensión:** AR-001 desea azar; AR-029 considera difícil repetirlo y elige logs por tick.

**Resolución:** el replay autoritativo usa eventos o estados por tick. Además se registra la semilla aleatoria cuando el juego pueda usarla. El azar permanece impredecible antes de jugar, pero auditable después.

### RD-006 — Desactivación y eliminación de datos

**Conflicto:** AR-066 prohíbe borrar y propone una marca activa; AR-079 permite borrar o anonimizar datos no esenciales.

**Resolución:** los registros que sostienen la historia competitiva nunca se eliminan físicamente; usan estados explícitos de ciclo de vida. Los datos personales pueden anonimizarse o eliminarse si la historia se conserva mediante un identificador no personal. No se impondrá una columna genérica active a todas las entidades: cada entidad tendrá estados que expresen su significado real.

### RD-007 — Permisos dinámicos

**Tensión:** AR-058, AR-059, AR-071 y AR-082 describen permisos como CRUD sobre tablas. Eso acoplaría seguridad y base de datos.

**Resolución:** los permisos serán dinámicos, pero expresados como **capacidades del negocio** y con alcance, por ejemplo concurso.publicar, envio.validar o partida.arbitrar. Ningún usuario recibe acceso directo a tablas.

### RD-008 — Pruebas por archivo y pruebas de flujo

**Tensión:** AR-094 pide una prueba por archivo y rechaza pruebas generales mal estructuradas; AR-103 exige validación real del sistema.

**Resolución:** los tests unitarios se organizan en un árbol separado y reflejan el archivo probado. Los recorridos que cruzan módulos tendrán suites explícitas de integración y extremo a extremo; no se mezclarán con unit tests ni se usarán como sustituto de ellos. Las pruebas beta complementan, no reemplazan, la automatización.

### RD-009 — Configurabilidad sin construir una superplataforma

**Tensión:** AR-022, AR-028, AR-031, AR-034 y AR-035 desean mucha configuración; AR-006, AR-081 y AR-093 limitan la ambición.

**Resolución:** el modelo admite puntos de variación, pero el MVP implementa una sola opción completa por punto. Se agrega una nueva variante únicamente cuando un concurso real la necesita.

### RD-010 — Errores del agente frente a errores de infraestructura

**Base:** AR-042, AR-050 y AR-051.

**Resolución:** un fallo atribuible al agente produce la consecuencia definida por el juego y no causa repetición. Un fallo de plataforma, juego o infraestructura puede anular la ejecución y crear un nuevo intento de la misma partida. Toda decisión queda auditada.

## 3. Decisiones posteriores al cuestionario

### DP-001 — Organización horizontal del repositorio

**Estado:** decidida por José Daniel el 2026-09-13.

**Decisión:** el repositorio de Agentrix seguirá el estilo de organización usado como referencia en Capibara: pocas carpetas, agrupadas por responsabilidad técnica, y archivos equivalentes por funcionalidad.

La adaptación técnica propuesta conserva el recorrido por nombre y añade `repository` como única capa respecto de la referencia: `open-api/contests.yaml` → `src/server/contests.go` → `src/service/contests.go` → `src/repository/contests.go` → `src/model/contest.go`. La razón es evitar mezclar SQL y persistencia con las reglas de los casos de uso. Esta adición forma parte de ATD-015 y aún debe aprobarse o retirarse; no se atribuye a la decisión textual de José Daniel.

No se usará `internal/` dividido por capacidades ni una jerarquía profunda de paquetes.

Los archivos de prueba se omiten del árbol arquitectónico mostrado por no ser relevantes para esta decisión. Esto no elimina ni modifica la estrategia de pruebas definida en RD-008.

Esta decisión reemplaza la estructura por capacidades que aparecía en la primera versión de la propuesta arquitectónica.

### DP-002 — Nombre definitivo Agentrix

**Estado:** decidida por José Daniel el 2026-09-13.

**Decisión:** Agentrix es el único nombre vigente del producto, repositorio y documentación. Los nombres anteriores dejan de usarse en contratos, código, textos y archivos activos.

Esta decisión no cambia el alcance ni las respuestas AR-001 a AR-111; únicamente fija la identidad del mismo sistema.

### DP-003 — Motor de juego oficial externo en Rust (Bevy + Rapier)

**Estado:** decidida el 2026-09-15 (ADR-001).

**Decisión:** el motor de juego oficial es un ejecutable externo e independiente escrito en Rust con Bevy y Rapier que corre en modo headless. Agentrix (en Go) conserva el plano de control, supervisión, cola y persistencia. La simulación y los resultados son autoritativos del motor. La comunicación inicial se realiza mediante el protocolo versionado `agentrix-engine/1` sobre `stdin/stdout` (JSON Lines). Los agentes no corren dentro del motor ni el motor accede a la base de datos de Agentrix. La implementación previa `ArenaBasicaEngine` en Go queda retirada por ser provisional.

DP-004 actualiza la biblioteca de física y fija el alcance ejecutable del MVP descrito por esta decisión.

### DP-004 — Corte vertical público de Starfighter

**Estado:** decidido explícitamente por José Daniel el 2026-09-18.

**Decisión:** el MVP de Agentrix implementa un único recorrido vertical público: carga de un bot Python, admisión mediante ZIP, partida de Starfighter para dos agentes en el motor Rust con Bevy y Avian2D, replay público autoritativo en NDJSON y reproducción en React con Canvas 2D.

Para este MVP:

- Starfighter es el único juego registrable y ejecutable;
- Python es el único lenguaje de agentes;
- el paquete contiene exactamente `agentrix.json` y `bot.py` en su raíz;
- el protocolo de bot usa solamente `init`, `perception`, `action` y `end` por JSON Lines;
- el ciclo autoritativo comienza en tick 0: `State[0]` → `Perception[0]` → `Action[0]` → `State[1]`;
- Go transporta las percepciones y acciones como JSON opaco; Rust interpreta el dominio espacial;
- un timeout por tick termina el proceso del bot y lo descalifica;
- Rust emite percepciones privadas y un snapshot público sanitizado por tick;
- Go escribe metadata, snapshots y resultado directamente como NDJSON, sin reconstruir el estado;
- la cola persistente reserva trabajos en PostgreSQL con `FOR UPDATE SKIP LOCKED`;
- el replay se consulta por HTTP y no se implementa transmisión en vivo por WebSocket.

Esta decisión retira Arena Básica y las configuraciones genéricas para registrar otros juegos del alcance activo. Aplica RD-009 mediante una sola opción completa. También reemplaza Rapier por Avian2D en DP-003. La incorporación de otro juego, lenguaje o transporte requiere una necesidad aprobada de un concurso posterior.

## 4. Parámetros configurables, no decisiones de arquitectura

Las respuestas delegan varios valores al comité. Se modelarán como configuración del concurso, categoría o juego:

- cantidad de agentes por partida;
- lenguajes y dependencias permitidos;
- memoria, CPU, almacenamiento, duración y timeout;
- frecuencia de ticks;
- mapas y distribución de semillas;
- forma de puntuación y estadísticas;
- criterio de desempate;
- formato competitivo;
- plazos entre envíos y fecha de congelamiento;
- política de memoria del agente;
- forma exacta del paquete presentado;
- reglas de sanción aplicables.

## 5. Especificaciones que deberán fijarse antes de producción

No bloquean el diseño conceptual, pero sí una implementación de producción:

| ID | Especificación pendiente | Momento límite |
| --- | --- | --- |
| PE-001 | Valores cuantitativos de escala simultánea y latencias | Antes de dimensionar infraestructura |
| PE-002 | Presupuesto máximo y equipo disponible para el evento | Antes de desplegar |
| PE-003 | Política institucional de datos, pagos y participantes menores | Antes de abrir inscripciones |
| PE-004 | Plazos y composición del proceso de reclamos | Antes de publicar reglas |
| PE-005 | Fórmula de puntuación del primer juego | Antes de ejecutar partidas oficiales |
| PE-006 | Runners y lenguajes del primer concurso | Antes de publicar el kit |
| PE-007 | Límites de recursos del primer concurso | Antes de validar envíos |
| PE-008 | Recuperación de cuenta y segundo factor para administradores | Antes de operar con usuarios reales |

## 6. Resultado de la revisión

No existe una contradicción que impida crear el modelo conceptual o el MVP. Las respuestas sí impiden declarar lista una versión de producción hasta completar PE-001 a PE-008. La arquitectura debe mantener esos valores fuera del código y permitir que el comité los configure o publique.
