> [!WARNING]
> Documento histórico. No es una fuente vigente de requisitos ni arquitectura.
> Consúltese `docs/index.md` y `docs/roadmap/current.md` para el estado actual.

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

### Paleta base aprobada para el primer corte público

José Daniel eligió la opción **celeste y arena** el 2026-09-18. Reemplaza la propuesta inicial con primario índigo `#6574D9`; no modifica el dominio ni los significados de estado.

| Uso | Color | Motivo |
| --- | --- | --- |
| Fondo | #F6FAFD | Superficie luminosa y descansada |
| Superficie | #FFFFFF | Jerarquía limpia |
| Texto principal | #233947 | Contraste sin negro absoluto |
| Primario | #346D8C | Acción y marca |
| Primario suave | #E1F0F7 | Superficies y selección |
| Arena suave | #F7EFDB | Acento cálido de fondo |
| Secundario | #64BFA5 | Progreso y éxito |
| Advertencia | #E9B760 | Atención sin estridencia |
| Peligro | #D96C7A | Fallos y acciones destructivas |
| Información | #62A8D8 | Estados informativos |

La paleta es una resolución visual derivada de AR-073 y AR-075. Los colores nunca comunican significado por sí solos. El texto y los controles deben conservar contraste suficiente sobre los fondos claros.

### Iconos del primer corte público

Se usa Lucide con trazos simples y tamaños consistentes para navegación, fechas y estados. Las acciones principales conservan una etiqueta de texto; los iconos decorativos se ocultan a tecnologías asistivas.

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
