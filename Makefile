export GO111MODULE=on

# If VERSION or LDFLAGS change, please also change
# npm/build.
VERSION = $(shell git describe --tags --dirty --always)
LDFLAGS = -s -X github.com/brimdata/zed/cli.Version=$(VERSION)
BUILD_COMMANDS = ./cmd/zed ./cmd/zq

# This enables a shortcut to run a single test from the ./ztests suite, e.g.:
#  make TEST=TestZed/ztests/suite/cut/cut
ifneq "$(TEST)" ""
test-one: test-run
endif

# Uncomment this to trigger re-builds of the peg files when the grammar
# is out of date.  We are commenting this out to work around issue #1717.
#PEG_DEP=peg

vet:
	@go vet -composites=false -stdmethods=false ./...

fmt:
	gofmt -s -w .
	git diff --exit-code -- '*.go'

tidy:
	go mod tidy
	git diff --exit-code -- go.mod go.sum

SAMPLEDATA:=zed-sample-data/README.md

$(SAMPLEDATA):
	git clone --depth=1 https://github.com/brimdata/zed-sample-data $(@D)

sampledata: $(SAMPLEDATA)

bin/minio: Makefile
	curl -o $@ --create-dirs https://dl.min.io/server/minio/release/$$(go env GOOS)-$$(go env GOARCH)/archive/minio.RELEASE.2022-05-04T07-45-27Z
	chmod +x $@

generate:
	@GOBIN="$(CURDIR)/bin" go install github.com/golang/mock/mockgen
	@PATH="$(CURDIR)/bin:$(PATH)" go generate ./...

test-generate: generate
	git diff --exit-code

test-unit:
	@go test -short ./...

test-system: build bin/minio
	@ZTEST_PATH="$(CURDIR)/dist:$(CURDIR)/bin" go test .

test-run: build bin/minio
	@ZTEST_PATH="$(CURDIR)/dist:$(CURDIR)/bin" go test . -v -run $(TEST)

test-heavy: build
	@PATH="$(CURDIR)/dist:$(PATH)" go test -tags=heavy ./mdtest

.PHONY: test-services
test-services: build
	@ZTEST_PATH="$(CURDIR)/dist:$(CURDIR)/bin" \
		ZTEST_TAG=services \
		go test -run TestZed/ppl/zqd/ztests/redis .

perf-compare: build $(SAMPLEDATA)
	scripts/perf-compare.sh

output-check: build $(SAMPLEDATA)
	scripts/output-check.sh

# If the build recipe changes, please also change npm/build.
build: $(PEG_DEP)
	@mkdir -p dist
	@go build -ldflags='$(LDFLAGS)' -o dist ./cmd/...

install:
	@go install -ldflags='$(LDFLAGS)' $(BUILD_COMMANDS)

.PHONY: installdev
installdev:
	@go install -ldflags='$(LDFLAGS)' ./cmd/...

PEG_GEN := $(addprefix compiler/parser/parser., go js es.js)
$(PEG_GEN): compiler/parser/Makefile compiler/parser/support.js compiler/parser/parser.peg
	$(MAKE) -C compiler/parser

# This rule is best for edit-compile-debug cycle of peg development.  It should
# properly trigger rebuilds of peg-generated code, but best to run
# "make -C compiler/parser" when changing versions of pigeon, pegjs, or javascript
# dependencies.
.PHONY: peg peg-run
peg: $(PEG_GEN)

peg-run: $(PEG_GEN)
	go run ./cmd/zc -repl

.PHONY: markdown-lint
markdown-lint:
	@npm install --no-save markdownlint-cli
	@npx markdownlint docs

# CI performs these actions individually since that looks nicer in the UI;
# this is a shortcut so that a local dev can easily run everything.
test-ci: fmt tidy vet test-generate test-unit test-system test-heavy

clean:
	@rm -rf dist

.PHONY: fmt tidy vet test-unit test-system test-heavy sampledata test-ci
.PHONY: perf-compare build install clean generate test-generate
