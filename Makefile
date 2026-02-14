.PHONY: dev build test clean generate-types install format format-check

# Development - runs API and web (add DESKTOP=1 for Electron)
dev:
	trap 'kill 0' EXIT; \
	(cd api && make dev) & \
	pnpm --filter @feather/web dev & \
	if [ "$(DESKTOP)" = "1" ]; then pnpm --filter @feather/desktop dev & fi; \
	wait

# Install all dependencies
install:
	pnpm install
	cd api && go mod download

# Generate types from OpenAPI spec
generate-types:
	cd api && make generate-types
	pnpm --filter @feather/api-client generate

# Build all (web first so it can be embedded into the Go binary)
build: generate-types
	pnpm --filter @feather/web build
	cd api && make build
	pnpm --filter @feather/desktop build
	pnpm --filter @feather/desktop make

# Run all tests
test:
	cd api && make test
	pnpm --filter @feather/web test:run
	pnpm -r typecheck

# Clean build artifacts
clean:
	cd api && make clean
	pnpm -r exec rm -rf dist

# Lint all
lint:
	cd api && make lint
	pnpm -r lint

# Format all code
format:
	cd api && make fmt
	pnpm format

# Check formatting (for CI)
format-check:
	cd api && test -z "$$(gofmt -l .)" || (echo "Go files not formatted"; exit 1)
	pnpm format:check
