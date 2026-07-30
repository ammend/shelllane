PREFIX ?= /usr/local
BINDIR ?= $(PREFIX)/bin
LIBEXECDIR ?= $(PREFIX)/lib/shelllane
STATE_DIR ?= /var/lib/shelllane/users
LOG_DIR ?= /var/log/shelllane
DESTDIR ?=
INSTALL ?= install
RUNTIME_USER ?= $(if $(SUDO_USER),$(SUDO_USER),$(USER))
CONFIG_DIR ?=

.PHONY: build install install-bin install-configs install-runtime-dirs uninstall test race vet configs-check check integration docker-integration ssh-integration clean

build:
	mkdir -p bin
	go build -trimpath -ldflags "-s -w" -o bin/shelllane ./cmd/shelllane

install: install-bin install-runtime-dirs install-configs

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

install-configs:
	@test -n "$(RUNTIME_USER)" || { echo "RUNTIME_USER is required"; exit 1; }
	@user_home="$$(getent passwd "$(RUNTIME_USER)" | cut -d: -f6)"; \
	test -n "$$user_home" || { echo "Cannot resolve HOME for $(RUNTIME_USER)"; exit 1; }; \
	config_dir="$(CONFIG_DIR)"; \
	config_home="$(XDG_CONFIG_HOME)"; \
	test -n "$$config_home" || config_home="$$user_home/.config"; \
	test -n "$$config_dir" || config_dir="$$config_home/shelllane"; \
	case "$$config_dir" in /*) ;; *) echo "CONFIG_DIR must be absolute"; exit 1;; esac; \
	config_parent="$(DESTDIR)$$(dirname "$$config_dir")"; \
	test -d "$$config_parent" || $(INSTALL) -d -m 0700 -o "$(RUNTIME_USER)" "$$config_parent"; \
	test -d "$(DESTDIR)$$config_dir" || $(INSTALL) -d -m 0700 -o "$(RUNTIME_USER)" "$(DESTDIR)$$config_dir"; \
	for source in configs/*.yaml; do \
		$(INSTALL) -m 0644 -o "$(RUNTIME_USER)" "$$source" "$(DESTDIR)$$config_dir/$$(basename "$$source")"; \
	done; \
	if test ! -e "$(DESTDIR)$$config_dir/config.yaml"; then \
		$(INSTALL) -m 0600 -o "$(RUNTIME_USER)" configs/local-host.yaml "$(DESTDIR)$$config_dir/config.yaml"; \
	fi; \
	echo "Installed configuration templates in $$config_dir"

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
