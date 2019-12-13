export GO111MODULE=on

vet:
	@go vet -copylocks ./...

fmt:
	@res=$$(go fmt ./...); \
	if [ -n "$${res}" ]; then \
		echo "go fmt failed on these files:"; echo "$${res}"; echo; \
		exit 1; \
	fi

test-unit:
	@go test -short ./...

test-system: build
	@go test -v -tags=system ./tests -args PATH=$(shell pwd)/dist

build:
	@mkdir -p dist
	@go build -ldflags='-s -X main.version=$(shell ./scripts/version.sh)' -o dist ./cmd/...

install:
	@go install -ldflags='-s -X main.version=$(shell ./scripts/version.sh)' ./cmd/...

clean:
	@rm -rf dist

.PHONY: vet test-unit test-system clean build
