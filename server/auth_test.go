package server_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"acorta/link"
)

// Cuerpos exactos de los 401 de la spec 008.
const (
	bodyMissingToken = `{"errors":["falta el token de API"]}` + "\n"
	bodyWrongToken   = `{"errors":["token de API incorrecto"]}` + "\n"
)

// doAuth lanza una petición con la cabecera Authorization dada; con auth
// vacío, la cabecera no se envía.
func doAuth(h http.Handler, method, path, auth, body string) *httptest.ResponseRecorder {
	var r *http.Request
	if body == "" {
		r = httptest.NewRequest(method, path, nil)
	} else {
		r = httptest.NewRequest(method, path, strings.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
	}
	if auth != "" {
		r.Header.Set("Authorization", auth)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w
}

// wantUnauthorized comprueba un 401 de la spec 008: estado, WWW-Authenticate,
// Content-Type JSON y el cuerpo exacto.
func wantUnauthorized(t *testing.T, w *httptest.ResponseRecorder, body string) {
	t.Helper()
	wantStatus(t, w, http.StatusUnauthorized)
	wantHeader(t, w, "WWW-Authenticate", "Bearer")
	wantHeader(t, w, "Content-Type", "application/json")
	if got := w.Body.String(); got != body {
		t.Errorf("cuerpo = %q, quería %q", got, body)
	}
}

// count devuelve cuántos enlaces hay en el almacén.
func (e *env) count() int {
	e.t.Helper()
	ls, err := e.svc.List(context.Background())
	if err != nil {
		e.t.Fatalf("listar: %v", err)
	}
	return len(ls)
}

// AUT-01: POST /api/links sin Authorization da 401 con el cuerpo y la cabecera
// de la spec, y no crea nada.
func TestAUT01_CrearSinToken(t *testing.T) {
	e := newEnv(t, staticFS())
	w := doAuth(e.h, http.MethodPost, "/api/links", "", `{"url":"https://example.com"}`)
	wantUnauthorized(t, w, bodyMissingToken)
	if n := e.count(); n != 0 {
		t.Errorf("AUT-01: hay %d enlaces, quería 0 (el POST rechazado no debe crear nada)", n)
	}
}

// AUT-02: DELETE /api/links/{código} sin Authorization da 401, exista o no el
// código, y no borra nada.
func TestAUT02_BorrarSinToken(t *testing.T) {
	e := newEnv(t, staticFS())
	e.create(link.Input{URL: "https://example.com"}) // aaaaaa

	t.Run("AUT-02 código existente", func(t *testing.T) {
		w := doAuth(e.h, http.MethodDelete, "/api/links/aaaaaa", "", "")
		wantUnauthorized(t, w, bodyMissingToken)
		if n := e.count(); n != 1 {
			t.Errorf("AUT-02: hay %d enlaces, quería 1 (el DELETE rechazado no debe borrar nada)", n)
		}
		wantStatus(t, do(e.h, http.MethodHead, "/aaaaaa", ""), http.StatusFound)
	})
	t.Run("AUT-02 código inexistente", func(t *testing.T) {
		w := doAuth(e.h, http.MethodDelete, "/api/links/zzzzzz", "", "")
		wantUnauthorized(t, w, bodyMissingToken)
	})
}

// AUT-03: con Authorization pero sin el token correcto, las dos rutas dan 401
// «token de API incorrecto». El esquema no distingue mayúsculas; el token sí.
// La cabecera es exactamente esquema, un espacio y token: no se recorta nada.
func TestAUT03_TokenIncorrecto(t *testing.T) {
	bad := []struct {
		name string
		auth string
	}{
		{"otro token", "Bearer otro-token-que-no-es-el-bueno"},
		{"esquema Basic", "Basic dXN1YXJpbzpjb250cmFzZcOxYQ=="},
		{"Bearer sin nada detrás", "Bearer"},
		{"Bearer con espacio y nada detrás", "Bearer "},
		{"dos espacios tras Bearer", "Bearer  " + testToken},
		{"espacio final tras el token", "Bearer " + testToken + " "},
		{"token sin esquema", testToken},
		{"token en mayúsculas", "Bearer " + strings.ToUpper(testToken)},
		{"token con prefijo", "Bearer x" + testToken},
		{"token con sufijo", "Bearer " + testToken + "x"},
	}
	for _, tc := range bad {
		t.Run("AUT-03 POST "+tc.name, func(t *testing.T) {
			e := newEnv(t, staticFS())
			w := doAuth(e.h, http.MethodPost, "/api/links", tc.auth, `{"url":"https://example.com"}`)
			wantUnauthorized(t, w, bodyWrongToken)
			if n := e.count(); n != 0 {
				t.Errorf("AUT-03: hay %d enlaces, quería 0", n)
			}
		})
		t.Run("AUT-03 DELETE "+tc.name, func(t *testing.T) {
			e := newEnv(t, staticFS())
			e.create(link.Input{URL: "https://example.com"}) // aaaaaa
			w := doAuth(e.h, http.MethodDelete, "/api/links/aaaaaa", tc.auth, "")
			wantUnauthorized(t, w, bodyWrongToken)
			if n := e.count(); n != 1 {
				t.Errorf("AUT-03: hay %d enlaces, quería 1", n)
			}
		})
	}

	// El esquema no distingue mayúsculas: «bearer» y «BEARER» con el token
	// correcto se aceptan.
	for _, scheme := range []string{"bearer", "BEARER", "BeArEr"} {
		t.Run("AUT-03 esquema "+scheme+" se acepta", func(t *testing.T) {
			e := newEnv(t, staticFS())
			auth := scheme + " " + testToken
			w := doAuth(e.h, http.MethodPost, "/api/links", auth, `{"url":"https://example.com"}`)
			wantStatus(t, w, http.StatusCreated)
			w = doAuth(e.h, http.MethodDelete, "/api/links/aaaaaa", auth, "")
			wantStatus(t, w, http.StatusNoContent)
		})
	}
}

// AUT-04: con el token correcto, las rutas protegidas se comportan como en
// la spec 004 (aquí: 201 al crear y 204 al borrar).
func TestAUT04_TokenCorrecto(t *testing.T) {
	e := newEnv(t, staticFS())
	auth := "Bearer " + testToken

	w := doAuth(e.h, http.MethodPost, "/api/links", auth, `{"url":"https://example.com"}`)
	wantStatus(t, w, http.StatusCreated)
	if h := w.Header().Get("WWW-Authenticate"); h != "" {
		t.Errorf("AUT-04: un 201 no debe llevar WWW-Authenticate, lleva %q", h)
	}
	if n := e.count(); n != 1 {
		t.Errorf("AUT-04: hay %d enlaces, quería 1", n)
	}

	w = doAuth(e.h, http.MethodDelete, "/api/links/aaaaaa", auth, "")
	wantStatus(t, w, http.StatusNoContent)
	if n := e.count(); n != 0 {
		t.Errorf("AUT-04: hay %d enlaces, quería 0", n)
	}
}

// AUT-05: el token se comprueba antes de leer el cuerpo y antes de buscar el
// código: un POST con cuerpo inválido da 401 (no 400) y un DELETE de un
// código inexistente da 401 (no 404).
func TestAUT05_TokenAntesQueCuerpo(t *testing.T) {
	t.Run("AUT-05 POST sin token con cuerpo inválido", func(t *testing.T) {
		e := newEnv(t, staticFS())
		for _, body := range []string{`esto no es JSON`, `{"url":"ftp://x","alias":"A"}`, `[1,2,3]`} {
			w := doAuth(e.h, http.MethodPost, "/api/links", "", body)
			wantUnauthorized(t, w, bodyMissingToken)
		}
		if n := e.count(); n != 0 {
			t.Errorf("AUT-05: hay %d enlaces, quería 0", n)
		}
	})
	t.Run("AUT-05 POST con token incorrecto y cuerpo inválido", func(t *testing.T) {
		e := newEnv(t, staticFS())
		w := doAuth(e.h, http.MethodPost, "/api/links", "Bearer malo", `esto no es JSON`)
		wantUnauthorized(t, w, bodyWrongToken)
	})
	t.Run("AUT-05 DELETE sin token de código inexistente", func(t *testing.T) {
		e := newEnv(t, staticFS())
		w := doAuth(e.h, http.MethodDelete, "/api/links/zzzzzz", "", "")
		wantUnauthorized(t, w, bodyMissingToken)
	})
	t.Run("AUT-05 DELETE con token incorrecto de código inexistente", func(t *testing.T) {
		e := newEnv(t, staticFS())
		w := doAuth(e.h, http.MethodDelete, "/api/links/zzzzzz", "Bearer malo", "")
		wantUnauthorized(t, w, bodyWrongToken)
	})
}

// AUT-06: las rutas públicas (listar, healthz, redirección, estáticos) y el
// 405 de API-12 no miran la cabecera: responden igual con Authorization
// ausente, correcta o incorrecta.
func TestAUT06_RutasPublicas(t *testing.T) {
	routes := []struct {
		method, path string
		status       int
	}{
		{http.MethodGet, "/api/links", http.StatusOK},
		{http.MethodHead, "/api/links", http.StatusOK},
		{http.MethodGet, "/api/healthz", http.StatusOK},
		{http.MethodHead, "/api/healthz", http.StatusOK},
		{http.MethodGet, "/aaaaaa", http.StatusFound},
		{http.MethodHead, "/aaaaaa", http.StatusFound},
		{http.MethodGet, "/zzzzzz", http.StatusNotFound},
		{http.MethodGet, "/", http.StatusOK},
		{http.MethodGet, "/assets/app-3f9a.js", http.StatusOK},
		{http.MethodGet, "/favicon.svg", http.StatusOK},
		{http.MethodGet, "/nada.txt", http.StatusNotFound},
		// El método se comprueba antes que el token (nota de API-12 en la 008).
		{http.MethodPut, "/api/links", http.StatusMethodNotAllowed},
		{http.MethodPost, "/api/healthz", http.StatusMethodNotAllowed},
		{http.MethodGet, "/api/links/aaaaaa", http.StatusMethodNotAllowed},
	}
	auths := []struct {
		name string
		auth string
	}{
		{"correcta", "Bearer " + testToken},
		{"incorrecta", "Bearer otro-token-que-no-es-el-bueno"},
		{"esquema Basic", "Basic dXN1YXJpbzpjb250cmFzZcOxYQ=="},
		{"Bearer sin nada detrás", "Bearer"},
	}
	for _, rt := range routes {
		t.Run("AUT-06 "+rt.method+" "+rt.path, func(t *testing.T) {
			e := newEnv(t, staticFS())
			e.create(link.Input{URL: "https://example.com"}) // aaaaaa

			// Referencia: la respuesta sin cabecera.
			ref := doAuth(e.h, rt.method, rt.path, "", "")
			wantStatus(t, ref, rt.status)
			if h := ref.Header().Get("WWW-Authenticate"); h != "" {
				t.Errorf("sin cabecera: no debería haber WWW-Authenticate, hay %q", h)
			}
			for _, a := range auths {
				w := doAuth(e.h, rt.method, rt.path, a.auth, "")
				if w.Code != ref.Code {
					t.Errorf("cabecera %s: estado = %d, quería %d (cuerpo %q)", a.name, w.Code, ref.Code, w.Body.String())
				}
				for _, h := range []string{"Content-Type", "Location", "Allow", "Cache-Control"} {
					if got, want := w.Header().Get(h), ref.Header().Get(h); got != want {
						t.Errorf("cabecera %s: %s = %q, quería %q", a.name, h, got, want)
					}
				}
				if h := w.Header().Get("WWW-Authenticate"); h != "" {
					t.Errorf("cabecera %s: no debería haber WWW-Authenticate, hay %q", a.name, h)
				}
				if got, want := w.Body.String(), ref.Body.String(); got != want {
					t.Errorf("cabecera %s: cuerpo = %q, quería %q", a.name, got, want)
				}
			}
		})
	}
}
