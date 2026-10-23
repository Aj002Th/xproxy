.PHONY: build test race test-cover vet lint fmt fmt-check tidy clean run install smoke snapshot release check

BINARY := xproxy

build:
	go build -o $(BINARY) .

test:
	go test ./... -count=1 -timeout 60s

race:
	go test ./... -race -count=1 -timeout 120s

test-cover:
	go test ./... -count=1 -timeout 60s -coverprofile=coverage.out
	go tool cover -func=coverage.out

vet:
	go vet ./...

# Requires staticcheck: go install honnef.co/go/tools/cmd/staticcheck@latest
lint:
	staticcheck ./...

fmt:
	go fmt ./...

fmt-check:
	@files="$$(gofmt -l .)"; if [ -n "$$files" ]; then echo "gofmt needed:"; echo "$$files"; exit 1; fi

tidy:
	go mod tidy

clean:
	rm -f $(BINARY) coverage.out
	rm -rf dist/

run: build
	./$(BINARY) --hosts ./hosts.example --listen 127.0.0.1:8888 --log-level info

install:
	go install .

# Portable end-to-end smoke test (no nc/lsof dependency).
smoke: build
	go test ./internal/proxy -run 'TestServeHTTPEndToEnd|TestHandlerConnectTunnel' -count=1 -v

# goreleaser: local snapshot (does not publish)
snapshot:
	goreleaser release --snapshot --clean

# goreleaser: publish release (triggered by CI, use tag push instead)
release:
	goreleaser release --clean

check: fmt-check vet lint race
