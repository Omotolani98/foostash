VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo none)
DATE    ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS := -s -w \
	-X github.com/Omotolani98/foostash/cli.Version=$(VERSION) \
	-X github.com/Omotolani98/foostash/cli.Commit=$(COMMIT) \
	-X github.com/Omotolani98/foostash/cli.Date=$(DATE)

.PHONY: build build-server run-server test vet install clean docker-build docker-up docker-down sdk-test sdk-vet

build:
	go build -ldflags="$(LDFLAGS)" -o bin/foostash ./cmd/foostash

build-server:
	go build -ldflags="$(LDFLAGS)" -o bin/foostash-server ./cmd/foostash-server

run-server:
	FOOSTASH_PG_URL=$${FOOSTASH_PG_URL:-postgres://foostash:foostash@localhost:5432/foostash?sslmode=disable} \
	FOOSTASH_LISTEN_ADDR=$${FOOSTASH_LISTEN_ADDR:-:8400} \
	go run ./cmd/foostash-server

test:
	go test ./... -v

vet:
	go vet ./...

install:
	go install -ldflags="$(LDFLAGS)" ./cmd/foostash

clean:
	rm -rf bin/

docker-build:
	docker compose build \
		--build-arg VERSION=$(VERSION) \
		--build-arg COMMIT=$(COMMIT) \
		--build-arg DATE=$(DATE)

docker-up:
	docker compose up -d

docker-down:
	docker compose down

sdk-test:
	cd sdk/go && go test ./... -v

sdk-vet:
	cd sdk/go && go vet ./...
