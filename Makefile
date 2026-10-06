.PHONY: all build install test clean

BINARY_NAME := huggingface-cli
INSTALL_DIR := $(HOME)/.local/bin
LDFLAGS := -ldflags="-s -w"

all: test build

build:
	mkdir -p bin
	go build $(LDFLAGS) -o bin/$(BINARY_NAME) ./cmd/huggingface-cli
	ln -sf $(BINARY_NAME) bin/hf

install: build
	mkdir -p $(INSTALL_DIR)
	install -m 755 bin/$(BINARY_NAME) $(INSTALL_DIR)/$(BINARY_NAME)
	ln -sf $(BINARY_NAME) $(INSTALL_DIR)/hf
	@echo "Installed $(BINARY_NAME) and hf to $(INSTALL_DIR)"

test:
	go test -v ./...

clean:
	rm -rf bin
