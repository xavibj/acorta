package shortener_test

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"reflect"
	"regexp"
	"sync"
	"testing"
	"time"

	"acorta/link"
	"acorta/shortener"
	"acorta/store"
)

var ctx = context.Background()

// t0 es el instante inicial del reloj de los tests.
var t0 = time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC)

// fakeClock es un reloj fijo que se puede adelantar.
type fakeClock struct {
	mu  sync.Mutex
	cur time.Time
}

func (c *fakeClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.cur
}

func (c *fakeClock) Advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.cur = c.cur.Add(d)
}

// codeGen es un generador determinista: devuelve la lista en orden y, cuando
// se agota, un error. Cuenta las llamadas.
type codeGen struct {
	mu    sync.Mutex
	codes []string
	calls int
	err   error // si no es nil, se devuelve cuando la lista se agota (en vez del error genérico)
}

func (g *codeGen) Next() (string, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.calls++
	if len(g.codes) == 0 {
		if g.err != nil {
			return "", g.err
		}
		return "", errors.New("generador agotado")
	}
	c := g.codes[0]
	g.codes = g.codes[1:]
	return c, nil
}

func (g *codeGen) Calls() int {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.calls
}

// env reúne el servicio y lo que los tests necesitan para manipularlo.
type env struct {
	svc   *shortener.Service
	clock *fakeClock
	gen   *codeGen
}

// newEnv monta un servicio sobre un almacén SQLite real en t.TempDir().
func newEnv(t *testing.T, codes ...string) *env {
	t.Helper()
	st, err := store.Open(filepath.Join(t.TempDir(), "acorta.db"))
	if err != nil {
		t.Fatalf("abrir el almacén: %v", err)
	}
	t.Cleanup(func() { st.Close() })
	e := &env{clock: &fakeClock{cur: t0}, gen: &codeGen{codes: codes}}
	e.svc = shortener.New(st, e.clock.Now, e.gen.Next)
	return e
}

// mustCreate crea un enlace y falla el test si hay error.
func (e *env) mustCreate(t *testing.T, in link.Input) link.Link {
	t.Helper()
	l, err := e.svc.Create(ctx, in)
	if err != nil {
		t.Fatalf("Create(%+v): error inesperado: %v", in, err)
	}
	return l
}

// mustList lista y falla el test si hay error.
func (e *env) mustList(t *testing.T) []link.Link {
	t.Helper()
	ls, err := e.svc.List(ctx)
	if err != nil {
		t.Fatalf("List: error inesperado: %v", err)
	}
	return ls
}

// find devuelve el enlace con ese código del listado.
func (e *env) find(t *testing.T, code string) (link.Link, bool) {
	t.Helper()
	for _, l := range e.mustList(t) {
		if l.Code == code {
			return l, true
		}
	}
	return link.Link{}, false
}

// visits devuelve las visitas de un código según el listado.
func (e *env) visits(t *testing.T, code string) int64 {
	t.Helper()
	l, ok := e.find(t, code)
	if !ok {
		t.Fatalf("el código %q no está en el listado", code)
	}
	return l.Visits
}

func in(url string) link.Input { return link.Input{URL: url} }

// ---------- Códigos generados ----------

// COD-02: un código ocupado o reservado se descarta y se reintenta (hasta 5 intentos); el error del generador o del almacén no se reintenta.
func TestCOD02_CodigoOcupadoOReservadoSeDescartaYSeReintenta(t *testing.T) {
	t.Run("ocupado y reservado se descartan", func(t *testing.T) {
		e := newEnv(t, "fresh1", "api", "fresh1", "fresh2")
		e.mustCreate(t, in("https://a.example")) // se queda con fresh1
		l := e.mustCreate(t, in("https://b.example"))
		if l.Code != "fresh2" {
			t.Errorf("Code = %q, quiero %q (fresh1 está ocupado y api reservado)", l.Code, "fresh2")
		}
		if n := e.gen.Calls(); n != 4 {
			t.Errorf("llamadas al generador = %d, quiero 4", n)
		}
		if _, ok := e.find(t, "api"); ok {
			t.Error("la palabra reservada \"api\" no debe guardarse")
		}
		if n := len(e.mustList(t)); n != 2 {
			t.Errorf("enlaces guardados = %d, quiero 2", n)
		}
	})

	t.Run("el quinto intento aún vale", func(t *testing.T) {
		e := newEnv(t, "dupe01", "dupe01", "dupe01", "dupe01", "dupe01", "dupe01", "fresh1")
		e.mustCreate(t, link.Input{URL: "https://a.example", Alias: "dupe01"})
		// Intentos 1 a 4 fallan (ocupado), el 5.º sale libre.
		e.gen.codes = []string{"dupe01", "dupe01", "dupe01", "dupe01", "fresh1"}
		l := e.mustCreate(t, in("https://b.example"))
		if l.Code != "fresh1" {
			t.Errorf("Code = %q, quiero %q", l.Code, "fresh1")
		}
	})

	t.Run("cinco fallos dan ErrNoCodeAvailable y no guardan nada", func(t *testing.T) {
		e := newEnv(t)
		e.mustCreate(t, link.Input{URL: "https://a.example", Alias: "dupe01"})
		e.gen.codes = []string{"dupe01", "api", "dupe01", "assets", "dupe01", "fresh1"}
		_, err := e.svc.Create(ctx, in("https://b.example"))
		if !errors.Is(err, link.ErrNoCodeAvailable) {
			t.Fatalf("err = %v, quiero ErrNoCodeAvailable", err)
		}
		if n := e.gen.Calls(); n != 5 {
			t.Errorf("llamadas al generador = %d, quiero exactamente 5", n)
		}
		if n := len(e.mustList(t)); n != 1 {
			t.Errorf("enlaces guardados = %d, quiero 1 (solo el alias previo)", n)
		}
	})

	t.Run("si el generador falla se devuelve su error envuelto, sin reintentos ni guardar nada", func(t *testing.T) {
		e := newEnv(t)
		errGen := errors.New("sin entropía")
		e.gen.err = errGen
		_, err := e.svc.Create(ctx, in("https://a.example"))
		if !errors.Is(err, errGen) {
			t.Fatalf("err = %v, quiero el error del generador (errors.Is)", err)
		}
		if errors.Is(err, link.ErrNoCodeAvailable) {
			t.Error("un fallo del generador no es ErrNoCodeAvailable")
		}
		if n := e.gen.Calls(); n != 1 {
			t.Errorf("llamadas al generador = %d, quiero 1 (sin reintentos)", n)
		}
		if n := len(e.mustList(t)); n != 0 {
			t.Errorf("enlaces guardados = %d, quiero 0", n)
		}
	})

	t.Run("si el generador falla tras un intento descartado, también se devuelve sin más intentos", func(t *testing.T) {
		e := newEnv(t, "api") // reservada; luego el generador falla
		errGen := errors.New("sin entropía")
		e.gen.err = errGen
		_, err := e.svc.Create(ctx, in("https://a.example"))
		if !errors.Is(err, errGen) {
			t.Fatalf("err = %v, quiero el error del generador (errors.Is)", err)
		}
		if n := e.gen.Calls(); n != 2 {
			t.Errorf("llamadas al generador = %d, quiero 2", n)
		}
	})
}

// COD-03: acortar dos veces la misma URL da dos códigos distintos.
func TestCOD03_LaMismaURLDaDosCodigosDistintos(t *testing.T) {
	e := newEnv(t, "aaaaaa", "bbbbbb")
	a := e.mustCreate(t, in("https://example.com/misma"))
	b := e.mustCreate(t, in("https://example.com/misma"))
	if a.Code == "" || b.Code == "" || a.Code == b.Code {
		t.Errorf("códigos %q y %q: quiero dos códigos no vacíos y distintos", a.Code, b.Code)
	}
	if n := len(e.mustList(t)); n != 2 {
		t.Errorf("enlaces guardados = %d, quiero 2", n)
	}
}

// COD-04: el generador y el reloj inyectados son los que usa el servicio.
func TestCOD04_GeneradorYRelojInyectadosSeUsan(t *testing.T) {
	e := newEnv(t, "zzzzzz")
	l := e.mustCreate(t, in("https://example.com"))
	if l.Code != "zzzzzz" {
		t.Errorf("Code = %q, quiero el del generador inyectado %q", l.Code, "zzzzzz")
	}
	if !l.CreatedAt.Equal(t0) {
		t.Errorf("CreatedAt = %v, quiero el del reloj inyectado %v", l.CreatedAt, t0)
	}
	if got := e.svc.Now(); !got.Equal(t0) {
		t.Errorf("Now() = %v, quiero %v", got, t0)
	}
	e.clock.Advance(90 * time.Minute)
	if got := e.svc.Now(); !got.Equal(t0.Add(90 * time.Minute)) {
		t.Errorf("Now() tras adelantar = %v, quiero %v", got, t0.Add(90*time.Minute))
	}
}

// COD-04: con nil hay valores por defecto (crypto/rand y time.Now) que funcionan.
func TestCOD04_ConNilHayValoresPorDefectoQueFuncionan(t *testing.T) {
	st, err := store.Open(filepath.Join(t.TempDir(), "acorta.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Close() })
	svc := shortener.New(st, nil, nil)

	before := time.Now()
	a, err := svc.Create(ctx, in("https://example.com"))
	if err != nil {
		t.Fatalf("Create con valores por defecto: %v", err)
	}
	b, err := svc.Create(ctx, in("https://example.com"))
	if err != nil {
		t.Fatalf("Create con valores por defecto (2): %v", err)
	}
	after := time.Now()

	re := regexp.MustCompile(`^[a-z0-9]{6}$`)
	for _, l := range []link.Link{a, b} {
		if !re.MatchString(l.Code) {
			t.Errorf("Code = %q, quiero 6 caracteres de [a-z0-9]", l.Code)
		}
		if l.CreatedAt.Before(before.Truncate(time.Second)) || l.CreatedAt.After(after) {
			t.Errorf("CreatedAt = %v, quiero un instante entre %v y %v (reloj real)", l.CreatedAt, before, after)
		}
	}
	if a.Code == b.Code {
		t.Errorf("dos códigos aleatorios iguales: %q", a.Code)
	}
	if d := time.Since(svc.Now()); d < 0 || d > time.Minute {
		t.Errorf("Now() por defecto = %v, quiero la hora real", svc.Now())
	}
}

// ---------- Alias ----------

// ALI-01: el alias es el código del enlace.
func TestALI01_ElAliasEsElCodigo(t *testing.T) {
	e := newEnv(t) // sin códigos: el generador no debe hacer falta
	l := e.mustCreate(t, link.Input{URL: "https://example.com", Alias: "oferta-otono"})
	if l.Code != "oferta-otono" {
		t.Errorf("Code = %q, quiero %q", l.Code, "oferta-otono")
	}
	if n := e.gen.Calls(); n != 0 {
		t.Errorf("con alias no debe llamarse al generador (llamadas = %d)", n)
	}
	got, ok := e.find(t, "oferta-otono")
	if !ok || got.URL != "https://example.com" {
		t.Errorf("el enlace no está guardado con ese código: %+v (encontrado=%v)", got, ok)
	}
}

// ALI-06: un alias en uso, aunque esté caducado, da *AliasTakenError y no guarda nada.
func TestALI06_AliasEnUsoDaAliasTakenError(t *testing.T) {
	cases := []struct {
		name    string
		expires string // caducidad del enlace que ocupa el alias
		advance time.Duration
	}{
		{"enlace vigente", "", 0},
		{"enlace caducado", "2026-10-01T12:00:10Z", time.Minute},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			e := newEnv(t)
			e.mustCreate(t, link.Input{URL: "https://original.example", Alias: "promo", ExpiresAt: tc.expires})
			e.clock.Advance(tc.advance)

			_, err := e.svc.Create(ctx, link.Input{URL: "https://otra.example", Alias: "promo"})
			var taken *link.AliasTakenError
			if !errors.As(err, &taken) {
				t.Fatalf("err = %v (%T), quiero *link.AliasTakenError", err, err)
			}
			if want := `alias: "promo" ya está en uso`; err.Error() != want {
				t.Errorf("mensaje = %q, quiero %q", err.Error(), want)
			}
			if taken.Alias != "promo" {
				t.Errorf("Alias = %q, quiero %q", taken.Alias, "promo")
			}
			ls := e.mustList(t)
			if len(ls) != 1 || ls[0].URL != "https://original.example" {
				t.Errorf("el listado debe seguir con el enlace original: %+v", ls)
			}
		})
	}

	t.Run("solo se comprueba cuando la validación ha pasado", func(t *testing.T) {
		e := newEnv(t)
		e.mustCreate(t, link.Input{URL: "https://original.example", Alias: "promo"})
		_, err := e.svc.Create(ctx, link.Input{URL: "", Alias: "promo"})
		var ve *link.ValidationError
		if !errors.As(err, &ve) {
			t.Fatalf("err = %v (%T), quiero *link.ValidationError", err, err)
		}
		var taken *link.AliasTakenError
		if errors.As(err, &taken) {
			t.Error("no debe devolverse AliasTakenError si la validación falla")
		}
	})
}

// ---------- Validación en Create ----------

// URL-03, ALI-05 y CAD-02 (solo como ejemplos): Create devuelve *link.ValidationError con los mensajes de link.Validate y no guarda nada.
func TestCreate_EntradaInvalidaDaValidationErrorYNoGuardaNada(t *testing.T) {
	cases := []struct {
		name string
		in   link.Input
	}{
		{"un error", link.Input{URL: "ftp://example.com"}},
		{"tres errores a la vez", link.Input{URL: "", Alias: "api", ExpiresAt: "mañana"}},
		{"caducidad pasada", link.Input{URL: "https://example.com", ExpiresAt: "2020-01-01T00:00:00Z"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			e := newEnv(t, "aaaaaa")
			_, want := link.Validate(tc.in, t0)
			if len(want) == 0 {
				t.Fatal("el caso de prueba debería ser inválido según link.Validate")
			}

			_, err := e.svc.Create(ctx, tc.in)
			var ve *link.ValidationError
			if !errors.As(err, &ve) {
				t.Fatalf("err = %v (%T), quiero *link.ValidationError", err, err)
			}
			if !reflect.DeepEqual(ve.Messages, want) {
				t.Errorf("Messages = %q, quiero los de link.Validate %q", ve.Messages, want)
			}
			if n := len(e.mustList(t)); n != 0 {
				t.Errorf("enlaces guardados = %d, quiero 0", n)
			}
		})
	}
}

// CreatedAt (spec 001, Tiempo): instante del reloj en UTC y sin fracciones de segundo.
func TestCreate_CreatedAtEsElRelojEnUTCSinFracciones(t *testing.T) {
	e := newEnv(t, "aaaaaa")
	zone := time.FixedZone("UTC+2", 2*3600)
	e.clock.cur = time.Date(2026, 10, 1, 12, 0, 0, 789_000_000, zone)

	l := e.mustCreate(t, in("https://example.com"))
	want := time.Date(2026, 10, 1, 10, 0, 0, 0, time.UTC)
	if !l.CreatedAt.Equal(want) {
		t.Errorf("CreatedAt = %v, quiero %v", l.CreatedAt, want)
	}
	if l.CreatedAt.Location() != time.UTC {
		t.Errorf("CreatedAt en zona %v, quiero UTC", l.CreatedAt.Location())
	}
	if l.CreatedAt.Nanosecond() != 0 {
		t.Errorf("CreatedAt tiene fracciones: %d ns", l.CreatedAt.Nanosecond())
	}
	got, ok := e.find(t, l.Code)
	if !ok || !got.CreatedAt.Equal(want) {
		t.Errorf("CreatedAt guardado = %v (encontrado=%v), quiero %v", got.CreatedAt, ok, want)
	}
}

// CAD-01 y CAD-04: el servicio guarda la caducidad ya validada.
func TestCreate_GuardaLaCaducidadValidada(t *testing.T) {
	// CAD-01 y CAD-04 los prueba link; aquí, que el servicio la pasa al almacén.
	e := newEnv(t, "aaaaaa", "bbbbbb")
	l := e.mustCreate(t, link.Input{URL: "https://example.com", ExpiresAt: "2026-12-31T23:59:59+02:00"})
	want := time.Date(2026, 12, 31, 21, 59, 59, 0, time.UTC)
	if l.ExpiresAt == nil || !l.ExpiresAt.Equal(want) {
		t.Errorf("ExpiresAt = %v, quiero %v", l.ExpiresAt, want)
	}
	n := e.mustCreate(t, in("https://example.com"))
	if n.ExpiresAt != nil {
		t.Errorf("sin expires_at, ExpiresAt = %v, quiero nil", n.ExpiresAt)
	}
}

// ---------- Caducidad ----------

// CAD-06: un enlace caducado sigue en el listado y su código sigue ocupado hasta borrarlo.
func TestCAD06_UnEnlaceCaducadoSigueEnElListadoYSuCodigoOcupado(t *testing.T) {
	e := newEnv(t)
	e.mustCreate(t, link.Input{URL: "https://example.com", Alias: "temporal", ExpiresAt: "2026-10-01T12:00:10Z"})
	e.clock.Advance(time.Hour)

	l, ok := e.find(t, "temporal")
	if !ok {
		t.Fatal("el enlace caducado debe seguir en el listado")
	}
	if !l.Expired(e.svc.Now()) {
		t.Error("el enlace debe aparecer como caducado en el momento de listar")
	}
	_, err := e.svc.Create(ctx, link.Input{URL: "https://otra.example", Alias: "temporal"})
	var taken *link.AliasTakenError
	if !errors.As(err, &taken) {
		t.Errorf("err = %v, quiero *AliasTakenError: el código sigue ocupado", err)
	}
	if err := e.svc.Delete(ctx, "temporal"); err != nil {
		t.Fatalf("borrar un caducado: %v", err)
	}
	if _, err := e.svc.Create(ctx, link.Input{URL: "https://otra.example", Alias: "temporal"}); err != nil {
		t.Errorf("tras borrarlo el código debe quedar libre: %v", err)
	}
}

// ---------- Resolver ----------

// RES-01: un código vigente devuelve su URL.
func TestRES01_ResolverDevuelveLaURL(t *testing.T) {
	e := newEnv(t)
	e.mustCreate(t, link.Input{URL: "https://example.com/a?b=1#c", Alias: "destino"})
	e.mustCreate(t, link.Input{URL: "https://example.com/futuro", Alias: "futuro", ExpiresAt: "2026-10-01T12:00:10Z"})

	for code, want := range map[string]string{
		"destino": "https://example.com/a?b=1#c",
		"futuro":  "https://example.com/futuro",
	} {
		got, err := e.svc.Resolve(ctx, code, true)
		if err != nil || got != want {
			t.Errorf("Resolve(%q) = %q, %v; quiero %q", code, got, err, want)
		}
	}
}

// RES-02: un código inexistente da ErrNotFound.
func TestRES02_CodigoInexistenteDaErrNotFound(t *testing.T) {
	e := newEnv(t)
	got, err := e.svc.Resolve(ctx, "nada", true)
	if !errors.Is(err, link.ErrNotFound) {
		t.Errorf("err = %v, quiero ErrNotFound", err)
	}
	if got != "" {
		t.Errorf("URL = %q, quiero vacía", got)
	}
}

// RES-03: un código caducado da ErrExpired (CAD-05: igual o posterior a expires_at).
func TestRES03_CodigoCaducadoDaErrExpired(t *testing.T) {
	e := newEnv(t)
	e.mustCreate(t, link.Input{URL: "https://example.com", Alias: "temporal", ExpiresAt: "2026-10-01T12:00:10Z"})

	e.clock.Advance(9 * time.Second) // 12:00:09: un segundo antes
	if _, err := e.svc.Resolve(ctx, "temporal", true); err != nil {
		t.Errorf("antes de la caducidad: err = %v, quiero nil", err)
	}
	e.clock.Advance(time.Second) // 12:00:10: justo en el instante (CAD-05)
	got, err := e.svc.Resolve(ctx, "temporal", true)
	if !errors.Is(err, link.ErrExpired) {
		t.Errorf("en el instante de caducidad: err = %v, quiero ErrExpired", err)
	}
	if got != "" {
		t.Errorf("URL = %q, quiero vacía", got)
	}
	e.clock.Advance(time.Hour)
	if _, err := e.svc.Resolve(ctx, "temporal", true); !errors.Is(err, link.ErrExpired) {
		t.Errorf("mucho después: err = %v, quiero ErrExpired", err)
	}
}

// RES-04: la búsqueda es exacta y distingue mayúsculas.
func TestRES04_LaBusquedaDistingueMayusculas(t *testing.T) {
	e := newEnv(t)
	e.mustCreate(t, link.Input{URL: "https://example.com", Alias: "oferta"})
	if _, err := e.svc.Resolve(ctx, "Oferta", true); !errors.Is(err, link.ErrNotFound) {
		t.Errorf("Resolve(\"Oferta\"): err = %v, quiero ErrNotFound", err)
	}
	if got, err := e.svc.Resolve(ctx, "oferta", true); err != nil || got != "https://example.com" {
		t.Errorf("Resolve(\"oferta\") = %q, %v", got, err)
	}
}

// ---------- Visitas ----------

// VIS-01: un enlace recién creado tiene 0 visitas.
func TestVIS01_UnEnlaceNuevoTieneCeroVisitas(t *testing.T) {
	e := newEnv(t, "aaaaaa")
	l := e.mustCreate(t, in("https://example.com"))
	if l.Visits != 0 {
		t.Errorf("Visits devueltas = %d, quiero 0", l.Visits)
	}
	if got := e.visits(t, l.Code); got != 0 {
		t.Errorf("Visits en el listado = %d, quiero 0", got)
	}
}

// VIS-02: 50 resoluciones simultáneas del mismo código dejan el contador en 50 (ejecutar con -race).
func TestVIS02_CincuentaResolucionesSimultaneasSuman50(t *testing.T) {
	e := newEnv(t)
	e.mustCreate(t, link.Input{URL: "https://example.com", Alias: "popular"})

	const n = 50
	var wg sync.WaitGroup
	start := make(chan struct{})
	errs := make(chan error, n)
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			if _, err := e.svc.Resolve(ctx, "popular", true); err != nil {
				errs <- err
			}
		}()
	}
	close(start)
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Errorf("Resolve concurrente: %v", err)
	}
	if got := e.visits(t, "popular"); got != n {
		t.Errorf("Visits = %d, quiero %d", got, n)
	}
}

// VIS-02: cada resolución con éxito suma exactamente 1 visita.
func TestVIS02_CadaResolucionConExitoSumaUna(t *testing.T) {
	e := newEnv(t)
	e.mustCreate(t, link.Input{URL: "https://example.com", Alias: "uno"})
	for i := int64(1); i <= 3; i++ {
		if _, err := e.svc.Resolve(ctx, "uno", true); err != nil {
			t.Fatal(err)
		}
		if got := e.visits(t, "uno"); got != i {
			t.Errorf("tras %d resoluciones Visits = %d", i, got)
		}
	}
}

// VIS-03: una resolución que falla (no existe, caducado) no suma nada.
func TestVIS03_UnaResolucionFallidaNoSumaNada(t *testing.T) {
	e := newEnv(t)
	e.mustCreate(t, link.Input{URL: "https://example.com", Alias: "vivo"})
	e.mustCreate(t, link.Input{URL: "https://example.com", Alias: "muerto", ExpiresAt: "2026-10-01T12:00:10Z"})
	if _, err := e.svc.Resolve(ctx, "muerto", true); err != nil { // una visita válida
		t.Fatal(err)
	}
	e.clock.Advance(time.Hour)

	if _, err := e.svc.Resolve(ctx, "muerto", true); !errors.Is(err, link.ErrExpired) {
		t.Fatalf("err = %v, quiero ErrExpired", err)
	}
	if _, err := e.svc.Resolve(ctx, "nada", true); !errors.Is(err, link.ErrNotFound) {
		t.Fatalf("err = %v, quiero ErrNotFound", err)
	}
	if got := e.visits(t, "muerto"); got != 1 {
		t.Errorf("Visits del caducado = %d, quiero 1 (la de antes de caducar)", got)
	}
	if got := e.visits(t, "vivo"); got != 0 {
		t.Errorf("Visits del vigente = %d, quiero 0", got)
	}
}

// VIS-04: resolver sin contar devuelve lo mismo que RES-01 a RES-04 y no suma.
func TestVIS04_ResolverSinContarNoSumaYDevuelveLoMismo(t *testing.T) {
	e := newEnv(t)
	e.mustCreate(t, link.Input{URL: "https://example.com/x", Alias: "vivo"})
	e.mustCreate(t, link.Input{URL: "https://example.com/y", Alias: "temporal", ExpiresAt: "2026-10-01T12:00:10Z"})

	got, err := e.svc.Resolve(ctx, "vivo", false)
	if err != nil || got != "https://example.com/x" {
		t.Errorf("Resolve(vivo, false) = %q, %v; quiero la URL y nil (RES-01)", got, err)
	}
	if _, err := e.svc.Resolve(ctx, "nada", false); !errors.Is(err, link.ErrNotFound) {
		t.Errorf("inexistente: err = %v, quiero ErrNotFound (RES-02)", err)
	}
	if _, err := e.svc.Resolve(ctx, "Vivo", false); !errors.Is(err, link.ErrNotFound) {
		t.Errorf("mayúsculas: err = %v, quiero ErrNotFound (RES-04)", err)
	}
	e.clock.Advance(time.Hour)
	if _, err := e.svc.Resolve(ctx, "temporal", false); !errors.Is(err, link.ErrExpired) {
		t.Errorf("caducado: err = %v, quiero ErrExpired (RES-03)", err)
	}
	for _, code := range []string{"vivo", "temporal"} {
		if v := e.visits(t, code); v != 0 {
			t.Errorf("Visits de %q = %d, quiero 0", code, v)
		}
	}
}

// ---------- Listar ----------

// LIS-01: del más reciente al más antiguo según el orden de creación.
func TestLIS01_ListaDelMasRecienteAlMasAntiguo(t *testing.T) {
	e := newEnv(t, "aaaaaa", "bbbbbb", "cccccc")
	// El reloj no avanza: el orden es el de creación, no el de la fecha.
	e.mustCreate(t, in("https://example.com/1"))
	e.mustCreate(t, in("https://example.com/2"))
	e.mustCreate(t, in("https://example.com/3"))

	var got []string
	for _, l := range e.mustList(t) {
		got = append(got, l.Code)
	}
	want := []string{"cccccc", "bbbbbb", "aaaaaa"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("orden = %v, quiero %v", got, want)
	}
}

// LIS-02: cada enlace lleva sus datos y se puede saber si está caducado con l.Expired(svc.Now()).
func TestLIS02_CadaEnlaceLlevaSusDatosYSiEstaCaducado(t *testing.T) {
	e := newEnv(t)
	e.mustCreate(t, link.Input{URL: "https://example.com/p", Alias: "perpetuo"})
	e.clock.Advance(time.Minute)
	e.mustCreate(t, link.Input{URL: "https://example.com/c", Alias: "corto", ExpiresAt: "2026-10-01T12:05:00Z"})
	for i := 0; i < 2; i++ {
		if _, err := e.svc.Resolve(ctx, "corto", true); err != nil {
			t.Fatal(err)
		}
	}
	e.clock.Advance(10 * time.Minute) // 12:11: "corto" ya caducó

	byCode := map[string]link.Link{}
	for _, l := range e.mustList(t) {
		byCode[l.Code] = l
	}
	p, c := byCode["perpetuo"], byCode["corto"]
	now := e.svc.Now()

	if p.URL != "https://example.com/p" || p.Visits != 0 || !p.CreatedAt.Equal(t0) || p.ExpiresAt != nil || p.Expired(now) {
		t.Errorf("perpetuo = %+v (expired=%v)", p, p.Expired(now))
	}
	wantExp := time.Date(2026, 10, 1, 12, 5, 0, 0, time.UTC)
	if c.URL != "https://example.com/c" || c.Visits != 2 || !c.CreatedAt.Equal(t0.Add(time.Minute)) ||
		c.ExpiresAt == nil || !c.ExpiresAt.Equal(wantExp) || !c.Expired(now) {
		t.Errorf("corto = %+v (expired=%v)", c, c.Expired(now))
	}
}

// LIS-03: sin enlaces, lista vacía (no nil) y sin error.
func TestLIS03_SinEnlacesDevuelveListaVaciaSinError(t *testing.T) {
	e := newEnv(t)
	got, err := e.svc.List(ctx)
	if err != nil {
		t.Fatalf("err = %v, quiero nil", err)
	}
	if got == nil || len(got) != 0 {
		t.Errorf("List = %#v, quiero una lista vacía no nil", got)
	}
}

// ---------- Borrar ----------

// BOR-01: borrar quita el enlace del listado y resolverlo da ErrNotFound.
func TestBOR01_BorrarQuitaElEnlaceDelListadoYDeResolver(t *testing.T) {
	e := newEnv(t)
	e.mustCreate(t, link.Input{URL: "https://example.com/1", Alias: "borrame"})
	e.mustCreate(t, link.Input{URL: "https://example.com/2", Alias: "quedate"})

	if err := e.svc.Delete(ctx, "borrame"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, ok := e.find(t, "borrame"); ok {
		t.Error("el enlace borrado sigue en el listado")
	}
	if _, ok := e.find(t, "quedate"); !ok {
		t.Error("Delete ha quitado otro enlace")
	}
	if _, err := e.svc.Resolve(ctx, "borrame", true); !errors.Is(err, link.ErrNotFound) {
		t.Errorf("Resolve tras borrar: err = %v, quiero ErrNotFound", err)
	}
}

// BOR-02: borrar un código inexistente da ErrNotFound.
func TestBOR02_BorrarUnCodigoInexistenteDaErrNotFound(t *testing.T) {
	e := newEnv(t)
	if err := e.svc.Delete(ctx, "nada"); !errors.Is(err, link.ErrNotFound) {
		t.Errorf("err = %v, quiero ErrNotFound", err)
	}
}

// BOR-03: tras borrar, el código vuelve a estar libre.
func TestBOR03_TrasBorrarElCodigoVuelveAEstarLibre(t *testing.T) {
	e := newEnv(t)
	e.mustCreate(t, link.Input{URL: "https://example.com/viejo", Alias: "reutil"})
	if err := e.svc.Delete(ctx, "reutil"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	l, err := e.svc.Create(ctx, link.Input{URL: "https://example.com/nuevo", Alias: "reutil"})
	if err != nil {
		t.Fatalf("recrear el alias: %v", err)
	}
	if l.Code != "reutil" || l.Visits != 0 {
		t.Errorf("enlace = %+v, quiero código reutil y 0 visitas", l)
	}
	if got, err := e.svc.Resolve(ctx, "reutil", false); err != nil || got != "https://example.com/nuevo" {
		t.Errorf("Resolve = %q, %v; quiero la URL nueva", got, err)
	}
}

// ---------- Fallos del almacén (no de negocio) ----------

var errBoom = errors.New("disco roto")

// brokenStore falla en todo con errBoom y cuenta las inserciones.
type brokenStore struct {
	mu      sync.Mutex
	inserts int
}

func (b *brokenStore) Insert(context.Context, link.Link) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.inserts++
	return fmt.Errorf("insertar: %w", errBoom)
}
func (b *brokenStore) Get(context.Context, string) (link.Link, error) {
	return link.Link{}, errBoom
}
func (b *brokenStore) List(context.Context) ([]link.Link, error) { return nil, errBoom }
func (b *brokenStore) Delete(context.Context, string) error      { return errBoom }
func (b *brokenStore) AddVisit(context.Context, string) error    { return errBoom }

// COD-02 y RES-05: un error del almacén que no es de negocio se devuelve envuelto, sin reintentos.
func TestFalloDelAlmacenSePropagaSinTraducirse(t *testing.T) {
	newBroken := func() (*shortener.Service, *brokenStore, *codeGen) {
		bs := &brokenStore{}
		gen := &codeGen{codes: []string{"aaaaaa", "bbbbbb", "cccccc", "dddddd", "eeeeee"}}
		clock := &fakeClock{cur: t0}
		return shortener.New(bs, clock.Now, gen.Next), bs, gen
	}

	t.Run("Create: un error de Insert que no es ErrCodeTaken no se reintenta", func(t *testing.T) {
		svc, bs, gen := newBroken()
		_, err := svc.Create(ctx, in("https://example.com"))
		if !errors.Is(err, errBoom) {
			t.Fatalf("err = %v, quiero el error del almacén", err)
		}
		if errors.Is(err, link.ErrNoCodeAvailable) {
			t.Error("un fallo del almacén no es ErrNoCodeAvailable")
		}
		if bs.inserts != 1 {
			t.Errorf("inserciones = %d, quiero 1 (sin reintentos)", bs.inserts)
		}
		if n := gen.Calls(); n != 1 {
			t.Errorf("llamadas al generador = %d, quiero 1 (sin reintentos)", n)
		}
	})
	t.Run("Create con alias", func(t *testing.T) {
		svc, _, _ := newBroken()
		_, err := svc.Create(ctx, link.Input{URL: "https://example.com", Alias: "promo"})
		var taken *link.AliasTakenError
		if !errors.Is(err, errBoom) || errors.As(err, &taken) {
			t.Errorf("err = %v, quiero el error del almacén", err)
		}
	})
	t.Run("Resolve", func(t *testing.T) {
		svc, _, _ := newBroken()
		if _, err := svc.Resolve(ctx, "abc", true); !errors.Is(err, errBoom) {
			t.Errorf("err = %v, quiero el error del almacén", err)
		}
	})
	t.Run("List", func(t *testing.T) {
		svc, _, _ := newBroken()
		if _, err := svc.List(ctx); !errors.Is(err, errBoom) {
			t.Errorf("err = %v, quiero el error del almacén", err)
		}
	})
	t.Run("Delete", func(t *testing.T) {
		svc, _, _ := newBroken()
		if err := svc.Delete(ctx, "abc"); !errors.Is(err, errBoom) {
			t.Errorf("err = %v, quiero el error del almacén", err)
		}
	})
}

// ---------- RES-05: no se puede contar la visita ----------

// visitStore devuelve siempre el mismo enlace en Get y hace que AddVisit
// devuelva addVisitErr.
type visitStore struct {
	brokenStore
	link        link.Link
	addVisitErr error
	addVisits   int
}

func (v *visitStore) Get(context.Context, string) (link.Link, error) { return v.link, nil }
func (v *visitStore) AddVisit(context.Context, string) error {
	v.addVisits++
	return v.addVisitErr
}

// RES-05: si no se puede sumar la visita no se devuelve la URL; si el enlace se borró entre medias, ErrNotFound.
func TestRES05_SiNoSePuedeSumarLaVisitaNoSeDevuelveLaURL(t *testing.T) {
	newSvc := func(addVisitErr error) (*shortener.Service, *visitStore) {
		vs := &visitStore{
			link:        link.Link{Code: "abc", URL: "https://example.com", CreatedAt: t0},
			addVisitErr: addVisitErr,
		}
		clock := &fakeClock{cur: t0}
		return shortener.New(vs, clock.Now, nil), vs
	}

	t.Run("AddVisit falla: se devuelve el error del almacén y ninguna URL", func(t *testing.T) {
		svc, vs := newSvc(fmt.Errorf("sumar visita: %w", errBoom))
		got, err := svc.Resolve(ctx, "abc", true)
		if !errors.Is(err, errBoom) {
			t.Errorf("err = %v, quiero el error del almacén (errors.Is)", err)
		}
		if got != "" {
			t.Errorf("URL = %q, quiero vacía", got)
		}
		if vs.addVisits != 1 {
			t.Errorf("llamadas a AddVisit = %d, quiero 1", vs.addVisits)
		}
	})

	t.Run("borrado entre Get y AddVisit: ErrNotFound", func(t *testing.T) {
		svc, _ := newSvc(fmt.Errorf("sumar visita a \"abc\": %w", link.ErrNotFound))
		got, err := svc.Resolve(ctx, "abc", true)
		if !errors.Is(err, link.ErrNotFound) {
			t.Errorf("err = %v, quiero ErrNotFound", err)
		}
		if got != "" {
			t.Errorf("URL = %q, quiero vacía", got)
		}
	})

	t.Run("sin contar visita no se llama a AddVisit", func(t *testing.T) {
		svc, vs := newSvc(errBoom)
		got, err := svc.Resolve(ctx, "abc", false)
		if err != nil || got != "https://example.com" {
			t.Errorf("Resolve = %q, %v; quiero la URL y nil", got, err)
		}
		if vs.addVisits != 0 {
			t.Errorf("llamadas a AddVisit = %d, quiero 0", vs.addVisits)
		}
	})
}
