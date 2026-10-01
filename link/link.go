// Package link es el dominio puro de acorta: el tipo Link, la validación de
// la entrada y la generación de códigos. No usa base de datos, red, reloj
// global ni aleatoriedad global.
package link

import (
	"errors"
	"io"
	"time"
)

// Link es un enlace acortado.
type Link struct {
	Code      string
	URL       string
	CreatedAt time.Time
	ExpiresAt *time.Time // nil: no caduca
	Visits    int64
}

// Expired indica si el enlace ha caducado en el instante now (CAD-05).
func (l Link) Expired(now time.Time) bool {
	return false
}

// Input es la petición de crear, tal como llega: todo texto.
type Input struct {
	URL       string
	Alias     string
	ExpiresAt string
}

// Valid es una entrada ya validada y limpia.
type Valid struct {
	URL       string
	Alias     string     // "" si hay que generar el código
	ExpiresAt *time.Time // UTC, sin fracciones de segundo
}

// Validate aplica URL-*, ALI-01 a ALI-05 y CAD-01 a CAD-04.
// Si hay errores devuelve sus mensajes en el orden url, alias, expires_at
// y un Valid vacío (el valor cero); si no los hay, la lista es nil.
func Validate(in Input, now time.Time) (Valid, []string) {
	return Valid{}, nil
}

// NewCode lee de r los bytes que necesite y devuelve un código (COD-01).
func NewCode(r io.Reader) (string, error) {
	return "", nil
}

// Reserved indica si code es una palabra reservada.
func Reserved(code string) bool {
	return false
}

var (
	ErrNotFound        = errors.New("enlace no encontrado")
	ErrExpired         = errors.New("enlace caducado")
	ErrCodeTaken       = errors.New("código en uso")
	ErrNoCodeAvailable = errors.New("no se ha podido generar un código libre")
)

// ValidationError agrupa los mensajes de Validate. Su Error() los une
// con "; ": `url: es obligatoria; alias: "api" está reservado`.
type ValidationError struct{ Messages []string }

func (e *ValidationError) Error() string { return "" }

// AliasTakenError es el ALI-06. Su mensaje es `alias: "promo" ya está en uso`.
type AliasTakenError struct{ Alias string }

func (e *AliasTakenError) Error() string { return "" }
