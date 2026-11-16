GO ?= go
GOFMT ?= gofmt
SHFMT ?= shfmt
DPRINT := ./.tools/dprint
GOLANGCI_VERSION := v2.11.3
SHFMT_VERSION := v3.12.0
SHELL_SCRIPTS := $(wildcard scripts/*.sh scripts/linux/*.sh tests/*.sh)
.PHONY: build fmt fmt-check lint test check tools format-tools package clean
build:
	$(GO) build -trimpath -o bin/afishk ./cmd/afishk
fmt:
	$(GOFMT) -w cmd internal
	$(SHFMT) -w -i 2 -ci $(SHELL_SCRIPTS)
	$(DPRINT) fmt
fmt-check:
	@command -v $(GOFMT) >/dev/null
	@files="$$($(GOFMT) -l cmd internal)" || exit; \
		test -z "$$files" || (echo 'Run make fmt'; exit 1)
	$(SHFMT) -d -i 2 -ci $(SHELL_SCRIPTS)
	$(DPRINT) check
lint:
	golangci-lint run
	shellcheck $(SHELL_SCRIPTS)

test:
	$(GO) test -race -count=1 ./...
check: fmt-check
	$(GO) vet ./...
	$(GO) test -race -count=1 ./...
	@for file in $(SHELL_SCRIPTS); do bash -n "$$file" || exit; done
	./tests/install.sh

tools: format-tools
	$(GO) install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_VERSION)
format-tools:
	$(GO) install mvdan.cc/sh/v3/cmd/shfmt@$(SHFMT_VERSION)
	./scripts/dprint.sh
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
	./scripts/linux/install.sh

.PHONY: windows release
windows:
	@test "$$(uname -s)" = Linux || { echo "Build Windows binaries on Linux."; exit 1; }
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 $(GO) build -trimpath -ldflags="-H windowsgui" -o bin/afishk.exe ./cmd/afishk
release:
	./scripts/package.sh $(VERSION)
	./scripts/package-windows.sh $(VERSION)
	cd dist && sha256sum afishk_$(VERSION)_linux_amd64.tar.gz afishk_$(VERSION)_linux_arm64.tar.gz afishk_$(VERSION)_windows_amd64_setup.exe afishk_$(VERSION)_windows_media_sources.tar.gz > SHA256SUMS
