# ADR 0011: Modelo de Seguridad RBAC, Sesiones Persistentes y Trazabilidad sin Fuga

## Estado
Aceptado (Accepted)

## Contexto
La transición arquitectural canónica eliminó definitivamente el modelo monolítico de `participant` como identidad global, separando la identidad del usuario (`User`) de la admisión al concurso (`ContestEntry`) y los agentes (`Agent`).
Para salvaguardar la plataforma contra accesos no autorizados, abuso de tokens y escalada de privilegios, se requerían mecanismos de control de acceso basados en roles y capacidades dinámicas (RBAC), revocación atómica ante reutilización de tokens y ofuscación estricta de credenciales en observabilidad.

## Decisión
1. **Roles Canónicos y Capacidades Dinámicas:**
   - Roles del sistema: `admin`, `organizer`, `pilot`, `spectator`.
   - Permisos y capacidades granulares (`matches:run`, `matches:schedule`, `contests:create`, `contests:manage`, `rankings:publish`, `users:view`, `agents:create`, `submissions:upload`).
   - Resolución dinámica desde base de datos (`user_role_capabilities`) con proyección en tokens de sesión (`/api/v1/me`).
2. **Ciclo de Vida de Sesiones y Detección de Reutilización:**
   - Almacenamiento de sesiones con hashing de tokens de refresco (`user_sessions`).
   - Rotación automática de refresh token en cada renovación.
   - Si se detecta un intento de intercambio con un refresh token previamente consumido o revocado, el sistema activa revocación inmediata de **todas** las sesiones activas del usuario (`ErrSessionReuseDetected`), invalidando cookies y tokens.
3. **Criptografía Robusta:**
   - Almacenamiento de contraseñas mediante hashing Bcrypt con coste de cómputo factor 12.
   - Comparación en tiempo constante para evitar ataques de canal lateral (timing attacks).
4. **Protección Perimetral y Redacción de Logs:**
   - Rate limiting por IP en endpoints de autenticación (`/auth/login`, `/auth/register`, `/auth/refresh`).
   - Logger unificado con filtro regex que ofusca tokens JWT, contraseñas, emails privados y rutas completas del sistema de archivos (`[redacted-path]`).

## Consecuencias
- **Positivas:**
  - Control granular de privilegios sin acoplamiento a nombres rígidos.
  - Imposibilidad de robo y reutilización silenciosa de tokens de refresco.
  - Trazabilidad y logs aptos para producción y cumplimiento de privacidad.
- **Negativas / Mitigaciones:**
  - Costo de cómputo en bcrypt (mitigado fijando coste 12 y controlando concurrencia).
  - Consulta a `user_sessions` en refresco (mitigado por índice primario y clave compuesta).
