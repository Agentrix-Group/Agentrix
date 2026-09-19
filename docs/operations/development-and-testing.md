# Desarrollo y pruebas

## Requisitos

- Go según `go.mod`.
- Rust estable según `agentrix_engine/rust-toolchain.toml`.
- Python 3.
- Node.js y npm compatibles con el lockfile web.
- PostgreSQL solo para pruebas de integración que lo declaren.
- `bwrap` y `zstd` para comprobar rutas opcionales actuales.

No instale dependencias como parte de una revisión sin autorización. Use modos readonly/offline cuando sea posible.

## Backend Go

```bash
GOCACHE=/tmp/agentrix-go-cache go build -mod=readonly ./...
GOCACHE=/tmp/agentrix-go-cache go test -mod=readonly ./...
GOCACHE=/tmp/agentrix-go-cache go vet -mod=readonly ./...
```

Estado observado el 2026-09-19:

- build: correcto;
- vet: correcto;
- tests: fallan en `src/executor` cuando procesos de bots iniciados mediante Bubblewrap aparecen como `crashed`; el caso de timeout termina como `score_limit`.

Ese resultado puede depender de las capacidades del entorno, pero sigue siendo un fallo real de la suite y no una validación superada.

## Web

Con dependencias ya presentes:

```bash
cd web
npm test -- --run
./node_modules/.bin/vite build --outDir /tmp/agentrix-web-build --emptyOutDir
```

El 2026-09-19 pasaron 23 pruebas en 6 archivos y el build terminó correctamente. JSDOM advirtió que `HTMLCanvasElement.getContext` no está implementado sin un paquete adicional; las pruebas verifican el componente, no el dibujo real del canvas.

## Motor Rust

Desde el repositorio hermano:

```bash
CARGO_TARGET_DIR=/tmp/agentrix-engine-target cargo test --locked --offline
```

El 2026-09-19 pasaron 11 unit tests y 1 integración del subproceso; 1 prueba estadística quedó ignorada por su duración. Esto prueba el engine en el entorno actual, no conformidad multiplataforma.

## Construcción integrada

```bash
make build-engine
make build
```

`make build-engine` espera `../agentrix_engine` salvo que se cambie `ENGINE_SRC`. `make build` ejecuta `gofmt -w`, `go vet`, tests y compilación, por lo que no es una comprobación de solo lectura.

## Base de datos

Los targets `db-setup`, `db-seed` y `db-reset` modifican PostgreSQL. `db-reset` elimina y recrea la base indicada. Solo deben ejecutarse contra un destino confirmado y con autorización explícita.

## Salidas generadas

No versionar ni interpretar como diseño:

- `bin/`;
- `artifacts/`;
- `web/dist/`;
- `target/`;
- archivos de cobertura;
- `__pycache__/` y `*.pyc`.
