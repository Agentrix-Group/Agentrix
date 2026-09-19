# Seguridad

## Frontera de confianza

El ZIP y el proceso del bot son no confiables. El motor también vive fuera de la API para limitar fallos, pero es un artefacto oficial. PostgreSQL, secretos y artefactos de otros participantes nunca deben ser accesibles desde un bot.

## Estado actual

| Control | Estado |
| --- | --- |
| ZIP limitado, rutas controladas y dos archivos exactos | Implementado |
| Sintaxis Python e interacción de admisión | Implementado |
| Proceso persistente y kill por timeout | Implementado en código; pruebas actuales fallan en entorno restringido |
| Red deshabilitada con Bubblewrap | Parcial |
| Filesystem mínimo | No implementado; se monta `/` en solo lectura |
| Entorno limpio | No implementado |
| Límites CPU/memoria/PIDs/output | No implementado |
| Fallo cerrado sin runtime | No implementado; existe fallback directo a Python |
| Secretos obligatorios en producción | No implementado; existen defaults de desarrollo |
| Pruebas de escape | No implementado |

El término “rootless OCI” no describe la implementación actual: Bubblewrap crea namespaces, pero no es un runtime OCI y su presencia no basta para probar que el entorno permite las operaciones requeridas.

## Requisitos de producción

- Runtime rootless explícitamente soportado y comprobado al iniciar el worker.
- Sin red, capacidades, dispositivos innecesarios ni namespaces compartidos.
- Root filesystem mínimo y de solo lectura; directorio temporal limitado.
- Variables de entorno permitidas mediante lista positiva.
- Límites efectivos de CPU, memoria, PIDs, tiempo y bytes de stdout/stderr.
- Terminación del grupo completo de procesos.
- Artefactos montados por digest y solo para el slot correspondiente.
- Worker se niega a reservar trabajos si no puede aplicar todas las garantías.
- Suite hostil que prueba red, filesystem, forks, output, señales y consumo de recursos.

## Secretos y observabilidad

No registrar tokens, credenciales, correo, username, código, payloads, SQL, rutas completas ni percepciones privadas. Logs operativos, auditoría, evidencia de partida y reporte sanitizado son productos distintos con retención y audiencia propias.
