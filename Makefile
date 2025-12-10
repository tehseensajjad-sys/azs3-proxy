.PHONY: all build test lint clean docker-build

# Build variables
BINARY_NAME=azs3-proxy
DOCKER_IMAGE=azs3-proxy
VERSION?=0.1.0

all: lint test build

build:
	go build -o bin/$(BINARY_NAME) ./cmd/proxy

test:
	go test -v -race ./...

lint:
	golangci-lint run

clean:
	go clean
	rm -rf bin/

docker-build:
	docker build -t $(DOCKER_IMAGE):$(VERSION) .
