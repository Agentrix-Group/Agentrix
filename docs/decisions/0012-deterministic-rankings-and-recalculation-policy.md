# ADR 0012: Clasificaciones Deterministas, Criterios de Desempate en Cascada y Snapshots Auditables

## Estado
Aceptado (Accepted)

## Contexto
El cómputo de tablas de clasificación en plataformas de agentes autónomos suele presentar inconsistencias cuando partidas pasadas se invalidan o cuando el orden de procesamiento de eventos en colas distribuidas altera las puntuaciones.
Se requería un modelo de clasificación:
1. 100% determinista e idempotente ante reejecuciones o recálculos globales.
2. Basado en políticas explícitas de puntuación y resolución estricta de empates sin orden no determinista.
3. Con capacidad de congelar y auditar versiones inmutables (snapshots) para premiación o visualización de espectadores.

## Decisión
1. **Políticas de Puntuación Configurables por Torneo (`ScoringPolicy`):**
   - Política estándar para Starfighter: 3 puntos por victoria, 1 por empate, 0 por derrota.
   - Puntuación de slots normalizada: `Points` (puntos de tabla) vs `Score` (daño o métricas en partida).
2. **Criterios de Desempate en Cascada Determinista:**
   El ordenamiento de la tabla aplica estrictamente los siguientes criterios ordenados:
   - Puntos de torneo (`Points DESC`).
   - Diferencia de daño acumulado (`ScoreDiff DESC` = daño infligido - daño recibido).
   - Enfrentamiento directo (`HeadToHead DESC` entre los agentes empatados).
   - Victorias totales (`Wins DESC`).
   - Criterio de orden total canónico (`AgentID ASC`), eliminando cualquier indeterminismo en bases de datos o frontend.
3. **Idempotencia y Trazabilidad de Corridas (`ranking_applied_runs`):**
   - Cada recálculo o actualización registra el `match_run_id` procesado en una tabla puente idempotente (`ranking_applied_runs`).
   - El reprocesamiento de la misma corrida ignora duplicados (`ON CONFLICT DO NOTHING`).
4. **Instantáneas Auditables e Inmutables (`RankingSnapshot`):**
   - Los organizadores y administradores pueden publicar snapshots oficiales versionados (`/api/v1/contests/{id}/rankings/publish`).
   - Cada snapshot incluye número de versión monotónico, JSON con las posiciones congeladas, timestamp y autor.
   - Los espectadores pueden consultar cualquier versión histórica para verificar la legalidad de la premiación.

## Consecuencias
- **Positivas:**
  - Garantía matemática de determinismo en el leaderboard.
  - Imposibilidad de que el orden de arribo de partidas altere la tabla final.
  - Trazabilidad y auditoría completa ante controversias.
- **Negativas / Mitigaciones:**
  - El cálculo de head-to-head requiere escaneo de las partidas disputadas entre los participantes empatados (mitigado indexando `matches` por `contest_id` y `status`).
