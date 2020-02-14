deps = $(CURDIR)/deps

PEGJS = $(deps)/bin/pegjs
PIGEON = $(deps)/bin/pigeon
GOIMPORTS = $(deps)/bin/goimports

PEGJS_VERSION = $(shell test -x $(PEGJS) && $(PEGJS) --version)
PEGJS_EXPECTED = 0.10.0
PIGEON_VERSION = $(shell test -d $(deps) && cd $(deps); go list -m github.com/mna/pigeon || echo "")
PIGEON_EXPECTED = v1.0.1-20190520151103-9fec3898cef8
GOIMPORTS_VERSION = $(shell test -d $(deps) && cd $(deps); go list -m golang.org/x/tools || echo "")
GOIMPORTS_EXPECTED = v0.0.0-20200204192400-7124308813f3

all: zql.go zql.js

run: all
	go run ./main

$(PEGJS):
ifeq (,$(findstring $(PEGJS_EXPECTED),$(PEGJS_VERSION)))
	mkdir -p $(deps)
	npm install --global --prefix $(deps) pegjs@0.10.0
endif

GODEPS :=
ifeq (,$(findstring $(PIGEON_EXPECTED),$(PIGEON_VERSION)))
  GODEPS += github.com/mna/pigeon@v1.0.1-0.20190520151103-9fec3898cef8
endif
ifeq (,$(findstring $(GOIMPORTS_EXPECTED),$(GOIMPORTS_VERSION)))
  GODEPS += golang.org/x/tools/cmd/goimports@v0.0.0-20200204192400-7124308813f3
endif

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
	cpp -E -P zql.peg | $(PEGJS) -o $@
