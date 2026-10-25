BINARY_NAME=shellix
ALIAS_NAME=sx
VERSION=0.1.0-alpha
BUILD_DIR=bin
LDFLAGS=-ldflags="-s -w -X shellix/internal/core.Version=$(VERSION)"

.PHONY: all build clean run test install

all: build

build:
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/shellix
	@ln -sf $(BINARY_NAME) $(BUILD_DIR)/$(ALIAS_NAME)
	@echo "Build complete: $(BUILD_DIR)/$(BINARY_NAME) (symlinked as $(BUILD_DIR)/$(ALIAS_NAME))"

run: build
	./$(BUILD_DIR)/$(BINARY_NAME)

test:
	go test -race -v ./...

clean:
	rm -rf $(BUILD_DIR)
