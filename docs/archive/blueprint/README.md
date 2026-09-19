> [!WARNING]
> Documento histórico. No es una fuente vigente de requisitos ni arquitectura.
> Consúltese `docs/index.md` y `docs/roadmap/current.md` para el estado actual.

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

0. [00_ADAPTACION_AGENTRIX.md](../audits/00_ADAPTACION_AGENTRIX.md): relación entre el blueprint y el prototipo actual de Agentrix.
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

La copia concatenada `BLUEPRINT_COMPLETO.md` fue retirada durante la limpieza
documental del 2026-09-19 porque duplicaba la fuente y estos documentos. El
historial Git conserva su contenido.

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
