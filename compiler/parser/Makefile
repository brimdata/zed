GOIMPORTS_VERSION = v0.0.0-20200204192400-7124308813f3
PEGJS_VERSION = 0.10.0
PIGEON_VERSION = v1.0.1-0.20190520151103-9fec3898cef8
ROLLUP_PLUGIN_COMMONJS_VERSION = 14.0.0
ROLLUP_VERSION = 2.23.1

deps = $(CURDIR)/deps
npm = npm --global --prefix $(deps)

PARSER_PREFIX = parser

all: $(PARSER_PREFIX).go $(PARSER_PREFIX).js $(PARSER_PREFIX).es.js

.PHONY: $(PARSER_PREFIX).go
$(PARSER_PREFIX).go:
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
	cpp -DGO -E -P parser.peg | $(deps)/bin/pigeon -o $@
	$(deps)/bin/goimports -w $@

.PHONY: $(PARSER_PREFIX).js
$(PARSER_PREFIX).js:
ifeq "$(shell $(deps)/bin/pegjs --version 2>&1 | fgrep $(PEGJS_VERSION))" ""
	$(npm) install pegjs@$(PEGJS_VERSION)
endif
	cpp -E -P parser.peg | $(deps)/bin/pegjs --allowed-start-rules start,Expr -o $@

.PHONY: $(PARSER_PREFIX).es.js
$(PARSER_PREFIX).es.js: $(PARSER_PREFIX).js
ifeq "$(shell $(deps)/bin/rollup --version 2>&1 | fgrep $(ROLLUP_VERSION))" ""
	$(npm) install rollup@$(ROLLUP_VERSION)
endif
ifeq "$(shell $(npm) list @rollup/plugin-commonjs 2>&1 | fgrep $(ROLLUP_PLUGIN_COMMONJS_VERSION))" ""
	$(npm) install @rollup/plugin-commonjs@$(ROLLUP_PLUGIN_COMMONJS_VERSION)
endif
	$(deps)/bin/rollup --format es --plugin commonjs $^ -o $@
