# Configurable variables:
# DOCKER_PLATFORM - Docker build platform (default: linux/arm64)
# REGISTRY_URL - Container registry URL (e.g., us-central1-docker.pkg.dev/my-project/registry)

.PHONY: build test clean local deploy image lint

build: clean
	go mod download
	go build ./...

test: build
	go test -v -race -count=1 -coverprofile=coverage.out ./...

# Clean up build artifacts
clean:
	go mod tidy

lint:
	docker run --rm -v $$(pwd):/app \
		-v $$(go env GOCACHE):/.cache/go-build -e GOCACHE=/.cache/go-build \
		-v $$(go env GOMODCACHE):/.cache/mod -e GOMODCACHE=/.cache/mod \
		-w /app \
		golangci/golangci-lint:v2.4.0 \
		golangci-lint run --fix --verbose --output.text.colors --timeout=10m
