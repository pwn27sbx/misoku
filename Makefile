.PHONY: all build install run test vet fmt lint snapshot clean

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -s -w -X main.version=$(VERSION)

all: fmt vet test build

build:
	go build -trimpath -ldflags "$(LDFLAGS)" -o bin/misoku ./cmd/misoku

install:
	go install -trimpath -ldflags "$(LDFLAGS)" ./cmd/misoku

run:
	go run ./cmd/misoku

test:
	go test ./...

vet:
	go vet ./...

fmt:
	gofmt -w .

# Local dry run of the release pipeline (requires goreleaser).
snapshot:
	goreleaser release --snapshot --clean

clean:
	rm -rf bin/ dist/
