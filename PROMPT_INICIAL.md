# Prompt inicial para Codex

Copia únicamente el contenido del siguiente bloque y pégalo en Codex después de abrirlo dentro de `~/Projects/Agentrix`.

```text
Estamos trabajando en Agentrix. El repositorio actual es un prototipo creado para probar ideas; no asumas que sus archivos están completos ni que representan decisiones aprobadas.

Antes de hacer cualquier cambio:

1. Lee AGENTS.md completo.
2. Lee docs/blueprint/README.md, docs/blueprint/source/cuestionario_maestro_agentrix.md, docs/blueprint/00_ADAPTACION_AGENTRIX.md y todos los documentos numerados de docs/blueprint/. No leas BLUEPRINT_COMPLETO.md en esta misma revisión porque duplica esos archivos.
3. Inspecciona el contenido real del repositorio. Ignora los *_test.go únicamente al evaluar la distribución de carpetas, pero no los borres ni los desprecies como mecanismo de verificación.
4. No modifiques, crees, muevas ni elimines ningún archivo durante esta primera tarea.
5. No instales dependencias ni ejecutes scripts de base de datos. Puedes usar comandos de lectura y comprobaciones no destructivas que funcionen con lo ya instalado.

Después entrégame un diagnóstico en español que contenga:

- un resumen honesto de qué hace actualmente el repositorio;
- una tabla por área usando solo: Verificado, Parcial, Stub, Contradictorio, Ausente o Aplazado;
- evidencia concreta con rutas de archivos para cada afirmación;
- resultado de compilación, go test y go vet si pueden ejecutarse sin alterar dependencias;
- diferencias entre el código actual y el modelo de dominio, los casos de uso, los flujos y la arquitectura del blueprint;
- detección de archivos generados o accidentales como binarios y __pycache__, sin eliminarlos;
- explicación de cómo se implementaría un único flujo vertical siguiendo la estructura open-api → server → service → repository → model;
- análisis concreto de si conviene conservar src/repository o retirarlo;
- propuesta del primer corte vertical, sus criterios de aceptación y la lista exacta de archivos que tocaría.

Termina con las decisiones que necesitas que yo tome. No programes hasta que apruebe expresamente el diagnóstico y el primer corte.
```

## Resultado esperado

Codex debe responder con una auditoría, no con una refactorización ni una lista de archivos creados. Si comienza a modificar código, detén la ejecución y recuérdale la regla de `AGENTS.md`.
