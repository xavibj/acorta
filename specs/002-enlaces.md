# 002 — Enlaces: reglas de dominio

Qué es un enlace y qué reglas cumple. Estas reglas viven en los paquetes `link` (validación y códigos, puro) y `shortener` (casos de uso). La API, la CLI y la web no añaden ni quitan ninguna.

Cada criterio tiene un identificador (`URL-04`). El test que lo comprueba lo nombra.

## El enlace

| Campo | Tipo | Notas |
|---|---|---|
| `Code` | texto | Único. Generado o alias |
| `URL` | texto | La URL de destino, tal como se guardó |
| `CreatedAt` | instante UTC | Lo pone el servicio al crear |
| `ExpiresAt` | instante UTC, opcional | Sin valor: no caduca |
| `Visits` | entero ≥ 0 | Empieza en 0 |

## Crear un enlace

Entrada: `url` (obligatoria), `alias` (opcional) y `expires_at` (opcional), los tres como texto. A los tres se les recortan los espacios de los extremos antes de validar; un `alias` o un `expires_at` que quedan vacíos cuentan como no enviados.

La validación devuelve **todos los errores a la vez**: como mucho uno por campo y en el orden `url`, `alias`, `expires_at`. Dentro de un campo se da solo el primer error, comprobando las reglas en el orden en que están escritas aquí: `ftp://` da el error de esquema (URL-06) y no el de dominio (URL-08); el alias `-Abc` da el de caracteres (ALI-03) y no el de guion (ALI-04). Si hay algún error no se guarda nada.

### URL de destino

- **URL-01** — Dado `https://example.com/a?b=1#c`, cuando se crea el enlace, entonces se acepta y la URL se guarda exactamente igual, sin normalizar.
- **URL-02** — Dado `  https://example.com  `, cuando se crea, entonces se guarda `https://example.com`.
- **URL-03** — Dada una URL vacía o solo con espacios, entonces error `url: es obligatoria`.
- **URL-04** — Dada una URL de más de 2048 bytes, entonces error `url: no puede superar los 2048 bytes`. Con 2048 exactos se acepta. Se miden bytes, no caracteres (una `ñ` son dos), y después de recortar los espacios.
- **URL-05** — Dado un texto que no se puede interpretar como URL (`http://exa mple.com`), entonces error `url: no es una URL válida`.
- **URL-06** — Dado un esquema que no es `http` ni `https` (`ftp://example.com/f`, `javascript:alert(1)`, `mailto:a@b.c`) o ninguno (`example.com`, `/ruta`), entonces error `url: debe empezar por http:// o https://`. No se adivina el esquema.
- **URL-07** — El esquema no distingue mayúsculas: `HTTPS://Example.com` se acepta y se guarda tal cual.
- **URL-08** — Dada una URL sin dominio (`https://`, `http:///ruta`, `https://:8080`), entonces error `url: falta el dominio`.

### Código generado

Cuando no se envía alias, acorta genera el código.

- **COD-01** — Un código generado tiene exactamente 6 caracteres del alfabeto `abcdefghijklmnopqrstuvwxyz0123456789`. Cada carácter sale de un byte de la fuente de azar: el byte `b` da el carácter de la posición `b % 36`, y los bytes de 252 en adelante se descartan y se lee otro, para que los 36 caracteres sean igual de probables. Los bytes `255, 0, 1, 2, 3, 4, 5` dan `abcdef`. Si la fuente falla o se agota, `NewCode` devuelve ese error envuelto.
- **COD-02** — Si el código generado ya existe o es una palabra reservada, se descarta y se genera otro, hasta 5 intentos en total. Si los 5 fallan, crear devuelve el error `ErrNoCodeAvailable` y no guarda nada. Solo se reintenta por esos dos motivos: si falla el generador, o el almacén da cualquier otro error, crear devuelve ese error envuelto, sin más intentos y sin guardar nada.
- **COD-03** — Acortar dos veces la misma URL crea dos enlaces con códigos distintos.
- **COD-04** — Los códigos salen de una fuente de azar criptográfica (`crypto/rand`): no son consecutivos ni predecibles. El generador se inyecta en el servicio para que los tests usen uno determinista.

### Alias

Cuando se envía alias, ese es el código. No se transforma: o cumple las reglas o es un error.

- **ALI-01** — Dado el alias `oferta-otono`, cuando se crea, entonces el código del enlace es `oferta-otono`.
- **ALI-02** — Dado un alias de menos de 3 o de más de 32 caracteres, entonces error `alias: debe tener entre 3 y 32 caracteres`. Con 3 y con 32 se acepta. Se cuentan caracteres, no bytes: `ñu` tiene 2 y da este error; `ñus` tiene 3 y da el de ALI-03.
- **ALI-03** — Dado un alias con algún carácter que no sea `a-z`, `0-9` o `-` (`Oferta`, `mi_enlace`, `año-nuevo`, `a b c`), entonces error `alias: solo admite minúsculas, números y guiones`. Las mayúsculas no se convierten.
- **ALI-04** — Dado un alias que empieza o termina por guion (`-oferta`, `oferta-`), entonces error `alias: no puede empezar ni terminar por guion`.
- **ALI-05** — Dado un alias que es una palabra reservada (`api`, `assets`), entonces error `alias: "api" está reservado`.
- **ALI-06** — Dado un alias que ya usa otro enlace, aunque ese enlace esté caducado, entonces crear devuelve un `*AliasTakenError`, con el mensaje `alias: "promo" ya está en uso`, y no guarda nada. Este error solo se comprueba cuando la validación ha pasado.

Las palabras reservadas son las rutas que el servidor necesita para sí: `api` y `assets`.

### Caducidad

- **CAD-01** — Sin `expires_at`, el enlace no caduca nunca.
- **CAD-02** — Dado un `expires_at` que no está en formato RFC 3339 (`mañana`, `2026-12-31`), entonces error `expires_at: debe tener formato RFC 3339 (2026-12-31T23:59:59Z)`.
- **CAD-03** — Dado un `expires_at` igual o anterior al momento de crear, entonces error `expires_at: debe ser una fecha futura`. Se comparan los dos instantes ya sin fracciones de segundo: con el reloj en `12:00:00`, un `expires_at` de `12:00:00.5` es igual y da error; uno de `12:00:01`, no.
- **CAD-04** — Un `expires_at` con otra zona horaria (`2026-12-31T23:59:59+02:00`) se acepta y se guarda convertido a UTC, sin fracciones de segundo.
- **CAD-05** — Un enlace está caducado cuando el momento actual es igual o posterior a su `expires_at`.
- **CAD-06** — Un enlace caducado no se borra solo: sigue en el listado, marcado como caducado, y su código sigue ocupado hasta que alguien lo borra.

## Resolver un código

Es lo que ocurre cuando alguien abre la URL corta.

- **RES-01** — Dado un código que existe y no ha caducado, cuando se resuelve, entonces devuelve su URL de destino.
- **RES-02** — Dado un código que no existe, entonces error `ErrNotFound`.
- **RES-03** — Dado un código caducado, entonces error `ErrExpired`.
- **RES-04** — La búsqueda es exacta y distingue mayúsculas: si existe `oferta`, resolver `Oferta` da `ErrNotFound`.
- **RES-05** — Si no se puede sumar la visita, no se devuelve la URL: resolver devuelve el error del almacén. Si lo que ha pasado es que alguien ha borrado el enlace entre que se encontró y se contó, el error es `ErrNotFound`.

## Visitas

- **VIS-01** — Un enlace recién creado tiene 0 visitas.
- **VIS-02** — Cada resolución con éxito suma exactamente 1 visita. Con 50 resoluciones simultáneas del mismo código, el contador acaba en 50.
- **VIS-03** — Una resolución que falla (no existe, caducado) no suma nada.
- **VIS-04** — Se puede resolver sin contar la visita (lo usa `HEAD`, ver spec 004): devuelve lo mismo que RES-01 a RES-04 y deja el contador como estaba.

## Listar

- **LIS-01** — Devuelve todos los enlaces, del más reciente al más antiguo según el orden en que se crearon.
- **LIS-02** — Cada enlace lleva código, URL, visitas, fecha de creación y caducidad (si tiene). Si está caducado en el momento de listar no es un campo guardado: lo calcula quien muestra la lista con `l.Expired(svc.Now())`, para que la API, la CLI y los tests usen el mismo reloj.
- **LIS-03** — Sin enlaces, devuelve una lista vacía, no un error.

## Borrar

- **BOR-01** — Dado un código que existe, cuando se borra, entonces el enlace desaparece del listado y resolverlo da `ErrNotFound`.
- **BOR-02** — Dado un código que no existe, entonces error `ErrNotFound`.
- **BOR-03** — Tras borrar un enlace, su código vuelve a estar libre: se puede crear otro con ese alias.

## Contrato de los paquetes

```go
package link

type Link struct {
    Code      string
    URL       string
    CreatedAt time.Time
    ExpiresAt *time.Time // nil: no caduca
    Visits    int64
}

// Expired indica si el enlace ha caducado en el instante now (CAD-05).
func (l Link) Expired(now time.Time) bool

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
func Validate(in Input, now time.Time) (Valid, []string)

// NewCode lee de r los bytes que necesite y devuelve un código (COD-01).
func NewCode(r io.Reader) (string, error)

// Reserved indica si code es una palabra reservada.
func Reserved(code string) bool

var (
    ErrNotFound        = errors.New("enlace no encontrado")
    ErrExpired         = errors.New("enlace caducado")
    ErrCodeTaken       = errors.New("código en uso")
    ErrNoCodeAvailable = errors.New("no se ha podido generar un código libre")
)

// ValidationError agrupa los mensajes de Validate. Su Error() los une
// con "; ": `url: es obligatoria; alias: "api" está reservado`.
type ValidationError struct{ Messages []string }

// AliasTakenError es el ALI-06. Su mensaje es `alias: "promo" ya está en uso`.
type AliasTakenError struct{ Alias string }
```

```go
package shortener

// Store es lo que el servicio necesita del almacén (lo implementa store.Store).
type Store interface {
    Insert(ctx context.Context, l link.Link) error // link.ErrCodeTaken si el código existe
    Get(ctx context.Context, code string) (link.Link, error)
    List(ctx context.Context) ([]link.Link, error)
    Delete(ctx context.Context, code string) error
    AddVisit(ctx context.Context, code string) error
}

type Service struct{ /* almacén, reloj, generador */ }

// New crea el servicio. now y newCode se pueden sustituir en los tests;
// con nil usa time.Now y link.NewCode sobre crypto/rand.
func New(s Store, now func() time.Time, newCode func() (string, error)) *Service

func (s *Service) Create(ctx context.Context, in link.Input) (link.Link, error)
func (s *Service) Resolve(ctx context.Context, code string, countVisit bool) (string, error)
func (s *Service) List(ctx context.Context) ([]link.Link, error)
func (s *Service) Delete(ctx context.Context, code string) error

// Now devuelve el instante actual según el reloj del servicio (para LIS-02).
func (s *Service) Now() time.Time
```

`Create` devuelve `*link.ValidationError`, `*link.AliasTakenError` o `link.ErrNoCodeAvailable` según el caso; cualquier otro error es un fallo del almacén.
