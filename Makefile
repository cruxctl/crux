GO ?= go

.PHONY: fmt test build build-all lint clean console-build

fmt:
	$(GO) fmt ./...

test:
	$(GO) test ./...

build:
	mkdir -p bin
	$(GO) build -o bin/crux ./cmd/crux
	$(GO) build -o bin/cruxd ./cmd/cruxd
	$(GO) build -o bin/crux-mcp ./cmd/crux-mcp

console-build:
	cd web/console && pnpm build

lint:
	$(GO) vet ./...

clean:
	rm -rf bin coverage.out
