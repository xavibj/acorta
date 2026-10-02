// Cliente de la API (spec 004). Cada función devuelve { ok, status, data, errors }:
//  - respuesta 2xx: ok = true y data = el JSON (null si no hay cuerpo, como en el 204);
//  - respuesta de error de la API: ok = false y errors = los mensajes tal como llegan;
//  - sin conexión o respuesta que no es JSON: ok = false, status = 0 y un único mensaje de conexión.
// status permite distinguir el 401 de la spec 008 sin reinterpretar los mensajes.

export const CONNECTION_ERROR = 'No se ha podido conectar con el servidor.'

const connectionFailure = { ok: false, status: 0, data: null, errors: [CONNECTION_ERROR] }

// ---------- token de API (spec 008) ----------

// Clave de sessionStorage donde la interfaz guarda el token (AUT-11).
const TOKEN_KEY = 'acorta_token'

export function loadToken() {
  return sessionStorage.getItem(TOKEN_KEY) ?? ''
}

// Guarda el token; vacío borra la clave.
export function saveToken(token) {
  if (token === '') sessionStorage.removeItem(TOKEN_KEY)
  else sessionStorage.setItem(TOKEN_KEY, token)
}

// Cabecera Authorization para crear y borrar (AUT-12). Sin token guardado, ninguna.
function authHeaders() {
  const token = loadToken()
  return token === '' ? {} : { Authorization: `Bearer ${token}` }
}

// ---------- peticiones ----------

async function request(path, options) {
  let response
  try {
    response = await fetch(path, options)
  } catch {
    return connectionFailure
  }

  // El 204 no tiene cuerpo: no es un fallo de conexión.
  if (response.status === 204) return { ok: true, status: 204, data: null, errors: [] }

  let body
  try {
    body = await response.json()
  } catch {
    return connectionFailure
  }

  if (response.ok) return { ok: true, status: response.status, data: body, errors: [] }
  return {
    ok: false,
    status: response.status,
    data: null,
    errors: Array.isArray(body?.errors) ? body.errors : [],
  }
}

// Listar es público: nunca envía el token (AUT-12).
export function listLinks() {
  return request('/api/links')
}

export function createLink(payload) {
  return request('/api/links', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json', ...authHeaders() },
    body: JSON.stringify(payload),
  })
}

export function deleteLink(code) {
  return request(`/api/links/${encodeURIComponent(code)}`, {
    method: 'DELETE',
    headers: authHeaders(),
  })
}
