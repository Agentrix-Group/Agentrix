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
