// Package server expone acorta por HTTP: la redirección de las URL cortas,
// la API JSON bajo /api/ y la interfaz web embebida.
package server

import (
	"context"
	"embed"
	"io/fs"
	"net/http"
	"time"

	"acorta/link"
)

// Service es lo que el servidor necesita de shortener.Service.
type Service interface {
	Create(ctx context.Context, in link.Input) (link.Link, error)
	Resolve(ctx context.Context, code string, countVisit bool) (string, error)
	List(ctx context.Context) ([]link.Link, error)
	Delete(ctx context.Context, code string) error
	Now() time.Time
}

//go:embed all:static
var embedded embed.FS

// Handler devuelve el servidor completo. baseURL se usa para short_url;
// static es el sistema de ficheros con sin-compilar.html y, si existe, dist/.
func Handler(svc Service, baseURL string, static fs.FS) http.Handler {
	// Esqueleto del paso rojo: todavía no implementado.
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "no implementado", http.StatusNotImplemented)
	})
}

// Static devuelve los ficheros embebidos en el binario (server/static/).
func Static() fs.FS {
	// Esqueleto del paso rojo: todavía no implementado.
	return nil
}
