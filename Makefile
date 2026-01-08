.PHONY: all build test lint clean docker-build run stop test-app collector benchmark

# Build variables
BINARY_NAME=azs3-proxy
DOCKER_IMAGE=azs3-proxy
VERSION?=0.1.0

all: lint test build

build:
	go clean
	rm -rf bin
	rm -rf azs3-proxy
	go build -o $(BINARY_NAME) ./cmd/proxy

run: build
	@echo "Checking for process on port 8080..."
	@pid=$$(lsof -ti :8080); \
	if [ -n "$$pid" ]; then \
		echo "Killing process $$pid on port 8080"; \
		kill -9 $$pid; \
	fi
	@if [ -f .env ]; then \
		set -a && . ./.env && set +a; \
	else \
		echo "Warning: .env file not found. Running without environment variables."; \
	fi; \
	env | grep AZURE; \
	./$(BINARY_NAME) --log-level=debug

stop: 
	./$(BINARY_NAME) --stop
	rm -rf *.pid

test:
	go test -v -race ./...

test-app:
	@echo "Starting interactive test client..."
	@cd examples/python-s3-client && python3 interactive_menu.py

test-compliance:
	go test -v -race ./test/compliance/...

lint:
	golangci-lint run

clean:
	go clean
	rm -rf bin
	rm -rf azs3-proxy

docker-build:
	docker build -t $(DOCKER_IMAGE):$(VERSION) .

update-coverage:
	@echo "Running tests and updating coverage..."
	@go test -count=1 -coverprofile=coverage.out ./... > /dev/null || { echo "Tests failed"; exit 1; }
	@go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"
	@COVERAGE=$$(go tool cover -func=coverage.out | grep total | awk '{print substr($$3, 1, length($$3)-1)}'); \
	echo "Current coverage: $$COVERAGE%"; \
	COLOR=$$(echo "$$COVERAGE" | awk '{if ($$1 >= 75.0) print "brightgreen"; else print "yellow"}'); \
	sed -i "s/coverage-[0-9.]*%25-[a-z]*/coverage-$$COVERAGE%25-$$COLOR/" README.md

pre-commit: lint test update-coverage

collector:
	@echo "Starting OpenTelemetry Collector..."
	@docker rm -f otel-collector 2>/dev/null || true
	@if [ -f .env ]; then \
		echo "Loading .env file..."; \
		export $$(grep -v '^#' .env | xargs); \
	fi; \
	docker run -d --name otel-collector \
		-p 4317:4317 \
		-p 4318:4318 \
		-p 55679:55679 \
		-e AZURE_MONITOR_CONNECTION_STRING="$${AZURE_MONITOR_CONNECTION_STRING}" \
		-v $$(pwd)/otel-collector-config.yaml:/etc/otelcol-contrib/config.yaml \
		otel/opentelemetry-collector-contrib:latest
	@echo "Collector started. View logs with: docker logs -f otel-collector"

benchmark: collector
	@echo "Cleaning up previous benchmark runs..."
	rm -rf warp_runs
	rm -f *.pid
	rm -f azs3-proxy.log
	rm -f $(BINARY_NAME)
	@echo "Running MinIO benchmarks..."
	@if [ -f .env ]; then \
		echo "Loading .env file..."; \
		set -a && . ./.env && set +a; \
	else \
		echo "Warning: .env file not found. Benchmarks might fail if Azure creds are missing."; \
	fi; \
	script -q -c "bash ./test/benchmarking/warp-test.sh" benchmark.log; \
	echo ""; \
	echo "Generating Benchmark Report..."; \
	python3 test/benchmarking/generate_report.py . > warp_report.md
	@echo "Benchmark report generated: warp_report.md"
	@echo "Stopping OpenTelemetry Collector..."
	docker rm -f otel-collector

