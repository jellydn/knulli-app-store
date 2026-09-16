.PHONY: all build build-ui catalogue check clean fmt gui test test-ui vet walkthrough

all: check build catalogue

build:
	mkdir -p build
	CGO_ENABLED=0 go build -trimpath -o build/knulli-app ./cmd/knulli-app

# Desktop GUI verification without a device: the keyboard drives every screen
# and the scratch root supplies the Knulli platform files the detector reads.
# Override GUI_DEVICE, GUI_ARCH, GUI_RESOLUTION or GUI_ROOT for another target.
GUI_DEVICE ?= trimui-smart-pro
GUI_ARCH ?= aarch64
GUI_RESOLUTION ?= 1280x720
GUI_ROOT ?= $(CURDIR)/build/scratch-root

gui: build-ui catalogue
	scripts/desktop-fixture.sh "$(GUI_DEVICE)" "$(GUI_ROOT)"
	./build/knulli-app-ui -input keyboard -windowed -root "$(GUI_ROOT)" -arch "$(GUI_ARCH)" -resolution "$(GUI_RESOLUTION)"

# Walks every GUI flow with the keyboard and writes one frame per step, then
# renders every screen state, so a machine with no device produces the screen
# evidence a PR review or a device test can lean on. Add WALK_FLAGS=--install to
# include the flows that download a package.
WALK_ROOT ?= $(CURDIR)/build/walkthrough-root
WALK_OUT ?= $(CURDIR)/build/walkthrough
WALK_FLAGS ?=

walkthrough: build-ui catalogue
	scripts/desktop-walk.sh "$(WALK_ROOT)" "$(WALK_OUT)" $(WALK_FLAGS)
	KNULLI_UI_SCREENSHOT_DIR="$(WALK_OUT)/screens" go test -tags sdl ./internal/sdlui

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
