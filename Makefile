BRANCH := $(shell git symbolic-ref --short HEAD 2>/dev/null || echo unknown)
VERSION := $(shell git describe --tags --always --dirty --abbrev=7 2>/dev/null || echo dev)
VERSION_WITH_V := $(if $(filter v%,$(VERSION)),$(VERSION),v$(VERSION))

all: build

build: # uses `CGO_ENABLED=0` to avoid platform-specific C dependencies.
	CGO_ENABLED=0 go build -ldflags "-s -w -X github.com/Meha555/go-pipeline/internal.Version=$(BRANCH)-$(VERSION_WITH_V)"

install:
	go install

clean:
	go clean

.PHONY: all build install clean
