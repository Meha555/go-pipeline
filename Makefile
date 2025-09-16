all: build

build: # uses `CGO_ENABLED=0` to avoid platform-specific C dependencies.
	CGO_ENABLED=0 go build -ldflags "-X github.com/Meha555/go-pipeline/internal.Version=$(shell git symbolic-ref HEAD | cut -b 12-)-$(shell git describe --tags --always --dirty --abbrev=7 2>/dev/null || echo dev)"

install:
	go install

clean:
	go clean

.PHONY: all build install clean