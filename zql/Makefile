
ZQL_DIR = $(dir $(abspath $(MAKEFILE_LIST)))

GOIMPORTS = goimports
PIGEON = pigeon
PEGJS = $(ZQL_DIR)/../node_modules/.bin/pegjs

all: zql.go zql.js

run: all
	go run ./main

$(PEGJS):
	$(error Can't find pegjs in $(PEGJS), do you need to run "npm install"?)

# XXX these shouldn't really be .PHONY but we don't have a good way to
# automatically track dependencies (parser-support.*, reglob, etc)
.PHONY: zql.go
zql.go:
	cpp -DGO -E -P zql.peg | $(PIGEON) -o $@
	$(GOIMPORTS) -w $@

.PHONY: zql.js
zql.js: $(PEGJS)
	cpp -E -P zql.peg | $(PEGJS) -o $@
