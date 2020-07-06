deps = $(CURDIR)/deps

PEGJS = $(deps)/bin/pegjs
ROLLUP = $(deps)/bin/rollup
PIGEON = $(deps)/bin/pigeon
GOIMPORTS = $(deps)/bin/goimports

PEGJS_VERSION = 0.10.0
PIGEON_VERSION = v1.0.1-0.20190520151103-9fec3898cef8
GOIMPORTS_VERSION = v0.0.0-20200204192400-7124308813f3

PEGJS_INSTALLED = $(shell test -x $(PEGJS) && $(PEGJS) --version)
PIGEON_INSTALLED = $(shell test -d $(deps) && (cd $(deps); go list -m -f {{.Version}} github.com/mna/pigeon || echo ""))
GOIMPORTS_INSTALLED = $(shell test -d $(deps) && (cd $(deps); go list -m -f {{.Version}} golang.org/x/tools || echo ""))

PEGJS_ARGS = --allowed-start-rules start,Expression

all: zql.go zql.js zql.es.js

run: all
	go run ./main

$(PEGJS):
ifneq ($(PEGJS_VERSION),$(PEGJS_INSTALLED))
	mkdir -p $(deps)
	npm install --global --prefix $(deps) pegjs@$(PEGJS_VERSION) rollup @rollup/plugin-commonjs
endif

GODEPS :=
ifneq ($(PIGEON_VERSION),$(PIGEON_INSTALLED))
  GODEPS += github.com/mna/pigeon@$(PIGEON_VERSION)
endif
ifneq ($(GOIMPORTS_VERSION),$(GOIMPORTS_INSTALLED))
  GODEPS += golang.org/x/tools/cmd/goimports@$(GOIMPORTS_VERSION)
endif

.PHONY: $(PIGEON) $(GOIMPORTS)
$(PIGEON) $(GOIMPORTS):
ifneq (,$(GODEPS))
	mkdir -p $(deps)
	echo 'module deps' > $(deps)/go.mod
	cd $(deps) && GOBIN=$(deps)/bin go get $(GODEPS)
endif

#
# The actual parsers depend on additional files
# (parser-support.*, reglob, etc.).  Actually building the parsers is
# pretty cheap so just do that every time rather than trying to track
# all the proper dependencies.
#
.PHONY: zql.go
zql.go: $(PIGEON) $(GOIMPORTS)
	cpp -DGO -E -P zql.peg | $(PIGEON) -o $@
	$(GOIMPORTS) -w $@

.PHONY: zql.js
zql.js: $(PEGJS)
	cpp -E -P zql.peg | $(PEGJS) $(PEGJS_ARGS) -o $@

.PHONY: zql.es.js
zql.es.js: zql.js
	$(ROLLUP) --format es --plugin commonjs -o zql.es.js zql.js
	