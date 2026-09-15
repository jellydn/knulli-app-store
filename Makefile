.PHONY: all build build-ui catalogue check clean fmt test test-ui vet

all: check build catalogue

build:
	mkdir -p build
	CGO_ENABLED=0 go build -trimpath -o build/knulli-app ./cmd/knulli-app

build-ui:
	mkdir -p build
	go build -tags sdl -trimpath -o build/knulli-app-ui ./cmd/knulli-app-ui

catalogue:
	go run ./cmd/knulli-app catalogue -output build/catalog-index.json

fmt:
	@files="$$(gofmt -l .)"; [ -z "$$files" ] || gofmt -w $$files

test:
	go test -race ./...

test-ui:
	go test -tags sdl ./...

vet:
	go vet ./...
	go vet -tags sdl ./...

check: fmt vet test

clean:
	rm -rf build
