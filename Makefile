.PHONY: build run test clean
.DEFAULT_GOAL := help

BINARY ?= bin/gh-helm
GO ?= go

help:
	@echo "gh-helm developer commands"
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
