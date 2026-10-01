package server_test

import (
	"io/fs"
	"mime"
	"net/http"
	"strings"
	"testing"
	"testing/fstest"

	"acorta/link"
	"acorta/server"
)

// handlerFor monta el Handler sobre un servicio dado y los estáticos de prueba.
func handlerFor(svc server.Service) http.Handler {
	return server.Handler(svc, baseURL, staticFS())
}

// wantMedia comprueba el tipo de contenido sin mirar parámetros como charset.
func wantMedia(t *testing.T, got string, want ...string) {
	t.Helper()
	mt, _, err := mime.ParseMediaType(got)
	if err != nil {
		t.Errorf("Content-Type = %q no es válido", got)
		return
	}
	for _, w := range want {
		if mt == w {
			return
		}
	}
	t.Errorf("Content-Type = %q, quería uno de %v", got, want)
}

// EST-01: con dist/index.html, GET / da 200 text/html con su contenido.
func TestEST01_Index(t *testing.T) {
	fsys := staticFS()
	e := newEnv(t, fsys)
	w := do(e.h, http.MethodGet, "/", "")
	wantStatus(t, w, http.StatusOK)
	wantMedia(t, w.Header().Get("Content-Type"), "text/html")
	want, _ := fs.ReadFile(fsys, "dist/index.html")
	if w.Body.String() != string(want) {
		t.Errorf("cuerpo = %q, quería %q", w.Body.String(), want)
	}
}

// EST-02: cada fichero de dist/ se sirve en su ruta sin prefijo, con su tipo.
func TestEST02_Ficheros(t *testing.T) {
	fsys := staticFS()
	tests := []struct {
		path, file string
		media      []string
	}{
		{"/assets/app-3f9a.js", "dist/assets/app-3f9a.js", []string{"text/javascript", "application/javascript"}},
		{"/assets/app-3f9a.css", "dist/assets/app-3f9a.css", []string{"text/css"}},
		{"/assets/js/chunk.js", "dist/assets/js/chunk.js", []string{"text/javascript", "application/javascript"}},
		{"/assets/fonts/a.woff2", "dist/assets/fonts/a.woff2", nil},
		{"/favicon.svg", "dist/favicon.svg", []string{"image/svg+xml"}},
	}
	for _, tc := range tests {
		t.Run("EST-02 "+tc.path, func(t *testing.T) {
			e := newEnv(t, fsys)
			w := do(e.h, http.MethodGet, tc.path, "")
			wantStatus(t, w, http.StatusOK)
			if tc.media != nil {
				wantMedia(t, w.Header().Get("Content-Type"), tc.media...)
			}
			want, _ := fs.ReadFile(fsys, tc.file)
			if w.Body.String() != string(want) {
				t.Errorf("cuerpo = %q, quería %q", w.Body.String(), want)
			}
		})
	}
}

// EST-03: un estático que no existe da 404 text/plain.
func TestEST03_NoExiste(t *testing.T) {
	e := newEnv(t, staticFS())
	for _, path := range []string{"/assets/nada.js", "/nada.txt"} {
		t.Run("EST-03 "+path, func(t *testing.T) {
			w := do(e.h, http.MethodGet, path, "")
			wantStatus(t, w, http.StatusNotFound)
			wantHeader(t, w, "Content-Type", "text/plain; charset=utf-8")
		})
	}
}

// EST-04: sin dist/index.html, GET / da la página de aviso y lo demás sigue funcionando.
func TestEST04_SinCompilar(t *testing.T) {
	fsys := fstest.MapFS{
		"sin-compilar.html": {Data: []byte("<h1>Frontend sin compilar</h1><p>Ejecuta make frontend</p>")},
	}
	e := newEnv(t, fsys)

	w := do(e.h, http.MethodGet, "/", "")
	wantStatus(t, w, http.StatusOK)
	wantMedia(t, w.Header().Get("Content-Type"), "text/html")
	body := w.Body.String()
	for _, s := range []string{"Frontend sin compilar", "make frontend"} {
		if !strings.Contains(body, s) {
			t.Errorf("EST-04: el aviso no contiene %q: %q", s, body)
		}
	}

	// La API y las redirecciones van igual.
	wantJSON(t, do(e.h, http.MethodGet, "/api/healthz", ""), http.StatusOK, `{"status":"ok"}`)
	l := e.create(link.Input{URL: "https://example.com/destino"})
	w = do(e.h, http.MethodGet, "/"+l.Code, "")
	wantStatus(t, w, http.StatusFound)
	wantHeader(t, w, "Location", "https://example.com/destino")
	wantStatus(t, do(e.h, http.MethodPost, "/api/links", `{"url":"https://example.com"}`), http.StatusCreated)
}

// EST-05: Static() contiene siempre sin-compilar.html.
func TestEST05_Embebido(t *testing.T) {
	st := server.Static()
	if st == nil {
		t.Fatal("EST-05: Static() devuelve nil")
	}
	data, err := fs.ReadFile(st, "sin-compilar.html")
	if err != nil {
		t.Fatalf("EST-05: leer sin-compilar.html: %v", err)
	}
	for _, s := range []string{"Frontend sin compilar", "make frontend"} {
		if !strings.Contains(string(data), s) {
			t.Errorf("EST-05: sin-compilar.html no contiene %q", s)
		}
	}
}
