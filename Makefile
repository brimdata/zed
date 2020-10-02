export GO111MODULE=on

# If VERSION or LDFLAGS change, please also change
# npm/build.
VERSION = $(shell git describe --tags --dirty --always)
LDFLAGS = -s -X github.com/brimsec/zq/cli.Version=$(VERSION)
ZEEKTAG = v3.0.2-brim3
ZEEKPATH = zeek-$(ZEEKTAG)
SURICATATAG = v5.0.3-brim3
SURICATAPATH = suricata-$(SURICATATAG)

# This enables a shortcut to run a single test from the ./tests suite, e.g.:
# make TEST=TestZTest/suite/cut/cut
ifneq "$(TEST)" ""
test-one: test-run
endif

vet:
	@go vet -composites=false -stdmethods=false ./...

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
	git clone --depth=1 --single-branch --branch eval https://github.com/brimsec/zq-sample-data $(@D)

sampledata: $(SAMPLEDATA)

bin/$(ZEEKPATH):
	@mkdir -p bin
	@curl -L -o bin/$(ZEEKPATH).zip \
		https://github.com/brimsec/zeek/releases/download/$(ZEEKTAG)/zeek-$(ZEEKTAG).$$(go env GOOS)-$$(go env GOARCH).zip
	@unzip -q bin/$(ZEEKPATH).zip -d bin \
		&& mv bin/zeek bin/$(ZEEKPATH)

bin/$(SURICATAPATH):
	@mkdir -p bin
	@curl -L -o bin/$(SURICATAPATH).zip \
		https://storage.googleapis.com/brimsec/suricata/suricata-$(SURICATATAG).$$(go env GOOS)-$$(go env GOARCH).zip
	@unzip -q bin/$(SURICATAPATH).zip -d bin \
		&& mv bin/suricata bin/$(SURICATAPATH)

bin/minio:
	@mkdir -p bin
	@echo 'module deps' > bin/go.mod
	@echo 'require github.com/minio/minio latest' >> bin/go.mod
	@echo 'replace github.com/minio/minio => github.com/brimsec/minio v0.0.0-20200716214025-90d56627f750' >> bin/go.mod
	@cd bin && GOBIN=$(CURDIR)/bin go install github.com/minio/minio

generate:
	@GOBIN=$(CURDIR)/bin go install github.com/golang/mock/mockgen
	@PATH=$(CURDIR)/bin:$(PATH) go generate ./...

test-generate: generate
	git diff --exit-code

test-unit:
	@go test -short ./...

test-system: build bin/minio bin/$(ZEEKPATH) bin/$(SURICATAPATH)
	@ZTEST_PATH=$(CURDIR)/dist:$(CURDIR)/bin:$(CURDIR)/bin/$(ZEEKPATH):$(CURDIR)/bin/$(SURICATAPATH) go test -v .

test-run: build bin/minio bin/$(ZEEKPATH) bin/$(SURICATAPATH)
	@ZTEST_PATH=$(CURDIR)/dist:$(CURDIR)/bin:$(CURDIR)/bin/$(ZEEKPATH):$(CURDIR)/bin/$(SURICATAPATH) go test -v . -run $(TEST)

test-heavy: build $(SAMPLEDATA)
	@go test -v -tags=heavy ./tests

test-zeek: bin/$(ZEEKPATH)
	@ZEEK=$(CURDIR)/bin/$(ZEEKPATH)/zeekrunner go test -v -run=PcapPost -tags=zeek ./zqd

perf-compare: build $(SAMPLEDATA)
	scripts/comparison-test.sh

zng-output-check: build $(SAMPLEDATA)
	scripts/zng-output-check.sh

# If the build recipe changes, please also change npm/build.
build:
	@mkdir -p dist
	@go build -ldflags='$(LDFLAGS)' -o dist ./cmd/...

install:
	@go install -ldflags='$(LDFLAGS)' ./cmd/...

docker:
	DOCKER_BUILDKIT=1 docker build --pull --rm \
		--build-arg LDFLAGS='$(LDFLAGS)' \
		-t zqd:latest -t localhost:5000/zqd:latest -t localhost:5000/zqd:$(VERSION) .
	docker push localhost:5000/zqd:latest
	docker push localhost:5000/zqd:$(VERSION)

create-release-assets:
	for os in darwin linux windows; do \
		zqdir=zq-$(VERSION).$${os}-amd64 ; \
		rm -rf dist/$${zqdir} ; \
		mkdir -p dist/$${zqdir} ; \
		cp LICENSE.txt acknowledgments.txt dist/$${zqdir} ; \
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
