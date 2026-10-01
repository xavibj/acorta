// Package link es el dominio puro de acorta: el tipo Link, la validación de
// la entrada y la generación de códigos. No usa base de datos, red, reloj
// global ni aleatoriedad global.
package link

import (
	"errors"
	"fmt"
	"io"
	"net/url"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	maxURLBytes = 2048
	minAlias    = 3
	maxAlias    = 32

	codeLen      = 6
	codeAlphabet = "abcdefghijklmnopqrstuvwxyz0123456789"
	// Los bytes de 252 en adelante se descartan: 252 es el mayor múltiplo de
	// 36 que cabe en un byte, y así los 36 caracteres son igual de probables.
	codeByteLimit = 252
)

// reserved son las rutas que el servidor necesita para sí.
var reserved = map[string]bool{"api": true, "assets": true}

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
	return l.ExpiresAt != nil && !now.Before(*l.ExpiresAt)
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
	var msgs []string

	rawURL := strings.TrimSpace(in.URL)
	if msg := checkURL(rawURL); msg != "" {
		msgs = append(msgs, msg)
	}

	alias := strings.TrimSpace(in.Alias)
	if alias != "" {
		if msg := checkAlias(alias); msg != "" {
			msgs = append(msgs, msg)
		}
	}

	expires, msg := parseExpiresAt(strings.TrimSpace(in.ExpiresAt), now)
	if msg != "" {
		msgs = append(msgs, msg)
	}

	if len(msgs) > 0 {
		return Valid{}, msgs
	}
	return Valid{URL: rawURL, Alias: alias, ExpiresAt: expires}, nil
}

// checkURL devuelve el primer error de la URL (ya recortada), o "".
func checkURL(raw string) string {
	if raw == "" {
		return "url: es obligatoria"
	}
	// Bytes, no caracteres: es el límite que importa para guardarla.
	if len(raw) > maxURLBytes {
		return "url: no puede superar los 2048 bytes"
	}
	u, err := url.Parse(raw)
	if err != nil {
		return "url: no es una URL válida"
	}
	// url.Parse ya pasa el esquema a minúsculas (URL-07).
	if u.Scheme != "http" && u.Scheme != "https" {
		return "url: debe empezar por http:// o https://"
	}
	if u.Hostname() == "" {
		return "url: falta el dominio"
	}
	return ""
}

// checkAlias devuelve el primer error del alias (ya recortado y no vacío), o "".
func checkAlias(alias string) string {
	// Caracteres, no bytes: "ñu" son 2.
	if n := utf8.RuneCountInString(alias); n < minAlias || n > maxAlias {
		return "alias: debe tener entre 3 y 32 caracteres"
	}
	for _, r := range alias {
		if !(r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r == '-') {
			return "alias: solo admite minúsculas, números y guiones"
		}
	}
	if strings.HasPrefix(alias, "-") || strings.HasSuffix(alias, "-") {
		return "alias: no puede empezar ni terminar por guion"
	}
	if Reserved(alias) {
		return fmt.Sprintf("alias: %q está reservado", alias)
	}
	return ""
}

// parseExpiresAt devuelve la caducidad en UTC y sin fracciones, nil si no se
// envió, o el mensaje de error.
func parseExpiresAt(raw string, now time.Time) (*time.Time, string) {
	if raw == "" {
		return nil, ""
	}
	t, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return nil, "expires_at: debe tener formato RFC 3339 (2026-12-31T23:59:59Z)"
	}
	// Se comparan los dos instantes ya sin fracciones de segundo (CAD-03).
	t = t.UTC().Truncate(time.Second)
	if !t.After(now.UTC().Truncate(time.Second)) {
		return nil, "expires_at: debe ser una fecha futura"
	}
	return &t, ""
}

// NewCode lee de r los bytes que necesite y devuelve un código (COD-01).
func NewCode(r io.Reader) (string, error) {
	code := make([]byte, 0, codeLen)
	var b [1]byte
	for len(code) < codeLen {
		if _, err := io.ReadFull(r, b[:]); err != nil {
			return "", fmt.Errorf("leyendo la fuente de azar: %w", err)
		}
		if b[0] >= codeByteLimit {
			continue
		}
		code = append(code, codeAlphabet[int(b[0])%len(codeAlphabet)])
	}
	return string(code), nil
}

// Reserved indica si code es una palabra reservada.
func Reserved(code string) bool {
	return reserved[code]
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

func (e *ValidationError) Error() string { return strings.Join(e.Messages, "; ") }

// AliasTakenError es el ALI-06. Su mensaje es `alias: "promo" ya está en uso`.
type AliasTakenError struct{ Alias string }

func (e *AliasTakenError) Error() string {
	return fmt.Sprintf("alias: %q ya está en uso", e.Alias)
}
