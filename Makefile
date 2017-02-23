
GO_SRC := $(shell find -type f -name '*.go' ! -path '*/vendor/*')

SRC_ROOT = github.com/wrouesnel/docker-simple-disk
PROGNAME := docker-simple-disk
VERSION ?= git:$(shell git describe --long --dirty)
TAG ?= latest
CONTAINER_NAME ?= wrouesnel/$(PROGNAME):$(TAG)
BUILD_CONTAINER ?= $(PROGNAME)_build

all: vet test style $(PROGNAME)

vet:
	go vet .

# Check code conforms to go fmt
style:
	! gofmt -l $(GO_SRC) 2>&1 | read 2>/dev/null

# Test everything
test:
	go test -coverprofile=coverage.out -v ./...
	
# Format the code
fmt:
	go fmt ./...

# Simple go build
$(PROGNAME): $(GO_SRC)
	GOOS=linux go build -a \
	-ldflags "-extldflags '-static' -X main.Version=$(VERSION)" \
	-o $(PROGNAME) .
	
.PHONY: vet test style
