package main

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"acorta/link"
	"acorta/store"
)

var t0 = time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)

// fakeEnv es el entorno de los tests: sin variables de entorno, reloj
// ajustable, códigos deterministas y un listen que no abre sockets.
type fakeEnv struct {
	now        time.Time
	codes      []string
	listenAddr string
	listenH    http.Handler
	listenOut  string // lo que había en stdout cuando se llamó a listen
	listenErr  error
	listened   bool
}

// setup instala el entorno falso y lo restaura al acabar. Los tests que lo
// usan no pueden ser paralelos (comparten la variable env).
func setup(t *testing.T) *fakeEnv {
	t.Helper()
	f := &fakeEnv{now: t0}
	old := env
	t.Cleanup(func() { env = old })
	env = environment{
		getenv: func(string) string { return "" },
		now:    func() time.Time { return f.now },
		newCode: func() (string, error) {
			if len(f.codes) == 0 {
				return "", errors.New("sin más códigos de prueba")
			}
			c := f.codes[0]
			f.codes = f.codes[1:]
			return c, nil
		},
	}
	return f
}

// exec ejecuta run con un buffer para cada salida.
func exec(args ...string) (code int, stdout, stderr string) {
	var o, e bytes.Buffer
	code = run(args, &o, &e)
	return code, o.String(), e.String()
}

func dbPath(t *testing.T) string { return filepath.Join(t.TempDir(), "acorta.db") }

// expect comprueba código de salida, stdout exacto y stderr exacto.
func expect(t *testing.T, args []string, wantCode int, wantOut, wantErr string) {
	t.Helper()
	code, out, errOut := exec(args...)
	if code != wantCode {
		t.Errorf("%v: código de salida = %d, quiero %d", args, code, wantCode)
	}
	if out != wantOut {
		t.Errorf("%v: stdout = %q, quiero %q", args, out, wantOut)
	}
	if errOut != wantErr {
		t.Errorf("%v: stderr = %q, quiero %q", args, errOut, wantErr)
	}
}

// Mensajes de uso: «uso: » + la sinopsis de la spec 005.
const (
	usageAdd   = "uso: acorta add [-alias ALIAS] [-ttl DURACIÓN] [-db RUTA] [-base-url URL] URL"
	usageList  = "uso: acorta list [-db RUTA]"
	usageRm    = "uso: acorta rm [-db RUTA] CÓDIGO"
	usageServe = "uso: acorta serve [-addr :8080] [-db RUTA] [-base-url URL]"
)

// storedLinks lee los enlaces directamente del almacén.
func storedLinks(t *testing.T, db string) []link.Link {
	t.Helper()
	s, err := store.Open(db)
	if err != nil {
		t.Fatalf("abrir el almacén: %v", err)
	}
	defer s.Close()
	ls, err := s.List(context.Background())
	if err != nil {
		t.Fatalf("listar: %v", err)
	}
	return ls
}

// CLI-01: add crea el enlace e imprime solo la URL corta.
func TestCLI01_AddPrintsShortURL(t *testing.T) {
	f := setup(t)
	f.codes = []string{"k3x9ab"}
	db := dbPath(t)
	expect(t, []string{"add", "-db", db, "https://example.com"}, 0, "http://localhost:8080/k3x9ab\n", "")
	ls := storedLinks(t, db)
	if len(ls) != 1 || ls[0].Code != "k3x9ab" || ls[0].URL != "https://example.com" || ls[0].ExpiresAt != nil {
		t.Errorf("CLI-01: enlaces guardados = %+v", ls)
	}
}

// CLI-02: con -alias, el alias es el código.
func TestCLI02_AddWithAlias(t *testing.T) {
	setup(t)
	db := dbPath(t)
	expect(t, []string{"add", "-db", db, "-alias", "promo", "https://example.com"}, 0, "http://localhost:8080/promo\n", "")
	if ls := storedLinks(t, db); len(ls) != 1 || ls[0].Code != "promo" {
		t.Errorf("CLI-02: enlaces guardados = %+v", ls)
	}
}

// CLI-03: -ttl fija la caducidad a now+ttl; un ttl inválido es un error de uso.
func TestCLI03_AddTTL(t *testing.T) {
	f := setup(t)
	f.codes = []string{"aaaaaa"}
	db := dbPath(t)
	expect(t, []string{"add", "-db", db, "-ttl", "72h", "https://example.com"}, 0, "http://localhost:8080/aaaaaa\n", "")
	ls := storedLinks(t, db)
	want := t0.Add(72 * time.Hour)
	if len(ls) != 1 || ls[0].ExpiresAt == nil || !ls[0].ExpiresAt.Equal(want) {
		t.Fatalf("CLI-03: caducidad guardada = %+v, quiero %v", ls, want)
	}
	// Con minutos también, y se ve en list.
	f.codes = []string{"bbbbbb"}
	expect(t, []string{"add", "-db", db, "-ttl", "90m", "https://example.org"}, 0, "http://localhost:8080/bbbbbb\n", "")
	_, out, _ := exec("list", "-db", db)
	if !strings.Contains(out, "2026-06-01T13:30:00Z") {
		t.Errorf("CLI-03: list no muestra la caducidad de 90m:\n%s", out)
	}
}

func TestCLI03_InvalidTTL(t *testing.T) {
	setup(t)
	const msg = "error: ttl: debe ser una duración positiva (ejemplos: 90m, 72h)\n"
	for _, ttl := range []string{"abc", "0", "0s", "-1h", "72", ""} {
		db := dbPath(t)
		expect(t, []string{"add", "-db", db, "-ttl", ttl, "https://example.com"}, 2, "", msg)
		if _, err := os.Stat(db); err == nil {
			if ls := storedLinks(t, db); len(ls) != 0 {
				t.Errorf("CLI-03: -ttl %q guardó %+v", ttl, ls)
			}
		}
	}
}

// CLI-04: un error por línea, stdout vacío, salida 1.
func TestCLI04_ValidationErrors(t *testing.T) {
	setup(t)
	db := dbPath(t)
	tests := []struct {
		name string
		args []string
		want string
	}{
		{"una URL mala", []string{"add", "-db", db, "ftp://example.com"},
			"error: url: debe empezar por http:// o https://\n"},
		{"url y alias", []string{"add", "-db", db, "-alias", "Promo", "ftp://example.com"},
			"error: url: debe empezar por http:// o https://\n" +
				"error: alias: solo admite minúsculas, números y guiones\n"},
		{"alias reservado", []string{"add", "-db", db, "-alias", "api", "https://example.com"},
			"error: alias: \"api\" está reservado\n"},
		{"URL vacía", []string{"add", "-db", db, "  "},
			"error: url: es obligatoria\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) { expect(t, tt.args, 1, "", tt.want) })
	}
	if ls := storedLinks(t, db); len(ls) != 0 {
		t.Errorf("CLI-04: se guardó algo pese a los errores: %+v", ls)
	}
}

func TestCLI04_AliasTaken(t *testing.T) {
	setup(t)
	db := dbPath(t)
	expect(t, []string{"add", "-db", db, "-alias", "promo", "https://example.com"}, 0, "http://localhost:8080/promo\n", "")
	expect(t, []string{"add", "-db", db, "-alias", "promo", "https://example.org"}, 1, "", "error: alias: \"promo\" ya está en uso\n")
	if ls := storedLinks(t, db); len(ls) != 1 || ls[0].URL != "https://example.com" {
		t.Errorf("CLI-04: enlaces tras el alias repetido = %+v", ls)
	}
}

// CLI-05: sin URL o con más de una, uso en stderr y salida 2.
func TestCLI05_AddArgumentCount(t *testing.T) {
	setup(t)
	db := dbPath(t)
	for _, args := range [][]string{
		{"add", "-db", db},
		{"add", "-db", db, "https://a.example", "https://b.example"},
	} {
		expect(t, args, 2, "", usageAdd+"\n")
	}
	if _, err := os.Stat(db); err == nil {
		if ls := storedLinks(t, db); len(ls) != 0 {
			t.Errorf("CLI-05: se guardó algo: %+v", ls)
		}
	}
}

// rows separa la salida de list en filas de campos (no fija el relleno).
func rows(out string) [][]string {
	var rs [][]string
	for _, line := range strings.Split(strings.TrimRight(out, "\n"), "\n") {
		rs = append(rs, strings.Fields(line))
	}
	return rs
}

func equalFields(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// CLI-06: tabla con cabecera, orden LIS-01 y las tres formas de CADUCA.
func TestCLI06_ListTable(t *testing.T) {
	f := setup(t)
	db := dbPath(t)
	f.codes = []string{"aaaaaa", "bbbbbb"}
	exec("add", "-db", db, "https://uno.example/a") // t0, no caduca
	f.now = t0.Add(time.Minute)
	exec("add", "-db", db, "-ttl", "1h", "https://dos.example/b") // caducará pronto
	f.now = t0.Add(2 * time.Minute)
	exec("add", "-db", db, "-alias", "promo", "-ttl", "72h", "https://tres.example/c")

	// Visitas sembradas directamente en el almacén.
	s, err := store.Open(db)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 3; i++ {
		if err := s.AddVisit(context.Background(), "aaaaaa"); err != nil {
			t.Fatal(err)
		}
	}
	s.Close()

	f.now = t0.Add(5 * time.Hour) // bbbbbb ya ha caducado, promo no
	code, out, errOut := exec("list", "-db", db)
	if code != 0 || errOut != "" {
		t.Fatalf("CLI-06: código=%d stderr=%q", code, errOut)
	}
	// Columnas alineadas con espacios: sin tabuladores y la URL empieza en
	// la misma posición en todas las filas (cuántos espacios, no importa).
	if strings.Contains(out, "\t") {
		t.Errorf("CLI-06: la tabla lleva tabuladores:\n%q", out)
	}
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	if len(lines) > 1 {
		// La posición se cuenta en caracteres, no en bytes: la «Ó» de CÓDIGO
		// ocupa dos bytes y una sola columna en pantalla.
		column := func(line, marca string) int {
			i := strings.Index(line, marca)
			if i < 0 {
				return -1
			}
			return utf8.RuneCountInString(line[:i])
		}
		col := column(lines[0], "URL")
		for _, l := range lines[1:] {
			if i := column(l, "https://"); i != col {
				t.Errorf("CLI-06: columna URL desalineada (%d, cabecera %d): %q", i, col, l)
			}
		}
	}
	got := rows(out)
	want := [][]string{
		{"CÓDIGO", "VISITAS", "CADUCA", "URL"},
		{"promo", "0", "2026-06-04T12:02:00Z", "https://tres.example/c"},
		{"bbbbbb", "0", "caducado", "https://dos.example/b"},
		{"aaaaaa", "3", "nunca", "https://uno.example/a"},
	}
	if len(got) != len(want) {
		t.Fatalf("CLI-06: filas = %d, quiero %d:\n%s", len(got), len(want), out)
	}
	for i := range want {
		if !equalFields(got[i], want[i]) {
			t.Errorf("CLI-06: fila %d = %q, quiero %q", i, got[i], want[i])
		}
	}
}

// CLI-07: sin enlaces, mensaje y salida 0.
func TestCLI07_ListEmpty(t *testing.T) {
	setup(t)
	expect(t, []string{"list", "-db", dbPath(t)}, 0, "No hay enlaces.\n", "")
}

// CLI-08: rm borra y confirma; si no existe, error y salida 1.
func TestCLI08_Remove(t *testing.T) {
	setup(t)
	db := dbPath(t)
	exec("add", "-db", db, "-alias", "promo", "https://example.com")
	expect(t, []string{"rm", "-db", db, "promo"}, 0, "Borrado: promo\n", "")
	if ls := storedLinks(t, db); len(ls) != 0 {
		t.Errorf("CLI-08: el enlace sigue ahí: %+v", ls)
	}
	expect(t, []string{"rm", "-db", db, "promo"}, 1, "", "error: enlace no encontrado\n")
}

// CLI-08 (uso): rm sin código o con varios es un error de uso.
func TestCLI08_RemoveArgumentCount(t *testing.T) {
	setup(t)
	db := dbPath(t)
	for _, args := range [][]string{{"rm", "-db", db}, {"rm", "-db", db, "a", "b"}} {
		expect(t, args, 2, "", usageRm+"\n")
	}
}

// CLI-09: serve abre la base, anuncia la dirección y sirve server.Handler.
func TestCLI09_Serve(t *testing.T) {
	f := setup(t)
	db := dbPath(t)
	var out bytes.Buffer
	// El handler se prueba dentro de listen, que es mientras se sirve: al
	// volver listen, serve cierra la base de datos.
	served := 0
	env.listen = func(addr string, h http.Handler) error {
		f.listened, f.listenAddr, f.listenH = true, addr, h
		f.listenOut = out.String()
		if h != nil {
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, httptest.NewRequest("GET", "/api/links", nil))
			served = rec.Code
		}
		return nil
	}
	var errOut bytes.Buffer
	code := run([]string{"serve", "-db", db, "-addr", ":9090"}, &out, &errOut)
	if code != 0 {
		t.Errorf("CLI-09: código = %d, quiero 0 (stderr %q)", code, errOut.String())
	}
	if !f.listened {
		t.Fatal("CLI-09: no se llamó a listen")
	}
	if f.listenAddr != ":9090" {
		t.Errorf("CLI-09: addr = %q, quiero :9090", f.listenAddr)
	}
	const msg = "acorta escuchando en http://localhost:9090\n"
	if out.String() != msg {
		t.Errorf("CLI-09: stdout = %q, quiero %q", out.String(), msg)
	}
	if f.listenOut != msg {
		t.Errorf("CLI-09: el mensaje debe escribirse antes de escuchar, había %q", f.listenOut)
	}
	if errOut.Len() != 0 {
		t.Errorf("CLI-09: stderr = %q, quiero vacío", errOut.String())
	}
	if _, err := os.Stat(db); err != nil {
		t.Errorf("CLI-09: serve no abrió la base de datos: %v", err)
	}
	if f.listenH == nil {
		t.Fatal("CLI-09: handler nil")
	}
	if served != 200 {
		t.Errorf("CLI-09: GET /api/links = %d, quiero 200 (el handler de server)", served)
	}
}

func TestCLI09_ServeDefaultAddr(t *testing.T) {
	f := setup(t)
	env.listen = func(addr string, h http.Handler) error { f.listenAddr = addr; return nil }
	expect(t, []string{"serve", "-db", dbPath(t)}, 0, "acorta escuchando en http://localhost:8080\n", "")
	if f.listenAddr != ":8080" {
		t.Errorf("CLI-09: addr por defecto = %q, quiero :8080", f.listenAddr)
	}
}

func TestCLI09_ServeHostInAddr(t *testing.T) {
	f := setup(t)
	env.listen = func(addr string, h http.Handler) error { f.listenAddr = addr; return nil }
	expect(t, []string{"serve", "-db", dbPath(t), "-addr", "127.0.0.1:9000"}, 0, "acorta escuchando en http://127.0.0.1:9000\n", "")
	if f.listenAddr != "127.0.0.1:9000" {
		t.Errorf("CLI-09: addr = %q, quiero 127.0.0.1:9000", f.listenAddr)
	}
}

func TestCLI09_ServeListenError(t *testing.T) {
	setup(t)
	env.listen = func(string, http.Handler) error { return errors.New("address already in use") }
	code, _, errOut := exec("serve", "-db", dbPath(t))
	if code != 1 {
		t.Errorf("CLI-09: código = %d, quiero 1", code)
	}
	if errOut != "error: address already in use\n" {
		t.Errorf("CLI-09: stderr = %q", errOut)
	}
}

// CLI-10: flag > entorno > por defecto, para -db y -base-url.
func TestCLI10_DBPrecedence(t *testing.T) {
	setup(t)
	env.getenv = os.Getenv
	envDB, flagDB := dbPath(t), dbPath(t)
	t.Setenv("ACORTA_DB", envDB)

	exec("add", "-alias", "delentorno", "https://example.com") // sin -db: usa el entorno
	exec("add", "-db", flagDB, "-alias", "delflag", "https://example.com")

	if ls := storedLinks(t, envDB); len(ls) != 1 || ls[0].Code != "delentorno" {
		t.Errorf("CLI-10: base del entorno = %+v", ls)
	}
	if ls := storedLinks(t, flagDB); len(ls) != 1 || ls[0].Code != "delflag" {
		t.Errorf("CLI-10: el flag debía ganar al entorno: %+v", ls)
	}
}

func TestCLI10_DBDefault(t *testing.T) {
	setup(t)
	env.getenv = os.Getenv
	t.Setenv("ACORTA_DB", "")
	t.Chdir(t.TempDir())
	exec("add", "-alias", "pordefecto", "https://example.com")
	if ls := storedLinks(t, "acorta.db"); len(ls) != 1 || ls[0].Code != "pordefecto" {
		t.Errorf("CLI-10: base por defecto acorta.db = %+v", ls)
	}
}

func TestCLI10_BaseURLPrecedence(t *testing.T) {
	setup(t)
	env.getenv = os.Getenv
	db := dbPath(t)
	t.Setenv("ACORTA_BASE_URL", "https://env.example/")
	expect(t, []string{"add", "-db", db, "-alias", "uno", "https://example.com"}, 0, "https://env.example/uno\n", "")
	expect(t, []string{"add", "-db", db, "-base-url", "https://flag.example///", "-alias", "dos", "https://example.com"}, 0, "https://flag.example/dos\n", "")
	t.Setenv("ACORTA_BASE_URL", "")
	expect(t, []string{"add", "-db", db, "-base-url", "http://localhost:8080/", "-alias", "tres", "https://example.com"}, 0, "http://localhost:8080/tres\n", "")
	expect(t, []string{"add", "-db", db, "-alias", "cuatro", "https://example.com"}, 0, "http://localhost:8080/cuatro\n", "")
}

// CLI-11: version.
func TestCLI11_Version(t *testing.T) {
	setup(t)
	expect(t, []string{"version"}, 0, "acorta "+version+"\n", "")
	if version != "dev" {
		t.Errorf("CLI-11: la versión por defecto = %q, quiero dev", version)
	}
	old := version
	t.Cleanup(func() { version = old })
	version = "1.2.3"
	expect(t, []string{"version"}, 0, "acorta 1.2.3\n", "")
}

// CLI-12: ayuda y subcomando desconocido.
func TestCLI12_Help(t *testing.T) {
	setup(t)
	for _, args := range [][]string{{}, {"help"}, {"-h"}} {
		code, out, errOut := exec(args...)
		if code != 0 {
			t.Errorf("CLI-12 %v: código = %d, quiero 0", args, code)
		}
		if errOut != "" {
			t.Errorf("CLI-12 %v: stderr = %q, quiero vacío", args, errOut)
		}
		for _, sub := range []string{"serve", "add", "list", "rm", "version", "help"} {
			if !strings.Contains(out, sub) {
				t.Errorf("CLI-12 %v: la ayuda no menciona %q:\n%s", args, sub, out)
			}
		}
	}
}

// CLI-12: -h dentro de un subcomando: uso y flags en stdout, salida 0.
func TestCLI12_SubcommandHelp(t *testing.T) {
	setup(t)
	tests := []struct {
		args  []string
		usage string
		flags []string
	}{
		{[]string{"add", "-h"}, usageAdd, []string{"-alias", "-ttl", "-db", "-base-url"}},
		{[]string{"list", "-h"}, usageList, []string{"-db"}},
		{[]string{"rm", "-h"}, usageRm, []string{"-db"}},
		{[]string{"serve", "-h"}, usageServe, []string{"-addr", "-db", "-base-url"}},
	}
	for _, tt := range tests {
		code, out, errOut := exec(tt.args...)
		if code != 0 {
			t.Errorf("CLI-12 %v: código = %d, quiero 0", tt.args, code)
		}
		if errOut != "" {
			t.Errorf("CLI-12 %v: stderr = %q, quiero vacío", tt.args, errOut)
		}
		if !strings.HasPrefix(out, tt.usage+"\n") {
			t.Errorf("CLI-12 %v: stdout debe empezar por %q: %q", tt.args, tt.usage, out)
		}
		for _, fl := range tt.flags {
			if !strings.Contains(out, fl) {
				t.Errorf("CLI-12 %v: stdout no lista el flag %s: %q", tt.args, fl, out)
			}
		}
	}
}

func TestCLI12_UnknownSubcommand(t *testing.T) {
	setup(t)
	code, out, errOut := exec("x")
	if code != 2 {
		t.Errorf("CLI-12: código = %d, quiero 2", code)
	}
	if out != "" {
		t.Errorf("CLI-12: stdout = %q, quiero vacío", out)
	}
	if !strings.HasPrefix(errOut, "error: subcomando desconocido \"x\"\n") {
		t.Errorf("CLI-12: stderr = %q", errOut)
	}
	for _, sub := range []string{"serve", "add", "list", "rm", "version", "help"} {
		if !strings.Contains(errOut, sub) {
			t.Errorf("CLI-12: stderr sin la ayuda (falta %q): %q", sub, errOut)
		}
	}
}

// Códigos de salida de la spec: un flag desconocido es un error de uso (2).
func TestCLI_UnknownFlagIsUsageError(t *testing.T) {
	setup(t)
	for _, c := range []struct {
		args  []string
		usage string
	}{
		{[]string{"list", "-db", dbPath(t), "-nope"}, usageList},
		{[]string{"add", "-nope", "https://example.com"}, usageAdd},
		{[]string{"rm", "-nope", "x"}, usageRm},
		{[]string{"serve", "-nope"}, usageServe},
	} {
		code, out, errOut := exec(c.args...)
		if code != 2 || out != "" {
			t.Errorf("flag desconocido %v: código=%d stdout=%q", c.args, code, out)
		}
		// El paquete flag puede añadir su propia línea; el uso debe estar.
		if !strings.Contains(errOut, c.usage+"\n") {
			t.Errorf("flag desconocido %v: stderr sin %q: %q", c.args, c.usage, errOut)
		}
	}
}

// CLI-13: una base que no se puede abrir da error y salida 1 en todo subcomando.
func TestCLI13_CannotOpenDB(t *testing.T) {
	setup(t)
	env.listen = func(string, http.Handler) error { t.Error("CLI-13: serve no debe escuchar"); return nil }
	bad := filepath.Join(t.TempDir(), "no-existe", "acorta.db")
	for _, args := range [][]string{
		{"add", "-db", bad, "https://example.com"},
		{"list", "-db", bad},
		{"rm", "-db", bad, "promo"},
		{"serve", "-db", bad},
	} {
		code, out, errOut := exec(args...)
		if code != 1 {
			t.Errorf("CLI-13 %v: código = %d, quiero 1", args[0], code)
		}
		if out != "" {
			t.Errorf("CLI-13 %v: stdout = %q, quiero vacío", args[0], out)
		}
		if !strings.HasPrefix(errOut, "error: ") || strings.Count(errOut, "\n") != 1 {
			t.Errorf("CLI-13 %v: stderr = %q, quiero una línea «error: …»", args[0], errOut)
		}
	}
}
