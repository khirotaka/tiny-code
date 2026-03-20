# Get the last part of the module path from go.mod
BINARY_NAME=$(shell basename $(shell grep -m 1 "module" go.mod | awk '{print $$2}'))
BINARY_PATH=bin/$(BINARY_NAME)

.PHONY: all build clean

all: build

# build the go application
build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p bin
	CGO_ENABLED=0 go build -ldflags="-s -w" -trimpath -o $(BINARY_PATH) .
	@echo "Build complete: $(BINARY_PATH)"

# remove the binary
clean:
	@echo "Cleaning up..."
	@rm -rf bin
	@echo "Cleanup complete."
