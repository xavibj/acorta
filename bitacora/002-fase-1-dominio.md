# 002 — Fase 1: el dominio (`link/`)

**Fecha**: 2026-10-01 · **Spec**: `specs/002-enlaces.md` · **Rutas**: `link/**`

El orquestador (la sesión principal de Claude Code) no escribe el código: se lo encarga a un agente que arranca sin saber nada del proyecto. Todo lo que ese agente necesita está en el repo. Aquí quedan, tal cual, lo que se le pidió y lo que contestó.

## Encargo: paso rojo

> **Qué leer antes.** Trabajas en el repositorio `/home/claude/Projects/acorta`. Antes de escribir nada lee, en este orden: `AGENTS.md`, `specs/001-plan-tecnico.md` y `specs/002-enlaces.md`. No necesitas leer las demás specs.
>
> **La tarea.** Eres el agente de la fase 1 de `specs/007-fases.md`: el paquete `link`, el dominio puro de acorta. Este encargo es solo el **paso rojo**:
>
> 1. Crea en `link/` los tipos, funciones y errores del bloque `package link` de «Contrato de los paquetes» (spec 002), con las firmas exactas, pero **sin implementar**: cada función devuelve valores cero. Lo justo para que los tests compilen.
> 2. Escribe los tests de todos los criterios de la spec 002 que son responsabilidad de `link`: URL-01 a URL-08, COD-01, ALI-01 a ALI-05, CAD-01 a CAD-05, además del orden y la forma de los errores de validación (todos a la vez, uno por campo, en el orden url, alias, expires_at), `Reserved`, `Link.Expired` y los mensajes de `ValidationError` y `AliasTakenError`. Los demás criterios (COD-02 a COD-04, ALI-06, CAD-06, RES, VIS, LIS, BOR) son del paquete `shortener` y no te tocan.
> 3. Ejecuta `go test ./link/` y comprueba que los tests **fallan por las aserciones**, no por errores de compilación.
>
> **El método.** Tests de tabla. Cada caso o cada test lleva en su nombre el identificador del criterio que comprueba (`URL-04`). Incluye los casos límite que la spec nombra (2048 bytes exactos, alias de 3 y de 32, caducidad igual al instante actual). El reloj se pasa por parámetro: usa un instante fijo, nunca `time.Now()`. Para `NewCode` usa un `io.Reader` determinista.
>
> **Los límites.** Solo puedes crear o modificar ficheros dentro de `link/`. No toques `go.mod` ni `go.sum`, no ejecutes `go mod tidy`, no añadas dependencias. No hagas commits. No implementes todavía. Algún caso puede pasar con la función vacía (por ejemplo, «esto no es una palabra reservada»): es normal, pero dilo en el informe.
>
> **Terminado es.** `go vet ./...` limpio, `gofmt -l .` vacío, y `go test ./link/` compilando y fallando.
>
> **El informe.** Al acabar, responde con: ficheros creados; cuántos tests o casos hay y cuántos fallan; una tabla criterio → nombre del test; cualquier ambigüedad o contradicción que hayas encontrado en la spec (no la resuelvas en silencio: dila); y las últimas líneas de la salida de `go test ./link/`.

## Informe: paso rojo

Lo que devolvió el agente, sin tocar (88 segundos de trabajo):

> Paso rojo hecho: `go test ./link/` compila y falla por las aserciones. `gofmt -l .` vacío y `go vet ./...` limpio. Sin commits; no toqué `go.mod`/`go.sum` ni nada fuera de `link/`.
>
> FICHEROS CREADOS
> - `link/link.go` — tipos, firmas y errores del contrato. Las funciones son esqueletos con valores cero: `Expired`=false, `Validate`=Valid{},nil, `NewCode`="",nil, `Reserved`=false, `Error()`="". Los centinela ya tienen sus mensajes reales.
> - `link/link_test.go` — todos los tests.
>
> RECUENTO
> 76 casos hoja: 63 fallan, 13 pasan con la función vacía (esperable). Los 13 que pasan:
> - Alias y expires_at vacíos o solo espacios (ALI-01 x2, CAD-01 x2).
> - "entrada correcta: sin mensajes" (`Validate` devuelve nil).
> - `Reserved` = false para API, apis, oferta y "".
> - `Link.Expired` = false sin caducidad (CAD-05 x2).
> - `errors.As` sobre `ValidationError`.
> - Los mensajes de los 4 centinela.
>
> CRITERIO -> TEST (el identificador va en el nombre de cada caso)
> - URL-01 a URL-08 -> TestValidateURL
> - COD-01 -> TestNewCode
> - ALI-01 a ALI-04 -> TestValidateAlias
> - ALI-05 -> TestValidateAlias y TestReserved
> - CAD-01 a CAD-04 -> TestValidateExpiresAt
> - CAD-05 -> TestLinkExpired
> - Orden y forma de los errores (todos a la vez, uno por campo, orden url/alias/expires_at) -> TestValidateAllErrorsAtOnce
> - `Valid` limpio -> TestValidateReturnsCleanValid
> - Mensajes de ValidationError, AliasTakenError (ALI-06) y centinela -> TestErrorMessages
>
> Casos límite cubiertos: 2048 y 2049 bytes; alias de 2, 3, 32 y 33; expires_at igual a now (también en otra zona horaria y en `Expired`).
>
> AMBIGUEDADES / CONTRADICCIONES DE LA SPEC (no resueltas en silencio)
> 1. ALI-06 y el texto final de la spec dicen que `Create` devuelve `ErrAliasTaken`, pero el contrato solo define `AliasTakenError` y `ErrCodeTaken`; `ErrAliasTaken` no existe. Asumí que ALI-06 se refiere a `*AliasTakenError`. La spec debería corregir el nombre.
> 2. Formato de `ValidationError.Error()` con varios mensajes: no está especificado. El test exige que con un mensaje sea ese mensaje, y con varios que los contenga todos y en orden. No fijé el separador.
> 3. ALI-02 dice "caracteres": no se sabe si son runas o bytes. `año-nuevo` falla en ALI-03 con cualquiera de las dos lecturas, así que no lo distingue. No lo testeé.
> 4. Orden de las reglas dentro de un campo, solo implícito. Asumí URL: obligatoria, longitud, URL válida, esquema, dominio. Alias: longitud, caracteres, guion, reservado. Así, `ftp://` da el error de esquema y `-Abc` da el de caracteres.
> 5. URL-04 habla de bytes y mi test usa solo ASCII. Tampoco se dice si los 2048 se miden antes o después de recortar espacios; asumí después.
> 6. CAD-03: no aclara si la comparación con `now` se hace antes o después de truncar las fracciones (por ejemplo 12:00:00.5 con now = 12:00:00). No testeado.
> 7. `NewCode`: la spec no dice si usa módulo o rechazo para los 36 símbolos. Mis tests solo comprueban longitud, alfabeto, determinismo con la misma fuente, códigos distintos con fuentes distintas, cobertura del alfabeto en 500 códigos y propagación de errores. Suponen que una fuente agotada da error y que un error del lector se envuelve con %w.
> 8. No se define qué `Valid` devuelve `Validate` cuando hay mensajes; no testeado.
>
> ULTIMAS LINEAS DE `go test ./link/`
> ```
>     --- FAIL: TestErrorMessages/ValidationError_con_un_mensaje (0.00s)
>         link_test.go:416: mensaje = ""
>     --- FAIL: TestErrorMessages/ValidationError_con_varios_mensajes_los_contiene_todos_en_orden (0.00s)
>         link_test.go:426: mensaje = "", debe contener "url: es obligatoria" y después "alias: \"api\" está reservado"
> FAIL
> FAIL	acorta/link	0.003s
> FAIL
> ```

## Revisión del orquestador

Comprobado de primera mano, sin fiarse del informe: solo hay ficheros nuevos en `link/`; `gofmt` y `go vet` limpios; los tests compilan y fallan; y los 20 criterios que tocan a `link` aparecen nombrados en algún test (los 4 que faltan, COD-02 a COD-04 y CAD-06, son del paquete `shortener`).

Lo valioso del paso rojo no fueron los tests, sino la lista de ambigüedades. Escribir un test obliga a decidir el resultado exacto, y ahí es donde una spec que parecía completa enseña sus huecos. Ocho, en una spec que ya había pasado una revisión humana. Ninguno se resolvió en el código: se resolvieron **en la spec**, antes de implementar.

| # | Hueco | Cómo queda la spec 002 |
|---|---|---|
| 1 | ALI-06 nombraba un error que no existe en el contrato | Corregido: `*AliasTakenError` |
| 2 | Separador de `ValidationError.Error()` | Los mensajes se unen con `; ` |
| 3 | ¿«Caracteres» son letras o bytes en ALI-02? | Letras: `ñu` son 2 |
| 4 | Orden de las reglas dentro de un campo | Explícito, con dos ejemplos (`ftp://`, `-Abc`) |
| 5 | ¿Cómo y cuándo se miden los 2048 de URL-04? | En bytes y después de recortar espacios |
| 6 | ¿Cuenta `12:00:00.5` como futuro si son las `12:00:00`? | No: se comparan sin fracciones de segundo |
| 7 | ¿Cómo se convierte un byte al azar en un carácter? | `b % 36`, descartando los bytes ≥ 252 para no favorecer a ninguno; con un ejemplo exacto |
| 8 | ¿Qué devuelve `Validate` cuando hay errores? | Un `Valid` vacío |

## Encargo: ajuste del rojo

> La spec 002 ha cambiado para cerrar las ambigüedades que encontraste. Vuelve a leer `specs/002-enlaces.md` (han cambiado el párrafo sobre el orden de los errores, URL-04, COD-01, ALI-02, ALI-06, CAD-03 y los comentarios de `Validate` y `ValidationError` en el contrato).
>
> Sigue siendo el **paso rojo**: ajusta solo los tests de `link/link_test.go` (y el comentario de `link.go` si copia el contrato) para que cubran lo que ahora la spec sí dice: el separador `; ` de `ValidationError.Error()`; ALI-02 contando caracteres y no bytes (`ñu`, `ñus`); los dos ejemplos del orden de las reglas (`ftp://`, `-Abc`); URL-04 en bytes con una URL que tenga caracteres de más de un byte, y medida después de recortar; CAD-03 con fracciones de segundo; COD-01 con el ejemplo exacto de la spec (`255, 0, 1, 2, 3, 4, 5` → `abcdef`), el descarte de bytes ≥ 252 y la fuente agotada; y que `Validate` devuelve un `Valid` vacío cuando hay mensajes. Si algún test anterior contradice la spec nueva, corrígelo.
>
> Mismos límites que antes: solo `link/`, sin implementar, sin commits. Terminado es `go vet ./...` limpio, `gofmt -l .` vacío y `go test ./link/` compilando y fallando.
>
> Informe: qué tests has añadido o cambiado, el recuento nuevo (casos, cuántos fallan), y si queda alguna ambigüedad.

## Informe: ajuste del rojo

> Ajuste del rojo hecho, todo en `link/`: `link/link_test.go` y comentarios de `link/link.go`. `gofmt -l .` está vacío, `go vet ./...` limpio y `go test ./link/` compila y falla por las aserciones. No hice commits ni toqué la bitácora, `go.mod` o `go.sum`, ni nada fuera de `link/`.
>
> **Recuento nuevo**
> - Hay 97 casos hoja (antes 76): 84 fallan y 13 pasan con el esqueleto vacío.
> - Los 13 que pasan son los mismos de antes: alias y `expires_at` vacíos, `Validate` sin errores que da nil, `Reserved` en negativo, `Expired` sin caducidad o antes de `expires_at`, `errors.As` de `ValidationError` y los mensajes de los centinela.
> - Hay una excepción en la intención. `TestValidateEmptyValidOnErrors` (Valid vacío cuando hay mensajes) fallaría de verdad en el verde si no se devolviera el valor cero. Hoy el esqueleto ya devuelve `Valid{}`, así que ese test falla antes de llegar a esa comprobación, porque `Validate` no devuelve mensajes. Es normal en el rojo.
>
> **Tests añadidos o cambiados** (en `link/link_test.go`)
> - **URL-04** (`TestValidateURL`), 4 casos nuevos: 2048 bytes exactos con `ñ` (1014 `ñ`, solo 1034 caracteres) se acepta; 2049 bytes con `ñ` se rechaza; 1015 `ñ` se rechaza (2050 bytes, muy por debajo de 2048 caracteres); 2048 bytes con espacios alrededor se acepta, porque se mide después de recortar.
> - **ALI-02** (`TestValidateAlias`), 4 casos nuevos: `ñu` da el error de longitud; `ñus` da el error de caracteres (ALI-03); 17 `ñ` (34 bytes) da el error de caracteres, no el de longitud; 33 `ñ` da el error de longitud.
> - **Orden de reglas dentro de un campo:** los dos ejemplos de la spec (`ftp://` y `-Abc`) ya estaban en `TestValidateAllErrorsAtOnce` y coinciden con la spec nueva, así que no los cambié.
> - **CAD-03** (`TestValidateExpiresAt`), 3 casos nuevos: `12:00:00.5` da error; `12:00:00.999` da error; `12:00:01.5` se acepta y se guarda truncado a `12:00:01`. Además, `TestValidateExpiresAtClockWithFraction` usa un reloj con fracciones (`12:00:00.7`) y comprueba que `expires_at` = `12:00:00.9` da error.
> - **Valid vacío con mensajes:** `TestValidateEmptyValidOnErrors` (nuevo) comprueba, con 4 entradas, que `Validate` devuelve `Valid{}` cuando hay mensajes.
> - **`ValidationError`** (`TestErrorMessages`): cambié el caso de varios mensajes. Antes comprobaba que los contuviera todos en orden; ahora exige exactamente `url: es obligatoria; alias: "api" está reservado`. Ninguno de los tests anteriores contradecía la spec nueva.
> - **COD-01** (`TestNewCode`): el ejemplo exacto `255,0,1,2,3,4,5` da `abcdef`; el byte `b` da la posición `b%36` (`35,36,251,0,1,2` da `9a9abc`); se descartan los bytes ≥ 252 intercalados; una fuente que solo da bytes ≥ 252, una que se agota a mitad de código y una vacía dan error y código vacío; un error propio del lector se conserva con `errors.Is`.
>
> **Ambigüedades que quedan:**
> 1. Cuando la fuente se agota a mitad de código, la spec dice que se devuelve "ese error envuelto" pero no cuál. Con `io.ReadFull` puede ser `io.EOF` o `io.ErrUnexpectedEOF`, así que el test acepta cualquiera de los dos.
> 2. La spec no dice si `NewCode` descarta los bytes ya leídos al fallar. Solo exigí código vacío con error.
> 3. La spec no precisa qué zona o ubicación tiene el `now` que se compara, pero basta con comparar instantes y el test lo cubre.

El orquestador da por buenas las tres que quedan: no cambian nada que un usuario pueda ver, y el test ya acepta las lecturas razonables. No toda ambigüedad merece una línea más en la spec.

**Commits**: `Spec 002: aclaraciones que destapó el paso rojo` y `link: tests de la spec 002 (rojo)`. 97 casos, 84 en rojo.

## Encargo: paso verde

> Los tests del paso rojo ya están en un commit. Ahora, el **paso verde**: implementa el paquete `link` hasta que pase `go test ./link/`.
>
> **El método.** Lo mínimo que haga pasar los tests y cumpla la spec 002; nada que la spec no pida. **No modifiques `link/link_test.go`**: si crees que un test está mal o contradice la spec, párate y dilo en el informe en vez de cambiarlo. Cuando esté en verde, repasa el código y límpialo (nombres, duplicación, comentarios en español que expliquen el porqué) sin cambiar el comportamiento, con los tests en verde antes y después.
>
> **Los límites.** Solo `link/`. Solo la biblioteca estándar. Sin commits. El paquete es puro: nada de `time.Now()`, de `crypto/rand` global ni de E/S; el reloj y la fuente de azar llegan por parámetro.
>
> **Terminado es.** `gofmt -l .` vacío, `go vet ./...` limpio y `go test ./...` en verde en todo el repo.
>
> **El informe.** Ficheros tocados; decisiones que has tomado y por qué; cualquier desviación de la spec; si has tocado algún test (no deberías); y la salida resumida de `go test ./...`.
