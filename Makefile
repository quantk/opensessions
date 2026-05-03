GO ?= go
CGO_ENABLED ?= 0
BINARY ?= opensession
BUILD_DIR ?= bin
CMD ?= ./cmd/opensession

.PHONY: build test run clean

build:
	mkdir -p $(BUILD_DIR)
	CGO_ENABLED=$(CGO_ENABLED) $(GO) build -o $(BUILD_DIR)/$(BINARY) $(CMD)

test:
	$(GO) test ./...

run:
	CGO_ENABLED=$(CGO_ENABLED) $(GO) run $(CMD)

clean:
	rm -rf $(BUILD_DIR)
