# 007 — Fases de implementación

La sesión principal es el **orquestador**: especifica, encarga, verifica y hace los commits. Cada fase la implementa un agente que solo puede tocar las rutas de su fila. El orden sigue las dependencias entre paquetes.

| Fase | Qué | Spec | Rutas que puede tocar | Depende de |
|---|---|---|---|---|
| 0 | Esqueleto: `go.mod`, `go.sum`, `Makefile`, `.gitignore` | 001 | las citadas (solo el orquestador) | — |
| 1 | Dominio | 002 | `link/**` | 0 |
| 2 | Almacén | 003 | `store/**` | 1 |
| 3 | Casos de uso | 002 | `shortener/**` | 1, 2 |
| 4 | API, redirección y estáticos | 004 | `server/**` (incluida `server/static/sin-compilar.html`) | 3 |
| 5 | CLI | 005 | `cmd/acorta/**` | 3, 4 |
| 6 | Interfaz web | 006 | `web/**` | 4 |
| 7 | Verificación de extremo a extremo | todas | ninguna (solo el orquestador) | 1–6 |
| 8a | Autenticación en la API | 008 | `server/**` | 7 |
| 8b | Autenticación en la CLI (`serve -token`) | 008 | `cmd/acorta/**` | 8a |
| 8c | Autenticación en la interfaz web | 008 | `web/**` | 8a |

Reglas:

- `go.mod` y `go.sum` solo se tocan en la fase 0. Ningún agente ejecuta `go mod tidy`.
- `server/static/dist/` es la salida del build de `web/` y no está en Git: ninguna fase la edita a mano. La fase 4 no necesita que exista, porque sus tests usan un sistema de ficheros en memoria (spec 004, EST).
- Las fases van en orden. La 5 y la 6 no comparten ficheros y pueden ir en paralelo; lo mismo la 8b y la 8c.
- La fase 8 es el primer cambio de requisitos después de la v1 (spec 008). Cambia la firma de `server.Handler`, así que la 8b (que la llama) no puede empezar antes de que la 8a esté en verde.
- Si una fase necesita cambiar algo fuera de sus rutas, no lo cambia: lo dice en su informe.

## Cada fase, en dos pasos

Para que el historial enseñe el ciclo, cada fase deja al menos dos commits:

1. **Rojo**: el agente escribe los tests de todos los criterios de su spec y lo mínimo para que compilen (tipos y funciones vacías), los ejecuta y enseña que fallan. El orquestador revisa que cada criterio tiene su test y hace el commit: `link: tests de la spec 002 (rojo)`.
2. **Verde**: el mismo agente implementa hasta que pasa la suite completa, sin modificar los tests. Si cree que un test está mal, se para y lo dice. El orquestador verifica y hace el commit: `link: validación y códigos (verde)`.
3. **Refactor** (si hace falta), con la suite en verde antes y después: `link: refactor …`.

## Bitácora

Por cada fase, el orquestador guarda en `bitacora/` el encargo que le dio al agente y el informe que recibió, tal cual. Es el material del que salen los capítulos del tutorial.

## Verificación final (fase 7)

Con el binario compilado (`make build`) y una base de datos nueva:

1. CLI: `add`, `add -alias`, `add -ttl`, `list`, `rm`, y los casos de error con sus códigos de salida.
2. `acorta serve` y, con `curl`: crear, redirigir (302, y la visita cuenta), `HEAD` (no cuenta), código inexistente (404), caducado (410), alias repetido (409), validación (400 con todos los errores), borrar (204 y después 404).
3. Navegador real a 390 px y a 1280 px: el recorrido de la spec 006.
4. Parar el servidor, arrancarlo de nuevo: los enlaces y las visitas siguen ahí.
