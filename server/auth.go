package server

import (
	"crypto/subtle"
	"net/http"
	"strings"
)

// authorized comprueba la cabecera Authorization de una ruta protegida
// (POST /api/links y DELETE /api/links/{código}, spec 008). Si falta o no
// trae el token correcto, responde 401 y devuelve false.
//
// Se llama después de comprobar el método (el 405 de API-12 va antes) y
// antes de leer el cuerpo o buscar el código (AUT-05).
func (s *server) authorized(w http.ResponseWriter, r *http.Request) bool {
	auth := r.Header.Get("Authorization")
	if auth == "" {
		writeUnauthorized(w, r, "falta el token de API")
		return false
	}
	// La cabecera es exactamente el esquema, un espacio y el token; no se
	// recorta nada, así que un espacio de más deja un token distinto (AUT-03).
	scheme, token, _ := strings.Cut(auth, " ")
	if !strings.EqualFold(scheme, "Bearer") || !s.tokenMatches(token) {
		writeUnauthorized(w, r, "token de API incorrecto")
		return false
	}
	return true
}

// tokenMatches compara en tiempo constante, para no filtrar por el tiempo de
// respuesta cuántos caracteres coinciden. Con el token del servidor vacío no
// acepta nada: ConstantTimeCompare da por iguales dos cadenas vacías y la
// spec dice que un token vacío no significa «sin autenticación».
func (s *server) tokenMatches(got string) bool {
	return s.token != "" && subtle.ConstantTimeCompare([]byte(got), []byte(s.token)) == 1
}
