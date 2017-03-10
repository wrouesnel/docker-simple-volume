
GO_SRC := $(shell find -type f -name '*.go' ! -path '*/vendor/*')

SRC_ROOT = github.com/wrouesnel/docker-simple-disk

VERSION ?= $(shell git describe --long --dirty)
TAG ?= latest
CONTAINER_NAME ?= wrouesnel/$(PROGNAME):$(TAG)
BUILD_CONTAINER ?= $(PROGNAME)_build

all: vet test style bin

vet:
	go vet ./...

# Check code conforms to go fmt
style:
	! gofmt -l $(GO_SRC) 2>&1 | read 2>/dev/null

# Test everything
test:
	go test -covermode=count -coverprofile=coverage.out -v ./...
	
# Format the code
fmt:
	go fmt ./...

bin: bin/docker-simple-disk bin/simple-test-query

# Simple go build
bin/docker-simple-disk: $(GO_SRC)
	GOOS=linux go build -ldflags "-X main.Version=$(VERSION)" \
	-o bin/docker-simple-disk ./cmd/docker-simple-disk

bin/simple-test-query: $(GO_SRC)
	GOOS=linux go build -ldflags "-X main.Version=$(VERSION)" \
	-o bin/simple-test-query ./cmd/simple-test-query

.PHONY: vet test style bin
