.PHONY: build run test clean dev generate-types

BINARY_NAME=feather
BUILD_DIR=./bin

generate-types:
	@mkdir -p internal/api
	oapi-codegen --config=oapi-codegen.yaml openapi.yaml

build:
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
