.PHONY: dev build test clean generate-types install

# Development - runs API and web client concurrently
dev:
	pnpm dev

# Install all dependencies
install:
	pnpm install
	cd api && go mod download

# Generate types from OpenAPI spec
generate-types:
	cd api && make generate-types
	pnpm --filter @feather/api-client generate

# Build all
build: generate-types
	cd api && make build
	pnpm --filter @feather/web build

# Run all tests
test:
	cd api && make test
	pnpm -r typecheck

# Clean build artifacts
clean:
	cd api && make clean
	pnpm -r exec rm -rf dist

# Lint all
lint:
	cd api && make lint
	pnpm -r lint
