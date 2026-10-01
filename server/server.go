// Package server expone acorta por HTTP: la redirección de las URL cortas,
// la API JSON bajo /api/ y la interfaz web embebida.
//
// Es un adaptador fino: traduce HTTP a llamadas de shortener y los errores
// de link/shortener a estados HTTP. No contiene reglas de negocio.
package server

import (
	"context"
	"embed"
	"io/fs"
	"net/http"
	"strings"
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

// "all:" para que entren también ficheros que empiecen por _ o ., que el
// build del frontend puede generar.
//
//go:embed all:static
var embedded embed.FS

// Static devuelve los ficheros embebidos en el binario (server/static/).
func Static() fs.FS {
	sub, err := fs.Sub(embedded, "static")
	if err != nil {
		// Solo falla con una ruta inválida, y la ruta es una constante.
		panic(err)
	}
	return sub
}

// server agrupa lo que comparten los manejadores.
type server struct {
	svc     Service
	baseURL string // sin barra final
	static  fs.FS
}

// Handler devuelve el servidor completo. baseURL se usa para short_url;
// static es el sistema de ficheros con sin-compilar.html y, si existe, dist/.
func Handler(svc Service, baseURL string, static fs.FS) http.Handler {
	return &server{svc: svc, baseURL: strings.TrimRight(baseURL, "/"), static: static}
}

// ServeHTTP reparte a mano en vez de usar http.ServeMux: el mux responde
// 404 y 405 en texto plano y sin el Allow que pide la spec, y aquí la API
// necesita contestar en JSON. El reparto es corto y se entiende de un vistazo:
//
//	/api/…              → API (JSON)
//	/                   → interfaz web
//	/assets/…           → estático, a cualquier profundidad
//	/{un segmento}.ext  → estático (un código nunca lleva punto)
//	/{un segmento}      → redirección
//	cualquier otra      → 404
func (s *server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	switch {
	case path == "/api" || strings.HasPrefix(path, "/api/"):
		s.serveAPI(w, r)
	case path == "/":
		s.serveIndex(w, r)
	case strings.HasPrefix(path, "/assets/"):
		s.serveFile(w, r, strings.TrimPrefix(path, "/"))
	case strings.Contains(path[1:], "/"):
		writePlain(w, r, http.StatusNotFound, "ruta no encontrada")
	case strings.Contains(path, "."):
		s.serveFile(w, r, strings.TrimPrefix(path, "/"))
	default:
		s.serveRedirect(w, r, strings.TrimPrefix(path, "/"))
	}
}

// allowReadOnly responde 405 si el método no es GET ni HEAD. Devuelve true
// si la petición ya está contestada.
func allowReadOnly(w http.ResponseWriter, r *http.Request) bool {
	if r.Method == http.MethodGet || r.Method == http.MethodHead {
		return false
	}
	w.Header().Set("Allow", "GET, HEAD")
	writePlain(w, r, http.StatusMethodNotAllowed, "método no permitido")
	return true
}
