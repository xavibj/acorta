package server

import (
	"bytes"
	"io/fs"
	"net/http"
	"time"
)

// serveIndex sirve la interfaz web, o la página de aviso si el frontend no
// se ha compilado (EST-04).
func (s *server) serveIndex(w http.ResponseWriter, r *http.Request) {
	if allowReadOnly(w, r) {
		return
	}
	if data, err := fs.ReadFile(s.static, "dist/index.html"); err == nil {
		serveBytes(w, r, "index.html", data)
		return
	}
	data, err := fs.ReadFile(s.static, "sin-compilar.html")
	if err != nil {
		writeInternal(w, r, err, false)
		return
	}
	serveBytes(w, r, "sin-compilar.html", data)
}

// serveFile sirve un fichero de dist/. name llega sin barra inicial
// (por ejemplo "assets/app.js"). Cualquier cosa que no sea un fichero
// regular (no existe, es una carpeta, ruta inválida) es un 404.
func (s *server) serveFile(w http.ResponseWriter, r *http.Request, name string) {
	if allowReadOnly(w, r) {
		return
	}
	full := "dist/" + name
	if !fs.ValidPath(full) {
		writePlain(w, r, http.StatusNotFound, "fichero no encontrado")
		return
	}
	info, err := fs.Stat(s.static, full)
	if err != nil || !info.Mode().IsRegular() {
		writePlain(w, r, http.StatusNotFound, "fichero no encontrado")
		return
	}
	data, err := fs.ReadFile(s.static, full)
	if err != nil {
		writeInternal(w, r, err, false)
		return
	}
	serveBytes(w, r, name, data)
}

// serveBytes usa http.ServeContent, que deduce el tipo de contenido por la
// extensión y ya sabe responder a HEAD sin cuerpo.
func serveBytes(w http.ResponseWriter, r *http.Request, name string, data []byte) {
	http.ServeContent(w, r, name, time.Time{}, bytes.NewReader(data))
}
