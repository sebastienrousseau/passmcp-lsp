# SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com>
# SPDX-License-Identifier: Apache-2.0
#
# The Unix install contract. GNU make reads this file before Makefile, so
# every developer target is pulled in from there and this file adds only
# install, uninstall and the staged smoke test packagers rely on.
#
# PREFIX defaults to /usr/local per the FHS; DESTDIR stages the tree
# elsewhere without changing the paths compiled into it, which is how
# distribution packages are built. For a home-directory install:
#   make install PREFIX=$$HOME/.local
#
# passmcp-lsp generates its shell completions from its flag set; it
# generates no manual page, so none is installed.

include Makefile

PREFIX ?= /usr/local
DESTDIR ?=
BINDIR = $(DESTDIR)$(PREFIX)/bin
DOCDIR = $(DESTDIR)$(PREFIX)/share/doc/passmcp-lsp
BASHCOMPDIR = $(DESTDIR)$(PREFIX)/share/bash-completion/completions
ZSHCOMPDIR = $(DESTDIR)$(PREFIX)/share/zsh/site-functions
FISHCOMPDIR = $(DESTDIR)$(PREFIX)/share/fish/vendor_completions.d

.PHONY: install uninstall install-smoke

install: completions
	install -d $(BINDIR)
	install -m 0755 build/passmcp-lsp $(BINDIR)/passmcp-lsp
	install -d $(BASHCOMPDIR) $(ZSHCOMPDIR) $(FISHCOMPDIR)
	install -m 0644 build/completions/passmcp-lsp.bash $(BASHCOMPDIR)/passmcp-lsp
	install -m 0644 build/completions/_passmcp-lsp $(ZSHCOMPDIR)/_passmcp-lsp
	install -m 0644 build/completions/passmcp-lsp.fish $(FISHCOMPDIR)/passmcp-lsp.fish
	install -d $(DOCDIR)
	install -m 0644 README.md CHANGELOG.md LICENSE SECURITY.md $(DOCDIR)/

uninstall:
	rm -f $(BINDIR)/passmcp-lsp
	rm -f $(BASHCOMPDIR)/passmcp-lsp
	rm -f $(ZSHCOMPDIR)/_passmcp-lsp
	rm -f $(FISHCOMPDIR)/passmcp-lsp.fish
	rm -rf $(DOCDIR)

# Stage an install under a temporary DESTDIR, check every file lands where
# the FHS says and the binary runs, then uninstall and check nothing is
# left behind.
install-smoke:
	@set -e; stage=$$(mktemp -d); trap 'rm -rf "$$stage"' EXIT; \
	$(MAKE) --no-print-directory install DESTDIR="$$stage" PREFIX=/usr >/dev/null; \
	for f in usr/bin/passmcp-lsp \
	         usr/share/bash-completion/completions/passmcp-lsp \
	         usr/share/zsh/site-functions/_passmcp-lsp \
	         usr/share/fish/vendor_completions.d/passmcp-lsp.fish \
	         usr/share/doc/passmcp-lsp/README.md \
	         usr/share/doc/passmcp-lsp/LICENSE; do \
	  test -f "$$stage/$$f" || { echo "install-smoke: missing $$f" >&2; exit 1; }; \
	done; \
	test -x "$$stage/usr/bin/passmcp-lsp" || { echo "install-smoke: the binary is not executable" >&2; exit 1; }; \
	"$$stage/usr/bin/passmcp-lsp" --version >/dev/null; \
	$(MAKE) --no-print-directory uninstall DESTDIR="$$stage" PREFIX=/usr >/dev/null; \
	left=$$(find "$$stage" -type f); \
	[ -z "$$left" ] || { echo "install-smoke: uninstall left $$left" >&2; exit 1; }; \
	echo "install-smoke: install and uninstall are correct under DESTDIR"
