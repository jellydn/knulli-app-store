.PHONY: all build catalogue check clean fmt test vet

all: check build catalogue

build:
	mkdir -p build
	CGO_ENABLED=0 go build -trimpath -o build/knulli-app ./cmd/knulli-app

catalogue:
	go run ./cmd/knulli-app catalogue -output build/catalog-index.json

fmt:
	@files="$$(gofmt -l .)"; [ -z "$$files" ] || gofmt -w $$files

test:
	go test -race ./...

vet:
	go vet ./...

check: fmt vet test

clean:
	rm -rf build
