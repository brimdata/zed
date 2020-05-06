export GO111MODULE=on

# If VERSION or LDFLAGS change, please also change
# npm/build.
VERSION = $(shell git describe --tags --dirty --always)
LDFLAGS = -s -X main.version=$(VERSION)
ZEEKTAG = v3.0.2-brim1
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


test-unit:
	@go test -short ./...

test-system: build
	@ZTEST_BINDIR=$(CURDIR)/dist go test -v ./tests

test-run: build
	@ZTEST_BINDIR=$(CURDIR)/dist go test -v ./tests -run $(TEST)

test-heavy: build $(SAMPLEDATA)
	@go test -v -tags=heavy ./tests

test-zeek: bin/$(ZEEKPATH)
	@ZEEK=$(CURDIR)/bin/$(ZEEKPATH)/zeek go test -v -run=PcapPost -tags=zeek ./zqd

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

# CI performs these actions individually since that looks nicer in the UI;
# this is a shortcut so that a local dev can easily run everything.
test-ci: fmt tidy vet test-unit test-system test-zeek test-heavy

clean:
	@rm -rf dist

.PHONY: fmt tidy vet test-unit test-system test-heavy sampledata test-ci
.PHONY: perf-compare build install create-release-assets clean
