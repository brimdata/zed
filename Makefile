export GO111MODULE=on

VERSION = $(shell git describe --tags --dirty)
LDFLAGS = -s -X main.version=$(VERSION)

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

test-unit:
	@go test -short ./...

test-system: build
	@ZTEST_ZQ=$(CURDIR)/dist/zq go test -v -tags=system ./tests -args PATH=$(shell pwd)/dist

test-heavy: build $(SAMPLEDATA)
	@go test -v -tags=heavy ./tests

perf-compare: build $(SAMPLEDATA)
	scripts/comparison-test.sh

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

clean:
	@rm -rf dist

.PHONY: vet fmt tidy sampledata test-unit test-system test-heavy perf-compare build install create-release-assets clean
