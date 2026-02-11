.PHONY: build run test clean dev generate generate-types fmt

BINARY_NAME=feather
BUILD_DIR=./bin
VERSION ?= dev

generate:
	@mkdir -p internal/openapi
	go generate ./internal/openapi/...

generate-types: generate

build: generate
	go build -ldflags "-X github.com/feather/api/internal/version.Version=$(VERSION)" -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/feather

run: build
	$(BUILD_DIR)/$(BINARY_NAME)

dev:
	go run ./cmd/feather

test:
	go test -v ./...

clean:
	rm -rf $(BUILD_DIR)
	go clean

deps:
	go mod tidy
	go mod download

lint:
	golangci-lint run ./...

fmt:
	gofmt -w .
