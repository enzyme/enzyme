.PHONY: dev build test clean generate-types install format format-check seed

# Development - runs API and web (add DESKTOP=1 for Electron, MOBILE=1 for Expo)
dev:
	trap 'kill 0' EXIT; \
	(cd server && make dev) & \
	pnpm --filter @enzyme/web dev & \
	if [ "$(DESKTOP)" = "1" ]; then pnpm --filter @enzyme/desktop dev & fi; \
	if [ "$(MOBILE)" = "1" ]; then pnpm --filter @enzyme/mobile dev & fi; \
	wait

# Install all dependencies
install:
	pnpm install
	cd server && go mod download

# Generate types from OpenAPI spec
generate-types:
	cd server && make generate-types
	pnpm --filter @enzyme/api-client generate

# Build all (web first so it can be embedded into the Go binary)
build: generate-types
	pnpm --filter @enzyme/web build
	pnpm --filter @enzyme/website build
	cd server && make build
	pnpm --filter @enzyme/desktop build
	pnpm --filter @enzyme/desktop make

# Run all tests
test:
	cd server && make test
	pnpm -r test:run
	pnpm -r typecheck

# Clean build artifacts
clean:
	cd server && make clean
	pnpm -r exec rm -rf dist
	pnpm --filter @enzyme/website exec rm -rf _site

# Lint all
lint:
	cd server && make lint
	pnpm -r lint

# Format all code
format:
	cd server && make fmt
	pnpm format

# Seed database with sample data
seed:
	cd server && make seed

# Check formatting (for CI)
format-check:
	cd server && test -z "$$(gofmt -l .)" || (echo "Go files not formatted"; exit 1)
	pnpm format:check
