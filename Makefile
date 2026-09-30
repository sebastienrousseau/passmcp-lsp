# SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com>
# SPDX-License-Identifier: Apache-2.0

.PHONY: all build test test-race coverage coverage-json vet lint format spdx-check smoke fuzz \
        crosscheck schema-check completions vscode readme-check name-guard lockstep family \
        versions docs help demo

# Every gate CI runs that needs no network, in the order the cheap ones fail
# first.
all: format vet lint spdx-check test smoke

build:
	CGO_ENABLED=0 go build -trimpath -o build/passmcp-lsp ./cmd/passmcp-lsp

test:
	go test ./... -cover

test-race:
	go test -race -shuffle=on -count=1 ./...

# The gate is 85% statement coverage in every package; ci.yml checks each.
coverage:
	@mkdir -p build
	go test -count=1 -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out | tail -1

# The shields.io endpoint document behind the README's coverage badge:
# statement coverage across the module, as CI measured it. The Manual
# workflow publishes it with GitHub Pages as coverage.json.
coverage-json: coverage
	go run ./scripts/coveragebadge -profile coverage.out > build/coverage.json
	@cat build/coverage.json

vet:
	go vet ./...

lint:
	golangci-lint run ./...

format:
	gofmt -l -w .

spdx-check:
	go run ./scripts/spdx_sweep.go

# The built server answers the handshake over stdio, publishes diagnostics
# for an opened server.json, and shuts down in order (scripts/smoke.go).
smoke: build
	go run scripts/smoke.go build/passmcp-lsp

# Each fuzz target for FUZZTIME, from its seeds and the committed corpus.
FUZZTIME ?= 10s
fuzz:
	go test ./internal/jsonrpc -run '^$$' -fuzz '^FuzzFraming$$' -fuzztime $(FUZZTIME)
	go test ./internal/jsondoc -run '^$$' -fuzz '^FuzzParse$$' -fuzztime $(FUZZTIME)

# The policy rules and the hover against the released passmcp this
# repository is in lockstep with. PASSMCP names the program; it defaults to
# passmcp on PATH.
PASSMCP ?= $(shell command -v passmcp)
crosscheck:
	@test -n "$(PASSMCP)" || { echo "crosscheck: install passmcp or set PASSMCP" >&2; exit 1; }
	PASSMCP="$(PASSMCP)" go test -count=1 -run 'Crosscheck|RealPassmcp' -v ./internal/check ./internal/guidance

# The embedded server.json schema is the registry's, byte for byte. Needs
# the network.
schema-check:
	@mkdir -p build
	curl -fsSL "$$(go run scripts/schemaurl.go)" -o build/server.schema.json
	cmp build/server.schema.json internal/check/schemas/server.schema.json
	@echo "schema-check: the embedded schema is the published one"

# Shell completions, generated from the flag set by the binary itself, into
# build/completions. bash is syntax-checked here; zsh and fish when present.
completions: build
	mkdir -p build/completions
	build/passmcp-lsp --completion bash > build/completions/passmcp-lsp.bash
	build/passmcp-lsp --completion zsh > build/completions/_passmcp-lsp
	build/passmcp-lsp --completion fish > build/completions/passmcp-lsp.fish
	bash -n build/completions/passmcp-lsp.bash
	if command -v zsh >/dev/null; then zsh -n build/completions/_passmcp-lsp; fi
	if command -v fish >/dev/null; then fish -n build/completions/passmcp-lsp.fish; fi

# The VS Code extension: install from the lock, test, and package the .vsix
# into editors/vscode/passmcp-lsp.vsix. Publishing it is the maintainer's.
vscode:
	cd editors/vscode && npm ci --no-audit --no-fund && npm test && npm run package

# The README follows the portfolio template and the family badge row.
readme-check:
	scripts/readme-check.sh

# A retired product name may not appear anywhere in the tree.
name-guard:
	./scripts/name-guard.sh

# The version is passmcp's latest release.
lockstep:
	scripts/lockstep.sh

# The family manifest in passmcp is the single source of what this repository is.
family:
	scripts/family.sh

# Every file that names the version names the newest CHANGELOG heading.
versions:
	scripts/verify-release-versions.sh "v$$(grep -Eo '^## \[[0-9]+\.[0-9]+\.[0-9]+\]' CHANGELOG.md | head -1 | tr -d '#[] ')"

# The manual, strictly: a broken link or nav entry fails.
docs:
	mkdocs build --strict --site-dir public

# The README demo (.github/demo.gif), rendered by VHS from .github/demo.tape:
# the Quick Start's misspelled policy flagged, then corrected. It runs in
# build/demo/work, which is git-ignored. Needs vhs, ttyd and ffmpeg.
demo: build
	rm -rf build/demo && mkdir -p build/demo/work
	PATH="$(CURDIR)/build:$$PATH" vhs .github/demo.tape

help:
	@printf '%s\n' "targets: all build test test-race coverage coverage-json vet lint format spdx-check smoke fuzz" \
	  "         crosscheck schema-check completions vscode readme-check name-guard lockstep family versions docs" \
	  "         demo" \
	  "GNUmakefile: install uninstall install-smoke (PREFIX, DESTDIR)"
