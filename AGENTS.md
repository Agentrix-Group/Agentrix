# Instrucciones de trabajo para Codex — Agentrix

## 1. Contexto

Agentrix es el nombre definitivo del repositorio, producto y sistema descrito en `docs/blueprint/`, según DP-002. No introduzcas nombres alternativos para el producto en código, contratos o documentación activa.

El código existente fue creado como experimento. Su presencia no demuestra que una función esté diseñada, terminada o aprobada.

## 2. Fuentes y prioridad

Cuando dos fuentes difieran, usa este orden:

1. La instrucción más reciente y explícita de José Daniel.
2. Las decisiones marcadas como decididas en `docs/blueprint/01_revision_y_resoluciones.md`.
3. El modelo de dominio, casos de uso, flujos y requisitos del blueprint.
4. Las propuestas técnicas ATD que José Daniel haya aprobado.
5. El código actual, únicamente como evidencia del prototipo.

No conviertas una suposición, un stub o una estructura existente en requisito.

## 3. Protocolo obligatorio antes de modificar código

En la primera sesión:

1. Lee este archivo y `docs/blueprint/README.md`.
2. Lee `docs/blueprint/source/cuestionario_maestro_agentrix.md`, todos los documentos numerados del blueprint y `docs/blueprint/00_ADAPTACION_AGENTRIX.md`.
3. Inspecciona el repositorio real.
4. Ejecuta solo comprobaciones no destructivas que no requieran instalar dependencias.
5. Clasifica lo encontrado como verificado, parcial, stub, contradictorio o ausente.
6. Presenta el diagnóstico y un primer corte vertical pequeño.
7. Espera la aprobación de José Daniel antes de editar archivos.

No hagas una reescritura general, no muevas carpetas y no borres el prototipo durante esta fase.

## 4. Organización del código

La decisión DP-001 fija una organización horizontal, poco profunda y por responsabilidad técnica. Conserva el recorrido reconocible por nombre:

`open-api/contests.yaml` → `src/server/contests.go` → `src/service/contests.go` → `src/repository/contests.go` → `src/model/contest.go`

Responsabilidades:

- `server`: transporte HTTP y WebSocket, validación de entrada y presentación de respuestas.
- `service`: casos de uso, autorización y transiciones del negocio.
- `repository`: consultas y persistencia, sin reglas del negocio.
- `model`: entidades, valores, invariantes y estados, sin HTTP ni SQL.
- `connection`: creación de conexiones, clientes y transacciones técnicas.
- `auth`: credenciales, sesiones y primitivas de autorización.
- `executor`: trabajos, procesos, sandbox y coordinación de ticks.
- `game`: contratos y registro de módulos de juego de la plataforma.
- `replay`: escritura, sellado y proyección de eventos.
- `tracer`: logs estructurados, correlación y métricas técnicas.
- `common`: solo conceptos realmente transversales; nunca código sin propietario.

No agregues otra capa, patrón, framework o carpeta sin explicar un problema real del corte actual y recibir aprobación.

La carpeta `repository` ya existe en el prototipo, pero ATD-015 todavía exige revisar si su separación concreta resulta útil. No la expandas ni la elimines por inercia: primero muestra cómo se usa actualmente y recomienda conservarla o retirarla con evidencia.

## 5. Límites de la primera revisión

- Ignora `*_test.go` al juzgar la distribución principal de directorios, pero no los borres.
- Trata `bin/`, `artifacts/` y `__pycache__/` como posibles salidas generadas, no como fuentes de requisitos.
- No instales paquetes, no cambies versiones y no levantes servicios externos sin autorización.
- No modifiques la base de datos real ni ejecutes scripts destructivos.
- No afirmes que algo funciona solo porque existe un archivo con ese nombre.
- Conserva los cambios ajenos y el historial Git.

## 6. Forma de colaboración

- Explica en español; mantén identificadores de código y contratos en inglés.
- Presenta primero el resultado y después la evidencia.
- Evita documentos duplicados o contradictorios.
- Relaciona cada cambio con una entidad, caso de uso, requisito o decisión del blueprint.
- Implementa un solo corte vertical aprobado a la vez.
- Antes de codificar un flujo, explica sus entidades, estados, autorización, entrada, salida y errores.
- Después de modificar, ejecuta verificaciones relevantes y comunica con precisión lo comprobado y lo que sigue incierto.

## 7. Prohibición principal

No intentes “completar Agentrix” en una sola ejecución. El objetivo es que José Daniel pueda entender y aprobar cada parte antes de que se convierta en código.

`docs/blueprint/BLUEPRINT_COMPLETO.md` existe para lectura humana continua y duplica los demás documentos. No lo leas además de los archivos separados durante una misma revisión.
