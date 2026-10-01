# 004 — Fase 3: los casos de uso (`shortener/`)

**Fecha**: 2026-10-01 · **Spec**: `specs/002-enlaces.md` · **Rutas**: `shortener/**`

## Encargo: paso rojo

> **Qué leer antes.** Trabajas en el repositorio `/home/claude/Projects/acorta`. Antes de escribir nada lee, en este orden: `AGENTS.md`, `specs/001-plan-tecnico.md` y `specs/002-enlaces.md` entera. Mira también el código que ya existe y que vas a usar: `link/link.go` (validación, códigos, errores) y `store/store.go` (el almacén SQLite, que cumple tu interfaz `Store`).
>
> **La tarea.** Eres el agente de la fase 3 de `specs/007-fases.md`: el paquete `shortener`, que une el dominio y el almacén en los casos de uso crear, resolver, listar y borrar. Este encargo es solo el **paso rojo**:
>
> 1. Crea en `shortener/` la interfaz `Store`, el tipo `Service` y sus funciones según el bloque `package shortener` de «Contrato de los paquetes» (spec 002), con las firmas exactas, pero **sin implementar**: lo justo para que los tests compilen.
> 2. Escribe los tests de los criterios que son responsabilidad del servicio: COD-02, COD-03, COD-04 (que el generador y el reloj inyectados se usan, y que con `nil` hay valores por defecto que funcionan), ALI-01 y ALI-06, CAD-06, RES-01 a RES-04, VIS-01 a VIS-04, LIS-01 a LIS-03 y BOR-01 a BOR-03. Añade los que comprueban que `Create` devuelve `*link.ValidationError` con los mensajes de `link.Validate` cuando la entrada no es válida y que entonces no guarda nada, y que `CreatedAt` es el instante del reloj en UTC y sin fracciones de segundo.
> 3. Ejecuta `go test ./shortener/` y comprueba que los tests **fallan por las aserciones**, no por errores de compilación ni por un pánico.
>
> **El método.** Los tests usan el almacén real (`store.Open` sobre `t.TempDir()`), un reloj fijo que se puede adelantar y un generador de códigos determinista (una lista de códigos preparada). No vuelvas a probar las reglas de validación una a una: ya están probadas en `link`. Cada test nombra el criterio que comprueba. VIS-02 se prueba con 50 goroutines de verdad y debe pasar con `go test -race`. Para los fallos del almacén (un error que no es de negocio) puedes usar un `Store` falso mínimo, solo en esos casos.
>
> **Los límites.** Solo puedes crear o modificar ficheros dentro de `shortener/`. No toques `link/`, `store/`, `go.mod` ni `go.sum`; no ejecutes `go mod tidy`; no añadas dependencias. No hagas commits. No implementes todavía. Si algún caso pasa con las funciones vacías, dilo en el informe.
>
> **Terminado es.** `go vet ./...` limpio, `gofmt -l .` vacío, y `go test ./shortener/` compilando y fallando; el resto del repo sigue en verde.
>
> **El informe.** Ficheros creados; cuántos tests hay y cuántos fallan; una tabla criterio → test; cualquier ambigüedad o contradicción de la spec, o cualquier cosa de `link` o `store` que no encaje con lo que necesitas (no la resuelvas en silencio); y las últimas líneas de `go test ./shortener/`.

## Informe: paso rojo

> Paso rojo hecho. `go test ./shortener/` compila y falla por aserciones, sin pánicos. `go vet ./...` y `gofmt -l .` salen limpios, y `link/` y `store/` siguen en verde. No he tocado nada fuera de `shortener/` ni hecho commits.
>
> **Ficheros creados**
> - `shortener/shortener.go`: interfaz `Store`, `Service` y `New`/`Create`/`Resolve`/`List`/`Delete`/`Now` con las firmas exactas de la spec. Cada función devuelve el valor cero (marcadas `TODO(paso verde)`); `New` solo guarda lo recibido, sin los valores por defecto.
> - `shortener/shortener_test.go`: almacén SQLite real en `t.TempDir()`, reloj fijo que se adelanta y generador de códigos con lista preparada. Un `Store` falso (`brokenStore`) solo para los fallos del almacén.
>
> **Resultado**
> - Hay 27 tests de nivel superior (más subtests) y fallan los 27.
> - Ningún test pasa con las funciones vacías, ni siquiera como subtest.
> - `go test -race` no da ningún `DATA RACE` en el test de VIS-02, pero esa prueba solo cobra sentido con la implementación.
>
> | Criterio | Test |
> |---|---|
> | COD-02 | `TestCOD02_CodigoOcupadoOReservadoSeDescartaYSeReintenta`: ocupado y reservado se descartan; el 5.º intento aún vale; 5 fallos dan `ErrNoCodeAvailable` con exactamente 5 llamadas y sin guardar nada; si el generador falla se devuelve su error |
> | COD-03 | `TestCOD03_LaMismaURLDaDosCodigosDistintos` |
> | COD-04 | `TestCOD04_GeneradorYRelojInyectadosSeUsan` y `TestCOD04_ConNilHayValoresPorDefectoQueFuncionan` (6 caracteres de `[a-z0-9]`, reloj real, dos códigos distintos) |
> | ALI-01 | `TestALI01_ElAliasEsElCodigo` (además comprueba que no se llama al generador) |
> | ALI-06 | `TestALI06_AliasEnUsoDaAliasTakenError`: enlace vigente y caducado, mensaje exacto, nada guardado; y alias ocupado con URL inválida da `ValidationError` |
> | Validación en Create | `TestCreate_EntradaInvalidaDaValidationErrorYNoGuardaNada`: los mensajes son iguales a los de `link.Validate` (1 y 3 errores, caducidad pasada) y no se guarda nada |
> | CreatedAt | `TestCreate_CreatedAtEsElRelojEnUTCSinFracciones`: reloj en UTC+2 con nanosegundos |
> | ExpiresAt guardado | `TestCreate_GuardaLaCaducidadValidada` |
> | CAD-06 | `TestCAD06_UnEnlaceCaducadoSigueEnElListadoYSuCodigoOcupado` |
> | RES-01 a RES-04 | `TestRES01_…`, `TestRES02_…`, `TestRES03_…` (incluye el límite exacto de CAD-05), `TestRES04_…` |
> | VIS-01 | `TestVIS01_UnEnlaceNuevoTieneCeroVisitas` |
> | VIS-02 | `TestVIS02_CincuentaResolucionesSimultaneasSuman50` (50 goroutines con barrera) y `TestVIS02_CadaResolucionConExitoSumaUna` |
> | VIS-03 | `TestVIS03_UnaResolucionFallidaNoSumaNada` |
> | VIS-04 | `TestVIS04_ResolverSinContarNoSumaYDevuelveLoMismo` (cubre RES-01 a RES-04 sin contar) |
> | LIS-01 a LIS-03 | `TestLIS01_…`, `TestLIS02_…`, `TestLIS03_…` |
> | BOR-01 a BOR-03 | `TestBOR01_…`, `TestBOR02_…`, `TestBOR03_…` |
> | Fallos del almacén | `TestFalloDelAlmacenSePropagaSinTraducirse` |
>
> **Ambigüedades de la spec, sin resolver por mi cuenta**
> 1. Si el generador inyectado devuelve un error, la spec no dice qué hace `Create`. COD-02 solo habla de colisiones y reservadas. Mi test exige que devuelva un error que no sea `ErrNoCodeAvailable` y que no guarde nada. No exijo que se envuelva con `%w`.
> 2. Si `AddVisit` falla después de un `Get` correcto en `Resolve`, no está especificado qué se devuelve: ¿error, o la URL igualmente? No lo he testeado.
> 3. Lo mismo para la carrera Get/AddVisit con un borrado entre medias, que daría `ErrNotFound` desde el almacén. Tampoco lo he testeado.
> 4. Un error de `Insert` que no sea `ErrCodeTaken` no se reintenta (una sola inserción) y se propaga tal cual. Es una decisión mía que encaja con «cualquier otro error es un fallo del almacén», pero no está escrita en COD-02.
> 5. LIS-02 pide «si está caducado» y `[]link.Link` no tiene ese campo. Lo he interpretado como `l.Expired(svc.Now())`, que es para lo que existe `Now()`.
> 6. Orden de comprobaciones en `Resolve`: con un código caducado y `countVisit=false` se devuelve `ErrExpired`. Lo deduzco de VIS-04, que dice «devuelve lo mismo que RES-01 a RES-04».
>
> **Sobre `link` y `store`**
> No hay nada que no encaje. `store.Open` y `*store.Store` cumplen `shortener.Store`, y `List` ordena por `id DESC`, como pide LIS-01.

## Revisión del orquestador

Comprobado: solo hay ficheros nuevos en `shortener/`; `gofmt` y `go vet` limpios; los 27 tests fallan; `link` y `store` siguen en verde; y todos los criterios de la spec 002 tienen ya algún test entre `link` y `shortener`.

Las ambigüedades 2 y 3 son de las buenas: preguntan qué pasa cuando algo falla **a mitad** de una operación. La spec solo describía los caminos limpios.

| # | Hueco | Cómo queda la spec 002 |
|---|---|---|
| 1 y 4 | ¿Qué pasa si falla el generador, o el almacén da otro error al crear? | COD-02: solo se reintenta por código ocupado o reservado; cualquier otro error se devuelve envuelto, sin más intentos |
| 2 | ¿Se redirige aunque no se haya podido contar la visita? | No. Criterio nuevo, **RES-05**: sin visita no hay URL |
| 3 | ¿Y si borran el enlace justo entre encontrarlo y contarlo? | RES-05: `ErrNotFound` |
| 5 | ¿Dónde está «caducado» en el listado? | LIS-02: no se guarda; lo calcula quien muestra la lista con el reloj del servicio |
| 6 | Orden de comprobaciones al resolver sin contar | Ya lo decía VIS-04; se queda como está |

## Encargo: ajuste del rojo

> La spec 002 ha cambiado para cerrar las ambigüedades que encontraste. Vuelve a leer en `specs/002-enlaces.md` COD-02 (solo se reintenta por código ocupado o reservado; el error del generador o cualquier otro error del almacén se devuelven envueltos, sin más intentos), el criterio nuevo RES-05 (si no se puede sumar la visita no se devuelve la URL; si el enlace se ha borrado entre medias, `ErrNotFound`) y LIS-02.
>
> Sigue siendo el **paso rojo**: ajusta solo `shortener/shortener_test.go` para cubrir lo que ahora la spec sí dice: que el error del generador y el del almacén llegan envueltos (`errors.Is`) y sin reintentos, y RES-05 en sus dos variantes (con un `Store` falso cuyo `AddVisit` falla, y con uno cuyo `AddVisit` devuelve `link.ErrNotFound` tras un `Get` correcto). Mismos límites: solo `shortener/`, sin implementar, sin commits.
>
> Informe: qué tests has añadido o cambiado, el recuento nuevo y si queda alguna ambigüedad.

## Informe: ajuste del rojo

> Ajuste del rojo hecho, solo en `shortener/shortener_test.go`, sin implementar y sin commits. `gofmt -l .` sale vacío y `go vet ./...` limpio. `go test ./shortener/` compila y falla por aserciones (27 de 27 tests de nivel superior, 0 pánicos). `link/` y `store/` siguen en verde.
>
> **Corrección a mi informe anterior**
> Dije 27 tests y eran 26. Ahora hay 27 con el nuevo de RES-05.
>
> **Cambios por la spec nueva (COD-02, RES-05, LIS-02)**
> - **COD-02, error del generador:** el generador falso admite ahora un error propio. El subtest antiguo se ha reemplazado por dos: si el generador falla a la primera, se comprueba con `errors.Is` el error propio, que no es `ErrNoCodeAvailable`, que hay 1 sola llamada y que no se guarda nada; si falla tras un código reservado descartado, se comprueba que hay 2 llamadas y no más.
> - **COD-02, error del almacén:** `TestFalloDelAlmacenSePropagaSinTraducirse` añade que, con un `Insert` roto, el generador se llama exactamente 1 vez.
> - **RES-05 (test nuevo):** `TestRES05_SiNoSePuedeSumarLaVisitaNoSeDevuelveLaURL`, con un `Store` falso cuyo `Get` va bien. Tres subtests: `AddVisit` falla (el error es el del almacén y la URL llega vacía); `AddVisit` devuelve `link.ErrNotFound` tras un `Get` correcto (el error es `ErrNotFound` y la URL llega vacía); con `countVisit=false` no se llama a `AddVisit` y se devuelve la URL.
> - **LIS-02:** no hay cambio. Mi test ya usaba `l.Expired(svc.Now())`.
>
> **Identificadores con guion**
> He añadido encima de cada una de las 27 funciones un comentario que empieza por su criterio con guion (`// BOR-01: …`). No he cambiado ningún nombre de función.
>
> **Ambigüedades pendientes**
> - Si `Insert` devuelve `ErrCodeTaken` con un alias explícito pero el `Get` previo no lo vio (carrera), la spec no dice qué error se devuelve. Lo natural sería `AliasTakenError`, pero no lo he testeado.
> - Con el almacén real no se puede provocar el borrado entre `Get` y `AddVisit`, así que RES-05 se prueba solo con el `Store` falso.

Dos notas del orquestador:

- El agente **se corrige a sí mismo** («dije 27 tests y eran 26»). Los informes de un agente son tan fiables como cualquier otro texto generado: por eso el orquestador cuenta los fallos con sus propias órdenes en vez de copiar la cifra.
- La primera ambigüedad pendiente da por hecho un `Get` previo al `Insert`. No debe haberlo: quien decide si un alias está ocupado es la restricción `UNIQUE` del almacén (ALI-06, ALM-03). Va explícito en el encargo del verde.

**Commits**: `Spec 002: qué pasa cuando algo falla a mitad (RES-05)` y `shortener: tests de los casos de uso (rojo)`. 27 tests, 27 en rojo.

## Encargo: paso verde

> Los tests del paso rojo ya están en un commit. Ahora, el **paso verde**: implementa el paquete `shortener` hasta que pase `go test ./shortener/`, también con `-race`.
>
> **El método.** Lo mínimo que haga pasar los tests y cumpla la spec 002. **No modifiques `shortener/shortener_test.go`**: si crees que un test está mal o contradice la spec, párate y dilo en el informe. No repitas reglas que ya viven en `link`: valida con `link.Validate` y genera con `link.NewCode`. Para saber si un alias o un código están ocupados **no hagas un `Get` previo**: inserta y deja que decida el almacén (`link.ErrCodeTaken`); con alias, ese error se convierte en `*link.AliasTakenError`. Cuando esté en verde, limpia el código sin cambiar el comportamiento. Comentarios en español que expliquen el porqué.
>
> **Los límites.** Solo `shortener/`. Solo la biblioteca estándar y los paquetes `link` del repo (el paquete no importa `store`: recibe la interfaz). Sin commits.
>
> **Terminado es.** `gofmt -l .` vacío, `go vet ./...` limpio, `go test ./...` y `go test -race ./shortener/` en verde.
>
> **El informe.** Ficheros tocados; decisiones que has tomado y por qué; desviaciones de la spec; si has tocado algún test (no deberías); y la salida resumida de `go test ./...`.

## Informe: paso verde

> Paso verde hecho. Todo `shortener` pasa: `go test ./...` da ok en `link`, `shortener` y `store`, y `go test -race ./shortener/` también. `gofmt -l .` sale vacío y `go vet ./...` limpio. No he tocado ningún test y no hay commits.
>
> **Fichero tocado:** `shortener/shortener.go`. El paquete solo importa la biblioteca estándar y `link`, no `store`.
>
> **Decisiones**
> - **`New`:** con `nil` usa `time.Now` y `link.NewCode(rand.Reader)` (COD-04).
> - **Reloj en `Create`:** se lee una sola vez, en UTC y truncado a segundos. Ese mismo instante se usa en `link.Validate` y como `CreatedAt`, para que no se desfasen.
> - **Alias:** `Insert` directo, sin `Get` previo. Si el almacén devuelve `ErrCodeTaken`, se convierte en `*link.AliasTakenError`. Cualquier otro error se devuelve envuelto con `%w`.
> - **Código generado:** hasta 5 intentos. Un código reservado se descarta sin insertarlo, y `ErrCodeTaken` hace reintentar. Si se agotan los 5, devuelve `link.ErrNoCodeAvailable`. El error del generador o cualquier otro error de `Insert` se devuelve envuelto, sin más intentos (COD-02).
> - **`Resolve`:** un enlace caducado da `ErrExpired` envuelto, también con `countVisit=false`. Si `AddVisit` falla, devuelve ese error y no la URL (RES-05). Un borrado entre `Get` y `AddVisit` da `ErrNotFound` desde el almacén. La atomicidad de VIS-02 la da el `UPDATE visits = visits + 1` del almacén, no el servicio.
> - **`List`, `Delete` y `Now`:** delegan en el almacén y el reloj.
>
> **Desviaciones de la spec:** ninguna.

## Verificación del orquestador

- `git diff -- shortener/shortener_test.go` vacío.
- `gofmt` y `go vet` limpios; `go test -count=1 ./...` en verde en `link`, `store` y `shortener`; `go test -race ./shortener/` en verde.
- `shortener` no importa `store`: depende de la interfaz, como pide la spec 001.

**Commit**: `shortener: crear, resolver, listar y borrar (verde)`.
