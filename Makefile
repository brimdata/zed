export GO111MODULE=on
GOIMPORTS=goimports
PIGEON=pigeon

vet:
	@go vet -copylocks ./...

fmt:
	@res=$$(go fmt ./...); \
	if [ -n "$${res}" ]; then \
		echo "go fmt failed on these files:"; echo "$${res}"; echo; \
		exit 1; \
	fi

test-unit: zql/zql.go js/zql.js
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

node_modules: package.json
	@npm install

zql: zql/zql.go js/zql.js

zql/zql.go: zql/zql.peg
	@cpp -DGO -E -P ./zql/zql.peg | $(PIGEON) -o $@
	@$(GOIMPORTS) -w $@

js/zql.js: node_modules zql/zql.peg
	@cpp -E -P ./zql/zql.peg \
		| npx pegjs -o $@ # \


.PHONY: vet test-unit test-system clean build
