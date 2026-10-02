# 008 — Autenticación

El primer cambio de requisitos después de la v1. La spec 000 lo dejaba fuera a propósito («será la primera ampliación tras la v1, como spec nueva»); esta es esa spec. Lo que cambia: **crear y borrar enlaces por la API exige un token**. Redirigir, listar y la comprobación de vida siguen siendo públicos. La CLI no cambia su comportamiento con los enlaces porque no pasa por la API (spec 005), pero `serve` necesita el token para arrancar.

Decisiones tomadas en la entrevista del 2026-10-03 (bitácora 010):

| Pregunta | Decisión |
|---|---|
| ¿Qué exige autenticación? | Crear y borrar. Listar y redirigir, públicos |
| ¿Qué credencial? | Un token de API único, configurado al arrancar el servidor |
| ¿Y si el servidor arranca sin token? | Se niega a arrancar |
| ¿Cómo se autentica la interfaz web? | Pide el token una vez y lo guarda en el navegador |

## El token

Un texto de **al menos 16 caracteres**, sin espacios en los extremos (se recortan). Llega al servidor por el flag `-token` de `serve` o por la variable de entorno `ACORTA_TOKEN`, con la misma precedencia que el resto de la configuración (spec 005). Viaja en cada petición protegida en la cabecera `Authorization: Bearer <token>`. La comparación se hace en tiempo constante (`crypto/subtle`), para no filtrar nada por el tiempo de respuesta.

El token nunca se escribe en el log ni en ninguna respuesta.

## API

Rutas protegidas: `POST /api/links` y `DELETE /api/links/{código}`. Las demás rutas de la spec 004 no cambian y no miran la cabecera.

- **AUT-01** — `POST /api/links` sin cabecera `Authorization` responde `401` con `{"errors":["falta el token de API"]}` y la cabecera `WWW-Authenticate: Bearer`. No se crea nada.
- **AUT-02** — `DELETE /api/links/{código}` sin cabecera `Authorization` responde `401` con el mismo cuerpo y la misma cabecera, aunque el código no exista. No se borra nada.
- **AUT-03** — Con cabecera `Authorization` pero sin el token correcto (otro token, el esquema que no es `Bearer`, `Bearer` sin nada detrás), las dos rutas responden `401` con `{"errors":["token de API incorrecto"]}` y `WWW-Authenticate: Bearer`. El esquema `Bearer` no distingue mayúsculas (`bearer` vale); el token sí. La cabecera es exactamente el esquema, un espacio y el token: no se recorta nada, así que `Bearer  token` (dos espacios) o `Bearer token ` (espacio final) son incorrectos.
- **AUT-04** — Con `Authorization: Bearer <token correcto>`, las dos rutas se comportan exactamente como dice la spec 004 (`201`, `400`, `409`, `503`, `204`, `404`).
- **AUT-05** — El token se comprueba **antes** de leer el cuerpo: un `POST` sin token con un cuerpo inválido responde `401`, no `400`. Un `DELETE` sin token de un código inexistente responde `401`, no `404`.
- **AUT-06** — `GET /api/links`, `GET /api/healthz`, `GET /{código}`, `HEAD /{código}` y los estáticos no exigen token y responden igual con una cabecera `Authorization` correcta, incorrecta o ausente.

El `405` de API-12 no cambia: el método se comprueba antes que el token, así que `PUT /api/links` sin token sigue dando `405`.

## CLI

- **AUT-07** — `acorta serve -token TOKEN` arranca con ese token; sin flag, usa `ACORTA_TOKEN`; el flag gana a la variable. La sinopsis pasa a ser `acorta serve [-addr :8080] [-db RUTA] [-base-url URL] [-token TOKEN]`.
- **AUT-08** — `acorta serve` sin token (ni flag ni variable, o una variable vacía) escribe `error: falta el token de API: usa -token o ACORTA_TOKEN` en `stderr`, sale con 1 y no llega a abrir la base de datos ni a escuchar.
- **AUT-09** — Un token de menos de 16 caracteres (contados después de recortar espacios) escribe `error: el token de API debe tener al menos 16 caracteres`, sale con 1 y no llega a escuchar.
- **AUT-10** — El mensaje de arranque de CLI-09 no cambia y no contiene el token. Tampoco aparece en la ayuda de `serve -h` más allá del nombre del flag y de la variable.

`add`, `list` y `rm` no cambian: trabajan sobre la base de datos directamente, con el servidor parado o en marcha, y no necesitan token.

## Interfaz web

La pantalla gana un campo **Token de API** (`type="password"`, con su `<label>`) y un botón **Guardar**, encima del formulario de creación.

- **AUT-11** — Al pulsar Guardar, el token (recortado) se guarda en `sessionStorage` con la clave `acorta_token` y el campo muestra el estado `Token guardado`. Al cargar la página, si hay un token guardado, el campo aparece relleno con él y en ese estado. Guardar un campo vacío borra el token guardado.
- **AUT-12** — Crear (WEB-01) y borrar (WEB-12) envían `Authorization: Bearer <token guardado>`. Sin token guardado no envían la cabecera. Cargar el listado (WEB-08, WEB-13) no la envía nunca.
- **AUT-13** — Si crear o borrar reciben un `401`, el error se muestra como cualquier otro de la API (WEB-04, WEB-12: tal como llega) y además el campo del token recibe el foco. No se borra el token guardado: puede ser una errata.

## Contrato del paquete

`Handler` recibe el token como cuarto parámetro. Un token vacío no significa «sin autenticación»: la CLI no deja que llegue vacío (AUT-08), y si llegara, ninguna petición protegida se aceptaría: sin cabecera, `falta el token de API`; con `Bearer` y cualquier cosa detrás, `token de API incorrecto`. No es un criterio aparte porque desde fuera del paquete no se puede provocar.

```go
// Handler devuelve el servidor completo. token es el token de API que exigen
// POST /api/links y DELETE /api/links/{código} (spec 008).
func Handler(svc Service, baseURL string, static fs.FS, token string) http.Handler
```

## Verificación

- AUT-01 a AUT-06: tests de `server` con `httptest`, como la spec 004. Los tests de la spec 004 que crean o borran envían el token correcto (AUT-04: con él, todo sigue igual); los de la spec 008 son los únicos que controlan la cabecera.
- AUT-07 a AUT-10: tests de `cmd/acorta` contra `run`, como la spec 005.
- AUT-11 a AUT-13: tests de componentes en `web/`, como la spec 006, con `sessionStorage` real de jsdom.
- Antes de cerrar, el orquestador comprueba con `curl` contra el binario: `POST` sin token (401), con token malo (401), con token bueno (201); `DELETE` igual; `GET /api/links` sin token (200); `serve` sin token (salida 1); y en el navegador, guardar el token, crear y borrar.
