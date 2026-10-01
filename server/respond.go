package server

import (
	"encoding/json"
	"log"
	"net/http"
)

// writePlain responde en texto plano; el cuerpo termina en salto de línea.
// En HEAD no se escribe cuerpo (el servidor real lo descartaría, pero así
// el comportamiento es el mismo con cualquier ResponseWriter).
func writePlain(w http.ResponseWriter, r *http.Request, status int, text string) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(status)
	if r.Method != http.MethodHead {
		w.Write([]byte(text + "\n"))
	}
}

// writeJSON responde con v en JSON y un salto de línea final.
func writeJSON(w http.ResponseWriter, r *http.Request, status int, v any) {
	body, err := json.Marshal(v)
	if err != nil {
		// No debería pasar con los tipos que usamos.
		log.Printf("server: serializar la respuesta: %v", err)
		status, body = http.StatusInternalServerError, []byte(`{"errors":["error interno"]}`)
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if r.Method != http.MethodHead {
		w.Write(append(body, '\n'))
	}
}

// writeErrors responde con la forma {"errors": [...]}.
func writeErrors(w http.ResponseWriter, r *http.Request, status int, msgs ...string) {
	writeJSON(w, r, status, struct {
		Errors []string `json:"errors"`
	}{msgs})
}

// writeInternal registra el detalle en el log y responde sin él.
func writeInternal(w http.ResponseWriter, r *http.Request, err error, asJSON bool) {
	log.Printf("server: %s %s: %v", r.Method, r.URL.Path, err)
	if asJSON {
		writeErrors(w, r, http.StatusInternalServerError, "error interno")
		return
	}
	writePlain(w, r, http.StatusInternalServerError, "error interno")
}
