# Configurable variables:
# DOCKER_PLATFORM - Docker build platform (default: linux/arm64)
# REGISTRY_URL - Container registry URL (e.g., us-central1-docker.pkg.dev/my-project/registry)

.PHONY: build test clean local deploy image lint

build: clean
	go mod download
	go build ./...

test: build
	go test -v -race -count=1 ./...

# Clean up build artifacts
clean:
	go mod tidy
