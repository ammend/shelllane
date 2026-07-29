.PHONY: build test race vet check integration clean

build:
	mkdir -p bin
	go build -trimpath -ldflags "-s -w" -o bin/shelllane ./cmd/shelllane

test:
	go test ./...

race:
	go test -race ./...

vet:
	go vet ./...

check: test race vet
	test -z "$$(gofmt -l .)"
	git diff --check

integration:
	./test/integration.sh

clean:
	rm -rf bin coverage.out
