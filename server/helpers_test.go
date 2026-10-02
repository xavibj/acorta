package server_test

import (
	"context"
	"encoding/json"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"acorta/link"
	"acorta/server"
	"acorta/shortener"
	"acorta/store"
)

const baseURL = "http://localhost:8080"

// testToken es el token de API con el que se monta el Handler en los tests.
const testToken = "token-de-prueba-0123456789"

// t0 es el instante fijo en el que arranca el reloj de los tests.
var t0 = time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC)

// staticFS es un frontend "compilado" en memoria.
func staticFS() fs.FS {
	return fstest.MapFS{
		"sin-compilar.html":         {Data: []byte("<h1>Frontend sin compilar</h1><p>make frontend</p>")},
		"dist/index.html":           {Data: []byte("<!doctype html><title>acorta</title><div id=app></div>")},
		"dist/assets/app-3f9a.js":   {Data: []byte("console.log('hola');")},
		"dist/assets/app-3f9a.css":  {Data: []byte("body{margin:0}")},
		"dist/favicon.svg":          {Data: []byte("<svg xmlns=\"http://www.w3.org/2000/svg\"/>")},
		"dist/assets/js/chunk.js":   {Data: []byte("export default 1;")},
		"dist/assets/fonts/a.woff2": {Data: []byte("wOF2datos")},
	}
}

// env es un servidor sobre el servicio real, con reloj y códigos controlados.
type env struct {
	t       *testing.T
	h       http.Handler
	svc     *shortener.Service
	clock   time.Time
	codes   []string // códigos que irá devolviendo el generador
	nextIdx int
}

// newEnv monta el servicio real sobre SQLite en t.TempDir(). El generador
// devuelve en orden codes; cuando se acaban, repite el último.
func newEnv(t *testing.T, static fs.FS, codes ...string) *env {
	t.Helper()
	if len(codes) == 0 {
		codes = []string{"aaaaaa", "bbbbbb", "cccccc", "dddddd"}
	}
	st, err := store.Open(filepath.Join(t.TempDir(), "acorta.db"))
	if err != nil {
		t.Fatalf("abrir el almacén: %v", err)
	}
	t.Cleanup(func() { st.Close() })
	e := &env{t: t, clock: t0, codes: codes}
	e.svc = shortener.New(st,
		func() time.Time { return e.clock },
		func() (string, error) {
			i := e.nextIdx
			if i >= len(e.codes) {
				i = len(e.codes) - 1
			}
			e.nextIdx++
			return e.codes[i], nil
		})
	e.h = server.Handler(e.svc, baseURL, static, testToken)
	return e
}

// advance adelanta el reloj del servicio.
func (e *env) advance(d time.Duration) { e.clock = e.clock.Add(d) }

// create crea un enlace directamente por el servicio.
func (e *env) create(in link.Input) link.Link {
	e.t.Helper()
	l, err := e.svc.Create(context.Background(), in)
	if err != nil {
		e.t.Fatalf("crear %+v: %v", in, err)
	}
	return l
}

// visits lee el contador de visitas de un código desde el servicio.
func (e *env) visits(code string) int64 {
	e.t.Helper()
	ls, err := e.svc.List(context.Background())
	if err != nil {
		e.t.Fatalf("listar: %v", err)
	}
	for _, l := range ls {
		if l.Code == code {
			return l.Visits
		}
	}
	e.t.Fatalf("el enlace %q no está en la lista", code)
	return 0
}

// do lanza una petición contra h con el token de API correcto (spec 008,
// AUT-04: con él, todo se comporta como en la spec 004). Los tests que
// controlan la cabecera Authorization usan doAuth (auth_test.go).
func do(h http.Handler, method, path, body string) *httptest.ResponseRecorder {
	return doAuth(h, method, path, "Bearer "+testToken, body)
}

// wantStatus comprueba el estado.
func wantStatus(t *testing.T, w *httptest.ResponseRecorder, want int) {
	t.Helper()
	if w.Code != want {
		t.Errorf("estado = %d, quería %d (cuerpo %q)", w.Code, want, w.Body.String())
	}
}

// wantHeader comprueba el valor exacto de una cabecera.
func wantHeader(t *testing.T, w *httptest.ResponseRecorder, name, want string) {
	t.Helper()
	if got := w.Header().Get(name); got != want {
		t.Errorf("cabecera %s = %q, quería %q", name, got, want)
	}
}

// wantPlain comprueba una respuesta text/plain cuyo cuerpo es el texto más un
// salto de línea final.
func wantPlain(t *testing.T, w *httptest.ResponseRecorder, status int, text string) {
	t.Helper()
	wantStatus(t, w, status)
	wantHeader(t, w, "Content-Type", "text/plain; charset=utf-8")
	if got := w.Body.String(); got != text+"\n" {
		t.Errorf("cuerpo = %q, quería %q", got, text+"\n")
	}
}

// wantJSON comprueba estado, Content-Type JSON, salto de línea final y que el
// cuerpo equivale (como JSON) a want.
func wantJSON(t *testing.T, w *httptest.ResponseRecorder, status int, want string) {
	t.Helper()
	wantStatus(t, w, status)
	wantHeader(t, w, "Content-Type", "application/json")
	body := w.Body.String()
	if !strings.HasSuffix(body, "\n") {
		t.Errorf("el cuerpo no termina en salto de línea: %q", body)
	}
	var got, exp any
	if err := json.Unmarshal([]byte(body), &got); err != nil {
		t.Errorf("el cuerpo no es JSON: %q", body)
		return
	}
	if err := json.Unmarshal([]byte(want), &exp); err != nil {
		t.Fatalf("el JSON esperado no es válido: %v", err)
	}
	g, _ := json.Marshal(got)
	x, _ := json.Marshal(exp)
	if string(g) != string(x) {
		t.Errorf("cuerpo = %s, quería %s", g, x)
	}
}

// decodeMap decodifica un cuerpo JSON de objeto.
func decodeMap(t *testing.T, w *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &m); err != nil {
		t.Fatalf("el cuerpo no es un objeto JSON: %q", w.Body.String())
	}
	return m
}

// fakeSvc es un servicio falso para forzar fallos. Cada función nula
// devuelve un valor vacío sin error.
type fakeSvc struct {
	create  func(link.Input) (link.Link, error)
	resolve func(code string, count bool) (string, error)
	list    func() ([]link.Link, error)
	del     func(code string) error
}

func (f fakeSvc) Create(_ context.Context, in link.Input) (link.Link, error) {
	if f.create == nil {
		return link.Link{}, nil
	}
	return f.create(in)
}

func (f fakeSvc) Resolve(_ context.Context, code string, count bool) (string, error) {
	if f.resolve == nil {
		return "", nil
	}
	return f.resolve(code, count)
}

func (f fakeSvc) List(context.Context) ([]link.Link, error) {
	if f.list == nil {
		return nil, nil
	}
	return f.list()
}

func (f fakeSvc) Delete(_ context.Context, code string) error {
	if f.del == nil {
		return nil
	}
	return f.del(code)
}

func (fakeSvc) Now() time.Time { return t0 }
