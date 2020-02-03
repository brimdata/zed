GOIMPORTS = goimports
PEGJS = pegjs
PIGEON = pigeon

.PHONY: gotools
gotools:
	@go install github.com/mna/pigeon
	@go install golang.org/x/tools/cmd/goimports

all: zql.go zql.js

run: all
	go run ./main

.PHONY: zql.go
zql.go: gotools
	cpp -DGO -E -P zql.peg | $(PIGEON) -o $@
	$(GOIMPORTS) -w $@

.PHONY: zql.js
zql.js:
	cpp -E -P zql.peg | $(PEGJS) -o $@
