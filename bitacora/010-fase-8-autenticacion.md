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

## Encargo: paso rojo (8b, `cmd/acorta`)

> **Qué leer antes.** Trabajas en el repositorio `/home/claude/Projects/acorta`. Antes de escribir nada lee, en este orden: `AGENTS.md`, `specs/001-plan-tecnico.md`, `specs/008-autenticacion.md` entera (sobre todo las secciones «El token», «CLI» y «Contrato del paquete») y `specs/005-cli.md` (la sinopsis de `serve`, la tabla de configuración, los códigos de salida y CLI-09, CLI-10 y CLI-13). Mira el código que vas a tocar: `cmd/acorta/main.go` (en especial `pick`, `options`, `parse` y el subcomando `serve`) y `cmd/acorta/main_test.go` (cómo se prueba `serve` sin abrir sockets, con el `listen` inyectado, y cómo se fijan variables de entorno en los tests). Mira también la firma nueva de `server.Handler` en `server/server.go`: ahora recibe el token como cuarto parámetro, y por eso `cmd/acorta` no compila ahora mismo.
>
> **La tarea.** Eres el agente de la fase 8b de `specs/007-fases.md`: la autenticación en la CLI. Este encargo es solo el **paso rojo**:
>
> 1. Haz que `cmd/acorta` vuelva a compilar con el mínimo cambio: la llamada a `server.Handler` pasa de momento una cadena vacía como token. Nada más en `main.go` (ni el flag, ni la variable de entorno, ni las comprobaciones: eso es el verde).
> 2. Escribe los tests de AUT-07 a AUT-10 en `cmd/acorta/main_test.go` (o en un fichero nuevo `cmd/acorta/auth_test.go`, como prefieras), con el estilo de los existentes: contra `run`, con `listen` inyectado, sin procesos ni sockets. Cada test lleva en el nombre de la función el identificador sin guion (`TestAUT07_…`) y, en un comentario justo encima, con guion (`// AUT-07: …`), como ya hacen los tests de este paquete. Cubre: el flag `-token` llega a `server.Handler` (necesitarás una forma de ver qué token recibió `Handler`: mira cómo el test de CLI-09 captura el handler y piensa en comprobarlo con una petición `POST /api/links` con y sin `Authorization` contra ese handler, que es lo que distingue un token de otro); la variable `ACORTA_TOKEN` cuando no hay flag; el flag gana a la variable; sin token (ni flag ni variable, y también con la variable vacía) `error: falta el token de API: usa -token o ACORTA_TOKEN` en `stderr`, salida 1, sin abrir la base de datos ni llamar a `listen`; token de menos de 16 caracteres tras recortar espacios (prueba con 15 y con 16, y con uno de 16 rodeado de espacios, que vale) `error: el token de API debe tener al menos 16 caracteres`, salida 1, sin `listen`; y que el mensaje de arranque de CLI-09 y la ayuda de `serve -h` no contienen el token (la ayuda puede nombrar el flag y la variable).
> 3. Ejecuta `go test ./cmd/acorta/` y comprueba que los tests nuevos **fallan por las aserciones**, no por errores de compilación ni por pánicos, y que los anteriores siguen en verde.
>
> **El método.** Tests de tabla donde haya varios casos. Para «sin abrir la base de datos» usa una ruta de base de datos en un directorio que no existe: si `serve` la abriera, fallaría por CLI-13 con otro mensaje; el test exige el mensaje del token y la salida 1. Fija las variables de entorno con `t.Setenv`. El orden de comprobación que pide la spec es: primero el token (AUT-08, AUT-09), después la base de datos.
>
> **Los límites.** Solo puedes crear o modificar ficheros dentro de `cmd/acorta/`. No toques `server/`, `web/`, `go.mod` ni `go.sum`; no ejecutes `go mod tidy`; no añadas dependencias. No hagas commits. No implementes el flag ni las comprobaciones todavía. Si algún caso pasa con el código sin implementar, dilo en el informe.
>
> **Terminado es.** `gofmt -l .` vacío, `go vet ./...` limpio en todo el repo (ahora que vuelve a compilar), `go test ./cmd/acorta/` compilando, con los tests anteriores en verde y los de AUT-07 a AUT-10 fallando por las aserciones; el resto de paquetes sigue en verde.
>
> **El informe.** Ficheros creados o tocados; cuántos tests o casos nuevos hay y cuántos fallan; una tabla criterio → nombre del test; cualquier ambigüedad o contradicción de la spec 008 o de la 005 (no la resuelvas en silencio: dila); y las últimas líneas de `go test ./cmd/acorta/`.

## Encargo: paso rojo (8c, `web`)

> **Qué leer antes.** Trabajas en el repositorio `/home/claude/Projects/acorta`. Antes de escribir nada lee, en este orden: `AGENTS.md`, `specs/001-plan-tecnico.md` (stack y órdenes de la interfaz), `specs/008-autenticacion.md` entera (sobre todo «Interfaz web», AUT-11 a AUT-13, y «API») y `specs/006-web.md`. Mira el código que vas a tocar: `web/src/api.js`, `web/src/App.vue`, `web/src/components/` y, con calma, `web/src/App.test.js`: cómo se monta la aplicación entera con `fetch` simulado, cómo se nombran los tests (`WEB-01: …`) y cómo se comprueban las peticiones.
>
> **La tarea.** Eres el agente de la fase 8c de `specs/007-fases.md`: la autenticación en la interfaz web. Este encargo es solo el **paso rojo**:
>
> 1. Crea el componente vacío `web/src/components/TokenField.vue` (un campo **Token de API** `type="password"` con su `<label>` y un botón **Guardar**, sin lógica todavía) y móntalo en `App.vue` encima del formulario de creación, para que los tests puedan encontrarlo. Nada más de implementación: ni `sessionStorage`, ni la cabecera, ni el foco.
> 2. Escribe los tests de AUT-11, AUT-12 y AUT-13 en `web/src/App.test.js` (o en un fichero nuevo `web/src/auth.test.js` que monte `App` igual), con el mismo estilo y nombrados `AUT-11: …`. Cubre: al pulsar Guardar se guarda en `sessionStorage` bajo `acorta_token` el token recortado y el campo muestra el estado `Token guardado`; al cargar la página con un token ya guardado, el campo aparece relleno y en ese estado; guardar vacío borra la clave; crear (WEB-01) y borrar (WEB-12) envían `Authorization: Bearer <token>` cuando hay token guardado y no envían la cabecera cuando no lo hay; cargar el listado (WEB-08) y Actualizar (WEB-13) no la envían nunca, con o sin token; ante un `401` al crear o al borrar, el error se muestra tal como llega (`{"errors":["falta el token de API"]}` → el texto `falta el token de API`), el campo del token recibe el foco y el token guardado no se borra.
> 3. Ejecuta `npm test` en `web/` y comprueba que los tests nuevos **fallan por las aserciones**, no por errores de sintaxis, y que los 38 anteriores siguen en verde.
>
> **El método.** Vitest con `@vue/test-utils` y el `fetch` simulado que ya usan los tests; `sessionStorage` real de jsdom (límpialo en un `beforeEach`). Para el foco, monta con `attachTo: document.body` y comprueba `document.activeElement`. Para la cabecera, inspecciona los argumentos con los que se llamó a `fetch` (`headers`). Comprueba la ausencia de cabecera, no solo la presencia.
>
> **Los límites.** Solo puedes crear o modificar ficheros dentro de `web/`. No toques `server/`, `cmd/`, `go.mod` ni `go.sum`. No añadas dependencias a `web/package.json`. No hagas commits. No implementes todavía. Si algún test pasa con el componente vacío, dilo en el informe. No ejecutes `npm run build` (escribe en `server/static/dist/`, que no está en Git; no hace falta en el rojo).
>
> **Terminado es.** `npm test` en `web/` compilando: los 38 tests anteriores en verde y los nuevos fallando por las aserciones.
>
> **El informe.** Ficheros creados o tocados; cuántos tests nuevos hay y cuántos fallan; una tabla criterio → nombre del test; cualquier ambigüedad o contradicción de la spec 008 o de la 006 (no la resuelvas en silencio: dila); y las últimas líneas de `npm test`.

## Informe: paso rojo (8b)

> ### Ficheros tocados (solo dentro de `cmd/acorta/`)
> - `/home/claude/Projects/acorta/cmd/acorta/main.go` — un solo cambio: `server.Handler(svc, base, server.Static(), "")` (línea 289). Nada más: ni flag, ni variable, ni comprobaciones, ni sinopsis.
> - `/home/claude/Projects/acorta/cmd/acorta/auth_test.go` — **nuevo**, con los tests AUT-07 a AUT-10 y sus ayudantes (`postLinks`, `serveProbe`, `wantServed`, `badDB`).
>
> No he tocado `main_test.go`, `server/`, `web/`, `go.mod`, `go.sum`; no he hecho `go mod tidy` ni commits. Ojo: `git status` muestra además `bitacora/010-…`, `web/src/App.vue`, `web/src/App.test.js` y `web/src/components/TokenField.vue` modificados o nuevos; **no son míos** (supongo que son de la fase 8c en paralelo).
>
> ### Tests nuevos
> 6 funciones, 17 casos en total (contando subtests de tabla y bucles). **Fallan los 6 / los 17**, todos por aserciones (no hay errores de compilación ni pánicos). Ningún caso pasa con el código sin implementar. Los 24 tests anteriores siguen en verde; `link`, `server`, `shortener`, `store` en verde; `gofmt -l .` vacío; `go vet ./...` limpio.
>
> Cómo se ve qué token recibió `Handler`: dentro del `listen` inyectado (mientras se sirve, antes de que `serve` cierre la base) se hace `POST /api/links` con un alias distinto por sonda, con y sin `Authorization`: `201` = el handler aceptó ese token; `401` = no. Así «solo el flag», «solo la variable» y «el flag gana» se distinguen de verdad (en el último, el token del entorno debe dar 401).
>
> | Criterio | Test | Casos |
> |---|---|---|
> | AUT-07 (fuente del token y precedencia) | `TestAUT07_TokenSource` | solo flag; solo `ACORTA_TOKEN`; flag gana a variable (3 subtests de tabla) |
> | AUT-07 (sinopsis con `[-token TOKEN]`) | `TestAUT07_Synopsis` | `serve -h` (stdout) y `serve -nope` (stderr) |
> | AUT-08 (falta el token) | `TestAUT08_MissingToken` | sin flag ni variable (unset); variable vacía; `-token ""` con variable vacía (3 subtests). Base en directorio inexistente + `listen` que hace `t.Error` |
> | AUT-09 (longitud mínima 16 tras recortar) | `TestAUT09_TokenLength` | 15 por flag; 15 por variable; 15 rodeado de espacios → error, salida 1, sin listen, base inexistente. 16 por flag; 16 por variable; 16 rodeado de espacios → arranca y el handler acepta el token **recortado** (6 subtests) |
> | AUT-10 (mensaje de arranque) | `TestAUT10_StartupMessageHidesToken` | con `-token` y con `-addr :9090`; stdout exacto, sin el token del flag ni el del entorno (2 casos) |
> | AUT-10 (ayuda) | `TestAUT10_HelpHidesToken` | `serve -h` con `ACORTA_TOKEN` definida; `serve -token X -h`: menciona `-token` y `ACORTA_TOKEN`, no contiene ningún token (2 casos) |
>
> ### Ambigüedades / avisos sobre las specs (no resueltas en silencio)
> 1. **Token solo de espacios** (`-token "   "` o `ACORTA_TOKEN="   "`): la spec 005 dice que una variable *vacía* cuenta como no definida, y AUT-08 habla de «variable vacía»; pero AUT-09 cuenta los caracteres *después de recortar*, así que `"   "` → 0 < 16 → mensaje de AUT-09. Leído literalmente es AUT-09, pero podría querer ser AUT-08. No lo he cubierto en ningún test; que lo decida el orquestador.
> 2. **El verde romperá tests existentes de la spec 005, legítimamente**: `TestCLI09_Serve`, `TestCLI09_ServeDefaultAddr`, `TestCLI09_ServeHostInAddr`, `TestCLI09_ServeListenError` y el caso `serve` de `TestCLI13_CannotOpenDB` arrancan `serve` sin token (pasarán a AUT-08 con salida 1); y la constante `usageServe` de `main_test.go` (usada por `TestCLI12_SubcommandHelp` y `TestCLI_UnknownFlagIsUsageError`) lleva la sinopsis vieja sin `[-token TOKEN]`. La spec 005 ya está actualizada (sinopsis y tabla de configuración), así que en el verde habrá que dar un token a esos tests y actualizar `usageServe`; no es «modificar un test para que pase» sino seguir la spec. No lo he hecho ahora porque el encargo pide que los anteriores sigan en verde en el paso rojo.
> 3. **AUT-09 y el token recortado**: la spec dice «sin espacios en los extremos (se recortan)», así que he asumido que `Handler` recibe el token recortado y lo compruebo (`Bearer 1234567890123456` da 201 cuando el flag fue `"  1234567890123456  "`). Si la intención fuera solo *contar* tras recortar pero pasar el valor tal cual, ese caso tendría que cambiar.
> 4. **AUT-09 no dice nada de la base de datos** («no llega a escuchar»), pero según tu indicación (token antes que base) los casos de error usan una base en directorio inexistente y exigen el mensaje del token; la spec 008 solo lo dice explícitamente para AUT-08.
>
> ### Últimas líneas de `go test ./cmd/acorta/`
> ```
> --- FAIL: TestAUT10_HelpHidesToken (0.00s)
>     auth_test.go:290: [serve -h]: la ayuda no menciona -token: "uso: acorta serve [-addr :8080] [-db RUTA] [-base-url URL]\n  -addr string\n ..."
>     auth_test.go:290: [serve -h]: la ayuda no menciona ACORTA_TOKEN: "..."
>     auth_test.go:286: [serve -token token-del-flag-0123456789 -h]: código=2 stderr="error: flag provided but not defined: -token\nuso: acorta serve [-addr :8080] [-db RUTA] [-base-url URL]\n", quiero 0 y vacío
>     auth_test.go:290: [serve -token token-del-flag-0123456789 -h]: la ayuda no menciona -token: ""
>     auth_test.go:290: [serve -token token-del-flag-0123456789 -h]: la ayuda no menciona ACORTA_TOKEN: ""
> FAIL
> FAIL	acorta/cmd/acorta	0.028s
> FAIL
> ```
> `go test ./...`: solo falla `acorta/cmd/acorta`; `link`, `server`, `shortener` y `store` → `ok`.

## Informe: paso rojo (8c)

> ### Ficheros creados o tocados (solo dentro de `web/`)
> - **Creado** `/home/claude/Projects/acorta/web/src/components/TokenField.vue`: `<label for="token">Token de API</label>`, `<input id="token" type="password">` y botón **Guardar** (`type="button"`). Sin lógica.
> - **Tocado** `/home/claude/Projects/acorta/web/src/App.vue`: importa `TokenField` y lo monta encima de `<LinkForm>`.
> - **Tocado** `/home/claude/Projects/acorta/web/src/App.test.js`: `sessionStorage.clear()` en el `beforeEach` global, ayudas nuevas (`headerOf`, `authHeaders`, `tokenField`, `saveToken`, `expectTokenFocused`) y un `describe('autenticación')` al final con 18 tests. No hay fichero `auth.test.js`: reutilizo las ayudas existentes (`mockApi`, `field`, `button`, `submit`, `settle`, `mountApp` ya monta con `attachTo: document.body`).
>
> No he tocado `server/`, `cmd/`, `go.mod`, `go.sum` ni `package.json`. Ojo: `git status` muestra cambios ajenos en `cmd/acorta/main.go`, `cmd/acorta/auth_test.go` y `bitacora/010-fase-8-autenticacion.md`; no son míos (serán de la fase 8b en paralelo). No he hecho commits ni `npm run build`.
>
> ### Resultado: 18 tests nuevos, 10 fallan, 8 pasan
> Los 38 anteriores siguen en verde (46 = 38 + 8). Los 10 fallos son todos `AssertionError`, ninguno de sintaxis o de montaje.
>
> | Criterio | Test | Rojo |
> |---|---|---|
> | AUT-11 | `AUT-11: el campo Token de API es type="password", tiene su <label> y va encima del formulario de creación` | pasa (estructural, lo cubre el componente vacío) |
> | AUT-11 | `AUT-11: al pulsar Guardar guarda el token recortado bajo acorta_token y muestra «Token guardado»` | falla (`expected null to be 'token-de-prueba-123456'`) |
> | AUT-11 | `AUT-11: al cargar con un token guardado, el campo aparece relleno y en estado «Token guardado»` | falla |
> | AUT-11 | `AUT-11: al cargar sin token guardado, el campo está vacío y no muestra «Token guardado»` | pasa (trivial sin lógica) |
> | AUT-11 | `AUT-11: guardar un campo vacío borra el token guardado` / `… solo espacios …` (it.each, 2) | fallan |
> | AUT-12 | `AUT-12: crear (WEB-01) envía Authorization: Bearer <token guardado>` | falla (`[undefined]` vs `['Bearer …']`) |
> | AUT-12 | `AUT-12: crear usa el token guardado con Guardar en la misma sesión, sin recargar` | falla |
> | AUT-12 | `AUT-12: crear sin token guardado no envía la cabecera Authorization` | pasa (hoy nadie envía la cabecera) |
> | AUT-12 | `AUT-12: tras borrar el token con Guardar vacío, crear ya no envía la cabecera` | pasa (ídem) |
> | AUT-12 | `AUT-12: borrar (WEB-12) envía Authorization: Bearer <token guardado>` | falla |
> | AUT-12 | `AUT-12: borrar sin token guardado no envía la cabecera Authorization` | pasa (ídem) |
> | AUT-12 | `AUT-12: cargar el listado (WEB-08) y Actualizar (WEB-13) no envían Authorization con/sin token guardado` (it.each, 2) | pasan (ídem; son guardas contra regresión) |
> | AUT-13 | `AUT-13: un 401 al crear sin token muestra «falta el token de API» tal cual y el campo del token recibe el foco` | falla (foco en `<body>`) |
> | AUT-13 | `AUT-13: un 401 al crear con token guardado muestra el error tal cual, da el foco al token y no lo borra` | falla (foco) |
> | AUT-13 | `AUT-13: un 401 al borrar muestra el error tal cual, el enlace sigue en el listado, el token recibe el foco y no se borra` | falla (foco) |
> | AUT-13 | `AUT-13: los demás errores al crear (400) no mueven el foco al campo del token` | pasa (guarda contra regresión) |
>
> Cómo se comprueba cada cosa: `sessionStorage` real de jsdom; la cabecera se inspecciona en `fetch.mock.calls[i][1].headers` sin distinguir mayúsculas y aceptando objeto plano o `Headers`; la ausencia se afirma con `toEqual([undefined])` para cada petición, no solo la presencia; el foco con `document.activeElement === input#token`.
>
> ### Ambigüedades y decisiones que he tomado en los tests (no están resueltas en la spec; revísalas)
> 1. **Spec 008, AUT-11**: no dice si al guardar vacío desaparece el estado «Token guardado». He asumido que sí (`not.toContain('Token guardado')` tras borrar). Tampoco dice si «Guardar» con solo espacios cuenta como vacío; por «recortado» he asumido que sí (caso `'   '`).
> 2. **Spec 006, Diseño** («los mensajes de estado y de error están en una región `aria-live="polite"`») aplicado a «Token guardado»: el test exige que ese texto esté dentro de un `[aria-live="polite"]`. Si el estado se quiere mostrar de otra forma (p. ej. solo visual), hay que decidirlo en la spec.
> 3. **Spec 008, AUT-13**: el texto de ejemplo es `falta el token de API`; con un token guardado el servidor respondería `token de API incorrecto` (AUT-03). He usado el primero sin token y el segundo con token, ambos «tal como llegan». No se especifica si editar el campo del token tras un 401 debe limpiar el error (WEB-06 habla de los campos del formulario); no lo he testeado.
> 4. **Spec 006 WEB-04** enumera `400, 409, 503`; el 401 de AUT-13 no está en esa lista, aunque AUT-13 dice «como cualquier otro». Aviso por si se quiere añadir el 401 a WEB-04.
> 5. **Mínimo de 16 caracteres**: la web no valida (spec 006: sin reglas de negocio), así que no hay test; el token de prueba cumple el mínimo por si en el futuro se decide validar en cliente.
> 6. **Restricción para el paso verde**: la ayuda `submit()` de los tests hace `wrapper.find('form')` (el primer `<form>` del documento). Si `TokenField` se implementa con un `<form>` (natural para guardar con Enter), romperá los 38 tests de WEB-01 a WEB-07. O se implementa sin `<form>` (botón `type="button"`, como está ahora), o antes se cambia la ayuda a algo más específico. No lo he resuelto porque tocar las ayudas afecta a tests existentes.
> 7. No se especifica si, tras guardar con espacios, el campo debe mostrar el valor recortado; no lo afirmo.
>
> ### Últimas líneas de `npm test` (en `web/`)
> ```
>  FAIL  src/App.test.js > autenticación > AUT-13: un 401 al borrar muestra el error tal cual, el enlace sigue en el listado, el token recibe el foco y no se borra
> AssertionError: el campo Token de API no tiene el foco: expected <body><div data-v-app>…(1)</div></body> to be <input id="token" …(3)></input> // Object.is equality
>
>  Test Files  1 failed (1)
>       Tests  10 failed | 46 passed (56)
>    Start at  01:06:16
>    Duration  552ms (environment 44%, tests 36%, transform 13%, import 6%)
> ```

## Revisión del orquestador (rojos 8b y 8c)

Comprobado de primera mano: cada agente tocó solo sus rutas; `gofmt` y `go vet ./...` limpios (el repo vuelve a compilar); 17 casos nuevos de la CLI en rojo y los 24 anteriores en verde; 10 de los 18 nuevos de la web en rojo y los 38 anteriores en verde. Los ocho de la web que pasan con el componente vacío son los negativos de siempre («no envía la cabecera») y uno estructural; quedan de guardia.

Once ambigüedades entre los dos. Las que cambian la spec:

| # | Hueco | Cómo queda |
|---|---|---|
| 8b-1 | Token de solo espacios: ¿AUT-08 o AUT-09? | AUT-08: lo que queda vacío al recortar es «falta el token». AUT-09 es de 1 a 15 |
| 8b-2 | Los tests de CLI-09 y CLI-13 arrancan `serve` sin token; `usageServe` lleva la sinopsis vieja | Se arreglan en el rojo, no en el verde: token por `t.Setenv`, sinopsis nueva. Escrito en «Verificación» |
| 8b-3, 8b-4 | ¿`Handler` recibe el token recortado? ¿Token antes que base de datos en AUT-09? | Sí a las dos; ahora lo dice AUT-09 |
| 8c-1, 8c-7 | Guardar vacío o solo espacios; el campo tras guardar con espacios | Borra la clave y quita el estado; el campo muestra el valor recortado (AUT-11) |
| 8c-2 | `Token guardado` en la región `aria-live` | Sí (AUT-11) |
| 8c-4 | WEB-04 no listaba el 401 | Lo lista, y dice que con él el foco va al token |
| 8c-6 | `TokenField` dentro de un `<form>` rompería `submit()` de los tests | Sin `<form>`; Guardar o Enter en el campo (AUT-11) |
| 8c-3 | ¿Editar el token tras un 401 limpia el error? | No se exige nada; se queda fuera |
| 8c-5 | ¿Valida la web los 16 caracteres? | No: sin reglas de negocio en la web (spec 006) |

## Encargo: ajuste del rojo (8b)

> Comprobado de primera mano (solo `cmd/acorta/`, gofmt y vet limpios, los 24 anteriores en verde, los 17 nuevos en rojo por aserciones). He cerrado tus ambigüedades en la spec 008; vuelve a leer AUT-08, AUT-09 y «Verificación»:
>
> - Un token que queda vacío después de recortar (`"   "`) es AUT-08 («falta el token»), no AUT-09. AUT-09 es de 1 a 15 caracteres tras recortar.
> - AUT-09 dice ya que el token se comprueba antes que la base de datos y que `Handler` recibe el token recortado: tus suposiciones 3 y 4 eran correctas y ahora están escritas.
> - «Verificación»: los tests de la spec 005 que arrancan `serve` lo hacen con un token válido y su sinopsis es la nueva; solo los de la 008 prueban qué pasa sin él. Es decir, tu ambigüedad 2 se resuelve ahora, en el rojo, no en el verde.
>
> Sigue siendo el **paso rojo**:
>
> 1. Añade a AUT-08 el caso del token solo de espacios, por flag y por variable → `error: falta el token de API: usa -token o ACORTA_TOKEN`, salida 1, sin abrir la base ni `listen`.
> 2. En `main_test.go`, da un token válido (por `t.Setenv("ACORTA_TOKEN", …)`, que no cambia los argumentos de los tests) a `TestCLI09_Serve`, `TestCLI09_ServeDefaultAddr`, `TestCLI09_ServeHostInAddr`, `TestCLI09_ServeListenError` y al caso `serve` de `TestCLI13_CannotOpenDB`, y actualiza la constante `usageServe` a la sinopsis nueva de la spec 005 (`acorta serve [-addr :8080] [-db RUTA] [-base-url URL] [-token TOKEN]`). Con el código sin implementar, los de CLI-09 y CLI-13 seguirán en verde (la variable se ignora) y los que usan `usageServe` pasarán a rojo: es correcto, porque la sinopsis todavía no existe. Dilo en el recuento.
> 3. Nada más en `main.go`.
>
> Mismos límites: solo `cmd/acorta/`, sin commits. Terminado es `gofmt -l .` vacío, `go vet ./...` limpio y `go test ./cmd/acorta/` compilando, con los fallos que acabo de describir y nada más en rojo.
>
> Informe: qué has cambiado, el recuento nuevo (qué tests fallan y por qué) y si queda alguna ambigüedad.

## Encargo: ajuste del rojo (8c)

> Comprobado de primera mano (solo `web/`, 46 en verde y 10 en rojo por aserciones). He cerrado tus ambigüedades en las specs; vuelve a leer AUT-11 de la 008 y WEB-04 de la 006:
>
> - AUT-11: Guardar **o Enter dentro del campo** guarda el token recortado; el campo pasa a mostrar el valor recortado; el estado `Token guardado` va en la región `aria-live` (tu suposición 2, correcta); guardar vacío o solo espacios borra la clave y quita el estado (tu suposición 1, correcta); y el campo **no está dentro de un `<form>`**, el formulario de la página es el de creación. Eso resuelve tu aviso 6 sin tocar las ayudas.
> - WEB-04 incluye ya el `401`, y dice que con él el foco va al campo del token.
> - Editar el campo del token después de un 401 no limpia el error: no se exige nada, así que no lo testees.
>
> Sigue siendo el **paso rojo**: añade los casos que faltan según lo anterior: Enter en el campo guarda igual que el botón; tras guardar `'  token-con-espacios-1234  '`, el campo muestra el valor recortado; y que `TokenField` no contiene ni está dentro de un `<form>`. Nada más de implementación.
>
> Mismos límites: solo `web/`, sin commits, sin `npm run build`. Terminado es `npm test` compilando, los 38 anteriores en verde y los nuevos que deban fallar, fallando por aserciones.
>
> Informe: qué has cambiado, el recuento nuevo y si queda alguna ambigüedad.

## Informes: ajuste de los rojos (8b y 8c)

8b:

> He releído AUT-08, AUT-09 y «Verificación» de la spec 008 tal como están ahora y he ajustado los tests. Sigue sin haber nada nuevo en `main.go` más allá del `""` de la primera vuelta. Solo `cmd/acorta/`, sin commits.
>
> ### Qué he cambiado
> - `/home/claude/Projects/acorta/cmd/acorta/auth_test.go`
>   - `TestAUT08_MissingToken`: dos casos nuevos, «flag solo de espacios» (`-token "   "`) y «variable solo de espacios» (`ACORTA_TOKEN="   "`), con el mismo contrato que el resto de la tabla: `error: falta el token de API: usa -token o ACORTA_TOKEN`, salida 1, base en directorio inexistente (si la abriera fallaría por CLI-13 con otro mensaje) y `listen` que hace `t.Error`. La tabla ahora lleva un campo `env` para el valor de la variable.
>   - Comentarios de AUT-08 y AUT-09 alineados con la redacción nueva («queda vacío tras recortar», «entre 1 y 15», «antes que la base de datos»). Las aserciones de AUT-09 no cambian: ya comprobaban la base inexistente y el token recortado en el handler, que ahora están escritos en la spec.
> - `/home/claude/Projects/acorta/cmd/acorta/main_test.go`
>   - `usageServe` → `uso: acorta serve [-addr :8080] [-db RUTA] [-base-url URL] [-token TOKEN]`.
>   - Nuevo `const validToken` y ayudante `withToken(t)`: pone `env.getenv = os.Getenv` (necesario: `setup` deja un `getenv` que siempre devuelve `""`, así que `t.Setenv` solo no llegaría a `run`), vacía `ACORTA_DB` y `ACORTA_BASE_URL` con `t.Setenv` para que no se cuele el entorno de quien ejecuta, y fija `ACORTA_TOKEN=validToken`. Los argumentos de los tests no cambian.
>   - `withToken(t)` añadido tras `setup(t)` en `TestCLI09_Serve`, `TestCLI09_ServeDefaultAddr`, `TestCLI09_ServeHostInAddr`, `TestCLI09_ServeListenError` y `TestCLI13_CannotOpenDB` (este último con un comentario: el caso `serve` arranca con token válido porque ahí se prueba la base, no el token).
>
> ### Recuento
> `gofmt -l .` vacío; `go vet ./...` limpio; `go test ./cmd/acorta/` compila; `link`, `server`, `shortener`, `store` en `ok`.
>
> En `cmd/acorta`: 30 funciones de test, **8 en rojo, 22 en verde**. Todo lo rojo es por aserciones; ni pánicos ni errores de compilación.
>
> | Test | Estado | Por qué |
> |---|---|---|
> | `TestAUT07_TokenSource` (3 subtests) | rojo | `-token` no existe (flag desconocido, salida 2); con variable, el handler recibe `""` y responde 401 al token |
> | `TestAUT07_Synopsis` | rojo | la sinopsis aún no lleva `[-token TOKEN]` |
> | `TestAUT08_MissingToken` (5 subtests, 2 nuevos) | rojo | sin flag abre la base y falla por CLI-13 con otro mensaje; con `-token` es flag desconocido (2) |
> | `TestAUT09_TokenLength` (6 subtests) | rojo | ídem; los casos de 16 fallan además porque el handler no acepta el token |
> | `TestAUT10_StartupMessageHidesToken` | rojo | `-token` desconocido |
> | `TestAUT10_HelpHidesToken` | rojo | la ayuda no menciona `-token` ni `ACORTA_TOKEN`; con `-token X -h` sale 2 |
> | `TestCLI12_SubcommandHelp` | rojo (**nuevo en rojo**) | `serve -h` no empieza por la sinopsis nueva de `usageServe` |
> | `TestCLI_UnknownFlagIsUsageError` | rojo (**nuevo en rojo**) | el uso de `serve -nope` no es la sinopsis nueva |
> | `TestCLI09_Serve`, `…DefaultAddr`, `…HostInAddr`, `…ListenError`, `TestCLI13_CannotOpenDB` | verde | la variable `ACORTA_TOKEN` se ignora con el código actual, como previste |
> | Resto de la 005 (CLI-01 a CLI-11, CLI-12 `Help` y `UnknownSubcommand`) | verde | sin cambios |
>
> Casos nuevos de la 008 en total: 19 (17 + 2), todos en rojo.
>
> ### Ambigüedades pendientes
> Ninguna de las cuatro queda abierta con la redacción nueva. Un solo aviso para el verde, no una ambigüedad: `withToken` vuelve a `os.Getenv` en cinco tests que antes tenían el entorno aislado; lo he blindado vaciando `ACORTA_DB` y `ACORTA_BASE_URL`, así que no debería cambiar nada aunque quien ejecute tenga esas variables definidas.

8c:

> ### Qué he cambiado
> Solo `/home/claude/Projects/acorta/web/src/App.test.js` (tres tests nuevos en el `describe('autenticación')`, después del test de «al pulsar Guardar»). Sin tocar `TokenField.vue`, `App.vue` ni nada fuera de `web/`; sin commits ni `npm run build`.
>
> | Criterio | Test | Estado |
> |---|---|---|
> | AUT-11 | `AUT-11: Enter dentro del campo guarda el token igual que el botón Guardar` | falla (`expected null to be 'token-de-prueba-123456'`) |
> | AUT-11 | `AUT-11: tras guardar, el campo pasa a mostrar el valor recortado` | falla (guarda `'  token-con-espacios-1234  '`, espera clave y campo con `token-con-espacios-1234`) |
> | AUT-11 | `AUT-11: el campo del token no contiene ni está dentro de un <form>; el único formulario es el de creación` | pasa con el componente vacío (ya es un `<section>` con botón `type="button"`); es una guarda: comprueba `closest('form')` nulo para el campo y para Guardar, que solo hay un `<form>` y que su `submit` es «Acortar» |
>
> Detalle del test de Enter: dispara `keydown` y `keyup` con `{ key: 'Enter' }` sobre el input (sin `<form>` no hay evento `submit`), así que la implementación puede escuchar cualquiera de los dos.
>
> ### Recuento nuevo
> `npm test` en `web/`: **59 tests, 47 en verde, 12 en rojo**. Los 38 anteriores (WEB-01 a WEB-14) siguen en verde. De los 21 nuevos (AUT-11 a AUT-13), 12 fallan, todos por `AssertionError` (ninguno de sintaxis ni de montaje), y 9 pasan con el componente vacío: los estructurales (type=password con label y posición; sin `<form>`), los triviales sin lógica (campo vacío al cargar sin token) y las guardas de ausencia de cabecera (crear/borrar sin token, listado y Actualizar con y sin token, 400 no mueve el foco).
>
> ### Ambigüedades
> Con las specs actualizadas (AUT-11 de la 008, WEB-04 de la 006) ya no queda ninguna que afecte a lo testeado. Dos notas menores, no bloqueantes:
> - El test de Enter no prescribe `keydown` frente a `keyup`; si la implementación quiere `keydown.enter` (lo habitual en Vue), encaja.
> - Sigue sin especificarse si editar el campo del token tras guardar debe quitar el estado «Token guardado» hasta volver a guardar; no lo testeo, como indicaste para el caso del 401.

Comprobado por el orquestador: en la CLI, 8 funciones en rojo (las 6 nuevas y las 2 de la sinopsis) y 22 en verde; en la web, 12 en rojo y 47 en verde, ninguno de la spec 006 entre los rojos.

**Commits**: `Spec 008 y 006: huecos que destaparon los pasos rojos de la CLI y la web`, `web: tests de la spec 008 (rojo)` y `cmd/acorta: tests de la spec 008 (rojo)`.

## Encargo: paso verde (8b)

> Los tests del paso rojo ya están en un commit. Ahora, el **paso verde** de la fase 8b: implementa en `cmd/acorta/main.go` el token de `serve` hasta que pase `go test ./cmd/acorta/`.
>
> **El método.** Lo mínimo que haga pasar los tests y cumpla las specs 008 y 005; nada que no pidan. Flag `-token` solo en `serve`, con `ACORTA_TOKEN` y la misma precedencia que `-db` y `-base-url` (`pick`). El valor se recorta; vacío → `error: falta el token de API: usa -token o ACORTA_TOKEN`, salida 1; de 1 a 15 caracteres → `error: el token de API debe tener al menos 16 caracteres`, salida 1; las dos comprobaciones antes de abrir la base de datos y sin llamar a `listen`. `server.Handler` recibe el token recortado. La sinopsis y la ayuda de `serve` llevan `[-token TOKEN]` y nombran la variable, sin imprimir nunca el valor; el mensaje de arranque no cambia. **No modifiques ningún `_test.go`**: si crees que un test está mal o contradice la spec, párate y dilo en el informe en vez de cambiarlo. Cuando esté en verde, repasa el código y límpialo sin cambiar el comportamiento, con los tests en verde antes y después.
>
> **Los límites.** Solo `cmd/acorta/`. Solo la biblioteca estándar. Sin commits. No toques `server/`, `web/`, `go.mod` ni `go.sum`.
>
> **Terminado es.** `gofmt -l .` vacío, `go vet ./...` limpio y `go test ./...` en verde en todo el repo.
>
> **El informe.** Ficheros tocados; decisiones que has tomado y por qué; desviaciones de **cualquier** spec, también del plan técnico (`specs/001`), aunque te parezcan justificadas; si has tocado algún test (no deberías); y la salida resumida de `go test ./...`.

## Encargo: paso verde (8c)

> Los tests del paso rojo ya están en un commit. Ahora, el **paso verde** de la fase 8c: implementa en `web/` el token hasta que pase `npm test`.
>
> **El método.** Lo mínimo que haga pasar los tests y cumpla las specs 008 y 006; nada que no pidan. `TokenField.vue` sin `<form>`: Guardar o Enter en el campo guardan el token recortado en `sessionStorage` (`acorta_token`), el campo pasa a mostrar el valor recortado y el estado `Token guardado` va en la región `aria-live`; vacío borra la clave y quita el estado; al montar, lee la clave. `api.js` añade `Authorization: Bearer <token>` solo en crear y borrar, y solo si hay token guardado; listar no la envía nunca. Ante un `401` en crear o borrar, el error se muestra como los demás y el campo del token recibe el foco, sin borrar el token guardado. Sin reglas de negocio en la web (no valida los 16 caracteres). Mobile first y los 44 px de alto de la spec 006 para el campo y el botón; mismo estilo (Tailwind) que el resto. **No modifiques ningún `*.test.js`**: si crees que un test está mal o contradice la spec, párate y dilo en el informe en vez de cambiarlo. Cuando esté en verde, repasa y limpia sin cambiar el comportamiento, con los tests en verde antes y después.
>
> **Los límites.** Solo `web/`. Sin dependencias nuevas en `package.json`. Sin commits. No toques `server/`, `cmd/`, `go.mod` ni `go.sum`. Puedes ejecutar `npm run build` al final para comprobar que compila (escribe en `server/static/dist/`, que no está en Git).
>
> **Terminado es.** `npm test` en verde (59 tests) y `npm run build` sin errores.
>
> **El informe.** Ficheros tocados; decisiones que has tomado y por qué; desviaciones de **cualquier** spec aunque te parezcan justificadas; si has tocado algún test (no deberías); el tamaño del bundle que da el build; y la salida resumida de `npm test`.

## Informe: paso verde (8b)

> ### Ficheros tocados
> - `/home/claude/Projects/acorta/cmd/acorta/main.go` — único fichero mío (+32/−3). **Ningún `_test.go` tocado**; `server/`, `web/`, `go.mod`, `go.sum` intactos; sin commits. `git status` muestra además `bitacora/010-fase-8-autenticacion.md` modificado: no es mío.
>
> ### Qué hay en `main.go`
> - `synopsisServe` → `acorta serve [-addr :8080] [-db RUTA] [-base-url URL] [-token TOKEN]` (y con ella `helpText`, que la reutiliza).
> - Constante `minTokenLen = 16` junto a los demás valores por defecto.
> - `options` gana el campo `token`.
> - `cmdServe`: flag `-token` (solo en `serve`) con ayuda `token de API, al menos 16 caracteres (ACORTA_TOKEN; obligatorio)`; tras `parse` y **antes de `open`**, `apiToken(o.token, stderr)`; `server.Handler(svc, base, server.Static(), token)`.
> - Nueva función `apiToken(flagVal, stderr) (token string, ok bool)`, con el mismo contrato que `open` («en caso de error ya lo ha escrito y devuelve ok=false»): `strings.TrimSpace(pick(flagVal, "ACORTA_TOKEN", ""))`; vacío → `error: falta el token de API: usa -token o ACORTA_TOKEN`; menos de `minTokenLen` → `error: el token de API debe tener al menos 16 caracteres`; devuelve el token recortado. El mensaje de arranque no cambia y el token no se escribe en ninguna salida.
> - Import nuevo: `unicode/utf8` (biblioteca estándar).
>
> ### Decisiones y por qué
> 1. **Precedencia con `pick`, como `-db` y `-base-url`**: flag > `ACORTA_TOKEN` > sin valor por defecto (`""`). Consecuencia: `-token ""` cae a la variable (coherente con «un valor vacío cuenta como no definido»), y `-token "   "` gana a la variable aunque esta sea válida y acaba en AUT-08 (el flag se dio, se recorta y queda vacío). Lo segundo es lo que exige el test «flag solo de espacios» y la redacción nueva de AUT-08.
> 2. **Recorte antes de las dos comprobaciones**, y es el valor recortado el que llega a `Handler` (AUT-09, y el test «16 rodeado de espacios» lo comprueba por el 201).
> 3. **Longitud en caracteres, no en bytes** (`utf8.RuneCountInString`): la spec dice «caracteres», y en este repo ya se cuenta así (`checkAlias` en `link`). Ningún test lo distingue (todos son ASCII); un token de 16 caracteres con acentos pasa, uno de 15 no.
> 4. **Primero `apiToken`, luego `open`**: los tests de AUT-08/AUT-09 lo exigen con la base en un directorio inexistente; es el orden de la spec.
> 5. **Refactor tras el verde**: la comprobación estaba inline en `cmdServe`; la saqué a `apiToken` para que `cmdServe` siga leyéndose como una secuencia (`parse` → token → `open` → `Handler` → `listen`) y para no tener el 16 dos veces (ayuda del flag y mensaje salen de `minTokenLen`). Tests en verde antes y después.
> 6. **No añado el token a `fakeEnv` ni a nada de `environment`**: la inyección existente (`getenv`, `listen`) bastaba.
>
> ### Desviaciones de las specs
> - Ninguna que yo vea respecto a 008 y 005. Una observación sobre `AGENTS.md` regla 7 / spec 001 («los mensajes de error de validación salen solo del paquete `link`»): los dos mensajes del token salen de `cmd/acorta`. Lo considero conforme, porque la spec 008 los define como errores de **configuración de la CLI** (AUT-08, AUT-09), no de validación de enlaces, igual que `ttl: debe ser una duración positiva` o `enlace no encontrado` ya viven en `main.go`; lo digo por si el orquestador quiere matizarlo en la 001.
> - No he actualizado ninguna spec: el código no contradice ninguna.
>
> ### Comprobación manual rápida (sin sockets)
> `go run ./cmd/acorta serve -h` muestra la sinopsis nueva y el flag `-token` con `ACORTA_TOKEN`, sin valor alguno; `ACORTA_TOKEN="   " … serve` → `falta el token`, salida 1; `serve -token "  corto  "` → `al menos 16 caracteres`, salida 1.
>
> ### Suite
> `gofmt -l .` vacío; `go vet ./...` limpio.
> ```
> ok  	acorta/cmd/acorta	0.033s
> ok  	acorta/link	(cached)
> ok  	acorta/server	(cached)
> ok  	acorta/shortener	(cached)
> ok  	acorta/store	(cached)
> ```
> En `cmd/acorta`: 30 funciones de test, todas en verde, incluidos los 19 casos de AUT-07 a AUT-10 y los dos de CLI-12 / flag desconocido que estaban en rojo por la sinopsis.

## Verificación del orquestador (verde 8b)

- `git diff` sobre los `_test.go`: vacío. `gofmt` y `go vet ./...` limpios; `go test -count=1 ./...` en verde en los cinco paquetes.
- Leído el diff de `main.go` (+32/−3) con la spec al lado: `apiToken` antes de `open`, recorte, los dos mensajes exactos, la sinopsis nueva, y `Handler` recibe el token recortado. La observación sobre la regla 7 es correcta y no hace falta matizar la spec: esos mensajes son de configuración de la CLI, como los que ya había.

**Commit**: `cmd/acorta: serve exige el token de API (verde)`.

## Informe: paso verde (8c)

> `npm test` en `web/`: **59 tests, 59 en verde**. `npm run build` sin errores. Ningún `*.test.js` tocado. Solo `web/`, sin commits ni dependencias nuevas.
>
> ### Ficheros tocados
> - `/home/claude/Projects/acorta/web/src/api.js`: el token vive aquí (`TOKEN_KEY = 'acorta_token'`, `loadToken()`, `saveToken()`; vacío → `removeItem`). `authHeaders()` devuelve `{ Authorization: 'Bearer <token>' }` solo si hay token guardado, y se mezcla en las cabeceras de `createLink` y `deleteLink`; `listLinks` no lo toca. El resultado de cada función pasa a `{ ok, status, data, errors }` (`status = 0` en fallo de conexión).
> - `/home/claude/Projects/acorta/web/src/components/TokenField.vue`: `<section>` sin `<form>`; `v-model` sobre `loadToken()` al montar; `save()` con Guardar o `@keydown.enter` recorta, guarda y actualiza el campo con el valor recortado; `saved` controla el estado `Token guardado`; `defineExpose({ focus })`. Campo y botón con `min-h-11` (44 px), `flex gap-2`, mismas clases Tailwind que el resto.
> - `/home/claude/Projects/acorta/web/src/components/LinkForm.vue`: nuevo evento `unauthorized`, emitido cuando `result.status === 401` (el error se muestra como los demás, sin cambios).
> - `/home/claude/Projects/acorta/web/src/App.vue`: `ref` a `TokenField`, `focusToken()`; lo llama con `@unauthorized` del formulario y con `status === 401` al borrar. El token guardado no se toca.
>
> ### Decisiones y por qué
> 1. **El token se lee en `api.js` en el momento de cada petición**, no se pasa por props. Así guardar en la misma sesión surte efecto sin recargar (test «sin recargar») y `listLinks` no puede enviarlo por accidente. No he creado un `token.js`: la spec 006 fija la estructura de `src/` y el token es la credencial del cliente de la API.
> 2. **`status` en el resultado** en vez de un `unauthorized: true`: es lo único que permite distinguir el 401 sin reinterpretar los mensajes (spec 001: la web no reescribe errores).
> 3. **El 401 llega a App por un evento** (`unauthorized`) y App enfoca el campo mediante `defineExpose({ focus })`. Es la vía mínima sin estado compartido ni `document.getElementById`.
> 4. **Sin reglas de negocio**: no se valida la longitud del token (spec 006).
> 5. Editar el campo del token después de guardar no quita el estado `Token guardado` (no se exige nada; mínimo).
>
> ### Desviaciones de las specs (a revisar)
> - **Spec 006, estructura**: `api.js` documentado como `fetch → { ok, data, errors }`; ahora devuelve también `status`. Conviene actualizar esa línea de la 006.
> - **Spec 006, Diseño («los mensajes de estado … están en una región `aria-live="polite"`») vs. tests WEB-04**: con una región `aria-live` persistente (vacía hasta guardar) en `TokenField`, los 3 tests de WEB-04 fallan, porque hacen `w.find('[aria-live="polite"]')` (la **primera** región del documento) y esperan que sea la del formulario; `TokenField` va encima. Lo he comprobado: 3 rojos por `expected '' to contain …`. Como no puedo tocar los tests, la región del token solo existe mientras hay estado (`<p v-if="saved" aria-live="polite">Token guardado</p>`). Cumple la letra de AUT-11 y pasa todos los tests, pero en accesibilidad es un compromiso: un nodo que nace ya con `aria-live` puede no anunciarse (la práctica recomendada es que la región exista antes de que cambie su contenido). Recomendación: cambiar WEB-04 a `findAll('[aria-live="polite"]').find((r) => r.text().includes(e))`, como ya hace el test de AUT-11, y entonces volver a la región persistente. Lo dejo anotado en un comentario del componente.
> - Ninguna otra: la 008 (AUT-11 a AUT-13) y la 006 se cumplen tal cual.
>
> ### Tests tocados
> Ninguno.
>
> ### Bundle (`npm run build`, escribe en `server/static/dist/`, fuera de Git)
> ```
> ../server/static/dist/index.html                  0.39 kB │ gzip:  0.26 kB
> ../server/static/dist/assets/index-Ca7beaVs.css  14.69 kB │ gzip:  3.89 kB
> ../server/static/dist/assets/index-B_dGrM_V.js   75.50 kB │ gzip: 29.04 kB
> ✓ built in 79ms
> ```
>
> ### `npm test` (antes y después de la limpieza, mismo resultado)
> ```
>  Test Files  1 passed (1)
>       Tests  59 passed (59)
>      Duration  554ms
> ```

## Verificación del orquestador (verde 8c)

- `git diff` sobre los `*.test.js`: vacío. `npm test`: 59 en verde. `npm run build` sin errores.
- Las dos desviaciones que declara son reales. La primera se cierra en la spec 006 en este mismo commit (`api.js` devuelve también `status`). La segunda es, otra vez, un test que daba por hecho algo que la spec nueva cambia: WEB-04 buscaba la primera región `aria-live` del documento. El agente no tocó el test y dejó la región del token sin persistir, con un comentario, en vez de hacer trampa: es exactamente lo que pide la regla 3. El orquestador corrigió los tres tests en un commit propio (`eab2569`: buscan la región del propio formulario; comprobado que pasan con el código en rojo y en verde, y que los 12 de la 008 seguían en rojo con el código en rojo) y pidió volver a la región persistente.
- Un tropiezo propio: la primera corrección del test buscaba la región *dentro* del `<form>`, y la de `LinkForm` está al lado, fuera de él. Lo cazaron los tres tests al pasar a rojo antes de confirmar; se corrigió con `--amend` antes de seguir.

**Encargo final del verde (8c)**:

> Bien visto lo de la región. Tenías razón en que los tests de WEB-04 daban por hecho una sola región `aria-live`: los he corregido yo en un commit aparte (`eab2569`, buscan ahora la región del propio formulario), así que ya no hay motivo para el compromiso. Última tarea del **paso verde**, en `web/src/components/TokenField.vue`: haz que la región `aria-live="polite"` del token exista siempre (sin `v-if`), vacía hasta que haya estado y con `Token guardado` cuando lo haya (como hace `LinkForm` con su `empty:hidden`), y quita el comentario que explicaba el compromiso. Nada más. Terminado es `npm test` en verde (59) y `npm run build` sin errores. No toques ningún test. Informe corto: qué has cambiado y la salida resumida de `npm test`.

Informe final del agente:

> **Cambio** (solo `/home/claude/Projects/acorta/web/src/components/TokenField.vue`): la región `<div aria-live="polite" class="empty:hidden mt-2">` existe siempre, como la de `LinkForm`; dentro, `<p v-if="saved">Token guardado</p>` aparece solo cuando hay estado. Quitado el comentario que explicaba el compromiso. Ningún test tocado; sobre `eab2569` los tests de WEB-04 ya no chocan.
>
> **`npm test`** en `web/`: 59 passed (59). **`npm run build`**: sin errores (`index-D4ysEhmi.js` 75.54 kB, gzip 29.04 kB; CSS 14.69 kB, gzip 3.89 kB).

**Commit**: `web: token de API en la interfaz (verde)`.

## Verificación de la fase 8 (el orquestador, sin agentes)

Con el binario compilado (`make build`) y una base de datos nueva:

- `serve` sin token: `error: falta el token de API: usa -token o ACORTA_TOKEN`, salida 1. Con `-token corto`: `error: el token de API debe tener al menos 16 caracteres`, salida 1. Con un token válido arranca y el mensaje de arranque no lo contiene; el log tampoco.
- `curl`: `POST /api/links` sin token → 401, `WWW-Authenticate: Bearer`, `falta el token de API`; con token malo → 401, `token de API incorrecto`; sin token y cuerpo inválido → 401 (no 400); con token bueno → 201. `GET /api/links` y `GET /promo` sin token → 200 y 302. `DELETE` sin token, exista o no el código → 401; `PUT /api/links` sin token → 405; `DELETE` con `bearer` en minúsculas y el token bueno → 204, y después `GET /promo` → 404. `acorta add` por CLI sigue funcionando sin token.
- Navegador real (Chromium sin interfaz) a 390 y a 1280 px: crear sin token muestra `falta el token de API` tal cual y el foco va al campo del token; guardar un token con espacios lo deja recortado en el campo y en `sessionStorage`, con `Token guardado`; crear y borrar funcionan con el token; tras recargar, el token sigue; un token equivocado muestra `token de API incorrecto` y no borra el guardado. Sin scroll horizontal. Campo y botón de 44 px.

## Estado al cerrar la fase 8

| | |
|---|---|
| Criterios en las specs | 113 (13 nuevos, AUT-01 a AUT-13) |
| Tests Go | 307 casos hoja (eran 246) |
| Tests de la interfaz | 59 (eran 38) |
| Commits de la fase | 10: 3 de spec, 3 en rojo, 3 en verde y 1 arreglo de tests |

## Lo que enseñó el cambio

- **Un cambio de requisitos empieza en la spec y la spec dice qué arrastra.** Una spec nueva (008) y cinco líneas en otras cuatro (000, 004, 005, 006, 007): el contrato de `Handler`, la sinopsis de `serve`, la tabla de configuración, el 401 en WEB-04, la fase nueva. Sin tocar `link`, `store` ni `shortener`.
- **El paso rojo volvió a ser un revisor de specs**: quince avisos entre los tres agentes (cuatro del servidor, cuatro de la CLI, siete de la web), doce cerrados en la spec antes de implementar (espacios en la cabecera, quién envía el token en los tests viejos, token de solo espacios, el orden token → base de datos, Enter en el campo, la región `aria-live`, el 401 en WEB-04…) y tres que se quedaron fuera a propósito.
- **Los tests viejos también cambian, pero en el rojo.** Tres veces un agente avisó de que los tests de la v1 iban a romperse en el verde (los de la API sin cabecera, los de `serve` sin token, la sinopsis vieja). Las tres veces se arregló en el paso rojo, con la spec actualizada, para que ningún commit verde tocara un test.
- **La regla 3 sigue funcionando.** El agente de la web encontró tres tests de WEB-04 que daban por hecho una sola región `aria-live`; no los tocó, dejó un compromiso comentado y lo dijo. El orquestador los corrigió en un commit propio, y al hacerlo se equivocó una vez: los tests lo cazaron antes de confirmar.
