// Cliente de la API (spec 004). Cada función devuelve { ok, data, errors }:
//  - respuesta 2xx: ok = true y data = el JSON (null si no hay cuerpo, como en el 204);
//  - respuesta de error de la API: ok = false y errors = los mensajes tal como llegan;
//  - sin conexión o respuesta que no es JSON: ok = false y un único mensaje de conexión.

export const CONNECTION_ERROR = 'No se ha podido conectar con el servidor.'

const connectionFailure = { ok: false, data: null, errors: [CONNECTION_ERROR] }

async function request(path, options) {
  let response
  try {
    response = await fetch(path, options)
  } catch {
    return connectionFailure
  }

  // El 204 no tiene cuerpo: no es un fallo de conexión.
  if (response.status === 204) return { ok: true, data: null, errors: [] }

  let body
  try {
    body = await response.json()
  } catch {
    return connectionFailure
  }

  if (response.ok) return { ok: true, data: body, errors: [] }
  return { ok: false, data: null, errors: Array.isArray(body?.errors) ? body.errors : [] }
}

export function listLinks() {
  return request('/api/links')
}

export function createLink(payload) {
  return request('/api/links', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(payload),
  })
}

export function deleteLink(code) {
  return request(`/api/links/${encodeURIComponent(code)}`, { method: 'DELETE' })
}
