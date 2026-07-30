PREFIX ?= /usr/local
BINDIR ?= $(PREFIX)/bin
LIBEXECDIR ?= $(PREFIX)/lib/shelllane
STATE_DIR ?= /var/lib/shelllane/users
LOG_DIR ?= /var/log/shelllane
DESTDIR ?=
INSTALL ?= install
RUNTIME_USER ?= $(if $(SUDO_USER),$(SUDO_USER),$(USER))

.PHONY: build install install-bin install-runtime-dirs uninstall test race vet configs-check check integration docker-integration ssh-integration clean

build:
	mkdir -p bin
	go build -trimpath -ldflags "-s -w" -o bin/shelllane ./cmd/shelllane

install: install-bin install-runtime-dirs

install-bin: build
	$(INSTALL) -d "$(DESTDIR)$(BINDIR)" "$(DESTDIR)$(LIBEXECDIR)"
	$(INSTALL) -m 0755 bin/shelllane "$(DESTDIR)$(BINDIR)/shelllane"
	ln -sfn "$(BINDIR)/shelllane" "$(DESTDIR)$(LIBEXECDIR)/bash"
	ln -sfn "$(BINDIR)/shelllane" "$(DESTDIR)$(LIBEXECDIR)/zsh"
	ln -sfn "$(BINDIR)/shelllane" "$(DESTDIR)$(LIBEXECDIR)/sh"

install-runtime-dirs:
	@test -n "$(RUNTIME_USER)" || { echo "RUNTIME_USER is required"; exit 1; }
	$(INSTALL) -d -m 0700 -o "$(RUNTIME_USER)" "$(DESTDIR)$(STATE_DIR)"
	$(INSTALL) -d -m 0700 -o "$(RUNTIME_USER)" "$(DESTDIR)$(LOG_DIR)"

uninstall:
	rm -f "$(DESTDIR)$(LIBEXECDIR)/bash" "$(DESTDIR)$(LIBEXECDIR)/zsh" "$(DESTDIR)$(LIBEXECDIR)/sh"
	rm -f "$(DESTDIR)$(BINDIR)/shelllane"
	@echo "Preserved configuration, state, and logs."

test:
	go test ./...

race:
	go test -race ./...

vet:
	go vet ./...

configs-check:
	@for config in configs/*.yaml; do \
		go run ./cmd/shelllane config check --config "$$config"; \
	done

check: test race vet configs-check
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
