// Package store guarda los enlaces en un fichero SQLite. No valida reglas de
// negocio: solo guarda y recupera (spec 003).
package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"time"

	"acorta/link"

	"modernc.org/sqlite"
)

// sqliteConstraintUnique es el código extendido SQLITE_CONSTRAINT_UNIQUE
// (2067). Se declara aquí para no importar el subpaquete de constantes.
const sqliteConstraintUnique = 2067

const schema = `CREATE TABLE IF NOT EXISTS links (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    code       TEXT    NOT NULL UNIQUE,
    url        TEXT    NOT NULL,
    created_at TEXT    NOT NULL,
    expires_at TEXT,
    visits     INTEGER NOT NULL DEFAULT 0
)`

// Store es el almacén de enlaces sobre SQLite.
type Store struct{ db *sql.DB }

// Open abre (o crea) la base de datos y su esquema.
func Open(path string) (*Store, error) {
	// Los PRAGMA van en la DSN y no en un Exec: así se aplican a cada
	// conexión nueva del pool, no solo a la primera.
	dsn := (&url.URL{
		Scheme: "file",
		Opaque: url.PathEscape(path),
		RawQuery: url.Values{"_pragma": {
			"journal_mode(WAL)",
			"busy_timeout(5000)",
		}}.Encode(),
	}).String()

	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("abrir %s: %w", path, err)
	}
	// Una sola conexión serializa las escrituras dentro del proceso y evita
	// "database is locked" entre goroutines; busy_timeout cubre a otros procesos.
	db.SetMaxOpenConns(1)

	// sql.Open es perezoso: Ping fuerza la apertura real y falla si la ruta
	// no se puede crear (ALM-09).
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("abrir %s: %w", path, err)
	}
	if _, err := db.Exec(schema); err != nil {
		db.Close()
		return nil, fmt.Errorf("crear esquema: %w", err)
	}
	return &Store{db: db}, nil
}

// Close cierra la base de datos.
func (s *Store) Close() error { return s.db.Close() }

// Insert guarda un enlace nuevo, tal como llega (también su Visits);
// link.ErrCodeTaken si el código ya existe.
func (s *Store) Insert(ctx context.Context, l link.Link) error {
	var expires any // nil se guarda como NULL: el enlace no caduca
	if l.ExpiresAt != nil {
		expires = formatTime(*l.ExpiresAt)
	}
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO links (code, url, created_at, expires_at, visits) VALUES (?, ?, ?, ?, ?)`,
		l.Code, l.URL, formatTime(l.CreatedAt), expires, l.Visits)
	if err != nil {
		// Se deja que la restricción UNIQUE decida: un SELECT previo no
		// evitaría la carrera con otra inserción concurrente.
		var serr *sqlite.Error
		if errors.As(err, &serr) && serr.Code() == sqliteConstraintUnique {
			return fmt.Errorf("insertar %q: %w", l.Code, link.ErrCodeTaken)
		}
		return fmt.Errorf("insertar %q: %w", l.Code, err)
	}
	return nil
}

// Get devuelve el enlace con ese código; link.ErrNotFound si no existe.
func (s *Store) Get(ctx context.Context, code string) (link.Link, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT code, url, created_at, expires_at, visits FROM links WHERE code = ?`, code)
	l, err := scanLink(row)
	if errors.Is(err, sql.ErrNoRows) {
		return link.Link{}, fmt.Errorf("obtener %q: %w", code, link.ErrNotFound)
	}
	if err != nil {
		return link.Link{}, fmt.Errorf("obtener %q: %w", code, err)
	}
	return l, nil
}

// List devuelve todos los enlaces, del último insertado al primero.
func (s *Store) List(ctx context.Context) ([]link.Link, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT code, url, created_at, expires_at, visits FROM links ORDER BY id DESC`)
	if err != nil {
		return nil, fmt.Errorf("listar: %w", err)
	}
	defer rows.Close()

	links := []link.Link{} // vacía, nunca nil
	for rows.Next() {
		l, err := scanLink(rows)
		if err != nil {
			return nil, fmt.Errorf("listar: %w", err)
		}
		links = append(links, l)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("listar: %w", err)
	}
	return links, nil
}

// Delete elimina un enlace; link.ErrNotFound si no existe.
func (s *Store) Delete(ctx context.Context, code string) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM links WHERE code = ?`, code)
	return affectedOne(res, err, "borrar", code)
}

// AddVisit suma 1 al contador en una sola sentencia, para que las visitas
// simultáneas no se pisen; link.ErrNotFound si no existe.
func (s *Store) AddVisit(ctx context.Context, code string) error {
	res, err := s.db.ExecContext(ctx, `UPDATE links SET visits = visits + 1 WHERE code = ?`, code)
	return affectedOne(res, err, "sumar visita a", code)
}

// affectedOne traduce el resultado de un UPDATE/DELETE por código: ninguna
// fila afectada significa que el código no existe.
func affectedOne(res sql.Result, err error, op, code string) error {
	if err != nil {
		return fmt.Errorf("%s %q: %w", op, code, err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("%s %q: %w", op, code, err)
	}
	if n == 0 {
		return fmt.Errorf("%s %q: %w", op, code, link.ErrNotFound)
	}
	return nil
}

// formatTime da el formato de la base de datos: RFC 3339, UTC y segundos.
// Truncate descarta las fracciones (no las redondea), como pide ALM-08.
func formatTime(t time.Time) string {
	return t.UTC().Truncate(time.Second).Format(time.RFC3339)
}

// scanner lo cumplen *sql.Row y *sql.Rows.
type scanner interface{ Scan(dest ...any) error }

func scanLink(sc scanner) (link.Link, error) {
	var (
		l       link.Link
		created string
		expires sql.NullString
	)
	if err := sc.Scan(&l.Code, &l.URL, &created, &expires, &l.Visits); err != nil {
		return link.Link{}, err
	}
	t, err := time.Parse(time.RFC3339, created)
	if err != nil {
		return link.Link{}, fmt.Errorf("created_at inválido: %w", err)
	}
	l.CreatedAt = t.UTC()
	if expires.Valid {
		e, err := time.Parse(time.RFC3339, expires.String)
		if err != nil {
			return link.Link{}, fmt.Errorf("expires_at inválido: %w", err)
		}
		e = e.UTC()
		l.ExpiresAt = &e
	}
	return l, nil
}
