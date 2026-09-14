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
