package link

import (
	"bytes"
	"errors"
	"io"
	"reflect"
	"strings"
	"testing"
	"time"
)

// Instante fijo para todos los tests: nunca se usa time.Now().
var now = time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC)

func ptr(t time.Time) *time.Time { return &t }

// validateOne valida una entrada y devuelve el resultado.
func validateOne(in Input) (Valid, []string) { return Validate(in, now) }

func TestValidateURL(t *testing.T) {
	long := "https://example.com/" + strings.Repeat("a", 2048-len("https://example.com/"))
	// Caracteres de dos bytes: 20 bytes de prefijo + 1014 ñ = 2048 bytes
	// exactos, aunque solo son 1034 caracteres.
	const prefix = "https://example.com/"
	multi2048 := prefix + strings.Repeat("ñ", 1014)
	tests := []struct {
		name    string
		url     string
		want    string // URL guardada si se acepta
		wantMsg string // mensaje si se rechaza
	}{
		{"URL-01 se guarda sin normalizar", "https://example.com/a?b=1#c", "https://example.com/a?b=1#c", ""},
		{"URL-02 recorta espacios", "  https://example.com  ", "https://example.com", ""},
		{"URL-03 vacía", "", "", "url: es obligatoria"},
		{"URL-03 solo espacios", "   \t ", "", "url: es obligatoria"},
		{"URL-04 2048 bytes exactos se acepta", long, long, ""},
		{"URL-04 2049 bytes", long + "a", "", "url: no puede superar los 2048 bytes"},
		{"URL-04 2048 bytes con caracteres de dos bytes se acepta", multi2048, multi2048, ""},
		{"URL-04 2049 bytes con caracteres de dos bytes", multi2048 + "a", "", "url: no puede superar los 2048 bytes"},
		{"URL-04 se miden bytes, no caracteres (menos de 2048 caracteres)", prefix + strings.Repeat("ñ", 1015), "", "url: no puede superar los 2048 bytes"},
		{"URL-04 se mide después de recortar espacios", "  " + long + "  ", long, ""},
		{"URL-05 espacio en el dominio", "http://exa mple.com", "", "url: no es una URL válida"},
		{"URL-06 ftp", "ftp://example.com/f", "", "url: debe empezar por http:// o https://"},
		{"URL-06 javascript", "javascript:alert(1)", "", "url: debe empezar por http:// o https://"},
		{"URL-06 mailto", "mailto:a@b.c", "", "url: debe empezar por http:// o https://"},
		{"URL-06 sin esquema", "example.com", "", "url: debe empezar por http:// o https://"},
		{"URL-06 ruta sin esquema", "/ruta", "", "url: debe empezar por http:// o https://"},
		{"URL-07 esquema en mayúsculas", "HTTPS://Example.com", "HTTPS://Example.com", ""},
		{"URL-07 http en http", "http://example.com", "http://example.com", ""},
		{"URL-08 sin nada tras el esquema", "https://", "", "url: falta el dominio"},
		{"URL-08 tres barras", "http:///ruta", "", "url: falta el dominio"},
		{"URL-08 solo puerto", "https://:8080", "", "url: falta el dominio"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, msgs := validateOne(Input{URL: tt.url})
			if tt.wantMsg != "" {
				if !reflect.DeepEqual(msgs, []string{tt.wantMsg}) {
					t.Fatalf("mensajes = %q, quiero [%q]", msgs, tt.wantMsg)
				}
				return
			}
			if len(msgs) != 0 {
				t.Fatalf("no esperaba errores, hay %q", msgs)
			}
			if got.URL != tt.want {
				t.Errorf("URL = %q, quiero %q", got.URL, tt.want)
			}
		})
	}
}

func TestValidateAlias(t *testing.T) {
	const (
		msgLen      = "alias: debe tener entre 3 y 32 caracteres"
		msgChars    = "alias: solo admite minúsculas, números y guiones"
		msgHyphen   = "alias: no puede empezar ni terminar por guion"
		validURL    = "https://example.com"
		thirtyTwo   = "abcdefghijklmnopqrstuvwxyz012345"
		thirtyThree = thirtyTwo + "6"
	)
	tests := []struct {
		name      string
		alias     string
		wantAlias string
		wantMsg   string
	}{
		{"ALI-01 alias válido", "oferta-otono", "oferta-otono", ""},
		{"ALI-01 se recortan espacios", "  oferta-otono  ", "oferta-otono", ""},
		{"ALI-01 vacío cuenta como no enviado", "", "", ""},
		{"ALI-01 solo espacios cuenta como no enviado", "   ", "", ""},
		{"ALI-02 2 caracteres", "ab", "", msgLen},
		{"ALI-02 3 caracteres se acepta", "abc", "abc", ""},
		{"ALI-02 32 caracteres se acepta", thirtyTwo, thirtyTwo, ""},
		{"ALI-02 33 caracteres", thirtyThree, "", msgLen},
		{"ALI-02 se cuentan caracteres, no bytes: ñu son 2", "ñu", "", msgLen},
		{"ALI-02 ñus son 3 caracteres: pasa la longitud y falla ALI-03", "ñus", "", msgChars},
		{"ALI-02 17 ñ son 34 bytes pero 17 caracteres: falla ALI-03, no la longitud", strings.Repeat("ñ", 17), "", msgChars},
		{"ALI-02 33 ñ son 33 caracteres", strings.Repeat("ñ", 33), "", msgLen},
		{"ALI-03 mayúscula", "Oferta", "", msgChars},
		{"ALI-03 guion bajo", "mi_enlace", "", msgChars},
		{"ALI-03 eñe", "año-nuevo", "", msgChars},
		{"ALI-03 espacios internos", "a b c", "", msgChars},
		{"ALI-04 empieza por guion", "-oferta", "", msgHyphen},
		{"ALI-04 termina por guion", "oferta-", "", msgHyphen},
		{"ALI-04 guion interior se acepta", "o-f-e", "o-f-e", ""},
		{"ALI-05 api", "api", "", `alias: "api" está reservado`},
		{"ALI-05 assets", "assets", "", `alias: "assets" está reservado`},
		{"ALI-05 no es reservado si solo contiene la palabra", "apis", "apis", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, msgs := validateOne(Input{URL: validURL, Alias: tt.alias})
			if tt.wantMsg != "" {
				if !reflect.DeepEqual(msgs, []string{tt.wantMsg}) {
					t.Fatalf("mensajes = %q, quiero [%q]", msgs, tt.wantMsg)
				}
				return
			}
			if len(msgs) != 0 {
				t.Fatalf("no esperaba errores, hay %q", msgs)
			}
			if got.Alias != tt.wantAlias {
				t.Errorf("Alias = %q, quiero %q", got.Alias, tt.wantAlias)
			}
		})
	}
}

func TestValidateExpiresAt(t *testing.T) {
	const (
		msgFormat = "expires_at: debe tener formato RFC 3339 (2026-12-31T23:59:59Z)"
		msgFuture = "expires_at: debe ser una fecha futura"
	)
	tests := []struct {
		name    string
		in      string
		want    *time.Time
		wantMsg string
	}{
		{"CAD-01 sin expires_at", "", nil, ""},
		{"CAD-01 solo espacios cuenta como no enviado", "  ", nil, ""},
		{"CAD-02 texto libre", "mañana", nil, msgFormat},
		{"CAD-02 solo fecha", "2026-12-31", nil, msgFormat},
		{"CAD-03 igual al instante de crear", "2026-10-01T12:00:00Z", nil, msgFuture},
		{"CAD-03 anterior", "2026-09-30T23:59:59Z", nil, msgFuture},
		{"CAD-03 un segundo después se acepta", "2026-10-01T12:00:01Z", ptr(now.Add(time.Second)), ""},
		{"CAD-03 con fracciones, mismo segundo que now: igual y da error", "2026-10-01T12:00:00.5Z", nil, msgFuture},
		{"CAD-03 con fracciones casi un segundo después: sigue siendo igual", "2026-10-01T12:00:00.999Z", nil, msgFuture},
		{"CAD-03 con fracciones, segundo siguiente: se acepta truncado", "2026-10-01T12:00:01.5Z", ptr(now.Add(time.Second)), ""},
		{"CAD-03 igual a now pero con otra zona", "2026-10-01T14:00:00+02:00", nil, msgFuture},
		{"CAD-04 zona horaria se convierte a UTC", "2026-12-31T23:59:59+02:00", ptr(time.Date(2026, 12, 31, 21, 59, 59, 0, time.UTC)), ""},
		{"CAD-04 sin fracciones de segundo", "2026-12-31T23:59:59.987Z", ptr(time.Date(2026, 12, 31, 23, 59, 59, 0, time.UTC)), ""},
		{"CAD-04 se recortan espacios", " 2026-12-31T23:59:59Z ", ptr(time.Date(2026, 12, 31, 23, 59, 59, 0, time.UTC)), ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, msgs := validateOne(Input{URL: "https://example.com", ExpiresAt: tt.in})
			if tt.wantMsg != "" {
				if !reflect.DeepEqual(msgs, []string{tt.wantMsg}) {
					t.Fatalf("mensajes = %q, quiero [%q]", msgs, tt.wantMsg)
				}
				return
			}
			if len(msgs) != 0 {
				t.Fatalf("no esperaba errores, hay %q", msgs)
			}
			switch {
			case tt.want == nil && got.ExpiresAt != nil:
				t.Fatalf("ExpiresAt = %v, quiero nil", got.ExpiresAt)
			case tt.want != nil && got.ExpiresAt == nil:
				t.Fatalf("ExpiresAt = nil, quiero %v", tt.want)
			case tt.want != nil:
				if !got.ExpiresAt.Equal(*tt.want) {
					t.Errorf("ExpiresAt = %v, quiero %v", got.ExpiresAt, tt.want)
				}
				if got.ExpiresAt.Location() != time.UTC {
					t.Errorf("ExpiresAt no está en UTC: %v", got.ExpiresAt.Location())
				}
				if got.ExpiresAt.Nanosecond() != 0 {
					t.Errorf("ExpiresAt tiene fracciones: %v", got.ExpiresAt)
				}
			}
		})
	}
}

// Orden y forma de los errores de validación: todos a la vez, uno por campo,
// en el orden url, alias, expires_at.
func TestValidateAllErrorsAtOnce(t *testing.T) {
	tests := []struct {
		name string
		in   Input
		want []string
	}{
		{
			"los tres campos fallan",
			Input{URL: "", Alias: "Mal_Alias", ExpiresAt: "mañana"},
			[]string{
				"url: es obligatoria",
				"alias: solo admite minúsculas, números y guiones",
				"expires_at: debe tener formato RFC 3339 (2026-12-31T23:59:59Z)",
			},
		},
		{
			"solo url y expires_at",
			Input{URL: "example.com", ExpiresAt: "2020-01-01T00:00:00Z"},
			[]string{
				"url: debe empezar por http:// o https://",
				"expires_at: debe ser una fecha futura",
			},
		},
		{
			"solo alias y expires_at, saltándose url",
			Input{URL: "https://example.com", Alias: "api", ExpiresAt: "2026-10-01T12:00:00Z"},
			[]string{
				`alias: "api" está reservado`,
				"expires_at: debe ser una fecha futura",
			},
		},
		{
			"un alias con varios fallos da solo el primero (longitud)",
			Input{URL: "https://example.com", Alias: "-A"},
			[]string{"alias: debe tener entre 3 y 32 caracteres"},
		},
		{
			"un alias con varios fallos da solo el primero (caracteres antes que guion)",
			Input{URL: "https://example.com", Alias: "-Abc"},
			[]string{"alias: solo admite minúsculas, números y guiones"},
		},
		{
			"una url con varios fallos da solo el primero (esquema antes que dominio)",
			Input{URL: "ftp://"},
			[]string{"url: debe empezar por http:// o https://"},
		},
		{
			"entrada correcta: sin mensajes",
			Input{URL: "https://example.com", Alias: "promo", ExpiresAt: "2027-01-01T00:00:00Z"},
			nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, msgs := validateOne(tt.in)
			if len(msgs) == 0 && len(tt.want) == 0 {
				return
			}
			if !reflect.DeepEqual(msgs, tt.want) {
				t.Errorf("mensajes = %q, quiero %q", msgs, tt.want)
			}
		})
	}
}

// Con mensajes, Validate devuelve el valor cero de Valid.
func TestValidateEmptyValidOnErrors(t *testing.T) {
	tests := []struct {
		name string
		in   Input
	}{
		{"solo falla la url", Input{URL: "", Alias: "promo", ExpiresAt: "2027-01-01T00:00:00Z"}},
		{"solo falla el alias", Input{URL: "https://example.com", Alias: "Promo", ExpiresAt: "2027-01-01T00:00:00Z"}},
		{"solo falla expires_at", Input{URL: "https://example.com", Alias: "promo", ExpiresAt: "mañana"}},
		{"fallan los tres", Input{URL: "x", Alias: "A", ExpiresAt: "y"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, msgs := validateOne(tt.in)
			if len(msgs) == 0 {
				t.Fatal("esperaba mensajes de error")
			}
			if !reflect.DeepEqual(got, Valid{}) {
				t.Errorf("Valid = %+v, quiero el valor cero", got)
			}
		})
	}
}

// CAD-03 con un reloj que también trae fracciones: se comparan sin ellas.
func TestValidateExpiresAtClockWithFraction(t *testing.T) {
	clock := time.Date(2026, 10, 1, 12, 0, 0, 700_000_000, time.UTC)
	_, msgs := Validate(Input{URL: "https://example.com", ExpiresAt: "2026-10-01T12:00:00.9Z"}, clock)
	if want := []string{"expires_at: debe ser una fecha futura"}; !reflect.DeepEqual(msgs, want) {
		t.Errorf("CAD-03 mensajes = %q, quiero %q", msgs, want)
	}
}

func TestValidateReturnsCleanValid(t *testing.T) {
	got, msgs := validateOne(Input{
		URL:       "  https://example.com/x  ",
		Alias:     " promo ",
		ExpiresAt: " 2026-12-31T23:59:59+02:00 ",
	})
	if len(msgs) != 0 {
		t.Fatalf("no esperaba errores, hay %q", msgs)
	}
	want := Valid{
		URL:       "https://example.com/x",
		Alias:     "promo",
		ExpiresAt: ptr(time.Date(2026, 12, 31, 21, 59, 59, 0, time.UTC)),
	}
	if got.URL != want.URL || got.Alias != want.Alias ||
		got.ExpiresAt == nil || !got.ExpiresAt.Equal(*want.ExpiresAt) {
		t.Errorf("Valid = %+v, quiero %+v", got, want)
	}
}

// cyclic es un io.Reader determinista e infinito: 0, 1, 2, … 255, 0, 1, …
type cyclic struct{ next byte }

func (c *cyclic) Read(p []byte) (int, error) {
	for i := range p {
		p[i] = c.next
		c.next++
	}
	return len(p), nil
}

func TestNewCode(t *testing.T) {
	const alphabet = "abcdefghijklmnopqrstuvwxyz0123456789"

	check := func(t *testing.T, code string) {
		t.Helper()
		if len(code) != 6 {
			t.Errorf("código %q: longitud %d, quiero 6", code, len(code))
		}
		for _, r := range code {
			if !strings.ContainsRune(alphabet, r) {
				t.Errorf("código %q: carácter %q fuera del alfabeto", code, r)
			}
		}
	}

	t.Run("COD-01 longitud y alfabeto", func(t *testing.T) {
		code, err := NewCode(&cyclic{})
		if err != nil {
			t.Fatal(err)
		}
		check(t, code)
	})

	t.Run("COD-01 muchos códigos, todos válidos", func(t *testing.T) {
		r := &cyclic{next: 7}
		seen := map[rune]bool{}
		for i := 0; i < 500; i++ {
			code, err := NewCode(r)
			if err != nil {
				t.Fatal(err)
			}
			check(t, code)
			for _, c := range code {
				seen[c] = true
			}
		}
		if len(seen) != len(alphabet) {
			t.Errorf("se han visto %d caracteres distintos, quiero los %d del alfabeto", len(seen), len(alphabet))
		}
	})

	t.Run("COD-01 determinista con la misma fuente", func(t *testing.T) {
		a, errA := NewCode(&cyclic{next: 3})
		b, errB := NewCode(&cyclic{next: 3})
		if errA != nil || errB != nil {
			t.Fatal(errA, errB)
		}
		if a != b {
			t.Errorf("misma fuente, códigos distintos: %q y %q", a, b)
		}
		check(t, a)
	})

	t.Run("COD-01 fuentes distintas dan códigos distintos", func(t *testing.T) {
		a, _ := NewCode(&cyclic{next: 0})
		b, _ := NewCode(&cyclic{next: 100})
		if a == b {
			t.Errorf("fuentes distintas, mismo código %q", a)
		}
	})

	t.Run("COD-01 ejemplo de la spec: 255,0,1,2,3,4,5 da abcdef", func(t *testing.T) {
		code, err := NewCode(bytes.NewReader([]byte{255, 0, 1, 2, 3, 4, 5}))
		if err != nil {
			t.Fatal(err)
		}
		if code != "abcdef" {
			t.Errorf("código = %q, quiero abcdef", code)
		}
	})

	t.Run("COD-01 el byte b da la posición b%36", func(t *testing.T) {
		// 35 -> '9'; 36 -> 'a'; 251 -> 251%36=35 -> '9'; 0,1,2 -> 'a','b','c'.
		code, err := NewCode(bytes.NewReader([]byte{35, 36, 251, 0, 1, 2}))
		if err != nil {
			t.Fatal(err)
		}
		if code != "9a9abc" {
			t.Errorf("código = %q, quiero 9a9abc", code)
		}
	})

	t.Run("COD-01 se descartan los bytes de 252 en adelante", func(t *testing.T) {
		code, err := NewCode(bytes.NewReader([]byte{252, 253, 254, 255, 0, 252, 1, 2, 255, 3, 4, 5}))
		if err != nil {
			t.Fatal(err)
		}
		if code != "abcdef" {
			t.Errorf("código = %q, quiero abcdef", code)
		}
	})

	t.Run("COD-01 fuente que solo da bytes descartables se agota con error", func(t *testing.T) {
		code, err := NewCode(bytes.NewReader([]byte{252, 253, 254, 255}))
		if err == nil || code != "" {
			t.Fatalf("code = %q, err = %v; quiero error y código vacío", code, err)
		}
		if !errors.Is(err, io.EOF) && !errors.Is(err, io.ErrUnexpectedEOF) {
			t.Errorf("err = %v, quiero que envuelva io.EOF o io.ErrUnexpectedEOF", err)
		}
	})

	t.Run("COD-01 fuente que se agota a mitad de código", func(t *testing.T) {
		code, err := NewCode(bytes.NewReader([]byte{0, 1, 2}))
		if err == nil || code != "" {
			t.Fatalf("code = %q, err = %v; quiero error y código vacío", code, err)
		}
		if !errors.Is(err, io.EOF) && !errors.Is(err, io.ErrUnexpectedEOF) {
			t.Errorf("err = %v, quiero que envuelva io.EOF o io.ErrUnexpectedEOF", err)
		}
	})

	t.Run("COD-01 fuente vacía", func(t *testing.T) {
		code, err := NewCode(strings.NewReader(""))
		if err == nil {
			t.Fatalf("esperaba error con la fuente agotada, devolvió %q", code)
		}
		if code != "" {
			t.Errorf("con error el código debe ser vacío, es %q", code)
		}
		if !errors.Is(err, io.EOF) && !errors.Is(err, io.ErrUnexpectedEOF) {
			t.Errorf("err = %v, quiero que envuelva io.EOF o io.ErrUnexpectedEOF", err)
		}
	})

	t.Run("COD-01 error propio del lector se conserva", func(t *testing.T) {
		boom := errors.New("boom")
		_, err := NewCode(errReader{boom})
		if !errors.Is(err, boom) {
			t.Errorf("err = %v, quiero que envuelva %v", err, boom)
		}
	})
}

type errReader struct{ err error }

func (e errReader) Read([]byte) (int, error) { return 0, e.err }

var _ io.Reader = errReader{}

func TestReserved(t *testing.T) {
	tests := []struct {
		code string
		want bool
	}{
		{"api", true},
		{"assets", true},
		{"API", false}, // la comparación es exacta
		{"apis", false},
		{"oferta", false},
		{"", false},
	}
	for _, tt := range tests {
		t.Run("ALI-05 Reserved("+tt.code+")", func(t *testing.T) {
			if got := Reserved(tt.code); got != tt.want {
				t.Errorf("Reserved(%q) = %v, quiero %v", tt.code, got, tt.want)
			}
		})
	}
}

func TestLinkExpired(t *testing.T) {
	tests := []struct {
		name    string
		expires *time.Time
		now     time.Time
		want    bool
	}{
		{"CAD-05 sin caducidad nunca caduca", nil, now.AddDate(100, 0, 0), false},
		{"CAD-05 antes de expires_at", ptr(now), now.Add(-time.Second), false},
		{"CAD-05 igual a expires_at ya caducó", ptr(now), now, true},
		{"CAD-05 después de expires_at", ptr(now), now.Add(time.Second), true},
		{"CAD-05 compara instantes, no zonas", ptr(now), now.In(time.FixedZone("x", 7200)), true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := Link{Code: "abc", URL: "https://example.com", ExpiresAt: tt.expires}
			if got := l.Expired(tt.now); got != tt.want {
				t.Errorf("Expired = %v, quiero %v", got, tt.want)
			}
		})
	}
}

func TestErrorMessages(t *testing.T) {
	t.Run("ALI-06 AliasTakenError", func(t *testing.T) {
		var err error = &AliasTakenError{Alias: "promo"}
		if got, want := err.Error(), `alias: "promo" ya está en uso`; got != want {
			t.Errorf("mensaje = %q, quiero %q", got, want)
		}
		var target *AliasTakenError
		if !errors.As(err, &target) || target.Alias != "promo" {
			t.Errorf("errors.As no recupera *AliasTakenError")
		}
	})

	t.Run("ValidationError con un mensaje", func(t *testing.T) {
		var err error = &ValidationError{Messages: []string{"url: es obligatoria"}}
		if got := err.Error(); got != "url: es obligatoria" {
			t.Errorf("mensaje = %q", got)
		}
	})

	t.Run("ValidationError con varios mensajes los une con punto y coma", func(t *testing.T) {
		msgs := []string{"url: es obligatoria", `alias: "api" está reservado`}
		got := (&ValidationError{Messages: msgs}).Error()
		if want := `url: es obligatoria; alias: "api" está reservado`; got != want {
			t.Errorf("mensaje = %q, quiero %q", got, want)
		}
	})

	t.Run("ValidationError se recupera con errors.As", func(t *testing.T) {
		err := error(&ValidationError{Messages: []string{"x"}})
		wrapped := errors.Join(errors.New("ctx"), err)
		var target *ValidationError
		if !errors.As(wrapped, &target) || len(target.Messages) != 1 {
			t.Errorf("errors.As no recupera *ValidationError")
		}
	})

	t.Run("errores centinela", func(t *testing.T) {
		tests := []struct {
			err  error
			want string
		}{
			{ErrNotFound, "enlace no encontrado"},
			{ErrExpired, "enlace caducado"},
			{ErrCodeTaken, "código en uso"},
			{ErrNoCodeAvailable, "no se ha podido generar un código libre"},
		}
		for _, tt := range tests {
			if tt.err.Error() != tt.want {
				t.Errorf("mensaje = %q, quiero %q", tt.err.Error(), tt.want)
			}
		}
	})
}
