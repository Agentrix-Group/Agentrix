# ADR-0003: Runtime de aislamiento y almacenamiento de artefactos

## Estado

accepted

## Fecha

2026-09-19

## Contexto

El MVP de Agentrix requiere ejecutar código Python de participantes no confiables sin comprometer la seguridad del host del worker, la red interna ni la estabilidad del sistema. El mecanismo anterior (`bwrap --ro-bind / /`) montaba la raíz completa del host y permitía un fallback directo a `python3`, lo cual es inaceptable para producción.

Asimismo, la persistencia de bundles de bots y replays sellados de partidas requiere una estrategia de almacenamiento escalable, inmutable y atómica que no sobrecargue la base de datos PostgreSQL con blobs binarios.

## Decisión

### 1. Runtime de aislamiento de bots
- **Tecnología oficial:** Podman rootless (con Docker rootless como alternativa compatible con OCI).
- **Fail-closed:** En producción, si el runtime de aislamiento no está disponible o falla su inicialización, el worker falla cerrado (`fail-closed`) y rechaza ejecutar el bot.
- **Modo desarrollo:** La ejecución directa con fallback solo se permite en desarrollo local mediante la variable explícita `AGENTRIX_DISABLE_SANDBOX=1`.
- **Restricciones del contenedor de bot:**
  - Un contenedor por bot por partida (`one container per bot per match`).
  - Red deshabilitada completamente (`--network none`).
  - Sistema de archivos raíz de solo lectura (`read-only rootfs`), con `/tmp` montado como `tmpfs` acotado.
  - Montaje únicamente del bundle del bot en modo solo lectura.
  - Usuario sin privilegios (`non-root UID/GID`).
  - Eliminación de todas las capabilities de Linux (`--cap-drop=ALL`), `no-new-privileges` y perfil seccomp restrictivo.
  - Límites estrictos de CPU, memoria (OOM handler), PIDs (prevención de fork bombs) y límites de salida (longitud de línea y bytes de stdout/stderr).
  - Variables de entorno limpiadas, inyectando exclusivamente allowlist básica necesaria para el runtime de Python.

### 2. Almacenamiento de artefactos
- **Tecnología oficial:** S3 / MinIO (API compatible con Amazon S3).
- **Desarrollo y CI:** Instancia local de MinIO en `docker-compose`.
- **Producción:** Servicio S3 administrado u Object Storage compatible.
- **Filesystem local:** Reservado exclusivamente para pruebas unitarias y desarrollo sin servicios externos.
- **Inmutabilidad y atomicidad:** Los replays se escriben primero en almacenamiento temporal, se validan y comprimen con Zstandard, y luego se publican atómicamente con digest SHA-256 inmutable.

## Consecuencias

- El host del worker queda completamente aislado de ataques de bots (lectura de `/etc`, exfiltración de red, agotamiento de recursos o procesos huérfanos).
- PostgreSQL almacena metadatos y referencias (`file_path` / URI, digest, tamaño), mientras los blobs residen en el Object Store.
