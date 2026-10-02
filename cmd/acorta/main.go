// Command acorta es el ejecutable: servidor y CLI sobre la misma base de datos.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"text/tabwriter"
	"time"
	"unicode/utf8"

	"acorta/link"
	"acorta/server"
	"acorta/shortener"
	"acorta/store"
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

// Sinopsis de cada subcomando, tal como las da la spec 005.
const (
	synopsisServe = "acorta serve [-addr :8080] [-db RUTA] [-base-url URL] [-token TOKEN]"
	synopsisAdd   = "acorta add [-alias ALIAS] [-ttl DURACIÓN] [-db RUTA] [-base-url URL] URL"
	synopsisList  = "acorta list [-db RUTA]"
	synopsisRm    = "acorta rm [-db RUTA] CÓDIGO"
)

const (
	defaultDB      = "acorta.db"
	defaultBaseURL = "http://localhost:8080"
	defaultAddr    = ":8080"
	// minTokenLen es la longitud mínima del token de API (spec 008).
	minTokenLen = 16
)

// helpText es la ayuda general.
const helpText = `uso: acorta <subcomando> [flags]

Subcomandos:
  ` + synopsisServe + `
  ` + synopsisAdd + `
  ` + synopsisList + `
  ` + synopsisRm + `
  acorta version
  acorta help
`

// run ejecuta la CLI y devuelve el código de salida (spec 005).
func run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprint(stdout, helpText)
		return 0
	}
	rest := args[1:]
	switch args[0] {
	case "help", "-h", "-help", "--help":
		fmt.Fprint(stdout, helpText)
		return 0
	case "version":
		fmt.Fprintln(stdout, "acorta", version)
		return 0
	case "serve":
		return cmdServe(rest, stdout, stderr)
	case "add":
		return cmdAdd(rest, stdout, stderr)
	case "list":
		return cmdList(rest, stdout, stderr)
	case "rm":
		return cmdRm(rest, stdout, stderr)
	default:
		fmt.Fprintf(stderr, "error: subcomando desconocido %q\n%s", args[0], helpText)
		return 2
	}
}

// options son los flags comunes a los subcomandos.
type options struct {
	db, baseURL, addr, alias, ttl, token string
}

// parse analiza los flags de un subcomando. Devuelve (código, true) si hay
// que terminar ya: 0 tras -h (uso y flags en stdout) o 2 si el uso es
// incorrecto (flag desconocido o argumentos que no son nargs).
func parse(name, synopsis string, fs *flag.FlagSet, args []string, nargs int, stdout, stderr io.Writer) (code int, done bool) {
	// flag no escribe nada por su cuenta: los mensajes los damos nosotros.
	fs.SetOutput(io.Discard)
	fs.Usage = func() {}
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			fmt.Fprintf(stdout, "uso: %s\n", synopsis)
			fs.SetOutput(stdout)
			fs.PrintDefaults()
			return 0, true
		}
		fmt.Fprintf(stderr, "error: %v\nuso: %s\n", err, synopsis)
		return 2, true
	}
	if fs.NArg() != nargs {
		fmt.Fprintf(stderr, "uso: %s\n", synopsis)
		return 2, true
	}
	return 0, false
}

// pick aplica la precedencia flag > entorno > por defecto. Un valor vacío
// cuenta como no definido.
func pick(flagVal, envName, def string) string {
	if flagVal != "" {
		return flagVal
	}
	if v := env.getenv(envName); v != "" {
		return v
	}
	return def
}

// open abre el almacén y arma el servicio. En caso de error ya lo ha
// escrito y devuelve ok=false.
func open(db string, stderr io.Writer) (*store.Store, *shortener.Service, bool) {
	st, err := store.Open(db)
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return nil, nil, false
	}
	return st, shortener.New(st, env.now, env.newCode), true
}

func fail(stderr io.Writer, err error) int {
	// El almacén envuelve el error con la operación ("borrar \"x\": ...");
	// al usuario le basta el mensaje del centinela.
	if errors.Is(err, link.ErrNotFound) {
		err = link.ErrNotFound
	}
	fmt.Fprintf(stderr, "error: %v\n", err)
	return 1
}

func cmdAdd(args []string, stdout, stderr io.Writer) int {
	var o options
	fs := flag.NewFlagSet("add", flag.ContinueOnError)
	fs.StringVar(&o.alias, "alias", "", "código personalizado en lugar de uno generado")
	fs.StringVar(&o.ttl, "ttl", "", "vida del enlace como duración de Go (90m, 72h); sin valor no caduca")
	fs.StringVar(&o.db, "db", "", "fichero de la base de datos (ACORTA_DB; por defecto "+defaultDB+")")
	fs.StringVar(&o.baseURL, "base-url", "", "URL base de las URL cortas (ACORTA_BASE_URL; por defecto "+defaultBaseURL+")")
	if code, done := parse("add", synopsisAdd, fs, args, 1, stdout, stderr); done {
		return code
	}

	// Se mira si -ttl se pasó, porque "" explícito también es un error.
	ttlSet := false
	fs.Visit(func(f *flag.Flag) { ttlSet = ttlSet || f.Name == "ttl" })
	var d time.Duration
	if ttlSet {
		var err error
		d, err = time.ParseDuration(o.ttl)
		if err != nil || d <= 0 {
			fmt.Fprintln(stderr, "error: ttl: debe ser una duración positiva (ejemplos: 90m, 72h)")
			return 2
		}
	}

	st, svc, ok := open(pick(o.db, "ACORTA_DB", defaultDB), stderr)
	if !ok {
		return 1
	}
	defer st.Close()

	in := link.Input{URL: fs.Arg(0), Alias: o.alias}
	if ttlSet {
		// La caducidad se calcula con el reloj del servicio, que es el que
		// validará que sea futura.
		in.ExpiresAt = svc.Now().Add(d).UTC().Format(time.RFC3339)
	}
	l, err := svc.Create(context.Background(), in)
	if err != nil {
		var ve *link.ValidationError
		if errors.As(err, &ve) {
			for _, m := range ve.Messages {
				fmt.Fprintf(stderr, "error: %s\n", m)
			}
			return 1
		}
		return fail(stderr, err)
	}
	base := strings.TrimRight(pick(o.baseURL, "ACORTA_BASE_URL", defaultBaseURL), "/")
	fmt.Fprintf(stdout, "%s/%s\n", base, l.Code)
	return 0
}

func cmdList(args []string, stdout, stderr io.Writer) int {
	var o options
	fs := flag.NewFlagSet("list", flag.ContinueOnError)
	fs.StringVar(&o.db, "db", "", "fichero de la base de datos (ACORTA_DB; por defecto "+defaultDB+")")
	if code, done := parse("list", synopsisList, fs, args, 0, stdout, stderr); done {
		return code
	}
	st, svc, ok := open(pick(o.db, "ACORTA_DB", defaultDB), stderr)
	if !ok {
		return 1
	}
	defer st.Close()

	links, err := svc.List(context.Background())
	if err != nil {
		return fail(stderr, err)
	}
	if len(links) == 0 {
		fmt.Fprintln(stdout, "No hay enlaces.")
		return 0
	}
	// Se lee el reloj una vez para que todas las filas usen el mismo instante.
	now := svc.Now()
	tw := tabwriter.NewWriter(stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "CÓDIGO\tVISITAS\tCADUCA\tURL")
	for _, l := range links {
		expires := "nunca"
		if l.Expired(now) {
			expires = "caducado"
		} else if l.ExpiresAt != nil {
			expires = l.ExpiresAt.UTC().Format(time.RFC3339)
		}
		fmt.Fprintf(tw, "%s\t%d\t%s\t%s\n", l.Code, l.Visits, expires, l.URL)
	}
	tw.Flush()
	return 0
}

func cmdRm(args []string, stdout, stderr io.Writer) int {
	var o options
	fs := flag.NewFlagSet("rm", flag.ContinueOnError)
	fs.StringVar(&o.db, "db", "", "fichero de la base de datos (ACORTA_DB; por defecto "+defaultDB+")")
	if code, done := parse("rm", synopsisRm, fs, args, 1, stdout, stderr); done {
		return code
	}
	st, svc, ok := open(pick(o.db, "ACORTA_DB", defaultDB), stderr)
	if !ok {
		return 1
	}
	defer st.Close()

	if err := svc.Delete(context.Background(), fs.Arg(0)); err != nil {
		return fail(stderr, err)
	}
	fmt.Fprintf(stdout, "Borrado: %s\n", fs.Arg(0))
	return 0
}

func cmdServe(args []string, stdout, stderr io.Writer) int {
	var o options
	fs := flag.NewFlagSet("serve", flag.ContinueOnError)
	fs.StringVar(&o.addr, "addr", defaultAddr, "dirección de escucha")
	fs.StringVar(&o.db, "db", "", "fichero de la base de datos (ACORTA_DB; por defecto "+defaultDB+")")
	fs.StringVar(&o.baseURL, "base-url", "", "URL base de las URL cortas (ACORTA_BASE_URL; por defecto "+defaultBaseURL+")")
	fs.StringVar(&o.token, "token", "", fmt.Sprintf("token de API, al menos %d caracteres (ACORTA_TOKEN; obligatorio)", minTokenLen))
	if code, done := parse("serve", synopsisServe, fs, args, 0, stdout, stderr); done {
		return code
	}
	// El token se comprueba antes de abrir la base de datos (spec 008).
	token, ok := apiToken(o.token, stderr)
	if !ok {
		return 1
	}
	st, svc, ok := open(pick(o.db, "ACORTA_DB", defaultDB), stderr)
	if !ok {
		return 1
	}
	defer st.Close()

	base := strings.TrimRight(pick(o.baseURL, "ACORTA_BASE_URL", defaultBaseURL), "/")
	h := server.Handler(svc, base, server.Static(), token)
	fmt.Fprintf(stdout, "acorta escuchando en %s\n", listenURL(o.addr))
	if err := env.listen(o.addr, h); err != nil {
		return fail(stderr, err)
	}
	return 0
}

// apiToken resuelve el token de API de serve (flag > ACORTA_TOKEN), recortado.
// Vacío tras recortar cuenta como no definido (AUT-08); de 1 a 15
// caracteres es demasiado corto (AUT-09). En caso de error ya lo ha escrito
// y devuelve ok=false. El token nunca se escribe en ninguna salida.
func apiToken(flagVal string, stderr io.Writer) (token string, ok bool) {
	token = strings.TrimSpace(pick(flagVal, "ACORTA_TOKEN", ""))
	if token == "" {
		fmt.Fprintln(stderr, "error: falta el token de API: usa -token o ACORTA_TOKEN")
		return "", false
	}
	if utf8.RuneCountInString(token) < minTokenLen {
		fmt.Fprintf(stderr, "error: el token de API debe tener al menos %d caracteres\n", minTokenLen)
		return "", false
	}
	return token, true
}

// listenURL convierte una dirección de escucha en la URL que se anuncia:
// sin host (":9090") se dice localhost.
func listenURL(addr string) string {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return "http://" + addr
	}
	if host == "" {
		host = "localhost"
	}
	return "http://" + net.JoinHostPort(host, port)
}
