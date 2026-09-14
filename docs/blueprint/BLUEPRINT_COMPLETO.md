# Agentrix — Blueprint de producto y sistema

**Versión:** 0.1  
**Fecha:** 2026-09-13  
**Estado:** línea base derivada y prototipo auditado; la implementación alineada con el blueprint permanece aplazada hasta aprobar un corte.

## Fuente

Este blueprint deriva exclusivamente de las respuestas AR-001 a AR-111 del cuestionario maestro. José Daniel declaró todas las respuestas como decisiones. No se reutilizó el diseño ni el código de intentos anteriores.

Las decisiones textuales del cuestionario son la fuente primaria. Cuando fue necesario convertir una intención general en una regla concreta, el resultado aparece identificado como **resolución derivada** o **parámetro pendiente**, nunca como una afirmación textual del cuestionario.

Las decisiones posteriores se identifican como **DP**. DP-001 fija una organización horizontal y poco profunda del repositorio, basada en la referencia Capibara proporcionada por José Daniel. DP-002 fija Agentrix como nombre único y definitivo.

## Resultado central

Agentrix será una plataforma universitaria para organizar concursos en los que participantes individuales o equipos presentan agentes que controlan jugadores dentro de juegos configurados por el comité. Los agentes perciben solo la información permitida, actúan por ticks bajo límites equivalentes y compiten en partidas cuyos eventos, resultados y estadísticas quedan registrados. La culminación es una final en vivo, mostrada en un auditorio, cuyo resultado nadie conoce de antemano.

## Orden de lectura

**Fuente original:** [source/cuestionario_maestro_agentrix.md](source/cuestionario_maestro_agentrix.md), con las respuestas AR-001 a AR-111 declaradas como decididas.

0. [00_ADAPTACION_AGENTRIX.md](00_ADAPTACION_AGENTRIX.md): relación entre el blueprint y el prototipo actual de Agentrix.
1. [01_revision_y_resoluciones.md](01_revision_y_resoluciones.md): decisiones firmes, contradicciones resueltas y parámetros aún configurables.
2. [02_vision_y_alcance.md](02_vision_y_alcance.md): propósito, experiencia, alcance y límites.
3. [03_dominio_y_modelo_clases.md](03_dominio_y_modelo_clases.md): glosario, entidades, relaciones, invariantes y diagramas de clases.
4. [04_actores_permisos_y_casos_uso.md](04_actores_permisos_y_casos_uso.md): actores, autorización y comportamiento esperado.
5. [05_flujos_estados_y_secuencias.md](05_flujos_estados_y_secuencias.md): ciclos de vida y recorridos críticos.
6. [06_ux_y_lenguaje_visual.md](06_ux_y_lenguaje_visual.md): navegación, pantallas y dirección visual.
7. [07_requisitos.md](07_requisitos.md): requisitos funcionales y de calidad verificables.
8. [08_arquitectura_tecnica.md](08_arquitectura_tecnica.md): módulos, procesos, datos, contratos, seguridad y estructura del código.
9. [09_mvp_y_hoja_de_ruta.md](09_mvp_y_hoja_de_ruta.md): primer corte vertical, etapas y riesgos.
10. [10_trazabilidad_y_aprobacion.md](10_trazabilidad_y_aprobacion.md): correspondencia con el cuestionario y puerta de entrada a implementación.

Para una lectura continua también está disponible [BLUEPRINT_COMPLETO.md](BLUEPRINT_COMPLETO.md). Reúne la fuente y estos documentos; si existiera una diferencia accidental, prevalecen los archivos separados.

## Jerarquía de decisiones

Cuando dos respuestas admiten interpretaciones incompatibles, se aplica este orden:

1. Corrección técnica y lógica.
2. Seguridad y aislamiento.
3. Aprendizaje del participante.
4. Flexibilidad que tenga un caso real.
5. Experiencia del espectador.
6. Conveniencia administrativa.
7. Funciones complementarias.

Esto combina AR-004 y AR-108.

## Regla de cambio

- José Daniel es la autoridad final (AR-102).
- Una modificación debe indicar la decisión y documentos afectados.
- No se borra una decisión anterior: se marca como reemplazada y se enlaza su sucesora.
- La nueva implementación alineada con el blueprint no comienza hasta aprobar la lista de control de 10_trazabilidad_y_aprobacion.md.
- Cada cambio de implementación vive en una rama y llega mediante pull request (AR-104).

## Estado actual

- Producto y dominio: definidos como línea base 0.1.
- Diagrama conceptual de clases: incluido.
- Casos de uso y estados: incluidos.
- UX y dirección visual: definidas a nivel de sistema.
- Arquitectura técnica: PostgreSQL aprobado; las demás ATD continúan según su estado en la lista de control.
- Código: existe un prototipo auditado y documentado en `00_ADAPTACION_AGENTRIX.md`; no equivale a implementación aprobada.

# Adaptación del blueprint al repositorio Agentrix

## 1. Nombre definitivo

**Agentrix** es el nombre único y definitivo del producto y del repositorio según DP-002. Los contratos, documentos, módulos y textos nuevos deben usar únicamente este nombre.

## 2. Naturaleza del código actual

El contenido versionado en el commit inicial es un prototipo creado para probar estructura, API, persistencia, ejecución y visualización. No constituye una implementación aprobada del blueprint y no convierte sus decisiones técnicas en requisitos.

La auditoría no destructiva del 2026-09-13 comprobó que el prototipo contiene:

- backend Go horizontal en `src/`;
- contratos HTTP en `open-api/` y contratos JSON en `contracts/`;
- persistencia y scripts para MySQL;
- API HTTP con autenticación JWT y CRUD básicos;
- cola de partidas en memoria y workers dentro del proceso de la API;
- motor de demostración Arena Básica implementado en Go y duplicado como referencia Python;
- ejecución directa de scripts Python del host;
- grabación de frames de replay en archivos JSON;
- archivos React que consumen la API, todavía sin punto de entrada que monte la aplicación;
- seis archivos de pruebas Go centrados en auth, common, config, game, replay y validadores HTTP.

## 3. Estado verificado por área

| Área | Estado | Evidencia resumida |
| --- | --- | --- |
| Compilación, `go test` y `go vet` | Verificado | Los tres comandos terminan correctamente con módulos en modo de solo lectura. |
| Organización horizontal DP-001 | Parcial | Existen `server`, `service`, `repository` y `model`, aunque algunas interfaces y responsabilidades son demasiado amplias. |
| Composición de `main.go` | Parcial | Compone dependencias, pero API y workers se ejecutan en el mismo proceso. |
| OpenAPI y handlers | Parcial | Hay contratos y rutas reales, pero no siempre coinciden con el acceso público o con las proyecciones del blueprint. |
| Modelo de dominio | Contradictorio | Predominan DTO de tablas, estados libres y `active`; faltan agregados e invariantes. |
| Identidad y autorización | Contradictorio | Hay JWT y permisos globales, pero no capacidades con alcance; el registro permite recibir `role_id` del cliente. |
| Participantes e inscripciones | Ausente | La entidad llamada `Participant` representa una cuenta; no existen inscripción, equipo, membresía ni congelamiento. |
| Juego Arena Básica | Parcial | El motor Go avanza por ticks y termina; no aplica percepciones privadas ni aislamiento de proceso. |
| Envíos y validación | Contradictorio | Se guarda código directamente, sin digest, manifest, validación reproducible ni inmutabilidad. |
| Sandbox y cola | Contradictorio | Python se ejecuta en el host y la cola se pierde al reiniciar. |
| Partidas, resultados y clasificación | Contradictorio | No hay snapshot, intentos, provisionalidad, incidentes ni política configurable. |
| Replay | Parcial | Conserva frames reproducibles, pero carece de checksum, sello, digest y proyección pública sanitizada. |
| Directo WebSocket | Ausente | No existe canal en vivo. |
| Web | Stub | Los componentes React no están montados; el HTML visible es estático, oscuro y en inglés. |
| Pruebas automatizadas | Parcial | No cubren `service`, `repository`, `executor`, seguridad, integración o extremo a extremo. |

La cantidad de archivos no se usa como medida de avance.

## 4. Comprobaciones realizadas

Se ejecutaron sin instalar ni actualizar dependencias y con cachés fuera del repositorio:

~~~text
go build -mod=readonly .      correcto
go test -mod=readonly ./...   correcto
go vet -mod=readonly ./...    correcto
~~~

Los tres archivos Python se analizaron sintácticamente y el motor de referencia completó una simulación de dos ticks. Esto no prueba aislamiento, integración con el backend ni seguridad.

No se levantó MySQL, no se ejecutaron scripts SQL y no se compiló la web. `script/capsule.sh` puede invocar MySQL y el SQL ensamblado elimina y vuelve a crear usuario y base; `web/node_modules/` no existe.

## 5. Diferencias estructurales que deben conservarse como deuda explícita

- El prototipo usa MySQL; la dirección aprobada mediante ATD-003 es PostgreSQL.
- `Cuenta` y `Participante` están fusionados.
- `Contest` referencia directamente un juego y una categoría, en lugar de contener categorías.
- Los permisos son globales y cercanos a CRUD; no expresan capacidad, alcance y condición.
- El supuesto sandbox entrega el estado completo de todos los jugadores y ejecuta Python en el host.
- Un fallo del agente puede activar una heurística sustituta, por lo que el resultado deja de representar al agente presentado.
- La cola no es persistente y no recupera trabajos.
- El resultado no nace provisional ni conserva intentos e incidentes.
- El replay no está sellado y puede contener información que no pertenece a una vista pública.
- La web no implementa todavía la experiencia responsive, luminosa y en español.

Estas diferencias se documentan por ahora. No autorizan una reescritura general ni implican que deban corregirse todas en el mismo corte.

## 6. ATD-015: alternativas para `repository`

El prototipo demuestra un uso concreto de `repository`: por ejemplo, `service.ListContests` delega la consulta y `repository.ListContests` contiene SQL y mapeo de filas. La separación evita que `service` conozca detalles de base de datos.

### Alternativa A — Conservar `repository` con interfaces pequeñas

- `service` conserva autorización, reglas y transiciones.
- `repository` conserva SQL, transacciones locales y mapeo de persistencia.
- `connection` crea conexiones y clientes, sin consultas del dominio.
- Cada servicio depende solo de la interfaz mínima que consume.
- Los archivos mantienen nombres paralelos por funcionalidad.

Consecuencia: aparece una delegación adicional en operaciones simples, pero las pruebas de casos de uso pueden sustituir persistencia sin mezclar SQL con reglas.

### Alternativa B — Retirar `repository` y llevar persistencia a `service`

- Reduce una llamada y una interfaz en consultas sencillas.
- Obliga a que `service` conozca SQL, transacciones y mapeo de filas.
- Aumenta el costo de probar autorización y reglas sin una base real.
- Dificulta mantener la frontera de PostgreSQL al crecer un caso de uso.

### Alternativa C — Retirar `repository` y llevar consultas a `connection`

- Mantiene SQL fuera de `service`.
- Convierte `connection` en una capa de dominio con consultas de concursos, envíos y partidas.
- Mezcla creación de clientes técnicos con semántica del negocio y pierde el recorrido paralelo de DP-001.

### Recomendación

Conservar la alternativa A, pero no conservar la interfaz monolítica actual. No se crearán repositorios para capacidades futuras y no se añadirá un patrón adicional por encima o por debajo. ATD-015 permanece pendiente de confirmación expresa.

## 7. Contrato corregido del primer flujo de consulta

El flujo candidato se documenta, pero queda aplazado hasta que José Daniel autorice implementación:

`GET /api/v1/contests` → `server` → `service` → `repository` → PostgreSQL → respuesta OpenAPI

### Entidad y visibilidad

- `Contest` posee identidad, nombre, descripción, estado y fechas públicas.
- `draft` nunca es visible públicamente.
- Un concurso que ya fue publicado sigue siendo visible si queda `suspended` o `cancelled`.
- `archived` se excluye por defecto.

### Autorización

- Consulta anónima y de solo lectura.
- No requiere crear una cuenta ni asumir el rol Espectador.
- Devuelve una proyección pública, nunca el modelo interno de persistencia.

### Entrada

- `state`: filtro opcional con un único estado público válido.
- `include_archived`: booleano opcional; su valor por defecto es `false`.
- No se incorpora paginación hasta que exista una necesidad real para la escala inicial.

### Salida

- `200 OK` con un arreglo de `PublicContestSummary`.
- El arreglo vacío se serializa como `[]`, no como `null`.
- Cada elemento contiene `id`, `name`, `description`, `state`, `starts_at` y `ends_at`.

### Errores

- `400 Bad Request` para un filtro inválido.
- `500 Internal Server Error` para un fallo de persistencia, sin detalles SQL y con `X-Request-Id`.
- La ausencia de concursos no es un error.

### Criterios de aceptación documentados

1. Una solicitud sin credenciales obtiene una respuesta pública.
2. Nunca aparece un concurso `draft`.
3. Solo se aceptan estados del catálogo público.
4. `archived` aparece únicamente cuando `include_archived=true`.
5. La respuesta no expone campos internos ni datos privados.
6. OpenAPI, handler, servicio, repositorio y modelo usan el mismo contrato.
7. Las pruebas unitarias, negativas y de integración con PostgreSQL pasan antes de considerar terminado el flujo.

## 8. Archivos generados y limpieza aprobada

- `bin/` y `artifacts/` permanecen ignorados.
- `script/capsule.sql` permanece ignorado y no debe ejecutarse durante una revisión.
- El bytecode Python versionado fue retirado y `.gitignore` cubre `__pycache__/` y `*.py[cod]`.
- El ZIP de traspaso, redundante después de extraer la documentación, fue retirado.

## 9. Estado de implementación

Por decisión de José Daniel, esta revisión solo documenta lo que ya existe y corrige el blueprint. Etapa 0, Etapa 1 y el flujo público de concursos quedan aplazados por ahora. PostgreSQL sigue siendo la dirección de persistencia; no se adaptará el código hasta aprobar un corte específico.

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

# Visión y alcance

## 1. Visión

Agentrix convierte la creación de agentes en una competencia universitaria observable. Cada participante desarrolla una estrategia, la prueba bajo reglas conocidas y presenta una versión final. La plataforma ejecuta a todos bajo límites equivalentes, conserva evidencia de lo ocurrido y culmina en una final en vivo cuyo resultado emerge de las decisiones de los agentes y de un azar controlado.

## 2. Problema

No existe una plataforma que reúna, bajo control de la universidad:

- definición del concurso y sus categorías;
- documentación de juegos y límites;
- recepción y validación de agentes;
- pruebas durante el periodo competitivo;
- ejecución aislada y comparable;
- clasificación, auditoría y resolución de incidentes;
- visualización comprensible para público no técnico;
- conservación del historial.

La oportunidad es construir una solución pequeña y ajustada al evento, no un producto masivo (AR-002, AR-006).

## 3. Objetivos

### O-01 — Evaluación rigurosa

Ejecutar agentes bajo reglas, recursos y condiciones comparables, separando fallos propios del agente de fallos de plataforma (AR-004, AR-033, AR-041, AR-042, AR-048).

### O-02 — Aprendizaje

Dar documentación, agentes de ejemplo, validación reproducible y tiempo para que los estudiantes prueben y mejoren estrategias (AR-003, AR-014, AR-032, AR-036).

### O-03 — Espectáculo comprensible

Mostrar las partidas con una narrativa visual clara y una final en vivo para el auditorio (AR-012, AR-052–AR-057).

### O-04 — Operación controlada

Permitir que el comité configure el concurso, supervise la ejecución, arbitre incidentes y preserve trazabilidad (AR-009, AR-019, AR-024, AR-050, AR-058–AR-061).

### O-05 — Reutilización acotada

Reutilizar la plataforma para concursos periódicos y distintos juegos, sin convertirla en infraestructura para eventos masivos (AR-006, AR-030, AR-062, AR-084).

## 4. Experiencia canónica

1. El comité crea un concurso y define categorías, reglas, fechas, juegos y límites.
2. Se publica el concurso y se difunde dentro de la universidad.
3. Una persona crea su cuenta, forma o integra un participante individual/equipo y se inscribe.
4. El participante consulta documentación y prueba agentes de referencia.
5. Durante el periodo competitivo presenta versiones de su agente.
6. Cada envío se valida en un entorno aislado y entrega un informe reproducible.
7. Los envíos aprobados pueden disputar partidas de prueba; resultados y estadísticas muestran la evolución.
8. Antes del cierre, cada participante designa un único agente aprobado.
9. El sistema congela selecciones y programa la etapa final.
10. La final se ejecuta en vivo, una partida a la vez, y se proyecta en el auditorio.
11. Los resultados quedan provisionales hasta superar comprobaciones o reclamos.
12. La clasificación final, estadísticas y replays permitidos pasan al archivo público.

## 5. Alcance organizativo inicial

- Una organización universitaria.
- Administración global a cargo de José Daniel.
- Aproximadamente 100 participantes.
- Concursos aproximadamente cada cuatro meses.
- Participación individual o por equipos.
- Posibilidad de representar una institución académica.
- Público con acceso de solo lectura.

## 6. Dentro de Agentrix

- cuentas, perfiles privados y autorización;
- administración del concurso;
- categorías, fases, rondas y calendario;
- publicación de reglas y resoluciones;
- participantes, equipos e inscripciones;
- catálogo y publicación de juegos;
- políticas de agentes y ejecución;
- envíos, artefactos, validaciones y selección final;
- programación, cola y ejecución de partidas;
- eventos por tick, resultados y estadísticas;
- clasificación, reclamos y resoluciones;
- directo, replay y archivo público;
- notificaciones dentro de la plataforma;
- auditoría y reportes operativos.

## 7. Fuera de alcance inicial

- operar múltiples organizaciones independientes;
- eventos masivos o infraestructura global;
- tienda, venta de productos o monetización;
- red social general;
- transmisión profesional integrada, clips automáticos o producción multicámara;
- soporte inmediato para todo lenguaje y formato de agente;
- visión computacional en el primer corte vertical;
- intervención humana durante la ejecución de un agente;
- acceso de agentes a Internet;
- modificación arbitraria de resultados sin resolución auditada.

El pago simbólico de inscripción puede registrarse en la plataforma, pero su procesamiento será manual en el MVP.

## 8. Principios de producto

1. **Rigor antes que apariencia.**
2. **El agente solo conoce lo permitido.**
3. **El error debe ser atribuible y reproducible.**
4. **El participante puede aprender de sus propios fallos.**
5. **La historia no se reescribe silenciosamente.**
6. **La configuración responde a casos reales.**
7. **Una partida nunca debe comprometer la plataforma.**
8. **El público comprende sin acceder a secretos competitivos.**
9. **La interfaz ofrece profundidad sin saturación.**
10. **No existe funcionalidad sin responsable, permiso y resultado.**

## 9. Señales de éxito

La primera versión demuestra éxito cuando:

- un usuario puede ingresar;
- puede presentar un agente;
- el sistema valida el agente y explica sus fallos;
- una partida con agentes llega a término;
- la ejecución no rompe el servidor;
- se conserva un registro por ticks;
- el resultado y un replay son visibles;
- otra persona puede comprender el recorrido mediante documentación;
- el código compila y sus pruebas automatizadas pasan.

La primera edición demuestra éxito cuando el concurso completo llega a una final en vivo, los resultados pueden defenderse con evidencia y los participantes pueden observar cómo mejoraron sus agentes durante el periodo competitivo.


# Dominio y modelo conceptual de clases

## 1. Lenguaje canónico

| Término | Definición |
| --- | --- |
| Organización | Institución que opera Agentrix. Solo existe una en el alcance inicial. |
| Concurso | Unidad completa publicada por el comité: reglas, calendario, categorías y resultados. |
| Categoría | Separación de participantes con juego, dificultad, percepción o políticas comunes. |
| Fase | Etapa temporal o competitiva dentro de una categoría. |
| Ronda | Agrupación de partidas dentro de una fase. No es sinónimo de partida. |
| Juego | Conjunto de reglas que define estado, percepciones, acciones y finalización. |
| Versión de juego | Publicación inmutable identificada por commit y digest. |
| Participante | Unidad inscrita; puede representar a una persona o un equipo. |
| Agente | Estrategia creada por un participante para controlar un jugador. |
| Versión de agente | Evolución concreta de un agente. |
| Envío | Intento de presentar una versión y su artefacto para validación. |
| Validación | Evaluación reproducible que declara un envío aprobado o rechazado. |
| Selección final | Designación de la única versión aprobada que representará a un participante. |
| Partida | Competencia identificable entre versiones de agentes bajo una configuración. |
| Ejecución | Intento técnico de realizar una partida. Una partida puede requerir otro intento por fallo de infraestructura. |
| Evento de tick | Hecho autoritativo registrado durante la ejecución. |
| Resultado | Consecuencia competitiva calculada a partir de una ejecución válida. |
| Clasificación | Orden agregado de los participantes según una política de puntuación. |
| Replay | Representación reproducible creada desde los eventos registrados. |
| Incidente | Situación que puede afectar la validez u operación. |
| Resolución | Decisión auditada del comité o árbitro sobre reglas, incidentes o resultados. |

Se evitan como entidades independientes iniciales: edición, temporada, torneo y evento. Pueden ser nombres de presentación del Concurso.

## 2. Contextos del dominio

| Contexto | Responsabilidad | No debe decidir |
| --- | --- | --- |
| Acceso | Cuentas, roles, permisos y alcances | Reglas competitivas |
| Concursos | Concurso, categorías, calendario, inscripción y selección final | Ejecutar código |
| Juegos | Versiones del juego, contrato, mapas y políticas | Identidad de usuarios |
| Agentes | Agentes, versiones, envíos, artefactos y validaciones | Clasificación |
| Partidas | Programación, slots, ejecuciones y eventos por tick | Cambiar reglas publicadas |
| Resultados | Resultados, estadísticas, clasificación, incidentes y resoluciones | Ejecutar agentes |
| Experiencia pública | Directo, replay y vistas públicas | Ser fuente del resultado |
| Operación | Colas, reportes, auditoría, respaldo y recuperación | Inventar decisiones de dominio |

## 3. Diagrama global

El mapa global sacrifica atributos y clases auxiliares para conservar orientación.

~~~mermaid
classDiagram
direction LR

class Cuenta
class Participante {
  tipo
  nombrePublico
}
class MiembroParticipante
class Concurso
class Categoria
class Inscripcion
class Juego
class VersionJuego
class Agente
class VersionAgente
class Envio
class Validacion
class SeleccionFinal
class Partida
class ParticipacionPartida
class EjecucionPartida
class ResultadoPartida
class Replay
class Clasificacion

Cuenta "1" -- "0..*" MiembroParticipante
Participante "1" *-- "1..*" MiembroParticipante
Concurso "1" *-- "1..*" Categoria
Participante "1" -- "0..*" Inscripcion
Categoria "1" -- "0..*" Inscripcion
Juego "1" *-- "1..*" VersionJuego
Categoria "1" --> "1" VersionJuego : usa
Participante "1" *-- "0..*" Agente
Agente "1" *-- "1..*" VersionAgente
VersionAgente "1" -- "1..*" Envio
Envio "1" *-- "0..*" Validacion
Inscripcion "1" -- "0..1" SeleccionFinal
SeleccionFinal "1" --> "1" VersionAgente
Categoria "1" *-- "0..*" Partida
Partida "1" *-- "2..*" ParticipacionPartida
ParticipacionPartida "*" --> "1" VersionAgente
Partida "1" *-- "1..*" EjecucionPartida
Partida "1" --> "0..1" ResultadoPartida
EjecucionPartida "1" --> "0..1" Replay
Categoria "1" --> "0..*" Clasificacion
~~~

## 4. Identidad y permisos

~~~mermaid
classDiagram
direction LR

class Cuenta {
  UUID id
  EstadoCuenta estado
  Instant creadoEn
}
class PerfilPrivado {
  nombre
  apellidos
  institucion
}
class Rol {
  UUID id
  string nombre
}
class Permiso {
  string capacidad
}
class AsignacionRol {
  TipoAlcance tipoAlcance
  UUID alcanceId
  Instant otorgadoEn
  Instant revocadoEn
}

Cuenta "1" *-- "1" PerfilPrivado
Cuenta "1" -- "0..*" AsignacionRol
Rol "1" -- "0..*" AsignacionRol
Rol "*" -- "*" Permiso
~~~

### Invariantes

- Una cuenta solo modifica recursos propios salvo permiso explícito con alcance.
- Los datos del perfil privado nunca forman parte automática del perfil público.
- Un permiso describe una capacidad del dominio, no una tabla o sentencia SQL.
- Una asignación revocada no vuelve a activarse; se crea una nueva asignación.
- Las acciones administrativas sensibles generan auditoría.

## 5. Concurso y juego

~~~mermaid
classDiagram
direction LR

class Concurso {
  UUID id
  string nombre
  EstadoConcurso estado
}
class Reglamento {
  UUID id
  int version
  EstadoPublicacion estado
}
class HitoCalendario {
  TipoHito tipo
  Instant programadoPara
}
class Categoria {
  UUID id
  string nombre
}
class Fase {
  UUID id
  TipoFase tipo
  int orden
}
class Ronda {
  UUID id
  int orden
}
class Juego {
  UUID id
  string nombre
}
class VersionJuego {
  UUID id
  string commitGit
  string digestArtefacto
  EstadoPublicacion estado
}
class ComponenteJuego {
  TipoComponente tipo
  string version
  string digest
}
class PoliticaCategoria {
  ModoPercepcion percepcion
  bool permiteMemoria
  int agentesPorPartida
}
class PerfilRecursos {
  Duration timeoutTick
  int cpuMax
  int memoriaMax
}

Concurso "1" *-- "1..*" Reglamento
Concurso "1" *-- "1..*" HitoCalendario
Concurso "1" *-- "1..*" Categoria
Categoria "1" *-- "1..*" Fase
Fase "1" *-- "1..*" Ronda
Categoria "1" *-- "1" PoliticaCategoria
PoliticaCategoria "1" *-- "1" PerfilRecursos
Juego "1" *-- "1..*" VersionJuego
VersionJuego "1" *-- "1..*" ComponenteJuego
Categoria "1" --> "1" VersionJuego : fija
~~~

### Invariantes

- Un concurso publicado tiene al menos una categoría, un reglamento publicado y calendario.
- Una categoría fija una versión de juego antes de aceptar partidas oficiales.
- Solo una versión de cada componente puede estar activa para nuevas partidas.
- Una partida conserva commit y digests usados aunque después cambie la versión activa.
- El reglamento usado por una partida es inmutable.
- Una resolución puede aclarar un vacío, pero nunca borrar el reglamento anterior.

## 6. Participantes, agentes y envíos

~~~mermaid
classDiagram
direction LR

class Participante {
  UUID id
  TipoParticipante tipo
  string nombrePublico
  EstadoParticipante estado
}
class MiembroParticipante {
  UUID cuentaId
  RolMiembro rol
}
class Inscripcion {
  UUID id
  EstadoInscripcion estado
  EstadoPago pago
}
class Agente {
  UUID id
  string nombre
}
class VersionAgente {
  UUID id
  int numero
  EstadoVersionAgente estado
}
class Envio {
  UUID id
  Instant recibidoEn
  EstadoEnvio estado
}
class ArtefactoAgente {
  string digest
  TipoArtefacto tipo
  int tamano
}
class Validacion {
  UUID id
  EstadoValidacion estado
  string informe
}
class SeleccionFinal {
  UUID id
  Instant confirmadaEn
}

Participante "1" *-- "1..*" MiembroParticipante
Participante "1" -- "0..*" Inscripcion
Participante "1" *-- "0..*" Agente
Agente "1" *-- "1..*" VersionAgente
VersionAgente "1" -- "1..*" Envio
Envio "1" *-- "1" ArtefactoAgente
Envio "1" *-- "0..*" Validacion
Inscripcion "1" -- "0..1" SeleccionFinal
SeleccionFinal "1" --> "1" VersionAgente
~~~

### Invariantes

- Un participante individual tiene exactamente un miembro; un equipo tiene uno o más.
- La membresía queda congelada al confirmar la inscripción.
- Cada envío es inmutable; una corrección crea otro envío o versión.
- Un envío aprobado referencia exactamente el artefacto validado por digest.
- Solo versiones aprobadas pueden seleccionarse para el cierre.
- Cada inscripción tiene como máximo una selección final vigente.
- Confirmar otra versión reemplaza la selección, no el historial, hasta la fecha de congelamiento.
- El código y modelo del agente nunca se publican sin autorización.

## 7. Partidas, resultados y replay

~~~mermaid
classDiagram
direction LR

class Partida {
  UUID id
  EstadoPartida estado
  Instant programadaPara
}
class SnapshotConfiguracion {
  string juegoDigest
  string protocoloDigest
  string reglasDigest
  string rendererDigest
  string semilla
}
class ParticipacionPartida {
  int slot
  PosicionInicial posicion
}
class EjecucionPartida {
  UUID id
  int intento
  EstadoEjecucion estado
  CausaFin causa
}
class EventoTick {
  int tick
  TipoEvento tipo
  bytes payload
}
class ResultadoPartida {
  UUID id
  EstadoResultado estado
}
class RendimientoParticipante {
  int posicion
  decimal puntuacion
}
class Estadistica {
  string clave
  decimal valor
}
class Replay {
  UUID id
  string digest
  EstadoReplay estado
}
class Incidente {
  UUID id
  OrigenFallo origen
  EstadoIncidente estado
}
class Resolucion {
  UUID id
  TipoDecision decision
  string justificacion
}

Partida "1" *-- "1" SnapshotConfiguracion
Partida "1" *-- "2..*" ParticipacionPartida
Partida "1" *-- "1..*" EjecucionPartida
EjecucionPartida "1" *-- "1..*" EventoTick
EjecucionPartida "1" --> "0..1" ResultadoPartida
ResultadoPartida "1" *-- "2..*" RendimientoParticipante
RendimientoParticipante "1" *-- "0..*" Estadistica
EjecucionPartida "1" --> "0..1" Replay
Partida "1" -- "0..*" Incidente
Incidente "1" --> "0..1" Resolucion
~~~

### Invariantes

- Una partida tiene la cantidad de slots exigida por su categoría.
- Cada slot referencia una versión de agente seleccionada o autorizada para ese tipo de partida.
- La configuración de una partida es un snapshot inmutable.
- Solo una ejecución válida puede originar el resultado vigente.
- Los intentos fallidos permanecen registrados.
- Los agentes nunca leen eventos privados de otros slots.
- Todo evento pertenece a un tick y una ejecución.
- Un resultado contiene rendimiento para todos los participantes de la partida.
- Un replay se genera solo desde un registro completo y verificado.
- Una corrección crea una resolución y una nueva versión del resultado; no sobrescribe evidencia.

## 8. Agregados y fronteras de consistencia

| Agregado | Raíz | Protege |
| --- | --- | --- |
| Cuenta | Cuenta | estado de cuenta, perfil privado y asignaciones propias |
| Concurso | Concurso | publicación, reglas, calendario y categorías |
| Participante | Participante | membresía e identidad pública |
| Inscripción | Inscripción | elegibilidad, categoría y selección final |
| Juego | Juego | publicaciones y componentes versionados |
| Agente | Agente | versiones pertenecientes al mismo participante |
| Envío | Envío | artefacto, validaciones y estado |
| Partida | Partida | slots, snapshot, intentos e incidentes |
| Resultado | ResultadoPartida | rendimientos, estadísticas y estado competitivo |
| Clasificación | Clasificación | posiciones calculadas bajo una política |

No se debe construir una transacción que modifique todos estos agregados a la vez. Los cambios entre agregados se coordinan mediante casos de uso y eventos del dominio.

## 9. Eventos del dominio

- ConcursoCreado
- ConcursoPublicado
- CalendarioReprogramado
- ReglamentoPublicado
- ResolucionDeReglaEmitida
- ParticipanteCreado
- MembresiaCongelada
- InscripcionConfirmada
- VersionJuegoPublicada
- EnvioRecibido
- ValidacionIniciada
- EnvioAprobado
- EnvioRechazado
- VersionFinalSeleccionada
- SeleccionFinalCongelada
- PartidaProgramada
- EjecucionIniciada
- TickRegistrado
- EjecucionFinalizada
- EjecucionFallida
- ResultadoProvisionalCreado
- IncidenteAbierto
- ResultadoConfirmado
- ReplayPublicado
- ClasificacionActualizada
- ConcursoFinalizado
- ConcursoArchivado

## 10. Reglas para evolucionar el diagrama

- El mapa global no excede aproximadamente veinte conceptos visibles.
- Un concepto aparece solo si tiene identidad, regla o relación relevante.
- Clases puramente técnicas no entran al modelo conceptual.
- Tablas y DTO no se confunden con entidades.
- Cada relación nueva cita un caso de uso y una respuesta AR.
- Los detalles se agregan primero a una vista local; solo suben al mapa global si ayudan a orientarse.

# Actores, permisos y casos de uso

## 1. Actores

| Actor | Objetivo | Alcance normal |
| --- | --- | --- |
| Administrador de plataforma | Mantener el sistema y resolver incidentes globales | Toda la organización |
| Miembro del comité | Crear y dirigir concursos, reglas y decisiones competitivas | Concursos asignados |
| Colaborador de juego | Diseñar, documentar, probar o revisar juegos | Juegos asignados |
| Operador de partidas | Supervisar cola, recursos, directo y fallos técnicos | Partidas asignadas |
| Árbitro | Revisar evidencia y resolver reclamos | Concurso o categoría asignada |
| Concursante | Inscribirse, preparar, enviar y seleccionar su agente | Recursos propios o de su equipo |
| Responsable de equipo | Administrar membresía antes del congelamiento | Su participante |
| Espectador | Consultar información pública y ver partidas | Solo lectura pública |
| Productor | Mostrar la vista preparada para el auditorio | Partida pública asignada |

Los roles son conjuntos de permisos. Una persona puede acumular roles, pero las capacidades de uno no se heredan por otro. En particular, ser Espectador nunca otorga arbitraje.

## 2. Modelo de autorización

Una autorización se decide con cuatro elementos:

1. **Cuenta:** quién intenta actuar.
2. **Capacidad:** qué acción del negocio intenta ejecutar.
3. **Alcance:** sobre qué organización, concurso, categoría, juego, participante o recurso propio.
4. **Condición:** en qué estado o periodo se permite.

Ejemplo: un concursante puede tener envio.crear sobre su inscripción mientras el periodo de envíos está abierto. Esto no equivale a poder insertar filas en una tabla.

## 3. Catálogo inicial de capacidades

### Plataforma

- plataforma.administrar
- cuenta.suspender
- rol.asignar
- auditoria.consultar
- respaldo.ejecutar
- incidente.tecnico.gestionar

### Concurso

- concurso.crear
- concurso.editar
- concurso.publicar
- calendario.reprogramar
- reglamento.publicar
- resolucion.emitir
- categoria.configurar
- inscripcion.evaluar
- membresia.congelar

### Juegos y agentes

- juego.crear
- juego.revisar
- juego.publicar
- envio.crear
- envio.consultar.propio
- envio.validar
- seleccion.final.confirmar
- seleccion.final.congelar

### Partidas y resultados

- partida.programar
- partida.operar
- partida.ver.publica
- evidencia.consultar.restringida
- resultado.publicar
- reclamo.crear
- reclamo.resolver
- partida.anular
- partida.repetir
- clasificacion.publicar

## 4. Matriz de permisos por defecto

| Acción | Admin | Comité | Colaborador | Operador | Árbitro | Concursante | Espectador |
| --- | :---: | :---: | :---: | :---: | :---: | :---: | :---: |
| Administrar cuentas y roles | Sí | No | No | No | No | No | No |
| Crear concurso | Sí | Sí | No | No | No | No | No |
| Publicar o cambiar calendario | Sí | Sí | No | No | No | No | Ver |
| Publicar reglamento | Sí | Sí | Colabora | No | No | Ver | Ver |
| Crear versión de juego | Sí | Autoriza | Sí | No | No | No | No |
| Revisar juego | Sí | Sí | Sí | Consulta | No | No | No |
| Gestionar participante | Sí | Evalúa | No | No | No | Propio | No |
| Crear envío | No | No | No | No | No | Propio | No |
| Ver artefacto del agente | Emergencia | Política | Validación asignada | No | Evidencia autorizada | Propio | No |
| Validar envío | Sí | Supervisa | Asignado | Opera | No | Ver informe propio | No |
| Seleccionar agente final | No | Congela | No | No | No | Propio | No |
| Programar partida | Sí | Sí | No | Opera | No | Ver | Ver pública |
| Operar ejecución | Sí | Supervisa | No | Sí | No | Ver propia | Ver pública |
| Resolver resultado | Sí | Sí | No | Evidencia | Sí | Reclama | Ver |
| Consultar auditoría | Sí | Su concurso | No | Técnica propia | Su caso | No | No |

“Emergencia” exige motivo, auditoría y alcance temporal. “Política” depende del reglamento publicado.

## 5. Catálogo de casos de uso

| ID | Caso de uso | Actor principal | Resultado |
| --- | --- | --- | --- |
| UC-001 | Registrar cuenta | Persona | Cuenta creada con perfil privado |
| UC-002 | Ingresar y cerrar sesión | Usuario | Sesión controlada |
| UC-003 | Gestionar participante individual | Concursante | Participante individual listo |
| UC-004 | Formar equipo | Responsable | Equipo con miembros elegibles |
| UC-005 | Crear concurso | Comité | Borrador administrable |
| UC-006 | Configurar categoría | Comité | Políticas y juego asociados |
| UC-007 | Publicar reglamento | Comité | Reglas inmutables disponibles |
| UC-008 | Publicar concurso | Comité | Concurso visible |
| UC-009 | Inscribir participante | Concursante | Solicitud registrada |
| UC-010 | Evaluar inscripción y pago | Comité | Inscripción confirmada |
| UC-011 | Reprogramar calendario | Comité | Nuevo horario notificado |
| UC-012 | Publicar versión de juego | Colaborador y comité | Artefactos versionados |
| UC-013 | Consultar kit del juego | Concursante | Documentación y ejemplo obtenidos |
| UC-014 | Crear agente | Concursante | Identidad del agente creada |
| UC-015 | Presentar versión | Concursante | Envío inmutable recibido |
| UC-016 | Validar envío | Sistema y revisor | Informe aprobado o rechazado |
| UC-017 | Ejecutar partida de prueba | Concursante | Evidencia de comportamiento propio |
| UC-018 | Seleccionar versión final | Concursante | Versión aprobada designada |
| UC-019 | Congelar selecciones | Comité | Participantes listos para el cierre |
| UC-020 | Programar partida | Comité | Partida con slots y configuración |
| UC-021 | Ejecutar partida oficial | Operador y sistema | Eventos completos y resultado provisional |
| UC-022 | Mostrar partida en vivo | Productor | Vista pública de una partida |
| UC-023 | Consultar resultado | Público | Resultado y estadísticas permitidas |
| UC-024 | Presentar reclamo | Concursante | Incidente competitivo abierto |
| UC-025 | Resolver incidente | Árbitro o comité | Resolución auditada |
| UC-026 | Confirmar resultado | Comité | Resultado definitivo |
| UC-027 | Actualizar clasificación | Sistema | Posiciones recalculadas |
| UC-028 | Ver replay | Público autorizado | Reproducción desde logs |
| UC-029 | Finalizar concurso | Comité | Ganadores y clasificación final |
| UC-030 | Archivar concurso | Comité | Historia disponible y protegida |
| UC-031 | Gestionar permisos | Administrador | Capacidades asignadas por alcance |
| UC-032 | Recuperar operación | Administrador | Servicio restablecido con evidencia |

## 6. Casos de uso críticos

### UC-015 — Presentar versión

**Precondiciones**

- Cuenta autenticada.
- Inscripción confirmada.
- Periodo de envíos abierto.
- Participante dentro del límite temporal de nuevos envíos.

**Flujo**

1. El concursante selecciona el agente y describe la versión.
2. Adjunta el paquete requerido por la categoría.
3. La plataforma valida estructura, tamaño y digest.
4. Guarda el artefacto fuera de la base de datos.
5. Crea un envío inmutable.
6. Encola su validación.
7. Muestra el estado y número de seguimiento.

**Resultado**

El envío queda recibido; todavía no es elegible para competir.

**Fallos**

- Formato o tamaño incorrecto: se rechaza antes de almacenar definitivamente.
- Periodo cerrado: no se crea envío.
- Transferencia incompleta: no se publica un registro parcial.

### UC-016 — Validar envío

**Precondiciones**

- Envío recibido.
- Runner permitido y perfil de recursos definidos.

**Flujo**

1. El sistema reserva capacidad.
2. Construye o prepara el artefacto en aislamiento.
3. Ejecuta comprobaciones estructurales.
4. Ejecuta pruebas de protocolo contra el juego o agente de referencia.
5. Clasifica cualquier fallo por origen.
6. Genera logs sanitizados y pasos de reproducción.
7. Aprueba o rechaza exactamente el digest examinado.

**Resultado**

El participante recibe un informe; un envío aprobado puede usarse en pruebas y selección final.

### UC-018 — Seleccionar versión final

**Precondiciones**

- Periodo de selección abierto.
- Versión perteneciente al participante.
- Envío aprobado para la categoría.

**Flujo**

1. El participante revisa sus versiones elegibles.
2. Confirma una versión.
3. La plataforma registra la selección y conserva la anterior.
4. Hasta el congelamiento puede confirmar otra versión aprobada.

**Resultado**

Existe como máximo una selección vigente por inscripción.

### UC-021 — Ejecutar partida oficial

**Precondiciones**

- Partida programada.
- Configuración y artefactos completos.
- Capacidad física disponible.
- Todas las selecciones son elegibles.

**Flujo**

1. Se crea un intento de ejecución.
2. Se verifican digests y límites.
3. Se inician juego y agentes en fronteras aisladas.
4. En cada tick el juego produce percepciones privadas.
5. Cada agente responde dentro del límite.
6. El juego valida acciones y avanza el estado.
7. Se persisten eventos por tick.
8. El juego determina la finalización o alcanza el límite.
9. Se genera resultado provisional y replay.
10. Se publican datos permitidos.

**Resultado**

La partida posee evidencia completa y un resultado provisional, o un incidente técnico que impide usar ese intento.

### UC-025 — Resolver incidente

**Precondiciones**

- Incidente abierto y evidencia preservada.
- Árbitro con alcance y sin conflicto de interés.

**Flujo**

1. Se clasifica el origen: agente, juego, plataforma o infraestructura.
2. Se revisan configuración, logs, digests y acciones.
3. Se aplica el reglamento o se emite una resolución para un vacío.
4. Se decide mantener, corregir, anular o repetir.
5. Se registra justificación y responsable.
6. Se recalcula clasificación si corresponde.

**Resultado**

La historia conserva estado anterior, decisión y efecto.

## 7. Reglas transversales

- Toda mutación tiene actor, capacidad, alcance y condición.
- Ocultar una opción en la interfaz no sustituye la autorización en backend.
- Las decisiones excepcionales no se ejecutan mediante edición directa de base de datos.
- Una intervención de emergencia crea un comando auditado.
- Un usuario nunca ve un artefacto de otro participante por pertenecer al mismo concurso.
- Los datos públicos provienen de una proyección explícita; no de exponer entidades internas.


# Flujos, estados y secuencias

## 1. Estado del concurso

~~~mermaid
stateDiagram-v2
    [*] --> Borrador
    Borrador --> Publicado: publicar reglas y calendario
    Publicado --> InscripcionAbierta: abrir inscripciones
    InscripcionAbierta --> Preparacion: cerrar inscripciones
    Preparacion --> EnCurso: inaugurar
    EnCurso --> SeleccionFinal: cerrar envios
    SeleccionFinal --> FinalEnVivo: congelar agentes
    FinalEnVivo --> Finalizado: confirmar resultados
    Finalizado --> Archivado: cerrar reclamos

    Publicado --> Cancelado: cancelar
    InscripcionAbierta --> Cancelado: cancelar
    Preparacion --> Suspendido: incidente
    EnCurso --> Suspendido: incidente
    FinalEnVivo --> Suspendido: incidente
    Suspendido --> EnCurso: reanudar
    Suspendido --> FinalEnVivo: reanudar final
    Suspendido --> Cancelado: cancelar
    Cancelado --> Archivado: documentar cierre
~~~

### Reglas

- Borrador es editable sin efecto público.
- Publicar congela la versión inicial del reglamento.
- Reprogramar un hito no cambia el estado ni el reglamento.
- EnCurso permite pruebas, mejoras y resultados no finales.
- SeleccionFinal impide nuevos envíos y permite elegir entre versiones aprobadas.
- FinalEnVivo muestra una sola partida pública a la vez.
- Archivado es terminal para operación normal.

## 2. Estado de la inscripción

~~~mermaid
stateDiagram-v2
    [*] --> Borrador
    Borrador --> Solicitada: enviar datos
    Solicitada --> EnEvaluacion: iniciar revision
    EnEvaluacion --> Confirmada: requisitos y pago validos
    EnEvaluacion --> RequiereCorreccion: faltan datos
    RequiereCorreccion --> Solicitada: corregir
    Confirmada --> Congelada: cerrar cambios
    Solicitada --> Retirada: retirar
    Confirmada --> Retirada: retirar antes del cierre
    Congelada --> Descalificada: sancion
    Retirada --> [*]
    Descalificada --> [*]
~~~

El sistema no rechaza a una persona por cupo, pero puede exigir corregir datos o ubicarla en una categoría compatible. Una descalificación posterior es una sanción competitiva y requiere resolución.

## 3. Estado del envío

~~~mermaid
stateDiagram-v2
    [*] --> Recibiendo
    Recibiendo --> Recibido: transferencia completa
    Recibiendo --> Fallido: transferencia incompleta
    Recibido --> EnCola: solicitar validacion
    EnCola --> Validando: reservar runner
    Validando --> Aprobado: todas las pruebas pasan
    Validando --> Rechazado: agente o paquete invalido
    Validando --> ValidacionInterrumpida: fallo de plataforma
    ValidacionInterrumpida --> EnCola: reintentar
    Aprobado --> Elegible: categoria vigente
    Elegible --> Seleccionado: participante confirma
    Seleccionado --> Elegible: reemplazar antes del cierre
    Seleccionado --> Congelado: cerrar seleccion
    Congelado --> Anulado: trampa comprobada
    Rechazado --> [*]
    Anulado --> [*]
~~~

Cada nuevo paquete crea otro Envío. Un envío rechazado no se edita. La selección vigente puede cambiar sin borrar selecciones anteriores.

## 4. Estado de una partida

~~~mermaid
stateDiagram-v2
    [*] --> Borrador
    Borrador --> Programada: completar slots y configuracion
    Programada --> EnCola: llega momento
    EnCola --> Preparando: reservar capacidad
    Preparando --> Ejecutando: verificar artefactos
    Ejecutando --> Procesando: juego finaliza
    Procesando --> ResultadoProvisional: registro completo
    ResultadoProvisional --> Confirmada: termina revision
    ResultadoProvisional --> BajoRevision: abrir incidente
    BajoRevision --> Confirmada: mantener resultado
    BajoRevision --> Anulada: invalidar ejecucion
    Anulada --> EnCola: programar nuevo intento

    Preparando --> FalloTecnico: infraestructura o juego
    Ejecutando --> FalloTecnico: infraestructura o juego
    Procesando --> FalloTecnico: evidencia incompleta
    FalloTecnico --> EnCola: reintentar
    FalloTecnico --> Cancelada: imposibilidad comprobada

    Confirmada --> [*]
    Cancelada --> [*]
~~~

Un timeout o acción inválida atribuible a un agente no lleva a FalloTecnico: el juego registra la consecuencia y la ejecución continúa o finaliza según su política.

## 5. Secuencia de presentación y validación

~~~mermaid
sequenceDiagram
    actor C as Concursante
    participant W as Aplicacion web
    participant A as API
    participant O as Almacen de artefactos
    participant Q as Cola
    participant V as Validador aislado
    participant J as Modulo de juego

    C->>W: Selecciona agente y paquete
    W->>A: Solicita recepcion
    A->>A: Autoriza participante y periodo
    A->>O: Guarda paquete
    O-->>A: Digest y ubicacion
    A->>A: Crea envio inmutable
    A->>Q: Encola validacion
    A-->>W: Envio recibido

    Q->>V: Reserva trabajo
    V->>O: Lee artefacto por digest
    V->>V: Verifica formato y prepara runner
    V->>J: Ejecuta prueba de protocolo
    J-->>V: Eventos y resultado de prueba
    V->>A: Publica informe sanitizado
    A-->>W: Actualiza estado
    W-->>C: Muestra aprobado o rechazado
~~~

### Garantías

- El registro se crea únicamente después de completar el artefacto.
- La validación examina el mismo digest que luego será elegible.
- Los logs no contienen secretos de otros participantes.
- Un fallo técnico puede reintentarse; un fallo del agente produce un informe nuevo.

## 6. Secuencia de ejecución por ticks

~~~mermaid
sequenceDiagram
    actor O as Operador
    participant A as API
    participant Q as Cola de partidas
    participant W as Worker
    participant G as Juego aislado
    participant X as Agente A
    participant Y as Agente B
    participant E as Registro de eventos
    participant L as Vista en vivo

    O->>A: Inicia partida programada
    A->>A: Autoriza y congela snapshot
    A->>Q: Encola partida
    Q->>W: Reserva trabajo
    W->>W: Verifica digests y recursos
    W->>G: Inicia juego con configuracion
    W->>X: Inicia artefacto A sin red
    W->>Y: Inicia artefacto B sin red

    loop Cada tick
        G-->>W: Percepcion privada por slot
        par Decisiones con presupuesto independiente
            W->>X: Percepcion A
            X-->>W: Accion A
        and
            W->>Y: Percepcion B
            Y-->>W: Accion B
        end
        W->>G: Acciones o consecuencias de timeout
        G-->>W: Eventos y nuevo estado
        W->>E: Persiste lote del tick
        E-->>L: Publica proyeccion permitida
    end

    G-->>W: Causa de finalizacion
    W->>E: Cierra registro y calcula digest
    W->>A: Resultado provisional y reporte
    A-->>L: Resultado y replay disponibles
~~~

### Garantías

- Los agentes no se comunican entre sí.
- Cada agente recibe solo su percepción.
- Los presupuestos de decisión son independientes.
- El motor del juego es la autoridad sobre acciones y finalización.
- La vista en vivo consume una proyección; no controla la partida.
- El resultado se acepta solo si el registro queda completo.

## 7. Secuencia de reclamo y resolución

~~~mermaid
sequenceDiagram
    actor C as Concursante
    participant A as API
    participant R as Arbitro
    participant E as Evidencia
    participant K as Clasificacion

    C->>A: Presenta reclamo
    A->>A: Verifica plazo y alcance
    A->>E: Inmoviliza evidencia relacionada
    A-->>R: Asigna incidente
    R->>E: Consulta logs, digests y reglas
    R->>A: Emite decision justificada
    alt Mantener resultado
        A->>A: Confirma resultado
    else Corregir
        A->>A: Crea nueva version del resultado
    else Repetir
        A->>A: Anula intento y programa otro
    end
    A->>K: Recalcula si corresponde
    A-->>C: Notifica resolucion
~~~

## 8. Flujo público del evento

1. El espectador entra sin conocer detalles técnicos.
2. Ve el concurso actual, su propósito y el próximo momento importante.
3. Abre una categoría y comprende objetivo, reglas resumidas y participantes.
4. Consulta la clasificación y evolución permitida.
5. Durante la final abre la vista en vivo preparada para proyección.
6. La interfaz destaca cambios relevantes sin revelar información privada.
7. Al terminar ve resultado, estadísticas explicadas y clasificación actualizada.
8. Después puede abrir el replay si el registro está completo y autorizado.


# UX y lenguaje visual

## 1. Dirección

La experiencia debe sentirse **clara, académica, competitiva, luminosa y cercana**. No debe parecer una tienda, una red social, un panel corporativo saturado ni una interfaz gamer oscura.

El evento es el protagonista. La personalización existe como acento —avatar, emblema, color de participante— y no como sistema de cosméticos.

## 2. Principios de interacción

1. Una pantalla tiene una tarea dominante.
2. La información más importante aparece primero; el detalle se despliega bajo demanda.
3. Un usuario no ve navegación para capacidades que no posee.
4. Ocultar navegación no reemplaza la autorización real.
5. Los estados se explican con texto, icono y color.
6. Cada error indica qué ocurrió, qué se conservó y qué puede hacerse.
7. El lenguaje usa términos del glosario y evita sinónimos técnicos.
8. Las operaciones peligrosas muestran alcance y consecuencia antes de confirmar.
9. La interfaz pública explica; la administrativa controla.
10. Todas las funciones son utilizables desde móvil, aunque la composición más densa priorice laptop.

## 3. Arquitectura de información

~~~mermaid
flowchart TD
    Inicio[Inicio publico] --> Concurso[Concurso actual]
    Inicio --> Historial[Historial]
    Concurso --> Reglas[Reglas y categorias]
    Concurso --> Clasificacion[Clasificacion]
    Concurso --> Directo[Partida en vivo]
    Concurso --> Participantes[Participantes]
    Directo --> Replay[Replay]

    Cuenta[Area autenticada] --> Panel[Mi panel]
    Panel --> MiParticipacion[Mi participacion]
    MiParticipacion --> MisAgentes[Mis agentes]
    MisAgentes --> Envios[Envios y validaciones]
    MiParticipacion --> Seleccion[Seleccion final]

    Administracion[Administracion] --> GestionConcurso[Gestion del concurso]
    Administracion --> GestionJuego[Juegos]
    Administracion --> Operacion[Operacion de partidas]
    Administracion --> Arbitraje[Incidentes]
    Administracion --> Seguridad[Cuentas y permisos]
~~~

La navegación se genera desde capacidades. Inicio público, concurso y directo permanecen separados del área administrativa.

## 4. Pantallas públicas

| Pantalla | Propósito | Contenido principal | Acción dominante |
| --- | --- | --- | --- |
| Inicio | Entender qué ocurre ahora | concurso vigente, fecha, llamada a participar o ver | Abrir concurso |
| Concurso | Comprender el evento | estado, hitos, categorías, resumen de reglas | Elegir categoría |
| Categoría | Entender una modalidad | juego, percepción, límites, participantes | Ver clasificación |
| Clasificación | Seguir evolución | posiciones, métricas explicadas, última actualización | Abrir participante |
| Participante público | Reconocer competidor | nombre público, institución opcional, resultados permitidos | Ver historial |
| Directo | Seguir una partida | arena, nombres, estado, eventos destacados | Ver partida |
| Resultado | Entender por qué terminó | posiciones, causa, estadísticas | Abrir replay |
| Replay | Explorar lo ocurrido | línea temporal, velocidad, cámara, eventos | Reproducir |
| Historial | Consultar concursos cerrados | ganadores, categorías, partidas y replays | Abrir concurso |

## 5. Pantallas del concursante

| Pantalla | Propósito | Contenido principal |
| --- | --- | --- |
| Mi panel | Saber qué hacer ahora | estado de inscripción, próximo hito, último envío |
| Mi participante | Gestionar identidad competitiva | nombre público, institución, miembros y estado |
| Inscripción | Completar requisitos | datos, categoría, pago registrado y observaciones |
| Kit del juego | Empezar a desarrollar | reglas, protocolo, límites, ejemplos y descargas |
| Mis agentes | Organizar estrategias | agentes y versiones |
| Nuevo envío | Presentar artefacto | paquete, metadatos, confirmación de digest |
| Informe de validación | Corregir un fallo | etapa, origen, logs sanitizados y reproducción |
| Partidas de prueba | Evaluar progreso | estado, resultado, estadísticas y replay propio |
| Selección final | Elegir representante | versiones aprobadas y fecha de congelamiento |
| Notificaciones | Conocer cambios | calendario, validaciones, resoluciones y alertas |

## 6. Pantallas de operación

| Pantalla | Propósito |
| --- | --- |
| Resumen operativo | Ver salud del concurso, trabajos e incidentes |
| Constructor de concurso | Configurar datos básicos y ciclo de vida |
| Categorías y reglas | Asociar juego, políticas, límites y reglamento |
| Calendario | Administrar hitos y comunicar reprogramaciones |
| Inscripciones | Evaluar datos, categoría y pago |
| Juegos | Revisar componentes, commits, digests y publicación |
| Validaciones | Supervisar cola e informes |
| Partidas | Programar slots y revisar configuración congelada |
| Directo | Preparar y controlar la única partida pública |
| Incidentes | Inspeccionar evidencia y emitir resoluciones |
| Resultados | Confirmar, corregir o anular con trazabilidad |
| Roles y permisos | Asignar capacidades por alcance |
| Auditoría | Consultar quién hizo qué, cuándo y por qué |

## 7. Dirección visual

### Personalidad

- clara y tranquila;
- técnica sin resultar fría;
- competitiva sin estética agresiva;
- moderna sin exceso de efectos;
- amigable para estudiantes;
- suficientemente sobria para una universidad.

### Paleta base propuesta

| Uso | Color | Motivo |
| --- | --- | --- |
| Fondo | #F7F8FC | Superficie luminosa y descansada |
| Superficie | #FFFFFF | Jerarquía limpia |
| Texto principal | #243047 | Contraste sin negro absoluto |
| Primario | #6574D9 | Acción y marca |
| Secundario | #64BFA5 | Progreso y éxito |
| Advertencia | #E9B760 | Atención sin estridencia |
| Peligro | #D96C7A | Fallos y acciones destructivas |
| Información | #62A8D8 | Estados informativos |

La paleta es una resolución visual derivada de AR-073 y AR-075; puede cambiar sin afectar el dominio. Los colores nunca comunican significado por sí solos.

### Tipografía

- Sans serif legible para interfaz y contenido.
- Monoespaciada únicamente para IDs, digests, logs y fragmentos técnicos.
- Escala breve y consistente; evitar títulos gigantes.
- Números tabulares en clasificación y estadísticas.

### Superficies y densidad

- Fondo claro y superficies blancas o pastel suave.
- Bordes finos; sombras solo cuando expresen elevación real.
- Espaciado suficiente entre grupos, no entre cada dato.
- Tablas para administración y ranking.
- Listas compactas para eventos, envíos y validaciones.
- Panel lateral o detalle progresivo para evidencia técnica.
- Una sola acción primaria por bloque.

## 8. Lenguaje visual de la arena

Cada juego proporciona su apariencia, pero debe respetar un contrato visual:

- cada agente tiene nombre, emblema, color y forma distinguible;
- aliados y rivales no dependen solo del color;
- el objetivo del juego permanece visible;
- peligro, daño, espera y eliminación tienen señales coherentes;
- los eventos importantes pueden destacarse sin cambiar el estado real;
- las cámaras automáticas usan datos públicos del juego;
- estadísticas detalladas no cubren la acción;
- la interfaz del auditorio elimina navegación y controles administrativos;
- el renderer nunca decide resultados.

## 9. Vista en vivo

Composición recomendada para auditorio:

1. Encabezado discreto: concurso, categoría y ronda.
2. Arena dominante.
3. Identidad y estado resumido de participantes.
4. Línea de eventos destacados.
5. Tiempo o tick y condición de victoria.
6. Resultado superpuesto únicamente al terminar.

La vista pública no muestra percepciones privadas, código, logs internos ni información que permita explotar una partida posterior.

## 10. Replay

Controles previstos:

- reproducir y pausar;
- avanzar o retroceder;
- velocidad;
- saltar a un evento destacado;
- cambiar cámara cuando el juego lo soporte;
- inspeccionar estado público del tick;
- comparar estadísticas permitidas.

Un control aparece solo si el registro contiene los datos necesarios. Si el replay es parcial, la interfaz lo declara explícitamente y desactiva funciones incompatibles.

## 11. Responsive

### Móvil

- navegación compacta por destinos principales;
- prioridad a estado actual, notificaciones, clasificación y directo;
- tablas transformadas en filas con encabezados persistentes;
- arena ajustada sin controles superpuestos;
- subida de artefactos disponible, con advertencia si el tamaño dificulta el proceso.

### Laptop

- experiencia completa;
- administración con tablas, filtros necesarios y panel de detalle;
- comparación de validaciones o resultados cuando aporte una decisión.

### Auditorio

- modo presentación sin navegación;
- tipografía y marcadores legibles a distancia;
- composición 16:9;
- reconexión automática a la proyección en vivo;
- pantalla de espera controlada cuando no hay partida pública.

## 12. Accesibilidad y estados

- Español como idioma inicial.
- Navegación completa con teclado.
- Foco visible.
- Contraste verificable.
- Movimiento reducido cuando el sistema lo solicite.
- Texto o patrón junto a todo significado de color.
- Etiquetas accesibles para controles de replay.
- Resumen textual del resultado.

Estados obligatorios por pantalla:

| Estado | Debe comunicar |
| --- | --- |
| Carga | qué se está obteniendo y si puede continuar |
| Vacío | por qué no hay elementos y cuál es la siguiente acción |
| En cola | posición o causa de espera cuando pueda conocerse |
| Error | origen conocido, efecto y recuperación |
| Sin permiso | destino seguro sin revelar datos protegidos |
| Interrumpido | qué se conservó y si habrá reintento |
| Éxito | objeto creado o cambiado y próximo paso |


# Requisitos verificables

## 1. Convenciones

- **Debe:** obligatorio para el alcance indicado.
- **Puede:** capacidad configurable u opcional.
- Los valores identificados como Parámetro deben fijarse por concurso o categoría.
- Ningún requisito autoriza una decisión técnica contraria a una regla de dominio.

## 2. Requisitos funcionales

### Identidad y acceso

| ID | Requisito | Fuente |
| --- | --- | --- |
| RF-001 | El sistema debe permitir registrar una cuenta con datos personales privados. | AR-020, AR-078 |
| RF-002 | El sistema debe permitir ingreso, cierre de sesión y recuperación controlada de acceso. | AR-072, AR-080 |
| RF-003 | El sistema debe impedir que una cuenta actúe sobre recursos ajenos sin una capacidad explícita. | AR-071, AR-080 |
| RF-004 | El administrador debe poder asignar roles y capacidades con alcance. | AR-058, AR-059 |
| RF-005 | La navegación debe ocultar destinos para los que la cuenta no posee ninguna capacidad. | AR-071, AR-072 |
| RF-006 | Toda operación administrativa sensible debe registrar actor, momento, alcance y motivo. | AR-061, AR-069 |

### Participantes e inscripciones

| ID | Requisito | Fuente |
| --- | --- | --- |
| RF-007 | Un participante debe poder ser individual o equipo. | AR-008 |
| RF-008 | Un participante puede declarar una institución académica representada. | AR-008, AR-020 |
| RF-009 | Un equipo debe tener uno o más miembros y un responsable. | AR-021 |
| RF-010 | La membresía debe quedar inmutable al congelar la inscripción. | AR-021 |
| RF-011 | El sistema debe registrar la solicitud, evaluación, categoría y estado de pago simbólico. | AR-020 |
| RF-012 | Una persona elegible no debe ser rechazada por cupo; puede requerir corrección o asignación de categoría. | AR-020 |
| RF-013 | Un participante solo puede inscribirse conforme a la política del concurso. | AR-018, AR-020 |

### Concurso

| ID | Requisito | Fuente |
| --- | --- | --- |
| RF-014 | El comité debe poder crear, configurar, publicar, suspender, finalizar y archivar concursos. | AR-009, AR-019 |
| RF-015 | Un concurso debe contener una o más categorías configurables. | AR-018, AR-022 |
| RF-016 | Una categoría debe fijar juego, percepción, memoria, recursos, puntuación y formato competitivo. | AR-030, AR-031, AR-041, AR-047 |
| RF-017 | El comité debe poder organizar fases y rondas sin confundirlas con partidas. | AR-018, AR-022 |
| RF-018 | El calendario debe poder reprogramarse y notificar a los participantes afectados. | AR-023 |
| RF-019 | El reglamento publicado debe conservarse inmutable. | AR-024 |
| RF-020 | El comité debe poder emitir una resolución pública y auditada para un vacío reglamentario. | AR-024, AR-050 |

### Juegos

| ID | Requisito | Fuente |
| --- | --- | --- |
| RF-021 | Un juego debe definir estado, percepciones privadas, acciones, avance y condiciones de finalización. | AR-025, AR-033 |
| RF-022 | Administradores y colaboradores autorizados deben poder crear, revisar y publicar juegos. | AR-026 |
| RF-023 | Una publicación de juego debe registrar commit de Git y digest de sus artefactos. | AR-027, RD-004 |
| RF-024 | Motor, protocolo, reglas, mapas, renderer y replay pueden evolucionar por separado. | AR-028 |
| RF-025 | Solo una versión de cada componente puede estar activa para nuevas partidas. | AR-027 |
| RF-026 | Una categoría debe poder usar un juego o modo de percepción diferente de otra. | AR-030 |
| RF-027 | El sistema debe proporcionar documentación y agentes de ejemplo por juego. | AR-014, AR-032 |

### Agentes y envíos

| ID | Requisito | Fuente |
| --- | --- | --- |
| RF-028 | La política debe admitir agentes algorítmicos o de aprendizaje automático si respetan límites. | AR-031 |
| RF-029 | La categoría debe declarar runners, dependencias y capacidades admitidas. | AR-034 |
| RF-030 | El participante debe poder mantener varios agentes y varias versiones. | AR-001, AR-037 |
| RF-031 | Cada envío debe pertenecer a una versión, participante y categoría. | AR-035, AR-037 |
| RF-032 | El artefacto recibido debe identificarse por un digest inmutable. | AR-035, AR-036 |
| RF-033 | Cada envío debe validarse estructuralmente y contra el protocolo del juego. | AR-032, AR-036 |
| RF-034 | Un rechazo debe incluir causa, logs sanitizados y pasos suficientes para reproducirlo. | AR-036 |
| RF-035 | Un fallo de infraestructura durante validación debe permitir reintento sin culpar al agente. | AR-042 |
| RF-036 | El participante solo puede seleccionar una versión previamente aprobada. | AR-037 |
| RF-037 | La inscripción debe tener una única selección final vigente al congelamiento. | AR-037 |
| RF-038 | Los envíos anteriores deben conservarse aunque dejen de ser la selección vigente. | AR-001, AR-037, AR-066 |

### Partidas y ejecución

| ID | Requisito | Fuente |
| --- | --- | --- |
| RF-039 | Una partida debe fijar participantes, juego, configuración, condiciones iniciales, propósito y momento. | AR-038 |
| RF-040 | La configuración usada por una partida debe conservarse como snapshot inmutable. | AR-044, RD-004 |
| RF-041 | Los agentes deben avanzar mediante ticks controlados por el juego. | AR-033, AR-039, AR-040 |
| RF-042 | Cada agente debe recibir solo la percepción correspondiente a su slot. | AR-025, AR-029 |
| RF-043 | Cada decisión debe tener un presupuesto independiente de recursos. | AR-033, AR-040, AR-041 |
| RF-044 | Una acción ausente, tardía o inválida debe transformarse en la consecuencia definida por el juego. | AR-041, AR-042 |
| RF-045 | Los agentes deben ejecutarse sin Internet y sin acceso a archivos o secretos de la plataforma. | AR-043 |
| RF-046 | Juego, agentes y plataforma deben tener fronteras que permitan atribuir fallos. | AR-042, AR-091 |
| RF-047 | Si no hay capacidad, la partida debe permanecer en cola sin iniciar parcialmente. | AR-045 |
| RF-048 | Un fallo de plataforma, juego o infraestructura puede crear un nuevo intento de la misma partida. | AR-042, AR-050, RD-010 |
| RF-049 | Un fallo propio de un agente no debe producir una repetición automática. | AR-051, RD-010 |
| RF-050 | La ejecución debe terminar por condición del juego o límite total. | AR-039 |
| RF-051 | Deben registrarse eventos suficientes por tick para reconstruir lo mostrado. | AR-029, AR-044 |

### Resultados, directo y archivo

| ID | Requisito | Fuente |
| --- | --- | --- |
| RF-052 | El resultado debe incluir todos los participantes, posiciones, puntuación, estadísticas y causa de fin permitidas. | AR-030, AR-046 |
| RF-053 | La clasificación debe aplicar la política configurada por la categoría. | AR-047 |
| RF-054 | Un empate puede compartir posición; una ronda de desempate requiere decisión del comité. | AR-047 |
| RF-055 | Todo resultado oficial nace provisional y se confirma si no existe incidente válido. | AR-050, AR-051 |
| RF-056 | Un participante debe poder presentar un reclamo bajo la política publicada. | AR-016, AR-050 |
| RF-057 | Una resolución debe poder mantener, corregir, anular o repetir sin borrar evidencia. | AR-050, AR-051, AR-066 |
| RF-058 | El sistema debe mostrar una sola partida pública en vivo a la vez. | AR-054 |
| RF-059 | La vista pública debe explicar el juego y destacar eventos interesantes sin cambiar el resultado. | AR-052, AR-053 |
| RF-060 | Un replay debe permitir únicamente controles respaldados por datos completos. | AR-055 |
| RF-061 | Código, modelos, credenciales y percepciones privadas no deben aparecer en directo, replay o resultados públicos. | AR-044, AR-082 |
| RF-062 | Deben conservarse resultados y estadísticas de todos los participantes. | AR-016, AR-030 |
| RF-063 | El concurso archivado debe conservar clasificación, partidas y replays autorizados. | AR-016, AR-051 |

### Operación

| ID | Requisito | Fuente |
| --- | --- | --- |
| RF-064 | El administrador debe disponer de un resumen de salud, colas e incidentes. | AR-058, AR-087 |
| RF-065 | El sistema debe generar reportes que separen fallos de agente, juego, plataforma e infraestructura. | AR-042, AR-087 |
| RF-066 | Las restauraciones o correcciones de emergencia deben realizarse mediante operaciones controladas y auditadas. | AR-061, AR-069 |
| RF-067 | La información crítica de partidas debe respaldarse y poder restaurarse. | AR-083 |
| RF-068 | Los registros competitivos históricos deben desactivarse o archivarse, no eliminarse físicamente. | AR-066, AR-079 |
| RF-069 | Los datos personales no necesarios deben poder anonimizarse sin destruir resultados. | AR-078, AR-079 |

## 3. Requisitos de calidad

| ID | Requisito | Verificación | Fuente |
| --- | --- | --- | --- |
| RNF-001 | Soportar aproximadamente 100 participantes en el alcance inicial. | Prueba de carga con escenario acordado | AR-006 |
| RNF-002 | La prioridad de recursos debe favorecer partidas sobre funciones secundarias. | Prueba de degradación | AR-086 |
| RNF-003 | Una falla de un agente no debe detener otro agente, el juego ni la API. | Prueba de aislamiento | AR-042, AR-091 |
| RNF-004 | Una falla del juego no debe derribar la plataforma. | Prueba de proceso fallido | AR-091 |
| RNF-005 | Los agentes no deben alcanzar Internet, host, base de datos ni artefactos ajenos. | Pruebas de escape y políticas | AR-043, AR-049 |
| RNF-006 | La plataforma debe funcionar para usuarios en Windows o Linux mediante navegador. | Matriz de compatibilidad | AR-090 |
| RNF-007 | La interfaz debe adaptarse a laptop y móvil. | Pruebas responsive | AR-076 |
| RNF-008 | Debe existir una composición 16:9 legible para auditorio. | Prueba visual a distancia | AR-012, AR-057 |
| RNF-009 | La interfaz inicial debe estar completa en español. | Revisión de cadenas | AR-077 |
| RNF-010 | La UI debe ser navegable por teclado y no depender solo de color. | Auditoría de accesibilidad | AR-075, AR-077 |
| RNF-011 | Toda partida oficial debe ser explicable mediante configuración, artefactos, eventos e informes. | Auditoría de muestra | AR-044, AR-051 |
| RNF-012 | Los artefactos y snapshots históricos deben verificarse por digest. | Prueba de integridad | AR-027, AR-044 |
| RNF-013 | El backend no debe implementarse con JavaScript. | Revisión de stack | AR-089 |
| RNF-014 | PostgreSQL debe ser el sistema de registro para datos estructurados. | Revisión de arquitectura | AR-089 |
| RNF-015 | Los límites y políticas variables no deben estar hardcodeados. | Pruebas con dos configuraciones | AR-094 |
| RNF-016 | Los módulos deben tener dependencias explícitas y no formar ciclos. | Análisis estático | AR-094 |
| RNF-017 | Cada archivo de lógica debe tener un test unitario reflejado en el árbol de pruebas. | Comprobación automatizada | AR-094 |
| RNF-018 | Los flujos críticos deben tener pruebas de integración y extremo a extremo separadas. | Ejecución de suites | AR-103, RD-008 |
| RNF-019 | La documentación y los diagramas deben actualizarse en la misma PR que cambie su comportamiento. | Lista de revisión | AR-101, AR-102 |
| RNF-020 | Compilación, pruebas y validaciones deben pasar antes de merge. | Pipeline de PR | AR-104, notas |

## 4. Valores por fijar

Los siguientes requisitos existen, pero sus objetivos numéricos permanecen como configuración o especificación pendiente:

- latencia de API;
- duración máxima de cola;
- tasa de ticks;
- timeout por decisión;
- CPU, memoria y almacenamiento por agente;
- cantidad de partidas simultáneas;
- espectadores simultáneos;
- tamaño máximo de artefacto y replay;
- retención de logs operativos;
- objetivo de recuperación y pérdida máxima de datos;
- presupuesto de infraestructura.

No deben convertirse en números arbitrarios dentro del código.


# Arquitectura técnica propuesta

## 1. Estado de estas decisiones

Las decisiones ATD-001 a ATD-015 son derivaciones profesionales, no texto literal del cuestionario. Requieren aprobación individual según AR-102. ATD-003 fue aprobada por José Daniel el 2026-09-13. ATD-015 aplica DP-001, que sí es una decisión directa y aprobada; su adaptación técnica exacta permanece abierta a revisión.

## 2. Principios

1. Monolito modular para el plano de control.
2. Procesos aislados para ejecutar juegos y agentes.
3. PostgreSQL como fuente de verdad estructurada.
4. Artefactos grandes fuera de la base de datos.
5. Contratos versionados en las fronteras.
6. Configuración publicada, no valores hardcodeados.
7. Eventos por tick como fuente del replay.
8. Permisos de negocio con alcance.
9. Historial inmutable para decisiones competitivas.
10. Pocas dependencias operativas en el MVP.
11. Organización horizontal y poco profunda del código.

## 3. Vista de contenedores

~~~mermaid
flowchart TB
    U[Usuarios] --> W[Aplicacion web]
    W --> A[API y plano de control]
    W <-->|WebSocket publico| L[Canal en vivo]

    A --> P[(PostgreSQL)]
    A --> O[(Almacen de artefactos)]
    A --> Q[(Cola persistente)]
    A --> L

    Q --> K[Worker de ejecucion]
    K --> O
    K --> S[Supervisor de sandbox]
    S --> G[Proceso de juego]
    S --> X[Sandbox de agentes]
    K --> P
    K --> L

    G -. eventos por tick .-> K
    X -. acciones limitadas .-> K
~~~

## 4. Planos y fronteras

### Plano de control

Responsable de:

- cuentas, roles y permisos;
- participantes e inscripciones;
- concursos, categorías, reglas y calendario;
- juegos y publicaciones;
- envíos y selección final;
- programación de partidas;
- resultados, clasificación, incidentes y auditoría;
- API para web.

No ejecuta código de participantes.

### Plano de ejecución

Responsable de:

- reservar trabajos;
- comprobar digests;
- preparar sandboxes;
- iniciar juego y agentes;
- aplicar límites;
- coordinar ticks;
- registrar eventos;
- clasificar fallos;
- producir evidencia técnica.

No modifica reglas, inscripciones ni permisos.

### Plano de visualización

Responsable de:

- representar eventos públicos;
- mostrar directo y replay;
- controlar cámara, velocidad y detalle;
- adaptar la experiencia a móvil, laptop y auditorio.

No calcula el estado autoritativo ni determina resultados.

## 5. Decisiones técnicas

### ATD-001 — Backend en Go

Se propone Go para API, coordinación y workers:

- cumple la prohibición de JavaScript en backend;
- permite binarios simples y despliegue controlado;
- ofrece concurrencia adecuada para colas, procesos y streaming;
- favorece código explícito y una estructura poco profunda.

No se utilizará reflexión o generación de capas para ocultar el flujo.

### ATD-002 — Frontend web en TypeScript

Se propone TypeScript con Vite y React para formularios, paneles y navegación. El visor 2D usará PixiJS como capa de render, sin convertir el motor visual en fuente de verdad.

React no define la arquitectura del backend. El frontend también usará una distribución horizontal y poco profunda, con responsabilidades reconocibles como páginas, componentes, servicios, modelos, estilos y visor.

### ATD-003 — PostgreSQL

**Estado:** aprobada por José Daniel el 2026-09-13.

PostgreSQL conserva:

- entidades y relaciones;
- estados y configuraciones;
- resultados y estadísticas indexables;
- permisos y auditoría;
- cola persistente inicial.

No almacena binarios grandes de agentes, juegos o replays.

El prototipo MySQL no se considera precedente arquitectónico. Su migración se realizará en un corte aprobado y no como cambio incidental dentro de otra funcionalidad.

### ATD-004 — Almacenamiento de objetos

Los artefactos se guardan mediante una interfaz compatible con almacenamiento local y S3:

- paquetes de agentes;
- builds;
- componentes de juego;
- logs completos;
- replays;
- exportaciones.

PostgreSQL conserva digest, tamaño, tipo, propietario lógico y ubicación opaca.

### ATD-005 — Cola en PostgreSQL para el MVP

Una tabla de trabajos con reserva transaccional evita introducir otro servidor desde el inicio. La interfaz de cola permanece separada para sustituirla si la carga real lo exige.

Tipos iniciales:

- validar envío;
- ejecutar partida de prueba;
- ejecutar partida oficial;
- generar replay;
- recalcular clasificación.

### ATD-006 — WebSocket para directo

El directo publica una proyección sanitizada de eventos por WebSocket. El registro persistente sigue siendo la fuente autoritativa; perder una conexión no pierde la partida. Un espectador que se reconecta recibe snapshot público y eventos posteriores.

### ATD-007 — Protocolo de agente por stdin/stdout

El MVP usa mensajes JSON por línea sobre entrada y salida estándar:

1. handshake de versión;
2. inicialización de slot y límites;
3. percepción de un tick;
4. acción para ese mismo tick;
5. finalización.

Ventajas:

- independencia del lenguaje;
- no requiere red;
- fácil reproducción local;
- capturable y limitable por proceso.

Cada mensaje incluye tipo, versión de protocolo, partida y tick. Una acción de otro tick es inválida. El límite de tamaño se configura.

### ATD-008 — Paquete de agente

El único formato implementado en el MVP será un archivo comprimido con:

- manifest declarativo;
- código o artefacto;
- entrypoint;
- runner solicitado;
- dependencias permitidas y fijadas;
- metadatos del agente.

El comité publica runners concretos. Python y un lenguaje compilado son candidatos para el primer concurso, pero PE-006 debe confirmarlos.

### ATD-009 — Sandbox Linux sin red

Los workers de ejecución oficiales operan sobre Linux. Cada agente se inicia:

- con usuario sin privilegios;
- sin capacidades del host;
- sin red;
- con filesystem raíz de solo lectura;
- con directorio temporal limitado;
- con CPU, memoria, procesos y tiempo limitados;
- con perfil de llamadas de sistema;
- sin montar secretos, base de datos ni artefactos ajenos.

Los contenedores OCI son empaquetado y aislamiento básico. Antes de producción debe realizarse un ejercicio de escape y considerar una barrera adicional de sandbox para código no confiable.

### ATD-010 — Juego fuera de la API

El motor del juego corre como proceso supervisado separado. Un fallo termina esa ejecución y genera un incidente; no detiene la API. El juego recibe la configuración congelada y solo comunica percepciones, validación de acciones, eventos y final de partida.

### ATD-011 — Replay autoritativo por eventos

Cada tick produce un lote ordenado:

- número de tick;
- acciones aceptadas o consecuencias;
- eventos públicos;
- cambios de estado necesarios;
- checksum acumulado.

Al finalizar se sella el log con digest. El renderer reconstruye la vista desde ese registro. Si una versión requiere snapshots completos, el juego los emite con frecuencia configurable.

### ATD-012 — RBAC con capacidades y alcance

Los roles agrupan capacidades. Las asignaciones pueden aplicar a:

- toda la plataforma;
- un concurso;
- una categoría;
- un juego;
- un participante propio.

La autorización vive en los casos de uso del backend. La interfaz usa la misma proyección de capacidades para construir navegación.

### ATD-013 — Versiones inmutables por digest

Un nombre de versión es informativo. La identidad técnica se determina por:

- commit de Git cuando existe;
- digest del artefacto;
- versión del contrato.

La partida conserva un snapshot con todos los identificadores usados. Nunca busca “la versión actual” al consultar historia.

### ATD-014 — Borrado controlado

- Dominio competitivo: estados y archivo, sin hard delete.
- Artefactos: política de retención, pero no se borran mientras una partida los referencie.
- Datos personales: anonimización o eliminación permitida si no destruye la historia.
- Logs operativos: retención limitada.
- Auditoría sensible: acceso restringido y retención definida.

### ATD-015 — Repositorio horizontal por responsabilidad técnica

**Estado:** alternativas documentadas; pendiente de confirmación. El estilo horizontal ya está decidido mediante DP-001.

El repositorio seguirá el estilo de la referencia Capibara:

- pocas carpetas y poca profundidad;
- una carpeta por responsabilidad técnica estable;
- archivos con el mismo nombre plural a través del flujo de una funcionalidad;
- `main.go` como raíz de composición, nunca como lugar de reglas;
- ninguna carpeta creada solo para anticipar una necesidad futura.

La auditoría confirmó que el prototipo sí usa `repository` para separar SQL de `service`, pero también concentra todas las funcionalidades en una única interfaz. Se consideran tres alternativas:

1. **Conservar `repository` con interfaces pequeñas por consumidor.** `service` protege reglas y autorización; `repository` contiene SQL, transacciones locales y mapeo; `connection` crea clientes técnicos. Mantiene el recorrido de DP-001 y facilita pruebas sin base real.
2. **Retirar `repository` y llevar SQL a `service`.** Reduce una delegación, pero mezcla casos de uso con persistencia y obliga a probar reglas junto a PostgreSQL.
3. **Retirar `repository` y llevar consultas a `connection`.** Mantiene SQL fuera de `service`, pero convierte una carpeta técnica en propietaria de semántica del dominio y rompe el paralelismo por funcionalidad.

La recomendación técnica es la primera alternativa. No añade otra capa: conserva la frontera que ya tiene un consumidor concreto, elimina la interfaz monolítica y crea únicamente los métodos necesarios para el corte actual. La distribución física no cambia las fronteras de ejecución: API, worker, juego y agentes siguen siendo procesos aislables.

## 6. Contrato de un módulo de juego

Cada publicación de juego declara:

| Componente | Responsabilidad |
| --- | --- |
| Manifest | Identidad, compatibilidad y digests |
| Motor | Estado autoritativo y avance por ticks |
| Esquema de percepción | Información privada entregada a cada slot |
| Esquema de acción | Acciones válidas |
| Reglas | Explicación humana y políticas |
| Mapas | Condiciones iniciales permitidas |
| Perfil de recursos | Presupuestos por agente y partida |
| Política de puntuación | Conversión de resultado a clasificación |
| Renderer | Representación de eventos |
| Esquema de replay | Eventos y snapshots necesarios |
| Agentes de referencia | Ejemplos y pruebas |

Los componentes pueden cambiar por separado, pero una VersiónJuego publicada fija el conjunto compatible.

## 7. Dependencias entre módulos

~~~mermaid
flowchart TB
    MAIN[main.go: composicion] --> SERVER[server: HTTP y WebSocket]
    MAIN --> EXEC[executor: workers]
    SERVER --> SERVICE[service: casos de uso]
    EXEC --> SERVICE
    SERVICE --> MODEL[model: dominio]
    SERVICE --> REPO[repository: persistencia]
    REPO --> CONN[connection: clientes tecnicos]
    EXEC --> GAME[game: contratos y ejecucion]
    EXEC --> REPLAY[replay: eventos y proyeccion]
~~~

Reglas:

- `server` convierte HTTP o WebSocket en llamadas de servicio; no contiene reglas ni SQL.
- `service` implementa casos de uso, autorización y transiciones; no conoce detalles de HTTP.
- `model` contiene entidades, valores y estados; no importa servidor, SQL ni infraestructura.
- `repository` contiene consultas y persistencia por funcionalidad; no decide reglas.
- `connection` crea y configura clientes, transacciones y conexiones; no contiene consultas del dominio.
- `auth` resuelve credenciales, sesiones y primitivas de autorización; los permisos concretos se verifican en `service`.
- `executor` reserva trabajos, supervisa procesos y coordina ticks; no cambia reglas competitivas.
- `game` define contratos, validación y registro de módulos de juego.
- `replay` codifica, sella y proyecta el registro autoritativo.
- `common` solo admite conceptos realmente transversales; no será una carpeta para código sin dueño.
- `cache` aparecerá únicamente cuando exista un dato concreto que justifique caché.
- `tracer` concentra logs estructurados, correlación y métricas técnicas.
- No se permiten dependencias cíclicas.

## 8. Estructura propuesta del repositorio

~~~text
agentrix/
├── Dockerfile
├── go.mod
├── go.sum
├── LICENSE
├── main.go
├── Makefile
├── README.md
├── open-api/
│   ├── openapi.yaml
│   ├── components/
│   │   ├── errors.yaml
│   │   ├── models.yaml
│   │   └── security.yaml
│   ├── auth.yaml
│   ├── contests.yaml
│   ├── categories.yaml
│   ├── participants.yaml
│   ├── games.yaml
│   ├── agents.yaml
│   ├── submissions.yaml
│   ├── matches.yaml
│   ├── results.yaml
│   ├── rankings.yaml
│   ├── replays.yaml
│   └── permissions.yaml
├── script/
│   ├── capsule.sh
│   ├── data/
│   │   ├── permissions.sql
│   │   ├── roles.sql
│   │   ├── game.sql
│   │   └── reference_agents.sql
│   └── database/
│       ├── database.sql
│       ├── tables.sql
│       ├── views.sql
│       └── user.sql
├── contracts/
│   ├── agent.schema.json
│   ├── game.schema.json
│   └── replay.schema.json
├── games/
│   └── arena-basica/
│       ├── manifest.yaml
│       ├── engine/
│       ├── renderer/
│       └── examples/
├── web/
│   ├── package.json
│   └── src/
│       ├── api/
│       ├── components/
│       ├── model/
│       ├── pages/
│       ├── service/
│       ├── style/
│       └── viewer/
└── src/
    ├── auth/
    │   └── auth.go
    ├── common/
    │   ├── code.go
    │   ├── constants.go
    │   └── error.go
    ├── config/
    │   └── config.go
    ├── connection/
    │   ├── connection.go
    │   ├── artifacts.go
    │   └── queue.go
    ├── model/
    │   ├── contest.go
    │   ├── category.go
    │   ├── participant.go
    │   ├── game.go
    │   ├── agent.go
    │   ├── submission.go
    │   ├── match.go
    │   ├── result.go
    │   ├── ranking.go
    │   └── replay.go
    ├── repository/
    │   ├── contests.go
    │   ├── participants.go
    │   ├── games.go
    │   ├── agents.go
    │   ├── submissions.go
    │   ├── matches.go
    │   ├── results.go
    │   └── rankings.go
    ├── server/
    │   ├── server.go
    │   ├── middleware.go
    │   ├── validate.go
    │   ├── contests.go
    │   ├── participants.go
    │   ├── games.go
    │   ├── agents.go
    │   ├── submissions.go
    │   ├── matches.go
    │   ├── results.go
    │   ├── rankings.go
    │   └── replays.go
    ├── service/
    │   ├── service.go
    │   ├── contests.go
    │   ├── participants.go
    │   ├── games.go
    │   ├── agents.go
    │   ├── submissions.go
    │   ├── matches.go
    │   ├── results.go
    │   ├── rankings.go
    │   └── replays.go
    ├── executor/
    │   ├── executor.go
    │   ├── worker.go
    │   └── sandbox.go
    ├── game/
    │   ├── game.go
    │   ├── registry.go
    │   └── validate.go
    ├── replay/
    │   ├── replay.go
    │   └── projector.go
    └── tracer/
        └── logger.go
~~~

### Cómo leerla

- `main.go` construye dependencias y selecciona el modo `server` o `worker`; no implementa casos de uso.
- `open-api` describe la API pública con un archivo por funcionalidad.
- `script/database` crea y evoluciona la estructura; `script/data` contiene datos iniciales explícitos.
- `contracts` contiene formatos compartidos con juegos, agentes o renderers que no pertenecen a HTTP.
- `games` contiene módulos de juego reemplazables; `src/game` contiene el código de la plataforma que los valida y ejecuta.
- `web` es una aplicación independiente, pero conserva la misma preferencia por carpetas técnicas poco profundas.
- `src` usa responsabilidades técnicas estables y archivos paralelos por funcionalidad.

El recorrido esperado de una funcionalidad es visible por nombre:

~~~text
open-api/contests.yaml
    -> src/server/contests.go
    -> src/service/contests.go
    -> src/repository/contests.go
    -> src/model/contest.go
~~~

No todas las operaciones atraviesan todas las capas. Por ejemplo, una validación puramente de dominio puede terminar en `model`, y una tarea del worker puede entrar por `executor` en vez de `server`.

Por decisión del usuario, los archivos y carpetas de prueba se omiten de este árbol para evaluar únicamente la distribución principal. La estrategia de la sección 12 permanece vigente.

No se crearán carpetas vacías “por arquitectura”. Cada una aparece cuando contiene una capacidad del corte en desarrollo.

## 9. Persistencia conceptual

| Grupo | Registros principales | Medio |
| --- | --- | --- |
| Acceso | cuentas, roles, permisos, asignaciones, sesiones | PostgreSQL |
| Concursos | concursos, reglas, hitos, categorías, fases, rondas | PostgreSQL |
| Participación | participantes, miembros, inscripciones, selecciones | PostgreSQL |
| Juegos | juegos, versiones, componentes y políticas | PostgreSQL + objetos |
| Agentes | agentes, versiones, envíos, validaciones | PostgreSQL + objetos |
| Partidas | partidas, slots, intentos, snapshots, incidentes | PostgreSQL |
| Eventos | índices y digests; log completo por ticks | PostgreSQL + objetos |
| Resultados | rendimientos, estadísticas, clasificaciones, resoluciones | PostgreSQL |
| Operación | trabajos, auditoría, reportes | PostgreSQL + objetos |

## 10. Consistencia y transacciones

Una transacción local protege un agregado. Los procesos largos usan estados:

- recibir artefacto antes de crear el envío;
- confirmar el registro antes de encolar;
- reservar un trabajo con un único propietario y lease;
- publicar resultado solo después de sellar el log;
- actualizar clasificación a partir de un resultado confirmado;
- reintentar operaciones idempotentes con una clave estable.

Nunca se mantiene una transacción de base de datos mientras un agente o juego se ejecuta.

## 11. Fallos y recuperación

| Origen | Ejemplo | Efecto |
| --- | --- | --- |
| Agente | timeout, crash, acción inválida | consecuencia competitiva configurada |
| Juego | excepción, estado imposible | ejecución inválida e incidente |
| Worker | proceso muerto, host reiniciado | lease expira y trabajo se recupera |
| Plataforma | API no disponible | partida en worker continúa y sincroniza después |
| Base de datos | conexión temporal perdida | buffers limitados; no se confirma resultado sin persistencia |
| Directo | WebSocket interrumpido | partida sigue; espectador reconecta |
| Replay | log incompleto | replay no se publica |

La prioridad de recuperación es: ejecución, evidencia, resultado, directo y funciones administrativas.

## 12. Estrategia de pruebas

### Unitarias

- árbol separado que refleja cada archivo de lógica;
- una responsabilidad y un sujeto claramente nombrados;
- sin base de datos, red o contenedores;
- cubren invariantes, estados y políticas.

### Integración

- PostgreSQL real para restricciones y transacciones;
- almacenamiento de artefactos;
- reserva y recuperación de cola;
- contratos entre worker, juego y agente;
- clasificación desde resultados.

### Extremo a extremo

- ingreso;
- inscripción;
- envío y validación;
- selección final;
- partida completa;
- directo y replay;
- reclamo y resolución.

### Seguridad

- acceso cruzado entre participantes;
- elevación de permisos;
- escapes de sandbox;
- red y filesystem prohibidos;
- artefactos maliciosos;
- saturación de salida o procesos.

### Beta

Usuarios reales validan comprensión, documentación, ergonomía y comportamiento del evento. No sustituyen tests automatizados.

## 13. Decisiones explícitamente aplazadas

- proveedor o modo definitivo de hosting;
- tecnología adicional de sandbox;
- runners del primer concurso;
- tasa de ticks y límites;
- formato definitivo del replay;
- SLO cuantitativo;
- servicio externo de correo;
- procesamiento digital de pagos;
- analítica de producto.

Estas decisiones tienen dependencias reales en PE-001 a PE-008.

# MVP y hoja de ruta

## 1. Objetivo del MVP

Demostrar, de extremo a extremo, que una persona puede ingresar, presentar un agente, recibir validación, ejecutar una partida segura y observar su resultado y replay.

El MVP no intenta organizar todavía una final universitaria completa. Comprueba el riesgo central antes de añadir amplitud.

## 2. Recorrido vertical

1. Un administrador prepara un concurso de demostración.
2. El concurso tiene una categoría y un juego básico por ticks.
3. Existen una cuenta administradora y una cuenta concursante.
4. El concursante consulta el contrato del juego.
5. Presenta un paquete de agente.
6. El sistema almacena su digest y lo valida en aislamiento.
7. El participante recibe un informe reproducible.
8. Si está aprobado, solicita o recibe una partida contra un agente de referencia.
9. La partida espera hasta disponer de capacidad.
10. El worker ejecuta juego y agentes sin red.
11. Se registran eventos por tick.
12. La partida termina sin comprometer la plataforma.
13. El sistema muestra resultado, estadísticas básicas y replay.

## 3. Alcance funcional del MVP

### Incluido

- registro o provisión controlada de cuentas;
- ingreso y cierre de sesión;
- roles Administrador y Concursante;
- un concurso de demostración;
- una categoría;
- un juego de arena básico;
- un protocolo de agente;
- un runner interpretado y uno compilado, sujetos a PE-006;
- paquete comprimido con manifest;
- almacenamiento de artefactos;
- validación estructural y de protocolo;
- cola persistente;
- ejecución aislada;
- límites de CPU, memoria, procesos, salida y tiempo;
- percepción privada por slot;
- acciones por ticks;
- agente de referencia;
- resultado provisional y confirmación automática sin incidentes;
- log sellado por ticks;
- replay posterior a la partida;
- vista pública simple;
- informe técnico para el propietario;
- auditoría de acciones sensibles.

### Excluido

- equipos;
- pago automatizado;
- múltiples categorías;
- constructor general de formatos competitivos;
- fase final completa;
- arbitraje avanzado;
- visión computacional;
- múltiples juegos activos;
- edición visual de permisos;
- personalización de perfil;
- transmisión y clips;
- soporte de múltiples organizaciones;
- aplicación nativa móvil;
- escalamiento horizontal automático.

El modelo conceptual conserva espacio para esas capacidades; el código del MVP no crea abstracciones sin uso.

## 4. Juego de demostración

El primer juego debe ser deliberadamente pequeño:

- arena 2D en cuadrícula o coordenadas discretas;
- dos agentes;
- percepción parcial;
- movimiento y una interacción simple;
- obstáculos o apariciones controladas;
- duración máxima;
- puntuación sencilla;
- eventos visuales claros;
- semilla registrada;
- estado suficiente para replay.

Su misión no es ser el juego definitivo. Debe probar contrato, aislamiento, ticks, azar, registro y visualización.

## 5. Criterios de aceptación del corte vertical

| ID | Criterio |
| --- | --- |
| CA-001 | Un usuario sin permiso no puede presentar un agente por otro participante. |
| CA-002 | Un paquete incompleto no crea un envío elegible. |
| CA-003 | El envío aprobado y el ejecutado tienen el mismo digest. |
| CA-004 | Un agente no puede acceder a Internet, host, base de datos ni rival. |
| CA-005 | El timeout de un agente no bloquea el tick indefinidamente. |
| CA-006 | El crash de un agente produce una consecuencia y un reporte atribuible. |
| CA-007 | El crash del juego invalida el intento sin derribar la API. |
| CA-008 | Reiniciar un worker permite recuperar un trabajo abandonado. |
| CA-009 | Cada tick del replay conserva orden y checksum. |
| CA-010 | El resultado contiene ambos participantes y una causa de finalización. |
| CA-011 | La vista puede reproducir la partida sin ejecutar nuevamente los agentes. |
| CA-012 | Perder la conexión del navegador no interrumpe la ejecución. |
| CA-013 | Los logs visibles al participante no revelan información privada del rival. |
| CA-014 | El recorrido completo tiene prueba automatizada. |
| CA-015 | Una persona nueva puede ejecutarlo siguiendo la documentación. |

## 6. Etapas

### Etapa 0 — Experimentos de riesgo

**Objetivo:** comprobar las fronteras antes de construir la plataforma.

- proceso de agente por stdin/stdout;
- presupuesto por tick;
- aislamiento sin red;
- terminación forzada;
- juego básico;
- log y reproducción desde ticks.

**Salida:** prototipo desechable con informe. No comparte código con producción salvo decisión posterior explícita.

### Etapa 1 — Esqueleto del repositorio

**Objetivo:** demostrar compilación, pruebas y dependencias.

- estructura mínima;
- comandos API y worker;
- configuración tipada;
- migraciones;
- checks de formato, compilación y tests;
- documentación de ejecución local.

No se crean todos los módulos por anticipado.

### Etapa 2 — Acceso mínimo

**Objetivo:** distinguir administrador y propietario de agente.

- Cuenta;
- credencial;
- sesión;
- roles fijos iniciales implementados sobre el modelo de capacidades;
- autorización de recursos propios;
- auditoría básica.

No incluye editor general de permisos.

### Etapa 3 — Concurso y juego de demostración

**Objetivo:** disponer de una configuración publicable.

- Concurso;
- Categoría;
- VersiónJuego;
- PolíticaCategoria;
- PerfilRecursos;
- agente de referencia;
- documentación del protocolo.

### Etapa 4 — Agente, envío y validación

**Objetivo:** transformar un paquete en versión aprobada o informe útil.

- Agente;
- VersionAgente;
- Envío;
- Artefacto;
- Validación;
- cola de validación;
- informe reproducible.

### Etapa 5 — Partida

**Objetivo:** ejecutar de forma segura y atribuible.

- Partida;
- slots;
- snapshot;
- worker;
- sandboxes;
- ticks;
- clasificación de fallos;
- resultado provisional.

### Etapa 6 — Resultado y replay

**Objetivo:** hacer visible el valor.

- eventos sellados;
- estadísticas básicas;
- resultado;
- replay;
- vista pública;
- reconexión.

### Etapa 7 — Endurecimiento y beta

**Objetivo:** entregar el MVP a usuarios reales.

- pruebas de seguridad;
- pruebas de carga con el escenario inicial;
- recuperación de jobs;
- respaldo;
- accesibilidad;
- documentación para concursantes;
- sesión beta;
- correcciones antes de ampliar alcance.

## 7. Orden posterior al MVP

1. Inscripciones reales y equipos.
2. Selección final y congelamiento.
3. Fases, rondas y clasificación configurable.
4. Final en vivo y vista de auditorio.
5. Incidentes, reclamos y resoluciones.
6. Múltiples categorías.
7. Nuevos juegos y runners.
8. Personalización ligera.
9. Estadísticas avanzadas.
10. Visión computacional, solo después de un experimento exitoso.

## 8. Riesgos y experimentos

| Riesgo | Experimento | Señal de éxito |
| --- | --- | --- |
| El protocolo es demasiado lento | 2 agentes durante miles de ticks | Presupuesto estable sin acumulación |
| El aislamiento es insuficiente | Suite de agente hostil | No alcanza recursos prohibidos |
| Los logs son demasiado grandes | Partida larga con máximo de eventos | Replay dentro de almacenamiento aceptable |
| El replay diverge | Reproducir varias veces el mismo log | Checksums visuales y estado final iguales |
| Un modelo ML no cabe | Agente de prueba con modelo realista | Inicio y tick dentro del perfil |
| El público no comprende | Sesión con estudiantes no técnicos | Explican objetivo y causa de fin |
| El informe no ayuda | Fallos sembrados en paquetes | Participante corrige sin acceso interno |
| La cola duplica trabajos | Caída de worker en puntos críticos | Un solo resultado vigente |

## 9. Definición de terminado por capacidad

Una capacidad está terminada cuando:

- el comportamiento y las invariantes están documentados;
- el caso de uso tiene criterios de aceptación;
- el código compila;
- los unit tests correspondientes pasan;
- las integraciones reales necesarias pasan;
- las autorizaciones negativas están probadas;
- los estados de UI relevantes existen;
- logs y errores permiten diagnosticarla;
- no contiene secretos ni valores variables hardcodeados;
- el diff fue revisado;
- la PR explica qué decisiones implementa;
- José Daniel aprueba el resultado.

## 10. Regla para Codex

Cada solicitud de implementación debe incluir:

1. un solo caso de uso o experimento;
2. documentos e IDs que lo justifican;
3. archivos previstos y motivo;
4. invariantes;
5. pruebas;
6. exclusiones;
7. verificación final.

Codex no debe generar la plataforma completa en una sola solicitud, crear carpetas futuras vacías, hacer commit o push sin autorización, ni introducir una abstracción sin un consumidor actual.


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
- [ ] Definir el próximo corte antes de abrir una rama. Etapa 0 y el flujo público de concursos están aplazados por ahora.

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

# Cuestionario maestro para definir Agentrix desde cero

> Este documento parte de una hoja en blanco. No presupone que deban conservarse el producto, la arquitectura, las entidades, los flujos ni las tecnologias de intentos anteriores.

## Como responder

1. Escribe debajo de **Respuesta** con toda la extension que necesites.
2. Comienza cada respuesta con uno de estos estados:
   - `[DECIDIDA]`: quieres que esta respuesta forme parte del diseño.
   - `[PROVISIONAL]`: es tu preferencia actual, pero puede depender de otras decisiones.
   - `[PENDIENTE]`: aun no tienes informacion suficiente.
   - `[NO APLICA]`: no corresponde al sistema; explica brevemente por que.
3. No necesitas responder en orden estricto, pero las secciones estan organizadas por dependencia.
4. Si una pregunta menciona algo que ya definiste, referencia su identificador —por ejemplo, `AR-018`— en lugar de repetir la respuesta.
5. Separa lo que deseas de lo que estas dispuesto a aceptar como compromiso.
6. Describe ejemplos concretos cuando una regla pueda interpretarse de varias maneras.

## Principios del cuestionario

- Cada pregunta cubre una decision diferente.
- Las preguntas tecnicas aparecen despues de las preguntas del producto y del dominio.
- Una pregunta no obliga a aceptar la existencia del concepto que menciona.
- Las cuestiones condicionales pueden marcarse como `[NO APLICA]`.
- Las contradicciones no se resolveran silenciosamente: se registraran y volveran como decisiones explicitas.
- Las respuestas seran la fuente para producir los diagramas; los diagramas no inventaran respuestas.

## Que se obtendra de las respuestas

| Secciones | Resultado derivado |
| --- | --- |
| 1–3 | Vision, actores y experiencia completa |
| 4–10 | Casos de uso, reglas, permisos y flujos |
| 11 | Glosario y diagrama conceptual de clases |
| 12 | Arquitectura de informacion y direccion visual |
| 13–15 | Requisitos de datos, calidad y arquitectura tecnica |
| 16–17 | Alcance, hoja de ruta y proceso de construccion |
| 18 | Revision de coherencia y asuntos pendientes |

---

# 1. Identidad, proposito y limites

## AR-001 — Definicion esencial

Si tuvieras que explicar Agentrix a una persona que nunca ha oido hablar del proyecto, ¿que dirias que es en una sola frase y que aclararias inmediatamente despues para evitar una interpretacion equivocada?

**Respuesta:**

concurso de agentes donde cada participante programa su propio agente para competir contra otros agentes, la idea es que al participante se le permita subir su propio agente que el considere, donde este pueda subir varios como mejoras a su anterior, otro nuevo completamente distinto con un limite de tiempo para poder subir el siguiente, donde este se enfreta a un cantidad limita por prueba a otros agentes que posen su propia logica, estos se encuentran en entornos que pueden ir cambiando en cada prueba, y estas partidas como tal pueden estar involucradas por factores aletorios que pondran a prueba la logica de lo agentes, no es algo que se puede decir deterministico, si no donde los mejores agentes casi siempre van a estar en el top, entonces para que sea mas emocionante entra el factor suerte que hace mas emocionante pero a la vez no afecta tanto

## AR-002 — Problema u oportunidad

¿Que problema real, necesidad u oportunidad origina el proyecto? Describe que ocurre hoy sin Agentrix, quien lo padece y por que merece construirse una solucion.

**Respuesta:**

el problema no hay una plataforma que pueda gestionar toda esta idea, la necesidad de no encontrar alternativas que cumplan con las espectativas para esto, la oportunidad que nace es que podemos hacer que este hecho a nuestra medida y exigencias, un control total sobre todo, 

## AR-003 — Valor para sus beneficiarios

¿Que cambio valioso debe producir Agentrix para cada grupo que se beneficie de el? Distingue beneficios practicos, educativos, competitivos, sociales y de entretenimiento cuando correspondan.

**Respuesta:**

el poder organizar concursos dentro de la universidad donde se puede mostrar desde un punto de vista distinto todo esto de los modelos de ia, agentes y ya no verlos solo como simples predictores de datos si no que tambien pueden ser aplicados a entornos de entretenimiento como los juegos entornos que pongan a prueba el entrenamiento de cada uno de los modelos y puedan mejoras sus enfoques cambios y estrategias

## AR-004 — Prioridad del producto

Si hubiera conflicto entre organizar competencias, evaluar agentes con rigor, facilitar el aprendizaje y crear un espectaculo atractivo, ¿como ordenarias esas prioridades y que sacrificios no aceptarias?

**Respuesta:**

evaluar agentes con rigor, facilitar el aprendizaje, crear un espectaculo atractivo, organizar competencias . prefiero realmente que la parte tecnica y logica salga perfecta sacrificando todo lo demas

## AR-005 — Principios no negociables

¿Que cualidades o principios debe respetar el sistema incluso si encarecen o retrasan su desarrollo? Explica por que cada uno es imprescindible.

**Respuesta:**

el poder gestionar toda la plataforma de manera que no haya conflictos logicos, ahora el sistema que va a ser capaz de manejar cualquier juego que se desee y donde cada jugador sea manejado por un agente o ia, y este sea eficiente y sencillo, lograr proporcionar todas la herramientas a los agentes para puedan desarrollarse sin poblemas dentro de los juegos y partidas sin errores y problemas

## AR-006 — Fuera de alcance

¿Que productos, funciones, publicos o problemas se parecen a Agentrix, pero deseas excluir deliberadamente? Incluye aquello que no quieres que el proyecto llegue a convertirse.

**Respuesta:**

como tal no se busca hacer una superplataforma capaz de albergar concuros gigantes lleno de muchas cosas, si que solo cumpla con los objetivos como para unas 100 participantes

---

# 2. Actores, responsabilidades e intereses

## AR-007 — Mapa de actores

¿Que tipos de personas, grupos u organizaciones interactuan con el sistema o son afectados por el? Para cada actor, indica su objetivo principal y si actua dentro o fuera de la plataforma.

**Respuesta:**

en su gran parte son estudiante de ingenieria
administradores: tiene control absoluto sobre el sistema
concursantes: solo tiene deciciones sobre el en el concurso
espectadores: solo puede ver como es que se estan desarrollando las cosas pero no puede interferir en nada

## AR-008 — Unidad participante

¿Quien compite realmente: una persona, un equipo, una institucion, un agente o alguna combinacion? Explica como se representa publicamente y quien responde por sus acciones.

**Respuesta:**

como tal todo esto depende de el concurso puede haber concurso donde la participacion es individual o en equipo, ademas estos pueden representar a alguna institucion academica

## AR-009 — Autoridad organizadora

¿Quien puede crear y dirigir una competencia, que decisiones puede tomar y ante quien responde? Distingue organizacion academica, administracion tecnica y autoridad sobre el reglamento si son responsabilidades diferentes.

**Respuesta:**

el cominite organizar con respecto todo aquello que requiero toma de deciones sobre un concurso y sus bases

## AR-010 — Creadores y operadores especializados

¿Se necesitan roles separados para diseñar juegos, redactar reglas, revisar agentes, operar ejecuciones, arbitrar resultados o producir la transmision? Define que responsabilidad no deberia combinarse en una sola persona.

**Respuesta:**

admin: diseñar juegos, redactar reglas, revisar agentes, operar ejecuciones, arbitrar resultados o producir la transmision
colaborador: diseñar juegos, redactar reglas, revisar agentes
espectador: arbitrar resultados o producir la transmision

## AR-011 — Publico y partes externas

¿Que clases de espectadores, patrocinadores, jurados, docentes, instituciones u otras partes externas deben considerarse? Indica que pueden observar, aportar, exigir o decidir.

**Respuesta:**

todos pueden observar, pero solo el comite organizador puede tomar las deciciones sobre aportar, exigir o decidir.

---

# 3. Experiencia completa del evento

## AR-012 — Historia principal de extremo a extremo

Narra un caso ideal desde que se anuncia una competencia hasta que se reconoce al ganador y se conserva su historia. Evita terminos tecnicos; describe personas, decisiones, acontecimientos y resultados.

**Respuesta:**

el ultimo de el evento recien se conocera a el ganador y esto sera en vivo en un entorno preparado para este momento un escenario nuevo frente a los mejores participantes durante el torneo y de ahi recien damos desarrollo en vivo a toda esta experiencia, asi nadie sabe quien va a ganar ni los admins ni el comite y nadie, en ese momento se conoce el desenlace, frente a todo el publico en pleno auditorio

## AR-013 — Descubrimiento y entrada

¿Como descubre una persona la plataforma o una competencia, que necesita entender antes de participar y que pasos debe completar para comenzar?

**Respuesta:**

se va a realizar publicidad por toda la unversidad y se proporciona las bases para concer mas sobre la edicion de este evento

## AR-014 — Preparacion previa

¿Que hacen organizadores, participantes y responsables del juego antes del inicio? Define que debe estar listo, que se comprueba y quien autoriza el paso a la siguiente etapa.

**Respuesta:**

debe de estar listo todo el entorno para dar inicio a el evento, tanto el juego como los agentes de prueba que porcionan los organizadores, los participantes deben de ir probando durante el concurso en cualquier momento de la duracion sus modelos, los responsables deben de vigilar que nada se rompa o corregir reportar los fallos

## AR-015 — Jornada o periodo competitivo

¿Que sucede durante la competencia desde la perspectiva de cada actor? Describe el ritmo, los puntos de decision, la comunicacion y aquello que hace reconocible el momento central del evento.

**Respuesta:**

los administradores se encargan de todo lo tecnico viendo que nada falle, los partipantes prueban y mejoran sus modelos, los espectadores tambien pueden ver como se va desarrollando que cambios a habido

## AR-016 — Cierre y continuidad

¿Que debe suceder despues de la ultima partida? Incluye publicacion de resultados, reconocimientos, reclamos, acceso historico, aprendizaje y preparacion de futuras ediciones.

**Respuesta:**
puede verse los resultados atravez de los dias y como han ido cambiando, el dia de el cierre se debe de tener algo mas pomposo y llamativo, donde al terminar la partida final se queden los tops de el concurso y ademas poder acceder alos datos y modelos de el concursos, tambien dar sugerencias y reclamos

## AR-017 — Frontera entre plataforma y mundo real

¿Que partes de la experiencia deben ocurrir necesariamente dentro de Agentrix y cuales pueden seguir siendo manuales o externas? Explica que informacion debe regresar al sistema cuando algo sucede fuera de el.

**Respuesta:**

el moverse por la plataforma como los menus opciones personalizacion de perfil revisar cosas ocurren fuera de el alcance de los agentes o ias, los modelos solo pueden reaccionar y funcionar dentro de ese entorno de el juego al inciar y terminar la partida, luego no pueden hacer nada y simplemente se guadan los datos y resultados 

---

# 4. Competencias, ediciones y participacion

## AR-018 — Conceptos organizativos

¿Que significan para ti terminos como competencia, concurso, torneo, evento, edicion, temporada, categoria, fase y ronda? Define cuales existen, cuales son sinonimos y como se contienen entre si.

**Respuesta:**

competencia: a el ranking en general durante el desarrollo los detalles de este y como se va desarrollando, tecnicamente engloba todo. 
concurso , torneo, evento, edicion, temporada : serian lo mismo presentado a la competencia como tal mas iria como el nombre y aspectos mas simples .
categoria: separacion entre participantes donde es decidida por el comite.
fase: cambios de la competencia durante el desarrollo de el concurso y representar etapas donde se determinan modificaciones por el comite.
ronda: a la partida o la separacion como las llaves en un torneo

## AR-019 — Ciclo de vida de una competencia

¿Por que estados atraviesa una competencia desde su idea inicial hasta su archivo? Para cada transicion importante, indica quien la provoca y que condicion debe cumplirse.

**Respuesta:**

todo los hacen los administradores inaguracion donde se muestra de que va esta edicion y una demostracion de como es que funciona y que se puede hacer con un ejemplo de prueba en vivo, durante la competencia la subida de modelos y los resultados de este, para el cierre la revelacion en vivo, todo esto ocurren en tiempos establecidos y personalizables en el sistema 

## AR-020 — Elegibilidad e inscripcion

¿Quien puede participar, que requisitos debe demostrar, como solicita o confirma su inscripcion y en que circunstancias puede ser aceptado, rechazado o retirado?

**Respuesta:**

sus datos personales como nombre y apellido universidad, se incribe y realiza un pago simbolico y despues es evaluado para ser puesto en alguna categoria correspondiente, no se rechaza a nadie a participar

## AR-021 — Formacion y cambio de equipos

**Condicional:** si existen equipos, ¿como se crean, quien los representa, como ingresan o salen miembros y que cambios se permiten despues de inscribirse o comenzar la competencia?

**Respuesta:**

en la incripcion forman el equipo inmutable por el resto de el concurso siempre y cuando los particpantes cumplan con los requisitos para participar, todo estos participantes son asignados a el id de el equipo

## AR-022 — Estructura competitiva

¿Que formatos debe poder expresar el sistema —por ejemplo, exhibicion, liga, grupos o eliminacion— y que elementos son comunes a todos? No elijas todavia un algoritmo; define la flexibilidad necesaria.

**Respuesta:**

todos pueden esos aspectos, mas no modificadorlos

## AR-023 — Calendario y coordinacion

¿Como se determinan fechas, rondas, emparejamientos y disponibilidad? Describe que puede cambiar, quien recibe avisos y que ocurre ante retrasos o ausencias.

**Respuesta:**

todo puede cambiar y se les notifica a los participantes en la plataforma

## AR-024 — Publicacion y cambio de reglas

¿Cuando se conocen y quedan fijadas las reglas de una competencia? Define como se corrigen errores, como se comunican los cambios y que proteccion reciben quienes ya desarrollaron un agente bajo una version anterior.

**Respuesta:**

las reglas quedan fijadas y ya si hay vacios legales son a decicion de el comite

---

# 5. El juego como concepto

## AR-025 — Definicion de juego

¿Que es un juego dentro de Agentrix y que debe proporcionar para ser utilizable? Distingue sus reglas, estado, acciones, condiciones de finalizacion y representacion observable.

**Respuesta:**

como tal este debe de proporcionar los datos o parametros que persive solo el jugador y no el de los demas, por el momento el mismo juego proporciona esos datos, se planea estar preparado para modelos de vision computacional por el momento

## AR-026 — Creacion y publicacion de juegos

¿Quien puede diseñar, implementar, revisar, aprobar, publicar, retirar o reemplazar un juego? Describe el recorrido desde una idea hasta un juego disponible para competencias.

**Respuesta:**

los administradores son responsables de todo eso,deben de haber todos esos procesos con el flujo estandar y clasico

## AR-027 — Identidad y evolucion de un juego

¿Que cambios hacen que siga siendo el mismo juego y cuales crean una version o un juego diferente? Explica que compatibilidad esperas entre agentes, partidas, resultados y visualizaciones de distintas versiones.

**Respuesta:**

solo la versionamos por github osea solo queda la actual

## AR-028 — Partes que componen un juego

¿Reglas, protocolo de comunicacion, motor, mapas, recursos visuales, visor y formato de replay forman una unidad inseparable o pueden evolucionar por separado? Define el comportamiento deseado, no la implementacion.

**Respuesta:**

pueden ir cambiando por separado

## AR-029 — Aleatoriedad y conocimiento del entorno

¿Que papel puede tener el azar? Indica que conocen los agentes antes y durante una partida, que informacion permanece oculta y que debe poder reproducirse posteriormente.

**Respuesta:**

al azar solo influye en aspectos mapas distintos, espawn de mounstrous pero estos como tal no aparecen de nada si no en zonas q no afecten a los jugadores de manera improvista, como tal los agentes solo pueden persivir el entorno solo desde su punto de vista no pueden tener un conocimiento total, todo esto haria q sea difil hacer un repeticion deterministica, por lo que seria mas logico un sistema de hacer las repeticiones por un guardados de logs guardados por tics y con esto poder reproducir el mismo escenario

## AR-030 — Reutilizacion y combinacion de juegos

¿Un juego puede usarse en multiples competencias o una competencia puede incluir multiples juegos? Explica como deberian compararse o conservarse sus resultados sin asumir que todos puntuan igual.

**Respuesta:**

si es posible, podria darse el caso que haya un categoria que si o si pide modelos por vision computacional y otro que se le entregan todos los parametros y en base a todo eso decir de manera mas facil, es decir una para personas no tienen tanta experiencia con estos temas, o tal  vez el juego de una categoria no puede ser reproducido para otra en base a las dificultades entonces se plantea otro, ademas con respecto a las puntuaciones deben ser guadadas todas las estadisticas de todos los participantes no se puede dejar a nadie fuera

---

# 6. Agentes, desarrollo y presentacion

## AR-031 — Definicion y limites de un agente

¿Que consideras un agente valido? Aclara si puede usar reglas escritas a mano, busqueda, aprendizaje automatico, modelos externos, memoria, intervencion humana u otros recursos.

**Respuesta:**

si todo eso es posible si se encuentra dentro de los limites establecidos para juego, puede que haya competidores que quieren participar con algortimos programados o otros que decidan entrenar su modelo con machine learning, si se permite tener memoria tambien se implementa o se activa esa opcion en el concurso

## AR-032 — Experiencia de creacion

¿Que recorrido deberia seguir una persona desde que conoce un juego hasta que logra ejecutar su primer agente valido? Describe documentacion, ejemplos, herramientas y ayuda que esperaria recibir.

**Respuesta:**

se deberia de ofrecer toda la documentacion sobre como se usa la plataforma y dentro de cada concurso se los detalles correspondientes a los juegos o juego ofrecido, con las limitaciones que se debe de cumplir, un sistema que consiga validar aspectos generales sobre sus agentes para ser considerado un envio valido 

## AR-033 — Contrato de interaccion con el juego

¿Que informacion recibe un agente, en que momento, que respuesta debe producir y como se interpreta? Incluye inicializacion, acciones sucesivas, finalizacion y errores de comunicacion.

**Respuesta:**

recibe todas las percepciones que se encuentran los limites de el juego, todo esto por tics, de manera que cada no haya el problema de que un agente ocupo mas recursos para decidir que otro y los demas se vean afectados, entonces se calcula todo lo correspondiente a cada tic de manera que nadie queda fuera y no se rompe nada

## AR-034 — Entornos y dependencias admitidas

¿Que lenguajes, versiones, bibliotecas, modelos, archivos o servicios externos deben permitirse? Explica si la plataforma debe ser abierta a muchos entornos o comenzar con un conjunto limitado.

**Respuesta:**

como tal siempre va a estar asociado a un conjunto limitado por el comite, entonces si luego de un revision de el modelo y no cumple los requisitos puede ser anulado el envio y dar como no valido

## AR-035 — Objeto presentado

¿Que entrega exactamente el participante: codigo, ejecutable, paquete, contenedor, repositorio, endpoint u otra cosa? Indica metadatos, responsable, licencia, visibilidad y evidencia de autoria necesarias.

**Respuesta:**

todo esto lo determina el comite, pero las opciones y el soporte para ofrecer todo esto debe de estar disponible, osea tiene que ser personalizable y configurable

## AR-036 — Validacion y retroalimentacion

¿Que comprobaciones debe superar una presentacion antes de competir y que informacion recibe el participante cuando alguna falla? Distingue errores corregibles de causas de rechazo definitivo.

**Respuesta:**

se le debe de proporcionar toda esa informacion sobre que paso y poder reproducir ese error y solucionarlo, los logs de el fallo o el motivo por cual se considero el fallo

## AR-037 — Versiones, plazos y conservacion

¿Cuantas veces puede presentarse un agente, cual version compite, cuando queda congelada y que se conserva despues? Incluye retiros, reemplazos y visibilidad historica.

**Respuesta:**

todos los envios quedan sugetos a un participante y este puede considerar participar con su mejor peor o regurlar version, si el lo considera, pero solo puede participar con un modelo el dia de el cierre, para tiene que confirmar con cual decide participar mientras este sea un modelo aprobado o probado durante la competencia, no se puede participar con modelo que nunca paso por ahi, entonces no se admiten modelos sorpresa

---

# 7. Partidas, simulacion y ejecucion

## AR-038 — Definicion de partida

¿Que elementos hacen que una partida sea una unidad identificable? Incluye participantes, juego, configuracion, condiciones iniciales, momento, proposito y resultado esperado.

**Respuesta:**

si incluye todo eso

## AR-039 — Ciclo de vida de una partida

¿Que estados atraviesa una partida desde que se propone hasta que su resultado queda confirmado? Define quien o que inicia cada transicion y cuales permiten volver atras.

**Respuesta:**

un tiempo limite por partida para evitar bucles, los agentes toman deciciones por tics, todo inicia cuando el tiempo comienza a correr y este termina o ya no hay participantes en la partida

## AR-040 — Modelo temporal

¿La interaccion ocurre por turnos, pasos simultaneos, tiempo real u otro modelo? Describe como avanza el juego y como se evita que la velocidad del hardware otorgue una ventaja injusta.

**Respuesta:**

todo esta controlado por tics y en un tiempo real definido

## AR-041 — Limites de recursos

¿Que limites deben aplicarse a tiempo de respuesta, CPU, memoria, almacenamiento, tamaño, duracion total u otros recursos? Explica si dependen del juego, la competencia o la infraestructura disponible.

**Respuesta:**

como tal se debe de dar limites bien definidos para los recursos que puede consumir cada agente no pueden pasar de ese limite si no estos mismos se veran perjudicados en el momento de la evaluacion

## AR-042 — Fallos y acciones invalidas

¿Que resultado produce un timeout, bloqueo, caida, salida incorrecta, accion ilegal o abandono? Distingue fallos del agente, del juego, de la plataforma y de la infraestructura.

**Respuesta:**

se deben se parar esos tres aspectos para poder ofrecer la mejor experiencia y no ocurra problemas, asi de esta manera tambien encontrar de manera mas sencilla el problema

## AR-043 — Aislamiento y capacidades

¿Que puede leer, escribir o contactar un agente mientras se ejecuta? Define las capacidades necesarias y las prohibidas respecto a red, archivos, procesos, secretos y otros participantes.

**Respuesta:**

tienen que estar controladas en un entorno donde no pueden acceder a internet archivos de el juego y informacion con respecto a la partida que no deberian de conocer

## AR-044 — Registro y reproducibilidad

¿Que hechos deben registrarse para explicar, auditar o reproducir una partida? Indica que grado de igualdad esperas al repetirla y que informacion puede mantenerse privada.

**Respuesta:**

se puede proporcionar todos los datos, pero no modelos ni codigo simplemente datos recolectados y generados durante la partida

## AR-045 — Colas, capacidad y reintentos

¿Como deberia comportarse el sistema cuando hay mas partidas que capacidad disponible? Define prioridades, espera aceptable, reintentos y el tratamiento de ejecuciones duplicadas o interrumpidas.

**Respuesta:**

se debe de poner en cola de espera hasta que las condiciones fisicas sean consideradas aptas para que pueda iniciar la partida correspondiente en la cola

---

# 8. Resultados, clasificacion y justicia

## AR-046 — Resultado de una partida

¿Que produce una partida ademas de un ganador o perdedor? Define puntuaciones, estadisticas, causas de finalizacion y evidencia necesaria para interpretar el resultado.

**Respuesta:**

se proporciona todo eso mencionado mientras no sean modelos o codigo de otros participantes

## AR-047 — Clasificacion y desempate

¿Como se agregan resultados para ordenar participantes y como se resuelven empates? Explica que debe poder variar entre competencias sin fijar todavia una formula universal.

**Respuesta:**

se hace en  base a los parametros de calificacion para cada juego, en caso de empate se coloca a ambos en la misma posicion y un posible desempate en caso no se llegue a un acuerdo entre competidores

## AR-048 — Condiciones comparables

¿Como se garantiza que los participantes enfrenten condiciones suficientemente equivalentes? Considera posiciones, mapas, rivales, semillas, orden, cantidad de partidas y variacion estadistica.

**Respuesta:**

se les da un tiempo considerable para poder preparse y elegir estrategias y mejoras que hagan mas sencillo acortar las brechas, ademas no hay factores de suerte que hagan conseguir ventajas injustas si no en base mas que todo a las deciciones de cada agente, cada escenario es evaluado y preparado para evitar esos problemas

## AR-049 — Integridad competitiva

¿Que conductas constituyen trampa, colusion, explotacion indebida o ventaja prohibida? Define que se previene automaticamente, que se investiga y que evidencia se necesita.

**Respuesta:**

cualquier cosa que vulnere informacion personal o acceso a modelos de otros participantes, modelos muy potentes que no cumplan con las limitaciones inpuestas en el concurso

## AR-050 — Reclamos y repeticion de partidas

¿Quien puede cuestionar un resultado, dentro de que plazo y mediante que proceso? Define cuando procede corregir, repetir, anular o mantener una partida.

**Respuesta:**

los administradores que posiblemente tambien sean parte de el comite

## AR-051 — Finalidad e historial de resultados

¿Cuando se considera definitivo un resultado y que puede cambiar despues? Explica como se muestran correcciones, sanciones o recalificaciones sin reescribir silenciosamente la historia.

**Respuesta:**

se considera definitivo cada resultado mientras no hay trampas o errores que pueden haber afectado los resultados y a otros participantes, mientras los errores que un mismo participante provoco y solo afecten a el y no a otros, no se realiza una reevaluacion, mientras no pase nada  de eso se considera definitivo

---

# 9. Experiencia publica, directo y replay

## AR-052 — Experiencia del espectador

¿Que deberia poder comprender y sentir un espectador que nunca ha programado? Describe que hace interesante seguir una partida y que informacion seria ruido innecesario.

**Respuesta:**

entender calaramente lo que trata el juego y en base a esto difrustar como los agentes van actuando y tomando deciones atravez de los agentes y poder mostrar claramente las escenas que se consideren interesantes osea tener un sistema capaz de detectar y darle mas enfoque a esas partes

## AR-053 — Informacion durante una partida

¿Que debe mostrarse en vivo: arena, participantes, decisiones, estado, estadisticas, eventos destacados u otros datos? Ordena la informacion por importancia.

**Respuesta:**

la partida en general y algunos datos que se consideren interesantes

## AR-054 — Directo, retraso y spoilers

¿La visualizacion debe ser realmente en vivo, retrasada o publicada al terminar? Define como afectan la integridad competitiva, los problemas de conexion y las distintas audiencias.

**Respuesta:**

en vivo una solo partida a la vez

## AR-055 — Replay

¿Que debe permitir una repeticion: pausar, acelerar, retroceder, cambiar camara, inspeccionar estados, ocultar informacion o comparar agentes? Define tambien quien puede verla y desde cuando.

**Respuesta:**

si puede permitir todo eso, pero solo en el caso que toda la informacion completa no se puede permitir opciones que no tengan toda la informacion completa para hacer eso y no romper nada en el proceso

## AR-056 — Perfiles, clasificaciones e historia publica

¿Que informacion publica debe existir sobre participantes, equipos, agentes, juegos, competencias y resultados anteriores? Aclara que informacion no debe convertirse en perfil publico.

**Respuesta:**

se debe de permitir todo eso pero, siempre y cuando este en los limites establecidos durante el concurso

## AR-057 — Produccion y difusion

¿La plataforma debe apoyar narracion, comentarios, pantallas para el recinto, transmision, clips, enlaces compartibles o integracion en otros sitios? Prioriza los usos reales esperados.

**Respuesta:**

solo mostrar la partida para ser compartida en la pantalla principal de el auditorio no se requiere algo tan complejo

---

# 10. Administracion, moderacion y operacion humana

## AR-058 — Consola de organizacion

¿Que necesita observar y controlar un organizador antes, durante y despues de una competencia? Separa configuracion normal de acciones excepcionales o peligrosas.

**Respuesta:**

los permisos se espera manejar permisos dinamicos por participantes administradores moderadores y todo eso, asi es mas facil administrar quienes pueden hacer o no las cosas

## AR-059 — Permisos

¿Que decisiones o datos requieren permisos diferentes? Construye una primera matriz indicando actor, accion permitida, alcance y condiciones especiales.

**Respuesta:**

con los permisos dimanicos con respecto a los cruds asignados a cada entidad o tabla de el modelo relacional

## AR-060 — Moderacion y sanciones

¿Que contenidos o conductas pueden moderarse, quien decide y que sanciones existen? Incluye advertencia, ocultamiento, descalificacion, suspension y apelacion cuando correspondan.

**Respuesta:**

en el caso de trampas o fallos tecnicos

## AR-061 — Intervencion excepcional

¿Que acciones manuales deben existir para resolver incidentes sin corromper el historial? Define que confirmaciones, justificaciones y registros exige cada intervencion.

**Respuesta:**

backups y hacer todo con operacion en espacios controlados

## AR-062 — Alcance organizacional y soporte

¿Agentrix sirve a una sola organizacion o a varias organizaciones independientes? Explica quien administra la plataforma global, quien brinda soporte y como se aislan responsabilidades y datos.

**Respuesta:**

ahora sirve solo para una y la administro yo como autor de la ide y desarrollador

---

# 11. Modelo conceptual y diagrama de clases

> Esta seccion consolida conceptos descubiertos en las secciones anteriores. Referencia respuestas previas en vez de volver a narrarlas.

## AR-063 — Lenguaje comun

¿Cual es el termino oficial para cada concepto del dominio y que terminos alternativos deben evitarse? Señala palabras que hoy sean ambiguas o tengan significados diferentes segun el actor.

**Respuesta:**

no nos compliquemos tanto mientras se entienda todo normal

## AR-064 — Entidades con identidad propia

¿Que conceptos deben poder reconocerse como el mismo objeto a lo largo del tiempo aunque cambien sus datos? Para cada uno, indica que lo identifica y cuando nace o deja de existir.

**Respuesta:**

todo esta presentado en el modelo relacional y de que dependen cada uno y como es que se deben de relacionar

## AR-065 — Valores sin identidad

¿Que conceptos se definen completamente por su valor y podrian reemplazarse por otro equivalente? Considera puntajes, periodos, limites, posiciones, configuraciones y otros valores compuestos.

**Respuesta:**

todo viene definido por el modelo relacional y si es que una oracion expresa correctemente la idea principal no hay problemas

## AR-066 — Pertenencia y composicion

¿Que objetos existen por si mismos y cuales solo tienen sentido como parte de otro? Indica que deberia ocurrir con los componentes cuando su propietario se archiva, elimina o reemplaza.

**Respuesta:**

nada se debe borrar por eso usamos una columna de active para determinar ese tipo de situaciones, borrar o eliminar no es una opcion, remplazar si lo es pero solo actualizando valores modificables que no rompan el modelo relacional

## AR-067 — Relaciones y cardinalidades

Para las entidades identificadas, ¿que relaciones son obligatorias, opcionales, uno a uno, uno a muchos o muchos a muchos? Describe tambien relaciones cuyo significado cambie con el tiempo.

**Respuesta:**

todas estas pueden exisitir y como tal se da por las relaciones que hay entre ellas por un id como claves primarias foraneas y todo eso 

## AR-068 — Invariantes del dominio

¿Que afirmaciones deben ser siempre verdaderas para que el sistema siga siendo coherente? Vincula cada regla con las entidades responsables de protegerla.

**Respuesta:**

validar cada cosa que se da va a establecer verificar los datos y cambios a realizar, validando todo y cada uno de los atributos

## AR-069 — Tiempo, versiones e historia

¿Que relaciones o atributos representan el estado actual y cuales deben conservar el estado que era valido en un momento historico? Indica donde una referencia viva seria peligrosa.

**Respuesta:**

todo debe de estar guardado y respaldado en logs en caso sean cambios muy importantes que solo se puedan hacer por medio de la manipulacion directa de la base de datos

## AR-070 — Eventos del dominio y vistas del modelo

¿Que acontecimientos importantes deberian tener nombre propio porque cambian el estado o interesan a varios actores? Despues, indica que grupos de entidades deberian aparecer juntos en el mapa global y cuales necesitan un diagrama separado.

**Respuesta:**

tecnicamente todo deberia de estar conectado, simplemente deben de estar aisalado la manipulacion de la plataforma por medio de una partida, y la manipulacion de una partida por medio de la plataforma

---

# 12. Experiencia de usuario y lenguaje visual

## AR-071 — Arquitectura de informacion

¿Como deberia organizarse la informacion desde la perspectiva de cada actor? Define los destinos principales de navegacion y que concepto sirve como punto de entrada a los demas.

**Respuesta:**

los administradores deberia poder admirar toda la informacion y los participantes solo aquella que le corresponda, todo esto con respecto a los permisos dinamicos con respecto a que pueden ver editar actualizar crear y cambiar

## AR-072 — Pantallas y tareas criticas

¿Que pantallas son necesarias para completar los recorridos esenciales? Para cada una, indica la tarea principal, informacion imprescindible y accion dominante.

**Respuesta:**

hay una pantalla principal al hacer login, y luego se puede visualizar todo aquello a lo que se tiene permiso, todo esto separado por secciones donde se agrupan las cosas relacionadas, si no tiene permisos no se puede ver esa informacion, como no tendria ni conocimiento de que existen esas opciones

## AR-073 — Personalidad visual

¿Que adjetivos deben describir la plataforma y que sensaciones debe evitar? Proporciona referencias visuales, productos, eventos o estilos que ayuden a reconocer ambos extremos.

**Respuesta:**

solo centracer en el concurso en ofrecer opciones completas pero sencillas, no ofrecer cosas como venda productos y ese tipo de cosas, se puede dar una pequeña libertad de personalizacion en ciertos aspectos esticos que hagan sentir un tanto especial al usuario

## AR-074 — Jerarquia y densidad

¿Que informacion debe destacar inmediatamente y cual puede revelarse bajo demanda? Explica el nivel de densidad adecuado para publico, participantes y organizadores.

**Respuesta:**

darle importancia al evento mas que todo y como participar y todos los detalles de el concurso correspondiente todo bien organizado y sin saturar de informacion a los usuarios

## AR-075 — Lenguaje visual de la arena

¿Como deben representarse agentes, equipos, acciones, terreno, peligro, progreso, daño, eventos y resultado para comprender la partida sin depender unicamente de texto o color?

**Respuesta:**

depende de el tipo de concurso y juego, pero todo esto debe de ser facil de entender usar colores y temas sencillos y pasteles, osea temas claros, lo oscuro deprime y cansa

## AR-076 — Dispositivos y contextos de uso

¿Donde se usara la plataforma: laptop, movil, proyector, pantalla del evento, transmision u otros contextos? Prioriza tamaños, controles y condiciones de red que el diseño debe soportar.

**Respuesta:**

en dispositivos como latop y moviles, buscar que como tal sea responsive todo

## AR-077 — Accesibilidad, idioma y estados de interfaz

¿Que necesidades de accesibilidad, idiomas, formatos regionales y diferencias culturales deben atenderse? Define ademas como deberian comunicarse carga, vacio, error, espera, bloqueo y exito.

**Respuesta:**

por momento accesible para todos, y el idioma por el momento seria en español, las interfaces todas dependen de los permisos dinamicos establecidos

---

# 13. Datos, identidad, seguridad y privacidad

## AR-078 — Inventario y propiedad de datos

¿Que datos necesita realmente el sistema, quien es responsable de cada conjunto y que nivel de sensibilidad tiene? Distingue datos personales, competitivos, tecnicos, publicos y operativos.

**Respuesta:**

los datos personales como tal son privados y no seran publicos en ningun momento, por eso la seguridad esta dado por un sistema de login y cuentas por usuario

## AR-079 — Conservacion, eliminacion y anonimizacion

¿Durante cuanto tiempo debe conservarse cada clase de informacion y que significa eliminarla? Explica que historia debe permanecer, que puede anonimizarse y que obligaciones tiene el sistema ante una solicitud de borrado.

**Respuesta:**

se conserva solo la informacion que no afecte el desarrollo de la competencia para futuras ediciones o concursos, lo demas que no represente romper el sistema puede ser borrado y anonimato si es que se requiere

## AR-080 — Requisitos de identidad y acceso

¿Que necesita demostrar una persona para realizar acciones sensibles? Define requisitos de registro, verificacion, ingreso, recuperacion, cierre de sesion y revocacion sin decidir todavia el mecanismo tecnico.

**Respuesta:**

solo se puede actuar sobre la informacion de cada cuenta personal, no sobre otras personas

## AR-081 — Amenazas y abusos relevantes

¿De quien y de que debe protegerse la plataforma? Considera suplantacion, robo de cuenta, codigo hostil, fuga de agentes, manipulacion de resultados, saturacion y abuso administrativo.

**Respuesta:**

no va a ser una superplataforma es algo mas sencillo y pequeño para eventos universitarios

## AR-082 — Secretos y propiedad intelectual

¿Que codigo, modelos, credenciales, estrategias o datos deben mantenerse secretos, incluso frente a organizadores u otros participantes? Define cuando y bajo que autorizacion pueden revelarse.

**Respuesta:**

todo gestionado por medio de los permisos dinamicos

## AR-083 — Trazabilidad, respaldo y obligaciones externas

¿Que informacion debe poder exportarse, respaldarse, restaurarse o auditarse? Incluye requisitos institucionales, legales, de consentimiento o proteccion de menores que puedan aplicar.

**Respuesta:**

mas que todo la informacion correspondiente a las partidas esos datos los importantes a ser respaldados

---

# 14. Requisitos de calidad

## AR-084 — Escala esperada

¿Cuantas organizaciones, competencias, participantes, agentes, partidas simultaneas, espectadores y años de historia esperas en una primera edicion y en un escenario exitoso posterior?

**Respuesta:**

una vez cada 4 meses organizados por la universidad 

## AR-085 — Rendimiento perceptible

¿Que operaciones necesitan respuesta inmediata y cuales pueden tardar o ejecutarse en segundo plano? Define tiempos aceptables desde la experiencia del usuario, no desde una tecnologia concreta.

**Respuesta:**

todo definido por los cruds y los persmisos dinamicos

## AR-086 — Disponibilidad y degradacion

¿En que momentos no puede fallar el sistema y que funciones pueden degradarse temporalmente? Describe que experiencia debe mantenerse si falla la visualizacion, ejecucion, almacenamiento o conexion.

**Respuesta:**

tiene que priorizarse las partidas los demas aspectos pueden quedar atraz, el poder mantener todo funcionando como los modelos y el juego es lo primordial

## AR-087 — Observacion y diagnostico

¿Que necesita conocer el equipo para detectar, explicar y resolver un problema? Distingue informacion util para participantes, organizadores, desarrolladores y operacion.

**Respuesta:**

un sistema de reporte sobre lo que sucedio

## AR-088 — Restricciones de costo y mantenimiento

¿Que limites existen sobre presupuesto, infraestructura, tiempo de administracion, consumo energetico o dependencia de proveedores? Define quien podra mantener el sistema y con que conocimientos.

**Respuesta:**

lo necesario para que funcione todo

---

# 15. Restricciones y decisiones de arquitectura

> Responde esta seccion desde las necesidades ya definidas. Una tecnologia no debe elegirse unicamente por familiaridad o popularidad.

## AR-089 — Tecnologias obligatorias, preferidas o prohibidas

¿Existe alguna restriccion real sobre lenguajes, frameworks, bases de datos, sistemas operativos, licencias o servicios? Para cada una, explica su origen y si es obligacion, preferencia o rechazo.

**Respuesta:**

como tal no pero evitar usar javascripts en backend y mas apego para postgre, lo demas puede ser cualquier cosa que funcione

## AR-090 — Entornos de ejecucion y despliegue

¿Donde debe poder desarrollarse, probarse y operar la plataforma? Incluye equipos disponibles, sistema operativo, red, infraestructura del evento y posibilidad o imposibilidad de usar servicios externos.

**Respuesta:**

en dispositivos con windows y linux, como tal no busca limitar nada 

## AR-091 — Fronteras tecnicas necesarias

¿Que partes necesitan aislamiento por seguridad, fallos, carga, propiedad o ritmo de cambio? Explica cuales podrian convivir inicialmente aunque sean conceptos diferentes.

**Respuesta:**

la cosa es buscar que si se rompre un juego solo afecte a el juego y rompa el servidor ni la plataforma, en el caso de romper la plataforma que este se restablesca y se puede recuperar parte de lo afectado, tambien aislar todo con referente a la base datos

## AR-092 — Persistencia y almacenamiento

¿Que tipos de informacion requieren consultas estructuradas, archivos grandes, transmision temporal, cache o almacenamiento inmutable? Indica durabilidad y recuperacion esperadas antes de elegir productos concretos.

**Respuesta:**

estructurar de manera correcta y bien hecho todo aquello que afecte a otras partes y esos flujos sean los correctos

## AR-093 — Contratos e integraciones

¿Que interfaces necesitan participantes, juegos, visor, administradores o sistemas externos? Define consumidores, estabilidad, autenticacion y politica de cambio esperada para cada contrato.

**Respuesta:**

por el momento todo eso esta bien, tampoco se busca hacer una superplataforma

## AR-094 — Modularidad y legibilidad del codigo

¿Que reglas deberian hacer que la estructura del proyecto sea facil de recorrer y que el codigo resulte agradable de leer? Incluye profundidad de carpetas, nombres, tamaño de componentes, abstracciones, dependencias y documentacion.

**Respuesta:**

que todo esto separado de manera clara y no combinar cosas con respecto a todo eso, evitar en  todo el sentido el harcodeo, mientras todo sea capaz de ser entendible y legible sin logica rara y confusa, todo bien estrucuctura, nada de mesclar archivos con test, ademas hacer un test por cada archivo donde este solo debe de poner a prueba las funciones y logica de el archivo original y nada mas, nada de pruebas generales y mal estructuradas, la documentacion igual

---

# 16. Primera version y evolucion

## AR-095 — Demostracion minima del valor central

¿Cual es la experiencia mas pequeña que demostraria que Agentrix merece existir? Debe poder observarse de principio a fin y no limitarse a infraestructura interna.

**Respuesta:**

debe ser facil de entender y ir a lo que viene, concursos donde los participantes ponen sus modeles a competir en un concurso y apreciar el proceso 

## AR-096 — Resultado minimo por actor

Para cada actor necesario en la primera version, ¿que unico resultado debe poder conseguir? Elimina temporalmente a los actores que no sean indispensables para demostrar el valor central.

**Respuesta:**

poder llevar a hacer una partida con agentes y que esta termine, no importa los resultados, con un sistema de login y cuentas para hacer todo eso

## AR-097 — Primer recorrido vertical

¿Que secuencia completa implementaremos primero, desde una accion visible hasta un resultado visible? Indica datos de entrada, decisiones, fallos basicos y evidencia de exito.

**Respuesta:**

la plataforma funcional con login y cuentas, y poder hacer un envio de un agente y inicie la partida y el agente pueda actuar sin problemas, no importa si el jeugo es sencillo o muy basico, simplemente llegar a ese punto

## AR-098 — Exclusiones de la primera version

¿Que funciones importantes se aplazaran conscientemente y que solucion manual o limitada se usara mientras tanto? Explica por que su ausencia no invalida la demostracion.

**Respuesta:**

toda aquello que se un complemente y no precindible para del desarrollo de una partida con agentes

## AR-099 — Riesgos que requieren experimento

¿Que supuestos tecnicos o de experiencia podrian invalidar el proyecto si fueran falsos? Define el experimento mas pequeño para comprobar cada uno antes de construir alrededor suyo.

**Respuesta:**

que los agentes no puedan desemvolverse de la manera esperada

## AR-100 — Camino de crecimiento

¿Que capacidades deberian añadirse despues de la primera version y que compatibilidad historica o de extensiones debe preservarse desde el inicio? Distingue evolucion probable de posibilidades meramente hipoteticas.

**Respuesta:**

el resto de funcionalidades y personalizacion, refuerzo en partes criticas y mas opciones como las estadisticas y mejoras con respecto a la experiencia

---

# 17. Forma de construir y controlar el proyecto

## AR-101 — Fuentes oficiales de verdad

¿Que documentos o modelos seran autoritativos para producto, dominio, arquitectura, interfaz, contratos y decisiones? Define como detectar que codigo y documentacion se contradicen.

**Respuesta:**

si el codigo no hace lo mismo que la documentacion o tambien puede ser al revez, dado ciertos puntos de funcionamiento esperado

## AR-102 — Decisiones y cambios

¿Quien confirma una decision, como se propone un cambio y como se conserva su justificacion? Define que modificaciones requieren actualizar diagramas, casos de uso o criterios de aceptacion antes de programar.

**Respuesta:**

yo confirmo todo eso, y debe de ser validado de la manera mas profesional posible

## AR-103 — Estrategia de pruebas y aceptacion

¿Que comportamientos deben demostrarse con pruebas y quien acepta que una capacidad esta correcta? Incluye dominio, integracion, ejecucion de agentes, seguridad, interfaz y reproduccion de partidas.

**Respuesta:**

todo es se optiene al provado por usuarios beta y testers reales que validen que todo esta bien

## AR-104 — Flujo de versiones y entregas

¿Como se organizaran ramas, revisiones, commits, versiones, entornos y despliegues? Define que acciones nunca deben automatizarse sin aprobacion explicita.

**Respuesta:**

segun se conveniente pero cada cambio en ramas para luego abrir una PR y con esto validar que es correcto para el merge

## AR-105 — Legibilidad y transferencia de conocimiento

¿Que deberia poder comprender una persona nueva durante su primer dia con el proyecto? Define la documentacion, recorridos guiados y estandares necesarios para que el conocimiento no dependa de una conversacion previa.

**Respuesta:**

todo eso, ya que al ser algo no tan convencional puede causar problemas el no entender todo aquello que quiero revisar y modificar

---

# 18. Revision final de coherencia

> Responde esta seccion al terminar las anteriores. Su proposito no es agregar alcance, sino descubrir contradicciones, vacios y decisiones sin fundamento.

## AR-106 — Sintesis verificable

Resume el sistema en un parrafo indicando quien obtiene que valor, mediante que experiencia y bajo que reglas esenciales. ¿Todas las secciones anteriores pueden reconocerse en esta sintesis?

**Respuesta:**

un sistema capaz de gestionar y alberar un concurso donde hay agentes que controlan a jugadores y estos reciben y actuan segun el entorno dado, buscando cumplir el objetivo trazado

## AR-107 — Trazabilidad del modelo

Revisa cada actor, caso de uso, entidad y pantalla propuesta: ¿que objetivo justifica su existencia y que respuesta de este cuestionario la respalda? Enumera elementos sin justificacion o respuestas sin representacion.

**Respuesta:**

si cubre una necesidad o proposito funcional esta justificado

## AR-108 — Conflictos y prioridades

¿Que respuestas compiten por tiempo, simplicidad, justicia, seguridad, espectaculo, flexibilidad o costo? Para cada conflicto, decide que principio domina y que consecuencia aceptas.

**Respuesta:**

tiempo seguridad y flexibilidad, lo demas puede esperar realmente 

## AR-109 — Responsabilidades y pasos ocultos

¿Existe alguna accion necesaria que no tenga responsable, permiso, entrada, salida o tratamiento de error? Incluye procesos manuales y dependencias externas.

**Respuesta:**

si no la tiene no se implementa y no deberia de existir

## AR-110 — Escenarios adversos

¿Que ocurre cuando usuarios se equivocan, actuan maliciosamente, llegan tarde, pierden conexion, presentan datos contradictorios o cuando la infraestructura falla en el peor momento?

**Respuesta:**

se toman las medidas correspondientes por parte de los encargados

## AR-111 — Preparacion para diseñar e implementar

¿Que preguntas continuan abiertas y cuales impiden crear el diagrama de clases conceptual, definir el MVP o elegir la arquitectura? Establece la condicion concreta para declarar cerrada la etapa de descubrimiento.

**Respuesta:**

tener un esqueleto simplificado de que si o si se tiene que tener, y que otras pueden esperar

---

# Espacio para notas transversales

Usa este espacio para observaciones que afecten varias preguntas. Referencia siempre los identificadores relacionados para que la nota pueda resolverse posteriormente.

**Notas:**

que todo salga bien y trabajar con cosas que si compilan y no simplemente hacer codigo

