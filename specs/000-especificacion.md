# 000 — Especificación de producto

## Qué es

**acorta** es un acortador de URLs autocontenido: un único binario Go que incluye la API REST, la CLI y una interfaz web en Vue. Convierte una URL larga en una corta (`http://localhost:8080/k3x9ab`) que redirige a la original, y cuenta cuántas veces se usa.

Es también el **proyecto conductor del tutorial [iadev](https://iadev.xavi.net)**: se construye con SDD + TDD y un agente de código, y cada capítulo del tutorial enseña fragmentos reales de este repositorio (specs, encargos, tests, diffs y commits). Por eso el historial y la bitácora importan tanto como el código.

## Objetivos

1. Que cualquiera pueda leerlo entero: poco código, comentado, sin magia.
2. Que cada comportamiento esté escrito en `specs/` antes que en el código y tenga al menos un test.
3. Que desplegar sea copiar un fichero: sin Node en producción, sin servicios externos.

## Fuera de la v1

- **Autenticación**: en la v1 cualquiera que llegue a la API puede crear y borrar. Está pensado para uso local. Será la primera ampliación tras la v1, como spec nueva.
- Editar un enlace ya creado (se borra y se crea otro).
- Paginación, búsqueda u ordenación configurable del listado.
- Analítica de visitas (origen, país, navegador, fechas): solo se guarda el total.
- Detección de bots o de visitas repetidas.
- Deduplicación: acortar dos veces la misma URL da dos enlaces distintos.
- Códigos QR, dominios personalizados, borrado automático de enlaces caducados.

## Historias de usuario

| # | Como… | quiero… | para… | Spec |
|---|---|---|---|---|
| H1 | usuario | acortar una URL y recibir un código generado | compartir un enlace corto | 002, 004 |
| H2 | visitante | abrir un enlace corto y llegar a la URL original | usar el enlace sin saber nada de acorta | 002, 004 |
| H3 | usuario | ver mis enlaces con sus visitas | saber cuáles se usan | 002, 004, 006 |
| H4 | usuario | elegir yo el código (un alias) | que el enlace sea legible: `/oferta-otono` | 002 |
| H5 | usuario | poner fecha de caducidad a un enlace | que deje de funcionar cuando ya no toca | 002 |
| H6 | usuario | borrar un enlace | que deje de redirigir | 002, 004 |
| H7 | usuario de terminal | hacer todo lo anterior desde la CLI | usarlo en scripts sin abrir el navegador | 005 |

## Glosario

| Término | Significado |
|---|---|
| **Enlace** | La pareja código → URL de destino, con su fecha de creación, su caducidad opcional y su contador de visitas |
| **Código** | El identificador del enlace en la URL corta. Lo genera acorta o lo elige el usuario (alias) |
| **Alias** | Un código elegido por el usuario |
| **URL de destino** | La URL larga a la que redirige el enlace |
| **URL corta** | La URL base del servidor más el código: `http://localhost:8080/k3x9ab` |
| **Visita** | Una redirección servida con éxito por `GET` |
| **Caducado** | Enlace cuya fecha de caducidad ya ha llegado: existe, pero no redirige |
| **Resolver** | Buscar un código para redirigir: comprueba que existe y que no ha caducado, y suma la visita |

## Índice de specs

| Spec | Contenido |
|---|---|
| `000-especificacion.md` | Este documento: producto, alcance, historias |
| `001-plan-tecnico.md` | Stack cerrado, paquetes, convenciones, definición de terminado |
| `002-enlaces.md` | Reglas de dominio: URL, código, alias, caducidad, visitas, borrado, listado |
| `003-almacen.md` | Esquema SQLite y API del paquete `store` |
| `004-api.md` | API REST y redirección |
| `005-cli.md` | Comandos y flags de la CLI |
| `006-web.md` | Interfaz web (Vue) |
| `007-fases.md` | Fases de implementación y propiedad de ficheros |

Las decisiones de producto que dieron lugar a estas specs están en `bitacora/001-entrevista.md`.
