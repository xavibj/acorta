# 003 — Fase 2: el almacén (`store/`)

**Fecha**: 2026-10-01 · **Spec**: `specs/003-almacen.md` · **Rutas**: `store/**`

## Encargo: paso rojo

> **Qué leer antes.** Trabajas en el repositorio `/home/claude/Projects/acorta`. Antes de escribir nada lee, en este orden: `AGENTS.md`, `specs/001-plan-tecnico.md` y `specs/003-almacen.md`. Mira también `link/link.go`: de ahí salen el tipo `link.Link` y los errores centinela que tu paquete devuelve. No necesitas las demás specs.
>
> **La tarea.** Eres el agente de la fase 2 de `specs/007-fases.md`: el paquete `store`, la persistencia en SQLite. Este encargo es solo el **paso rojo**:
>
> 1. Crea en `store/` el tipo `Store` y las funciones de la sección «API» de la spec 003, con las firmas exactas, pero **sin implementar**: lo justo para que los tests compilen.
> 2. Escribe los tests de los criterios ALM-01 a ALM-08.
> 3. Ejecuta `go test ./store/` y comprueba que los tests **fallan por las aserciones**, no por errores de compilación ni por un pánico.
>
> **El método.** Base de datos real en `t.TempDir()`, nunca en memoria ni simulada. Cada test nombra el criterio que comprueba (`ALM-03`). ALM-07 se prueba con 50 goroutines de verdad y debe pasar también con `go test -race`. ALM-02 debe cubrir un enlace con caducidad y otro sin ella.
>
> **Los límites.** Solo puedes crear o modificar ficheros dentro de `store/`. No toques `go.mod` ni `go.sum` (la dependencia `modernc.org/sqlite` ya está), no ejecutes `go mod tidy`, no añadas dependencias. No hagas commits. No implementes todavía. Si algún caso pasa con las funciones vacías, dilo en el informe.
>
> **Terminado es.** `go vet ./...` limpio, `gofmt -l .` vacío, y `go test ./store/` compilando y fallando.
>
> **El informe.** Ficheros creados; cuántos tests hay y cuántos fallan; una tabla criterio → test; cualquier ambigüedad o contradicción de la spec (no la resuelvas en silencio); y las últimas líneas de `go test ./store/`.

## Informe: paso rojo

> Paso rojo terminado. Los 8 tests fallan por aserciones, sin errores de compilación ni pánicos.
>
> **Ficheros creados (solo en `store/`)**
> - `store/store.go`: `Store{db *sql.DB}` y las firmas exactas de la spec 003. `Open` devuelve `&Store{}, nil`, `Close` y los métodos de escritura devuelven `nil`, `Get` el valor cero y `List` `nil, nil`. No hay implementación.
> - `store/store_test.go`: usa un fichero real en `t.TempDir()` y comprueba los errores con `errors.Is`. Importa `modernc.org/sqlite` solo en el test, para abrir una conexión independiente y leer el esquema y los valores crudos de ALM-01 y ALM-08. No toqué `go.mod` ni `go.sum`.
>
> **Resultado:** 8 funciones de test (10 con los subtests de ALM-02) y las 8 fallan. Con `-race` también fallan, sin avisos de carrera. Ningún caso pasa con las funciones vacías.
>
> `gofmt -l .` está vacío, `go vet ./store/` está limpio y `go vet ./...` también.
>
> | Criterio | Test | Qué comprueba |
> |---|---|---|
> | ALM-01 | `TestALM01_AbrirCreaFicheroYEsquema` | El fichero existe, hay tabla `links`, y reabrir la misma ruta no falla ni pierde datos |
> | ALM-02 | `TestALM02_PersistenciaTrasReabrir` | Dos subtests, sin caducidad y con caducidad. Cierra, reabre y compara todos los campos |
> | ALM-03 | `TestALM03_CodigoRepetido` | Devuelve `ErrCodeTaken` y el original no cambia |
> | ALM-04 | `TestALM04_GetNoExisteYMayusculas` | `ErrNotFound` para `zzz`, `abc`, `ABC`, `""` y `"Abc "`, y acierta con `"Abc"` |
> | ALM-05 | `TestALM05_ListaDelUltimoAlPrimero` | Lista vacía no `nil`; orden c,b,a con el mismo `created_at` |
> | ALM-06 | `TestALM06_Borrar` | Borra, `ErrNotFound` al borrar de nuevo, y se puede reinsertar el código |
> | ALM-07 | `TestALM07_VisitasConcurrentes` | 50 goroutines dan 50 visitas, y `ErrNotFound` si el código no existe |
> | ALM-08 | `TestALM08_InstantesEnUTCYSegundos` | Entrada en zona +02:00 con 123 ms. `Get` devuelve UTC sin fracción. En disco quedan `2026-06-01T10:30:45Z` y `2026-12-31T21:59:58Z` |
>
> **Ambigüedades de la spec (no las he resuelto por mi cuenta)**
> 1. **Visits en `Insert`:** la spec no dice si `Insert` guarda el `Visits` que trae el `Link` o lo fuerza a 0. ALM-02 pide "mismos valores en todos sus campos", pero mi test inserta siempre con `Visits: 0`.
> 2. **Fracciones de segundo (ALM-08):** no dice si se truncan o se redondean. Usé 123 ms, donde ambas opciones dan igual.
> 3. **Orden de `List` (ALM-05):** "del último insertado al primero" lo interpreto como orden por `id` descendente, no por `created_at`. Por eso el test usa el mismo `created_at` en los tres enlaces.
> 4. **Casos no cubiertos:** la spec no define `Insert`, `Get` o `AddVisit` tras `Close`, ni `Open` con una ruta inválida. No los he testeado.
> 5. **`go.mod`:** `modernc.org/sqlite` figura como `// indirect`. Funciona, pero cuando se importe en `store.go` el orquestador querrá moverla a directa. Yo no puedo tocar `go.mod`.

## Revisión del orquestador

Comprobado: solo hay ficheros nuevos en `store/`, `gofmt` y `go vet` limpios, los 8 tests fallan y los 8 criterios tienen su test.

De las ambigüedades, cuatro acaban en la spec 003 y una se descarta:

| # | Hueco | Qué se hace |
|---|---|---|
| 1 | ¿`Insert` respeta `Visits`? | Sí: guarda el enlace tal como llega (ALM-02) |
| 2 | ¿Truncar o redondear? | Truncar: `10:30:45.900` → `10:30:45` (ALM-08) |
| 3 | Orden de `List` | Por inserción (`id`), no por fecha (ALM-05) |
| 4a | `Open` con una ruta imposible | Criterio nuevo, **ALM-09**: la CLI lo necesita (CLI-13 promete un error claro si no se puede abrir la base de datos) y nadie lo garantizaba |
| 4b | Usar el almacén después de `Close` | Se queda sin especificar: nadie lo hace |
| 5 | `go.mod` marca SQLite como indirecta | Lo arregla el orquestador en el commit del verde: es el único que toca `go.mod` |

El 4a es el tipo de hueco que solo se ve al cruzar dos specs: la 005 prometía algo que la 003 no obligaba a nadie a cumplir.

## Encargo: ajuste del rojo

> La spec 003 ha cambiado para cerrar las ambigüedades que encontraste. Vuelve a leer `specs/003-almacen.md`: han cambiado ALM-02 (`Insert` guarda `Visits` tal como llega), ALM-05 (orden por inserción, no por fecha), ALM-08 (las fracciones se descartan, no se redondean) y hay un criterio nuevo, ALM-09 (`Open` con una ruta que no se puede crear devuelve error).
>
> Sigue siendo el **paso rojo**: ajusta solo `store/store_test.go` para cubrir lo que ahora la spec sí dice (un enlace insertado con visitas distintas de cero, una fracción de 900 ms, y ALM-09). Mismos límites: solo `store/`, sin implementar, sin commits. Si ALM-09 pasa con el esqueleto vacío, haz que el test falle por algo que el esqueleto no cumpla o dilo en el informe.
>
> Informe: qué tests has añadido o cambiado, el recuento nuevo y si queda alguna ambigüedad.

## Informe: ajuste del rojo

> Ajuste del rojo hecho. Solo he tocado `store/store_test.go`. No he implementado nada ni hecho commits.
>
> **Cambios**
> - **ALM-02:** he añadido el subtest "con visitas" (`Visits: 7`). El test ya comparaba `Visits`, así que ahora exige que `Insert` lo guarde tal cual.
> - **ALM-05:** los enlaces `a`, `b` y `c` se insertan en ese orden, pero con fechas que no siguen el orden de inserción: `a` tiene `created_at` +2h, y `b` y `c` comparten segundo. Ordenar por `created_at` daría un resultado distinto de `[c b a]`, así que el test solo pasa si el orden es el de inserción.
> - **ALM-08:** la fracción pasa de 123 ms a 900 ms, en `created` y en `expires`. Redondear daría `:46` y `:59`, y el test espera `:45` y `:58`, tanto en el valor devuelto como en el texto crudo del disco.
> - **ALM-09, test nuevo `TestALM09_RutaImposible`:** abre una ruta dentro de un directorio inexistente. Exige que `Open` devuelva error y un `*Store` nil, y que no se haya creado el directorio. Con el esqueleto falla porque `Open` devuelve `nil` y un `Store` no nil.
>
> **Recuento:** hay 9 funciones de test (ALM-01 a ALM-09) y las 9 fallan, por aserciones y sin pánicos. Contando subtests fallan 12 con `-race`, sin avisos de carrera. `gofmt -l .` está vacío y `go vet ./...` está limpio en todo el repo.
>
> **Ambigüedades pendientes**
> - La spec no dice qué pasa con `Insert`, `Get` o `AddVisit` después de `Close`. Tampoco lo he testeado.
> - ALM-09 dice "no deja nada abierto". Lo compruebo solo mediante `Store == nil`, porque no hay una forma observable de ver conexiones abiertas.

**Commits**: `Spec 003: aclaraciones del paso rojo y criterio ALM-09` y `store: tests de la spec 003 (rojo)`. 9 tests, 9 en rojo.

## Encargo: paso verde

> Los tests del paso rojo ya están en un commit. Ahora, el **paso verde**: implementa el paquete `store` hasta que pase `go test ./store/`, también con `-race`.
>
> **El método.** Lo mínimo que haga pasar los tests y cumpla la spec 003: el esquema y los `PRAGMA` exactos de la spec, `database/sql` con el controlador `modernc.org/sqlite` (nombre del controlador: `sqlite`). **No modifiques `store/store_test.go`**: si crees que un test está mal o contradice la spec, párate y dilo en el informe. Para detectar el código repetido usa el error de restricción `UNIQUE` de SQLite, no un `SELECT` previo (entre el `SELECT` y el `INSERT` puede colarse otro). Cuando esté en verde, limpia el código sin cambiar el comportamiento. Comentarios en español que expliquen el porqué.
>
> **Los límites.** Solo `store/`. No toques `go.mod` ni `go.sum` ni ejecutes `go mod tidy`: si la compilación te pide algo de ahí, dilo en el informe y lo arregla el orquestador. Sin commits.
>
> **Terminado es.** `gofmt -l .` vacío, `go vet ./...` limpio, `go test ./...` y `go test -race ./store/` en verde.
>
> **El informe.** Ficheros tocados; decisiones que has tomado y por qué (en especial: cómo reconoces el error `UNIQUE` y cómo evitas `database is locked` con escrituras simultáneas); desviaciones de la spec; si has tocado algún test (no deberías); y la salida resumida de `go test ./...`.
