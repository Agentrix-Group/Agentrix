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

