.PHONY: build test lint

build:
	- GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build -v -ldflags="-X 'main.version=`git describe --tags --abbre v=0`'" -o ./bin/podgen ./cmd/podgen

test:
	- CGO_ENABLED=0 go test ./...

lint:
	- golangci-lint run