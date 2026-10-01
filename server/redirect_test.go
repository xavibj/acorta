package server_test

import (
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	"acorta/link"
)

// RED-01: un código vigente redirige con 302, Location, no-store y suma visita.
func TestRED01_Redirige(t *testing.T) {
	e := newEnv(t, staticFS())
	e.create(link.Input{URL: "https://example.com/una/ruta?x=1"})

	w := do(e.h, http.MethodGet, "/aaaaaa", "")
	wantStatus(t, w, http.StatusFound)
	wantHeader(t, w, "Location", "https://example.com/una/ruta?x=1")
	wantHeader(t, w, "Cache-Control", "no-store")
	if got := e.visits("aaaaaa"); got != 1 {
		t.Errorf("RED-01: visitas = %d, quería 1", got)
	}
	do(e.h, http.MethodGet, "/aaaaaa", "")
	if got := e.visits("aaaaaa"); got != 2 {
		t.Errorf("RED-01: visitas tras la segunda petición = %d, quería 2", got)
	}
}

// RED-02: un código inexistente da 404 text/plain «enlace no encontrado».
func TestRED02_NoExiste(t *testing.T) {
	e := newEnv(t, staticFS())
	w := do(e.h, http.MethodGet, "/zzzzzz", "")
	wantPlain(t, w, http.StatusNotFound, "enlace no encontrado")
}

// RED-03: un código caducado da 410 «enlace caducado» y no suma visita.
func TestRED03_Caducado(t *testing.T) {
	e := newEnv(t, staticFS())
	e.create(link.Input{URL: "https://example.com", ExpiresAt: "2026-10-01T13:00:00Z"})
	e.advance(2 * time.Hour)

	w := do(e.h, http.MethodGet, "/aaaaaa", "")
	wantPlain(t, w, http.StatusGone, "enlace caducado")
	if loc := w.Header().Get("Location"); loc != "" {
		t.Errorf("RED-03: no debería haber Location, hay %q", loc)
	}
	if got := e.visits("aaaaaa"); got != 0 {
		t.Errorf("RED-03: visitas = %d, quería 0", got)
	}
}

// RED-04: HEAD da el mismo estado y cabeceras que GET, sin cuerpo y sin visita.
func TestRED04_Head(t *testing.T) {
	e := newEnv(t, staticFS())
	e.create(link.Input{URL: "https://example.com/destino"})                                 // aaaaaa
	e.create(link.Input{URL: "https://example.com/otro", ExpiresAt: "2026-10-01T13:00:00Z"}) // bbbbbb
	e.advance(2 * time.Hour)

	tests := []struct {
		name string
		path string
		want int
	}{
		{"vigente", "/aaaaaa", http.StatusFound},
		{"caducado", "/bbbbbb", http.StatusGone},
		{"inexistente", "/zzzzzz", http.StatusNotFound},
	}
	for _, tc := range tests {
		t.Run("RED-04 "+tc.name, func(t *testing.T) {
			head := do(e.h, http.MethodHead, tc.path, "")
			wantStatus(t, head, tc.want)
			if head.Body.Len() != 0 {
				t.Errorf("HEAD tiene cuerpo: %q", head.Body.String())
			}
			if tc.path == "/aaaaaa" {
				if got := e.visits("aaaaaa"); got != 0 {
					t.Errorf("HEAD sumó visita: %d", got)
				}
			}
			get := do(e.h, http.MethodGet, tc.path, "")
			if head.Code != get.Code {
				t.Errorf("estado HEAD = %d, GET = %d", head.Code, get.Code)
			}
			for _, h := range []string{"Location", "Cache-Control", "Content-Type"} {
				if hv, gv := head.Header().Get(h), get.Header().Get(h); hv != gv {
					t.Errorf("cabecera %s: HEAD %q, GET %q", h, hv, gv)
				}
			}
		})
	}
	// Solo el GET del enlace vigente cuenta.
	if got := e.visits("aaaaaa"); got != 1 {
		t.Errorf("RED-04: visitas tras HEAD+GET = %d, quería 1", got)
	}
}

// RED-05: otros métodos sobre /{código} dan 405 con Allow: GET, HEAD.
func TestRED05_MetodoNoPermitido(t *testing.T) {
	e := newEnv(t, staticFS())
	e.create(link.Input{URL: "https://example.com"})
	for _, m := range []string{http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete} {
		for _, path := range []string{"/aaaaaa", "/zzzzzz"} {
			t.Run("RED-05 "+m+" "+path, func(t *testing.T) {
				w := do(e.h, m, path, "")
				wantStatus(t, w, http.StatusMethodNotAllowed)
				wantHeader(t, w, "Allow", "GET, HEAD")
			})
		}
	}
	if got := e.visits("aaaaaa"); got != 0 {
		t.Errorf("RED-05: visitas = %d, quería 0", got)
	}
}

// RED-06: una ruta de varios segmentos que no es de la API ni estática da 404 text/plain.
func TestRED06_VariosSegmentos(t *testing.T) {
	e := newEnv(t, staticFS())
	for _, path := range []string{"/a/b", "/aaaaaa/extra", "/a/b/c"} {
		t.Run("RED-06 "+path, func(t *testing.T) {
			w := do(e.h, http.MethodGet, path, "")
			wantStatus(t, w, http.StatusNotFound)
			wantHeader(t, w, "Content-Type", "text/plain; charset=utf-8")
		})
	}
}

// RED-07: un fallo del servicio que no es «no existe» ni «caducado» da 500
// text/plain «error interno», sin Location y sin filtrar el detalle.
func TestRED07_ErrorInterno(t *testing.T) {
	svc := fakeSvc{resolve: func(string, bool) (string, error) {
		return "", errors.New("disco lleno: secreto-interno")
	}}
	h := handlerFor(svc)
	for _, m := range []string{http.MethodGet, http.MethodHead} {
		t.Run("RED-07 "+m, func(t *testing.T) {
			w := do(h, m, "/aaaaaa", "")
			wantStatus(t, w, http.StatusInternalServerError)
			wantHeader(t, w, "Content-Type", "text/plain; charset=utf-8")
			wantHeader(t, w, "Location", "")
			if m == http.MethodGet {
				if got := w.Body.String(); got != "error interno\n" {
					t.Errorf("cuerpo = %q, quería %q", got, "error interno\n")
				}
			} else if w.Body.Len() != 0 {
				t.Errorf("HEAD tiene cuerpo: %q", w.Body.String())
			}
			if strings.Contains(w.Body.String(), "secreto-interno") {
				t.Errorf("la respuesta filtra el detalle: %q", w.Body.String())
			}
		})
	}
}
