# Despliegue

## Estado actual

**No existe un despliegue completo del MVP.** El `Dockerfile` compila y copia únicamente el servidor Go a una imagen distroless. No incluye:

- `starfighter-engine`;
- Python;
- runtime de aislamiento;
- bots de referencia;
- frontend construido;
- herramientas o configuración de artefactos.

Además, `main.go` inicia API y pool de workers en el mismo proceso. La variable `MODE` cambia aspectos de configuración y logging, pero no selecciona un rol de despliegue.

## Desarrollo local

`make run` inicia el proceso combinado. Requiere que existan el manifest de Starfighter, el artifact store local y, para una partida real, el engine compilado. Si PostgreSQL no está disponible, la aplicación continúa con cola en memoria.

Ese fallback es de desarrollo y no demuestra durabilidad.

## Objetivo aprobado

- Un mismo código componible con modos explícitos `api` y `worker`.
- PostgreSQL obligatorio fuera de desarrollo.
- Worker con engine, Python y runtime rootless comprobados al arrancar.
- API sin necesidad de ejecutar código no confiable.
- Artefactos persistentes compartidos o almacenamiento de objetos.
- Migraciones ejecutadas como paso controlado, no al iniciar cada réplica.
- Imágenes reproducibles identificadas por digest.
- Health/readiness separados para API y worker.
- Producción falla cerrado ante dependencias de seguridad ausentes.

## Puerta de despliegue

No declarar una imagen lista hasta verificar desde una imagen limpia:

1. arranque de API y worker por separado;
2. conexión PostgreSQL obligatoria;
3. disponibilidad del engine correcto por digest;
4. sandbox rootless y límites efectivos;
5. ejecución de una partida y commit único de resultado/replay;
6. servicio de la web y replay público;
7. recuperación de worker sin duplicación.
