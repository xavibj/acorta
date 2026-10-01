# 001 — Plan técnico

Arquitectura: binario Go único con frontend Vue embebido, API REST y CLI.

```
┌──────────────────────────── binario acorta ─────────────────────────────┐
│  cmd/acorta/    CLI: run() testeable + subcomando serve                  │
│      │                                                                   │
│      ├── shortener/   casos de uso: crear, resolver, listar, borrar      │
│      │       ├── link/    dominio puro: tipos, validación, códigos       │
│      │       └── store/   SQLite                                         │
│      └── server/      Handler(): API REST + redirección                  │
│              └── static/  (go:embed) aviso + dist/, la salida de web/    │
└──────────────────────────────────────────────────────────────────────────┘
   navegador ── GET /{código} ──▶ redirección
   navegador ── /  y  /api/… ───▶ interfaz web y JSON, mismo origen
   terminal  ── acorta add … ───▶ CLI, directa a shortener (sin HTTP)
```

## Stack (cerrado)

| Pieza | Elección |
|---|---|
| Lenguaje | Go 1.27, `CGO_ENABLED=0` |
| HTTP | Biblioteca estándar: `net/http` y `encoding/json`. El reparto de rutas se hace a mano en `server`, sin `http.ServeMux`: son pocas rutas, la API tiene que contestar los 404 y 405 en JSON con un `Allow` exacto, y la regla que separa un código de un fichero estático (un segmento sin punto) no se expresa con patrones |
| Base de datos | SQLite con `modernc.org/sqlite` (Go puro, sin CGO) a través de `database/sql` |
| CLI | Paquete `flag` de la biblioteca estándar |
| Frontend | Vue 3 (SFC con `<script setup>`, JavaScript), Vite, Tailwind CSS v4 |
| Tests de frontend | Vitest + `@vue/test-utils` + `jsdom` |

**Dependencias Go permitidas**: solo `modernc.org/sqlite` y sus indirectas. **Dependencias npm permitidas**: `vue`; en desarrollo, `vite`, `@vitejs/plugin-vue`, `tailwindcss`, `@tailwindcss/vite`, `vitest`, `@vue/test-utils`, `jsdom`. Añadir cualquier otra exige cambiar antes esta spec.

## Paquetes

| Paquete | Responsabilidad | Regla |
|---|---|---|
| `link/` | Tipo `Link`, validación de la entrada, generación de códigos, errores centinela | **Puro**: sin base de datos, sin red, sin reloj global ni aleatoriedad global. El tiempo y la fuente de azar se reciben por parámetro |
| `store/` | Persistencia en SQLite | No valida reglas de negocio; solo guarda y recupera |
| `shortener/` | `Service` con los casos de uso. Une `link` y `store` | Recibe el almacén (interfaz), el reloj y el generador de códigos: en los tests se sustituyen |
| `server/` | `Handler() http.Handler`: API JSON, redirección y estáticos embebidos | Adaptador fino: traduce HTTP ↔ `shortener`. Sin lógica de negocio |
| `cmd/acorta/` | `main` y `run(args, stdout, stderr) int` | Adaptador fino: traduce argumentos ↔ `shortener` |
| `web/` | Proyecto Vite (fuente del frontend). No se embebe | Su build escribe en `server/static/dist/`, que no está en Git |

Módulo Go: `acorta`. Dependencias entre paquetes, en un solo sentido: `link` ← `store` ← `shortener` ← `server` ← `cmd/acorta`.

## Convenciones

- **Idioma**: identificadores en inglés (`Link`, `Create`, `ErrNotFound`); comentarios en español, porque el código lo van a leer quienes siguen el tutorial; todo texto que ve el usuario (errores, CLI, web) en español.
- **Los mensajes de error de validación salen de un único sitio**, el paquete `link`. La API, la CLI y la web los muestran tal cual; nadie los reescribe.
- **Tiempo**: siempre UTC, precisión de segundos, formato RFC 3339 (`2026-12-31T23:59:59Z`) en JSON y en la base de datos.
- **Errores**: envueltos con `%w`; se comparan con `errors.Is` contra los centinela de `link`.
- **Tests**: de tabla; nombre o comentario con el identificador del criterio que comprueban (`URL-04`); base de datos real en `t.TempDir()`; `httptest` para `server`; la CLI se prueba contra `run()` sin lanzar procesos ni abrir sockets.
- **El frontend compilado no va en Git**: `server/static/dist/` está en `.gitignore`. En el repo solo hay código que se puede leer y revisar; lo generado se genera. `make build` compila siempre el frontend antes que el binario, así que no pueden quedar desfasados.
- **Un clon sin compilar funciona**: `go build` y `go test ./...` no necesitan Node. Sin frontend compilado, la API, la CLI y las redirecciones van igual y `/` muestra un aviso (spec 004, EST-04). El binario completo, con la interfaz, sí necesita Node para construirse.

## Órdenes

```sh
make test       # go test ./... + tests de web/
make check      # gofmt -l . (vacío) + go vet ./... + make test + build de web/
make frontend   # cd web && npm ci && npm run build  → server/static/dist/
make build      # frontend + go build -o bin/acorta ./cmd/acorta
```

## Definición de terminado

Un cambio está terminado cuando, sobre **todo** el repositorio y no solo lo tocado:

1. `gofmt -l .` no lista nada, `go vet ./...` no avisa y `go test ./...` pasa.
2. Si toca `web/`: sus tests pasan y `npm run build` termina sin errores.
3. Cada criterio nuevo o cambiado de la spec tiene al menos un test que lo nombra.
4. Si el cambio contradice una spec, la spec se ha actualizado en el mismo cambio.
