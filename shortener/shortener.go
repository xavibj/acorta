// Package shortener son los casos de uso de acorta: crear, resolver, listar y
// borrar enlaces. Une el dominio (link) y el almacén (store).
package shortener

import (
	"context"
	"time"

	"acorta/link"
)

// Store es lo que el servicio necesita del almacén (lo implementa store.Store).
type Store interface {
	Insert(ctx context.Context, l link.Link) error // link.ErrCodeTaken si el código existe
	Get(ctx context.Context, code string) (link.Link, error)
	List(ctx context.Context) ([]link.Link, error)
	Delete(ctx context.Context, code string) error
	AddVisit(ctx context.Context, code string) error
}

// Service ejecuta los casos de uso sobre un almacén, un reloj y un generador
// de códigos.
type Service struct {
	store   Store
	now     func() time.Time
	newCode func() (string, error)
}

// New crea el servicio. now y newCode se pueden sustituir en los tests;
// con nil usa time.Now y link.NewCode sobre crypto/rand.
func New(s Store, now func() time.Time, newCode func() (string, error)) *Service {
	// TODO(paso verde): valores por defecto para now y newCode.
	return &Service{store: s, now: now, newCode: newCode}
}

// Create valida la entrada y guarda el enlace.
func (s *Service) Create(ctx context.Context, in link.Input) (link.Link, error) {
	return link.Link{}, nil // TODO(paso verde)
}

// Resolve devuelve la URL de destino de un código; si countVisit, suma una visita.
func (s *Service) Resolve(ctx context.Context, code string, countVisit bool) (string, error) {
	return "", nil // TODO(paso verde)
}

// List devuelve todos los enlaces, del más reciente al más antiguo.
func (s *Service) List(ctx context.Context) ([]link.Link, error) {
	return nil, nil // TODO(paso verde)
}

// Delete borra un enlace por su código.
func (s *Service) Delete(ctx context.Context, code string) error {
	return nil // TODO(paso verde)
}

// Now devuelve el instante actual según el reloj del servicio (para LIS-02).
func (s *Service) Now() time.Time {
	return time.Time{} // TODO(paso verde)
}
