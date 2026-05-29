# logify — convenience targets.
#
#   make            build the local binary
#   make install    install ./logify into ~/.local/bin
#   make release    build all platform binaries into dist/
#   make clean      remove ./logify and dist/

VERSION ?= $(shell [ -f VERSION ] && cat VERSION || echo dev)
LDFLAGS = -s -w -X main.version=$(VERSION)

.PHONY: build install release clean fmt vet test

build:
	go build -ldflags='$(LDFLAGS)' -o logify ./cmd/logify

install: build
	@mkdir -p $$HOME/.local/bin
	@install -m 0755 logify $$HOME/.local/bin/logify
	@echo "installed $$HOME/.local/bin/logify"
	@case ":$$PATH:" in *":$$HOME/.local/bin:"*) ;; *)\
	  echo "warning: ~/.local/bin is not in PATH";; esac

release:
	bash scripts/build-all.sh

clean:
	rm -f logify
	rm -rf dist/

fmt:
	gofmt -w .

vet:
	go vet ./...

test:
	go test ./...
