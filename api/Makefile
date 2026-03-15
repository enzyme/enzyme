.PHONY: build run test clean dev generate generate-types fmt embed-web seed

BINARY_NAME=enzyme
BUILD_DIR=./bin
VERSION ?= dev
WEB_DIST_SRC=../apps/web/dist
WEB_DIST_DST=./internal/web/dist

generate:
	@mkdir -p internal/openapi
	oapi-codegen --config oapi-codegen.yaml openapi.yaml

generate-types: generate

embed-web:
	@if [ -d "$(WEB_DIST_SRC)" ] && [ -f "$(WEB_DIST_SRC)/index.html" ]; then \
		echo "Embedding web client from $(WEB_DIST_SRC)"; \
		rm -rf $(WEB_DIST_DST)/assets $(WEB_DIST_DST)/*.html $(WEB_DIST_DST)/*.js $(WEB_DIST_DST)/*.ico $(WEB_DIST_DST)/*.svg $(WEB_DIST_DST)/*.png $(WEB_DIST_DST)/*.json; \
		cp -r $(WEB_DIST_SRC)/* $(WEB_DIST_DST)/; \
	else \
		echo "Web dist not found at $(WEB_DIST_SRC), skipping embed"; \
	fi

build: generate embed-web
	go build -ldflags "-X github.com/enzyme/api/internal/version.Version=$(VERSION)" -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/enzyme

run: build
	$(BUILD_DIR)/$(BINARY_NAME)

dev:
	go run ./cmd/enzyme

test:
	go test -v ./...

clean:
	rm -rf $(BUILD_DIR)
	cd $(WEB_DIST_DST) && find . -not -name '.gitkeep' -not -name '.' -depth -delete 2>/dev/null || true
	go clean

deps:
	go mod tidy
	go mod download

lint:
	golangci-lint run ./...

seed:
	go run ./cmd/enzyme seed

fmt:
	gofmt -w .
