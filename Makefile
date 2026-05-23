.PHONY: all build test clean install

BINARY_NAME=yc
BUILD_DIR=build

all: build

build:
	@mkdir -p $(BUILD_DIR)
	go build -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/yc

test:
	go test ./...

test-verbose:
	go test -v ./...

clean:
	rm -rf $(BUILD_DIR)

install: build
	@echo "Install to /usr/local/bin requires sudo"
	sudo cp $(BUILD_DIR)/$(BINARY_NAME) /usr/local/bin/

fmt:
	go fmt ./...

vet:
	go vet ./...