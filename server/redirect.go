package server

import (
	"errors"
	"net/http"

	"acorta/link"
)

// serveRedirect atiende GET y HEAD /{código}. HEAD no cuenta la visita.
func (s *server) serveRedirect(w http.ResponseWriter, r *http.Request, code string) {
	// Se comprueba el método antes de mirar el código: un POST es 405 exista o no.
	if allowReadOnly(w, r) {
		return
	}
	dest, err := s.svc.Resolve(r.Context(), code, r.Method == http.MethodGet)
	switch {
	case err == nil:
		w.Header().Set("Location", dest)
		// 302 y no-store: que el navegador no se salte el servidor (RED-01).
		w.Header().Set("Cache-Control", "no-store")
		w.WriteHeader(http.StatusFound)
	case errors.Is(err, link.ErrNotFound):
		writePlain(w, r, http.StatusNotFound, "enlace no encontrado")
	case errors.Is(err, link.ErrExpired):
		writePlain(w, r, http.StatusGone, "enlace caducado")
	default:
		writeInternal(w, r, err, false)
	}
}
