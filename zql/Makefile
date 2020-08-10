GOIMPORTS_VERSION = v0.0.0-20200204192400-7124308813f3
PEGJS_VERSION = 0.10.0
PIGEON_VERSION = v1.0.1-0.20190520151103-9fec3898cef8

deps = $(CURDIR)/deps
npm = npm --global --prefix $(deps)

all: zql.go zql.js zql.es.js

run: zql.go zql.js
	go run ./main

.PHONY: zql.go
zql.go:
ifeq "$(shell go version -m $(deps)/bin/goimports 2>&1 | fgrep $(GOIMPORTS_VERSION))" ""
	mkdir -p $(deps)
	echo 'module deps' > $(deps)/go.mod
	cd $(deps) && GOBIN=$(deps)/bin go get golang.org/x/tools/cmd/goimports@$(GOIMPORTS_VERSION)
endif
ifeq "$(shell go version -m $(deps)/bin/pigeon 2>&1 | fgrep $(PIGEON_VERSION))" ""
	mkdir -p $(deps)
	echo 'module deps' > $(deps)/go.mod
	cd $(deps) && GOBIN=$(deps)/bin go get github.com/mna/pigeon@$(PIGEON_VERSION)
endif
	cpp -DGO -E -P zql.peg | $(deps)/bin/pigeon -o $@
	$(deps)/bin/goimports -w $@

.PHONY: zql.js
zql.js:
ifeq "$(shell $(deps)/bin/pegjs --version 2>&1 | fgrep $(PEGJS_VERSION))" ""
	$(npm) install pegjs@$(PEGJS_VERSION) rollup @rollup/plugin-commonjs
endif
	cpp -E -P zql.peg | $(deps)/bin/pegjs --allowed-start-rules start,Expression -o $@

.PHONY: zql.es.js
zql.es.js: zql.js
	$(deps)/bin/rollup --format es --plugin commonjs $^ -o $@
