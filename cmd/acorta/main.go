// Command acorta es el ejecutable: servidor y CLI sobre la misma base de datos.
package main

import (
	"io"
	"net/http"
	"os"
	"time"
)

// version se fija al compilar con -ldflags "-X main.version=…".
var version = "dev"

// environment agrupa todo lo que run toma del exterior, para sustituirlo en
// los tests sin procesos ni sockets.
type environment struct {
	getenv  func(string) string                     // lee variables de entorno
	now     func() time.Time                        // reloj del servicio
	newCode func() (string, error)                  // generador de códigos; nil: el real
	listen  func(addr string, h http.Handler) error // escucha y sirve; bloquea
}

// env es el entorno real; los tests lo reemplazan.
var env = environment{
	getenv:  os.Getenv,
	now:     time.Now,
	newCode: nil,
	listen:  http.ListenAndServe,
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

// run ejecuta la CLI y devuelve el código de salida (spec 005).
func run(args []string, stdout, stderr io.Writer) int {
	// Esqueleto del paso rojo: todavía no implementado.
	return 99
}
