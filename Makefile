.PHONY: all build build-ui catalogue check clean cover cover-untested fmt gui test test-ui vet walkthrough

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

# Walks the offline GUI flows with the keyboard and records opening, post-key,
# and operation-completion frames. It also renders every screen state, so a
# machine with no device produces review evidence. Add WALK_FLAGS=--install to
# include the network-backed package lifecycle flows.
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

# Coverage gate. A total percentage alone would not notice a new package
# arriving with no tests at all, so this is two checks: every package holding
# production Go files must have a test file, and the total must stay above
# COVER_MIN. The sdl-tagged GUI packages need libsdl2-dev, so they are measured
# by test-ui instead of here.
COVER_MIN ?= 70

cover: cover-untested
	@mkdir -p build
	go test -covermode=atomic -coverprofile=build/cover.out ./...
	@go tool cover -func=build/cover.out | tail -1
	@total=$$(go tool cover -func=build/cover.out | awk '/^total:/ {gsub(/%/,"",$$NF); print $$NF}'); \
	awk -v total="$$total" -v min="$(COVER_MIN)" 'BEGIN { if (total + 0 < min + 0) { printf "coverage %.1f%% is below the %d%% floor\n", total, min; exit 1 } }'

cover-untested:
	@missing=""; \
	for dir in $$(go list -f '{{.Dir}}' ./...); do \
	  ls "$$dir"/*_test.go >/dev/null 2>&1 && continue; \
	  missing="$$missing $${dir#$(CURDIR)/}"; \
	done; \
	if [ -n "$$missing" ]; then \
	  echo "no test file in:$$missing"; \
	  exit 1; \
	fi

vet:
	go vet ./...
	go vet -tags sdl ./...

check: fmt vet test

clean:
	rm -rf build
