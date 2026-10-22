.PHONY: all build run test test-v clean lint tidy

APP_NAME = content-engine
GO = go

all: test build

build:
	@mkdir -p bin
	$(GO) build -o bin/$(APP_NAME) ./cmd/content-engine

run:
	$(GO) run ./cmd/content-engine

run-once:
	$(GO) run ./cmd/content-engine --mode=once

run-schedule:
	$(GO) run ./cmd/content-engine --mode=schedule

test:
	$(GO) test -v ./...

clean:
	@rm -rf bin/ tmp/*.mp4 data/*.db* data/*.jsonl

tidy:
	$(GO) mod tidy
