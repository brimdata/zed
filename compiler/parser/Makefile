GOIMPORTS_VERSION = v0.0.0-20200204192400-7124308813f3
PEGJS_VERSION = 0.10.0
PIGEON_VERSION = v1.2.1
ROLLUP_PLUGIN_COMMONJS_VERSION = 14.0.0
ROLLUP_VERSION = 2.23.1

# GNU cpp needs -std=gnu90 to prevent errors on \uHHHH sequences, and Clang cpp
# with -std=gnu90 needs -Wno-unicode to prevent warnings on \uHHHH sequences.
cpp = cpp -E -P -std=gnu90 -Wno-unicode
deps = $(CURDIR)/deps
npm = npm --global --prefix $(deps)

all: parser.go

.PHONY: parser.go
parser.go:
ifeq "$(shell go version -m $(deps)/bin/goimports 2>&1 | fgrep $(GOIMPORTS_VERSION))" ""
	GOBIN=$(deps)/bin go install golang.org/x/tools/cmd/goimports@$(GOIMPORTS_VERSION)
endif
ifeq "$(shell go version -m $(deps)/bin/pigeon 2>&1 | fgrep $(PIGEON_VERSION))" ""
	GOBIN=$(deps)/bin go install github.com/mna/pigeon@$(PIGEON_VERSION)
endif
	$(cpp) -DGO parser.peg | $(deps)/bin/pigeon -o $@
	$(deps)/bin/goimports -w $@

.PHONY: parser.js
parser.js:
ifeq "$(shell $(deps)/bin/pegjs --version 2>&1 | fgrep $(PEGJS_VERSION))" ""
	$(npm) install pegjs@$(PEGJS_VERSION)
endif
	$(cpp) parser.peg | $(deps)/bin/pegjs --allowed-start-rules start,Expr -o $@

.PHONY: parser.es.js
parser.es.js: parser.js
ifeq "$(shell $(deps)/bin/rollup --version 2>&1 | fgrep $(ROLLUP_VERSION))" ""
	$(npm) install rollup@$(ROLLUP_VERSION)
endif
ifeq "$(shell $(npm) list @rollup/plugin-commonjs 2>&1 | fgrep $(ROLLUP_PLUGIN_COMMONJS_VERSION))" ""
	$(npm) install @rollup/plugin-commonjs@$(ROLLUP_PLUGIN_COMMONJS_VERSION)
endif
	$(deps)/bin/rollup --format es --plugin commonjs $^ -o $@
