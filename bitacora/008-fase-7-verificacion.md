# 008 — Fase 7: verificación de extremo a extremo

**Fecha**: 2026-10-01 · **Spec**: `specs/007-fases.md`, «Verificación final» · **Quién**: el orquestador, sin agentes

Los tests prueban piezas. Esta fase prueba la historia completa con lo que un usuario tendría delante: el binario compilado con `make build`, una base de datos nueva, una terminal, `curl` y un navegador.

## Qué se comprobó

**CLI y HTTP: 44 comprobaciones, todas correctas.**

- CLI: `add` (código de 6 caracteres), `add -alias`, `add -ttl`, alias repetido (salida 1 y su mensaje), validación (salida 1, `stdout` vacío, dos líneas de error), `add` sin URL (salida 2 y uso), subcomando desconocido (2), base de datos en un directorio inexistente (1), `list` y `version`.
- Servidor: mensaje de arranque, `healthz`, `GET /promo` → 302 con `Location` y `Cache-Control: no-store`; dos `GET` y un `HEAD` dejan el contador en 2; código inexistente → 404; un enlace con `-ttl 2s` da 410 a los dos segundos y aparece caducado en la lista; crear por API → 201 con su `short_url`; alias repetido → 409; tres errores de validación a la vez → 400; cuerpo `null` → 400; `PUT /api/links` → 405 con `Allow: GET, HEAD, POST`; ruta de API desconocida → 404 en JSON; borrar → 204 y después 404; `GET /` sirve la interfaz y su JavaScript llega con el tipo correcto.
- Persistencia: se para el servidor y se arranca otra vez sobre el mismo fichero: el enlace conserva sus 2 visitas. `acorta rm` con el servidor en marcha borra el enlace y la URL corta pasa a dar 404.

**Navegador real** (Chromium sin interfaz contra el binario), a 390 px en claro, a 1280 px en oscuro y a 360 px:

- Enviar una URL `ftp://` con alias `A` muestra los dos errores de la API a la vez; al editar un campo, desaparecen.
- Crear un enlace con alias y caducidad de 7 días: el formulario se vacía, el enlace aparece el primero con `0 visitas` y su `Caduca el …`.
- Copiar deja la URL corta en el portapapeles y el botón dice `Copiado`.
- Se visita la URL corta desde fuera, se pulsa Actualizar y pasa a `1 visita`.
- Repetir el alias muestra `alias: "…" ya está en uso` y conserva lo escrito.
- Borrar quita el enlace y su URL corta da 404.
- Sin scroll horizontal en ninguno de los tres anchos y todos los controles miden al menos 44 px de alto.

## Lo que falló

Seis comprobaciones del primer intento, y las seis eran errores **del guion de verificación**, no de acorta: el guion llamaba a `list -base-url`, un flag que la spec acababa de eliminar, y esperaba los cuerpos de texto sin su salto de línea final, que la spec 004 exige. El binario hacía lo que dice la spec; el guion estaba escrito de memoria. Es el mismo error que en las fases 5 y 6, cometido ahora por el orquestador: una comprobación que no sale de la spec comprueba lo que uno cree recordar.

En el navegador, la consola registra los 400 y 409 como «error al cargar un recurso». Es el navegador contando una respuesta esperada; la interfaz los trata y los muestra.

## Estado al cerrar la v1

| | |
|---|---|
| Criterios en las specs | 100 |
| Tests Go | 97 funciones, 278 casos contando subtests |
| Tests de la interfaz | 38 |
| Cobertura por paquete | `link` 100 %, `shortener` 100 %, `cmd/acorta` 98 %, `server` 93 %, `store` 85 % |
| Código Go | unas 1.200 líneas, más 3.100 de tests |
| Commits | uno por paso: spec, rojo, verde, y las correcciones de spec y de tests entre medias |

## Lo que enseñó construirlo así

- **El paso rojo es un revisor de specs.** En cada fase, escribir los tests destapó entre tres y nueve huecos en una spec que ya había pasado una revisión humana. Se cerraron en la spec, no en el código, y de ahí salieron cuatro criterios nuevos (RES-05, RED-07, ALM-09 y los EST del cambio del frontend).
- **Los tests también se equivocan.** Tres veces un agente llegó al verde «menos uno o dos» y los tests que fallaban estaban mal: buscaban un botón por un texto que ya había cambiado, medían columnas en bytes, usaban un servidor después de cerrar su base de datos. Las tres veces el agente paró y lo dijo, porque la regla estaba escrita. Las tres correcciones están en commits propios.
- **Verde no es conforme.** El servidor pasó todos sus tests ignorando una línea del plan técnico. Lo cazó leer el informe con la spec al lado. Desde entonces los encargos piden las desviaciones de *cualquier* spec.
- **Los informes se verifican.** Un agente contó mal sus propios tests; otro dijo «sin desviaciones» con una desviación en el párrafo anterior. El orquestador ejecuta sus propias órdenes y cuenta él.
