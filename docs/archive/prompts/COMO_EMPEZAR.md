> [!WARNING]
> Documento histórico. No es una fuente vigente de requisitos ni arquitectura.
> Consúltese `docs/index.md` y `docs/roadmap/current.md` para el estado actual.

# Cómo pasar el blueprint a Agentrix

## 1. Descargar

Descarga `agentrix-codex-handoff.zip`, normalmente en `~/Downloads/`.

## 2. Revisar el contenido antes de extraer

```bash
cd ~/Projects/Agentrix
unzip -l ~/Downloads/agentrix-codex-handoff.zip
```

El paquete está preparado para extraerse directamente en la raíz del repositorio. Añade `AGENTS.md`, `PROMPT_INICIAL.md`, `COMO_EMPEZAR.md` y `docs/blueprint/`. No debe reemplazar el `README.md` existente.

## 3. Extraer

```bash
cd ~/Projects/Agentrix
unzip ~/Downloads/agentrix-codex-handoff.zip -d .
```

Si `unzip` anuncia que reemplazará un archivo existente, responde que no y revisa primero ese archivo. Según el árbol compartido inicialmente, no debería existir ninguna de estas rutas.

## 4. Verificar lo añadido

```bash
git status --short
tree -L 2 docs
```

Deberías ver únicamente documentación nueva. Todavía no hagas commit si primero quieres revisar el diagnóstico de Codex.

## 5. Abrir Codex en el repositorio

```bash
cd ~/Projects/Agentrix
codex
```

Abre `PROMPT_INICIAL.md`, copia el bloque marcado como `text` y pégalo en Codex.

## 6. Qué debe ocurrir

Codex leerá el blueprint, inspeccionará el prototipo y responderá con un diagnóstico. No debe modificar el código en esa primera ejecución. Después podrás traer aquí ese diagnóstico o aprobar en Codex un único corte vertical.

## 7. Si algo sale mal

- Si Codex intenta reestructurar todo: deténlo y recuérdale que primero debe cumplir `AGENTS.md`.
- Si no encuentra los documentos: confirma que existe `~/Projects/Agentrix/docs/blueprint/README.md`.
- Si `codex` no abre: ejecuta `codex --version` y conserva el mensaje de error completo.
- Si el ZIP no está en `~/Downloads/`: reemplaza esa ruta por la ubicación real del archivo.
