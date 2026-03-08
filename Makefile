.PHONY: build run test clean
.DEFAULT_GOAL := help

BINARY ?= bin/max-ops
GO ?= go

help:
	@echo "max-ops developer commands"
	@echo "  make build    Build ./$(BINARY)"
	@echo "  make run      Build and run"
	@echo "  make test     Run tests"
	@echo "  make clean    Clean"

build:
	@mkdir -p $(dir $(BINARY))
	$(GO) build -o $(BINARY) .

run: build
	./$(BINARY)

test:
	$(GO) test ./...

clean:
	rm -rf bin
