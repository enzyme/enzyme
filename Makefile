.PHONY: dev build test clean generate-types install format format-check seed

# Development - runs API and web (add DESKTOP=1 for Electron)
dev:
	trap 'kill 0' EXIT; \
	(cd api && make dev) & \
	pnpm --filter @enzyme/web dev & \
	if [ "$(DESKTOP)" = "1" ]; then pnpm --filter @enzyme/desktop dev & fi; \
	wait

# Install all dependencies
install:
	pnpm install
	cd api && go mod download

# Generate types from OpenAPI spec
generate-types:
	cd api && make generate-types
	pnpm --filter @enzyme/api-client generate

# Build all (web first so it can be embedded into the Go binary)
build: generate-types
	pnpm --filter @enzyme/web build
	pnpm --filter @enzyme/website build
	cd api && make build
	pnpm --filter @enzyme/desktop build
	pnpm --filter @enzyme/desktop make

# Run all tests
test:
	cd api && make test
	pnpm --filter @enzyme/web test:run
	pnpm -r typecheck

# Clean build artifacts
clean:
	cd api && make clean
	pnpm -r exec rm -rf dist
	pnpm --filter @enzyme/website exec rm -rf _site

# Lint all
lint:
	cd api && make lint
	pnpm -r lint

# Format all code
format:
	cd api && make fmt
	pnpm format

# Seed database with sample data
seed:
	cd api && make seed

# Check formatting (for CI)
format-check:
	cd api && test -z "$$(gofmt -l .)" || (echo "Go files not formatted"; exit 1)
	pnpm format:check
