package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Tests de la spec 008 para serve (AUT-07 a AUT-10). Como los de la spec 005:
// contra run, con listen inyectado, sin procesos ni sockets.

// Tokens de prueba: todos de 16 caracteres o más, y distintos entre sí.
const (
	flagToken = "token-del-flag-0123456789"
	envToken  = "token-del-entorno-0123456789"
)

// Mensajes de la spec 008 (AUT-08 y AUT-09), ya como línea de stderr.
const (
	errMissingToken = "error: falta el token de API: usa -token o ACORTA_TOKEN\n"
	errShortToken   = "error: el token de API debe tener al menos 16 caracteres\n"
)

// Sinopsis de serve con el token (AUT-07).
const usageServeToken = "uso: acorta serve [-addr :8080] [-db RUTA] [-base-url URL] [-token TOKEN]"

// badDB es una ruta en un directorio que no existe: si serve llegara a abrir
// la base de datos, fallaría por CLI-13 con otro mensaje; los tests de
// AUT-08 y AUT-09 exigen el mensaje del token, así que el token se comprueba
// antes.
func badDB(t *testing.T) string { return filepath.Join(t.TempDir(), "no-existe", "acorta.db") }

// postLinks hace POST /api/links contra h con esa cabecera Authorization
// (vacía: sin cabecera) y devuelve el estado. 201 significa que el handler
// aceptó el token; 401, que no. Cada petición lleva un alias distinto para
// no chocar con las anteriores (409) ni depender del generador de códigos.
func postLinks(h http.Handler, auth string, n int) int {
	body := fmt.Sprintf(`{"url":"https://example.com","alias":"sonda%d"}`, n)
	r := httptest.NewRequest(http.MethodPost, "/api/links", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	if auth != "" {
		r.Header.Set("Authorization", auth)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w.Code
}

// serveProbe ejecuta serve con args y, dentro de listen (mientras se sirve:
// al volver, serve cierra la base de datos), hace un POST /api/links por cada
// cabecera de auths ("" es sin cabecera). Devuelve el código de salida, las
// salidas y los estados HTTP en el mismo orden que auths. Si no se llegó a
// escuchar, statuses es nil.
func serveProbe(t *testing.T, f *fakeEnv, args []string, auths ...string) (code int, out, errOut string, statuses []int) {
	t.Helper()
	env.listen = func(addr string, h http.Handler) error {
		f.listened, f.listenAddr, f.listenH = true, addr, h
		if h == nil {
			t.Fatal("serve: handler nil")
		}
		for i, a := range auths {
			statuses = append(statuses, postLinks(h, a, i))
		}
		return nil
	}
	code, out, errOut = exec(args...)
	return code, out, errOut, statuses
}

// wantServed comprueba que serve arrancó bien (salida 0, sin stderr, listen
// llamado) y que el handler contestó a cada sonda con el estado esperado.
func wantServed(t *testing.T, f *fakeEnv, code int, errOut string, got, want []int, auths []string) {
	t.Helper()
	if code != 0 {
		t.Errorf("código = %d, quiero 0 (stderr %q)", code, errOut)
	}
	if errOut != "" {
		t.Errorf("stderr = %q, quiero vacío", errOut)
	}
	if !f.listened {
		t.Fatal("no se llamó a listen")
	}
	if len(got) != len(want) {
		t.Fatalf("sondas contestadas = %v, quiero %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("POST /api/links con Authorization %q = %d, quiero %d", auths[i], got[i], want[i])
		}
	}
}

// AUT-07: -token arranca con ese token; sin flag, ACORTA_TOKEN; el flag gana a
// la variable. Lo que distingue un token de otro es qué acepta el handler.
func TestAUT07_TokenSource(t *testing.T) {
	tests := []struct {
		name  string
		env   string   // valor de ACORTA_TOKEN ("" es vacía)
		flags []string // flags de serve además de -db
		auths []string // cabeceras que se prueban
		want  []int    // estado esperado por cabecera
	}{
		{
			name:  "solo el flag",
			flags: []string{"-token", flagToken},
			auths: []string{"Bearer " + flagToken, ""},
			want:  []int{201, 401},
		},
		{
			name:  "solo la variable",
			env:   envToken,
			auths: []string{"Bearer " + envToken, ""},
			want:  []int{201, 401},
		},
		{
			name:  "el flag gana a la variable",
			env:   envToken,
			flags: []string{"-token", flagToken},
			auths: []string{"Bearer " + flagToken, "Bearer " + envToken, ""},
			want:  []int{201, 401, 401},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := setup(t)
			env.getenv = os.Getenv
			t.Setenv("ACORTA_TOKEN", tt.env)
			args := append([]string{"serve", "-db", dbPath(t)}, tt.flags...)
			code, out, errOut, got := serveProbe(t, f, args, tt.auths...)
			wantServed(t, f, code, errOut, got, tt.want, tt.auths)
			if want := "acorta escuchando en http://localhost:8080\n"; out != want {
				t.Errorf("stdout = %q, quiero %q", out, want)
			}
		})
	}
}

// AUT-07: la sinopsis de serve incluye [-token TOKEN], tanto en la ayuda
// (-h, stdout) como en el mensaje de uso (flag desconocido, stderr).
func TestAUT07_Synopsis(t *testing.T) {
	setup(t)
	code, out, errOut := exec("serve", "-h")
	if code != 0 || errOut != "" {
		t.Errorf("serve -h: código=%d stderr=%q, quiero 0 y vacío", code, errOut)
	}
	if !strings.HasPrefix(out, usageServeToken+"\n") {
		t.Errorf("serve -h: stdout debe empezar por %q: %q", usageServeToken, out)
	}
	code, out, errOut = exec("serve", "-nope")
	if code != 2 || out != "" {
		t.Errorf("serve -nope: código=%d stdout=%q, quiero 2 y vacío", code, out)
	}
	if !strings.Contains(errOut, usageServeToken+"\n") {
		t.Errorf("serve -nope: stderr sin %q: %q", usageServeToken, errOut)
	}
}

// AUT-08: sin token (ni flag ni variable, o un valor que queda vacío tras
// recortar espacios) serve escribe el error, sale con 1 y no abre la base de
// datos ni escucha.
func TestAUT08_MissingToken(t *testing.T) {
	tests := []struct {
		name  string
		unset bool   // ACORTA_TOKEN sin definir (si no, definida con env)
		env   string // valor de ACORTA_TOKEN
		flags []string
	}{
		{name: "ni flag ni variable", unset: true},
		{name: "variable vacía"},
		{name: "flag vacío y variable vacía", flags: []string{"-token", ""}},
		{name: "flag solo de espacios", flags: []string{"-token", "   "}},
		{name: "variable solo de espacios", env: "   "},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := setup(t)
			env.getenv = os.Getenv
			t.Setenv("ACORTA_TOKEN", tt.env) // t.Setenv restaura el valor original al acabar
			if tt.unset {
				os.Unsetenv("ACORTA_TOKEN")
			}
			env.listen = func(string, http.Handler) error {
				f.listened = true
				t.Error("AUT-08: serve no debe escuchar sin token")
				return nil
			}
			args := append([]string{"serve", "-db", badDB(t)}, tt.flags...)
			expect(t, args, 1, "", errMissingToken)
		})
	}
}

// AUT-09: un token de entre 1 y 15 caracteres, contados tras recortar los
// espacios de los extremos, es un error: serve no abre la base ni escucha
// (el token se comprueba antes que la base de datos). Con 16 vale, y
// el handler recibe el token recortado.
func TestAUT09_TokenLength(t *testing.T) {
	const (
		token15 = "123456789012345"
		token16 = "1234567890123456"
	)
	tests := []struct {
		name   string
		flag   string // valor de -token ("" es sin flag)
		env    string // valor de ACORTA_TOKEN
		accept string // "" si debe fallar; si no, el token que debe aceptar el handler
	}{
		{name: "15 por flag", flag: token15},
		{name: "15 por variable", env: token15},
		{name: "15 rodeado de espacios", flag: "  " + token15 + "  "},
		{name: "16 por flag", flag: token16, accept: token16},
		{name: "16 por variable", env: token16, accept: token16},
		{name: "16 rodeado de espacios", flag: "  " + token16 + "  ", accept: token16},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := setup(t)
			env.getenv = os.Getenv
			t.Setenv("ACORTA_TOKEN", tt.env)
			var flags []string
			if tt.flag != "" {
				flags = []string{"-token", tt.flag}
			}
			if tt.accept == "" {
				env.listen = func(string, http.Handler) error {
					f.listened = true
					t.Error("AUT-09: serve no debe escuchar con un token corto")
					return nil
				}
				args := append([]string{"serve", "-db", badDB(t)}, flags...)
				expect(t, args, 1, "", errShortToken)
				return
			}
			args := append([]string{"serve", "-db", dbPath(t)}, flags...)
			auths := []string{"Bearer " + tt.accept, ""}
			code, _, errOut, got := serveProbe(t, f, args, auths...)
			wantServed(t, f, code, errOut, got, []int{201, 401}, auths)
		})
	}
}

// AUT-10: el mensaje de arranque de CLI-09 no cambia y no lleva el token.
func TestAUT10_StartupMessageHidesToken(t *testing.T) {
	f := setup(t)
	env.getenv = os.Getenv
	t.Setenv("ACORTA_TOKEN", envToken)
	for _, c := range []struct {
		flags []string
		want  string
	}{
		{[]string{"-token", flagToken}, "acorta escuchando en http://localhost:8080\n"},
		{[]string{"-addr", ":9090"}, "acorta escuchando en http://localhost:9090\n"},
	} {
		f.listened = false
		env.listen = func(string, http.Handler) error { f.listened = true; return nil }
		args := append([]string{"serve", "-db", dbPath(t)}, c.flags...)
		code, out, errOut := exec(args...)
		if code != 0 || errOut != "" {
			t.Errorf("%v: código=%d stderr=%q, quiero 0 y vacío", args, code, errOut)
		}
		if !f.listened {
			t.Errorf("%v: no se llamó a listen", args)
		}
		if out != c.want {
			t.Errorf("%v: stdout = %q, quiero %q", args, out, c.want)
		}
		for _, tok := range []string{flagToken, envToken} {
			if strings.Contains(out+errOut, tok) {
				t.Errorf("%v: la salida contiene el token %q: %q %q", args, tok, out, errOut)
			}
		}
	}
}

// AUT-10: la ayuda de serve -h nombra el flag y la variable, pero no el
// token, aunque esté definido en el entorno o se haya pasado por flag.
func TestAUT10_HelpHidesToken(t *testing.T) {
	setup(t)
	env.getenv = os.Getenv
	t.Setenv("ACORTA_TOKEN", envToken)
	for _, args := range [][]string{
		{"serve", "-h"},
		{"serve", "-token", flagToken, "-h"},
	} {
		code, out, errOut := exec(args...)
		if code != 0 || errOut != "" {
			t.Errorf("%v: código=%d stderr=%q, quiero 0 y vacío", args, code, errOut)
		}
		for _, name := range []string{"-token", "ACORTA_TOKEN"} {
			if !strings.Contains(out, name) {
				t.Errorf("%v: la ayuda no menciona %s: %q", args, name, out)
			}
		}
		for _, tok := range []string{flagToken, envToken} {
			if strings.Contains(out, tok) {
				t.Errorf("%v: la ayuda contiene el token %q: %q", args, tok, out)
			}
		}
	}
}
