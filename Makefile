export GO111MODULE=on

vet:
	@go vet -copylocks ./...

test-unit:
	@go test -short ./...

test-system: build
	@$(MAKE) -C test

build:
	@mkdir -p dist 
	@go build -o dist ./cmd/...

install:
	@go install ./cmd/...

clean:
	@rm -rf dist

.PHONY: vet test-unit test-system clean build

