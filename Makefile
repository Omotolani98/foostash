VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo none)
DATE    ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS := -s -w \
	-X github.com/Omotolani98/foostash/cli.Version=$(VERSION) \
	-X github.com/Omotolani98/foostash/cli.Commit=$(COMMIT) \
	-X github.com/Omotolani98/foostash/cli.Date=$(DATE)

.PHONY: build test vet install clean

build:
	go build -ldflags="$(LDFLAGS)" -o bin/foostash ./cmd/foostash

test:
	go test ./... -v

vet:
	go vet ./...

install:
	go install -ldflags="$(LDFLAGS)" ./cmd/foostash

clean:
	rm -rf bin/
