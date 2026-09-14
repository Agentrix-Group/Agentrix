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

