# Image URL to use all building/pushing image targets

BINDIR=bin
BIN_NAME=ovirt-image-upload
REV=$(shell git describe --long --tags --match='v*' --always --dirty)

all: build

# Run tests
.PHONY: test
test:
	go test ./pkg/... ./cmd/... -coverprofile cover.out

# Build the binary
.PHONY: build
build: 
	go build -o $(BINDIR)/$(BIN_NAME) -ldflags '-X main.version=$(REV) -extldflags "-static"' `git rev-parse --show-toplevel` 

.PHONY: verify
verify:
	hack/verify-gofmt.sh
	hack/verify-govet.sh

.PHONY: vendor
vendor:
	go mod tidy
	go mod vendor
	go mod verify

