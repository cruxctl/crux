GO ?= go

.PHONY: fmt test build lint clean

fmt:
	$(GO) fmt ./...

test:
	$(GO) test ./...

build:
	mkdir -p bin
	$(GO) build -o bin/crux ./cmd/crux
	$(GO) build -o bin/cruxd ./cmd/cruxd

lint:
	$(GO) vet ./...

clean:
	rm -rf bin coverage.out

