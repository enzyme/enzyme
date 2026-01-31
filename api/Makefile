.PHONY: build run test clean dev generate generate-types

BINARY_NAME=feather
BUILD_DIR=./bin

generate:
	@mkdir -p internal/api
	go generate ./internal/api/...

generate-types: generate

build: generate
	go build -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/feather

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
	go vet ./...
