export GO111MODULE=on

# If VERSION or LDFLAGS change, please also change
# npm/build.
VERSION = $(shell git describe --tags --dirty --always)
LDFLAGS = -s -X main.version=$(VERSION)
ZEEKTAG = v3.0.2-brim3
ZEEKPATH = zeek-$(ZEEKTAG)

# This enables a shortcut to run a single test from the ./tests suite, e.g.:
# make TEST=TestZTest/suite/cut/cut
ifneq "$(TEST)" ""
test-one: test-run
endif

vet:
	@go vet -copylocks ./...

fmt:
	@res=$$(go fmt ./...); \
	if [ -n "$${res}" ]; then \
		echo "go fmt failed on these files:"; echo "$${res}"; echo; \
		exit 1; \
	fi

tidy:
	go mod tidy
	git diff --exit-code -- go.mod go.sum

SAMPLEDATA:=zq-sample-data/README.md

$(SAMPLEDATA):
	git clone --depth=1 https://github.com/brimsec/zq-sample-data $(@D)

sampledata: $(SAMPLEDATA)

bin/$(ZEEKPATH):
	@mkdir -p bin
	@curl -L -o bin/$(ZEEKPATH).zip \
		https://github.com/brimsec/zeek/releases/download/$(ZEEKTAG)/zeek-$(ZEEKTAG).$$(go env GOOS)-$$(go env GOARCH).zip
	@unzip -q bin/$(ZEEKPATH).zip -d bin \
		&& mv bin/zeek bin/$(ZEEKPATH)

bin/minio:
	@GOBIN=$(CURDIR)/bin go install github.com/minio/minio

generate:
	@GOBIN=$(CURDIR)/bin go install github.com/golang/mock/mockgen
	@PATH=$(CURDIR)/bin:$(PATH) go generate ./...

test-generate: generate
	git diff --exit-code

test-unit:
	@go test -short ./...

test-system: build bin/minio
	@ZTEST_PATH=$(CURDIR)/dist:$(CURDIR)/bin go test -v ./tests

test-run: build bin/minio
	@ZTEST_PATH=$(CURDIR)/dist:$(CURDIR)/bin go test -v ./tests -run $(TEST)

test-heavy: build $(SAMPLEDATA)
	@go test -v -tags=heavy ./tests

test-zeek: bin/$(ZEEKPATH)
	@ZEEK=$(CURDIR)/bin/$(ZEEKPATH)/zeekrunner go test -v -run=PcapPost -tags=zeek ./zqd

perf-compare: build $(SAMPLEDATA)
	scripts/comparison-test.sh

# If the build recipe changes, please also change npm/build.
build:
	@mkdir -p dist
	@go build -ldflags='$(LDFLAGS)' -o dist ./cmd/...

install:
	@go install -ldflags='$(LDFLAGS)' ./cmd/...

create-release-assets:
	for os in darwin linux windows; do \
		zqdir=zq-$(VERSION).$${os}-amd64 ; \
		rm -rf dist/$${zqdir} ; \
		mkdir -p dist/$${zqdir} ; \
		GOOS=$${os} GOARCH=amd64 go build -ldflags='$(LDFLAGS)' -o dist/$${zqdir} ./cmd/... ; \
	done
	rm -rf dist/release && mkdir -p dist/release
	cd dist && for d in zq-$(VERSION)* ; do \
		zip -r release/$${d}.zip $${d} ; \
	done

build-python-wheel: build-python-lib
	pip3 wheel --no-deps -w dist ./python

build-python-lib:
	@mkdir -p python/build/zqext
	go build -buildmode=c-archive -o python/build/zqext/libzqext.a ./python/src/zqext.go

clean-python:
	@rm -rf python/build

# CI performs these actions individually since that looks nicer in the UI;
# this is a shortcut so that a local dev can easily run everything.
test-ci: fmt tidy vet test-generate test-unit test-system test-zeek test-heavy

clean: clean-python
	@rm -rf dist

.PHONY: fmt tidy vet test-unit test-system test-heavy sampledata test-ci
.PHONY: perf-compare build install create-release-assets clean clean-python
.PHONY: build-python-wheel generate test-generate bin/minio
