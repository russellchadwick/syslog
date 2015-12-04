GO ?= go
# Allow setting of go build flags from the command line.
GOFLAGS :=

# Variables to be overridden on the command line, e.g.
#   make test PKG=./storage TESTFLAGS=--vmodule=multiraft=1
PKG          := ./...
TAGS         :=

.PHONY: all
all: build test check

.PHONY: build
build:
	$(GO) build $(GOFLAGS) -v

.PHONY: check
check:
	@echo "gometalinter"
	@! gometalinter $(PKG) --disable=structcheck --disable=aligncheck --deadline=60s | \
	grep -vE '(Godeps|vendor)'
	@echo "gofmt (simplify)"
	@! gofmt -s -d -l . 2>&1 | grep -vE '^\.git/'
	@echo "goimports"
	@! goimports -l . | grep -vF 'No Exceptions'

.PHONY: test
test:
	$(GO) test -tags '$(TAGS)' $(GOFLAGS) -i $(PKG)
