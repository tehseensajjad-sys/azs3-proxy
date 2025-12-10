.PHONY: all build test lint clean docker-build

# Build variables
BINARY_NAME=azs3-proxy
DOCKER_IMAGE=azs3-proxy
VERSION?=0.1.0

all: lint test build

build:
	go clean
	rm -rf /bin/
	go build -o bin/$(BINARY_NAME) ./cmd/proxy

test:
	go test -v -race ./...

test-compliance:
	go test -v -race ./test/compliance/...

lint:
	golangci-lint run

clean:
	go clean
	rm -rf bin/

docker-build:
	docker build -t $(DOCKER_IMAGE):$(VERSION) .

update-coverage:
	@echo "Running tests and updating coverage..."
	@go test -count=1 -coverprofile=coverage.out ./... > /dev/null || { echo "Tests failed"; exit 1; }
	@COVERAGE=$$(go tool cover -func=coverage.out | grep total | awk '{print substr($$3, 1, length($$3)-1)}'); \
	echo "Current coverage: $$COVERAGE%"; \
	COLOR=$$(echo "$$COVERAGE" | awk '{if ($$1 >= 75.0) print "brightgreen"; else print "yellow"}'); \
	sed -i "s/coverage-[0-9.]*%25-[a-z]*/coverage-$$COVERAGE%25-$$COLOR/" README.md

pre-commit: lint test update-coverage
