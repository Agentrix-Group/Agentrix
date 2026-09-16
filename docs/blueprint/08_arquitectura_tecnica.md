# Arquitectura técnica propuesta

## 1. Estado de estas decisiones

Las decisiones ATD-001 a ATD-016 son derivaciones profesionales, no texto literal del cuestionario. Requieren aprobación individual según AR-102. ATD-003 fue aprobada por José Daniel el 2026-09-13. ATD-015 aplica DP-001, que sí es una decisión directa y aprobada; su adaptación técnica exacta permanece abierta a revisión. ATD-016 fue aprobada por José Daniel el 2026-09-15.

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

#### Supervisor de ejecución (Worker en Go)
Responsable de:
- iniciar y detener el subproceso del motor;
- iniciar y aislar los procesos no confiables de los agentes;
- aplicar límites de tiempo, memoria y llamadas a los agentes;
- enviar percepciones y recibir acciones;
- validar mensajes estructuralmente;
- detectar timeout, crash y violaciones del protocolo de los agentes;
- entregar las acciones válidas al motor;
- registrar fallos como evidencia técnica;
- recolectar eventos, hashes y resultados del motor;
- distinguir fallos del motor de fallos de los agentes.

#### Motor de simulación (Rust + Bevy + Rapier)
Responsable de:
- mantener el estado autoritativo del mundo;
- procesar ticks con timestep fijo en modo headless;
- aplicar reglas de juego físicas y espaciales;
- resolver físicas y colisiones con Rapier;
- calcular percepciones por slot;
- emitir eventos de simulación por tick;
- calcular el resultado competitivo y hashes de estado deterministas;
- proporcionar información suficiente para el replay sin ejecutar agentes directamente.

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

### ATD-010 — Motor de juego oficial externo en Rust (Bevy + Rapier)

**Estado:** formalizada mediante ADR-001 y DP-003.

El motor de juego corre como un ejecutable externo independiente en Rust (utilizando Bevy para el mundo y los ticks, y Rapier para la física y colisiones espaciales), operando en modo headless y fuera de la API y de la base de datos de Agentrix.

Principios del motor:
- El motor es autoritativo respecto a la simulación y el resultado competitivo.
- La comunicación se efectúa mediante el protocolo versionado `agentrix-engine/1` en JSON Lines sobre `stdin` y `stdout` (reservando `stdout` exclusivamente para mensajes de protocolo y `stderr` para logs y diagnóstico).
- Los agentes participantes no corren dentro del motor; son procesos no confiables supervisados por Agentrix fuera de la simulación.
- Un fallo del motor termina la ejecución aislada y genera un incidente; no detiene la API ni corrompe el estado de Agentrix.
- La implementación embebida en Go `ArenaBasicaEngine` fue provisional y queda reemplazada por la abstracción de cliente de subproceso.

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

### ATD-016 — Consola operativa serena y eventos estructurados

**Estado:** aprobada por José Daniel el 2026-09-15.

`tracer` ofrece dos presentaciones del mismo evento operativo:

- consola local compacta, en español, sin caller ni stack trace automáticos;
- JSON estructurado en producción, con nombre estable de evento y campos completos.

Cada evento declara nivel, subsistema, nombre técnico en inglés, mensaje humano y campos tipados. La correlación puede incluir `request_id`, `actor_id`, `job_id`, `match_id` y `attempt`; la consola acorta identificadores y el JSON conserva sus valores completos.

Política de niveles:

- `DEBUG`: consultas, rechazos esperables y diagnóstico solicitado;
- `INFO`: inicio, disponibilidad y transiciones operativas relevantes;
- `WARN`: degradación o consecuencia recuperable que requiere atención;
- `ERROR`: operación que no pudo completarse.

Un fallo se registra una sola vez por su propietario operativo. `repository` devuelve errores sin imprimirlos; `service` registra únicamente transiciones que coordina; `server` cierra cada solicitud con una única línea; `executor` posee el ciclo de la partida. Los fallos se atribuyen a `agent`, `game`, `platform` o `infrastructure`.

La consola no es una fuente de verdad ni un almacén general. Permanecen separados:

- auditoría de acciones sensibles;
- eventos autoritativos por tick y replay;
- informes sanitizados para participantes;
- métricas técnicas agregadas.

No se escriben tokens, credenciales, código, payloads, SQL, correo, username, percepción privada ni rutas completas en eventos operativos normales. Los incidentes repetidos por tick se resumen al terminar la partida. La retención de logs operativos continúa limitada según ATD-014.

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
