# 009 — La regla que faltaba: preguntar antes

**Fecha**: 2026-10-02 · **Cambia**: `AGENTS.md` (regla 10)

acorta empezó con una entrevista de cuatro preguntas (`001-entrevista.md`) y con una lista de veintiuna decisiones por defecto. Funcionó bien. Pero en `AGENTS.md` no había ninguna regla que lo pidiera.

Lo destapó Xavi al leer el capítulo del tutorial que cuenta esa entrevista:

> quien le pide o de donde saca la IA que te tiene que preguntar y cuantas preguntas?

La respuesta honesta es que no salió del proyecto. Salió de tres sitios ajenos a él:

- **La metodología de Xavi**, escrita en otro sitio: primero la spec, y pedir confirmación si algo altera el producto.
- **La herramienta.** Claude Code le da al agente una herramienta para preguntar, que admite como mucho cuatro preguntas por tanda, cada una con entre dos y cuatro opciones, y pide marcar la recomendada. Que fueran cuatro preguntas con opciones y recomendación es, en buena parte, el formato de esa herramienta.
- **El criterio del agente**, solo para elegir cuáles cuatro.

Es decir: con otro agente, o con el mismo sin esa herramienta, el proyecto habría empezado de otra manera, y nada en el repositorio lo habría impedido. Es el mismo fallo que el tutorial critica: una decisión importante que no estaba escrita en ningún sitio.

## Qué cambia

`AGENTS.md` gana la regla 10: algo nuevo empieza por preguntas, como mucho cinco, solo las que cambien qué se construye, con opciones y recomendación; lo demás lo decide el agente, lo escribe en la spec y lo enumera en `bitacora/`; y no se implementa sin visto bueno.

Va al final, con el número 10, para no renumerar las anteriores: las bitácoras y el tutorial citan «la regla 3».

## La comprobación

El 2 de octubre de 2026 se puso un `AGENTS.md` con esta regla en una carpeta vacía y se le dio a dos agentes, sin ninguna otra instrucción, el encargo de una línea del capítulo 1 del tutorial, el mismo con el que once agentes se habían puesto a programar:

> Hazme un acortador de URLs en JavaScript: un módulo sin dependencias que guarde los enlaces en memoria.

| Agente | Qué hizo |
|---|---|
| Codex, GPT-6.1-Sol | Cuatro preguntas con su recomendación. Ningún fichero de código. Se paró: «pediré tu revisión antes de implementar, como exige la regla 5 de AGENTS.md» |
| pi, deepseek-v4-pro | Cinco preguntas con opciones y la recomendada marcada, más la lista de lo que decidiría por defecto. Ningún fichero de código. Se paró |

Una regla de seis líneas en un fichero que el agente lee siempre hizo lo que no hizo ninguno de los once encargos sin ella.
