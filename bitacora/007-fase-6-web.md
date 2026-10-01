# 007 — Fase 6: la interfaz web (`web/`)

**Fecha**: 2026-10-01 · **Spec**: `specs/006-web.md` · **Rutas**: `web/**`

## Encargo: paso rojo

> **Qué leer antes.** Trabajas en el repositorio `/home/claude/Projects/acorta`. Antes de escribir nada lee, en este orden: `AGENTS.md`, `specs/001-plan-tecnico.md` (stack y dependencias npm permitidas), `specs/006-web.md` entera y, de `specs/004-api.md`, las secciones «El enlace en JSON» y «API»: es el contrato de lo que tu código llama. No necesitas leer el código Go.
>
> **La tarea.** Eres el agente de la fase 6 de `specs/007-fases.md`: el frontend en `web/`. Este encargo es solo el **paso rojo**:
>
> 1. Monta el proyecto: `web/package.json` (scripts `dev`, `build` y `test`; `test` ejecuta Vitest una vez, sin modo observador), `web/vite.config.js` (plugins de Vue y Tailwind, `build.outDir` en `../server/static/dist` con `emptyOutDir`, proxy de `/api` a `http://localhost:8080`, y la configuración de Vitest con `jsdom`), `web/index.html`, `web/src/main.js`, la hoja de estilos con `@import "tailwindcss";`, `web/src/api.js` y los componentes `App.vue`, `LinkForm.vue` y `LinkList.vue`. Los componentes y `api.js` quedan **vacíos de comportamiento**: lo justo para que los tests se puedan importar y montar.
> 2. Instala las dependencias con `npm install` dentro de `web/` (solo las que permite la spec 001) y deja el `package-lock.json`.
> 3. Escribe los tests de WEB-01 a WEB-14 con Vitest y `@vue/test-utils`.
> 4. Ejecuta `npm test` en `web/` y comprueba que los tests **fallan por las aserciones**, no porque no se puedan importar o montar.
>
> **El método.** Tests de componentes que hacen lo que haría una persona: escribir en los campos, pulsar botones y mirar lo que se ve. `fetch`, `navigator.clipboard`, `window.confirm` y el reloj se simulan (`vi.fn`, `vi.useFakeTimers`). Comprueba las peticiones exactas (método, ruta y cuerpo) y los textos exactos de la spec. Cada test nombra el criterio que comprueba (`WEB-04`). No pruebes detalles de implementación (nombres de variables internas, clases CSS).
>
> **Los límites.** Solo puedes crear o modificar ficheros dentro de `web/`. No toques el código Go, el `Makefile`, `.gitignore` ni `server/static/`. Solo las dependencias npm de la spec 001. No hagas commits. No implementes todavía. `web/node_modules/` ya está en `.gitignore`.
>
> **Terminado es.** `npm test` se ejecuta y falla por las aserciones, y `npm run build` termina sin errores (el esqueleto tiene que compilar). El código Go no cambia: `go test ./...` sigue como estaba.
>
> **El informe.** Ficheros creados; versiones instaladas; cuántos tests hay y cuántos fallan; una tabla criterio → test; cualquier ambigüedad o contradicción de la spec, o cualquier cosa del contrato de la API que no encaje (no la resuelvas en silencio); y las últimas líneas de `npm test`.

## Informe: paso rojo

> **Paso rojo de la fase 6 hecho: 35 tests, los 35 fallan por aserciones, y `npm run build` compila.**
>
> Todos los fallos son `AssertionError`, como «no hay un `<label>` URL», «no hay botón Borrar» o «expected 'acorta' to contain …». No hay errores de importación ni de montaje. No se ha tocado nada fuera de `web/` ni se ha hecho ningún commit.
>
> **Ficheros creados (todo en `web/`)**
> - `package.json` (scripts `dev`, `build` y `test`, con `test` = `vitest run`) y `package-lock.json`.
> - `vite.config.js`: plugins de Vue y Tailwind, `outDir` en `../server/static/dist` con `emptyOutDir`, proxy de `/api` a `:8080` y Vitest con `jsdom`.
> - `index.html`, `src/main.js` y `src/style.css` (`@import "tailwindcss";`).
> - `src/api.js`: stubs de `listLinks`, `createLink` y `deleteLink`, que devuelven `{ok:false, data:null, errors:[]}`.
> - `src/App.vue`, `src/components/LinkForm.vue` y `src/components/LinkList.vue`: esqueletos vacíos. `App` solo pinta un `<h1>acorta</h1>` y monta los otros dos.
> - `src/App.test.js`: todos los tests, montando `App` entera.
>
> **Versiones instaladas:** vue 3.5.43, vite 8.3.2, @vitejs/plugin-vue 6.0.9, tailwindcss y @tailwindcss/vite 4.3.3, vitest 5.0.3, @vue/test-utils 2.5.1, jsdom 30.1.1. Son solo las dependencias que permite la spec 001.
>
> **Cómo están escritos los tests**
> - `fetch` se simula con un enrutador `método ruta`. Se comprueban método, ruta y cuerpo exactos.
> - El portapapeles y `confirm` se simulan, y el reloj con `vi.useFakeTimers` y `setSystemTime`.
> - Los campos se localizan por su `<label>` (URL, Alias, Caducidad) y los botones por su texto exacto. Esos textos son parte del contrato: Acortar, Copiar, Copiado, Borrar, Actualizar y Reintentar.
>
> | Criterio | Tests |
> |---|---|
> | WEB-01 | url sola; con alias; caducidad 1 hora, 1 día, 7 días y 30 días (`it.each`, 4 casos) |
> | WEB-02 | campos y botón deshabilitados en curso y de nuevo habilitados al terminar |
> | WEB-03 | URL corta con Copiar, formulario vacío, enlace el primero y una sola llamada GET |
> | WEB-04 | `it.each` con 400, 409 y 503: todos los errores, dentro de `aria-live="polite"`, y lo escrito se conserva |
> | WEB-05 | red caída; respuesta no JSON (un único mensaje en ambos) |
> | WEB-06 | editar URL, alias o caducidad borra los errores (3 casos); borra el resultado anterior; borra el mensaje de conexión |
> | WEB-07 | botón deshabilitado con URL vacía; `type`/`inputmode` del campo y opciones del desplegable |
> | WEB-08 | un GET y orden de llegada |
> | WEB-09 | URL corta con `target=_blank`, destino y plural de visitas; `Caduca el …`; `Caducado`; sin caducidad no muestra nada |
> | WEB-10 | mensaje de lista vacía; no aparece cuando hay enlaces |
> | WEB-11 | copia la URL de la fila correcta; «Copiado» 2 s con reloj simulado (1900 ms sigue, 2000 ms vuelve a Copiar) |
> | WEB-12 | cancelar no hace DELETE; confirmar hace DELETE y quita el enlace con 204; fallo 404 muestra el error y el enlace sigue |
> | WEB-13 | Actualizar repite el GET y muestra las visitas nuevas; no hay sondeo en 10 minutos |
> | WEB-14 | 500 → mensaje y Reintentar repite el GET; sin conexión → mensaje |
>
> **Ambigüedades de la spec que he tenido que decidir (a confirmar)**
> 1. **WEB-09, zona horaria.** `Caduca el 31/12/2026 23:59` no dice si es hora local o UTC. El test fija `process.env.TZ='UTC'` y usa `2026-12-31T23:59:59Z`, que da la misma salida en ambos casos. Conviene que la spec lo fije (probablemente local).
> 2. **Nombres de botones no dados por la spec.** WEB-14 pide «un botón para reintentar» y WEB-13 dice «Actualizar» sin más. Asumo `Reintentar` y `Actualizar`.
> 3. **WEB-03, «vacía el formulario».** Compruebo que URL y Alias quedan vacíos. No compruebo si Caducidad vuelve a «Nunca», porque la spec no lo dice.
> 4. **WEB-03, dos botones Copiar.** El resultado destacado y la fila nueva del listado llevan cada uno un botón Copiar, así que hay 2. El test asume ese reparto.
> 5. **WEB-09, «Caducado» frente a «Caduca el».** La spec no aclara si un enlace con `expired: true` debe omitir además el `Caduca el …`. Solo exijo que aparezca `Caducado`.
> 6. **WEB-04, región `aria-live`.** El test exige que los errores estén dentro de `[aria-live="polite"]`.
> 7. **Contrato de la API.** Nada que no encaje. La 204 de DELETE no tiene cuerpo, así que `api.js` no debe hacer `json()` ciegamente con ella.
> 8. **WEB-14 y errores de la API.** El test cubre un 500 con errores y un fallo de red, y ambos deben dar ese mismo mensaje.

## Revisión del orquestador

Comprobado: solo hay ficheros nuevos en `web/`; `npm test` ejecuta 35 tests y fallan los 35; los 14 criterios tienen su test; `server/static/dist/` está ignorado por Git; las dependencias son las de la spec 001.

Cinco de las ambigüedades acaban en la spec 006:

| # | Hueco | Cómo queda |
|---|---|---|
| 1 | ¿La fecha de caducidad se ve en hora local o en UTC? | Hora local del navegador, formato `dd/mm/aaaa hh:mm` (WEB-09) |
| 2 | Nombres de los botones | `Actualizar` y `Reintentar` (WEB-13, WEB-14) |
| 3 | ¿«Vaciar el formulario» incluye la caducidad? | Sí: vuelve a «Nunca» (WEB-03) |
| 4 | ¿Cuántos botones Copiar hay tras crear? | Dos: el del resultado destacado y el de la fila (WEB-03) |
| 5 | ¿Un enlace caducado enseña también su fecha? | No: solo la etiqueta `Caducado` (WEB-09) |

## Encargo: ajuste del rojo

> La spec 006 ha cambiado para cerrar las ambigüedades que encontraste. Vuelve a leer `specs/006-web.md`: han cambiado WEB-03 (la caducidad vuelve a «Nunca»; el resultado destacado tiene su propio Copiar; no se vuelve a pedir la lista), WEB-09 (hora local del navegador con formato `dd/mm/aaaa hh:mm`; un enlace caducado muestra solo `Caducado`, sin la fecha), WEB-12 (la `204` sin cuerpo no es un error de conexión), WEB-13 y WEB-14 (los botones se llaman `Actualizar` y `Reintentar`).
>
> Sigue siendo el **paso rojo**: ajusta solo los tests de `web/src/` para cubrir lo que ahora la spec sí dice. Para la hora local, haz que el test se ejecute con una zona horaria que no sea UTC (por ejemplo `Europe/Madrid`) y comprueba que la fecha mostrada es la local. Mismos límites: solo `web/`, sin implementar, sin commits.
>
> Informe: qué tests has añadido o cambiado, el recuento nuevo y si queda alguna ambigüedad.

## Informe: ajuste del rojo

> Ajuste del rojo hecho. Ahora hay 38 tests y los 38 fallan, todos por `AssertionError`: faltan etiquetas y botones, o no aparece el texto esperado. No hay errores de importación ni de montaje. `npm run build` compila. Solo he tocado `web/src/App.test.js`; no he implementado nada ni hecho commits.
>
> **Cambios en los tests**
> - **Zona horaria:** el fichero ahora fija `process.env.TZ = 'Europe/Madrid'` en lugar de UTC. El test de invierno comprueba que `new Date('2026-12-31T22:59:59Z').getHours()` vale 23, para detectar si el test corriera por error en UTC.
> - **WEB-03:** el test único se ha dividido en tres. Con el listado vacío, aparecen dos botones Copiar (resultado destacado y fila) y los dos copian la URL corta nueva. Tras crear, el formulario queda vacío: URL y alias en blanco y Caducidad otra vez en «Nunca». El enlace nuevo sale el primero del listado y hay una sola petición `GET /api/links`.
> - **WEB-03, error corregido:** mi versión anterior esperaba 2 botones Copiar con un enlace viejo ya en el listado. Habría dado 3 con una implementación correcta. Ahora el recuento se hace con el listado vacío.
> - **WEB-09, caducidad:** hora local en invierno (`2026-12-31T22:59:59Z` se muestra como `Caduca el 31/12/2026 23:59`); hora local en verano, test nuevo (`2026-07-01T07:05:00Z` se muestra como `Caduca el 01/07/2026 09:05`); enlace caducado: con `expired: true` aparece `Caducado`, y no aparecen ni «Caduca el» ni la fecha.
> - **Sin cambios** en WEB-12, WEB-13 ni WEB-14: ya cubrían lo que ahora dice la spec.
>
> **Recuento:** de 35 a 38 tests.
>
> **Ambigüedades:** ninguna pendiente.

Un detalle del informe que merece quedarse: al releer la spec, el agente encontró **un error en uno de sus propios tests** (esperaba 2 botones Copiar donde una implementación correcta daría 3). Un test equivocado en rojo es difícil de ver: falla igual que uno correcto. Se habría descubierto en el verde, como un test «imposible de pasar», y la tentación entonces es retorcer el código para satisfacerlo.

**Commits**: `Spec 006: aclaraciones que destapó el paso rojo` y `web: proyecto Vue y tests de la spec 006 (rojo)`. 38 tests, 38 en rojo.

## Encargo: paso verde

> Los tests del paso rojo ya están en un commit. Ahora, el **paso verde**: implementa la interfaz hasta que pase `npm test` en `web/`.
>
> **El método.** Lo que pide la spec 006 y nada más. **No modifiques `web/src/App.test.js`**: si crees que un test está mal o contradice la spec, párate y dilo en el informe. Sigue la estructura de la spec (`api.js`, `App.vue`, `LinkForm.vue`, `LinkList.vue`). La interfaz no contiene reglas de negocio: no valides URLs ni alias en el navegador; los errores son los que devuelve la API. Aplica la sección «Diseño»: mobile first con Tailwind, una columna, sin scroll horizontal a 360 px, controles de al menos 44 px de alto, cada campo con su `<label>`, tema claro y oscuro con `prefers-color-scheme`, y las URL largas cortadas sin desbordar. Cuida el aspecto: es la cara del proyecto en un tutorial. Cuando esté en verde, limpia el código sin cambiar el comportamiento.
>
> **Los límites.** Solo `web/`. Solo las dependencias de la spec 001. Sin commits.
>
> **Terminado es.** `npm test` en verde y `npm run build` sin errores ni avisos.
>
> **El informe.** Ficheros tocados; decisiones que has tomado y por qué; desviaciones de la spec; si has tocado algún test (no deberías); el tamaño del bundle que da el build; y la salida resumida de `npm test`.

## Informe: paso verde

> Paso verde casi cerrado: `npm test` da 37 de 38 en verde y `npm run build` termina sin errores ni avisos. El que falla es el test «WEB-03: al crear muestra la URL corta con su propio Copiar, además del de su fila», y creo que el error está en el test, no en el código. No he tocado `web/src/App.test.js` y no he hecho commits.
>
> **El test que falla.** Es el bucle `for (const i of [0, 1])` de ese test, en `web/src/App.test.js` (hacia la línea 204).
> - Hace clic en el Copiar de posición 0 y busca después `button(w, 'Copiar', 1)`.
> - Por WEB-11, el botón pulsado pasa a ser «Copiado» durante dos segundos, así que `buttons(w, 'Copiar')` solo devuelve ya 1 elemento y la posición 1 no existe.
> - La implementación copia la URL correcta con los dos botones y cumple WEB-03 y WEB-11. Cualquier implementación fiel a WEB-11 fallaría igual ahí.
> - Hace falta decidir cómo corregir el test. Por ejemplo, localizar los dos botones antes del bucle (`const bs = buttons(w,'Copiar')`) y pulsar `bs[0]` y `bs[1]`.
>
> **Ficheros tocados (todos en `web/src/`)**
> - `api.js`: `listLinks`, `createLink` y `deleteLink` devuelven `{ ok, data, errors }`. Un `204` es ok con `data` nulo. Un error de red o una respuesta que no es JSON da el único mensaje de conexión (WEB-05).
> - `App.vue`: guarda la lista, la carga y los errores de borrado. Pide la lista al montar y al pulsar Actualizar o Reintentar. Añade al principio el enlace creado sin volver a pedir la lista, y pide confirmación para borrar.
> - `components/LinkForm.vue`: formulario y resultado destacado. Los errores y el resultado se descartan al editar un campo. Si el envío tiene éxito, el formulario vuelve al estado inicial. No valida nada en el navegador.
> - `components/LinkList.vue`: listado, estados de vacío, de error y de carga, y errores de borrado.
> - `components/CopyButton.vue`: nuevo. Lo usan el formulario y cada fila del listado, y muestra «Copiado» dos segundos.
> - `style.css`: añade `color-scheme: light dark`.
>
> **Decisiones**
> - **Limpieza de errores:** uso `@input` y `@change` en vez de un `watch`. Un `watch` borraría el resultado justo al vaciar el formulario tras crear.
> - **Envío:** `expires_at` se calcula como `Date.now() + plazo` en UTC, sin milisegundos. El botón Acortar está deshabilitado con la URL vacía. También ignoro un segundo envío mientras hay una petición en curso.
> - **Región `aria-live`:** hay una en el formulario (errores y resultado) y otra en el listado (errores de borrado y de carga).
> - **Diseño:** Tailwind v4, una columna con `max-w-2xl`, controles de `min-h-11` (44 px) y cada campo con su `<label>`. Tema claro y oscuro con las variantes `dark:` de Tailwind. La URL corta usa `break-all` y la de destino `truncate`, para que nada desborde a 360 px.
>
> **Desviaciones de la spec:** ninguna. Añadí `CopyButton.vue`, que la spec no lista, solo para no duplicar el botón.
>
> **Bundle (`npm run build`)**: `index.html` 0,39 kB · CSS 14,45 kB (3,83 kB con gzip) · JS 73,39 kB (28,51 kB con gzip).
>
> **`npm test`:** `Tests 1 failed | 37 passed (38)`.

## Verificación del orquestador: un test equivocado

Este es el momento para el que existe la regla 3 de `AGENTS.md` («no modifiques un test para que pase; si crees que está mal, párate y dilo»). El agente tenía 37 de 38 en verde y dos atajos a mano: tocar el test, o retorcer el código para satisfacerlo (por ejemplo, que el botón no cambie a «Copiado» si hay otro, rompiendo WEB-11 de una forma que ningún otro test habría visto). No tomó ninguno. Paró y lo explicó.

El orquestador leyó el test y le dio la razón: el test pulsaba el primer botón Copiar y luego buscaba «el segundo botón que dice Copiar», pero el primero ya decía «Copiado». El test contradecía a WEB-11; ninguna implementación correcta podía pasarlo.

Quién corrige un test es una decisión, no un trámite: la corrección la hace el orquestador, es mínima (localizar los dos botones antes de pulsar) y va **en su propio commit**, separada de la implementación, para que en el historial se vea que el test cambió, por qué, y que no fue el implementador quien lo movió para llegar al verde.

Después:

- `npm test`: 38 de 38 en verde. `npm run build`: sin errores ni avisos.
- `git diff -- web/src/App.test.js`: solo las 6 líneas de la corrección.
- La spec 006 no listaba `CopyButton.vue`: se añade a su estructura en el mismo commit que el componente.
- La pasada en navegador real queda para la fase 7, cuando exista el servidor.

**Commits**: `web: corrige un test de WEB-03 que contradecía a WEB-11` y `web: interfaz Vue (verde)`.
