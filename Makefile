.PHONY: build test race vet check integration docker-integration ssh-integration clean

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

docker-integration:
	./test/docker-integration.sh

ssh-integration:
	./test/ssh-integration.sh

clean:
	rm -rf bin coverage.out
