# 003 — Almacén (SQLite)

El paquete `store` guarda los enlaces en un fichero SQLite. No conoce las reglas de negocio de la spec 002: no valida URLs ni alias ni decide si un enlace ha caducado.

## Fichero

Un único fichero, por defecto `acorta.db` en el directorio de trabajo. Quién decide la ruta (flag, variable de entorno) está en la spec 005.

## Esquema

```sql
CREATE TABLE IF NOT EXISTS links (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    code       TEXT    NOT NULL UNIQUE,
    url        TEXT    NOT NULL,
    created_at TEXT    NOT NULL,            -- RFC 3339, UTC
    expires_at TEXT,                        -- RFC 3339, UTC; NULL: no caduca
    visits     INTEGER NOT NULL DEFAULT 0
);
```

Al abrir se aplican `PRAGMA journal_mode = WAL` y `PRAGMA busy_timeout = 5000`.

## API

```go
package store

type Store struct{ /* *sql.DB */ }

// Open abre (o crea) la base de datos y su esquema.
func Open(path string) (*Store, error)
func (s *Store) Close() error

func (s *Store) Insert(ctx context.Context, l link.Link) error
func (s *Store) Get(ctx context.Context, code string) (link.Link, error)
func (s *Store) List(ctx context.Context) ([]link.Link, error)
func (s *Store) Delete(ctx context.Context, code string) error
func (s *Store) AddVisit(ctx context.Context, code string) error
```

## Criterios

- **ALM-01** — Dada una ruta que no existe, cuando se abre, entonces se crea el fichero con el esquema. Abrir dos veces la misma ruta no falla ni pierde datos.
- **ALM-02** — Dado un enlace insertado, cuando se cierra el almacén y se vuelve a abrir, entonces `Get` lo devuelve con los mismos valores en todos sus campos (incluida una caducidad nula). `Insert` guarda el enlace tal como llega, también su `Visits`: no lo pone a 0.
- **ALM-03** — Dado un código que ya existe, cuando se inserta otro enlace con ese código, entonces error `link.ErrCodeTaken` y el enlace original no cambia.
- **ALM-04** — `Get` de un código que no existe devuelve `link.ErrNotFound`. La comparación es exacta y distingue mayúsculas.
- **ALM-05** — `List` devuelve los enlaces del último insertado al primero; sin enlaces, una lista vacía (no `nil` con error). El orden es el de inserción (la columna `id`), no el de `created_at`: dos enlaces creados en el mismo segundo siguen teniendo un orden.
- **ALM-06** — `Delete` de un código que existe lo elimina; de uno que no existe devuelve `link.ErrNotFound`. Tras borrarlo, se puede insertar otro enlace con el mismo código.
- **ALM-07** — `AddVisit` suma 1 al contador en una sola sentencia (`UPDATE … SET visits = visits + 1`). Con 50 llamadas simultáneas sobre el mismo código, el contador acaba en 50. Sobre un código que no existe devuelve `link.ErrNotFound`.
- **ALM-08** — Los instantes se guardan y se devuelven en UTC con precisión de segundos. Las fracciones se descartan, no se redondean: `10:30:45.900` se guarda como `10:30:45`.
- **ALM-09** — `Open` con una ruta que no se puede crear (un directorio que no existe) devuelve un error y no deja nada abierto.
