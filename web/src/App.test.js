import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { DOMWrapper, mount } from '@vue/test-utils'
import { nextTick } from 'vue'
import App from './App.vue'

// WEB-09: la caducidad se muestra en la hora local del navegador. Los tests
// corren en Europe/Madrid (UTC+1 en invierno, UTC+2 en verano), distinta de UTC,
// para que mostrar la hora UTC por error se detecte.
process.env.TZ = 'Europe/Madrid'

// ---------- ayudas ----------

function link(over = {}) {
  return {
    code: 'k3x9ab',
    url: 'https://example.com/una/ruta/larga',
    short_url: 'http://localhost:8080/k3x9ab',
    visits: 0,
    created_at: '2026-10-01T12:00:00Z',
    expires_at: null,
    expired: false,
    ...over,
  }
}

// Respuesta simulada de fetch. Sin cuerpo (204), json() falla como en la realidad.
function res(status, body) {
  return {
    ok: status >= 200 && status < 300,
    status,
    json: async () => {
      if (body === undefined) throw new SyntaxError('Unexpected end of JSON input')
      return body
    },
  }
}
const notJSON = () => ({ ok: true, status: 200, json: async () => { throw new SyntaxError('no es JSON') } })

// mockApi({ 'GET /api/links': [r1, r2], 'POST /api/links': r }) — cada valor es
// una respuesta, una función o una lista (se consume en orden; la última se repite).
// Lo que no se declara: GET /api/links → 200 [].
function mockApi(routes = {}) {
  const queues = {}
  for (const [k, v] of Object.entries(routes)) queues[k] = Array.isArray(v) ? [...v] : [v]
  const fn = vi.fn(async (url, opts = {}) => {
    const key = `${(opts.method || 'GET').toUpperCase()} ${url}`
    const q = queues[key]
    if (!q) {
      if (key === 'GET /api/links') return res(200, [])
      throw new Error(`petición no simulada: ${key}`)
    }
    const next = q.length > 1 ? q.shift() : q[0]
    const r = typeof next === 'function' ? next(opts) : next
    if (r instanceof Error) throw r
    return r
  })
  vi.stubGlobal('fetch', fn)
  return fn
}

// Peticiones realizadas, normalizadas: { method, path, body }.
function calls(fn) {
  return fn.mock.calls.map(([url, opts = {}]) => ({
    method: (opts.method || 'GET').toUpperCase(),
    path: String(url),
    body: opts.body ? JSON.parse(opts.body) : undefined,
  }))
}
const callsTo = (fn, method, path) => calls(fn).filter((c) => c.method === method && c.path === path)

async function settle() {
  for (let i = 0; i < 10; i++) await Promise.resolve()
  await nextTick()
  for (let i = 0; i < 10; i++) await Promise.resolve()
  await nextTick()
}

async function mountApp() {
  const wrapper = mount(App, { attachTo: document.body })
  await settle()
  return wrapper
}

// Campo localizado por su <label>, como lo haría una persona.
function field(wrapper, labelText) {
  const label = wrapper.findAll('label').find((l) => l.text().trim().startsWith(labelText))
  expect(label, `no hay un <label> «${labelText}»`).toBeTruthy()
  const control = label.element.control
  expect(control, `el <label> «${labelText}» no está asociado a ningún campo`).toBeTruthy()
  return new DOMWrapper(control)
}

const buttons = (wrapper, text) => wrapper.findAll('button').filter((b) => b.text().trim() === text)

function button(wrapper, text, index = 0) {
  const b = buttons(wrapper, text)[index]
  expect(b, `no hay un botón «${text}» (posición ${index})`).toBeTruthy()
  return b
}

function submitButton(wrapper) {
  const b = wrapper.find('button[type="submit"]')
  expect(b.exists(), 'no hay botón de envío').toBe(true)
  return b
}

async function fill(wrapper, { url, alias, expiry } = {}) {
  if (url !== undefined) await field(wrapper, 'URL').setValue(url)
  if (alias !== undefined) await field(wrapper, 'Alias').setValue(alias)
  if (expiry !== undefined) {
    const select = field(wrapper, 'Caducidad')
    const opt = select.findAll('option').find((o) => o.text().trim() === expiry)
    expect(opt, `no hay opción «${expiry}»`).toBeTruthy()
    await select.setValue(opt.element.value)
  }
}

async function submit(wrapper) {
  await submitButton(wrapper).trigger('click')
  await wrapper.find('form').trigger('submit')
  await settle()
}

const shortHrefs = (wrapper) => wrapper.findAll('a[target="_blank"]').map((a) => a.attributes('href'))

let confirmMock, writeText

beforeEach(() => {
  sessionStorage.clear() // AUT-11: el token guardado no debe filtrarse entre tests
  confirmMock = vi.fn(() => true)
  vi.stubGlobal('confirm', confirmMock)
  window.confirm = confirmMock
  writeText = vi.fn().mockResolvedValue(undefined)
  Object.defineProperty(navigator, 'clipboard', { value: { writeText }, configurable: true })
})

afterEach(() => {
  vi.useRealTimers()
  vi.unstubAllGlobals()
  document.body.innerHTML = ''
})

// ---------- formulario ----------

describe('formulario', () => {
  it('WEB-01: envía solo la url si no hay alias y la caducidad es Nunca', async () => {
    const fn = mockApi({ 'POST /api/links': res(201, link()) })
    const w = await mountApp()
    await fill(w, { url: 'https://example.com' })
    await submit(w)
    const posts = callsTo(fn, 'POST', '/api/links')
    expect(posts).toHaveLength(1)
    expect(posts[0].body).toEqual({ url: 'https://example.com' })
  })

  it('WEB-01: envía alias si se ha escrito', async () => {
    const fn = mockApi({ 'POST /api/links': res(201, link({ code: 'promo' })) })
    const w = await mountApp()
    await fill(w, { url: 'https://example.com', alias: 'promo' })
    await submit(w)
    expect(callsTo(fn, 'POST', '/api/links')[0].body).toEqual({ url: 'https://example.com', alias: 'promo' })
  })

  it.each([
    ['1 hora', '2026-10-01T13:00:00Z'],
    ['1 día', '2026-10-02T12:00:00Z'],
    ['7 días', '2026-10-08T12:00:00Z'],
    ['30 días', '2026-10-31T12:00:00Z'],
  ])('WEB-01: con caducidad «%s» envía expires_at = ahora + plazo (%s)', async (expiry, esperado) => {
    vi.useFakeTimers({ toFake: ['Date', 'setTimeout', 'clearTimeout'] })
    vi.setSystemTime(new Date('2026-10-01T12:00:00Z'))
    const fn = mockApi({ 'POST /api/links': res(201, link()) })
    const w = await mountApp()
    await fill(w, { url: 'https://example.com', expiry })
    await submit(w)
    expect(callsTo(fn, 'POST', '/api/links')[0].body).toEqual({ url: 'https://example.com', expires_at: esperado })
  })

  it('WEB-02: mientras la petición está en curso, botón y campos están deshabilitados', async () => {
    let resolve
    const pending = new Promise((r) => { resolve = r })
    mockApi({ 'POST /api/links': () => pending })
    const w = await mountApp()
    await fill(w, { url: 'https://example.com' })
    await submit(w)
    expect(submitButton(w).attributes('disabled')).toBeDefined()
    expect(field(w, 'URL').attributes('disabled')).toBeDefined()
    expect(field(w, 'Alias').attributes('disabled')).toBeDefined()
    expect(field(w, 'Caducidad').attributes('disabled')).toBeDefined()
    resolve(res(201, link()))
    await settle()
    expect(field(w, 'URL').attributes('disabled')).toBeUndefined()
    expect(field(w, 'Alias').attributes('disabled')).toBeUndefined()
    expect(field(w, 'Caducidad').attributes('disabled')).toBeUndefined()
  })

  it('WEB-03: al crear muestra la URL corta con su propio Copiar, además del de su fila', async () => {
    const nuevo = link({ code: 'nuevo1', short_url: 'http://localhost:8080/nuevo1' })
    mockApi({ 'GET /api/links': res(200, []), 'POST /api/links': res(201, nuevo) })
    const w = await mountApp()
    await fill(w, { url: 'https://example.com', alias: 'nuevo1' })
    await submit(w)
    expect(w.text()).toContain('http://localhost:8080/nuevo1')
    // resultado destacado + fila del listado
    // Se localizan los dos antes de pulsar: el pulsado pasa a decir «Copiado»
    // durante dos segundos (WEB-11) y dejaría de encontrarse por su texto.
    const copiar = buttons(w, 'Copiar')
    expect(copiar).toHaveLength(2)
    for (const boton of copiar) {
      writeText.mockClear()
      await boton.trigger('click')
      await settle()
      expect(writeText).toHaveBeenCalledWith('http://localhost:8080/nuevo1')
    }
  })

  it('WEB-03: vacía el formulario (URL y alias en blanco, caducidad otra vez en Nunca)', async () => {
    mockApi({ 'POST /api/links': res(201, link()) })
    const w = await mountApp()
    await fill(w, { url: 'https://example.com', alias: 'nuevo1', expiry: '7 días' })
    await submit(w)
    expect(field(w, 'URL').element.value).toBe('')
    expect(field(w, 'Alias').element.value).toBe('')
    expect(field(w, 'Caducidad').element.selectedOptions[0].textContent.trim()).toBe('Nunca')
  })

  it('WEB-03: el enlace nuevo aparece el primero del listado sin volver a pedir la lista', async () => {
    const viejo = link({ code: 'viejo', short_url: 'http://localhost:8080/viejo' })
    const nuevo = link({ code: 'nuevo1', short_url: 'http://localhost:8080/nuevo1' })
    const fn = mockApi({ 'GET /api/links': res(200, [viejo]), 'POST /api/links': res(201, nuevo) })
    const w = await mountApp()
    await fill(w, { url: 'https://example.com', alias: 'nuevo1' })
    await submit(w)
    expect(shortHrefs(w)).toEqual(['http://localhost:8080/nuevo1', 'http://localhost:8080/viejo'])
    expect(callsTo(fn, 'GET', '/api/links')).toHaveLength(1)
  })

  it.each([
    [400, ['url: debe empezar por http:// o https://', 'alias: debe tener entre 3 y 32 caracteres']],
    [409, ['alias: "promo" ya está en uso']],
    [503, ['no se ha podido generar un código libre; inténtalo de nuevo']],
  ])('WEB-04: con %i muestra todos los errores tal cual, en la región aria-live, y conserva lo escrito', async (status, errors) => {
    mockApi({ 'POST /api/links': res(status, { errors }) })
    const w = await mountApp()
    await fill(w, { url: 'ftp://x', alias: 'A', expiry: '7 días' })
    await submit(w)
    // La del formulario, no la primera del documento: desde la spec 008 hay
    // otra región aria-live encima, la del token.
    const live = w.find('form').element.parentElement.querySelector('[aria-live="polite"]')
    expect(live, 'falta la región aria-live="polite" del formulario').toBeTruthy()
    for (const e of errors) {
      expect(w.text()).toContain(e)
      expect(live.textContent).toContain(e)
    }
    expect(field(w, 'URL').element.value).toBe('ftp://x')
    expect(field(w, 'Alias').element.value).toBe('A')
    expect(field(w, 'Caducidad').element.selectedOptions[0].textContent.trim()).toBe('7 días')
    expect(buttons(w, 'Copiar')).toHaveLength(0)
  })

  it('WEB-05: si la petición no llega al servidor muestra un único mensaje', async () => {
    mockApi({ 'POST /api/links': new TypeError('Failed to fetch') })
    const w = await mountApp()
    await fill(w, { url: 'https://example.com' })
    await submit(w)
    const aparece = w.text().split('No se ha podido conectar con el servidor.').length - 1
    expect(aparece).toBe(1)
  })

  it('WEB-05: si la respuesta no es JSON muestra un único mensaje', async () => {
    mockApi({ 'POST /api/links': notJSON() })
    const w = await mountApp()
    await fill(w, { url: 'https://example.com' })
    await submit(w)
    const aparece = w.text().split('No se ha podido conectar con el servidor.').length - 1
    expect(aparece).toBe(1)
  })

  it.each([
    ['la URL', (w) => fill(w, { url: 'https://otra.com' })],
    ['el alias', (w) => fill(w, { alias: 'otro' })],
    ['la caducidad', (w) => fill(w, { expiry: '1 hora' })],
  ])('WEB-06: al editar %s desaparecen los errores', async (_, editar) => {
    mockApi({ 'POST /api/links': res(400, { errors: ['url: debe empezar por http:// o https://'] }) })
    const w = await mountApp()
    await fill(w, { url: 'ftp://x' })
    await submit(w)
    expect(w.text()).toContain('url: debe empezar por http:// o https://')
    await editar(w)
    expect(w.text()).not.toContain('url: debe empezar por http:// o https://')
  })

  it('WEB-06: al editar un campo desaparece el resultado del envío anterior', async () => {
    mockApi({ 'POST /api/links': res(201, link({ short_url: 'http://localhost:8080/nuevo1', code: 'nuevo1' })) })
    const w = await mountApp()
    await fill(w, { url: 'https://example.com' })
    await submit(w)
    expect(buttons(w, 'Copiar')).toHaveLength(2) // resultado + fila del listado
    await fill(w, { url: 'https://otra.com' })
    expect(buttons(w, 'Copiar')).toHaveLength(1) // queda solo el del listado
  })

  it('WEB-06: al editar un campo desaparece el mensaje de conexión', async () => {
    mockApi({ 'POST /api/links': new TypeError('Failed to fetch') })
    const w = await mountApp()
    await fill(w, { url: 'https://example.com' })
    await submit(w)
    expect(w.text()).toContain('No se ha podido conectar con el servidor.')
    await fill(w, { alias: 'x' })
    expect(w.text()).not.toContain('No se ha podido conectar con el servidor.')
  })

  it('WEB-07: el botón Acortar está deshabilitado mientras la URL esté vacía', async () => {
    mockApi()
    const w = await mountApp()
    expect(submitButton(w).text().trim()).toBe('Acortar')
    expect(submitButton(w).attributes('disabled')).toBeDefined()
    await fill(w, { url: 'https://example.com' })
    expect(submitButton(w).attributes('disabled')).toBeUndefined()
    await fill(w, { url: '' })
    expect(submitButton(w).attributes('disabled')).toBeDefined()
  })

  it('WEB-07: el campo URL es de texto con inputmode url y las opciones de caducidad son las de la spec', async () => {
    mockApi()
    const w = await mountApp()
    const url = field(w, 'URL')
    expect(url.attributes('type')).toBe('text')
    expect(url.attributes('inputmode')).toBe('url')
    const opciones = field(w, 'Caducidad').findAll('option').map((o) => o.text().trim())
    expect(opciones).toEqual(['Nunca', '1 hora', '1 día', '7 días', '30 días'])
  })
})

// ---------- listado ----------

describe('listado', () => {
  it('WEB-08: al cargar hace GET /api/links y muestra los enlaces en el orden que llegan', async () => {
    const fn = mockApi({
      'GET /api/links': res(200, [
        link({ code: 'ccc', short_url: 'http://localhost:8080/ccc' }),
        link({ code: 'aaa', short_url: 'http://localhost:8080/aaa' }),
        link({ code: 'bbb', short_url: 'http://localhost:8080/bbb' }),
      ]),
    })
    const w = await mountApp()
    expect(callsTo(fn, 'GET', '/api/links')).toHaveLength(1)
    expect(shortHrefs(w)).toEqual([
      'http://localhost:8080/ccc',
      'http://localhost:8080/aaa',
      'http://localhost:8080/bbb',
    ])
  })

  it('WEB-09: cada enlace muestra URL corta (en otra pestaña), destino y visitas con su plural', async () => {
    mockApi({
      'GET /api/links': res(200, [
        link({ code: 'cero', short_url: 'http://localhost:8080/cero', url: 'https://a.example/cero', visits: 0 }),
        link({ code: 'uno', short_url: 'http://localhost:8080/uno', url: 'https://a.example/uno', visits: 1 }),
        link({ code: 'dos', short_url: 'http://localhost:8080/dos', url: 'https://a.example/dos', visits: 2 }),
      ]),
    })
    const w = await mountApp()
    const a = w.find('a[href="http://localhost:8080/uno"]')
    expect(a.exists()).toBe(true)
    expect(a.attributes('target')).toBe('_blank')
    for (const u of ['https://a.example/cero', 'https://a.example/uno', 'https://a.example/dos']) {
      expect(w.text()).toContain(u)
    }
    expect(w.text()).toContain('0 visitas')
    expect(w.text()).toMatch(/\b1 visita\b(?!s)/)
    expect(w.text()).toContain('2 visitas')
  })

  it('WEB-09: la caducidad se muestra en hora local (invierno, UTC+1)', async () => {
    mockApi({ 'GET /api/links': res(200, [link({ expires_at: '2026-12-31T22:59:59Z' })]) })
    const w = await mountApp()
    expect(new Date('2026-12-31T22:59:59Z').getHours(), 'el test debe correr fuera de UTC').toBe(23)
    expect(w.text()).toContain('Caduca el 31/12/2026 23:59')
    expect(w.text()).not.toContain('Caducado')
  })

  it('WEB-09: la caducidad se muestra en hora local (verano, UTC+2) con formato dd/mm/aaaa hh:mm', async () => {
    mockApi({ 'GET /api/links': res(200, [link({ expires_at: '2026-07-01T07:05:00Z' })]) })
    const w = await mountApp()
    expect(w.text()).toContain('Caduca el 01/07/2026 09:05')
  })

  it('WEB-09: un enlace caducado muestra solo «Caducado», sin la fecha', async () => {
    mockApi({ 'GET /api/links': res(200, [link({ expires_at: '2026-01-01T10:30:00Z', expired: true })]) })
    const w = await mountApp()
    expect(w.text()).toContain('Caducado')
    expect(w.text()).not.toContain('Caduca el')
    expect(w.text()).not.toContain('01/01/2026')
  })

  it('WEB-09: un enlace sin caducidad no muestra nada de caducidad', async () => {
    mockApi({ 'GET /api/links': res(200, [link()]) })
    const w = await mountApp()
    expect(shortHrefs(w)).toHaveLength(1) // el listado sí se ha pintado
    expect(w.text()).not.toMatch(/Caduca el|Caducado/)
  })

  it('WEB-10: sin enlaces muestra el mensaje de listado vacío', async () => {
    mockApi({ 'GET /api/links': res(200, []) })
    const w = await mountApp()
    expect(w.text()).toContain('Aún no hay enlaces. Crea el primero arriba.')
  })

  it('WEB-10: con enlaces no muestra el mensaje de listado vacío', async () => {
    mockApi({ 'GET /api/links': res(200, [link()]) })
    const w = await mountApp()
    expect(shortHrefs(w)).toHaveLength(1)
    expect(w.text()).not.toContain('Aún no hay enlaces.')
  })

  it('WEB-11: Copiar copia la URL corta y el botón muestra «Copiado» dos segundos', async () => {
    vi.useFakeTimers({ toFake: ['Date', 'setTimeout', 'clearTimeout'] })
    mockApi({
      'GET /api/links': res(200, [
        link({ code: 'uno', short_url: 'http://localhost:8080/uno' }),
        link({ code: 'dos', short_url: 'http://localhost:8080/dos' }),
      ]),
    })
    const w = await mountApp()
    await button(w, 'Copiar', 1).trigger('click')
    await settle()
    expect(writeText).toHaveBeenCalledTimes(1)
    expect(writeText).toHaveBeenCalledWith('http://localhost:8080/dos')
    expect(buttons(w, 'Copiado')).toHaveLength(1)
    vi.advanceTimersByTime(1900)
    await settle()
    expect(buttons(w, 'Copiado')).toHaveLength(1)
    vi.advanceTimersByTime(100)
    await settle()
    expect(buttons(w, 'Copiado')).toHaveLength(0)
    expect(buttons(w, 'Copiar')).toHaveLength(2)
  })

  it('WEB-12: Borrar pide confirmación «¿Borrar /promo?» y, si se cancela, no hace nada', async () => {
    const fn = mockApi({ 'GET /api/links': res(200, [link({ code: 'promo', short_url: 'http://localhost:8080/promo' })]) })
    confirmMock.mockReturnValue(false)
    const w = await mountApp()
    await button(w, 'Borrar').trigger('click')
    await settle()
    expect(confirmMock).toHaveBeenCalledWith('¿Borrar /promo?')
    expect(callsTo(fn, 'DELETE', '/api/links/promo')).toHaveLength(0)
    expect(shortHrefs(w)).toHaveLength(1)
  })

  it('WEB-12: si se confirma hace DELETE y, con 204, quita el enlace del listado', async () => {
    const fn = mockApi({
      'GET /api/links': res(200, [
        link({ code: 'promo', short_url: 'http://localhost:8080/promo' }),
        link({ code: 'otro', short_url: 'http://localhost:8080/otro' }),
      ]),
      'DELETE /api/links/promo': res(204),
    })
    const w = await mountApp()
    await button(w, 'Borrar', 0).trigger('click')
    await settle()
    expect(confirmMock).toHaveBeenCalledWith('¿Borrar /promo?')
    expect(callsTo(fn, 'DELETE', '/api/links/promo')).toHaveLength(1)
    expect(shortHrefs(w)).toEqual(['http://localhost:8080/otro'])
  })

  it('WEB-12: si el borrado falla muestra el error y el enlace sigue en el listado', async () => {
    mockApi({
      'GET /api/links': res(200, [link({ code: 'promo', short_url: 'http://localhost:8080/promo' })]),
      'DELETE /api/links/promo': res(404, { errors: ['enlace no encontrado'] }),
    })
    const w = await mountApp()
    await button(w, 'Borrar').trigger('click')
    await settle()
    expect(w.text()).toContain('enlace no encontrado')
    expect(shortHrefs(w)).toEqual(['http://localhost:8080/promo'])
  })

  it('WEB-13: Actualizar vuelve a pedir el listado y muestra las visitas nuevas', async () => {
    const fn = mockApi({
      'GET /api/links': [
        res(200, [link({ visits: 0 })]),
        res(200, [link({ visits: 5 })]),
      ],
    })
    const w = await mountApp()
    expect(w.text()).toContain('0 visitas')
    await button(w, 'Actualizar').trigger('click')
    await settle()
    expect(callsTo(fn, 'GET', '/api/links')).toHaveLength(2)
    expect(w.text()).toContain('5 visitas')
    expect(w.text()).not.toContain('0 visitas')
  })

  it('WEB-13: no hay sondeo automático', async () => {
    vi.useFakeTimers({ toFake: ['Date', 'setTimeout', 'clearTimeout', 'setInterval', 'clearInterval'] })
    const fn = mockApi()
    await mountApp()
    vi.advanceTimersByTime(10 * 60 * 1000)
    await settle()
    expect(callsTo(fn, 'GET', '/api/links')).toHaveLength(1)
  })

  it('WEB-14: si falla la carga muestra el mensaje y un botón para reintentar', async () => {
    const fn = mockApi({
      'GET /api/links': [res(500, { errors: ['error interno'] }), res(200, [link()])],
    })
    const w = await mountApp()
    expect(w.text()).toContain('No se ha podido cargar la lista de enlaces.')
    expect(w.text()).not.toContain('Aún no hay enlaces.')
    await button(w, 'Reintentar').trigger('click')
    await settle()
    expect(callsTo(fn, 'GET', '/api/links')).toHaveLength(2)
    expect(w.text()).not.toContain('No se ha podido cargar la lista de enlaces.')
    expect(shortHrefs(w)).toHaveLength(1)
  })

  it('WEB-14: también falla la carga si no hay conexión', async () => {
    mockApi({ 'GET /api/links': new TypeError('Failed to fetch') })
    const w = await mountApp()
    expect(w.text()).toContain('No se ha podido cargar la lista de enlaces.')
  })
})

// ---------- autenticación (spec 008) ----------

const TOKEN_KEY = 'acorta_token'
const TOKEN = 'token-de-prueba-123456' // 22 caracteres: cumple el mínimo de 16 de la spec 008

// Valor de una cabecera de una llamada a fetch, sin distinguir mayúsculas en el
// nombre. undefined si la petición no la lleva (o no lleva cabeceras).
function headerOf([, opts = {}], name) {
  const h = opts.headers
  if (!h) return undefined
  if (typeof h.get === 'function') return h.get(name) ?? undefined
  const key = Object.keys(h).find((k) => k.toLowerCase() === name.toLowerCase())
  return key === undefined ? undefined : h[key]
}

// Cabeceras Authorization de todas las peticiones a method+path, en orden.
function authHeaders(fn, method, path) {
  return fn.mock.calls
    .filter(([url, opts = {}]) => (opts.method || 'GET').toUpperCase() === method && String(url) === path)
    .map((c) => headerOf(c, 'Authorization'))
}

const tokenField = (wrapper) => field(wrapper, 'Token de API')

async function saveToken(wrapper, value) {
  await tokenField(wrapper).setValue(value)
  await button(wrapper, 'Guardar').trigger('click')
  await settle()
}

const expectTokenFocused = (wrapper) =>
  expect(document.activeElement, 'el campo Token de API no tiene el foco').toBe(tokenField(wrapper).element)

describe('autenticación', () => {
  it('AUT-11: el campo Token de API es type="password", tiene su <label> y va encima del formulario de creación', async () => {
    mockApi()
    const w = await mountApp()
    const token = tokenField(w)
    expect(token.attributes('type')).toBe('password')
    expect(button(w, 'Guardar').exists()).toBe(true)
    // encima del formulario: el campo precede en el documento al campo URL
    const antes = token.element.compareDocumentPosition(field(w, 'URL').element)
    expect(antes & Node.DOCUMENT_POSITION_FOLLOWING, 'el token no va encima del formulario').toBeTruthy()
  })

  it('AUT-11: al pulsar Guardar guarda el token recortado bajo acorta_token y muestra «Token guardado»', async () => {
    mockApi()
    const w = await mountApp()
    expect(w.text()).not.toContain('Token guardado')
    await saveToken(w, `  ${TOKEN}  `)
    expect(sessionStorage.getItem(TOKEN_KEY)).toBe(TOKEN)
    expect(w.text()).toContain('Token guardado')
    // es un mensaje de estado: va en una región aria-live="polite" (spec 006, Diseño)
    const live = w.findAll('[aria-live="polite"]').find((r) => r.text().includes('Token guardado'))
    expect(live, 'el estado «Token guardado» no está en una región aria-live="polite"').toBeTruthy()
  })

  it('AUT-11: Enter dentro del campo guarda el token igual que el botón Guardar', async () => {
    mockApi()
    const w = await mountApp()
    await tokenField(w).setValue(`  ${TOKEN}  `)
    await tokenField(w).trigger('keydown', { key: 'Enter' })
    await tokenField(w).trigger('keyup', { key: 'Enter' })
    await settle()
    expect(sessionStorage.getItem(TOKEN_KEY)).toBe(TOKEN)
    expect(w.text()).toContain('Token guardado')
  })

  it('AUT-11: tras guardar, el campo pasa a mostrar el valor recortado', async () => {
    mockApi()
    const w = await mountApp()
    await saveToken(w, '  token-con-espacios-1234  ')
    expect(sessionStorage.getItem(TOKEN_KEY)).toBe('token-con-espacios-1234')
    expect(tokenField(w).element.value).toBe('token-con-espacios-1234')
  })

  it('AUT-11: el campo del token no contiene ni está dentro de un <form>; el único formulario es el de creación', async () => {
    mockApi()
    const w = await mountApp()
    expect(tokenField(w).element.closest('form'), 'el campo del token está dentro de un <form>').toBeNull()
    expect(button(w, 'Guardar').element.closest('form'), 'el botón Guardar está dentro de un <form>').toBeNull()
    const forms = w.findAll('form')
    expect(forms).toHaveLength(1)
    expect(forms[0].find('button[type="submit"]').text().trim()).toBe('Acortar')
  })

  it('AUT-11: al cargar con un token guardado, el campo aparece relleno y en estado «Token guardado»', async () => {
    sessionStorage.setItem(TOKEN_KEY, TOKEN)
    mockApi()
    const w = await mountApp()
    expect(tokenField(w).element.value).toBe(TOKEN)
    expect(w.text()).toContain('Token guardado')
  })

  it('AUT-11: al cargar sin token guardado, el campo está vacío y no muestra «Token guardado»', async () => {
    mockApi()
    const w = await mountApp()
    expect(tokenField(w).element.value).toBe('')
    expect(w.text()).not.toContain('Token guardado')
  })

  it.each([
    ['vacío', ''],
    ['solo espacios', '   '],
  ])('AUT-11: guardar un campo %s borra el token guardado', async (_, value) => {
    sessionStorage.setItem(TOKEN_KEY, TOKEN)
    mockApi()
    const w = await mountApp()
    expect(w.text()).toContain('Token guardado')
    await saveToken(w, value)
    expect(sessionStorage.getItem(TOKEN_KEY)).toBeNull()
    expect(w.text()).not.toContain('Token guardado')
  })

  it('AUT-12: crear (WEB-01) envía Authorization: Bearer <token guardado>', async () => {
    sessionStorage.setItem(TOKEN_KEY, TOKEN)
    const fn = mockApi({ 'POST /api/links': res(201, link()) })
    const w = await mountApp()
    await fill(w, { url: 'https://example.com' })
    await submit(w)
    expect(authHeaders(fn, 'POST', '/api/links')).toEqual([`Bearer ${TOKEN}`])
    // y sigue enviando el cuerpo de WEB-01 como JSON
    expect(callsTo(fn, 'POST', '/api/links')[0].body).toEqual({ url: 'https://example.com' })
    expect(headerOf(fn.mock.calls.find(([, o = {}]) => o.method === 'POST'), 'Content-Type')).toBe('application/json')
  })

  it('AUT-12: crear usa el token guardado con Guardar en la misma sesión, sin recargar', async () => {
    const fn = mockApi({ 'POST /api/links': res(201, link()) })
    const w = await mountApp()
    await saveToken(w, TOKEN)
    await fill(w, { url: 'https://example.com' })
    await submit(w)
    expect(authHeaders(fn, 'POST', '/api/links')).toEqual([`Bearer ${TOKEN}`])
  })

  it('AUT-12: crear sin token guardado no envía la cabecera Authorization', async () => {
    const fn = mockApi({ 'POST /api/links': res(201, link()) })
    const w = await mountApp()
    await fill(w, { url: 'https://example.com' })
    await submit(w)
    expect(authHeaders(fn, 'POST', '/api/links')).toEqual([undefined])
  })

  it('AUT-12: tras borrar el token con Guardar vacío, crear ya no envía la cabecera', async () => {
    sessionStorage.setItem(TOKEN_KEY, TOKEN)
    const fn = mockApi({ 'POST /api/links': res(201, link()) })
    const w = await mountApp()
    await saveToken(w, '')
    await fill(w, { url: 'https://example.com' })
    await submit(w)
    expect(authHeaders(fn, 'POST', '/api/links')).toEqual([undefined])
  })

  it('AUT-12: borrar (WEB-12) envía Authorization: Bearer <token guardado>', async () => {
    sessionStorage.setItem(TOKEN_KEY, TOKEN)
    const fn = mockApi({
      'GET /api/links': res(200, [link({ code: 'promo', short_url: 'http://localhost:8080/promo' })]),
      'DELETE /api/links/promo': res(204),
    })
    const w = await mountApp()
    await button(w, 'Borrar').trigger('click')
    await settle()
    expect(authHeaders(fn, 'DELETE', '/api/links/promo')).toEqual([`Bearer ${TOKEN}`])
    expect(shortHrefs(w)).toEqual([])
  })

  it('AUT-12: borrar sin token guardado no envía la cabecera Authorization', async () => {
    const fn = mockApi({
      'GET /api/links': res(200, [link({ code: 'promo', short_url: 'http://localhost:8080/promo' })]),
      'DELETE /api/links/promo': res(204),
    })
    const w = await mountApp()
    await button(w, 'Borrar').trigger('click')
    await settle()
    expect(authHeaders(fn, 'DELETE', '/api/links/promo')).toEqual([undefined])
  })

  it.each([
    ['con token guardado', true],
    ['sin token guardado', false],
  ])('AUT-12: cargar el listado (WEB-08) y Actualizar (WEB-13) no envían Authorization %s', async (_, saved) => {
    if (saved) sessionStorage.setItem(TOKEN_KEY, TOKEN)
    const fn = mockApi({ 'GET /api/links': res(200, [link()]) })
    const w = await mountApp()
    await button(w, 'Actualizar').trigger('click')
    await settle()
    expect(authHeaders(fn, 'GET', '/api/links')).toEqual([undefined, undefined])
  })

  it('AUT-13: un 401 al crear sin token muestra «falta el token de API» tal cual y el campo del token recibe el foco', async () => {
    mockApi({ 'POST /api/links': res(401, { errors: ['falta el token de API'] }) })
    const w = await mountApp()
    await fill(w, { url: 'https://example.com', alias: 'promo' })
    await submit(w)
    expect(w.text()).toContain('falta el token de API')
    expectTokenFocused(w)
    // como cualquier otro error de la API (WEB-04): conserva lo escrito y no hay resultado
    expect(field(w, 'URL').element.value).toBe('https://example.com')
    expect(field(w, 'Alias').element.value).toBe('promo')
    expect(buttons(w, 'Copiar')).toHaveLength(0)
    expect(sessionStorage.getItem(TOKEN_KEY)).toBeNull()
  })

  it('AUT-13: un 401 al crear con token guardado muestra el error tal cual, da el foco al token y no lo borra', async () => {
    sessionStorage.setItem(TOKEN_KEY, TOKEN)
    mockApi({ 'POST /api/links': res(401, { errors: ['token de API incorrecto'] }) })
    const w = await mountApp()
    await fill(w, { url: 'https://example.com' })
    await submit(w)
    expect(w.text()).toContain('token de API incorrecto')
    expectTokenFocused(w)
    expect(sessionStorage.getItem(TOKEN_KEY)).toBe(TOKEN)
    expect(tokenField(w).element.value).toBe(TOKEN)
  })

  it('AUT-13: un 401 al borrar muestra el error tal cual, el enlace sigue en el listado, el token recibe el foco y no se borra', async () => {
    sessionStorage.setItem(TOKEN_KEY, TOKEN)
    mockApi({
      'GET /api/links': res(200, [link({ code: 'promo', short_url: 'http://localhost:8080/promo' })]),
      'DELETE /api/links/promo': res(401, { errors: ['falta el token de API'] }),
    })
    const w = await mountApp()
    await button(w, 'Borrar').trigger('click')
    await settle()
    expect(w.text()).toContain('falta el token de API')
    expect(shortHrefs(w)).toEqual(['http://localhost:8080/promo'])
    expectTokenFocused(w)
    expect(sessionStorage.getItem(TOKEN_KEY)).toBe(TOKEN)
  })

  it('AUT-13: los demás errores al crear (400) no mueven el foco al campo del token', async () => {
    sessionStorage.setItem(TOKEN_KEY, TOKEN)
    mockApi({ 'POST /api/links': res(400, { errors: ['url: debe empezar por http:// o https://'] }) })
    const w = await mountApp()
    await fill(w, { url: 'ftp://x' })
    await submit(w)
    expect(w.text()).toContain('url: debe empezar por http:// o https://')
    expect(document.activeElement).not.toBe(tokenField(w).element)
  })
})
