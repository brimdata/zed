GOIMPORTS_VERSION = v0.19.0
PIGEON_VERSION = v1.2.1

# GNU cpp needs -std=gnu90 to prevent errors on \uHHHH sequences, and Clang cpp
# with -std=gnu90 needs -Wno-unicode to prevent warnings on \uHHHH sequences.
cpp = cpp -E -P -std=gnu90 -Wno-unicode
deps = $(CURDIR)/deps

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
