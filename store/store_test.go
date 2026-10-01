package store

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"acorta/link"

	_ "modernc.org/sqlite"
)

// openTemp abre un almacén real en un directorio temporal y lo cierra al acabar.
func openTemp(t *testing.T) (*Store, string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "acorta.db")
	s, err := Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s, path
}

func ptr(t time.Time) *time.Time { return &t }

func mustInsert(t *testing.T, s *Store, l link.Link) {
	t.Helper()
	if err := s.Insert(context.Background(), l); err != nil {
		t.Fatalf("Insert(%q): %v", l.Code, err)
	}
}

var base = time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)

func TestALM01_AbrirCreaFicheroYEsquema(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "acorta.db")

	s, err := Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Errorf("ALM-01: el fichero no se ha creado: %v", err)
	}
	mustInsert(t, s, link.Link{Code: "uno", URL: "https://example.com/1", CreatedAt: base})
	if err := s.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	// Segunda apertura de la misma ruta: no falla y conserva los datos.
	s2, err := Open(path)
	if err != nil {
		t.Fatalf("ALM-01: segunda apertura: %v", err)
	}
	defer s2.Close()
	got, err := s2.Get(ctx, "uno")
	if err != nil {
		t.Fatalf("ALM-01: Get tras reabrir: %v", err)
	}
	if got.URL != "https://example.com/1" {
		t.Errorf("ALM-01: datos perdidos al reabrir: %+v", got)
	}

	// El esquema existe de verdad: se comprueba con una conexión independiente.
	raw, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	defer raw.Close()
	var n int
	err = raw.QueryRow(`SELECT count(*) FROM sqlite_master WHERE type='table' AND name='links'`).Scan(&n)
	if err != nil || n != 1 {
		t.Errorf("ALM-01: tabla links ausente (n=%d, err=%v)", n, err)
	}
}

func TestALM02_PersistenciaTrasReabrir(t *testing.T) {
	cases := []struct {
		name string
		l    link.Link
	}{
		{"sin caducidad", link.Link{Code: "sinfin", URL: "https://example.com/a?x=1&y=2", CreatedAt: base}},
		{"con visitas", link.Link{Code: "visto", URL: "https://example.com/v", CreatedAt: base, Visits: 7}},
		{"con caducidad", link.Link{Code: "conFin", URL: "https://example.com/ñandú", CreatedAt: base,
			ExpiresAt: ptr(base.Add(48 * time.Hour))}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			path := filepath.Join(t.TempDir(), "acorta.db")
			s, err := Open(path)
			if err != nil {
				t.Fatalf("Open: %v", err)
			}
			mustInsert(t, s, tc.l)
			if err := s.Close(); err != nil {
				t.Fatalf("Close: %v", err)
			}
			s, err = Open(path)
			if err != nil {
				t.Fatalf("reabrir: %v", err)
			}
			defer s.Close()

			got, err := s.Get(ctx, tc.l.Code)
			if err != nil {
				t.Fatalf("ALM-02: Get: %v", err)
			}
			if got.Code != tc.l.Code || got.URL != tc.l.URL || got.Visits != tc.l.Visits {
				t.Errorf("ALM-02: got %+v, want %+v", got, tc.l)
			}
			if !got.CreatedAt.Equal(tc.l.CreatedAt) {
				t.Errorf("ALM-02: CreatedAt = %v, want %v", got.CreatedAt, tc.l.CreatedAt)
			}
			switch {
			case tc.l.ExpiresAt == nil && got.ExpiresAt != nil:
				t.Errorf("ALM-02: ExpiresAt = %v, want nil", *got.ExpiresAt)
			case tc.l.ExpiresAt != nil && (got.ExpiresAt == nil || !got.ExpiresAt.Equal(*tc.l.ExpiresAt)):
				t.Errorf("ALM-02: ExpiresAt = %v, want %v", got.ExpiresAt, *tc.l.ExpiresAt)
			}
		})
	}
}

func TestALM03_CodigoRepetido(t *testing.T) {
	ctx := context.Background()
	s, _ := openTemp(t)
	orig := link.Link{Code: "promo", URL: "https://example.com/original", CreatedAt: base}
	mustInsert(t, s, orig)

	err := s.Insert(ctx, link.Link{Code: "promo", URL: "https://example.com/otro", CreatedAt: base.Add(time.Hour)})
	if !errors.Is(err, link.ErrCodeTaken) {
		t.Errorf("ALM-03: err = %v, want link.ErrCodeTaken", err)
	}
	got, err := s.Get(ctx, "promo")
	if err != nil {
		t.Fatalf("ALM-03: Get: %v", err)
	}
	if got.URL != orig.URL || !got.CreatedAt.Equal(orig.CreatedAt) {
		t.Errorf("ALM-03: el original cambió: %+v", got)
	}
}

func TestALM04_GetNoExisteYMayusculas(t *testing.T) {
	ctx := context.Background()
	s, _ := openTemp(t)
	mustInsert(t, s, link.Link{Code: "Abc", URL: "https://example.com", CreatedAt: base})

	for _, code := range []string{"zzz", "abc", "ABC", "", "Abc "} {
		if _, err := s.Get(ctx, code); !errors.Is(err, link.ErrNotFound) {
			t.Errorf("ALM-04: Get(%q) err = %v, want link.ErrNotFound", code, err)
		}
	}
	if got, err := s.Get(ctx, "Abc"); err != nil || got.Code != "Abc" {
		t.Errorf("ALM-04: Get(\"Abc\") = %+v, %v", got, err)
	}
}

func TestALM05_ListaDelUltimoAlPrimero(t *testing.T) {
	ctx := context.Background()
	s, _ := openTemp(t)

	got, err := s.List(ctx)
	if err != nil {
		t.Fatalf("ALM-05: List vacía: %v", err)
	}
	if got == nil || len(got) != 0 {
		t.Errorf("ALM-05: lista vacía = %#v, want slice vacío no nil", got)
	}

	// El orden lo da la inserción, no created_at: "a" es el más reciente por
	// fecha pero el primero insertado; "b" y "c" comparten segundo.
	fechas := map[string]time.Time{"a": base.Add(2 * time.Hour), "b": base, "c": base}
	for _, c := range []string{"a", "b", "c"} {
		mustInsert(t, s, link.Link{Code: c, URL: "https://example.com/" + c, CreatedAt: fechas[c]})
	}
	got, err = s.List(ctx)
	if err != nil {
		t.Fatalf("ALM-05: List: %v", err)
	}
	var codes []string
	for _, l := range got {
		codes = append(codes, l.Code)
	}
	if len(codes) != 3 || codes[0] != "c" || codes[1] != "b" || codes[2] != "a" {
		t.Errorf("ALM-05: orden = %v, want [c b a]", codes)
	}
}

func TestALM06_Borrar(t *testing.T) {
	ctx := context.Background()
	s, _ := openTemp(t)
	mustInsert(t, s, link.Link{Code: "x", URL: "https://example.com/1", CreatedAt: base})

	if err := s.Delete(ctx, "x"); err != nil {
		t.Errorf("ALM-06: Delete existente: %v", err)
	}
	if _, err := s.Get(ctx, "x"); !errors.Is(err, link.ErrNotFound) {
		t.Errorf("ALM-06: Get tras borrar err = %v, want link.ErrNotFound", err)
	}
	if err := s.Delete(ctx, "x"); !errors.Is(err, link.ErrNotFound) {
		t.Errorf("ALM-06: Delete inexistente err = %v, want link.ErrNotFound", err)
	}
	if err := s.Insert(ctx, link.Link{Code: "x", URL: "https://example.com/2", CreatedAt: base}); err != nil {
		t.Errorf("ALM-06: reinsertar código borrado: %v", err)
	}
	if got, err := s.Get(ctx, "x"); err != nil || got.URL != "https://example.com/2" {
		t.Errorf("ALM-06: Get tras reinsertar = %+v, %v", got, err)
	}
}

func TestALM07_VisitasConcurrentes(t *testing.T) {
	ctx := context.Background()
	s, _ := openTemp(t)
	mustInsert(t, s, link.Link{Code: "hot", URL: "https://example.com", CreatedAt: base})

	const n = 50
	errs := make(chan error, n)
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			errs <- s.AddVisit(ctx, "hot")
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Errorf("ALM-07: AddVisit: %v", err)
		}
	}
	got, err := s.Get(ctx, "hot")
	if err != nil {
		t.Fatalf("ALM-07: Get: %v", err)
	}
	if got.Visits != n {
		t.Errorf("ALM-07: visits = %d, want %d", got.Visits, n)
	}

	if err := s.AddVisit(ctx, "nope"); !errors.Is(err, link.ErrNotFound) {
		t.Errorf("ALM-07: AddVisit inexistente err = %v, want link.ErrNotFound", err)
	}
}

func TestALM08_InstantesEnUTCYSegundos(t *testing.T) {
	ctx := context.Background()
	madrid := time.FixedZone("CEST", 2*3600)
	// 900 ms de fracción: redondear daría :46 y :59; la spec exige descartar.
	created := time.Date(2026, 6, 1, 12, 30, 45, 900_000_000, madrid)
	expires := time.Date(2026, 12, 31, 23, 59, 58, 900_000_000, madrid)
	wantCreated := time.Date(2026, 6, 1, 10, 30, 45, 0, time.UTC)
	wantExpires := time.Date(2026, 12, 31, 21, 59, 58, 0, time.UTC)

	s, path := openTemp(t)
	mustInsert(t, s, link.Link{Code: "t", URL: "https://example.com", CreatedAt: created, ExpiresAt: &expires})

	got, err := s.Get(ctx, "t")
	if err != nil {
		t.Fatalf("ALM-08: Get: %v", err)
	}
	if got.CreatedAt != wantCreated || got.CreatedAt.Location() != time.UTC {
		t.Errorf("ALM-08: CreatedAt = %v (%v), want %v UTC", got.CreatedAt, got.CreatedAt.Location(), wantCreated)
	}
	if got.ExpiresAt == nil || *got.ExpiresAt != wantExpires || got.ExpiresAt.Location() != time.UTC {
		t.Errorf("ALM-08: ExpiresAt = %v, want %v UTC", got.ExpiresAt, wantExpires)
	}

	// Formato en disco: RFC 3339, UTC, sin fracciones.
	raw, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	defer raw.Close()
	var c, e string
	if err := raw.QueryRow(`SELECT created_at, expires_at FROM links WHERE code = 't'`).Scan(&c, &e); err != nil {
		t.Fatalf("ALM-08: leer en crudo: %v", err)
	}
	if c != "2026-06-01T10:30:45Z" || e != "2026-12-31T21:59:58Z" {
		t.Errorf("ALM-08: en disco created_at=%q expires_at=%q", c, e)
	}
}

func TestALM09_RutaImposible(t *testing.T) {
	// El directorio padre no existe, así que el fichero no se puede crear.
	path := filepath.Join(t.TempDir(), "no-existe", "acorta.db")
	s, err := Open(path)
	if err == nil {
		t.Errorf("ALM-09: Open(%q) no devolvió error", path)
	}
	if s != nil {
		t.Errorf("ALM-09: Open devolvió un Store no nil junto al error; no debe quedar nada abierto")
		s.Close()
	}
	if _, statErr := os.Stat(filepath.Dir(path)); statErr == nil {
		t.Errorf("ALM-09: se ha creado el directorio que no existía")
	}
}
