# Cuestionario maestro para definir Agentrix desde cero

> Este documento parte de una hoja en blanco. No presupone que deban conservarse el producto, la arquitectura, las entidades, los flujos ni las tecnologias de intentos anteriores.

## Como responder

1. Escribe debajo de **Respuesta** con toda la extension que necesites.
2. Comienza cada respuesta con uno de estos estados:
   - `[DECIDIDA]`: quieres que esta respuesta forme parte del diseño.
   - `[PROVISIONAL]`: es tu preferencia actual, pero puede depender de otras decisiones.
   - `[PENDIENTE]`: aun no tienes informacion suficiente.
   - `[NO APLICA]`: no corresponde al sistema; explica brevemente por que.
3. No necesitas responder en orden estricto, pero las secciones estan organizadas por dependencia.
4. Si una pregunta menciona algo que ya definiste, referencia su identificador —por ejemplo, `AR-018`— en lugar de repetir la respuesta.
5. Separa lo que deseas de lo que estas dispuesto a aceptar como compromiso.
6. Describe ejemplos concretos cuando una regla pueda interpretarse de varias maneras.

## Principios del cuestionario

- Cada pregunta cubre una decision diferente.
- Las preguntas tecnicas aparecen despues de las preguntas del producto y del dominio.
- Una pregunta no obliga a aceptar la existencia del concepto que menciona.
- Las cuestiones condicionales pueden marcarse como `[NO APLICA]`.
- Las contradicciones no se resolveran silenciosamente: se registraran y volveran como decisiones explicitas.
- Las respuestas seran la fuente para producir los diagramas; los diagramas no inventaran respuestas.

## Que se obtendra de las respuestas

| Secciones | Resultado derivado |
| --- | --- |
| 1–3 | Vision, actores y experiencia completa |
| 4–10 | Casos de uso, reglas, permisos y flujos |
| 11 | Glosario y diagrama conceptual de clases |
| 12 | Arquitectura de informacion y direccion visual |
| 13–15 | Requisitos de datos, calidad y arquitectura tecnica |
| 16–17 | Alcance, hoja de ruta y proceso de construccion |
| 18 | Revision de coherencia y asuntos pendientes |

---

# 1. Identidad, proposito y limites

## AR-001 — Definicion esencial

Si tuvieras que explicar Agentrix a una persona que nunca ha oido hablar del proyecto, ¿que dirias que es en una sola frase y que aclararias inmediatamente despues para evitar una interpretacion equivocada?

**Respuesta:**

concurso de agentes donde cada participante programa su propio agente para competir contra otros agentes, la idea es que al participante se le permita subir su propio agente que el considere, donde este pueda subir varios como mejoras a su anterior, otro nuevo completamente distinto con un limite de tiempo para poder subir el siguiente, donde este se enfreta a un cantidad limita por prueba a otros agentes que posen su propia logica, estos se encuentran en entornos que pueden ir cambiando en cada prueba, y estas partidas como tal pueden estar involucradas por factores aletorios que pondran a prueba la logica de lo agentes, no es algo que se puede decir deterministico, si no donde los mejores agentes casi siempre van a estar en el top, entonces para que sea mas emocionante entra el factor suerte que hace mas emocionante pero a la vez no afecta tanto

## AR-002 — Problema u oportunidad

¿Que problema real, necesidad u oportunidad origina el proyecto? Describe que ocurre hoy sin Agentrix, quien lo padece y por que merece construirse una solucion.

**Respuesta:**

el problema no hay una plataforma que pueda gestionar toda esta idea, la necesidad de no encontrar alternativas que cumplan con las espectativas para esto, la oportunidad que nace es que podemos hacer que este hecho a nuestra medida y exigencias, un control total sobre todo, 

## AR-003 — Valor para sus beneficiarios

¿Que cambio valioso debe producir Agentrix para cada grupo que se beneficie de el? Distingue beneficios practicos, educativos, competitivos, sociales y de entretenimiento cuando correspondan.

**Respuesta:**

el poder organizar concursos dentro de la universidad donde se puede mostrar desde un punto de vista distinto todo esto de los modelos de ia, agentes y ya no verlos solo como simples predictores de datos si no que tambien pueden ser aplicados a entornos de entretenimiento como los juegos entornos que pongan a prueba el entrenamiento de cada uno de los modelos y puedan mejoras sus enfoques cambios y estrategias

## AR-004 — Prioridad del producto

Si hubiera conflicto entre organizar competencias, evaluar agentes con rigor, facilitar el aprendizaje y crear un espectaculo atractivo, ¿como ordenarias esas prioridades y que sacrificios no aceptarias?

**Respuesta:**

evaluar agentes con rigor, facilitar el aprendizaje, crear un espectaculo atractivo, organizar competencias . prefiero realmente que la parte tecnica y logica salga perfecta sacrificando todo lo demas

## AR-005 — Principios no negociables

¿Que cualidades o principios debe respetar el sistema incluso si encarecen o retrasan su desarrollo? Explica por que cada uno es imprescindible.

**Respuesta:**

el poder gestionar toda la plataforma de manera que no haya conflictos logicos, ahora el sistema que va a ser capaz de manejar cualquier juego que se desee y donde cada jugador sea manejado por un agente o ia, y este sea eficiente y sencillo, lograr proporcionar todas la herramientas a los agentes para puedan desarrollarse sin poblemas dentro de los juegos y partidas sin errores y problemas

## AR-006 — Fuera de alcance

¿Que productos, funciones, publicos o problemas se parecen a Agentrix, pero deseas excluir deliberadamente? Incluye aquello que no quieres que el proyecto llegue a convertirse.

**Respuesta:**

como tal no se busca hacer una superplataforma capaz de albergar concuros gigantes lleno de muchas cosas, si que solo cumpla con los objetivos como para unas 100 participantes

---

# 2. Actores, responsabilidades e intereses

## AR-007 — Mapa de actores

¿Que tipos de personas, grupos u organizaciones interactuan con el sistema o son afectados por el? Para cada actor, indica su objetivo principal y si actua dentro o fuera de la plataforma.

**Respuesta:**

en su gran parte son estudiante de ingenieria
administradores: tiene control absoluto sobre el sistema
concursantes: solo tiene deciciones sobre el en el concurso
espectadores: solo puede ver como es que se estan desarrollando las cosas pero no puede interferir en nada

## AR-008 — Unidad participante

¿Quien compite realmente: una persona, un equipo, una institucion, un agente o alguna combinacion? Explica como se representa publicamente y quien responde por sus acciones.

**Respuesta:**

como tal todo esto depende de el concurso puede haber concurso donde la participacion es individual o en equipo, ademas estos pueden representar a alguna institucion academica

## AR-009 — Autoridad organizadora

¿Quien puede crear y dirigir una competencia, que decisiones puede tomar y ante quien responde? Distingue organizacion academica, administracion tecnica y autoridad sobre el reglamento si son responsabilidades diferentes.

**Respuesta:**

el cominite organizar con respecto todo aquello que requiero toma de deciones sobre un concurso y sus bases

## AR-010 — Creadores y operadores especializados

¿Se necesitan roles separados para diseñar juegos, redactar reglas, revisar agentes, operar ejecuciones, arbitrar resultados o producir la transmision? Define que responsabilidad no deberia combinarse en una sola persona.

**Respuesta:**

admin: diseñar juegos, redactar reglas, revisar agentes, operar ejecuciones, arbitrar resultados o producir la transmision
colaborador: diseñar juegos, redactar reglas, revisar agentes
espectador: arbitrar resultados o producir la transmision

## AR-011 — Publico y partes externas

¿Que clases de espectadores, patrocinadores, jurados, docentes, instituciones u otras partes externas deben considerarse? Indica que pueden observar, aportar, exigir o decidir.

**Respuesta:**

todos pueden observar, pero solo el comite organizador puede tomar las deciciones sobre aportar, exigir o decidir.

---

# 3. Experiencia completa del evento

## AR-012 — Historia principal de extremo a extremo

Narra un caso ideal desde que se anuncia una competencia hasta que se reconoce al ganador y se conserva su historia. Evita terminos tecnicos; describe personas, decisiones, acontecimientos y resultados.

**Respuesta:**

el ultimo de el evento recien se conocera a el ganador y esto sera en vivo en un entorno preparado para este momento un escenario nuevo frente a los mejores participantes durante el torneo y de ahi recien damos desarrollo en vivo a toda esta experiencia, asi nadie sabe quien va a ganar ni los admins ni el comite y nadie, en ese momento se conoce el desenlace, frente a todo el publico en pleno auditorio

## AR-013 — Descubrimiento y entrada

¿Como descubre una persona la plataforma o una competencia, que necesita entender antes de participar y que pasos debe completar para comenzar?

**Respuesta:**

se va a realizar publicidad por toda la unversidad y se proporciona las bases para concer mas sobre la edicion de este evento

## AR-014 — Preparacion previa

¿Que hacen organizadores, participantes y responsables del juego antes del inicio? Define que debe estar listo, que se comprueba y quien autoriza el paso a la siguiente etapa.

**Respuesta:**

debe de estar listo todo el entorno para dar inicio a el evento, tanto el juego como los agentes de prueba que porcionan los organizadores, los participantes deben de ir probando durante el concurso en cualquier momento de la duracion sus modelos, los responsables deben de vigilar que nada se rompa o corregir reportar los fallos

## AR-015 — Jornada o periodo competitivo

¿Que sucede durante la competencia desde la perspectiva de cada actor? Describe el ritmo, los puntos de decision, la comunicacion y aquello que hace reconocible el momento central del evento.

**Respuesta:**

los administradores se encargan de todo lo tecnico viendo que nada falle, los partipantes prueban y mejoran sus modelos, los espectadores tambien pueden ver como se va desarrollando que cambios a habido

## AR-016 — Cierre y continuidad

¿Que debe suceder despues de la ultima partida? Incluye publicacion de resultados, reconocimientos, reclamos, acceso historico, aprendizaje y preparacion de futuras ediciones.

**Respuesta:**
puede verse los resultados atravez de los dias y como han ido cambiando, el dia de el cierre se debe de tener algo mas pomposo y llamativo, donde al terminar la partida final se queden los tops de el concurso y ademas poder acceder alos datos y modelos de el concursos, tambien dar sugerencias y reclamos

## AR-017 — Frontera entre plataforma y mundo real

¿Que partes de la experiencia deben ocurrir necesariamente dentro de Agentrix y cuales pueden seguir siendo manuales o externas? Explica que informacion debe regresar al sistema cuando algo sucede fuera de el.

**Respuesta:**

el moverse por la plataforma como los menus opciones personalizacion de perfil revisar cosas ocurren fuera de el alcance de los agentes o ias, los modelos solo pueden reaccionar y funcionar dentro de ese entorno de el juego al inciar y terminar la partida, luego no pueden hacer nada y simplemente se guadan los datos y resultados 

---

# 4. Competencias, ediciones y participacion

## AR-018 — Conceptos organizativos

¿Que significan para ti terminos como competencia, concurso, torneo, evento, edicion, temporada, categoria, fase y ronda? Define cuales existen, cuales son sinonimos y como se contienen entre si.

**Respuesta:**

competencia: a el ranking en general durante el desarrollo los detalles de este y como se va desarrollando, tecnicamente engloba todo. 
concurso , torneo, evento, edicion, temporada : serian lo mismo presentado a la competencia como tal mas iria como el nombre y aspectos mas simples .
categoria: separacion entre participantes donde es decidida por el comite.
fase: cambios de la competencia durante el desarrollo de el concurso y representar etapas donde se determinan modificaciones por el comite.
ronda: a la partida o la separacion como las llaves en un torneo

## AR-019 — Ciclo de vida de una competencia

¿Por que estados atraviesa una competencia desde su idea inicial hasta su archivo? Para cada transicion importante, indica quien la provoca y que condicion debe cumplirse.

**Respuesta:**

todo los hacen los administradores inaguracion donde se muestra de que va esta edicion y una demostracion de como es que funciona y que se puede hacer con un ejemplo de prueba en vivo, durante la competencia la subida de modelos y los resultados de este, para el cierre la revelacion en vivo, todo esto ocurren en tiempos establecidos y personalizables en el sistema 

## AR-020 — Elegibilidad e inscripcion

¿Quien puede participar, que requisitos debe demostrar, como solicita o confirma su inscripcion y en que circunstancias puede ser aceptado, rechazado o retirado?

**Respuesta:**

sus datos personales como nombre y apellido universidad, se incribe y realiza un pago simbolico y despues es evaluado para ser puesto en alguna categoria correspondiente, no se rechaza a nadie a participar

## AR-021 — Formacion y cambio de equipos

**Condicional:** si existen equipos, ¿como se crean, quien los representa, como ingresan o salen miembros y que cambios se permiten despues de inscribirse o comenzar la competencia?

**Respuesta:**

en la incripcion forman el equipo inmutable por el resto de el concurso siempre y cuando los particpantes cumplan con los requisitos para participar, todo estos participantes son asignados a el id de el equipo

## AR-022 — Estructura competitiva

¿Que formatos debe poder expresar el sistema —por ejemplo, exhibicion, liga, grupos o eliminacion— y que elementos son comunes a todos? No elijas todavia un algoritmo; define la flexibilidad necesaria.

**Respuesta:**

todos pueden esos aspectos, mas no modificadorlos

## AR-023 — Calendario y coordinacion

¿Como se determinan fechas, rondas, emparejamientos y disponibilidad? Describe que puede cambiar, quien recibe avisos y que ocurre ante retrasos o ausencias.

**Respuesta:**

todo puede cambiar y se les notifica a los participantes en la plataforma

## AR-024 — Publicacion y cambio de reglas

¿Cuando se conocen y quedan fijadas las reglas de una competencia? Define como se corrigen errores, como se comunican los cambios y que proteccion reciben quienes ya desarrollaron un agente bajo una version anterior.

**Respuesta:**

las reglas quedan fijadas y ya si hay vacios legales son a decicion de el comite

---

# 5. El juego como concepto

## AR-025 — Definicion de juego

¿Que es un juego dentro de Agentrix y que debe proporcionar para ser utilizable? Distingue sus reglas, estado, acciones, condiciones de finalizacion y representacion observable.

**Respuesta:**

como tal este debe de proporcionar los datos o parametros que persive solo el jugador y no el de los demas, por el momento el mismo juego proporciona esos datos, se planea estar preparado para modelos de vision computacional por el momento

## AR-026 — Creacion y publicacion de juegos

¿Quien puede diseñar, implementar, revisar, aprobar, publicar, retirar o reemplazar un juego? Describe el recorrido desde una idea hasta un juego disponible para competencias.

**Respuesta:**

los administradores son responsables de todo eso,deben de haber todos esos procesos con el flujo estandar y clasico

## AR-027 — Identidad y evolucion de un juego

¿Que cambios hacen que siga siendo el mismo juego y cuales crean una version o un juego diferente? Explica que compatibilidad esperas entre agentes, partidas, resultados y visualizaciones de distintas versiones.

**Respuesta:**

solo la versionamos por github osea solo queda la actual

## AR-028 — Partes que componen un juego

¿Reglas, protocolo de comunicacion, motor, mapas, recursos visuales, visor y formato de replay forman una unidad inseparable o pueden evolucionar por separado? Define el comportamiento deseado, no la implementacion.

**Respuesta:**

pueden ir cambiando por separado

## AR-029 — Aleatoriedad y conocimiento del entorno

¿Que papel puede tener el azar? Indica que conocen los agentes antes y durante una partida, que informacion permanece oculta y que debe poder reproducirse posteriormente.

**Respuesta:**

al azar solo influye en aspectos mapas distintos, espawn de mounstrous pero estos como tal no aparecen de nada si no en zonas q no afecten a los jugadores de manera improvista, como tal los agentes solo pueden persivir el entorno solo desde su punto de vista no pueden tener un conocimiento total, todo esto haria q sea difil hacer un repeticion deterministica, por lo que seria mas logico un sistema de hacer las repeticiones por un guardados de logs guardados por tics y con esto poder reproducir el mismo escenario

## AR-030 — Reutilizacion y combinacion de juegos

¿Un juego puede usarse en multiples competencias o una competencia puede incluir multiples juegos? Explica como deberian compararse o conservarse sus resultados sin asumir que todos puntuan igual.

**Respuesta:**

si es posible, podria darse el caso que haya un categoria que si o si pide modelos por vision computacional y otro que se le entregan todos los parametros y en base a todo eso decir de manera mas facil, es decir una para personas no tienen tanta experiencia con estos temas, o tal  vez el juego de una categoria no puede ser reproducido para otra en base a las dificultades entonces se plantea otro, ademas con respecto a las puntuaciones deben ser guadadas todas las estadisticas de todos los participantes no se puede dejar a nadie fuera

---

# 6. Agentes, desarrollo y presentacion

## AR-031 — Definicion y limites de un agente

¿Que consideras un agente valido? Aclara si puede usar reglas escritas a mano, busqueda, aprendizaje automatico, modelos externos, memoria, intervencion humana u otros recursos.

**Respuesta:**

si todo eso es posible si se encuentra dentro de los limites establecidos para juego, puede que haya competidores que quieren participar con algortimos programados o otros que decidan entrenar su modelo con machine learning, si se permite tener memoria tambien se implementa o se activa esa opcion en el concurso

## AR-032 — Experiencia de creacion

¿Que recorrido deberia seguir una persona desde que conoce un juego hasta que logra ejecutar su primer agente valido? Describe documentacion, ejemplos, herramientas y ayuda que esperaria recibir.

**Respuesta:**

se deberia de ofrecer toda la documentacion sobre como se usa la plataforma y dentro de cada concurso se los detalles correspondientes a los juegos o juego ofrecido, con las limitaciones que se debe de cumplir, un sistema que consiga validar aspectos generales sobre sus agentes para ser considerado un envio valido 

## AR-033 — Contrato de interaccion con el juego

¿Que informacion recibe un agente, en que momento, que respuesta debe producir y como se interpreta? Incluye inicializacion, acciones sucesivas, finalizacion y errores de comunicacion.

**Respuesta:**

recibe todas las percepciones que se encuentran los limites de el juego, todo esto por tics, de manera que cada no haya el problema de que un agente ocupo mas recursos para decidir que otro y los demas se vean afectados, entonces se calcula todo lo correspondiente a cada tic de manera que nadie queda fuera y no se rompe nada

## AR-034 — Entornos y dependencias admitidas

¿Que lenguajes, versiones, bibliotecas, modelos, archivos o servicios externos deben permitirse? Explica si la plataforma debe ser abierta a muchos entornos o comenzar con un conjunto limitado.

**Respuesta:**

como tal siempre va a estar asociado a un conjunto limitado por el comite, entonces si luego de un revision de el modelo y no cumple los requisitos puede ser anulado el envio y dar como no valido

## AR-035 — Objeto presentado

¿Que entrega exactamente el participante: codigo, ejecutable, paquete, contenedor, repositorio, endpoint u otra cosa? Indica metadatos, responsable, licencia, visibilidad y evidencia de autoria necesarias.

**Respuesta:**

todo esto lo determina el comite, pero las opciones y el soporte para ofrecer todo esto debe de estar disponible, osea tiene que ser personalizable y configurable

## AR-036 — Validacion y retroalimentacion

¿Que comprobaciones debe superar una presentacion antes de competir y que informacion recibe el participante cuando alguna falla? Distingue errores corregibles de causas de rechazo definitivo.

**Respuesta:**

se le debe de proporcionar toda esa informacion sobre que paso y poder reproducir ese error y solucionarlo, los logs de el fallo o el motivo por cual se considero el fallo

## AR-037 — Versiones, plazos y conservacion

¿Cuantas veces puede presentarse un agente, cual version compite, cuando queda congelada y que se conserva despues? Incluye retiros, reemplazos y visibilidad historica.

**Respuesta:**

todos los envios quedan sugetos a un participante y este puede considerar participar con su mejor peor o regurlar version, si el lo considera, pero solo puede participar con un modelo el dia de el cierre, para tiene que confirmar con cual decide participar mientras este sea un modelo aprobado o probado durante la competencia, no se puede participar con modelo que nunca paso por ahi, entonces no se admiten modelos sorpresa

---

# 7. Partidas, simulacion y ejecucion

## AR-038 — Definicion de partida

¿Que elementos hacen que una partida sea una unidad identificable? Incluye participantes, juego, configuracion, condiciones iniciales, momento, proposito y resultado esperado.

**Respuesta:**

si incluye todo eso

## AR-039 — Ciclo de vida de una partida

¿Que estados atraviesa una partida desde que se propone hasta que su resultado queda confirmado? Define quien o que inicia cada transicion y cuales permiten volver atras.

**Respuesta:**

un tiempo limite por partida para evitar bucles, los agentes toman deciciones por tics, todo inicia cuando el tiempo comienza a correr y este termina o ya no hay participantes en la partida

## AR-040 — Modelo temporal

¿La interaccion ocurre por turnos, pasos simultaneos, tiempo real u otro modelo? Describe como avanza el juego y como se evita que la velocidad del hardware otorgue una ventaja injusta.

**Respuesta:**

todo esta controlado por tics y en un tiempo real definido

## AR-041 — Limites de recursos

¿Que limites deben aplicarse a tiempo de respuesta, CPU, memoria, almacenamiento, tamaño, duracion total u otros recursos? Explica si dependen del juego, la competencia o la infraestructura disponible.

**Respuesta:**

como tal se debe de dar limites bien definidos para los recursos que puede consumir cada agente no pueden pasar de ese limite si no estos mismos se veran perjudicados en el momento de la evaluacion

## AR-042 — Fallos y acciones invalidas

¿Que resultado produce un timeout, bloqueo, caida, salida incorrecta, accion ilegal o abandono? Distingue fallos del agente, del juego, de la plataforma y de la infraestructura.

**Respuesta:**

se deben se parar esos tres aspectos para poder ofrecer la mejor experiencia y no ocurra problemas, asi de esta manera tambien encontrar de manera mas sencilla el problema

## AR-043 — Aislamiento y capacidades

¿Que puede leer, escribir o contactar un agente mientras se ejecuta? Define las capacidades necesarias y las prohibidas respecto a red, archivos, procesos, secretos y otros participantes.

**Respuesta:**

tienen que estar controladas en un entorno donde no pueden acceder a internet archivos de el juego y informacion con respecto a la partida que no deberian de conocer

## AR-044 — Registro y reproducibilidad

¿Que hechos deben registrarse para explicar, auditar o reproducir una partida? Indica que grado de igualdad esperas al repetirla y que informacion puede mantenerse privada.

**Respuesta:**

se puede proporcionar todos los datos, pero no modelos ni codigo simplemente datos recolectados y generados durante la partida

## AR-045 — Colas, capacidad y reintentos

¿Como deberia comportarse el sistema cuando hay mas partidas que capacidad disponible? Define prioridades, espera aceptable, reintentos y el tratamiento de ejecuciones duplicadas o interrumpidas.

**Respuesta:**

se debe de poner en cola de espera hasta que las condiciones fisicas sean consideradas aptas para que pueda iniciar la partida correspondiente en la cola

---

# 8. Resultados, clasificacion y justicia

## AR-046 — Resultado de una partida

¿Que produce una partida ademas de un ganador o perdedor? Define puntuaciones, estadisticas, causas de finalizacion y evidencia necesaria para interpretar el resultado.

**Respuesta:**

se proporciona todo eso mencionado mientras no sean modelos o codigo de otros participantes

## AR-047 — Clasificacion y desempate

¿Como se agregan resultados para ordenar participantes y como se resuelven empates? Explica que debe poder variar entre competencias sin fijar todavia una formula universal.

**Respuesta:**

se hace en  base a los parametros de calificacion para cada juego, en caso de empate se coloca a ambos en la misma posicion y un posible desempate en caso no se llegue a un acuerdo entre competidores

## AR-048 — Condiciones comparables

¿Como se garantiza que los participantes enfrenten condiciones suficientemente equivalentes? Considera posiciones, mapas, rivales, semillas, orden, cantidad de partidas y variacion estadistica.

**Respuesta:**

se les da un tiempo considerable para poder preparse y elegir estrategias y mejoras que hagan mas sencillo acortar las brechas, ademas no hay factores de suerte que hagan conseguir ventajas injustas si no en base mas que todo a las deciciones de cada agente, cada escenario es evaluado y preparado para evitar esos problemas

## AR-049 — Integridad competitiva

¿Que conductas constituyen trampa, colusion, explotacion indebida o ventaja prohibida? Define que se previene automaticamente, que se investiga y que evidencia se necesita.

**Respuesta:**

cualquier cosa que vulnere informacion personal o acceso a modelos de otros participantes, modelos muy potentes que no cumplan con las limitaciones inpuestas en el concurso

## AR-050 — Reclamos y repeticion de partidas

¿Quien puede cuestionar un resultado, dentro de que plazo y mediante que proceso? Define cuando procede corregir, repetir, anular o mantener una partida.

**Respuesta:**

los administradores que posiblemente tambien sean parte de el comite

## AR-051 — Finalidad e historial de resultados

¿Cuando se considera definitivo un resultado y que puede cambiar despues? Explica como se muestran correcciones, sanciones o recalificaciones sin reescribir silenciosamente la historia.

**Respuesta:**

se considera definitivo cada resultado mientras no hay trampas o errores que pueden haber afectado los resultados y a otros participantes, mientras los errores que un mismo participante provoco y solo afecten a el y no a otros, no se realiza una reevaluacion, mientras no pase nada  de eso se considera definitivo

---

# 9. Experiencia publica, directo y replay

## AR-052 — Experiencia del espectador

¿Que deberia poder comprender y sentir un espectador que nunca ha programado? Describe que hace interesante seguir una partida y que informacion seria ruido innecesario.

**Respuesta:**

entender calaramente lo que trata el juego y en base a esto difrustar como los agentes van actuando y tomando deciones atravez de los agentes y poder mostrar claramente las escenas que se consideren interesantes osea tener un sistema capaz de detectar y darle mas enfoque a esas partes

## AR-053 — Informacion durante una partida

¿Que debe mostrarse en vivo: arena, participantes, decisiones, estado, estadisticas, eventos destacados u otros datos? Ordena la informacion por importancia.

**Respuesta:**

la partida en general y algunos datos que se consideren interesantes

## AR-054 — Directo, retraso y spoilers

¿La visualizacion debe ser realmente en vivo, retrasada o publicada al terminar? Define como afectan la integridad competitiva, los problemas de conexion y las distintas audiencias.

**Respuesta:**

en vivo una solo partida a la vez

## AR-055 — Replay

¿Que debe permitir una repeticion: pausar, acelerar, retroceder, cambiar camara, inspeccionar estados, ocultar informacion o comparar agentes? Define tambien quien puede verla y desde cuando.

**Respuesta:**

si puede permitir todo eso, pero solo en el caso que toda la informacion completa no se puede permitir opciones que no tengan toda la informacion completa para hacer eso y no romper nada en el proceso

## AR-056 — Perfiles, clasificaciones e historia publica

¿Que informacion publica debe existir sobre participantes, equipos, agentes, juegos, competencias y resultados anteriores? Aclara que informacion no debe convertirse en perfil publico.

**Respuesta:**

se debe de permitir todo eso pero, siempre y cuando este en los limites establecidos durante el concurso

## AR-057 — Produccion y difusion

¿La plataforma debe apoyar narracion, comentarios, pantallas para el recinto, transmision, clips, enlaces compartibles o integracion en otros sitios? Prioriza los usos reales esperados.

**Respuesta:**

solo mostrar la partida para ser compartida en la pantalla principal de el auditorio no se requiere algo tan complejo

---

# 10. Administracion, moderacion y operacion humana

## AR-058 — Consola de organizacion

¿Que necesita observar y controlar un organizador antes, durante y despues de una competencia? Separa configuracion normal de acciones excepcionales o peligrosas.

**Respuesta:**

los permisos se espera manejar permisos dinamicos por participantes administradores moderadores y todo eso, asi es mas facil administrar quienes pueden hacer o no las cosas

## AR-059 — Permisos

¿Que decisiones o datos requieren permisos diferentes? Construye una primera matriz indicando actor, accion permitida, alcance y condiciones especiales.

**Respuesta:**

con los permisos dimanicos con respecto a los cruds asignados a cada entidad o tabla de el modelo relacional

## AR-060 — Moderacion y sanciones

¿Que contenidos o conductas pueden moderarse, quien decide y que sanciones existen? Incluye advertencia, ocultamiento, descalificacion, suspension y apelacion cuando correspondan.

**Respuesta:**

en el caso de trampas o fallos tecnicos

## AR-061 — Intervencion excepcional

¿Que acciones manuales deben existir para resolver incidentes sin corromper el historial? Define que confirmaciones, justificaciones y registros exige cada intervencion.

**Respuesta:**

backups y hacer todo con operacion en espacios controlados

## AR-062 — Alcance organizacional y soporte

¿Agentrix sirve a una sola organizacion o a varias organizaciones independientes? Explica quien administra la plataforma global, quien brinda soporte y como se aislan responsabilidades y datos.

**Respuesta:**

ahora sirve solo para una y la administro yo como autor de la ide y desarrollador

---

# 11. Modelo conceptual y diagrama de clases

> Esta seccion consolida conceptos descubiertos en las secciones anteriores. Referencia respuestas previas en vez de volver a narrarlas.

## AR-063 — Lenguaje comun

¿Cual es el termino oficial para cada concepto del dominio y que terminos alternativos deben evitarse? Señala palabras que hoy sean ambiguas o tengan significados diferentes segun el actor.

**Respuesta:**

no nos compliquemos tanto mientras se entienda todo normal

## AR-064 — Entidades con identidad propia

¿Que conceptos deben poder reconocerse como el mismo objeto a lo largo del tiempo aunque cambien sus datos? Para cada uno, indica que lo identifica y cuando nace o deja de existir.

**Respuesta:**

todo esta presentado en el modelo relacional y de que dependen cada uno y como es que se deben de relacionar

## AR-065 — Valores sin identidad

¿Que conceptos se definen completamente por su valor y podrian reemplazarse por otro equivalente? Considera puntajes, periodos, limites, posiciones, configuraciones y otros valores compuestos.

**Respuesta:**

todo viene definido por el modelo relacional y si es que una oracion expresa correctemente la idea principal no hay problemas

## AR-066 — Pertenencia y composicion

¿Que objetos existen por si mismos y cuales solo tienen sentido como parte de otro? Indica que deberia ocurrir con los componentes cuando su propietario se archiva, elimina o reemplaza.

**Respuesta:**

nada se debe borrar por eso usamos una columna de active para determinar ese tipo de situaciones, borrar o eliminar no es una opcion, remplazar si lo es pero solo actualizando valores modificables que no rompan el modelo relacional

## AR-067 — Relaciones y cardinalidades

Para las entidades identificadas, ¿que relaciones son obligatorias, opcionales, uno a uno, uno a muchos o muchos a muchos? Describe tambien relaciones cuyo significado cambie con el tiempo.

**Respuesta:**

todas estas pueden exisitir y como tal se da por las relaciones que hay entre ellas por un id como claves primarias foraneas y todo eso 

## AR-068 — Invariantes del dominio

¿Que afirmaciones deben ser siempre verdaderas para que el sistema siga siendo coherente? Vincula cada regla con las entidades responsables de protegerla.

**Respuesta:**

validar cada cosa que se da va a establecer verificar los datos y cambios a realizar, validando todo y cada uno de los atributos

## AR-069 — Tiempo, versiones e historia

¿Que relaciones o atributos representan el estado actual y cuales deben conservar el estado que era valido en un momento historico? Indica donde una referencia viva seria peligrosa.

**Respuesta:**

todo debe de estar guardado y respaldado en logs en caso sean cambios muy importantes que solo se puedan hacer por medio de la manipulacion directa de la base de datos

## AR-070 — Eventos del dominio y vistas del modelo

¿Que acontecimientos importantes deberian tener nombre propio porque cambian el estado o interesan a varios actores? Despues, indica que grupos de entidades deberian aparecer juntos en el mapa global y cuales necesitan un diagrama separado.

**Respuesta:**

tecnicamente todo deberia de estar conectado, simplemente deben de estar aisalado la manipulacion de la plataforma por medio de una partida, y la manipulacion de una partida por medio de la plataforma

---

# 12. Experiencia de usuario y lenguaje visual

## AR-071 — Arquitectura de informacion

¿Como deberia organizarse la informacion desde la perspectiva de cada actor? Define los destinos principales de navegacion y que concepto sirve como punto de entrada a los demas.

**Respuesta:**

los administradores deberia poder admirar toda la informacion y los participantes solo aquella que le corresponda, todo esto con respecto a los permisos dinamicos con respecto a que pueden ver editar actualizar crear y cambiar

## AR-072 — Pantallas y tareas criticas

¿Que pantallas son necesarias para completar los recorridos esenciales? Para cada una, indica la tarea principal, informacion imprescindible y accion dominante.

**Respuesta:**

hay una pantalla principal al hacer login, y luego se puede visualizar todo aquello a lo que se tiene permiso, todo esto separado por secciones donde se agrupan las cosas relacionadas, si no tiene permisos no se puede ver esa informacion, como no tendria ni conocimiento de que existen esas opciones

## AR-073 — Personalidad visual

¿Que adjetivos deben describir la plataforma y que sensaciones debe evitar? Proporciona referencias visuales, productos, eventos o estilos que ayuden a reconocer ambos extremos.

**Respuesta:**

solo centracer en el concurso en ofrecer opciones completas pero sencillas, no ofrecer cosas como venda productos y ese tipo de cosas, se puede dar una pequeña libertad de personalizacion en ciertos aspectos esticos que hagan sentir un tanto especial al usuario

## AR-074 — Jerarquia y densidad

¿Que informacion debe destacar inmediatamente y cual puede revelarse bajo demanda? Explica el nivel de densidad adecuado para publico, participantes y organizadores.

**Respuesta:**

darle importancia al evento mas que todo y como participar y todos los detalles de el concurso correspondiente todo bien organizado y sin saturar de informacion a los usuarios

## AR-075 — Lenguaje visual de la arena

¿Como deben representarse agentes, equipos, acciones, terreno, peligro, progreso, daño, eventos y resultado para comprender la partida sin depender unicamente de texto o color?

**Respuesta:**

depende de el tipo de concurso y juego, pero todo esto debe de ser facil de entender usar colores y temas sencillos y pasteles, osea temas claros, lo oscuro deprime y cansa

## AR-076 — Dispositivos y contextos de uso

¿Donde se usara la plataforma: laptop, movil, proyector, pantalla del evento, transmision u otros contextos? Prioriza tamaños, controles y condiciones de red que el diseño debe soportar.

**Respuesta:**

en dispositivos como latop y moviles, buscar que como tal sea responsive todo

## AR-077 — Accesibilidad, idioma y estados de interfaz

¿Que necesidades de accesibilidad, idiomas, formatos regionales y diferencias culturales deben atenderse? Define ademas como deberian comunicarse carga, vacio, error, espera, bloqueo y exito.

**Respuesta:**

por momento accesible para todos, y el idioma por el momento seria en español, las interfaces todas dependen de los permisos dinamicos establecidos

---

# 13. Datos, identidad, seguridad y privacidad

## AR-078 — Inventario y propiedad de datos

¿Que datos necesita realmente el sistema, quien es responsable de cada conjunto y que nivel de sensibilidad tiene? Distingue datos personales, competitivos, tecnicos, publicos y operativos.

**Respuesta:**

los datos personales como tal son privados y no seran publicos en ningun momento, por eso la seguridad esta dado por un sistema de login y cuentas por usuario

## AR-079 — Conservacion, eliminacion y anonimizacion

¿Durante cuanto tiempo debe conservarse cada clase de informacion y que significa eliminarla? Explica que historia debe permanecer, que puede anonimizarse y que obligaciones tiene el sistema ante una solicitud de borrado.

**Respuesta:**

se conserva solo la informacion que no afecte el desarrollo de la competencia para futuras ediciones o concursos, lo demas que no represente romper el sistema puede ser borrado y anonimato si es que se requiere

## AR-080 — Requisitos de identidad y acceso

¿Que necesita demostrar una persona para realizar acciones sensibles? Define requisitos de registro, verificacion, ingreso, recuperacion, cierre de sesion y revocacion sin decidir todavia el mecanismo tecnico.

**Respuesta:**

solo se puede actuar sobre la informacion de cada cuenta personal, no sobre otras personas

## AR-081 — Amenazas y abusos relevantes

¿De quien y de que debe protegerse la plataforma? Considera suplantacion, robo de cuenta, codigo hostil, fuga de agentes, manipulacion de resultados, saturacion y abuso administrativo.

**Respuesta:**

no va a ser una superplataforma es algo mas sencillo y pequeño para eventos universitarios

## AR-082 — Secretos y propiedad intelectual

¿Que codigo, modelos, credenciales, estrategias o datos deben mantenerse secretos, incluso frente a organizadores u otros participantes? Define cuando y bajo que autorizacion pueden revelarse.

**Respuesta:**

todo gestionado por medio de los permisos dinamicos

## AR-083 — Trazabilidad, respaldo y obligaciones externas

¿Que informacion debe poder exportarse, respaldarse, restaurarse o auditarse? Incluye requisitos institucionales, legales, de consentimiento o proteccion de menores que puedan aplicar.

**Respuesta:**

mas que todo la informacion correspondiente a las partidas esos datos los importantes a ser respaldados

---

# 14. Requisitos de calidad

## AR-084 — Escala esperada

¿Cuantas organizaciones, competencias, participantes, agentes, partidas simultaneas, espectadores y años de historia esperas en una primera edicion y en un escenario exitoso posterior?

**Respuesta:**

una vez cada 4 meses organizados por la universidad 

## AR-085 — Rendimiento perceptible

¿Que operaciones necesitan respuesta inmediata y cuales pueden tardar o ejecutarse en segundo plano? Define tiempos aceptables desde la experiencia del usuario, no desde una tecnologia concreta.

**Respuesta:**

todo definido por los cruds y los persmisos dinamicos

## AR-086 — Disponibilidad y degradacion

¿En que momentos no puede fallar el sistema y que funciones pueden degradarse temporalmente? Describe que experiencia debe mantenerse si falla la visualizacion, ejecucion, almacenamiento o conexion.

**Respuesta:**

tiene que priorizarse las partidas los demas aspectos pueden quedar atraz, el poder mantener todo funcionando como los modelos y el juego es lo primordial

## AR-087 — Observacion y diagnostico

¿Que necesita conocer el equipo para detectar, explicar y resolver un problema? Distingue informacion util para participantes, organizadores, desarrolladores y operacion.

**Respuesta:**

un sistema de reporte sobre lo que sucedio

## AR-088 — Restricciones de costo y mantenimiento

¿Que limites existen sobre presupuesto, infraestructura, tiempo de administracion, consumo energetico o dependencia de proveedores? Define quien podra mantener el sistema y con que conocimientos.

**Respuesta:**

lo necesario para que funcione todo

---

# 15. Restricciones y decisiones de arquitectura

> Responde esta seccion desde las necesidades ya definidas. Una tecnologia no debe elegirse unicamente por familiaridad o popularidad.

## AR-089 — Tecnologias obligatorias, preferidas o prohibidas

¿Existe alguna restriccion real sobre lenguajes, frameworks, bases de datos, sistemas operativos, licencias o servicios? Para cada una, explica su origen y si es obligacion, preferencia o rechazo.

**Respuesta:**

como tal no pero evitar usar javascripts en backend y mas apego para postgre, lo demas puede ser cualquier cosa que funcione

## AR-090 — Entornos de ejecucion y despliegue

¿Donde debe poder desarrollarse, probarse y operar la plataforma? Incluye equipos disponibles, sistema operativo, red, infraestructura del evento y posibilidad o imposibilidad de usar servicios externos.

**Respuesta:**

en dispositivos con windows y linux, como tal no busca limitar nada 

## AR-091 — Fronteras tecnicas necesarias

¿Que partes necesitan aislamiento por seguridad, fallos, carga, propiedad o ritmo de cambio? Explica cuales podrian convivir inicialmente aunque sean conceptos diferentes.

**Respuesta:**

la cosa es buscar que si se rompre un juego solo afecte a el juego y rompa el servidor ni la plataforma, en el caso de romper la plataforma que este se restablesca y se puede recuperar parte de lo afectado, tambien aislar todo con referente a la base datos

## AR-092 — Persistencia y almacenamiento

¿Que tipos de informacion requieren consultas estructuradas, archivos grandes, transmision temporal, cache o almacenamiento inmutable? Indica durabilidad y recuperacion esperadas antes de elegir productos concretos.

**Respuesta:**

estructurar de manera correcta y bien hecho todo aquello que afecte a otras partes y esos flujos sean los correctos

## AR-093 — Contratos e integraciones

¿Que interfaces necesitan participantes, juegos, visor, administradores o sistemas externos? Define consumidores, estabilidad, autenticacion y politica de cambio esperada para cada contrato.

**Respuesta:**

por el momento todo eso esta bien, tampoco se busca hacer una superplataforma

## AR-094 — Modularidad y legibilidad del codigo

¿Que reglas deberian hacer que la estructura del proyecto sea facil de recorrer y que el codigo resulte agradable de leer? Incluye profundidad de carpetas, nombres, tamaño de componentes, abstracciones, dependencias y documentacion.

**Respuesta:**

que todo esto separado de manera clara y no combinar cosas con respecto a todo eso, evitar en  todo el sentido el harcodeo, mientras todo sea capaz de ser entendible y legible sin logica rara y confusa, todo bien estrucuctura, nada de mesclar archivos con test, ademas hacer un test por cada archivo donde este solo debe de poner a prueba las funciones y logica de el archivo original y nada mas, nada de pruebas generales y mal estructuradas, la documentacion igual

---

# 16. Primera version y evolucion

## AR-095 — Demostracion minima del valor central

¿Cual es la experiencia mas pequeña que demostraria que Agentrix merece existir? Debe poder observarse de principio a fin y no limitarse a infraestructura interna.

**Respuesta:**

debe ser facil de entender y ir a lo que viene, concursos donde los participantes ponen sus modeles a competir en un concurso y apreciar el proceso 

## AR-096 — Resultado minimo por actor

Para cada actor necesario en la primera version, ¿que unico resultado debe poder conseguir? Elimina temporalmente a los actores que no sean indispensables para demostrar el valor central.

**Respuesta:**

poder llevar a hacer una partida con agentes y que esta termine, no importa los resultados, con un sistema de login y cuentas para hacer todo eso

## AR-097 — Primer recorrido vertical

¿Que secuencia completa implementaremos primero, desde una accion visible hasta un resultado visible? Indica datos de entrada, decisiones, fallos basicos y evidencia de exito.

**Respuesta:**

la plataforma funcional con login y cuentas, y poder hacer un envio de un agente y inicie la partida y el agente pueda actuar sin problemas, no importa si el jeugo es sencillo o muy basico, simplemente llegar a ese punto

## AR-098 — Exclusiones de la primera version

¿Que funciones importantes se aplazaran conscientemente y que solucion manual o limitada se usara mientras tanto? Explica por que su ausencia no invalida la demostracion.

**Respuesta:**

toda aquello que se un complemente y no precindible para del desarrollo de una partida con agentes

## AR-099 — Riesgos que requieren experimento

¿Que supuestos tecnicos o de experiencia podrian invalidar el proyecto si fueran falsos? Define el experimento mas pequeño para comprobar cada uno antes de construir alrededor suyo.

**Respuesta:**

que los agentes no puedan desemvolverse de la manera esperada

## AR-100 — Camino de crecimiento

¿Que capacidades deberian añadirse despues de la primera version y que compatibilidad historica o de extensiones debe preservarse desde el inicio? Distingue evolucion probable de posibilidades meramente hipoteticas.

**Respuesta:**

el resto de funcionalidades y personalizacion, refuerzo en partes criticas y mas opciones como las estadisticas y mejoras con respecto a la experiencia

---

# 17. Forma de construir y controlar el proyecto

## AR-101 — Fuentes oficiales de verdad

¿Que documentos o modelos seran autoritativos para producto, dominio, arquitectura, interfaz, contratos y decisiones? Define como detectar que codigo y documentacion se contradicen.

**Respuesta:**

si el codigo no hace lo mismo que la documentacion o tambien puede ser al revez, dado ciertos puntos de funcionamiento esperado

## AR-102 — Decisiones y cambios

¿Quien confirma una decision, como se propone un cambio y como se conserva su justificacion? Define que modificaciones requieren actualizar diagramas, casos de uso o criterios de aceptacion antes de programar.

**Respuesta:**

yo confirmo todo eso, y debe de ser validado de la manera mas profesional posible

## AR-103 — Estrategia de pruebas y aceptacion

¿Que comportamientos deben demostrarse con pruebas y quien acepta que una capacidad esta correcta? Incluye dominio, integracion, ejecucion de agentes, seguridad, interfaz y reproduccion de partidas.

**Respuesta:**

todo es se optiene al provado por usuarios beta y testers reales que validen que todo esta bien

## AR-104 — Flujo de versiones y entregas

¿Como se organizaran ramas, revisiones, commits, versiones, entornos y despliegues? Define que acciones nunca deben automatizarse sin aprobacion explicita.

**Respuesta:**

segun se conveniente pero cada cambio en ramas para luego abrir una PR y con esto validar que es correcto para el merge

## AR-105 — Legibilidad y transferencia de conocimiento

¿Que deberia poder comprender una persona nueva durante su primer dia con el proyecto? Define la documentacion, recorridos guiados y estandares necesarios para que el conocimiento no dependa de una conversacion previa.

**Respuesta:**

todo eso, ya que al ser algo no tan convencional puede causar problemas el no entender todo aquello que quiero revisar y modificar

---

# 18. Revision final de coherencia

> Responde esta seccion al terminar las anteriores. Su proposito no es agregar alcance, sino descubrir contradicciones, vacios y decisiones sin fundamento.

## AR-106 — Sintesis verificable

Resume el sistema en un parrafo indicando quien obtiene que valor, mediante que experiencia y bajo que reglas esenciales. ¿Todas las secciones anteriores pueden reconocerse en esta sintesis?

**Respuesta:**

un sistema capaz de gestionar y alberar un concurso donde hay agentes que controlan a jugadores y estos reciben y actuan segun el entorno dado, buscando cumplir el objetivo trazado

## AR-107 — Trazabilidad del modelo

Revisa cada actor, caso de uso, entidad y pantalla propuesta: ¿que objetivo justifica su existencia y que respuesta de este cuestionario la respalda? Enumera elementos sin justificacion o respuestas sin representacion.

**Respuesta:**

si cubre una necesidad o proposito funcional esta justificado

## AR-108 — Conflictos y prioridades

¿Que respuestas compiten por tiempo, simplicidad, justicia, seguridad, espectaculo, flexibilidad o costo? Para cada conflicto, decide que principio domina y que consecuencia aceptas.

**Respuesta:**

tiempo seguridad y flexibilidad, lo demas puede esperar realmente 

## AR-109 — Responsabilidades y pasos ocultos

¿Existe alguna accion necesaria que no tenga responsable, permiso, entrada, salida o tratamiento de error? Incluye procesos manuales y dependencias externas.

**Respuesta:**

si no la tiene no se implementa y no deberia de existir

## AR-110 — Escenarios adversos

¿Que ocurre cuando usuarios se equivocan, actuan maliciosamente, llegan tarde, pierden conexion, presentan datos contradictorios o cuando la infraestructura falla en el peor momento?

**Respuesta:**

se toman las medidas correspondientes por parte de los encargados

## AR-111 — Preparacion para diseñar e implementar

¿Que preguntas continuan abiertas y cuales impiden crear el diagrama de clases conceptual, definir el MVP o elegir la arquitectura? Establece la condicion concreta para declarar cerrada la etapa de descubrimiento.

**Respuesta:**

tener un esqueleto simplificado de que si o si se tiene que tener, y que otras pueden esperar

---

# Espacio para notas transversales

Usa este espacio para observaciones que afecten varias preguntas. Referencia siempre los identificadores relacionados para que la nota pueda resolverse posteriormente.

**Notas:**

que todo salga bien y trabajar con cosas que si compilan y no simplemente hacer codigo

