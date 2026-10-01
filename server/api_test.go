package server_test

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"acorta/link"
	"acorta/server"
)

// API-01: healthz responde 200 {"status":"ok"}.
func TestAPI01_Healthz(t *testing.T) {
	e := newEnv(t, staticFS())
	wantJSON(t, do(e.h, http.MethodGet, "/api/healthz", ""), http.StatusOK, `{"status":"ok"}`)
	wantHeadEmpty(t, e.h, "/api/healthz")
}

// wantHeadEmpty comprueba que HEAD responde 200 application/json sin cuerpo.
func wantHeadEmpty(t *testing.T, h http.Handler, path string) {
	t.Helper()
	w := do(h, http.MethodHead, path, "")
	wantStatus(t, w, http.StatusOK)
	wantHeader(t, w, "Content-Type", "application/json")
	if w.Body.Len() != 0 {
		t.Errorf("HEAD %s tiene cuerpo: %q", path, w.Body.String())
	}
}

// API-02 (short_url): una URL base con barra final no duplica la barra.
func TestAPI02_BarraFinalEnBaseURL(t *testing.T) {
	for _, base := range []string{"http://localhost:8080/", "http://localhost:8080"} {
		t.Run("API-02 base "+base, func(t *testing.T) {
			e := newEnv(t, staticFS())
			h := server.Handler(e.svc, base, staticFS())
			w := do(h, http.MethodPost, "/api/links", `{"url":"https://example.com"}`)
			wantStatus(t, w, http.StatusCreated)
			if got := decodeMap(t, w)["short_url"]; got != "http://localhost:8080/aaaaaa" {
				t.Errorf("short_url = %v, quería http://localhost:8080/aaaaaa", got)
			}
			w = do(h, http.MethodGet, "/api/links", "")
			if !strings.Contains(w.Body.String(), `"short_url":"http://localhost:8080/aaaaaa"`) {
				t.Errorf("la lista no lleva el short_url sin doble barra: %q", w.Body.String())
			}
		})
	}
}

// API-02: crear solo con url da 201 y el enlace en JSON.
func TestAPI02_Crear(t *testing.T) {
	e := newEnv(t, staticFS())
	w := do(e.h, http.MethodPost, "/api/links", `{"url":"https://example.com"}`)
	wantJSON(t, w, http.StatusCreated, `{
		"code": "aaaaaa",
		"url": "https://example.com",
		"short_url": "http://localhost:8080/aaaaaa",
		"visits": 0,
		"created_at": "2026-10-01T12:00:00Z",
		"expires_at": null,
		"expired": false
	}`)
	// El campo expires_at debe estar presente y ser null, no omitido.
	if m := decodeMap(t, w); m != nil {
		if v, ok := m["expires_at"]; !ok || v != nil {
			t.Errorf("expires_at = %v (presente: %v), quería null presente", v, ok)
		}
	}
	// Y el enlace existe de verdad.
	wantStatus(t, do(e.h, http.MethodGet, "/aaaaaa", ""), http.StatusFound)
}

// API-03: alias y expires_at válidos se respetan, con la caducidad en UTC.
func TestAPI03_AliasYCaducidad(t *testing.T) {
	e := newEnv(t, staticFS())
	w := do(e.h, http.MethodPost, "/api/links",
		`{"url":"https://example.com","alias":"promo","expires_at":"2026-12-31T23:59:59+02:00"}`)
	wantJSON(t, w, http.StatusCreated, `{
		"code": "promo",
		"url": "https://example.com",
		"short_url": "http://localhost:8080/promo",
		"visits": 0,
		"created_at": "2026-10-01T12:00:00Z",
		"expires_at": "2026-12-31T21:59:59Z",
		"expired": false
	}`)
}

// API-04: la validación fallida da 400 con todos los mensajes, en orden.
func TestAPI04_Validacion(t *testing.T) {
	tests := []struct {
		name, body, want string
	}{
		{"ejemplo de la spec", `{"url":"ftp://x","alias":"A"}`,
			`{"errors":["url: debe empezar por http:// o https://","alias: debe tener entre 3 y 32 caracteres"]}`},
		{"los tres campos", `{"url":"","alias":"Ab","expires_at":"mañana"}`,
			`{"errors":["url: es obligatoria","alias: debe tener entre 3 y 32 caracteres","expires_at: debe tener formato RFC 3339 (2026-12-31T23:59:59Z)"]}`},
		{"objeto vacío", `{}`, `{"errors":["url: es obligatoria"]}`},
		{"alias reservado", `{"url":"https://example.com","alias":"api"}`,
			`{"errors":["alias: \"api\" está reservado"]}`},
	}
	for _, tc := range tests {
		t.Run("API-04 "+tc.name, func(t *testing.T) {
			e := newEnv(t, staticFS())
			wantJSON(t, do(e.h, http.MethodPost, "/api/links", tc.body), http.StatusBadRequest, tc.want)
			if ls, _ := e.svc.List(t.Context()); len(ls) != 0 {
				t.Errorf("se guardó algo: %v", ls)
			}
		})
	}
}

// API-05: alias en uso da 409.
func TestAPI05_AliasEnUso(t *testing.T) {
	e := newEnv(t, staticFS())
	wantStatus(t, do(e.h, http.MethodPost, "/api/links", `{"url":"https://example.com","alias":"promo"}`), http.StatusCreated)
	w := do(e.h, http.MethodPost, "/api/links", `{"url":"https://otra.example.com","alias":"promo"}`)
	wantJSON(t, w, http.StatusConflict, `{"errors":["alias: \"promo\" ya está en uso"]}`)
}

// API-06: cuerpo que no es un objeto JSON válido, campos desconocidos, tipos
// incorrectos, datos sobrantes o más de 64 KiB dan 400 con el mensaje fijo.
func TestAPI06_CuerpoInvalido(t *testing.T) {
	const want = `{"errors":["el cuerpo debe ser un objeto JSON con url, alias y expires_at"]}`
	big := `{"url":"https://example.com/` + strings.Repeat("a", 64*1024) + `"}`
	tests := []struct{ name, body string }{
		{"no es JSON", `esto no es json`},
		{"JSON truncado", `{"url":"https://example.com"`},
		{"null", `null`},
		{"array", `[{"url":"https://example.com"}]`},
		{"cadena", `"https://example.com"`},
		{"campo desconocido", `{"url":"https://example.com","color":"rojo"}`},
		{"url con tipo incorrecto", `{"url":5}`},
		{"alias con tipo incorrecto", `{"url":"https://example.com","alias":true}`},
		{"expires_at con tipo incorrecto", `{"url":"https://example.com","expires_at":20261231}`},
		{"datos después del objeto", `{"url":"https://example.com"} {"url":"https://otra.com"}`},
		{"basura después del objeto", `{"url":"https://example.com"} x`},
		{"más de 64 KiB", big},
	}
	for _, tc := range tests {
		t.Run("API-06 "+tc.name, func(t *testing.T) {
			e := newEnv(t, staticFS())
			wantJSON(t, do(e.h, http.MethodPost, "/api/links", tc.body), http.StatusBadRequest, want)
			if ls, _ := e.svc.List(t.Context()); len(ls) != 0 {
				t.Errorf("se guardó algo: %v", ls)
			}
		})
	}
	t.Run("API-06 cuerpo vacío", func(t *testing.T) {
		e := newEnv(t, staticFS())
		wantJSON(t, do(e.h, http.MethodPost, "/api/links", ""), http.StatusBadRequest, want)
	})
}

// API-07: sin código libre, 503 con el mensaje fijo.
func TestAPI07_SinCodigoLibre(t *testing.T) {
	const want = `{"errors":["no se ha podido generar un código libre; inténtalo de nuevo"]}`
	t.Run("API-07 servicio falso", func(t *testing.T) {
		svc := fakeSvc{create: func(link.Input) (link.Link, error) {
			return link.Link{}, fmt.Errorf("crear: %w", link.ErrNoCodeAvailable)
		}}
		h := handlerFor(svc)
		wantJSON(t, do(h, http.MethodPost, "/api/links", `{"url":"https://example.com"}`), http.StatusServiceUnavailable, want)
	})
	t.Run("API-07 servicio real", func(t *testing.T) {
		// El generador devuelve siempre el mismo código: el segundo enlace no encuentra hueco.
		e := newEnv(t, staticFS(), "aaaaaa")
		wantStatus(t, do(e.h, http.MethodPost, "/api/links", `{"url":"https://example.com"}`), http.StatusCreated)
		wantJSON(t, do(e.h, http.MethodPost, "/api/links", `{"url":"https://example.com/2"}`), http.StatusServiceUnavailable, want)
	})
}

// API-08: listar da un array con el más reciente primero; sin enlaces, [].
func TestAPI08_Listar(t *testing.T) {
	t.Run("API-08 vacío", func(t *testing.T) {
		e := newEnv(t, staticFS())
		w := do(e.h, http.MethodGet, "/api/links", "")
		wantJSON(t, w, http.StatusOK, `[]`)
		wantHeadEmpty(t, e.h, "/api/links")
		if got := strings.TrimSpace(w.Body.String()); got != "[]" {
			t.Errorf("cuerpo = %q, quería [] (no null)", got)
		}
	})
	t.Run("API-08 orden y campos", func(t *testing.T) {
		e := newEnv(t, staticFS())
		e.create(link.Input{URL: "https://example.com/1"}) // aaaaaa
		e.advance(time.Minute)
		e.create(link.Input{URL: "https://example.com/2", ExpiresAt: "2026-10-01T12:30:00Z"}) // bbbbbb
		e.advance(time.Minute)
		e.create(link.Input{URL: "https://example.com/3", Alias: "promo"})
		wantStatus(t, do(e.h, http.MethodGet, "/aaaaaa", ""), http.StatusFound)
		e.advance(time.Hour) // bbbbbb caduca

		w := do(e.h, http.MethodGet, "/api/links", "")
		wantJSON(t, w, http.StatusOK, `[
			{"code":"promo","url":"https://example.com/3","short_url":"http://localhost:8080/promo","visits":0,"created_at":"2026-10-01T12:02:00Z","expires_at":null,"expired":false},
			{"code":"bbbbbb","url":"https://example.com/2","short_url":"http://localhost:8080/bbbbbb","visits":0,"created_at":"2026-10-01T12:01:00Z","expires_at":"2026-10-01T12:30:00Z","expired":true},
			{"code":"aaaaaa","url":"https://example.com/1","short_url":"http://localhost:8080/aaaaaa","visits":1,"created_at":"2026-10-01T12:00:00Z","expires_at":null,"expired":false}
		]`)
	})
}

// API-09: borrar da 204 sin cuerpo y el código deja de existir.
func TestAPI09_Borrar(t *testing.T) {
	e := newEnv(t, staticFS())
	e.create(link.Input{URL: "https://example.com"})
	w := do(e.h, http.MethodDelete, "/api/links/aaaaaa", "")
	wantStatus(t, w, http.StatusNoContent)
	if w.Body.Len() != 0 {
		t.Errorf("el 204 tiene cuerpo: %q", w.Body.String())
	}
	if ct := w.Header().Get("Content-Type"); ct != "" {
		t.Errorf("el 204 lleva Content-Type %q, no debería llevar ninguno", ct)
	}
	wantPlain(t, do(e.h, http.MethodGet, "/aaaaaa", ""), http.StatusNotFound, "enlace no encontrado")
}

// API-10: borrar uno que no existe da 404 JSON.
func TestAPI10_BorrarInexistente(t *testing.T) {
	e := newEnv(t, staticFS())
	wantJSON(t, do(e.h, http.MethodDelete, "/api/links/zzzzzz", ""), http.StatusNotFound, `{"errors":["enlace no encontrado"]}`)
}

// API-11: ruta desconocida bajo /api/ da 404 JSON.
func TestAPI11_RutaDesconocida(t *testing.T) {
	e := newEnv(t, staticFS())
	for _, path := range []string{"/api/nada", "/api/", "/api/links/a/b", "/api/healthz/extra"} {
		t.Run("API-11 "+path, func(t *testing.T) {
			wantJSON(t, do(e.h, http.MethodGet, path, ""), http.StatusNotFound, `{"errors":["ruta no encontrada"]}`)
		})
	}
}

// API-12: método no admitido en una ruta de la API da 405 JSON con Allow exacto.
func TestAPI12_MetodoNoPermitido(t *testing.T) {
	tests := []struct {
		method, path, allow string
	}{
		{http.MethodPost, "/api/healthz", "GET, HEAD"},
		{http.MethodDelete, "/api/healthz", "GET, HEAD"},
		{http.MethodPut, "/api/links", "GET, HEAD, POST"},
		{http.MethodDelete, "/api/links", "GET, HEAD, POST"},
		{http.MethodGet, "/api/links/aaaaaa", "DELETE"},
		{http.MethodHead, "/api/links/aaaaaa", "DELETE"},
		{http.MethodPost, "/api/links/aaaaaa", "DELETE"},
	}
	for _, tc := range tests {
		t.Run("API-12 "+tc.method+" "+tc.path, func(t *testing.T) {
			e := newEnv(t, staticFS())
			w := do(e.h, tc.method, tc.path, "")
			wantStatus(t, w, http.StatusMethodNotAllowed)
			wantHeader(t, w, "Allow", tc.allow)
			wantHeader(t, w, "Content-Type", "application/json")
			if tc.method != http.MethodHead {
				wantJSON(t, w, http.StatusMethodNotAllowed, `{"errors":["método no permitido"]}`)
			}
		})
	}
}

// API-13: un fallo interno da 500 «error interno» sin filtrar el detalle.
func TestAPI13_ErrorInterno(t *testing.T) {
	boom := errors.New("disco lleno: secreto-interno")
	svc := fakeSvc{
		create:  func(link.Input) (link.Link, error) { return link.Link{}, fmt.Errorf("guardar: %w", boom) },
		list:    func() ([]link.Link, error) { return nil, boom },
		del:     func(string) error { return boom },
		resolve: func(string, bool) (string, error) { return "", boom },
	}
	h := handlerFor(svc)
	tests := []struct{ method, path, body string }{
		{http.MethodPost, "/api/links", `{"url":"https://example.com"}`},
		{http.MethodGet, "/api/links", ""},
		{http.MethodDelete, "/api/links/aaaaaa", ""},
	}
	for _, tc := range tests {
		t.Run("API-13 "+tc.method+" "+tc.path, func(t *testing.T) {
			w := do(h, tc.method, tc.path, tc.body)
			wantJSON(t, w, http.StatusInternalServerError, `{"errors":["error interno"]}`)
			if strings.Contains(w.Body.String(), "secreto-interno") {
				t.Errorf("la respuesta filtra el detalle: %q", w.Body.String())
			}
		})
	}
}
