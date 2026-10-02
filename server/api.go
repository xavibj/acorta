package server

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"acorta/link"
)

// maxBody es el tamaño máximo del cuerpo de POST /api/links (API-06).
const maxBody = 64 << 10

// linkJSON es el enlace tal como sale en la API.
type linkJSON struct {
	Code      string  `json:"code"`
	URL       string  `json:"url"`
	ShortURL  string  `json:"short_url"`
	Visits    int64   `json:"visits"`
	CreatedAt string  `json:"created_at"`
	ExpiresAt *string `json:"expires_at"` // null si no caduca
	Expired   bool    `json:"expired"`
}

func (s *server) toJSON(l link.Link) linkJSON {
	j := linkJSON{
		Code:      l.Code,
		URL:       l.URL,
		ShortURL:  s.baseURL + "/" + l.Code,
		Visits:    l.Visits,
		CreatedAt: l.CreatedAt.UTC().Format(time.RFC3339),
		// Se calcula al responder, con el reloj del servicio (LIS-02).
		Expired: l.Expired(s.svc.Now()),
	}
	if l.ExpiresAt != nil {
		e := l.ExpiresAt.UTC().Format(time.RFC3339)
		j.ExpiresAt = &e
	}
	return j
}

// serveAPI reparte las rutas de /api/. Lo hace a mano para poder responder
// 404 y 405 en JSON y con el Allow exacto de cada ruta (API-11, API-12).
func (s *server) serveAPI(w http.ResponseWriter, r *http.Request) {
	rest := strings.TrimPrefix(r.URL.Path, "/api/")
	switch {
	case rest == "healthz":
		if methodNotAllowed(w, r, "GET, HEAD", http.MethodGet, http.MethodHead) {
			return
		}
		writeJSON(w, r, http.StatusOK, map[string]string{"status": "ok"})
	case rest == "links":
		switch r.Method {
		case http.MethodGet, http.MethodHead:
			s.listLinks(w, r)
		case http.MethodPost:
			if !s.authorized(w, r) {
				return
			}
			s.createLink(w, r)
		default:
			methodNotAllowed(w, r, "GET, HEAD, POST")
		}
	case strings.HasPrefix(rest, "links/") && len(rest) > len("links/") && !strings.Contains(rest[len("links/"):], "/"):
		// Primero el método (405 de API-12), luego el token (AUT-05).
		if methodNotAllowed(w, r, "DELETE", http.MethodDelete) || !s.authorized(w, r) {
			return
		}
		s.deleteLink(w, r, strings.TrimPrefix(rest, "links/"))
	default:
		writeErrors(w, r, http.StatusNotFound, "ruta no encontrada")
	}
}

// methodNotAllowed responde 405 con el Allow dado si el método de r no está
// entre los admitidos. Devuelve true si ya ha respondido. Sin métodos
// admitidos responde siempre (lo usa el caso de /api/links, que ya filtró).
func methodNotAllowed(w http.ResponseWriter, r *http.Request, allow string, ok ...string) bool {
	for _, m := range ok {
		if r.Method == m {
			return false
		}
	}
	w.Header().Set("Allow", allow)
	writeErrors(w, r, http.StatusMethodNotAllowed, "método no permitido")
	return true
}

func (s *server) listLinks(w http.ResponseWriter, r *http.Request) {
	links, err := s.svc.List(r.Context())
	if err != nil {
		writeInternal(w, r, err, true)
		return
	}
	out := make([]linkJSON, 0, len(links)) // [] y no null cuando no hay enlaces
	for _, l := range links {
		out = append(out, s.toJSON(l))
	}
	writeJSON(w, r, http.StatusOK, out)
}

func (s *server) createLink(w http.ResponseWriter, r *http.Request) {
	in, ok := decodeInput(w, r)
	if !ok {
		writeErrors(w, r, http.StatusBadRequest, "el cuerpo debe ser un objeto JSON con url, alias y expires_at")
		return
	}
	l, err := s.svc.Create(r.Context(), in)

	var invalid *link.ValidationError
	var taken *link.AliasTakenError
	switch {
	case err == nil:
		writeJSON(w, r, http.StatusCreated, s.toJSON(l))
	case errors.As(err, &invalid):
		writeErrors(w, r, http.StatusBadRequest, invalid.Messages...)
	case errors.As(err, &taken):
		writeErrors(w, r, http.StatusConflict, taken.Error())
	case errors.Is(err, link.ErrNoCodeAvailable):
		writeErrors(w, r, http.StatusServiceUnavailable, "no se ha podido generar un código libre; inténtalo de nuevo")
	default:
		writeInternal(w, r, err, true)
	}
}

// decodeInput lee el cuerpo como un único objeto JSON de hasta 64 KiB, sin
// campos desconocidos ni datos detrás. false si no cumple.
func decodeInput(w http.ResponseWriter, r *http.Request) (link.Input, bool) {
	var body struct {
		URL       string `json:"url"`
		Alias     string `json:"alias"`
		ExpiresAt string `json:"expires_at"`
	}
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxBody))
	dec.DisallowUnknownFields()
	// Puntero: un `null` decodifica sin error y dejaría el objeto vacío.
	p := &body
	if err := dec.Decode(&p); err != nil || p == nil {
		return link.Input{}, false
	}
	// Después del objeto no puede haber nada más.
	if _, err := dec.Token(); err != io.EOF {
		return link.Input{}, false
	}
	return link.Input{URL: body.URL, Alias: body.Alias, ExpiresAt: body.ExpiresAt}, true
}

func (s *server) deleteLink(w http.ResponseWriter, r *http.Request, code string) {
	err := s.svc.Delete(r.Context(), code)
	switch {
	case err == nil:
		w.WriteHeader(http.StatusNoContent) // sin cuerpo ni Content-Type
	case errors.Is(err, link.ErrNotFound):
		writeErrors(w, r, http.StatusNotFound, "enlace no encontrado")
	default:
		writeInternal(w, r, err, true)
	}
}
