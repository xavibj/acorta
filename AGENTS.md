# AGENTS.md — reglas para agentes que trabajan en acorta

1. **SDD**: `specs/` es el contrato. Lee `specs/001-plan-tecnico.md` y la spec
   de tu fase antes de escribir código. Nada se implementa si no está
   especificado. Si tu cambio contradice una spec, no la ignores: actualízala
   en el mismo cambio y dilo en el informe.
2. **TDD**: tests primero, uno al menos por criterio de la spec (el test nombra
   el identificador: `URL-04`). Ejecútalos y confirma que fallan antes de
   implementar. Después, lo mínimo para el verde. Refactor al final.
3. **No modifiques un test para que pase.** Si crees que un test o un criterio
   están mal, párate y dilo.
4. **Terminado** = `gofmt -l .` vacío, `go vet ./...` y `go test ./...` limpios
   en TODO el repo; si tocas `web/`, también sus tests y `npm run build`.
5. **Propiedad de ficheros**: toca solo las rutas de tu fase en
   `specs/007-fases.md`. Nunca `go.mod`/`go.sum`, nunca `go mod tidy`.
6. **Dependencias**: solo las de `specs/001-plan-tecnico.md`.
7. Identificadores en inglés; comentarios y textos para el usuario en español.
   Los mensajes de error de validación salen solo del paquete `link`.
8. Temporales fuera del repo. No hagas commits: los hace el orquestador.
9. Informe final compacto: ficheros tocados, decisiones y su porqué,
   desviaciones de la spec, y la salida resumida de la suite.
