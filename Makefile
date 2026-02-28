.PHONY: build test lint release snapshot

GITREV=$(shell git describe --abbrev=0 --always --tags)

build:
	GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build -o ./bin/podgen ./cmd/podgen

test:
	- CGO_ENABLED=0 go test ./...

lint:
	- golangci-lint run

release:
	goreleaser release --rm-dist

snapshot:
	goreleaser release --snapshot --rm-dist