.PHONY: build test test-cover vet clean run install fmt tidy smoke snapshot release

BINARY := xproxy

build:
	go build -o $(BINARY) .

test:
	go test ./... -count=1 -timeout 30s

test-cover:
	go test ./... -count=1 -timeout 30s -coverprofile=coverage.out
	go tool cover -func=coverage.out

vet:
	go vet ./...

clean:
	rm -f $(BINARY) coverage.out
	rm -rf dist/

run: build
	./$(BINARY) --hosts ./hosts.example --listen 127.0.0.1:8888 --log-level info

install:
	go install .

fmt:
	go fmt ./...

tidy:
	go mod tidy

# Full end-to-end smoke test: upstream → proxy → client
smoke: build
	@echo "127.0.0.1 example.com" > /tmp/test-hosts; \
		( printf 'HTTP/1.1 200 OK\r\nContent-Length: 4\r\n\r\npong'; sleep 10; ) | nc -l 127.0.0.1 19998 & \
		./$(BINARY) --hosts /tmp/test-hosts --listen 127.0.0.1:19999 --log-level error & \
		sleep 2; \
		echo "--- smoke test ---"; \
		RESP=$$(curl -s -x http://127.0.0.1:19999 http://example.com:19998/); \
		if echo "$$RESP" | grep -q pong; then \
			echo "PASS: got '$$RESP'"; \
		else \
			echo "FAIL: got '$$RESP'"; \
			exit 1; \
		fi; \
		kill $$(lsof -t -i :19999) 2>/dev/null || true; \
		kill $$(lsof -t -i :19998) 2>/dev/null || true; \
		rm -f /tmp/test-hosts

# goreleaser: local snapshot (does not publish)
snapshot:
	goreleaser release --snapshot --clean

# goreleaser: publish release (triggered by CI, use tag push instead)
release:
	goreleaser release --clean
