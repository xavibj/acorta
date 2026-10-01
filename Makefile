# Las órdenes de la spec 001. El frontend se añade en la fase 6; hasta
# entonces, los pasos de web/ se saltan si el proyecto no existe.
.PHONY: test check frontend build

WEB := $(wildcard web/package.json)

test:
	go test ./...
ifneq ($(WEB),)
	cd web && npm test
endif

check:
	@test -z "$$(gofmt -l .)" || { echo "gofmt: ficheros sin formatear:"; gofmt -l .; exit 1; }
	go vet ./...
	$(MAKE) test
ifneq ($(WEB),)
	cd web && npm run build
endif

frontend:
	cd web && npm ci && npm run build

build: frontend
	CGO_ENABLED=0 go build -trimpath -ldflags='-s -w' -o bin/acorta ./cmd/acorta
