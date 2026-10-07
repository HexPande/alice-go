.PHONY: run build package test lint fmt

HTTP_ADDR ?= 0.0.0.0:8090

run:
	go run ./cmd/alice serve --http=$(HTTP_ADDR) --dir=./pb_data

build:
	go build -o bin/alice ./cmd/alice

package:
	bash scripts/package.sh

test:
	go test -race ./...

lint:
	go vet ./...
	golangci-lint run ./...

fmt:
	gofmt -w cmd internal
