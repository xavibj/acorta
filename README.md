# acorta

Acortador de URLs en un único binario Go, con API REST, CLI e interfaz web en Vue.

Es el proyecto conductor del tutorial [iadev.xavi.net](https://iadev.xavi.net): está construido con especificaciones (SDD), tests primero (TDD) y agentes de código, y el tutorial enseña cómo, con fragmentos reales de este repositorio. Por eso aquí importa tanto el código como el camino: cada funcionalidad tiene su criterio en [`specs/`](specs/000-especificacion.md), su test, un commit en rojo y otro en verde, y [`bitacora/`](bitacora/) guarda los encargos que se dieron a los agentes y lo que contestaron.

## Usarlo

Hace falta Go 1.27 y, para la interfaz web, Node 22 o superior.

```sh
git clone https://github.com/xavibj/acorta.git && cd acorta
make build                      # compila el frontend y el binario: bin/acorta
bin/acorta serve                # http://localhost:8080
```

Desde la terminal, con el servidor en marcha o parado:

```
$ bin/acorta add -alias demo -ttl 72h https://example.com/una/ruta/larga
http://localhost:8080/demo
$ bin/acorta add https://go.dev/doc/
http://localhost:8080/k3x9ab
$ bin/acorta list
CÓDIGO  VISITAS  CADUCA                URL
k3x9ab  0        nunca                 https://go.dev/doc/
demo    0        2026-10-04T17:31:14Z  https://example.com/una/ruta/larga
$ bin/acorta rm demo
Borrado: demo
```

Por HTTP:

```sh
curl -X POST -d '{"url":"https://example.com","alias":"promo"}' localhost:8080/api/links
curl -i localhost:8080/promo          # 302 a https://example.com, y cuenta la visita
curl localhost:8080/api/links
curl -X DELETE localhost:8080/api/links/promo
```

Sin Node también funciona: `go build ./cmd/acorta` da un binario con la API, la CLI y las redirecciones, y en `/` un aviso de que falta compilar el frontend.

La v1 no tiene autenticación: está pensada para uso local.

## Desarrollarlo

```sh
make test       # tests de Go y de la interfaz
make check      # formato, go vet, tests y build del frontend
```

Las reglas están en [`AGENTS.md`](AGENTS.md): nada se implementa sin estar en una spec, los tests van antes y fallan antes de pasar, y nadie cambia un test para que pase.

| Dónde | Qué |
|---|---|
| `link/` | Dominio puro: validación, códigos, errores |
| `store/` | SQLite |
| `shortener/` | Casos de uso: crear, resolver, listar, borrar |
| `server/` | API JSON, redirección y la interfaz embebida |
| `cmd/acorta/` | CLI |
| `web/` | Interfaz en Vue 3 (su build no está en Git) |
| `specs/` | El contrato: 100 criterios numerados |
| `bitacora/` | Cómo se construyó, fase a fase |

## Licencia

[MIT](LICENSE).
