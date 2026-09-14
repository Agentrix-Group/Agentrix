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

