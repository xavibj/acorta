# 010 — Fase 8: autenticación, el primer cambio de requisitos

**Fecha**: 2026-10-03 · **Spec**: `specs/008-autenticacion.md` (nueva) y retoques en 000, 004, 005, 006 y 007 · **Rutas**: `server/**` (8a), `cmd/acorta/**` (8b), `web/**` (8c)

La v1 dejó la autenticación fuera a propósito (spec 000, bitácora 001, pregunta 4). Es el cambio de requisitos del capítulo 09 del tutorial, y se hace igual que todo lo demás: primero la spec, después los tests, después el código.

## Entrevista

El encargo de Xavi fue una palabra: «Autenticación». Antes de escribir nada, cuatro preguntas con opciones y recomendación (regla 10 de `AGENTS.md`):

| # | Pregunta | Opciones | Respuesta |
|---|---|---|---|
| 1 | ¿Qué operaciones pasan a exigir autenticación? | **Crear y borrar** (recomendada: es lo que cambia quién puede escribir) · Crear, borrar y listar · Toda la API | Crear y borrar |
| 2 | ¿Qué forma tiene la credencial? | **Un token de API** (recomendada: un secreto único al arrancar, sin usuarios ni contraseñas) · Usuario y contraseña · Varios tokens con nombre | Un token de API |
| 3 | ¿Qué pasa si el servidor arranca sin token? | **Se niega a arrancar** (recomendada: nadie expone sin querer un acortador abierto) · Sigue abierto, con aviso | Se niega a arrancar |
| 4 | ¿Cómo se autentica la interfaz web? | **Pide el token una vez** y lo guarda en el navegador (recomendada) · Campo de token en cada formulario | Pide el token una vez |

Xavi aceptó las cuatro recomendaciones.

## Decisiones tomadas por defecto (a revisar)

Lo que la spec 008 decide sin que nadie lo haya preguntado:

| # | La pregunta que nadie hizo | Lo que dice la spec | Dónde |
|---|---|---|---|
| 1 | ¿Cómo viaja el token? | Cabecera `Authorization: Bearer <token>`; el esquema no distingue mayúsculas, el token sí | 008, «El token», AUT-03 |
| 2 | ¿Hay un tamaño mínimo? | 16 caracteres, después de recortar espacios; `serve` se niega con menos | 008, AUT-09 |
| 3 | ¿Qué responde una petición sin token o con uno malo? | `401` con `{"errors":["falta el token de API"]}` o `{"errors":["token de API incorrecto"]}` y `WWW-Authenticate: Bearer` | AUT-01 a AUT-03 |
| 4 | ¿Se comprueba el token antes o después de validar el cuerpo? | Antes: un `POST` sin token con cuerpo inválido da `401`, no `400` | AUT-05 |
| 5 | ¿Y antes o después del método? | Después: `PUT /api/links` sin token sigue dando `405` (API-12) | 008, «API» |
| 6 | ¿Se compara en tiempo constante? | Sí, `crypto/subtle` | 008, «El token» |
| 7 | ¿Sale el token en el log o en la ayuda? | Nunca; el mensaje de arranque no cambia | AUT-10 |
| 8 | ¿Cambian `add`, `list` y `rm`? | No: no pasan por la API | 008, «CLI» |
| 9 | ¿Cómo entra el token en `Handler`? | Cuarto parámetro; vacío no significa «abierto» | 008, «Contrato» |
| 10 | ¿Dónde guarda el token el navegador? | `sessionStorage`, clave `acorta_token`; se pierde al cerrar la pestaña | AUT-11 |
| 11 | ¿Qué hace la web con un `401`? | Muestra el error tal como llega y da el foco al campo del token; no borra el token guardado | AUT-13 |
| 12 | ¿Envía el token al listar? | No: listar es público y la cabecera no se manda donde no hace falta | AUT-12 |
| 13 | ¿Una fase o varias? | Tres: 8a `server`, 8b `cmd/acorta`, 8c `web`; 8b y 8c en paralelo tras 8a, porque cambia la firma de `Handler` | 007 |

**Commit**: `Spec 008: autenticación con token de API`.

## Encargo: paso rojo (8a, `server`)

> **Qué leer antes.** Trabajas en el repositorio `/home/claude/Projects/acorta`. Antes de escribir nada lee, en este orden: `AGENTS.md`, `specs/001-plan-tecnico.md`, `specs/008-autenticacion.md` entera y, de `specs/004-api.md`, las secciones «API» y «Contrato del paquete». Mira el código que vas a tocar: `server/server.go`, `server/api.go`, `server/respond.go` y los tests existentes en `server/*_test.go` (en especial `server/helpers_test.go`, que es donde se construye el `Handler` para los tests).
>
> **La tarea.** Eres el agente de la fase 8a de `specs/007-fases.md`: la autenticación en el paquete `server`. Este encargo es solo el **paso rojo**:
>
> 1. Cambia la firma de `Handler` a la del contrato de la spec 008: `Handler(svc Service, baseURL string, static fs.FS, token string) http.Handler`. Por ahora **sin implementar** la comprobación: el parámetro se acepta y se ignora. Ajusta las llamadas a `Handler` que haya en los tests existentes para que compilen (pásales un token de prueba de al menos 16 caracteres), sin cambiar nada más de esos tests.
> 2. Escribe los tests de AUT-01 a AUT-06 en un fichero nuevo, `server/auth_test.go`. Cada test o caso lleva en su nombre el identificador del criterio, con guion (`AUT-03`), como en los subtests de `api_test.go`.
> 3. Ejecuta `go test ./server/` y comprueba que los tests nuevos **fallan por las aserciones**, no por errores de compilación ni por pánicos, y que los tests anteriores siguen en verde.
>
> **El método.** `httptest` contra `Handler`, con el servicio real sobre `store.Open` en `t.TempDir()`, como hacen los tests existentes (reutiliza los helpers de `helpers_test.go`). Comprueba en cada caso estado, cabecera `WWW-Authenticate` cuando la spec la exige, `Content-Type` y cuerpo exacto. AUT-05 necesita dos casos: `POST` sin token con cuerpo inválido (401, no 400) y `DELETE` sin token de un código inexistente (401, no 404). AUT-06 recorre las rutas públicas con la cabecera ausente, correcta e incorrecta. AUT-03 incluye otro token, el esquema `Basic`, `Bearer` sin nada detrás y `bearer` en minúsculas con el token correcto (que debe aceptarse). AUT-04 comprueba que, con el token correcto, crear devuelve 201 y borrar 204 (basta con eso: el resto de códigos ya lo cubren los tests de la spec 004). Comprueba también que un `POST` rechazado no ha creado nada (el listado sigue vacío) y que un `DELETE` rechazado no ha borrado nada.
>
> **Los límites.** Solo puedes crear o modificar ficheros dentro de `server/`. No toques `cmd/`, `web/`, `go.mod` ni `go.sum`; no ejecutes `go mod tidy`; no añadas dependencias (`crypto/subtle` es de la biblioteca estándar y aún no lo necesitas). No hagas commits. No implementes la comprobación todavía. Si algún caso pasa con el `Handler` sin implementar, dilo en el informe. Ten en cuenta que al cambiar la firma de `Handler`, `cmd/acorta` dejará de compilar: es esperado, no lo arregles tú (es la fase 8b); `go vet ./...` fallará solo por eso, así que para «terminado» usa `go vet ./server/`.
>
> **Terminado es.** `gofmt -l .` vacío, `go vet ./server/` limpio, `go test ./server/` compilando, con los tests anteriores en verde y los de `auth_test.go` fallando por las aserciones.
>
> **El informe.** Responde con: ficheros creados o tocados; cuántos tests o casos nuevos hay y cuántos fallan; una tabla criterio → nombre del test; cualquier ambigüedad o contradicción que hayas encontrado en la spec 008 (no la resuelvas en silencio: dila); y las últimas líneas de la salida de `go test ./server/`.

## Informe: paso rojo (8a)

Lo que devolvió el agente, sin tocar (unos tres minutos de trabajo):

> ### Ficheros tocados (todos en `/home/claude/Projects/acorta/server/`)
> - `server.go` — `Handler` pasa a `Handler(svc Service, baseURL string, static fs.FS, token string) http.Handler`. El parámetro se acepta y se ignora (`_ = token` con un `TODO(8a)`); el comentario de doc ya es el del contrato de la 008.
> - `helpers_test.go` — nueva constante `testToken = "token-de-prueba-0123456789"` (26 caracteres) y la llamada a `server.Handler` en `newEnv` pasa `testToken`. Nada más.
> - `static_test.go` (`handlerFor`) y `api_test.go` (`TestAPI02_BarraFinalEnBaseURL`) — solo se añade `testToken` a la llamada a `Handler`.
> - `auth_test.go` — **nuevo**: helpers `doAuth` (petición con/sin `Authorization`), `wantUnauthorized` (401 + `WWW-Authenticate: Bearer` + `Content-Type: application/json` + cuerpo exacto, incluido el `\n` final) y `env.count()` (número de enlaces en el almacén), más los seis tests.
>
> No se ha tocado `cmd/`, `web/`, `go.mod`, `go.sum`; sin `go mod tidy`, sin dependencias nuevas, sin commits. Como se esperaba, `cmd/acorta` no compila (`main.go:289: not enough arguments in call to server.Handler`): queda para la 8b.
>
> ### Recuento
> - 6 funciones `TestAUT0x_…` con 39 subtests (45 entradas en `-v`).
> - **Fallan 26** (4 funciones: AUT-01, AUT-02, AUT-03, AUT-05, y 22 subtests), todas por aserciones (`estado = 201, quería 401`, `cabecera WWW-Authenticate = "", quería "Bearer"`, `hay 1 enlaces, quería 0`, etc.). Ningún pánico ni error de compilación.
> - **Pasan con el `Handler` sin implementar** (como advertías que podía ocurrir, lo digo): `TestAUT04_TokenCorrecto`, `TestAUT06_RutasPublicas` (los 14 subtests) y los 3 subtests `AUT-03 esquema bearer/BEARER/BeArEr se acepta`. Es lo lógico: son los criterios «positivos» (con token correcto o en rutas públicas, todo sigue igual) y un servidor que no comprueba nada los cumple. Su valor es de regresión para el paso verde: vigilan que la implementación no bloquee el token válido ni toque las rutas públicas.
> - Los 26 tests anteriores (API, RED, EST) siguen en verde.
>
> ### Criterio → test
> | Criterio | Test |
> |---|---|
> | AUT-01 | `TestAUT01_CrearSinToken` (401, cabecera, cuerpo exacto; el listado sigue vacío) |
> | AUT-02 | `TestAUT02_BorrarSinToken` → subtests `AUT-02 código existente` (401 y el enlace sigue existiendo, `HEAD /aaaaaa` da 302) y `AUT-02 código inexistente` |
> | AUT-03 | `TestAUT03_TokenIncorrecto` → `AUT-03 POST|DELETE {otro token, esquema Basic, Bearer sin nada detrás, Bearer con espacio y nada detrás, token sin esquema, token en mayúsculas, token con prefijo, token con sufijo}` (16 casos, comprobando además que no crean ni borran) y `AUT-03 esquema bearer|BEARER|BeArEr se acepta` (201 y 204) |
> | AUT-04 | `TestAUT04_TokenCorrecto` (201 al crear, 204 al borrar, sin `WWW-Authenticate`, el almacén refleja ambas operaciones) |
> | AUT-05 | `TestAUT05_TokenAntesQueCuerpo` → `AUT-05 POST sin token con cuerpo inválido` (tres cuerpos: no-JSON, validación fallida, lista → 401, no 400), `AUT-05 POST con token incorrecto y cuerpo inválido`, `AUT-05 DELETE sin token de código inexistente` (401, no 404), `AUT-05 DELETE con token incorrecto de código inexistente` |
> | AUT-06 | `TestAUT06_RutasPublicas` → `AUT-06 {GET,HEAD} /api/links`, `{GET,HEAD} /api/healthz`, `{GET,HEAD} /aaaaaa`, `GET /zzzzzz`, `GET /`, `GET /assets/app-3f9a.js`, `GET /favicon.svg`, `GET /nada.txt`, y los 405 `PUT /api/links`, `POST /api/healthz`, `GET /api/links/aaaaaa` (la nota de API-12 en la 008). Cada ruta se pide sin cabecera (referencia) y luego con cabecera correcta, incorrecta, `Basic` y `Bearer` a secas; se comparan estado, `Content-Type`, `Location`, `Allow`, `Cache-Control`, cuerpo, y que nunca aparece `WWW-Authenticate` |
>
> ### Ambigüedades / avisos sobre la spec 008 (no resueltos en silencio)
> 1. **Los tests existentes de la spec 004 quedarán en rojo en el paso verde.** `TestAPI02_*`, `API-03`, `API-04`, `API-05`, `API-06`, `API-07`, `API-09`, `API-10`, `API-13` hacen `POST`/`DELETE` sin `Authorization` y esperan 201/400/409/503/204/404. En cuanto se implemente la comprobación darán 401. AUT-04 lo justifica (con token correcto se comportan «exactamente como la spec 004»), pero el agente del paso verde tendrá que añadir la cabecera a esos tests (lo más limpio: que `do` en `helpers_test.go` envíe siempre `Bearer testToken` y que `auth_test.go` sea el único que la controle con `doAuth`). Conviene que el encargo del paso verde lo autorice explícitamente, porque la regla 3 de AGENTS.md («no modifiques un test para que pase») podría frenarlo.
> 2. **Espacios dentro de la cabecera.** La spec recorta el token al configurarlo, pero no dice qué pasa con `Authorization: Bearer  token` (dos espacios) o `Bearer token ` (espacio final). He tratado `Bearer ` (espacio y nada detrás) como «sin nada detrás» → 401 incorrecto. No he testeado los casos de espacios alrededor del token; la implementación debería decidir (yo recomendaría comparación estricta sin recortar) y, si se decide, añadirlo a la spec.
> 3. **Token vacío en `Handler`.** El contrato dice que «ninguna petición protegida se aceptaría», pero no es un criterio numerado ni dice con qué cuerpo (lo natural: `Bearer ` vacío → «token de API incorrecto», ausente → «falta el token de API»). No tiene test; si se quiere cubrir, necesita un identificador.
> 4. **`WWW-Authenticate: Bearer` exacto.** Lo compruebo con igualdad exacta. RFC 6750 permite `realm`/`error=…`; si alguien quisiera añadirlos, contradice la spec tal como está. Dicho por si acaso.
> 5. Sin contradicciones entre 004 y 008: la sección «Contrato del paquete» de la 004 ya trae la firma de cuatro parámetros.
>
> ### Estado de «terminado»
> - `gofmt -l .` → vacío.
> - `go vet ./server/` → limpio.
> - `go test ./server/` compila; anteriores en verde, `auth_test.go` en rojo por aserciones.
>
> Últimas líneas de `go test ./server/`:
> ```
>     --- FAIL: TestAUT05_TokenAntesQueCuerpo/AUT-05_DELETE_con_token_incorrecto_de_código_inexistente (0.00s)
>         auth_test.go:189: estado = 404, quería 401 (cuerpo "{\"errors\":[\"enlace no encontrado\"]}\n")
>         auth_test.go:189: cabecera WWW-Authenticate = "", quería "Bearer"
>         auth_test.go:189: cuerpo = "{\"errors\":[\"enlace no encontrado\"]}\n", quería "{\"errors\":[\"token de API incorrecto\"]}\n"
> FAIL
> FAIL	acorta/server	0.094s
> FAIL
> ```

## Revisión del orquestador (rojo 8a)

Comprobado de primera mano: solo hay cambios en `server/`; `gofmt` y `go vet ./server/` limpios; los 26 tests de las specs 004 siguen en verde y los de `auth_test.go` fallan por aserciones; los seis criterios AUT-01 a AUT-06 aparecen nombrados, con guion, en los tests.

De las ambigüedades:

| # | Hueco | Qué se hace |
|---|---|---|
| 1 | Los tests de la spec 004 darán 401 en el verde | Es del paso rojo, no del verde: `do` en `helpers_test.go` envía siempre el token correcto, y solo `auth_test.go` controla la cabecera. Escrito en la spec 008 («Verificación»). Así el commit verde no toca ningún test |
| 2 | Espacios en la cabecera | Comparación estricta, sin recortar: `Bearer  token` y `Bearer token ` son incorrectos. Escrito en AUT-03, con dos casos nuevos |
| 3 | Token vacío en `Handler` | Se queda sin criterio: desde fuera del paquete no se puede provocar (AUT-08 lo impide). Dicho en el «Contrato» |
| 4 | `WWW-Authenticate: Bearer` exacto | Es lo que dice la spec; se queda |

## Encargo: ajuste del rojo (8a)

> He comprobado el rojo de primera mano (solo `server/`, gofmt y vet limpios, 26 fallos por aserciones, los 26 anteriores en verde) y he cerrado en la spec 008 dos de tus ambigüedades: en AUT-03, la cabecera es exactamente esquema + un espacio + token, sin recortar (`Bearer  token` con dos espacios y `Bearer token ` con espacio final son incorrectos); y en «Verificación», que los tests de la spec 004 que crean o borran envían el token correcto, y solo los de `auth_test.go` controlan la cabecera. Vuelve a leer `specs/008-autenticacion.md` (AUT-03, «Contrato» y «Verificación»).
>
> Sigue siendo el **paso rojo**, y resuelve ahora tu ambigüedad 1 para que el paso verde no tenga que tocar ningún test:
>
> 1. En `server/helpers_test.go`, haz que `do` (y cualquier otro helper por el que pasen los tests de la spec 004) envíe siempre `Authorization: Bearer <testToken>`. Los tests de `auth_test.go` deben seguir controlando la cabecera ellos mismos con `doAuth` (incluido el caso «sin cabecera»). Con el `Handler` sin implementar, todo lo anterior debe seguir en verde.
> 2. Añade a AUT-03 los dos casos de espacios: `Bearer  token` (dos espacios) y `Bearer token ` (espacio final) → 401 «token de API incorrecto».
> 3. No implementes nada en `server.go`.
>
> Mismos límites: solo `server/`, sin commits, sin tocar `go.mod`. Terminado es `gofmt -l .` vacío, `go vet ./server/` limpio y `go test ./server/` compilando, con los tests anteriores en verde y los de `auth_test.go` fallando por las aserciones.
>
> Informe: qué has cambiado, el recuento nuevo (casos, cuántos fallan) y si queda alguna ambigüedad.

## Informe: ajuste del rojo (8a)

> He releído AUT-03, «Contrato del paquete» y «Verificación» de la spec 008 con tus cambios.
>
> ### Qué he cambiado (solo `server/`, sin commits, sin tocar `go.mod`; `server.go` sin implementación)
> - `/home/claude/Projects/acorta/server/helpers_test.go` — `do` ahora delega en `doAuth` enviando siempre `Authorization: Bearer <testToken>` (comentario que remite a AUT-04 y a que solo `auth_test.go` controla la cabecera). Era el único helper que construía peticiones en los tests de la 004 (ninguno llama a `httptest.NewRequest` ni a `ServeHTTP` directamente), así que todos los tests de API/RED/EST pasan por él. No se ha tocado ningún test de la 004.
> - `/home/claude/Projects/acorta/server/auth_test.go` — `doAuth` sigue siendo el único que decide la cabecera (incluido `""` = ausente). En la tabla de AUT-03 he añadido los dos casos de espacios, cada uno para `POST` y `DELETE`: `dos espacios tras Bearer` (`"Bearer  " + testToken`) y `espacio final tras el token` (`"Bearer " + testToken + " "`) → 401 «token de API incorrecto», y además que no crean ni borran. El comentario de AUT-03 cita la regla nueva (esquema, un espacio y token, sin recortar).
>
> Nota: `git status` muestra `specs/008-autenticacion.md` modificada; es tu cambio, yo no la he tocado.
>
> ### Recuento nuevo
> - 6 funciones `TestAUT0x_…` con 43 subtests (49 entradas en `-v`; antes 45).
> - **Fallan 30**: las 4 funciones AUT-01, AUT-02, AUT-03, AUT-05 y 26 subtests (AUT-02: 2; AUT-03: 20 = 10 variantes × POST/DELETE; AUT-05: 4). Todas por aserciones (`estado = 201, quería 401`, `WWW-Authenticate = "", quería "Bearer"`, `hay 1 enlaces, quería 0`…); cero pánicos, compila.
> - **Pasan 19** con el `Handler` sin implementar, como antes y por la misma razón (criterios positivos): `TestAUT04_TokenCorrecto`, `TestAUT06_RutasPublicas` con sus 14 subtests, y los 3 subtests `AUT-03 esquema bearer|BEARER|BeArEr se acepta`.
> - Los 26 tests anteriores (API, RED, EST) siguen en verde, ahora enviando el token correcto; en el paso verde no habrá que tocar ninguno.
>
> ### Terminado
> - `gofmt -l .` → vacío.
> - `go vet ./server/` → limpio.
> - `go test ./server/` → compila; últimas líneas: `FAIL` / `FAIL	acorta/server	0.100s` / `FAIL`.
> - `cmd/acorta` sigue sin compilar por la firma de `Handler` (esperado, fase 8b).
>
> ### Ambigüedades que quedan
> - Ninguna que afecte a los tests de la 8a. Las dos que cerraste (espacios en la cabecera; quién envía el token en los tests de la 004) están cubiertas.
> - Queda, como ya dice la spec, que el token vacío en `Handler` no tiene criterio ni test («no se puede provocar desde fuera del paquete»); si en el paso verde se quiere cubrir, sería un test interno (`package server`) y necesitaría identificador. Lo dejo apuntado, no es bloqueante.

Comprobado otra vez por el orquestador, contando casos hoja con `go test -json`: 44 casos de `auth_test.go`, 27 en rojo y 17 en verde (el agente cuenta funciones y subtests juntos, de ahí sus 30 y 19); los 26 de las specs 004 en verde; `git diff` de `api_test.go` y `static_test.go` se reduce al cuarto parámetro de `Handler`.

**Commits**: `Spec 008: la cabecera no se recorta, y quién envía el token en los tests` y `server: tests de la spec 008 (rojo)`.

## Encargo: paso verde (8a)

> Los tests del paso rojo ya están en un commit. Ahora, el **paso verde** de la fase 8a: implementa la comprobación del token en `server/` hasta que pase `go test ./server/`.
>
> **El método.** Lo mínimo que haga pasar los tests y cumpla la spec 008; nada que la spec no pida. El orden de comprobaciones es el de la spec: método (405 de API-12) antes que token, y token antes de leer el cuerpo (AUT-05). Comparación en tiempo constante con `crypto/subtle` (biblioteca estándar). El esquema `Bearer` sin distinguir mayúsculas; el token exacto, sin recortar (AUT-03). Las rutas públicas no miran la cabecera (AUT-06). Las respuestas 401 son JSON como el resto de la API (`respond.go`) y llevan `WWW-Authenticate: Bearer`. El token no se escribe en el log. **No modifiques ningún `_test.go`**: si crees que un test está mal o contradice la spec, párate y dilo en el informe en vez de cambiarlo. Cuando esté en verde, repasa el código y límpialo (nombres, duplicación, comentarios en español que expliquen el porqué) sin cambiar el comportamiento, con los tests en verde antes y después.
>
> **Los límites.** Solo `server/`. Solo la biblioteca estándar. Sin commits. No toques `cmd/` (sigue sin compilar hasta la 8b), `web/`, `go.mod` ni `go.sum`.
>
> **Terminado es.** `gofmt -l .` vacío, `go vet ./server/` limpio y `go test ./server/` en verde; `go test ./link/ ./store/ ./shortener/` siguen en verde.
>
> **El informe.** Ficheros tocados; decisiones que has tomado y por qué; desviaciones de **cualquier** spec, también del plan técnico (`specs/001`), aunque te parezcan justificadas; si has tocado algún test (no deberías); y la salida resumida de `go test ./server/`.

## Informe: paso verde (8a)

> `go test ./server/` en verde (135 entradas PASS, 0 FAIL, con `-count=1`); `go test ./link/ ./store/ ./shortener/` en verde; `gofmt -l .` vacío; `go vet ./server/` limpio. Ningún `_test.go` tocado. Sin commits, solo biblioteca estándar, solo `server/`.
>
> Nota: `git status` muestra también `bitacora/010-fase-8-autenticacion.md` modificada; no es mío (no he tocado nada fuera de `server/`).
>
> ### Ficheros tocados (`/home/claude/Projects/acorta/server/`)
> - `auth.go` — **nuevo**. `(*server).authorized(w, r) bool`: lee `Authorization`; vacía → 401 «falta el token de API»; si no, `strings.Cut(auth, " ")` separa esquema y token, y si el esquema no es `Bearer` (`strings.EqualFold`, sin distinguir mayúsculas) o el token no coincide → 401 «token de API incorrecto». `(*server).tokenMatches(got)`: `subtle.ConstantTimeCompare` y, además, `s.token != ""`.
> - `respond.go` — `writeUnauthorized(w, r, msg)`: pone `WWW-Authenticate: Bearer` y delega en `writeErrors` con 401 (JSON, `\n` final, como el resto de la API).
> - `api.go` — `POST /api/links`: `authorized` antes de `createLink` (antes de leer el cuerpo). `DELETE /api/links/{código}`: `methodNotAllowed(...) || !s.authorized(...)` antes de `deleteLink`, con comentario del orden (405 antes que token, token antes que buscar el código).
> - `server.go` — campo `token` en `server`, `Handler` lo guarda; fuera el `TODO(8a)`.
>
> ### Decisiones y por qué
> 1. **Sin recortar nada en la cabecera.** `Cut` por el primer espacio: `Bearer` solo → token vacío; `Bearer ` → token vacío; `Bearer  x` → token ` x`; `Bearer x ` → token `x `. Todos incorrectos, como manda AUT-03 tras tu cambio.
> 2. **Token del servidor vacío no acepta nada.** `subtle.ConstantTimeCompare` devuelve 1 con dos cadenas vacías, así que `Bearer ` (vacío detrás) habría pasado con `Handler(..., "")`. El guard `s.token != ""` cumple el párrafo de «Contrato del paquete» (sin cabecera → «falta…», con `Bearer` y lo que sea → «incorrecto»). Está comentado en el código. No tiene test, como dice la spec.
> 3. **Tiempo constante sobre longitudes distintas.** `ConstantTimeCompare` devuelve 0 de inmediato si las longitudes difieren, así que filtra la longitud del token, no su contenido. Es lo que pide la spec («`crypto/subtle`») y el uso estándar; ocultar también la longitud (hash SHA-256 de ambos antes de comparar) sería añadir algo que la spec no pide. Lo apunto por si se quiere endurecer la spec.
> 4. **Solo la primera cabecera `Authorization`.** `r.Header.Get` devuelve la primera si hay varias. La spec no contempla el caso; es el comportamiento habitual.
> 5. **Los mensajes 401 viven en `server`**, como «ruta no encontrada» o «método no permitido»: no son errores de validación de `link` (regla 7 de AGENTS.md), sino del adaptador HTTP.
> 6. **Nada en el log**: ningún `log.Printf` nuevo; el único existente (`writeInternal`) sigue sin tocar la cabecera.
> 7. Repaso final: no había duplicación que limpiar; los comentarios explican el porqué (orden de comprobaciones, el porqué del guard de token vacío, el porqué de no recortar). Tests en verde antes y después del repaso.
>
> ### Desviaciones de specs
> - **Ninguna del comportamiento** de la 008 ni de la 004.
> - **Plan técnico (001) y AGENTS.md, regla 4**: «terminado» exige `go vet ./...` y `go test ./...` en todo el repo, y `cmd/acorta` no compila (`main.go:289: not enough arguments in call to server.Handler`) hasta la 8b. Es la desviación temporal prevista en `specs/007-fases.md` (la 8b depende de la 8a). Para este encargo se ha usado `go vet ./server/` y los cuatro paquetes de Go que compilan, como pediste.
>
> ### Salida resumida
> ```
> ok  	acorta/server	0.099s
> ok  	acorta/link	(cached)
> ok  	acorta/store	(cached)
> ok  	acorta/shortener	(cached)
> ```
> `go test ./server/ -v`: 135 PASS, 0 FAIL (26 de la 004 + 49 entradas de `auth_test.go` + subtests de redirect/static).

## Verificación del orquestador (verde 8a)

- `git diff --stat -- '*_test.go'`: vacío. `gofmt` y `go vet ./server/` limpios; `go test -count=1` en verde en `server`, `link`, `store` y `shortener`; cobertura de `server` 93,3 %.
- Leído `server/auth.go` entero (unas 40 líneas) con la spec 008 al lado: el orden es método → token → cuerpo; `strings.Cut` por el primer espacio hace exactamente lo que dice AUT-03; el guard del token vacío cumple el párrafo del contrato; `crypto/subtle` como pide la spec. Las dos decisiones que la spec no cubre (solo la primera cabecera `Authorization`; la comparación filtra la longitud) se quedan como están: ninguna cambia lo que ve un usuario.
- La desviación que declara (el repo entero no compila) es la prevista en la spec 007 y se cierra en la 8b.

**Commit**: `server: token de API en crear y borrar (verde)`.
