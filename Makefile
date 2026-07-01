.PHONY: check fmt vet test test-race build test-lua lint lua-fmt

check: fmt vet lint test test-race build lua-fmt test-lua

fmt:
	go fmt ./...

vet:
	go vet ./...

lint:
	golangci-lint run

test:
	go test ./...

test-race:
	go test -race ./...

build:
	go build ./...

lua-fmt:
	stylua --check lua/ tests/

test-lua:
	lua tests/run.lua
