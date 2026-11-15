GO ?= go
GOFMT ?= gofmt
SHFMT ?= shfmt
DPRINT := ./.tools/dprint
GOLANGCI_VERSION := v2.11.3
SHFMT_VERSION := v3.12.0
.PHONY: build fmt fmt-check lint test check tools format-tools package clean
build:
	$(GO) build -trimpath -o bin/afishk ./cmd/afishk
fmt:
	$(GOFMT) -w cmd internal
	$(SHFMT) -w -i 2 -ci scripts/*.sh
	$(DPRINT) fmt
fmt-check:
	@command -v $(GOFMT) >/dev/null
	@files="$$($(GOFMT) -l cmd internal)" || exit; \
		test -z "$$files" || (echo 'Run make fmt'; exit 1)
	$(SHFMT) -d -i 2 -ci scripts/*.sh
	$(DPRINT) check
lint:
	golangci-lint run
	shellcheck scripts/*.sh

test:
	$(GO) test -race -count=1 ./...
check: fmt-check
	$(GO) vet ./...
	$(GO) test -race -count=1 ./...
	@for file in scripts/*.sh; do bash -n "$$file" || exit; done

tools: format-tools
	$(GO) install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_VERSION)
format-tools:
	$(GO) install mvdan.cc/sh/v3/cmd/shfmt@$(SHFMT_VERSION)
	./scripts/install-dprint.sh
	$(DPRINT) install
package:
	./scripts/package.sh $(VERSION)
clean:
	rm -rf bin dist

# Per-user installation. Override with environment variables or make VAR=value.
BINDIR ?= $(HOME)/.local/bin
CONFIGDIR ?= $(if $(XDG_CONFIG_HOME),$(XDG_CONFIG_HOME),$(HOME)/.config)/afishk
SYSTEMD_USER_DIR ?= $(if $(XDG_CONFIG_HOME),$(XDG_CONFIG_HOME),$(HOME)/.config)/systemd/user
APPLICATIONSDIR ?= $(if $(XDG_DATA_HOME),$(XDG_DATA_HOME),$(HOME)/.local/share)/applications
BINARY ?= bin/afishk
export BINDIR CONFIGDIR SYSTEMD_USER_DIR APPLICATIONSDIR BINARY
.PHONY: install
install:
	./scripts/install.sh
