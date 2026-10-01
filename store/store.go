// Package store guarda los enlaces en un fichero SQLite. No valida reglas de
// negocio: solo guarda y recupera (spec 003).
package store

import (
	"context"
	"database/sql"

	"acorta/link"
)

// Store es el almacén de enlaces sobre SQLite.
type Store struct{ db *sql.DB }

// Open abre (o crea) la base de datos y su esquema.
func Open(path string) (*Store, error) { return &Store{}, nil }

// Close cierra la base de datos.
func (s *Store) Close() error { return nil }

// Insert guarda un enlace nuevo; ErrCodeTaken si el código ya existe.
func (s *Store) Insert(ctx context.Context, l link.Link) error { return nil }

// Get devuelve el enlace con ese código; ErrNotFound si no existe.
func (s *Store) Get(ctx context.Context, code string) (link.Link, error) {
	return link.Link{}, nil
}

// List devuelve todos los enlaces, del último insertado al primero.
func (s *Store) List(ctx context.Context) ([]link.Link, error) { return nil, nil }

// Delete elimina un enlace; ErrNotFound si no existe.
func (s *Store) Delete(ctx context.Context, code string) error { return nil }

// AddVisit suma 1 al contador de visitas; ErrNotFound si no existe.
func (s *Store) AddVisit(ctx context.Context, code string) error { return nil }
