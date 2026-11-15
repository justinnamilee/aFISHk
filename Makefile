GO ?= go
GOLANGCI_VERSION := v2.11.3
.PHONY: build fmt lint test check tools package clean
build:
	$(GO) build -trimpath -o bin/afishk ./cmd/afishk
fmt:
	gofmt -w cmd internal
lint:
	golangci-lint run

test:
	$(GO) test -race -count=1 ./...
check:
	@test -z "$$(gofmt -l cmd internal)" || (echo 'Run make fmt'; exit 1)
	$(GO) vet ./...
	$(GO) test -race -count=1 ./...
	@for file in scripts/*.sh; do bash -n "$$file" || exit; done

tools:
	$(GO) install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_VERSION)
package:
	./scripts/package.sh $(VERSION)
clean:
	rm -rf bin dist
