// Package shortener son los casos de uso de acorta: crear, resolver, listar y
// borrar enlaces. Une el dominio (link) y el almacén (store).
package shortener

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
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

// maxCodeAttempts son los intentos en total para dar con un código generado
// que no esté ocupado ni reservado (COD-02).
const maxCodeAttempts = 5

// New crea el servicio. now y newCode se pueden sustituir en los tests;
// con nil usa time.Now y link.NewCode sobre crypto/rand.
func New(s Store, now func() time.Time, newCode func() (string, error)) *Service {
	if now == nil {
		now = time.Now
	}
	if newCode == nil {
		newCode = func() (string, error) { return link.NewCode(rand.Reader) }
	}
	return &Service{store: s, now: now, newCode: newCode}
}

// Now devuelve el instante actual según el reloj del servicio (para LIS-02).
func (s *Service) Now() time.Time { return s.now() }

// Create valida la entrada y guarda el enlace. Devuelve *link.ValidationError,
// *link.AliasTakenError o link.ErrNoCodeAvailable según el caso; cualquier
// otro error es un fallo del generador o del almacén.
func (s *Service) Create(ctx context.Context, in link.Input) (link.Link, error) {
	// El reloj se lee una sola vez para que validación y CreatedAt coincidan.
	// Truncar a segundos y UTC es la precisión que usa todo el sistema.
	now := s.now().UTC().Truncate(time.Second)

	valid, msgs := link.Validate(in, now)
	if len(msgs) > 0 {
		return link.Link{}, &link.ValidationError{Messages: msgs}
	}

	l := link.Link{URL: valid.URL, CreatedAt: now, ExpiresAt: valid.ExpiresAt}

	if valid.Alias != "" {
		l.Code = valid.Alias
		// No se hace un Get previo: la restricción UNIQUE del almacén decide
		// y así no hay carrera entre dos creaciones simultáneas.
		if err := s.store.Insert(ctx, l); err != nil {
			if errors.Is(err, link.ErrCodeTaken) {
				return link.Link{}, &link.AliasTakenError{Alias: valid.Alias}
			}
			return link.Link{}, fmt.Errorf("guardar el enlace: %w", err)
		}
		return l, nil
	}

	for i := 0; i < maxCodeAttempts; i++ {
		code, err := s.newCode()
		if err != nil {
			return link.Link{}, fmt.Errorf("generar el código: %w", err)
		}
		if link.Reserved(code) {
			continue
		}
		l.Code = code
		err = s.store.Insert(ctx, l)
		if err == nil {
			return l, nil
		}
		if !errors.Is(err, link.ErrCodeTaken) {
			return link.Link{}, fmt.Errorf("guardar el enlace: %w", err)
		}
		// Código ocupado: se descarta y se prueba con otro.
	}
	return link.Link{}, link.ErrNoCodeAvailable
}

// Resolve devuelve la URL de destino de un código; si countVisit, suma una
// visita (VIS-02). Los fallos no suman nada (VIS-03).
func (s *Service) Resolve(ctx context.Context, code string, countVisit bool) (string, error) {
	l, err := s.store.Get(ctx, code)
	if err != nil {
		return "", err
	}
	if l.Expired(s.now()) {
		return "", fmt.Errorf("resolver %q: %w", code, link.ErrExpired)
	}
	if countVisit {
		// Si no se puede contar no se devuelve la URL (RES-05); si el enlace
		// se borró entre medias, el almacén ya da ErrNotFound.
		if err := s.store.AddVisit(ctx, code); err != nil {
			return "", err
		}
	}
	return l.URL, nil
}

// List devuelve todos los enlaces, del más reciente al más antiguo. Si están
// caducados lo calcula quien los muestra con l.Expired(s.Now()) (LIS-02).
func (s *Service) List(ctx context.Context) ([]link.Link, error) {
	return s.store.List(ctx)
}

// Delete borra un enlace por su código; su código queda libre (BOR-03).
func (s *Service) Delete(ctx context.Context, code string) error {
	return s.store.Delete(ctx, code)
}
