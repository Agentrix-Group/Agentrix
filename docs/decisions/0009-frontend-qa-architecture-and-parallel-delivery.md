# ADR-0009: Arquitectura de Frontend, Estrategia de QA y Entrega Paralela por Vertical Slices

## Estado

accepted

## Fecha

2026-09-20

## Contexto

Tras la certificación canónica del MVP en el backend (`Agentrix@1ee7e59`, ADR-0008), la auditoría integral de la interfaz web (`web/`) y la estrategia de control de calidad reveló que el frontend actual constituye una demo funcional de lectura y administración básica, pero presenta deficiencias estructurales que impiden considerarlo listo para el producto final:

1. **Ciclo de vida de submissions desincronizado (F0.1):** La UI asume que la subida de un bundle concluye de forma síncrona en estado `ready`, impidiendo reflejar el pipeline de admisión asíncrono con estados `pending_validation`, `validating`, `ready` y `rejected`.
2. **Ausencia de creación guiada de partidas (F0.2):** Aunque el endpoint de creación existe en la API, la interfaz carece de asistente para configurar partidas, seleccionar submissions compatibles o establecer semillas deterministas.
3. **Manejo de sesión incompleto (F0.3):** Solo se almacena el `access_token` en `localStorage`; no se conserva el `refresh_token`, las respuestas `401` no disparan renovación ni informan sobre caducidad de sesión, y el logout es exclusivamente local.
4. **Capa HTTP débil y acoplada (F0.4):** Se envía `Content-Type: application/json` incluso en peticiones `GET` (provocando preflights innecesarios en peticiones cross-origin), no hay soporte para timeout ni `AbortSignal`, se ignoran códigos `204 No Content`, y no se extraen ni propagan identificadores de correlación (`X-Correlation-ID`).
5. **Fuga de rutas internas del servidor (F0.5):** La interfaz expone directamente `code_path` en la tabla de envíos, revelando rutas absolutas del sistema de archivos del servidor.
6. **Silenciamiento de errores en la interfaz (F0.6):** Se encontraron bloques `.catch(() => {})` en catálogos y clasificaciones, y la página de inicio enmascaraba caídas de red convirtiéndolas en estados vacíos ("no hay partidas"), dificultando las labores de QA.
7. **Viewer de repeticiones no escalable (F0.7):** Se descarga el NDJSON completo en memoria, se ejecuta en el hilo principal sin Web Worker ni `requestAnimationFrame`, y las dimensiones de la arena (2000×1000) están fijadas sin consultar los metadatos autoritativos.
8. **Ausencia de routing real (F0.8):** La navegación operaba únicamente mediante estado local de React (`useState`), imposibilitando el uso de botones de navegación del explorador (Back/Forward), recarga de página y enlaces compartibles directos a repeticiones o clasificaciones.
9. **Brechas de QA y CI:** La suite se limitaba a pruebas unitarias y DOM simulado en `jsdom` (sin validación de renderizado Canvas real), sin pruebas E2E en exploradores reales, sin verificación de accesibilidad automatizada (WCAG AA) y con un trabajo de CI rotulado con "lint" sin contar con el script correspondiente.

## Decisión

Adoptar una reestructuración de la arquitectura de frontend, un modelo de entrega sincronizado por **vertical slices** y una pirámide de aseguramiento de calidad en 9 capas:

### 1. Entrega paralela por Vertical Slices
Ningún cambio de backend o contrato se considerará terminado sin incluir:
- Contrato/esquema validado.
- Implementación y pruebas de servicio.
- Fixtures/seeds unificados.
- Adaptación del cliente frontend y estados UX (`loading`, `error`, `empty`, `success`).
- Pruebas unitarias, de integración y smoke E2E.
- Trazabilidad y accesibilidad (WCAG 2.2 AA en flujos críticos).

### 2. Arquitectura de frontend modular (Sprint FQ-0 en adelante)
- **Routing del navegador:** Enrutador ligero basado en el estándar HTML5 History API (`pushState`, `popstate`), con soporte de rutas dinámicas (`/`, `/matches`, `/matches/:id`, `/replays/:id`, `/rankings`, `/agents`, `/auth`), preservando enlaces directos y navegación nativa del explorador.
- **Error Boundary:** Captura global y a nivel de vista de errores de renderizado en React, presentando al usuario un identificador de incidente sanitizado (`incident_id`) y opción de recuperación/reintento.
- **Cliente HTTP centralizado:** Cliente tipado con `AbortSignal`, timeout por defecto (10s), manejo correcto de cabeceras (sin `Content-Type` en peticiones sin cuerpo), soporte para `204 No Content`, extracción de `X-Correlation-ID` y jerarquía de errores (`ApiClientError`, `NetworkError`, `TimeoutError`).
- **Estados explícitos sin silenciamiento:** Toda vista con carga asíncrona implementa el patrón de cuatro estados (`loading`, `error`, `empty`, `success`), con mensajes traducidos y botón de reintento.
- **Privacidad y DTOs limpios:** Supresión total de `code_path` en las respuestas de la API (`json:"-"`) y en la interfaz de usuario, sustituyéndolo por metadatos seguros (versión, estado, identificador resumido).

### 3. Estrategia de QA en 9 capas
- **Capa 1 (Estáticos):** Formato, linting, validación estricta de paridad i18n y build de producción.
- **Capa 2 (Unitarias):** Reductores, cliente HTTP, gestión de sesión, parser de repeticiones y helpers de autorización.
- **Capa 3 (Integración de componentes):** React Testing Library con simulación a nivel de red (HTTP), evaluando combinaciones de estados y códigos HTTP (401, 403, 404, 409, 422, 500).
- **Capa 4 (Contratos):** Validación de esquemas OpenAPI y detección de roturas de contrato en CI.
- **Capa 5 (E2E explorador real):** Playwright sobre stack efímero cubriendo el recorrido crítico completo: registro -> agente -> bundle -> admisión -> inscripción -> ejecución -> replay -> ranking.
- **Capa 6 (Accesibilidad):** Auditorías automatizadas con axe-core, navegación completa por teclado, `aria-live` y compatibilidad con lectores de pantalla.
- **Capa 7 (Regresión visual):** Capturas doradas de frames del viewer y pantallas en resolución de escritorio y móvil.
- **Capa 8 (Rendimiento):** Presupuestos de bundle en Vite, métricas Core Web Vitals y prueba de estabilidad de memoria en streaming de repeticiones.
- **Capa 9 (Seguridad cliente):** Política de Content Security Policy (CSP), protección contra XSS en datos de usuario y ausencia de credenciales o rutas internas en artefactos.

### 4. Roadmap sincronizado
- **Sprint FQ-0:** Cimientos de calidad (routing nativo, Error Boundary, cliente HTTP tipado, saneamiento de errores/paths, paridad y pruebas base).
- **Sprint FQ-1:** Identidad, capacidades y sesión persistente/renovable.
- **Sprint FQ-2:** Flujo de admisión asíncrona de bots y feedback de validación.
- **Sprint FQ-3:** Creación, configuración y seguimiento en vivo de partidas.
- **Sprint FQ-4:** Visor de repeticiones de alto rendimiento con Web Worker y verificación de integridad criptográfica.
- **Sprint FQ-5:** Experiencia de concurso y rankings en tiempo real.
- **Sprint FQ-6:** Panel operativo, métricas y endurecimiento para release final.

## Consecuencias

- Se garantiza la paridad estricta entre las capacidades del backend y la experiencia del usuario final.
- Se previene la acumulación de deuda técnica en la interfaz y la aparición de fallos masivos de integración en la fase de entrega.
- Las rutas del explorador son indexables y compartibles, permitiendo abrir partidas y repeticiones directamente mediante su URL.
- La exposición accidental de infraestructura interna del servidor queda cerrada tanto en el transporte HTTP como en la presentación.
