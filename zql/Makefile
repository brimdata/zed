deps = $(CURDIR)/deps

all: zql.go zql.js

run: all
	go run ./main

.PHONY: zql.go
zql.go:
	mkdir -p $(deps)
	echo 'module deps' > $(deps)/go.mod
	cd $(deps) && GOBIN=$(deps)/bin go get \
		github.com/mna/pigeon@v1.0.1-0.20190520151103-9fec3898cef8 \
		golang.org/x/tools/cmd/goimports@v0.0.0-20200204192400-7124308813f3
	cpp -DGO -E -P zql.peg | $(deps)/bin/pigeon -o $@
	$(deps)/bin/goimports -w $@

.PHONY: zql.js
zql.js:
	npm install --global --prefix $(deps) pegjs@0.10.0
	cpp -E -P zql.peg | $(deps)/bin/pegjs -o $@
