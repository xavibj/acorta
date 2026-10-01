# 001 — De la idea a la spec: la entrevista

**Fecha**: 2026-10-01 · **Quién**: Xavi (decide) y Claude Code (pregunta y redacta) · **Resultado**: `specs/000` a `007`, `AGENTS.md`

Así empezó acorta. No hay código todavía: solo una idea de una línea, cuatro preguntas y la lista de todo lo que la idea no decía.

## 1. La idea

El encargo completo, tal como se escribió:

> proyecto conductor: acortador de urls en go y frontend en vuejs (ver doc)

«El doc» es la guía de arquitectura que Xavi usa en sus herramientas: un binario Go único con el frontend Vue embebido, API REST y CLI. Eso fija el *cómo* a grandes rasgos. Del *qué* solo dice «acortador de URLs».

## 2. Las cuatro preguntas

Antes de escribir una línea de spec, el agente preguntó lo que no podía decidir él: lo que cambia el producto. Cada pregunta venía con opciones y con una recomendación razonada.

| # | Pregunta | Opciones | Respuesta de Xavi |
|---|---|---|---|
| 1 | ¿Cómo se llama y dónde vive el código? | **acorta, repo propio** (recomendada: historial limpio para repetirlo commit a commit) · acorta dentro del repo de la web | acorta, repo propio |
| 2 | Además del núcleo (crear, redirigir, listar), ¿qué entra en la v1? | Contador de visitas · Borrar enlaces · Alias personalizado · Caducidad (se podían marcar varias) | **Las cuatro** |
| 3 | ¿Dónde se guardan los enlaces? | **SQLite sin CGO** (recomendada) · Fichero JSON · Solo en memoria | SQLite sin CGO |
| 4 | ¿Quién puede crear y borrar enlaces en la v1? | **Sin autenticación** (recomendada: llegará después como spec nueva) · Token de API desde la v1 · Usuario y contraseña | Sin autenticación |

Consecuencia de la respuesta 2: como las cuatro funcionalidades entran en la v1, el cambio de requisitos que el tutorial necesita más adelante será la **autenticación**, que obliga a tocar specs que ya existen.

## 3. Lo que la idea no decía

Con las cuatro respuestas ya se puede escribir la spec, pero al escribirla aparecen decenas de decisiones pequeñas que nadie ha tomado. El agente las resolvió con el criterio más simple o más seguro y **las dejó a la vista** para que Xavi las revise. Esta lista es la diferencia entre «hazme un acortador» y un contrato:

| # | La pregunta que nadie había hecho | Lo que dice la spec | Dónde |
|---|---|---|---|
| 1 | ¿Cómo son los códigos generados? | 6 caracteres, minúsculas y dígitos, aleatorios (no consecutivos: no se pueden adivinar los de otros) | COD-01, COD-04 |
| 2 | ¿Y si el código generado ya existe? | Se genera otro, hasta 5 intentos | COD-02 |
| 3 | ¿Acortar dos veces la misma URL da el mismo enlace? | No: dos enlaces distintos | COD-03 |
| 4 | ¿Qué URL se aceptan? | Solo `http` y `https`, con dominio, hasta 2048 bytes. No se adivina el esquema de `example.com` | URL-04 a URL-08 |
| 5 | ¿Se «arregla» la URL (minúsculas, barra final)? | No. Se guarda tal cual, solo se recortan espacios | URL-01, URL-02 |
| 6 | ¿Qué alias valen? | De 3 a 32, minúsculas, dígitos y guiones, sin guion en los extremos | ALI-02 a ALI-04 |
| 7 | ¿`Oferta` se convierte en `oferta`? | No: es un error. Nada se transforma en silencio | ALI-03 |
| 8 | ¿Puede un alias llamarse `api`? | No: `api` y `assets` están reservados para el propio servidor | ALI-05 |
| 9 | ¿301 o 302? | 302. Un 301 se queda en la caché del navegador y las visitas siguientes no se contarían | RED-01 |
| 10 | ¿Qué es exactamente una visita? | Una redirección servida por `GET`. `HEAD` no cuenta. No se filtran bots | VIS-02, VIS-04 |
| 11 | ¿Qué ve quien abre un enlace caducado? | Un `410 Gone`, distinto del `404` de uno que no existe | RED-03 |
| 12 | ¿Qué pasa con un enlace caducado? | Sigue en la lista, marcado, y su código sigue ocupado hasta que se borra | CAD-06 |
| 13 | ¿La caducidad es una fecha o un plazo? | Una fecha en la API (`expires_at`); un plazo en la CLI (`-ttl 72h`) y en la web (desplegable) | CAD-02, CLI-03, WEB-01 |
| 14 | ¿Se puede reutilizar el código de un enlace borrado? | Sí | BOR-03 |
| 15 | Si hay varios errores, ¿se dicen todos? | Sí, todos a la vez, uno por campo, y con el mismo texto en la API, la CLI y la web | spec 002, API-04 |
| 16 | ¿En qué idioma están los errores? | En español, y salen de un único paquete | spec 001 |
| 17 | ¿La CLI habla con el servidor o con la base de datos? | Con la base de datos: funciona con el servidor parado | spec 005 |
| 18 | ¿Hay paginación? ¿Se puede editar un enlace? | No, en la v1 | spec 000 |
| 19 | ¿Comentarios en español o en inglés? | Identificadores en inglés, comentarios en español: el código lo leerán quienes siguen el tutorial | spec 001 |
| 20 | ¿El frontend compilado va en Git? | ~~Sí: así `go build` funciona sin Node~~ **No** (cambiado en la revisión, ver más abajo) | spec 001 |
| 21 | ¿Se prueba el frontend? | Sí: tests de componentes con Vitest, además del build y de una pasada en navegador real | spec 006 |

## 4. Revisión

Xavi revisa las specs y la lista anterior. Los cambios que pide se aplican a la spec antes de escribir el primer test.

### Cambio 1: el frontend compilado sale de Git

La decisión 20 la había tomado el agente copiando la costumbre de otro proyecto de Xavi. En la revisión, Xavi preguntó:

> Frontend compilado en Git, es lo mejor?

Al tener que defenderla, el agente contestó que no. Tiene una ventaja (`go build` funciona sin Node) y tres inconvenientes que aquí pesan más:

1. Cada commit que toque `web/` arrastraría un bundle minificado con nombres que cambian en cada build: un diff que nadie puede leer, en un repo cuyo historial es material de un tutorial sobre revisar diffs.
2. Nada garantiza que lo compilado corresponda al código fuente: basta olvidar un build, o que un agente edite el resultado a mano.
3. Lo que hay en el repo debería poder revisarse y salir de la spec. Un artefacto generado no cumple ninguna de las dos cosas.

Xavi: «si cambialo». Lo que cambió en el contrato, antes de que existiera una sola línea de código:

| Spec | Cambio |
|---|---|
| 001 | `server/static/dist/` va en `.gitignore`; `make build` compila siempre el frontend; un clon sin compilar sigue construyendo y pasando los tests |
| 004 | El servidor recibe el sistema de ficheros de los estáticos en vez de usar siempre el embebido. Criterios EST reescritos y dos nuevos: EST-04 (página de aviso si el frontend no está compilado) y EST-05 (el aviso sí está en Git) |
| 006 | El build de Vite escribe en `server/static/dist/` |
| 007 | La fase 0 ya no crea un `index.html` provisional; la fase 4 es dueña de todo `server/`; la fase 6, solo de `web/` |

Coste del cambio: unos minutos y cuatro ficheros de texto. El mismo cambio con el servidor y el frontend ya escritos habría tocado código, tests, el Makefile y el historial.
