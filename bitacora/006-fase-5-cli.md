# 006 — Fase 5: la CLI (`cmd/acorta/`)

**Fecha**: 2026-10-01 · **Spec**: `specs/005-cli.md` · **Rutas**: `cmd/acorta/**`

## Encargo: paso rojo

> **Qué leer antes.** Trabajas en el repositorio `/home/claude/Projects/acorta`. Antes de escribir nada lee, en este orden: `AGENTS.md`, `specs/001-plan-tecnico.md` y `specs/005-cli.md` entera. Mira el código que vas a usar: `link/link.go` (errores y `Input`), `store/store.go` (`Open`), `shortener/shortener.go` (`New` y los casos de uso) y, de `server/`, solo las firmas de `Handler` y `Static`.
>
> **La tarea.** Eres el agente de la fase 5 de `specs/007-fases.md`: el ejecutable `acorta`. Este encargo es solo el **paso rojo**:
>
> 1. Crea `cmd/acorta/main.go` con `main` y con `run(args []string, stdout, stderr io.Writer) int` **sin implementar** (devuelve siempre un código que ningún criterio espera, por ejemplo 99, y no escribe nada). Si para probar sin procesos ni sockets necesitas inyectar el entorno, el reloj, el generador de códigos o la función que escucha, decide ahora cómo (variables del paquete o una estructura con las dependencias) y deja el hueco preparado: `run` conserva su firma.
> 2. Escribe los tests de CLI-01 a CLI-13 contra `run`.
> 3. Ejecuta `go test ./cmd/acorta/` y comprueba que los tests **fallan por las aserciones**, no por errores de compilación ni por un pánico.
>
> **El método.** Nada de lanzar el binario ni de abrir sockets: todo contra `run`, con `bytes.Buffer` para `stdout` y `stderr` y la base de datos en `t.TempDir()` (pasada con `-db`). Comprueba siempre las tres cosas: código de salida, `stdout` exacto y `stderr`. Para CLI-01 inyecta un generador de códigos determinista; para CLI-03, un reloj fijo, y comprueba la caducidad leyendo el enlace con `list` o directamente del almacén. Para CLI-09 inyecta la función que escucha y comprueba la dirección, el mensaje de arranque y el caso de error. Para CLI-10 usa `t.Setenv`. Cada test nombra el criterio que comprueba.
>
> **Los límites.** Solo puedes crear o modificar ficheros dentro de `cmd/acorta/`. No toques los demás paquetes, `go.mod` ni `go.sum`; no ejecutes `go mod tidy`; no añadas dependencias (solo `flag` y la biblioteca estándar). No hagas commits. No implementes todavía.
>
> **Terminado es.** `go vet ./...` limpio, `gofmt -l .` vacío, y `go test ./cmd/acorta/` compilando y fallando; el resto del repo sigue en verde.
>
> **El informe.** Ficheros creados; cuántos tests hay y cuántos fallan; una tabla criterio → test; cómo has resuelto la inyección de dependencias; cualquier ambigüedad o contradicción de la spec, o cualquier cosa de los otros paquetes que no encaje (no la resuelvas en silencio); y las últimas líneas de `go test ./cmd/acorta/`.

## Informe: paso rojo

> Paso rojo de la fase 5 hecho. `cmd/acorta/` compila y los 22 tests fallan.
>
> **Ficheros creados** (solo en `cmd/acorta/`):
> - `main.go`: `main`, `var version = "dev"`, la estructura `environment`, la variable `env` y `run` sin implementar (devuelve 99 y no escribe nada).
> - `main_test.go`: los tests.
>
> **Estado**
> - `gofmt -l .` está vacío y `go vet ./cmd/acorta/` y `go vet ./...` salen limpios.
> - Hay 22 tests y fallan los 22. Fallan por las aserciones (código 99 en lugar del esperado, stdout y stderr vacíos), sin errores de compilación ni pánicos.
> - Una excepción: `TestCLI06_ListTable` falla al preparar el escenario, no en una aserción. Usa `AddVisit` sobre un enlace que `add` todavía no ha creado. Con `add` implementado ese fallo desaparece.
>
> | Criterio | Test |
> |---|---|
> | CLI-01 | `TestCLI01_AddPrintsShortURL` |
> | CLI-02 | `TestCLI02_AddWithAlias` |
> | CLI-03 | `TestCLI03_AddTTL` (72h leído del almacén y 90m visto en `list`), `TestCLI03_InvalidTTL` (`abc`, `0`, `0s`, `-1h`, `72` y vacío: salida 2, mensaje exacto y nada guardado) |
> | CLI-04 | `TestCLI04_ValidationErrors` (varias líneas, stdout vacío, salida 1), `TestCLI04_AliasTaken` |
> | CLI-05 | `TestCLI05_AddArgumentCount` (sin URL y con dos URL) |
> | CLI-06 | `TestCLI06_ListTable` (cabecera, orden, visitas, `nunca`, `caducado` y fecha RFC 3339) |
> | CLI-07 | `TestCLI07_ListEmpty` |
> | CLI-08 | `TestCLI08_Remove`, `TestCLI08_RemoveArgumentCount` (el uso de `rm` no está en un criterio; es una extensión mía) |
> | CLI-09 | `TestCLI09_Serve`, `TestCLI09_ServeDefaultAddr`, `TestCLI09_ServeListenError` |
> | CLI-10 | `TestCLI10_DBPrecedence`, `TestCLI10_DBDefault`, `TestCLI10_BaseURLPrecedence` (la barra final se quita) |
> | CLI-11 | `TestCLI11_Version` |
> | CLI-12 | `TestCLI12_Help` (sin argumentos, `help` y `-h`), `TestCLI12_UnknownSubcommand` |
> | CLI-13 | `TestCLI13_CannotOpenDB` (`add`, `list`, `rm` y `serve`) |
> | Códigos de salida de la spec, no un criterio numerado | `TestCLI_UnknownFlagIsUsageError` (salida 2) |
>
> **Inyección de dependencias.** Es una variable del paquete `env` de tipo `environment`, con cuatro campos: `getenv func(string) string`, `now func() time.Time`, `newCode func() (string, error)` (con `nil` para el generador real) y `listen func(addr string, h http.Handler) error`. El valor real es `os.Getenv`, `time.Now`, `nil` y `http.ListenAndServe`. `run` conserva su firma y debe leer siempre `env`. Los tests no son paralelos, porque comparten `env`.
>
> **Ambigüedades de la spec y cosas que no encajan** (no las he resuelto en silencio; los tests fijan lo que he elegido):
> 1. **Formato de la tabla de `list`.** La spec da las columnas pero no el relleno. Los tests comparan por campos (`strings.Fields`), así que vale `tabwriter` o tabuladores.
> 2. **Texto de uso (CLI-05).** La spec no lo fija. Los tests solo exigen salida 2, stdout vacío y que stderr contenga `acorta add` (o `acorta rm`).
> 3. **Mensaje de arranque con una dirección con host.** Con `-addr 127.0.0.1:9000` la spec no dice qué imprimir. Solo he probado `:9090` y `:8080`.
> 4. **Formato de errores multilínea y orden de los mensajes.** CLI-04 dice «una línea `error: …` por cada mensaje», y `ValidationError.Messages` ya los trae en orden.
> 5. **Variable de entorno vacía.** Los tests asumen que `ACORTA_DB=""` y `ACORTA_BASE_URL=""` equivalen a «no definida». La spec no lo dice.
> 6. **CLI-13 en `serve`.** El test exige que `serve` no llame a `listen` si no abre la base de datos.
> 7. **`-ttl ""` y `-ttl 72`.** Los he incluido como inválidos, con el mismo mensaje y salida 2.
> 8. **`-h` dentro de un subcomando** (`acorta add -h`). La spec no lo define y no lo he probado.
> 9. **`TestCLI09_Serve`** hace además `GET /api/links` contra el handler recibido y espera 200. Eso depende de que el otro agente termine `server.Handler`.

## Revisión del orquestador

Comprobado: solo hay ficheros nuevos en `cmd/acorta/`; `gofmt` y `go vet` limpios; los 22 tests fallan; los 13 criterios tienen su test.

Casi todo lo que el agente eligió era razonable, así que la revisión consiste sobre todo en **escribir en la spec lo que el test ya da por hecho**: si no, el test estaría comprobando algo que nadie ha pedido. Un test que va por delante de la spec es tan sospechoso como un código que va por delante del test.

| # | Hueco | Cómo queda la spec 005 |
|---|---|---|
| 1 | Relleno de la tabla | Alineada con espacios; el número exacto no es contrato (CLI-06) |
| 2 | Texto del mensaje de uso | `uso: ` + la sinopsis del subcomando |
| 3 | Mensaje de arranque con host | `http://127.0.0.1:9000`; sin host, `localhost` (CLI-09) |
| 5 | Variable de entorno vacía | Cuenta como no definida |
| 6 | `serve` sin base de datos | No llega a escuchar (CLI-13) |
| 7 | `-ttl` vacío o sin unidad | Inválido (CLI-03) |
| 8 | `acorta add -h` | Uso y flags en `stdout`, salida 0 (CLI-12) |
| — | `rm` sin código, y flags desconocidos | Uso y salida 2 (CLI-08 y «Códigos de salida») |

## Encargo: ajuste del rojo

> La spec 005 ha cambiado para cerrar las ambigüedades que encontraste. Vuelve a leer `specs/005-cli.md` entera: han cambiado «Configuración» (variable de entorno vacía), «Códigos de salida» (el mensaje de uso tiene formato: `uso: ` + la sinopsis; flag desconocido), CLI-03 (`-ttl` vacío o sin unidad), CLI-06 (tabla alineada), CLI-08 (`rm` sin código o con varios), CLI-09 (mensaje de arranque con host), CLI-12 (`-h` dentro de un subcomando) y CLI-13 (`serve` no llega a escuchar).
>
> Sigue siendo el **paso rojo**: ajusta solo `cmd/acorta/main_test.go` para cubrir lo que ahora la spec sí dice (el formato exacto del mensaje de uso, el mensaje de arranque con `-addr 127.0.0.1:9000`, y `acorta add -h`). Mismos límites: solo `cmd/acorta/`, sin implementar, sin commits.
>
> Informe: qué tests has añadido o cambiado, el recuento nuevo y si queda alguna ambigüedad.

## Informe: ajuste del rojo

> Ajuste del rojo hecho. Solo he tocado `cmd/acorta/main_test.go`; `main.go` y la bitácora siguen como estaban.
>
> **Estado:** `gofmt -l .` está vacío y `go vet ./cmd/acorta/` sale limpio. Hay 24 tests y fallan los 24, sin pánicos (antes eran 22).
>
> **Tests nuevos**
> - `TestCLI09_ServeHostInAddr`: con `-addr 127.0.0.1:9000` espera `acorta escuchando en http://127.0.0.1:9000\n` y que `listen` reciba esa dirección.
> - `TestCLI12_SubcommandHelp`: `add -h`, `list -h`, `rm -h` y `serve -h` salen con 0, con stderr vacío. Stdout empieza por la línea de uso y lista los flags del subcomando.
>
> **Tests cambiados**
> - Se añaden constantes con el uso exacto de cada subcomando.
> - `TestCLI05_AddArgumentCount`: ahora exige salida 2, stdout vacío y stderr exactamente `uso: acorta add [-alias ALIAS] [-ttl DURACIÓN] [-db RUTA] [-base-url URL] URL\n`.
> - `TestCLI08_RemoveArgumentCount`: exige stderr exactamente `uso: acorta rm [-db RUTA] CÓDIGO\n`.
> - `TestCLI_UnknownFlagIsUsageError`: ahora prueba `list`, `add`, `rm` y `serve`. Exige salida 2, stdout vacío y que stderr contenga la línea de uso.
> - `TestCLI06_ListTable`: además comprueba que la tabla no lleve tabuladores y que la columna `URL` empiece en la misma posición en todas las filas.
>
> **Ambigüedades que quedan** (ninguna bloqueante): el contenido exacto de la ayuda tras la línea de uso en `add -h`; el texto que añade `flag` para un flag desconocido; `version` y `help` no tienen sinopsis propia.

**Commits**: `Spec 005: lo que los tests de la CLI daban por hecho` y `cmd/acorta: tests de la CLI (rojo)`. 24 tests, 24 en rojo.

## Encargo: paso verde

> Los tests del paso rojo ya están en un commit, y el paquete `server` ya está implementado y en verde. Ahora, el **paso verde**: implementa la CLI hasta que pase `go test ./cmd/acorta/`.
>
> **El método.** Lo mínimo que haga pasar los tests y cumpla la spec 005. **No modifiques `cmd/acorta/main_test.go`**: si crees que un test está mal o contradice la spec, párate y dilo en el informe. La CLI es un adaptador fino: paquete `flag`, un `flag.FlagSet` por subcomando, y todo lo demás delegado en `store`, `shortener` y `server`. No repitas reglas ni mensajes de validación: los errores de `link` se imprimen tal cual, con el prefijo `error: `. `serve` pasa `server.Static()` a `server.Handler`. Cierra siempre el almacén. Cuando esté en verde, limpia el código sin cambiar el comportamiento. Comentarios en español que expliquen el porqué.
>
> **Los límites.** Solo `cmd/acorta/`. No toques los demás paquetes, `go.mod` ni `go.sum`. Solo la biblioteca estándar y los paquetes del repo. Sin commits.
>
> **Terminado es.** `gofmt -l .` vacío, `go vet ./...` limpio y `go test ./...` en verde en todo el repo.
>
> **El informe.** Ficheros tocados; decisiones que has tomado y por qué; desviaciones de **cualquier** spec, también del plan técnico (`specs/001`), aunque te parezcan justificadas; si has tocado algún test (no deberías); y la salida resumida de `go test ./...`.

## Informe: paso verde

> Paso verde casi hecho: 22 de los 24 tests de `cmd/acorta` pasan y los 2 que fallan tienen un fallo del propio test. Me paro en esos dos sin tocar `main_test.go`, como pide el encargo.
>
> **Ficheros tocados:** solo `cmd/acorta/main.go`. No he tocado ningún test, ni la bitácora, ni otros paquetes, ni `go.mod`/`go.sum`. No hay commits. `gofmt -l .` está vacío y `go vet ./...` sale limpio.
>
> 1. **`TestCLI06_ListTable` (CLI-06): el test mide en bytes lo que la tabla alinea en caracteres.**
>    - `strings.Index(lines[0], "URL")` devuelve un desplazamiento en bytes. La cabecera `CÓDIGO` tiene una `Ó` de 2 bytes, así que el desplazamiento sale 40 en la cabecera y 39 en las filas.
>    - `text/tabwriter`, que es lo que pide la spec, alinea por caracteres. Visualmente la tabla está bien alineada.
>    - Para pasar el test habría que desalinear la tabla a ojo. Propuesta de arreglo del test: comparar con `utf8.RuneCountInString(lines[0][:idx])`.
> 2. **`TestCLI09_Serve` (CLI-09): el test usa el handler después de que `run` haya cerrado la base de datos.**
>    - Tras `run`, el test hace `GET /api/links` contra el handler que guardó y obtiene 500, porque el almacén ya está cerrado.
>    - En el uso real `listen` bloquea y el almacén sigue abierto mientras se sirve. Solo se cierra al volver `listen`.
>    - Si no cerrara el almacén, rompería lo que pide el encargo, «Cierra siempre el almacén». Propuesta: hacer la petición dentro del stub de `listen`, que es cuando el almacén sigue abierto.
>
> **Decisiones**
> - Cada subcomando tiene su `flag.FlagSet` con `ContinueOnError` y la salida de `flag` silenciada. Los mensajes los escribo yo: `-h` imprime `uso: …` y los flags en stdout y sale con 0; un flag desconocido imprime `error: …` y `uso: …` en stderr y sale con 2; un número de argumentos incorrecto imprime solo `uso: …` y sale con 2.
> - Para `-ttl` miro con `fs.Visit` si se pasó el flag, porque `-ttl ""` explícito también es error (salida 2). La caducidad se calcula con `svc.Now()` y se pasa como `ExpiresAt` en RFC 3339, así que la validación de «fecha futura» sigue siendo de `link`.
> - Errores de validación: una línea `error: …` por cada mensaje de `ValidationError.Messages`, sin reescribirlos.
> - Cuando el error es `link.ErrNotFound` imprimo solo el mensaje del centinela (`enlace no encontrado`). El almacén lo envuelve con la operación (`borrar "promo": …`) y CLI-08 pide el mensaje corto.
> - `list` usa `text/tabwriter` con relleno de 2 espacios. Lee el reloj una sola vez para marcar `caducado`.
> - El mensaje de arranque de `serve` usa `net.SplitHostPort`. Sin host dice `localhost`.
> - `serve` abre la base de datos antes de escuchar y pasa `server.Static()` a `server.Handler`.
>
> **Desviaciones de la spec o del plan técnico:**
> - `serve` no maneja señales ni hace apagado ordenado. Con el `listen` inyectado (`http.ListenAndServe` en producción) un Ctrl-C mata el proceso sin cerrar el almacén. No está en la spec y no lo he añadido. Convendría decidirlo.
> - `list` acepta `-base-url` porque la sinopsis lo incluye, pero no lo usa en la salida.
> - Con `-h` en un subcomando, `PrintDefaults` imprime los flags con el formato propio de `flag`. La spec no fija ese formato.

## Verificación del orquestador

Otra vez la regla 3, y esta vez por partida doble. Los dos tests estaban mal de formas distintas y las dos son instructivas:

- **CLI-06** comparaba posiciones en bytes. En inglés ese test habría pasado siempre. Falló porque la cabecera dice `CÓDIGO`: la `Ó` ocupa dos bytes y una columna. La tabla estaba bien; el test, no. La salida fácil para el agente era quitar el acento o rellenar a mano un espacio de más: código peor para contentar a un test equivocado.
- **CLI-09** comprobaba el servidor **después** de que `serve` hubiera terminado y cerrado la base de datos. Para pasarlo había que dejar la base de datos abierta para siempre, es decir, introducir una fuga de recursos.

El orquestador leyó los dos tests, confirmó los dos diagnósticos y los corrigió él, en un commit propio y anterior al de la implementación.

Las «desviaciones» del informe también tenían premio. Esta vez el encargo pedía listar desviaciones de **cualquier** spec (lección de la fase 4), y salieron tres:

| Lo que dijo el agente | Qué se hace |
|---|---|
| `list` acepta `-base-url` pero no lo usa | Era un error **de la spec**: la sinopsis de `list` lo incluía y su tabla no muestra URL cortas. Se quita de la spec 005 y de los tests, y el agente lo quita del código |
| `serve` no cierra el almacén con Ctrl-C | Se queda así en la v1. SQLite en modo WAL soporta que el proceso muera sin cerrar; un apagado ordenado es una mejora, no un fallo |
| El detalle de un flag mal usado lo escribe la biblioteca, en inglés | Se acepta y se dice en la spec, en vez de traducir mensajes ajenos comparando cadenas |

**Commits**: `cmd/acorta: corrige dos tests que ninguna implementación correcta podía pasar` y `Spec 005: list no lleva -base-url, y el error de flags`.

## Encargo: cierre del verde

> Tenías razón en los dos tests: los ha corregido el orquestador (CLI-06 cuenta caracteres; CLI-09 prueba el handler dentro de `listen`) y ya están en un commit. Y también en que `-base-url` no pintaba nada en `list`: era un error de la spec. Vuelve a leer `specs/005-cli.md`: `list` ya no lleva `-base-url` (sinopsis y tabla de «Configuración»).
>
> Quita `-base-url` de `list` en `cmd/acorta/main.go` y deja `go test ./...` en verde en todo el repo. No toques `main_test.go`. Mismos límites: solo `cmd/acorta/`, sin commits.
>
> Informe: qué has cambiado y la salida resumida de `go test ./...`.

## Informe: cierre del verde

> Hecho: `list` ya no tiene `-base-url` y `go test ./...` pasa en todo el repo.
>
> **Cambio en `cmd/acorta/main.go`:** la sinopsis `synopsisList` ahora es `acorta list [-db RUTA]`, igual que la spec. La ayuda general la hereda. He quitado de `cmdList` el flag `-base-url`. No he tocado `main_test.go`, ni nada fuera de `cmd/acorta/`, ni la bitácora. No hay commits.
>
> **Comprobaciones:** `gofmt -l .` sale vacío y `go vet ./...` sale limpio. `go test ./...`: `cmd/acorta`, `link`, `server`, `shortener` y `store` en verde.

`make check` completo, ejecutado por el orquestador: formato, `go vet`, los cinco paquetes Go en verde, los 38 tests de la interfaz en verde y el build del frontend sin errores.

**Commit**: `cmd/acorta: la CLI (verde)`.
