# Flujos, estados y secuencias

## 1. Estado del concurso

~~~mermaid
stateDiagram-v2
    [*] --> Borrador
    Borrador --> Publicado: publicar reglas y calendario
    Publicado --> InscripcionAbierta: abrir inscripciones
    InscripcionAbierta --> Preparacion: cerrar inscripciones
    Preparacion --> EnCurso: inaugurar
    EnCurso --> SeleccionFinal: cerrar envios
    SeleccionFinal --> FinalEnVivo: congelar agentes
    FinalEnVivo --> Finalizado: confirmar resultados
    Finalizado --> Archivado: cerrar reclamos

    Publicado --> Cancelado: cancelar
    InscripcionAbierta --> Cancelado: cancelar
    Preparacion --> Suspendido: incidente
    EnCurso --> Suspendido: incidente
    FinalEnVivo --> Suspendido: incidente
    Suspendido --> EnCurso: reanudar
    Suspendido --> FinalEnVivo: reanudar final
    Suspendido --> Cancelado: cancelar
    Cancelado --> Archivado: documentar cierre
~~~

### Reglas

- Borrador es editable sin efecto público.
- Publicar congela la versión inicial del reglamento.
- Reprogramar un hito no cambia el estado ni el reglamento.
- EnCurso permite pruebas, mejoras y resultados no finales.
- SeleccionFinal impide nuevos envíos y permite elegir entre versiones aprobadas.
- FinalEnVivo muestra una sola partida pública a la vez.
- Archivado es terminal para operación normal.

## 2. Estado de la inscripción

~~~mermaid
stateDiagram-v2
    [*] --> Borrador
    Borrador --> Solicitada: enviar datos
    Solicitada --> EnEvaluacion: iniciar revision
    EnEvaluacion --> Confirmada: requisitos y pago validos
    EnEvaluacion --> RequiereCorreccion: faltan datos
    RequiereCorreccion --> Solicitada: corregir
    Confirmada --> Congelada: cerrar cambios
    Solicitada --> Retirada: retirar
    Confirmada --> Retirada: retirar antes del cierre
    Congelada --> Descalificada: sancion
    Retirada --> [*]
    Descalificada --> [*]
~~~

El sistema no rechaza a una persona por cupo, pero puede exigir corregir datos o ubicarla en una categoría compatible. Una descalificación posterior es una sanción competitiva y requiere resolución.

## 3. Estado del envío

~~~mermaid
stateDiagram-v2
    [*] --> Recibiendo
    Recibiendo --> Recibido: transferencia completa
    Recibiendo --> Fallido: transferencia incompleta
    Recibido --> EnCola: solicitar validacion
    EnCola --> Validando: reservar runner
    Validando --> Aprobado: todas las pruebas pasan
    Validando --> Rechazado: agente o paquete invalido
    Validando --> ValidacionInterrumpida: fallo de plataforma
    ValidacionInterrumpida --> EnCola: reintentar
    Aprobado --> Elegible: categoria vigente
    Elegible --> Seleccionado: participante confirma
    Seleccionado --> Elegible: reemplazar antes del cierre
    Seleccionado --> Congelado: cerrar seleccion
    Congelado --> Anulado: trampa comprobada
    Rechazado --> [*]
    Anulado --> [*]
~~~

Cada nuevo paquete crea otro Envío. Un envío rechazado no se edita. La selección vigente puede cambiar sin borrar selecciones anteriores.

## 4. Estado de una partida

~~~mermaid
stateDiagram-v2
    [*] --> Borrador
    Borrador --> Programada: completar slots y configuracion
    Programada --> EnCola: llega momento
    EnCola --> Preparando: reservar capacidad
    Preparando --> Ejecutando: verificar artefactos
    Ejecutando --> Procesando: juego finaliza
    Procesando --> ResultadoProvisional: registro completo
    ResultadoProvisional --> Confirmada: termina revision
    ResultadoProvisional --> BajoRevision: abrir incidente
    BajoRevision --> Confirmada: mantener resultado
    BajoRevision --> Anulada: invalidar ejecucion
    Anulada --> EnCola: programar nuevo intento

    Preparando --> FalloTecnico: infraestructura o juego
    Ejecutando --> FalloTecnico: infraestructura o juego
    Procesando --> FalloTecnico: evidencia incompleta
    FalloTecnico --> EnCola: reintentar
    FalloTecnico --> Cancelada: imposibilidad comprobada

    Confirmada --> [*]
    Cancelada --> [*]
~~~

Un timeout o acción inválida atribuible a un agente no lleva a FalloTecnico: el juego registra la consecuencia y la ejecución continúa o finaliza según su política.

## 5. Secuencia de presentación y validación

~~~mermaid
sequenceDiagram
    actor C as Concursante
    participant W as Aplicacion web
    participant A as API
    participant O as Almacen de artefactos
    participant Q as Cola
    participant V as Validador aislado
    participant J as Modulo de juego

    C->>W: Selecciona agente y paquete
    W->>A: Solicita recepcion
    A->>A: Autoriza participante y periodo
    A->>O: Guarda paquete
    O-->>A: Digest y ubicacion
    A->>A: Crea envio inmutable
    A->>Q: Encola validacion
    A-->>W: Envio recibido

    Q->>V: Reserva trabajo
    V->>O: Lee artefacto por digest
    V->>V: Verifica formato y prepara runner
    V->>J: Ejecuta prueba de protocolo
    J-->>V: Eventos y resultado de prueba
    V->>A: Publica informe sanitizado
    A-->>W: Actualiza estado
    W-->>C: Muestra aprobado o rechazado
~~~

### Garantías

- El registro se crea únicamente después de completar el artefacto.
- La validación examina el mismo digest que luego será elegible.
- Los logs no contienen secretos de otros participantes.
- Un fallo técnico puede reintentarse; un fallo del agente produce un informe nuevo.

## 6. Secuencia de ejecución por ticks

~~~mermaid
sequenceDiagram
    actor O as Operador
    participant A as API
    participant Q as Cola de partidas
    participant W as Worker
    participant G as Juego aislado
    participant X as Agente A
    participant Y as Agente B
    participant E as Registro de eventos
    participant L as Vista en vivo

    O->>A: Inicia partida programada
    A->>A: Autoriza y congela snapshot
    A->>Q: Encola partida
    Q->>W: Reserva trabajo
    W->>W: Verifica digests y recursos
    W->>G: Inicia juego con configuracion
    W->>X: Inicia artefacto A sin red
    W->>Y: Inicia artefacto B sin red

    loop Cada tick
        G-->>W: Percepcion privada por slot
        par Decisiones con presupuesto independiente
            W->>X: Percepcion A
            X-->>W: Accion A
        and
            W->>Y: Percepcion B
            Y-->>W: Accion B
        end
        W->>G: Acciones o consecuencias de timeout
        G-->>W: Eventos y nuevo estado
        W->>E: Persiste lote del tick
        E-->>L: Publica proyeccion permitida
    end

    G-->>W: Causa de finalizacion
    W->>E: Cierra registro y calcula digest
    W->>A: Resultado provisional y reporte
    A-->>L: Resultado y replay disponibles
~~~

### Garantías

- Los agentes no se comunican entre sí.
- Cada agente recibe solo su percepción.
- Los presupuestos de decisión son independientes.
- El motor del juego es la autoridad sobre acciones y finalización.
- La vista en vivo consume una proyección; no controla la partida.
- El resultado se acepta solo si el registro queda completo.

## 7. Secuencia de reclamo y resolución

~~~mermaid
sequenceDiagram
    actor C as Concursante
    participant A as API
    participant R as Arbitro
    participant E as Evidencia
    participant K as Clasificacion

    C->>A: Presenta reclamo
    A->>A: Verifica plazo y alcance
    A->>E: Inmoviliza evidencia relacionada
    A-->>R: Asigna incidente
    R->>E: Consulta logs, digests y reglas
    R->>A: Emite decision justificada
    alt Mantener resultado
        A->>A: Confirma resultado
    else Corregir
        A->>A: Crea nueva version del resultado
    else Repetir
        A->>A: Anula intento y programa otro
    end
    A->>K: Recalcula si corresponde
    A-->>C: Notifica resolucion
~~~

## 8. Flujo público del evento

1. El espectador entra sin conocer detalles técnicos.
2. Ve el concurso actual, su propósito y el próximo momento importante.
3. Abre una categoría y comprende objetivo, reglas resumidas y participantes.
4. Consulta la clasificación y evolución permitida.
5. Durante la final abre la vista en vivo preparada para proyección.
6. La interfaz destaca cambios relevantes sin revelar información privada.
7. Al terminar ve resultado, estadísticas explicadas y clasificación actualizada.
8. Después puede abrir el replay si el registro está completo y autorizado.

