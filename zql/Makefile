GOIMPORTS = goimports
PEGJS = pegjs
PIGEON = pigeon

all: zql.go zql.js

run: all
	go run ./main

.PHONY: zql.go
zql.go:
	cpp -DGO -E -P zql.peg | $(PIGEON) -o $@
	$(GOIMPORTS) -w $@

.PHONY: zql.js
zql.js:
	cpp -E -P zql.peg | $(PEGJS) -o $@

.PHONY: docker-build
docker-build:
	docker run --rm -it -v $$(pwd):/src -w /src $$(docker build -q .) make all
