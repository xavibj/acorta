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
