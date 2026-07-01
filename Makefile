.PHONY: check fmt vet test test-race build test-lua

check: fmt vet test test-race build test-lua

fmt:
	go fmt ./...

vet:
	go vet ./...

test:
	go test ./...

test-race:
	go test -race ./...

build:
	go build ./...

test-lua:
	lua tests/run.lua
