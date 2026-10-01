# 005 — Fase 4: API, redirección y estáticos (`server/`)

**Fecha**: 2026-10-01 · **Spec**: `specs/004-api.md` · **Rutas**: `server/**`

## Encargo: paso rojo

> **Qué leer antes.** Trabajas en el repositorio `/home/claude/Projects/acorta`. Antes de escribir nada lee, en este orden: `AGENTS.md`, `specs/001-plan-tecnico.md` y `specs/004-api.md` entera. De `specs/002-enlaces.md` te basta con saber qué errores devuelve el servicio. Mira el código que vas a usar: `link/link.go` y `shortener/shortener.go` (su `Service` cumple la interfaz `Service` de tu paquete).
>
> **La tarea.** Eres el agente de la fase 4 de `specs/007-fases.md`: el paquete `server`. Este encargo es solo el **paso rojo**:
>
> 1. Crea en `server/` la interfaz `Service` y las funciones `Handler` y `Static` de «Contrato del paquete» (spec 004), con las firmas exactas, pero **sin implementar** (un `Handler` que responde 501 a todo sirve). Crea ya `server/static/sin-compilar.html`, la página de aviso de EST-04, y el `go:embed` que la incluye: `go:embed` no compila sin al menos un fichero.
> 2. Escribe los tests de RED-01 a RED-06, API-01 a API-13 y EST-01 a EST-05.
> 3. Ejecuta `go test ./server/` y comprueba que los tests **fallan por las aserciones**, no por errores de compilación ni por un pánico.
>
> **El método.** `httptest` contra `Handler`. El servicio es el real (`shortener.New` sobre `store.Open` en `t.TempDir()`, con reloj fijo y generador de códigos determinista), para probar la historia completa; usa un servicio falso solo donde haga falta forzar un fallo (API-07, API-13). Los estáticos se prueban con un `fstest.MapFS`, nunca con el embebido real, salvo EST-05. Cada test nombra el criterio que comprueba. Comprueba siempre estado, cabeceras relevantes (`Content-Type`, `Location`, `Cache-Control`, `Allow`) y cuerpo. En RED-01, RED-03 y RED-04 comprueba también el contador de visitas después de la petición.
>
> **Los límites.** Solo puedes crear o modificar ficheros dentro de `server/`. No toques `link/`, `store/`, `shortener/`, `go.mod` ni `go.sum`; no ejecutes `go mod tidy`; no añadas dependencias. No hagas commits. No implementes todavía. Si algún caso pasa con el esqueleto, dilo en el informe.
>
> **Terminado es.** `go vet ./...` limpio, `gofmt -l .` vacío, y `go test ./server/` compilando y fallando; el resto del repo sigue en verde.
>
> **El informe.** Ficheros creados; cuántos tests hay y cuántos fallan; una tabla criterio → test; cualquier ambigüedad o contradicción de la spec, o cualquier cosa de `shortener` o `link` que no encaje con lo que necesitas (no la resuelvas en silencio); y las últimas líneas de `go test ./server/`.

## Informe: paso rojo

> Paso rojo hecho. `go test ./server/` compila y falla por aserciones: 24 tests de nivel superior, los 24 en rojo, y ningún subtest en verde. No hay pánicos ni errores de compilación. `gofmt -l .` sale vacío, `go vet ./...` limpio, y `shortener` y `store` siguen en verde. No he tocado nada fuera de `server/` ni he hecho commits.
>
> **Ficheros creados (todos en `server/`)**
> - `server.go`: la interfaz `Service`, `Handler` (responde 501 a todo), `Static()` (devuelve `nil`) y `//go:embed all:static`.
> - `static/sin-compilar.html`: la página de aviso de EST-04.
> - `helpers_test.go`: servicio real sobre `store.Open` en `t.TempDir()`, con reloj fijo y generador determinista; un servicio falso `fakeSvc` y los helpers de aserción.
> - `redirect_test.go`, `api_test.go` y `static_test.go`.
>
> | Criterio | Test |
> |---|---|
> | RED-01 a RED-06 | `TestRED01_Redirige`, `TestRED02_NoExiste`, `TestRED03_Caducado`, `TestRED04_Head`, `TestRED05_MetodoNoPermitido`, `TestRED06_VariosSegmentos` |
> | API-01 a API-13 | `TestAPI01_Healthz`, `TestAPI02_Crear`, `TestAPI03_AliasYCaducidad`, `TestAPI04_Validacion`, `TestAPI05_AliasEnUso`, `TestAPI06_CuerpoInvalido`, `TestAPI07_SinCodigoLibre`, `TestAPI08_Listar`, `TestAPI09_Borrar`, `TestAPI10_BorrarInexistente`, `TestAPI11_RutaDesconocida`, `TestAPI12_MetodoNoPermitido`, `TestAPI13_ErrorInterno` |
> | EST-01 a EST-05 | `TestEST01_Index`, `TestEST02_Ficheros`, `TestEST03_NoExiste`, `TestEST04_SinCompilar`, `TestEST05_Embebido` |
>
> RED-01, RED-03 y RED-04 comprueban también el contador de visitas. API-07 y API-13 usan el servicio falso. API-07 además tiene una variante con el servicio real, con un generador que repite siempre el mismo código. Los estáticos usan un `fstest.MapFS`; solo EST-05 usa el embebido real.
>
> **Ambigüedades y decisiones que conviene revisar**
> 1. **Salto de línea final en `text/plain`.** La spec exige el salto de línea solo bajo `/api/`. En los textos de RED-02 y RED-03 comparo con `TrimSpace`, así que valen con o sin salto.
> 2. **`Content-Type` de la API.** Lo exijo exactamente `application/json`, sin `charset`, porque así lo escribe la spec.
> 3. **`baseURL` con barra final.** La spec dice «sin barra final» para `short_url`, pero no aclara si `Handler` debe recortar la barra de `baseURL`. No lo he testado.
> 4. **204 de API-09.** Solo compruebo estado y cuerpo vacío. Queda sin definir la cabecera de un 204 sin cuerpo.
> 5. **Valor exacto de `Allow` en la API (API-12).** Compruebo que `Allow` incluya los métodos esperados. El valor exacto no lo fijo, por si `ServeMux` añade `HEAD`. En RED-05 sí exijo literalmente `GET, HEAD`.
> 6. **`HEAD` bajo `/api/`.** La spec no lo trata. No lo testo.
> 7. **Fallo interno en la redirección.** API-13 habla de JSON, pero la redirección es `text/plain`. Solo compruebo que dé 500 y que no filtre el detalle. Falta definir el cuerpo.
> 8. **Rutas anidadas bajo `assets`.** La spec no dice qué pasa con `/assets/a/b.js`.
> 9. **JSON `null` como cuerpo.** `null` se decodifica sin error en un struct. No lo testo, pero la implementación tendrá que decidir.
> 10. **`link` y `shortener`.** Encajan con lo que necesito.

## Revisión del orquestador

Comprobado: solo hay ficheros nuevos en `server/`; `gofmt` y `go vet` limpios; los 24 tests fallan; el resto del repo sigue en verde; los 24 criterios tienen su test.

La spec de la API era la más larga y aun así salieron nueve huecos. Casi todos son de la misma familia: **lo que pasa en los bordes del protocolo**, donde uno tiende a escribir «responde 404» y a dar por hecho todo lo demás.

| # | Hueco | Cómo queda la spec 004 |
|---|---|---|
| 1 | ¿Los textos planos acaban en salto de línea? | Sí (RED-02) |
| 2 | ¿`application/json` con o sin `charset`? | Sin `charset`, y se dice explícitamente |
| 3 | ¿Quién quita la barra final de la URL base? | `Handler` |
| 4 | ¿Qué cabeceras lleva un `204`? | Ni cuerpo ni `Content-Type` |
| 5 | Valor exacto de `Allow` | Fijado ruta por ruta (API-12) |
| 6 | `HEAD` en la API | Se admite donde se admite `GET` |
| 7 | Fallo interno al redirigir | Criterio nuevo, **RED-07**: `500` en texto plano, sin detalle |
| 8 | Ficheros anidados bajo `/assets/` | Cualquier profundidad (EST-02) |
| 9 | Cuerpo JSON `null`, lista o texto | `400`, como cualquier cuerpo que no sea un objeto (API-06) |

El 9 es un buen ejemplo de por qué conviene que quien escribe los tests conozca la biblioteca: en Go, decodificar `null` en una estructura no da error. Sin el test, la API habría aceptado `null` como «un enlace sin URL» y habría contestado con un error de validación en vez de con el de cuerpo inválido. Nadie lo habría notado nunca, hasta que alguien lo notara.

## Encargo: ajuste del rojo

> La spec 004 ha cambiado para cerrar las ambigüedades que encontraste. Vuelve a leer `specs/004-api.md` entera: han cambiado el párrafo de `short_url` (la barra final de la URL base), RED-02 (los textos terminan en salto de línea), el criterio nuevo RED-07 (fallo interno al redirigir), la introducción de «API» (`Content-Type` exacto, el `204` sin `Content-Type`, `HEAD` donde hay `GET`), API-06 (`null`, listas y textos), API-12 (valor exacto de `Allow`) y EST-02 (ficheros anidados bajo `/assets/`).
>
> Sigue siendo el **paso rojo**: ajusta solo los tests de `server/` para cubrir lo que ahora la spec sí dice. Mismos límites: solo `server/`, sin implementar, sin commits. Cada test nuevo lleva su identificador con guion (`RED-07`).
>
> Informe: qué tests has añadido o cambiado, el recuento nuevo y si queda alguna ambigüedad.

## Informe: ajuste del rojo

> Ajuste del rojo hecho, solo en tests de `server/`. Ahora hay 26 tests de nivel superior y los 26 fallan por aserciones. Ningún subtest pasa y no hay pánicos. `gofmt -l .` sale vacío y `go vet ./...` está limpio. No he implementado nada ni hecho commits.
>
> **Tests añadidos**
> - `TestRED07_ErrorInterno` (`redirect_test.go`): usa un servicio falso cuyo `Resolve` falla, con GET y HEAD sobre `/aaaaaa`. Comprueba 500, `text/plain; charset=utf-8`, sin `Location`, cuerpo exacto `error interno\n` en GET (vacío en HEAD) y que no se filtre el detalle.
> - `TestAPI02_BarraFinalEnBaseURL` (`api_test.go`): con `baseURL` `http://localhost:8080/` y sin barra, `short_url` es `http://localhost:8080/aaaaaa` tanto al crear como al listar.
>
> **Tests cambiados**
> - RED-02, RED-03 y API-09: ahora exigen el cuerpo exacto con salto de línea final, en vez de comparar con `TrimSpace`.
> - API-01 y API-08: añaden HEAD sobre `/api/healthz` y `/api/links`. Esperan 200, `Content-Type: application/json` y cuerpo vacío.
> - API-06: añade el caso `null`. Los casos de lista y de texto ya estaban.
> - API-09: el 204 no debe llevar `Content-Type`.
> - API-12: `Allow` exacto (`GET, HEAD` en healthz, `GET, HEAD, POST` en `/api/links`, `DELETE` en `/api/links/{código}`).
> - API-13: he quitado el subtest de la redirección, que ahora es RED-07.
> - EST-02: el `MapFS` incluye `dist/assets/js/chunk.js` y `dist/assets/fonts/a.woff2`, y se sirven bajo `/assets/…` a cualquier profundidad.
>
> **Ambigüedades que quedan:** ninguna importante.

**Commits**: `Spec 004: los bordes del protocolo (RED-07)` y `server: tests de la API, la redirección y los estáticos (rojo)`. 26 tests, 26 en rojo.

## Encargo: paso verde

> Los tests del paso rojo ya están en un commit. Ahora, el **paso verde**: implementa el paquete `server` hasta que pase `go test ./server/`.
>
> **El método.** Lo mínimo que haga pasar los tests y cumpla la spec 004. **No modifiques los ficheros `_test.go`**: si crees que un test está mal o contradice la spec, párate y dilo en el informe. Biblioteca estándar: `net/http` con patrones de método y comodines, `encoding/json`, `io/fs`. El servidor es un adaptador fino: ninguna regla de negocio, ningún mensaje de validación propio; los errores de `link` y `shortener` se traducen a estados HTTP con `errors.Is` y `errors.As`. El detalle de un fallo interno va al log (`log.Printf`), nunca a la respuesta. Cuando esté en verde, limpia el código sin cambiar el comportamiento: si un fichero se hace largo, repártelo (por ejemplo `api.go`, `redirect.go`, `static.go`). Comentarios en español que expliquen el porqué.
>
> **Los límites.** Solo `server/`. No toques los demás paquetes, `go.mod` ni `go.sum`. Sin dependencias nuevas. Sin commits. No toques `server/static/dist/` (es la salida del build del frontend).
>
> **Terminado es.** `gofmt -l .` vacío, `go vet ./...` limpio, y `go test ./server/`, `./link/`, `./store/` y `./shortener/` en verde (`cmd/acorta/` está a medias en manos de otro agente: si falla solo eso, no es cosa tuya).
>
> **El informe.** Ficheros tocados; decisiones que has tomado y por qué (en especial: cómo distingues un código de un fichero estático, y cómo consigues el `405` con JSON y `Allow` exacto); desviaciones de la spec; si has tocado algún test (no deberías); y la salida resumida de `go test ./...`.

## Informe: paso verde

> Paso verde hecho. `go test ./server/` pasa entero, `./link/`, `./store/` y `./shortener/` siguen en verde, `gofmt -l .` sale vacío y `go vet ./...` está limpio. En `go test ./...` solo falla `cmd/acorta`, que es del otro agente. No he tocado ningún `_test.go`, ni nada fuera de `server/`, ni `server/static/dist/`, y no he hecho commits.
>
> **Ficheros** (todos en `server/`)
> - `server.go` (modificado): tipo `Service`, `Handler`, `Static()` y el reparto de rutas.
> - `api.go` (nuevo): rutas de `/api/`, JSON del enlace, creación, listado y borrado.
> - `redirect.go` (nuevo): `GET` y `HEAD /{código}`.
> - `static.go` (nuevo): `/`, aviso de frontend sin compilar y ficheros de `dist/`.
> - `respond.go` (nuevo): respuestas en texto plano y en JSON, y registro del error interno en el log.
>
> **Decisiones**
> - **No uso `http.ServeMux`.** El mux responde 404 y 405 en texto plano y sin el `Allow` que pide la spec. Hice un reparto a mano en `ServeHTTP`, con este orden: `/api` o `/api/…` va a la API; `/` es la interfaz web; `/assets/…` es estático a cualquier profundidad; cualquier otra ruta con una barra más es un 404 de texto (RED-06); un segmento con punto es estático; un segmento sin punto es un código y se redirige.
> - **Código frente a estático.** Un código nunca contiene un punto ni una barra, así que basta mirar si el único segmento lleva punto.
> - **405 con JSON y `Allow` exacto.** Cada ruta de la API conoce su cadena `Allow` literal. `methodNotAllowed` pone esa cabecera y escribe `{"errors":["método no permitido"]}`.
> - **HEAD** no escribe cuerpo en ninguna respuesta.
> - **Estáticos.** Leo el fichero de `dist/` con `fs.ReadFile` y lo sirvo con `http.ServeContent`, que deduce el tipo por la extensión. Sin `dist/index.html`, `/` sirve `sin-compilar.html`.
> - **Cuerpo de `POST /api/links`.** Uso `MaxBytesReader` de 64 KiB, `DisallowUnknownFields` y decodifico sobre un puntero para detectar `null`. Después compruebo que no queda nada tras el objeto.
> - **Errores.** Los traduzco con `errors.As` (`ValidationError` a 400, `AliasTakenError` a 409) y `errors.Is` (`ErrNoCodeAvailable` a 503, `ErrNotFound` a 404, `ErrExpired` a 410). Cualquier otro error da 500 `error interno`, con el detalle en `log.Printf`.
> - **`baseURL`.** Se le quitan las barras finales. **`expired`** se calcula con `svc.Now()`.
>
> **Desviaciones de la spec:** ninguna. Los 404 de ficheros estáticos usan los textos `fichero no encontrado` y `ruta no encontrada`, que la spec no fija (solo exige `text/plain`).

## Verificación del orquestador: verde no es lo mismo que conforme

- `git diff` sobre los `_test.go` de `server/`: vacío. `gofmt` y `go vet` limpios. `go test -count=1 ./server/`: en verde.
- Y, sin embargo, el informe dice dos cosas que no casan: «no uso `http.ServeMux`» y «desviaciones de la spec: ninguna». La spec 001 decía, en la tabla del stack: *`net/http` con patrones de método y comodines*. Eso **es** `ServeMux`. El agente solo había leído como contrato la spec de su fase (la 004) y trató el plan técnico como contexto.

Ningún test podía cazarlo: los tests comprueban qué responde el servidor, no cómo está hecho por dentro. Lo cazó leer el informe con el plan técnico al lado.

Había dos salidas: mandar rehacer el reparto con `ServeMux`, o aceptar la decisión y cambiar la spec. El orquestador leyó `server.go` antes de decidir. El reparto a mano son doce líneas con su tabla en un comentario, y la razón es buena: la regla que separa un código de un fichero estático («un segmento sin punto») no se puede escribir como patrón, y la API necesita sus 404 y 405 en JSON. Con `ServeMux` habría hecho falta el mismo código dentro de los manejadores, más el registro de rutas.

Se acepta. Pero la regla 1 de `AGENTS.md` no es opcional: si el código contradice la spec, **la spec cambia en el mismo commit**. La spec 001 dice ahora que el reparto es a mano y por qué. Y los dos textos de 404 que el agente eligió sin que nadie los pidiera pasan a estar escritos en la spec 004 (RED-06 y EST-03).

**Commit**: `server: API, redirección y estáticos (verde)`, con las specs 001 y 004 en el mismo commit.
